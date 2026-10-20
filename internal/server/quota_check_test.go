package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

func observeUnknownQuota(t *testing.T, s *Server, id string) state.QuotaIntent {
	t.Helper()
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "unknown-reset-episode", SourceGeneration: s.registry.Generation(id),
		ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: time.Now().UTC().Add(-61 * time.Minute),
		State: state.QuotaUnknownReset, Check: true})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func waitQuota(t *testing.T, s *Server, id string, done func(state.QuotaIntent) bool) state.QuotaIntent {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		q, err := s.stateStore.GetQuota(id)
		if err == nil && done(q) {
			return q
		}
		if time.Now().After(deadline) {
			t.Fatalf("quota did not settle: %+v %v", q, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func promptLines(t *testing.T, log string) []string {
	t.Helper()
	raw, _ := os.ReadFile(log)
	if strings.TrimSpace(string(raw)) == "" {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

// FS-01.A32: a due unknown-reset check runs outside the interrupted
// conversation; only its success lets the ordinary continuation run.
func TestQuotaCheckReadyAuthorizesOneContinuation(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	before := len(readEvents(t, s, id))
	observeUnknownQuota(t, s, id)
	s.dispatchQuotaContinuations(context.Background())
	q := waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.CheckReady && q.State == state.QuotaScheduled })
	if q.ChecksStarted != 1 || q.NextAction != state.QuotaActionContinue || q.ResetAt != nil || q.CheckSession != "" {
		t.Fatalf("ready check = %+v", q)
	}
	lines := promptLines(t, log)
	if len(lines) != 1 || !strings.Contains(lines[0], "Reply with exactly OK") {
		t.Fatalf("prompts after check = %q", lines)
	}
	if got := len(readEvents(t, s, id)); got != before {
		t.Fatalf("check changed the interrupted transcript: %d -> %d events", before, got)
	}
	s.dispatchQuotaContinuations(context.Background())
	waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.State == state.QuotaContinued })
	lines = promptLines(t, log)
	if len(lines) != 2 || strings.Contains(lines[1], "Reply with exactly OK") {
		t.Fatalf("continuation prompts = %q", lines)
	}
}

// A check refused by quota leaves the conversation, transcript and native
// session untouched, reschedules one hour later and spends one check.
func TestQuotaCheckFailureIsQuiet(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	// The check reuses the live original's frozen environment, so the fake's
	// failure must be in place at launch (the original is never prompted here).
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"check-quota","revision":1,"category":"limit","severity":"error","title":"You've hit your session limit","actions":[]}`)
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	before := len(readEvents(t, s, id))
	observeUnknownQuota(t, s, id)
	started := time.Now().UTC()
	s.dispatchQuotaContinuations(context.Background())
	q := waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.ChecksStarted == 1 && q.State == state.QuotaScheduled })
	if q.NextAction != state.QuotaActionCheck || q.ResetAt != nil || q.DueAt == nil || q.DueAt.Before(started.Add(59*time.Minute)) {
		t.Fatalf("after quota check = %+v", q)
	}
	p, _ := s.stateStore.QuotaProjection(id)
	if p.ChecksRemaining != 5 || p.NextAction != "check" {
		t.Fatalf("projection = %+v", p)
	}
	if got := len(readEvents(t, s, id)); got != before {
		t.Fatalf("failed check changed the transcript: %d -> %d events", before, got)
	}
	if lines := promptLines(t, log); len(lines) != 1 || !strings.Contains(lines[0], "Reply with exactly OK") {
		t.Fatalf("prompts = %q, want only the isolated check", lines)
	}
}

// Q7 / FS-01.R49: a check refused with an explicit spend-cap requirement stays
// manual instead of spending the remaining allowance on hourly checks.
func TestQuotaCheckInterventionStaysManual(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"credits:error","revision":1,"category":"limit","severity":"error","title":"You've hit your monthly spend limit · ask your admin to raise it at claude.ai/admin-settings/usage","actions":[]}`)
	s, ts, _ := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	observeUnknownQuota(t, s, id)
	s.dispatchQuotaContinuations(context.Background())
	q := waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.ChecksStarted == 1 && q.State != state.QuotaClaimed })
	if q.State != state.QuotaUnknownReset || q.NextAction != state.QuotaActionContinue || q.DueAt != nil || q.CheckReady {
		t.Fatalf("intervention check = %+v", q)
	}
}

// Q3 / TS-10.R46: a check tests the selection the interrupted work actually
// runs under. A live original keeps its launch selection despite catalog
// edits; a continuation whose selection changed after its check stays manual.
func TestQuotaCheckFollowsInterruptedSelection(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	remap := func(modelID string) {
		t.Helper()
		a, err := s.stateStore.ReadAgent(id)
		if err != nil {
			t.Fatal(err)
		}
		b, err := s.configStore.ReadBackends()
		if err != nil {
			t.Fatal(err)
		}
		be := b.Backends[a.Backend]
		model := be.Models[a.Model]
		model.Model = modelID
		be.Models[a.Model] = model
		b.Backends[a.Backend] = be
		if err := s.configStore.WriteBackends(b); err != nil {
			t.Fatal(err)
		}
	}
	before, err := s.quotaCheckSpec(id)
	if err != nil {
		t.Fatal(err)
	}
	remap("another-model-for-new-launches")
	if live, err := s.quotaCheckSpec(id); err != nil || live.ModelID != before.ModelID || len(live.Env) == 0 {
		t.Fatalf("live check model = %q (%v), original %q", live.ModelID, err, before.ModelID)
	}

	// Stopped: a resume continues the frozen contract, so the check probes it
	// too, and the continuation must still match what was checked.
	generation := s.registry.Generation(id)
	if resp, body := post(t, ts.URL+"/api/sessions/"+id+"/stop", nil); resp.StatusCode != 200 {
		t.Fatalf("stop status = %d: %s", resp.StatusCode, body)
	}
	if _, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "stopped-episode", SourceGeneration: generation,
		ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"}, ObservedAt: time.Now().UTC().Add(-61 * time.Minute),
		State: state.QuotaUnknownReset, Check: true}); err != nil {
		t.Fatal(err)
	}
	s.dispatchQuotaContinuations(context.Background())
	ready := waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.CheckReady && q.State == state.QuotaScheduled })
	if !s.quotaCheckStillMatches(ready) {
		t.Fatal("unchanged stopped selection does not match its check")
	}
	remap("yet-another-model")
	if !s.quotaCheckStillMatches(ready) {
		t.Fatal("a catalogue edit changed the frozen continuation selection")
	}
	a, err := s.stateStore.ReadAgent(id)
	if err != nil {
		t.Fatal(err)
	}
	a.Effort = "changed-after-check"
	if err := s.stateStore.WriteAgent(a); err != nil {
		t.Fatal(err)
	}
	s.dispatchQuotaContinuations(context.Background())
	q := waitQuota(t, s, id, func(q state.QuotaIntent) bool { return q.State == state.QuotaNeedsAttention })
	if q.ReasonCode != "check_unsupported" {
		t.Fatalf("changed selection after check = %+v", q)
	}
	if lines := promptLines(t, log); len(lines) != 1 || !strings.Contains(lines[0], "Reply with exactly OK") {
		t.Fatalf("prompts = %q, want only the isolated check", lines)
	}
}

// Q3 / TS-10.R47: cleanup retries the exact probe with the selection that
// created it, not whatever the catalog says now.
func TestQuotaCleanupUsesCreatingSelection(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	s, ts, _ := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	q := observeUnknownQuota(t, s, id)
	if err := s.stateStore.ClaimQuota(id, q.EpisodeID, q.Revision, "token"); err != nil {
		t.Fatal(err)
	}
	q, _ = s.stateStore.GetQuota(id)
	if err := s.stateStore.StartQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, "codex-acp"); err != nil {
		t.Fatal(err)
	}
	q, _ = s.stateStore.GetQuota(id)
	if err := s.stateStore.RecordQuotaCheckSession(id, q.EpisodeID, q.Revision, q.AttemptToken, "probe-exact"); err != nil {
		t.Fatal(err)
	}
	q, _ = s.stateStore.GetQuota(id)
	if err := s.stateStore.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, state.QuotaCheckResult{Outcome: state.QuotaCheckFailed}); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	delLog := dir + "/delete.json"
	s.quotaCleanupSpecs = map[string]runtime.QuotaCheckSpec{id + "\x00probe-exact": {BackendType: "codex-acp",
		Env: []string{"HOME=" + dir, "FAKEACP_SCENARIO=stream_text", "FAKEACP_CAPS=1", "FAKEACP_DELETE_LOG=" + delLog}}}
	s.cleanupQuotaChecks(context.Background())
	if deleted, _ := os.ReadFile(delLog); !strings.Contains(string(deleted), `"probe-exact"`) {
		t.Fatalf("cleanup did not use the creating selection: %s", deleted)
	}
	if q, _ := s.stateStore.GetQuota(id); q.CheckSession != "" {
		t.Fatalf("cleanup reference unresolved: %+v", q)
	}
}

// Disabling Auto continue cancels a pending check before any provider work.
func TestQuotaCheckDisableCancelsBeforeProviderWork(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	observeUnknownQuota(t, s, id)
	if w := quotaHTTP(t, s, "PUT", "/api/config", `{"auto_continue":false}`); w.Code != 200 {
		t.Fatalf("disable: %d %s", w.Code, w.Body.String())
	}
	s.dispatchQuotaContinuations(context.Background())
	time.Sleep(200 * time.Millisecond)
	q, _ := s.stateStore.GetQuota(id)
	if q.State != state.QuotaCancelled || q.ChecksStarted != 0 || len(promptLines(t, log)) != 0 {
		t.Fatalf("disabled check = %+v prompts=%d", q, len(promptLines(t, log)))
	}
}
