package roadrunner

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/radovskyb/watcher"
	"github.com/roadrunner-server/pool/pool/static_pool"
)

type fakeWorkerResponse struct {
	body []byte
	err  error
}

func (f fakeWorkerResponse) Body() []byte {
	return f.body
}

func (f fakeWorkerResponse) Error() error {
	return f.err
}

func TestClassifyWorkerExecutionResponseOK(t *testing.T) {
	err := classifyWorkerExecutionResponse(fakeWorkerResponse{body: []byte("OK")})
	if err != nil {
		t.Fatalf("expected OK response to succeed, got %v", err)
	}
}

func TestClassifyWorkerExecutionResponseErrorBody(t *testing.T) {
	err := classifyWorkerExecutionResponse(fakeWorkerResponse{body: []byte("ERROR")})
	if err == nil {
		t.Fatal("expected ERROR response to fail")
	}
	if !strings.Contains(err.Error(), "ERROR") {
		t.Fatalf("expected error to mention ERROR, got %v", err)
	}
}

func TestClassifyWorkerExecutionResponseTransportError(t *testing.T) {
	expected := errors.New("exec failed")

	err := classifyWorkerExecutionResponse(fakeWorkerResponse{err: expected})
	if !errors.Is(err, expected) {
		t.Fatalf("expected transport error %v, got %v", expected, err)
	}
}

func TestClassifyWorkerExecutionResponseUnexpectedBody(t *testing.T) {
	err := classifyWorkerExecutionResponse(fakeWorkerResponse{body: []byte("MAYBE")})
	if err == nil {
		t.Fatal("expected unexpected response to fail")
	}
	if !strings.Contains(err.Error(), "unexpected response") {
		t.Fatalf("expected unexpected response error, got %v", err)
	}
}

func TestClassifyWorkerExecutionResponseNil(t *testing.T) {
	err := classifyWorkerExecutionResponse(nil)
	if err == nil {
		t.Fatal("expected nil response to fail")
	}
	if !strings.Contains(err.Error(), "nil response") {
		t.Fatalf("expected nil response error, got %v", err)
	}
}

func TestClassifyWorkerResponseClosedChannel(t *testing.T) {
	responses := make(chan *static_pool.PExec)
	close(responses)

	err := classifyWorkerResponse(t.Context(), responses)
	if err == nil {
		t.Fatal("expected closed response channel to fail")
	}
	if !strings.Contains(err.Error(), "no response") {
		t.Fatalf("expected no response error, got %v", err)
	}
}

func TestClassifyWorkerResponseContextDeadline(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := classifyWorkerResponse(ctx, make(chan *static_pool.PExec))
	if err == nil {
		t.Fatal("expected context cancellation to fail")
	}
}

func TestScheduleDebouncedEventCoalescesByPath(t *testing.T) {
	pending := make(map[string]*pendingFileEvent)
	ready := make(chan debouncedFileEvent, 8)

	scheduleDebouncedEvent(pending, ready, watcher.Event{Path: "result.game", Op: watcher.Create}, 20*time.Millisecond)
	scheduleDebouncedEvent(pending, ready, watcher.Event{Path: "result.game", Op: watcher.Write}, 20*time.Millisecond)
	scheduleDebouncedEvent(pending, ready, watcher.Event{Path: "other.game", Op: watcher.Create}, 20*time.Millisecond)

	var resultReady debouncedFileEvent
	var otherReady debouncedFileEvent
	deadline := time.After(500 * time.Millisecond)

	for resultReady.path == "" || otherReady.path == "" {
		select {
		case eventRef := <-ready:
			pendingEvent := pending[eventRef.path]
			if pendingEvent == nil || pendingEvent.seq != eventRef.seq {
				continue
			}
			switch eventRef.path {
			case "result.game":
				resultReady = eventRef
			case "other.game":
				otherReady = eventRef
			}
		case <-deadline:
			t.Fatal("timed out waiting for debounced events")
		}
	}

	if got := pending[resultReady.path].event.Op; got != watcher.Write {
		t.Fatalf("expected latest result.game event to be WRITE, got %s", got)
	}
	if got := pending[otherReady.path].event.Op; got != watcher.Create {
		t.Fatalf("expected other.game event to be CREATE, got %s", got)
	}

	stopPendingEvents(pending)
}
