package roadrunner

import (
	"net/http"
	"testing"

	"go.uber.org/zap"
)

func TestStatusReturnsUnavailableWithoutPool(t *testing.T) {
	p := &Plugin{}

	st, err := p.Status()
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if st.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, st.Code)
	}
}

func TestReadyReturnsUnavailableWithoutPool(t *testing.T) {
	p := &Plugin{}

	st, err := p.Ready()
	if err != nil {
		t.Fatalf("Ready returned error: %v", err)
	}
	if st.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, st.Code)
	}
}

func TestResetReturnsErrorWithoutPool(t *testing.T) {
	p := &Plugin{log: zap.NewNop()}

	if err := p.Reset(); err == nil {
		t.Fatal("expected Reset to fail without a worker pool")
	}
}

func TestStopIsIdempotent(t *testing.T) {
	p := &Plugin{stopCh: make(chan struct{})}

	if err := p.Stop(t.Context()); err != nil {
		t.Fatalf("first Stop returned error: %v", err)
	}
	if err := p.Stop(t.Context()); err != nil {
		t.Fatalf("second Stop returned error: %v", err)
	}
}
