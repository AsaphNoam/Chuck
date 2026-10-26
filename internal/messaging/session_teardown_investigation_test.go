package messaging

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// INV §4 requires registration teardown to release its MCP sessions. A client
// process can exit without sending MCP DELETE, so token revocation must own
// cleanup; successful initialization alone must not retain it forever.
func TestInvestigationMEM1RevokeReleasesMCPSession(t *testing.T) {
	t.Skip("MEM-1: token revocation leaves MCP transport sessions alive; unskip with the lifecycle fix")
	srv := New(newStore(t), nil)
	srv.RegisterSession("tok-exited", "a_exited", "generation-1")
	t.Cleanup(func() {
		for session := range srv.mcp.Sessions() {
			_ = session.Close()
		}
	})

	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"exited-agent","version":"1"}}}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set(TokenHeader, "tok-exited")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Header().Get("Mcp-Session-Id") == "" {
		t.Fatalf("initialize: HTTP %d, session %q, body %s", w.Code, w.Header().Get("Mcp-Session-Id"), w.Body.String())
	}

	// The HTTP request has finished and the client exits without DELETE.
	srv.Revoke("tok-exited")
	retained := 0
	for range srv.mcp.Sessions() {
		retained++
	}
	if retained != 0 {
		t.Fatalf("revoked generation retained %d MCP sessions; want 0", retained)
	}
}
