package roadrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/radovskyb/watcher"
	rrErrors "github.com/roadrunner-server/errors"
	"github.com/roadrunner-server/goridge/v3/pkg/frame"
	"github.com/roadrunner-server/pool/payload"
	"github.com/roadrunner-server/pool/pool/static_pool"
	"go.uber.org/zap"
)

const (
	workerResponseOK    = "OK"
	workerResponseError = "ERROR"
)

func (p *Plugin) listener() error {
	w := watcher.New()
	w.FilterOps(watcher.Rename, watcher.Move, watcher.Create, watcher.Write)

	if p.cfg.Regexp != "" {
		r := regexp.MustCompile(p.cfg.Regexp)
		w.AddFilterHook(watcher.RegexFilterHook(r, false))
	}

	dirs := p.cfg.WatchDirs()
	for _, dir := range dirs {
		if err := w.Add(dir); err != nil {
			return err
		}
	}

	debounce, err := p.cfg.DebounceDuration()
	if err != nil {
		return err
	}

	p.watcher = w
	stopCh := p.stopCh

	p.log.Debug("Starting file watch", zap.Strings("dirs", dirs), zap.String("regexp", p.cfg.Regexp), zap.Duration("debounce", debounce))

	go p.watchEvents(w, debounce, stopCh)

	go func() {
		if err := w.Start(time.Millisecond * 100); err != nil {
			p.log.Error("file watcher stopped with error", zap.Error(err))
		}
	}()
	w.Wait()

	return nil
}

type debouncedFileEvent struct {
	path string
	seq  uint64
}

type pendingFileEvent struct {
	event watcher.Event
	seq   uint64
	timer *time.Timer
}

func (p *Plugin) watchEvents(w *watcher.Watcher, debounce time.Duration, stopCh <-chan struct{}) {
	pending := make(map[string]*pendingFileEvent)
	ready := make(chan debouncedFileEvent, 1024)

	for {
		select {
		case <-stopCh:
			stopPendingEvents(pending)
			p.log.Debug("------> file watch poller was stopped <------")
			return
		case event := <-w.Event:
			p.log.Debug("Received a file event", zap.String("event", event.String()))

			p.metrics.CountEvents()

			if debounce > 0 {
				scheduleDebouncedEvent(pending, ready, event, debounce)
				p.log.Debug("file event scheduled by debounce", zap.String("path", event.Path), zap.Duration("debounce", debounce))
				continue
			}

			p.dispatchEvent(event)
		case eventRef := <-ready:
			pendingEvent, ok := pending[eventRef.path]
			if !ok || pendingEvent.seq != eventRef.seq {
				continue
			}
			event := pendingEvent.event
			delete(pending, eventRef.path)

			p.dispatchEvent(event)
		case err := <-w.Error:
			p.log.Error(err.Error())
		case <-w.Closed:
			stopPendingEvents(pending)
			p.log.Debug("File watch closing")
			return
		}
	}
}

func scheduleDebouncedEvent(pending map[string]*pendingFileEvent, ready chan<- debouncedFileEvent, event watcher.Event, debounce time.Duration) {
	path := event.Path
	current, ok := pending[path]
	if !ok {
		current = &pendingFileEvent{}
		pending[path] = current
	}

	if current.timer != nil {
		current.timer.Stop()
	}

	current.event = event
	current.seq++
	seq := current.seq
	current.timer = time.AfterFunc(debounce, func() {
		ready <- debouncedFileEvent{path: path, seq: seq}
	})
}

func stopPendingEvents(pending map[string]*pendingFileEvent) {
	for path, event := range pending {
		if event.timer != nil {
			event.timer.Stop()
		}
		delete(pending, path)
	}
}

func (p *Plugin) dispatchEvent(event watcher.Event) {
	start := time.Now().UTC()

	eventDetails := map[string]interface{}{
		"directory": p.watchedDirectoryForEvent(event.Path),
		"file":      event.Name(),
		"op":        event.Op.String(),
		"path":      event.Path,
		"eventTime": event.ModTime().String(),
	}

	eventDetailsBytes, err := json.Marshal(eventDetails)
	if err != nil {
		p.log.Error("Failed to marshal event details", zap.Error(err))
		return
	}

	pld := payload.Payload{
		Body:  eventDetailsBytes,
		Codec: frame.CodecRaw,
	}

	p.log.Debug("Sending event", zap.String("payload", pld.String()))

	execErr := p.executePayload(&pld)
	if execErr != nil {
		p.metrics.CountJobErr()

		p.log.Error("notification processed with errors", zap.Error(execErr), zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
		return
	}

	p.metrics.CountJobOk()

	p.log.Debug("notification was processed successfully", zap.Time("start", start), zap.Int64("elapsed", time.Since(start).Milliseconds()))
}

func (p *Plugin) watchedDirectoryForEvent(path string) string {
	eventPath, err := filepath.Abs(path)
	if err != nil {
		eventPath = filepath.Clean(path)
	}

	for _, dir := range p.cfg.WatchDirs() {
		watchDir, err := filepath.Abs(dir)
		if err != nil {
			watchDir = filepath.Clean(dir)
		}
		rel, err := filepath.Rel(watchDir, eventPath)
		if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return dir
		}
		if err == nil && rel == "." {
			return dir
		}
	}

	if p.cfg.Dir != "" {
		return p.cfg.Dir
	}
	dirs := p.cfg.WatchDirs()
	if len(dirs) > 0 {
		return dirs[0]
	}
	return ""
}

func (p *Plugin) executePayload(pld *payload.Payload) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Protect from pool reset while Exec is using the pool.
	p.mu.RLock()
	pool := p.workersPool
	if pool == nil {
		p.mu.RUnlock()
		return rrErrors.Str("worker pool is not initialized")
	}
	responses, execErr := pool.Exec(ctx, pld, nil)
	p.mu.RUnlock()

	if execErr != nil {
		return execErr
	}

	return classifyWorkerResponse(ctx, responses)
}

func classifyWorkerResponse(ctx context.Context, responses <-chan *static_pool.PExec) error {
	select {
	case <-ctx.Done():
		return rrErrors.E(rrErrors.Op("file_watch_worker_response"), rrErrors.ExecTTL, ctx.Err())
	case response, ok := <-responses:
		if !ok {
			return rrErrors.Str("worker returned no response")
		}
		return classifyWorkerExecutionResponse(response)
	}
}

type workerExecutionResponse interface {
	Body() []byte
	Error() error
}

func classifyWorkerExecutionResponse(response workerExecutionResponse) error {
	if response == nil {
		return rrErrors.Str("worker returned nil response")
	}
	if err := response.Error(); err != nil {
		return err
	}

	// Workers use a small text protocol: OK means the event was accepted,
	// ERROR means the worker handled the request but could not process it.
	// Anything else is treated as failed so metrics do not report false success.
	body := bytes.TrimSpace(response.Body())
	switch string(body) {
	case workerResponseOK:
		return nil
	case workerResponseError:
		return rrErrors.Str("worker returned ERROR")
	default:
		return rrErrors.Errorf("worker returned unexpected response %q", string(body))
	}
}
