package roadrunner

import (
	"context"
	"encoding/json"
	"github.com/radovskyb/watcher"
	"github.com/roadrunner-server/goridge/v3/pkg/frame"
	"github.com/roadrunner-server/sdk/v4/payload"
	"go.uber.org/zap"
	"regexp"
	"time"
)

func (p *Plugin) listener() {
	w := watcher.New()
	w.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Write)

	if p.cfg.Regexp != "" {
		r := regexp.MustCompile(p.cfg.Regexp)
		w.AddFilterHook(watcher.RegexFilterHook(r, false))
	}

	p.log.Debug("Starting watching on " + p.cfg.Dir + "( " + p.cfg.Regexp + " )")

	go func() {
		for {
			select {
			case <-p.stopCh:
				p.log.Debug("------> file watch poller was stopped <------")
				return
			case event := <-w.Event:
				start := time.Now().UTC()

				p.log.Debug("Received a file event", zap.String("event", event.String()))

				eventDetails := map[string]interface{}{
					"directory": p.cfg.Dir,
					"file":      event.Name(),
					"op":        event.Op.String(),
					"path":      event.Path,
					"eventTime": event.ModTime().String(),
				}

				eventDetailsBytes, err := json.Marshal(eventDetails)
				if err != nil {
					p.log.Error("Failed to marshal event details", zap.Error(err))
					continue
				}

				p.metrics.CountEvents()

				// protect from the pool reset
				p.mu.RLock()
				re, err := p.workersPool.Exec(
					context.Background(),
					&payload.Payload{
						Body:  eventDetailsBytes,
						Codec: frame.CodecRaw,
					},
					p.stopCh,
				)
				p.mu.RUnlock()

				if err != nil {
					p.metrics.CountJobErr()

					p.log.Error("notification processed with errors", zap.Error(err), zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
					continue
				}

				var resp *payload.Payload

				select {
				case pld := <-re:
					if pld.Error() != nil {
						p.metrics.CountJobErr()

						p.log.Error("notification processed with errors", zap.Error(err), zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
						continue
					}

					// streaming is not supported
					if pld.Payload().Flags&frame.STREAM != 0 {
						p.metrics.CountJobErr()

						p.log.Warn("streaming is not supported",
							zap.Time("start", start),
							zap.Int64("elapsed", time.Since(start).Milliseconds()))

						p.log.Error("notification execute failed", zap.Error(err))
						continue
					}

					// assign the payload
					resp = pld.Payload()
				default:
					// should never happen
					p.metrics.CountJobErr()
					p.log.Error("worker null response, this is not expected")
				}

				// if response is nil or body is nil, just acknowledge the job
				if resp == nil || resp.Body == nil {
					p.log.Debug("notification was processed successfully", zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))

					p.metrics.CountJobOk()
					continue
				}

				// TODO: handle the response protocol

				p.metrics.CountJobOk()

				p.log.Debug("notification was processed successfully", zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
			case err := <-w.Error:
				p.log.Error(err.Error())
			case <-w.Closed:
				p.log.Debug("File watch closing")
				return
			}
		}
	}()

	if err := w.Add(p.cfg.Dir); err != nil {
		p.log.Error(err.Error())
	}

	if err := w.Start(time.Millisecond * 100); err != nil {
		p.log.Error(err.Error())
	}
}
