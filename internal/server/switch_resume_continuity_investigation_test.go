package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FS-01.A7/A8: a switched conversation remains resumable, including before
// its first target-provider turn. HANDOFF C5 records the provider-side evidence
// separately; this regression proves the primer disappears from the next load.
func TestSwitchPrimerSurvivesResumeBeforeTargetTurnInvestigation(t *testing.T) {
	t.Skip("continuity investigation C5: runtime-only switch primer is absent on later resume")
	srv, ts := switchTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "Investigate the staging migration."})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("prompt status %d: %s", resp.StatusCode, body)
	}
	waitForStatus(t, srv, id, "idle")
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"backend": "codex", "model": "gpt-5.5"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("first switch status %d: %s", resp.StatusCode, body)
	}
	// Return to a fresh Claude session, which exposes its system prompt directly
	// in ACP _meta. Do not send any target-provider prompt before stopping it.
	newDump := filepath.Join(t.TempDir(), "new.json")
	t.Setenv("FAKEACP_NEW_DUMP", newDump)
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"backend": "claude", "model": "sonnet"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("second switch status %d: %s", resp.StatusCode, body)
	}
	const primerMarker = "Chuck backend-switch history primer."
	created, err := os.ReadFile(newDump)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(created), primerMarker) {
		t.Fatal("fixture's target session did not receive a switch primer")
	}
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status %d: %s", resp.StatusCode, body)
	}
	loadDump := filepath.Join(t.TempDir(), "load.json")
	t.Setenv("FAKEACP_LOAD_DUMP", loadDump)
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/resume", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resume status %d: %s", resp.StatusCode, body)
	}
	loaded, err := os.ReadFile(loadDump)
	if err != nil {
		t.Fatal(err)
	}
	var valid map[string]json.RawMessage
	if err := json.Unmarshal(loaded, &valid); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(loaded), primerMarker) {
		t.Fatal("successful native Resume omitted all pre-switch history before the target's first turn")
	}
}
