package messaging

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
