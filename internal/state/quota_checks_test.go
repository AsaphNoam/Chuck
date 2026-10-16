package state

import (
	"errors"
	"testing"
	"time"
)

func observeCheck(t *testing.T, st *Store, id, episode, generation, turn string, observed time.Time) QuotaIntent {
	t.Helper()
	q, err := st.ObserveQuota(QuotaObservation{AgentID: id, EpisodeID: episode, SourceGeneration: generation, ExecutingTurn: turn,
		Owner: QuotaOwner{Kind: "ordinary"}, ObservedAt: observed, State: QuotaUnknownReset, Check: true})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

// claimCheck admits a due check and charges it, returning the started claim.
func claimCheck(t *testing.T, st *Store, id string) QuotaIntent {
	t.Helper()
	q, err := st.GetQuota(id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, formatTime(timeNow().Add(-time.Second)), id); err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(id, q.EpisodeID, q.Revision, "check-token-"+q.EpisodeID); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.StartQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, "codex-acp"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	return q
}

// FS-01.A32 / TS-10.R45–R46: an unknown reset schedules an hourly check
// without inventing a reset; quota results reschedule until six are spent.
func TestQuotaCheckChainChargesAllowanceAndExhausts(t *testing.T) {
	st, _ := newTestStore(t)
	id := "quota_check_chain"
	observed := timeNow().Truncate(time.Second)
	if err := st.WriteAgent(testAgent(id, observed)); err != nil {
		t.Fatal(err)
	}
	q := observeCheck(t, st, id, "episode-1", "g1", "t1", observed)
	if q.State != QuotaScheduled || q.NextAction != QuotaActionCheck || q.ResetAt != nil || q.DueAt == nil || !q.DueAt.Equal(observed.Add(time.Hour)) || q.ChecksStarted != 0 {
		t.Fatalf("observation = %+v, want hourly check with no reset", q)
	}
	p, _ := st.QuotaProjection(id)
	if p.NextAction != "check" || p.ChecksRemaining != 6 || p.ResetAt != nil {
		t.Fatalf("projection = %+v", p)
	}
	for i := 1; i <= MaxQuotaChecks; i++ {
		q = claimCheck(t, st, id)
		if q.ChecksStarted != i || !q.CheckStarted {
			t.Fatalf("check %d charged %+v", i, q)
		}
		if err := st.MarkQuotaProviderStarted(id, q.EpisodeID, q.Revision, q.AttemptToken, "g2", "t2"); !errors.Is(err, ErrQuotaConflict) {
			t.Fatalf("check receipt satisfied original-work provider start: %v", err)
		}
		if err := st.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, QuotaCheckResult{Outcome: QuotaCheckQuota, Cleaned: true}); err != nil {
			t.Fatal(err)
		}
		q, _ = st.GetQuota(id)
		if i < MaxQuotaChecks && (q.State != QuotaScheduled || q.NextAction != QuotaActionCheck || q.ResetAt != nil) {
			t.Fatalf("after check %d: %+v", i, q)
		}
	}
	if q.State != QuotaNeedsAttention || q.ReasonCode != "check_exhausted" || q.DueAt != nil {
		t.Fatalf("exhausted = %+v", q)
	}
	if _, err := st.db.Exec(`UPDATE quota_intents SET checks_started=7 WHERE agent_id=?`, id); err == nil {
		t.Fatal("SQLite accepted more than six checks")
	}
}

// TS-10.R46: a ready check grants one continuation; renewed quota from that
// exact continuation holds rather than looping (FS-01.R51). A known reset
// from a check switches to continue after the buffer.
func TestQuotaCheckReadyThenRenewedQuotaHolds(t *testing.T) {
	st, _ := newTestStore(t)
	id := "quota_check_ready"
	observed := timeNow().Truncate(time.Second)
	if err := st.WriteAgent(testAgent(id, observed)); err != nil {
		t.Fatal(err)
	}
	observeCheck(t, st, id, "episode-1", "g1", "t1", observed)
	q := claimCheck(t, st, id)
	if err := st.RecordQuotaCheckSession(id, q.EpisodeID, q.Revision, q.AttemptToken, "probe-session"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, QuotaCheckResult{Outcome: QuotaCheckReady, Cleaned: true}); err != nil {
		t.Fatal(err)
	}
	if err := st.ResolveQuotaCheckCleanup(id, "probe-session"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if q.State != QuotaScheduled || q.NextAction != QuotaActionContinue || !q.CheckReady || q.ResetAt != nil || q.CheckSession != "" {
		t.Fatalf("ready = %+v", q)
	}
	if err := st.ClaimQuota(id, q.EpisodeID, q.Revision, "continue-token"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.MarkQuotaProviderStarted(id, q.EpisodeID, q.Revision, "continue-token", "g2", "t2"); err != nil {
		t.Fatal(err)
	}
	renewed := observeCheck(t, st, id, "episode-renewed", "g2", "t2", timeNow())
	if renewed.State != QuotaNeedsAttention || renewed.ReasonCode != "check_ready_but_quota" || renewed.ChainID != "episode-1" || renewed.ChecksStarted != 1 {
		t.Fatalf("renewed after ready = %+v", renewed)
	}

	// A different interruption starts a fresh chain and allowance.
	fresh := observeCheck(t, st, id, "episode-other", "g3", "t3", timeNow())
	if fresh.ChainID != "episode-other" || fresh.ChecksStarted != 0 || fresh.NextAction != QuotaActionCheck {
		t.Fatalf("fresh chain = %+v", fresh)
	}
	q = claimCheck(t, st, id)
	reset := timeNow().Add(2 * time.Hour).Truncate(time.Second)
	if err := st.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, QuotaCheckResult{Outcome: QuotaCheckQuota, ResetAt: &reset, ResetSource: "text", Cleaned: true}); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if q.NextAction != QuotaActionContinue || q.ResetAt == nil || !q.ResetAt.Equal(reset) || !q.DueAt.Equal(reset.Add(QuotaResetBuffer)) || q.ChecksStarted != 1 {
		t.Fatalf("check reset = %+v", q)
	}
}

// TS-10.R47: a failed cleanup holds and keeps its exact reference across a
// new episode; restart turns a started check into check_interrupted.
func TestQuotaCheckCleanupAndRestart(t *testing.T) {
	st, _ := newTestStore(t)
	id := "quota_check_cleanup"
	observed := timeNow().Truncate(time.Second)
	if err := st.WriteAgent(testAgent(id, observed)); err != nil {
		t.Fatal(err)
	}
	observeCheck(t, st, id, "episode-1", "g1", "t1", observed)
	q := claimCheck(t, st, id)
	if err := st.RecordQuotaCheckSession(id, q.EpisodeID, q.Revision, q.AttemptToken, "probe-1"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.FinishQuotaCheck(id, q.EpisodeID, q.Revision, q.AttemptToken, QuotaCheckResult{Outcome: QuotaCheckReady, Cleaned: false}); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if q.State != QuotaNeedsAttention || q.ReasonCode != "check_cleanup_failed" || q.CheckReady || q.CheckSession != "probe-1" {
		t.Fatalf("cleanup failure = %+v", q)
	}
	q = observeCheck(t, st, id, "episode-2", "g2", "t2", timeNow())
	if q.CheckSession != "probe-1" || q.CheckBackend != "codex-acp" {
		t.Fatalf("new episode dropped cleanup reference: %+v", q)
	}
	if _, err := st.db.Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, formatTime(timeNow().Add(-time.Second)), id); err != nil {
		t.Fatal(err)
	}
	if err := st.ClaimQuota(id, q.EpisodeID, q.Revision, "tok"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.StartQuotaCheck(id, q.EpisodeID, q.Revision, "tok", "codex-acp"); !errors.Is(err, ErrQuotaConflict) {
		t.Fatalf("second check started with an unresolved cleanup reference: %v", err)
	}
	pending, err := st.PendingQuotaCheckCleanups(MaxQuotaDueBatch)
	if err != nil || len(pending) != 1 || pending[0].CheckSession != "probe-1" {
		t.Fatalf("pending cleanups = %+v, %v", pending, err)
	}
	if err := st.ResolveQuotaCheckCleanup(id, "probe-1"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if err := st.StartQuotaCheck(id, q.EpisodeID, q.Revision, "tok", "codex-acp"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverQuotaClaims(); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(id)
	if q.State != QuotaNeedsAttention || q.ReasonCode != "check_interrupted" || q.ChecksStarted != 1 || q.DueAt != nil {
		t.Fatalf("restart = %+v", q)
	}
}

// TS-02.R50: migration keeps old states inert, starts allowance at zero and
// raises a known-reset due only to reset + 60 seconds.
func TestQuotaCheckMigrationFromV53(t *testing.T) {
	home := t.TempDir()
	s, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	now := timeNow().Truncate(time.Second)
	for _, id := range []string{"a_unknown", "a_known", "a_later"} {
		if err := s.WriteAgent(testAgent(id, now)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(`DROP TABLE quota_intents; DELETE FROM schema_migrations WHERE version = 54`); err != nil {
		t.Fatal(err)
	}
	for _, m := range migrations {
		if m.version == 49 {
			if _, err := s.db.Exec(m.sql); err != nil {
				t.Fatal(err)
			}
		}
	}
	reset := now.Add(time.Hour)
	later := reset.Add(10 * time.Minute)
	insert := `INSERT INTO quota_intents(agent_id,episode_id,revision,source_generation,executing_turn,owner_json,observed_at,reset_at,due_at,state,updated_at)
VALUES(?,?,1,'g','t','{"kind":"ordinary"}',?,?,?,?,?)`
	for _, row := range [][]any{
		{"a_unknown", "e1", formatTime(now), nil, nil, "unknown_reset", formatTime(now)},
		{"a_known", "e2", formatTime(now), formatTime(reset), formatTime(reset), "scheduled", formatTime(now)},
		{"a_later", "e3", formatTime(now), formatTime(reset), formatTime(later), "scheduled", formatTime(now)},
	} {
		if _, err := s.db.Exec(insert, row...); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	unknown, _ := s.GetQuota("a_unknown")
	if unknown.State != QuotaUnknownReset || unknown.DueAt != nil || unknown.NextAction != QuotaActionContinue || unknown.ChecksStarted != 0 || unknown.ChainID != "e1" {
		t.Fatalf("old unknown row = %+v, want inert", unknown)
	}
	known, _ := s.GetQuota("a_known")
	if known.DueAt == nil || !known.DueAt.Equal(reset.Add(time.Minute)) {
		t.Fatalf("known due = %v, want reset + 60s", known.DueAt)
	}
	kept, _ := s.GetQuota("a_later")
	if kept.DueAt == nil || !kept.DueAt.Equal(later) {
		t.Fatalf("later due = %v, want unchanged", kept.DueAt)
	}
	p, _ := s.QuotaProjection("a_known")
	if p.NextAction != "continue" || p.ChecksRemaining != 6 {
		t.Fatalf("projection = %+v", p)
	}
}
