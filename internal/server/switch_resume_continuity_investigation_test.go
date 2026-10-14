package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A7/A8: a switched conversation remains resumable, including before
// its first target-provider turn. HANDOFF C5 records the provider-side evidence
// separately; this regression proves the bridge survives later native loads.
func TestSwitchPrimerSurvivesResumeBeforeTargetTurnInvestigation(t *testing.T) {
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
	snapBeforeStop, err := srv.stateStore.ReadSession(id)
	if err != nil || snapBeforeStop.LastContextPct != 0 || snapBeforeStop.LastContext != nil {
		t.Fatalf("fresh handoff retained source usage: %+v %v", snapBeforeStop, err)
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
	status, err := srv.stateStore.ReadStatus(id)
	if err != nil || status.ContextPct != 0 || status.ContextCounts != nil {
		t.Fatalf("immediate resume restored source usage: %+v %v", status, err)
	}
	// Native model changes and completed target turns do not retire the bridge.
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"model": "opus"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("model switch status %d: %s", resp.StatusCode, body)
	}
	loaded, err = os.ReadFile(loadDump)
	if err != nil || strings.Count(string(loaded), primerMarker) != 1 {
		t.Fatalf("model switch bridge missing or stacked: %s, %v", loaded, err)
	}
	// Compaction may alter native history; it does not clear the independently owned bridge.
	t.Setenv("FAKEACP_SCENARIO", "notice_flow")
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"model": "sonnet"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch before compaction: %d %s", resp.StatusCode, body)
	}
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "Continue the migration."})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("target prompt: %d %s", resp.StatusCode, body)
	}
	waitForStatus(t, srv, id, "idle")
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop after target turn: %d %s", resp.StatusCode, body)
	}
	// Replace all in-process server/runtime objects while reopening the same database.
	reopened, err := state.Open(srv.configStore.Home())
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	registry := runtime.NewRegistry(reopened)
	fresh := New(srv.configStore, reopened, registry, srv.cfg, srv.log)
	registry.Chat().SetCommand(buildFakeACP(t))
	defer registry.Shutdown(context.Background())
	if ae := fresh.resumeSession(context.Background(), id, resumeOverride{}); ae != nil {
		t.Fatal(ae)
	}
	loaded, err = os.ReadFile(loadDump)
	if err != nil || strings.Count(string(loaded), primerMarker) != 1 {
		t.Fatalf("cold resume bridge missing or stacked: %s, %v", loaded, err)
	}
}

// FS-01.A7/A8: explicit model choice outlives its launch inheritance provenance.
func TestExplicitSwitchModelSurvivesResume(t *testing.T) {
	srv, ts := switchTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET launch_config_json = ? WHERE agent_id = ?`, `{"version":1,"native_inherited":true,"generation":"original"}`, id); err != nil {
		t.Fatal(err)
	}
	resp, body := post(t, ts.URL+"/api/sessions/"+id+"/switch-runtime", map[string]string{"model": "opus"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("switch: %d %s", resp.StatusCode, body)
	}
	snap, err := srv.stateStore.ReadSession(id)
	if err != nil || frozenModelInherited(snap.LaunchConfig) || !strings.Contains(string(snap.LaunchConfig), `"native_inherited":true`) || !strings.Contains(string(snap.LaunchConfig), `"generation":"original"`) {
		t.Fatalf("current choice/provenance not preserved: %s %v", snap.LaunchConfig, err)
	}
	loadDump := filepath.Join(t.TempDir(), "load.json")
	t.Setenv("FAKEACP_LOAD_DUMP", loadDump)
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop: %d %s", resp.StatusCode, body)
	}
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/resume", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resume: %d %s", resp.StatusCode, body)
	}
	loaded, err := os.ReadFile(loadDump)
	backends, _ := srv.configStore.ReadBackends()
	if err != nil || !strings.Contains(string(loaded), backends.Backends["claude"].Models["opus"].Model) {
		t.Fatalf("resume omitted explicit model: %s %v", loaded, err)
	}
}

// FS-01.A7: stopped identity overrides cannot bypass the switch handoff policy.
func TestResumeRejectsIdentityOverridesAndNativeLoadLoss(t *testing.T) {
	srv, ts := switchTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	prior := runningSessionID(t, srv, id)
	resp, body := post(t, ts.URL+"/api/sessions/"+id+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop: %d %s", resp.StatusCode, body)
	}
	for _, request := range []map[string]string{{"backend": "codex", "model": "gpt-5.5"}, {"model": "opus"}} {
		resp, body = post(t, ts.URL+"/api/sessions/"+id+"/resume", request)
		if resp.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("override resume: %d %s", resp.StatusCode, body)
		}
	}
	t.Setenv("FAKEACP_LOAD_REJECT_DATA", `{"detail":"missing rollout"}`)
	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/resume", nil)
	if resp.StatusCode < 400 {
		t.Fatalf("native loss returned success: %d %s", resp.StatusCode, body)
	}
	snap, err := srv.stateStore.ReadSession(id)
	if err != nil || snap.LastSessionID != prior || snap.Model != "sonnet" || snap.Backend != "claude" {
		t.Fatalf("failed resume changed conversation: %+v %v", snap, err)
	}
	if _, err := srv.stateStore.ReadRunning(id); err != state.ErrNotFound {
		t.Fatalf("failed resume left running agent: %v", err)
	}
}

// FS-01.A9/INV §5: switch owns the lifecycle before its first identity read.
func TestSwitchClaimsBeforeReadingPartialTarget(t *testing.T) {
	srv, ts := switchTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	db := srv.stateStore.DB()
	db.SetMaxOpenConns(1)
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	done := make(chan int, 1)
	go func() {
		req := newLocalRequest(http.MethodPost, ts.URL+"/api/sessions/"+id+"/switch-runtime", strings.NewReader(`{"model":"opus"}`))
		req.SetPathValue("id", id)
		response := httptest.NewRecorder()
		srv.handleSwitchRuntime(response, req)
		done <- response.Code
	}()
	deadline := time.Now().Add(2 * time.Second)
	for !srv.lifecycleInFlight(id) && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	claimed := srv.lifecycleInFlight(id)
	// Supply a newer untouched field while the read is waiting on the DB connection.
	_, err = conn.ExecContext(context.Background(), `UPDATE agents SET name = 'Latest name' WHERE agent_id = ?`, id)
	_ = conn.Close()
	if err != nil {
		t.Fatal(err)
	}
	if status := <-done; status != http.StatusOK {
		t.Fatalf("switch: %d", status)
	}
	if !claimed {
		t.Fatal("switch read identity before acquiring lifecycle claim")
	}
	a, err := srv.stateStore.ReadAgent(id)
	if err != nil || a.Name != "Latest name" || a.Model != "opus" {
		t.Fatalf("partial switch overwrote latest identity: %+v %v", a, err)
	}
}

// FS-01.A9: rollback restores the bridge/provenance even after target metadata writes.
func TestSwitchRollbackRestoresPreviousBridge(t *testing.T) {
	srv, ts := switchTestServer(t)
	t.Setenv("FAKEACP_LOAD_EMPTY", "1")
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	const bridge = "previous handoff instructions"
	const provenance = `{"native_inherited":true,"current_model_explicit":true}`
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET history_primer = ?, launch_config_json = ? WHERE agent_id = ?`, bridge, provenance, id); err != nil {
		t.Fatal(err)
	}
	snap, err := srv.stateStore.ReadSession(id)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := srv.stateStore.ReadAgent(id)
	prior := runningSessionID(t, srv, id)
	resp, body := post(t, ts.URL+"/api/sessions/"+id+"/stop", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("stop: %d %s", resp.StatusCode, body)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET history_primer = 'target bridge', launch_config_json = '{}' WHERE agent_id = ?`, id); err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	srv.rollbackSwitch(context.Background(), recorder, a, prior, snap, errors.New("target startup failed"))
	if recorder.Code < 400 {
		t.Fatalf("rollback unexpectedly returned success: %s", recorder.Body.String())
	}
	got, err := srv.stateStore.ReadSession(id)
	if err != nil || got.HistoryPrimer != bridge || string(got.LaunchConfig) != provenance || runningSessionID(t, srv, id) != prior {
		t.Fatalf("rollback lost previous continuity: %+v %v", got, err)
	}
}
