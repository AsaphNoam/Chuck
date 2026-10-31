package messaging

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// INV §4 requires registration teardown to release its MCP sessions. A client
// process can exit without sending MCP DELETE, so token revocation must own
// cleanup for exactly that launch, leaving other launches' sessions alive.
func TestRevokeReleasesOnlyItsMCPSessions(t *testing.T) {
	srv := New(newStore(t), nil)
	srv.RegisterSession("tok-exited", "a_exited", "generation-1")
	srv.RegisterSession("tok-live", "a_live", "generation-1")
	t.Cleanup(func() {
		for session := range srv.mcp.Sessions() {
			_ = session.Close()
		}
	})

	exited := initializeMCP(t, srv, "tok-exited")
	live := initializeMCP(t, srv, "tok-live")

	// The HTTP request has finished and the client exits without DELETE.
	srv.Revoke("tok-exited")
	if got := mcpSessionIDs(srv); len(got) != 1 || !got[live] || got[exited] {
		t.Fatalf("sessions after revoke = %v; want only live %q", got, live)
	}
}

// pausedInitializeResponse holds the SDK's initialize response after the SDK
// session exists, so a teardown can land while the request is in flight.
type pausedInitializeResponse struct {
	*httptest.ResponseRecorder
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (w *pausedInitializeResponse) pause() {
	w.once.Do(func() { close(w.entered); <-w.release })
}

func (w *pausedInitializeResponse) WriteHeader(code int) {
	w.pause()
	w.ResponseRecorder.WriteHeader(code)
}

func (w *pausedInitializeResponse) Write(body []byte) (int, error) {
	w.pause()
	return w.ResponseRecorder.Write(body)
}

// ARCH-NEXT-1: Revoke that lands while initialize is still writing its
// response must not leave the late-created SDK session retained.
func TestRevokeDuringInitializeReleasesTheLateSession(t *testing.T) {
	srv := New(newStore(t), nil)
	srv.RegisterSession("tok-stopping", "a_stopping", "generation-1")
	srv.RegisterSession("tok-live", "a_live", "generation-1")
	t.Cleanup(func() {
		for session := range srv.mcp.Sessions() {
			_ = session.Close()
		}
	})
	live := initializeMCP(t, srv, "tok-live")

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"agent","version":"1"}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(TokenHeader, "tok-stopping")
	w := &pausedInitializeResponse{ResponseRecorder: httptest.NewRecorder(), entered: make(chan struct{}), release: make(chan struct{})}
	done := make(chan struct{})
	go func() { srv.Handler().ServeHTTP(w, req); close(done) }()
	select {
	case <-w.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("no initialize response")
	}
	if got := mcpSessionIDs(srv); len(got) != 2 {
		t.Fatalf("expected the in-flight SDK session alongside live, got %v", got)
	}
	srv.Revoke("tok-stopping")
	close(w.release)
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("initialize did not finish")
	}
	if got := mcpSessionIDs(srv); len(got) != 1 || !got[live] {
		t.Fatalf("sessions after in-flight revoke = %v; want only live %q", got, live)
	}
}

// An unregistered token keeps its session so tool calls get session_unknown.
func TestUnregisteredTokenKeepsItsMCPSession(t *testing.T) {
	srv := New(newStore(t), nil)
	t.Cleanup(func() {
		for session := range srv.mcp.Sessions() {
			_ = session.Close()
		}
	})
	id := initializeMCP(t, srv, "tok-unknown")
	if got := mcpSessionIDs(srv); !got[id] {
		t.Fatalf("unregistered session %q closed; sessions %v", id, got)
	}
}

func initializeMCP(t *testing.T, srv *Server, token string) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"agent","version":"1"}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(TokenHeader, token)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	id := w.Header().Get(mcpSessionHeader)
	if w.Code != http.StatusOK || id == "" {
		t.Fatalf("initialize: HTTP %d, session %q, body %s", w.Code, id, w.Body.String())
	}
	return id
}

func mcpSessionIDs(srv *Server) map[string]bool {
	ids := map[string]bool{}
	for session := range srv.mcp.Sessions() {
		ids[session.ID()] = true
	}
	return ids
}
