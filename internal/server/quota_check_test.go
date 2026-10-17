package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

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
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	before := len(readEvents(t, s, id))
	observeUnknownQuota(t, s, id)
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"check-quota","revision":1,"category":"limit","severity":"error","title":"You've hit your session limit","actions":[]}`)
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
