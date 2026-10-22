package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/bus"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// stallingWriter passes the hydration burst through, then blocks its first
// later write until released — a browser that stopped draining its stream.
type stallingWriter struct {
	header  http.Header
	mu      sync.Mutex
	body    strings.Builder
	stall   bool
	stalled chan struct{}
	release chan struct{}
}

func (w *stallingWriter) Header() http.Header { return w.header }
func (w *stallingWriter) WriteHeader(int)     {}
func (w *stallingWriter) Flush()              {}

func (w *stallingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	stall := w.stall
	w.stall = false
	w.mu.Unlock()
	if stall {
		close(w.stalled)
		<-w.release
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.Write(p)
}

func (w *stallingWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.body.String()
}

// TS-03.R9 (CR-A7-1): a subscriber that loses an event must not keep a healthy
// stream whose pings hide the loss. The overflowed stream ends, and the
// browser's reconnect receives the state it missed in the fresh snapshot.
func TestSSEOverflowEndsStreamAndReconnectCarriesLostState(t *testing.T) {
	s := testServer(t, false)
	w := &stallingWriter{header: http.Header{}, stalled: make(chan struct{}), release: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handleEvents(w, httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx))
	}()
	deadline := time.Now().Add(3 * time.Second)
	for !strings.Contains(w.String(), `"hydrated":true`) {
		if time.Now().After(deadline) {
			t.Fatal("stream never hydrated")
		}
		time.Sleep(5 * time.Millisecond)
	}

	w.mu.Lock()
	w.stall = true
	w.mu.Unlock()
	agent := "a_lost"
	s.eventBus.Publish("notification", nil, map[string]any{})
	<-w.stalled
	s.eventBus.PublishStateUpdate(state.AgentStateUpdate{AgentState: state.AgentState{AgentID: agent, State: "done"}})
	for i := 0; i < bus.BufferSize+1; i++ {
		s.eventBus.Publish("notification", nil, map[string]any{})
	}
	close(w.release)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("overflowed stream stayed open, so the lost state is never resnapshotted")
	}

	reconnect := httptest.NewRecorder()
	rctx, rcancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer rcancel()
	s.handleEvents(reconnect, httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(rctx))
	body := reconnect.Body.String()
	if !strings.Contains(body, `"agent_id":"a_lost"`) || !strings.Contains(body, `"state":"done"`) {
		t.Fatalf("reconnect snapshot lacks the lost terminal state:\n%s", body)
	}
}
