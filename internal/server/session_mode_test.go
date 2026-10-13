package server

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/runtime"
)

func getModes(t *testing.T, url string) (int, runtime.SessionModes, []byte) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var modes runtime.SessionModes
	_ = json.Unmarshal(body, &modes)
	return resp.StatusCode, modes, body
}

// FS-03.A67, TS-03.R70 — the live session's advertised modes round-trip
// through the read and set routes; an unoffered mode is a typed invalid_field
// and the set route shares the per-agent lifecycle claim.
func TestSessionModeRoutes(t *testing.T) {
	t.Setenv("FAKEACP_MODE_OPTIONS", "default,acceptEdits,plan")
	srv, ts, launched := newLiveConfigServer(t)
	id := launched.Agent.AgentID
	base := ts.URL + "/api/sessions/" + id

	status, modes, body := getModes(t, base+"/modes")
	if status != http.StatusOK || modes.Current != "default" || len(modes.Options) != 3 {
		t.Fatalf("GET modes = %d %s", status, body)
	}

	resp, body := post(t, base+"/mode", map[string]any{"mode": "acceptEdits"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST mode = %d %s", resp.StatusCode, body)
	}
	if status, modes, body = getModes(t, base+"/modes"); status != http.StatusOK || modes.Current != "acceptEdits" {
		t.Fatalf("GET after set = %d %s", status, body)
	}

	resp, body = post(t, base+"/mode", map[string]any{"mode": "bypassPermissions"})
	if resp.StatusCode != http.StatusBadRequest || apiErrorCode(t, body) != runtime.CodeInvalidField {
		t.Fatalf("unoffered mode = %d %s", resp.StatusCode, body)
	}
	resp, body = post(t, base+"/mode", map[string]any{})
	if resp.StatusCode != http.StatusUnprocessableEntity || apiErrorCode(t, body) != runtime.CodeValidation {
		t.Fatalf("missing mode = %d %s", resp.StatusCode, body)
	}

	if !srv.claimLifecycle(id) {
		t.Fatal("claimLifecycle should succeed on an idle agent")
	}
	resp, body = post(t, base+"/mode", map[string]any{"mode": "plan"})
	srv.releaseLifecycle(id)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("mode during lifecycle transition = %d %s", resp.StatusCode, body)
	}

	if status, _, body = getModes(t, ts.URL+"/api/sessions/missing/modes"); status != http.StatusNotFound {
		t.Fatalf("unknown agent = %d %s", status, body)
	}
}
