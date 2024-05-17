package roadrunner

import (
	"context"
	"github.com/radovskyb/watcher"
	"github.com/roadrunner-server/goridge/v3/pkg/frame"
	"github.com/roadrunner-server/sdk/v4/payload"
	"go.uber.org/zap"
	"regexp"
	"time"
)

func (p *Plugin) listener() {
	w := watcher.New()
	w.SetMaxEvents(1)
	w.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Write)

	if p.cfg.regexp != "" {
		r := regexp.MustCompile(p.cfg.regexp)
		w.AddFilterHook(watcher.RegexFilterHook(r, false))
	}

	go func() {
		for {
			select {
			case <-p.stopCh:
				p.log.Debug("------> job poller was stopped <------")
				return
			case event := <-w.Event:
				start := time.Now().UTC()

				p.log.Debug("Received a file event", zap.String("event", event.String()))

				// Create payload
				pld := payload.Payload{
					Body:  []byte(event.Name()),
					Codec: frame.CodecRaw,
				}

				p.metrics.CountEvents()

				// protect from the pool reset
				p.mu.RLock()
				re, err := p.workersPool.Exec(context.Background(), &pld, nil)
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
				return
			}
		}
	}()

	if err := w.Add(p.cfg.dir); err != nil {
		p.log.Error(err.Error())
	}
}
