package server

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// TestQuotaSnapshotSchedulesAndPostponesRecovery is FS-01.A41 end to end: a
// failure with no reset is scheduled from the snapshot's exhausted window with
// no check, and the pre-check moves a due continuation to a strictly later
// reported reset without starting a turn.
func TestQuotaSnapshotSchedulesAndPostponesRecovery(t *testing.T) {
	first := time.Now().UTC().Add(20 * time.Hour).Truncate(time.Second)
	snapshot := func(reset time.Time) []byte {
		return []byte(`{"rate_limits_available":true,"rate_limits":{"seven_day":{"utilization":100,"resets_at":"` + reset.Format(time.RFC3339) + `"}}}`)
	}
	file := filepath.Join(t.TempDir(), "quota.json")
	if err := os.WriteFile(file, snapshot(first), 0o600); err != nil {
		t.Fatal(err)
	}
	failure := `{"id":"snapshot-quota","revision":1,"category":"limit","severity":"error","title":"quota","actions":[]}`
	t.Setenv("FAKEACP_FAILURE_UPDATE", failure)
	t.Setenv("FAKEACP_FAILURE_RESULT", failure)
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_QUOTA", file)
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/prompt", map[string]string{"text": "first"}); resp.StatusCode != http.StatusAccepted {
		t.Fatalf("prompt = %d: %s", resp.StatusCode, body)
	}
	waitPrompts(t, promptLog, 1)
	var q state.QuotaIntent
	var err error
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if q, err = srv.stateStore.GetQuota(id); err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("quota after interruption: %v", err)
	}
	if q.State != state.QuotaScheduled || q.NextAction != state.QuotaActionContinue || q.ChecksStarted != 0 ||
		q.ResetSource != "snapshot" || q.ResetAt == nil || !q.ResetAt.Equal(first) {
		t.Fatalf("interruption = %+v, want continue at the snapshot reset", q)
	}

	later := first.Add(24 * time.Hour)
	if err := os.WriteFile(file, snapshot(later), 0o600); err != nil {
		t.Fatal(err)
	}
	// Make it due now; the pre-check runs before the continuation starts.
	past := time.Now().UTC().Add(-time.Minute).Format(time.RFC3339)
	if _, err := srv.stateStore.DB().Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, past, id); err != nil {
		t.Fatal(err)
	}
	srv.dispatchQuotaContinuations(context.Background())
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if q, err = srv.stateStore.GetQuota(id); err == nil && q.ResetAt != nil && q.ResetAt.Equal(later) {
			break
		}
	}
	if q.State != state.QuotaScheduled || q.ResetAt == nil || !q.ResetAt.Equal(later) ||
		!q.DueAt.Equal(later.Add(state.QuotaResetBuffer)) || q.ChecksStarted != 0 || q.ReasonCode != "" {
		t.Fatalf("after pre-check = %+v, want the wait moved to %s", q, later)
	}
	if got := promptCount(t, promptLog); got != 1 {
		t.Fatalf("provider prompts = %d, want the pre-check to start nothing", got)
	}

	// A deferred retry (here: busy behind a manual action) re-claims on every
	// dispatch tick; it must not read the provider again (FS-09.R83). A read
	// would report a strictly later reset and postpone instead of starting.
	if err := os.WriteFile(file, snapshot(later.Add(24*time.Hour)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`UPDATE quota_intents SET state='deferred', reason_code='busy', due_at=? WHERE agent_id=?`, past, id); err != nil {
		t.Fatal(err)
	}
	srv.dispatchQuotaContinuations(context.Background())
	waitPrompts(t, promptLog, 2)
}
