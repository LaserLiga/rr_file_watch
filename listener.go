package roadrunner

import (
	"context"
	"encoding/json"
	"github.com/radovskyb/watcher"
	"github.com/roadrunner-server/goridge/v3/pkg/frame"
	"github.com/roadrunner-server/pool/payload"
	"go.uber.org/zap"
	"regexp"
	"time"
)

func (p *Plugin) listener() {
	go func() {
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

					pld := payload.Payload{
						Body:  eventDetailsBytes,
						Codec: frame.CodecRaw,
					}

					p.log.Debug("Sending event", zap.String("payload", pld.String()))

					// protect from the pool reset
					p.mu.RLock()

					ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*10))
					_, execErr := p.workersPool.Exec(ctx, &pld, nil)
					cancel()
					p.mu.RUnlock()

					if execErr != nil {
						p.metrics.CountJobErr()

						p.log.Error("notification processed with errors", zap.Error(execErr), zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
						continue
					}

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
	}()
}
