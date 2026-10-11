package state

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func quotaFixture(t *testing.T, st *Store, id, episode string, due time.Time) QuotaIntent {
	t.Helper()
	now := due.Add(-time.Minute)
	if err := st.WriteAgent(testAgent(id, now)); err != nil {
		t.Fatal(err)
	}
	got, err := st.ObserveQuota(QuotaObservation{AgentID: id, EpisodeID: episode, SourceGeneration: "generation-1", ExecutingTurn: "turn-1", Owner: QuotaOwner{Kind: "ordinary"}, ObservedAt: now, ResetAt: &due, DueAt: &due, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestQuotaObservationIsBoundedIdempotentAndOwnerClosed(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(time.Hour).UTC().Truncate(time.Second)
	first := quotaFixture(t, st, "quota_a", "episode-1", due)
	if _, err := st.ObserveQuota(QuotaObservation{AgentID: first.AgentID, EpisodeID: first.EpisodeID, SourceGeneration: "changed", ExecutingTurn: "changed", Owner: QuotaOwner{Kind: "ordinary"}, ObservedAt: due, State: QuotaUnknownReset}); err != nil {
		t.Fatal(err)
	}
	second, err := st.GetQuota(first.AgentID)
	if err != nil || second.Revision != first.Revision || second.SourceGeneration != first.SourceGeneration {
		t.Fatalf("same episode changed: %+v err=%v", second, err)
	}
	if _, err := validateQuotaOwner(QuotaOwner{Kind: "ordinary", TaskID: "unexpected"}); !errors.Is(err, ErrQuotaOwner) {
		t.Fatalf("open owner accepted: %v", err)
	}
	if _, err := validateQuotaOwner(QuotaOwner{Kind: "room", RoomID: "r", AttemptID: "a", Turn: "other"}); !errors.Is(err, ErrQuotaOwner) {
		t.Fatalf("unknown room turn accepted: %v", err)
	}
	if _, err := st.DueQuotaIntents(due, MaxQuotaDueBatch+1); !errors.Is(err, ErrQuotaConflict) {
		t.Fatalf("oversized due batch err=%v", err)
	}
}

func TestQuotaManualTurnBindsAndResolvesOnlyExactCurrentEpisode(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(time.Hour).UTC().Truncate(time.Second)
	q := quotaFixture(t, st, "quota_manual_binding", "episode-manual", due)
	if err := st.BindQuotaManualTurn(q.AgentID, "manual-generation", "manual-turn"); err != nil {
		t.Fatal(err)
	}
	bound, err := st.GetQuota(q.AgentID)
	if err != nil || bound.ManualGeneration != "manual-generation" || bound.ManualTurn != "manual-turn" || bound.Revision != q.Revision+1 {
		t.Fatalf("manual binding: %+v err=%v", bound, err)
	}
	if err := st.ResolveQuota(q.AgentID, "manual-generation", "stale-turn"); err != nil {
		t.Fatal(err)
	}
	stillPending, _ := st.GetQuota(q.AgentID)
	if stillPending.State != QuotaScheduled {
		t.Fatalf("stale manual completion cleared episode: %+v", stillPending)
	}
	if err := st.ResolveQuota(q.AgentID, "manual-generation", "manual-turn"); err != nil {
		t.Fatal(err)
	}
	resolved, _ := st.GetQuota(q.AgentID)
	if resolved.State != QuotaContinued {
		t.Fatalf("exact manual completion did not clear episode: %+v", resolved)
	}
}

func TestQuotaManualTurnReassignsActiveTaskAndResolvesCancelledEpisode(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(time.Hour).UTC().Truncate(time.Second)
	q := quotaFixture(t, st, "quota_manual_task", "episode-task", due)
	task := newTask(t, st, "proj", "quota owner task")
	if _, err := st.DB().Exec(`UPDATE tasks SET assigned_agent_id=?, assigned_generation=?, state=? WHERE task_id=?`, q.AgentID, "generation-1", TaskRunning, task.TaskID); err != nil {
		t.Fatal(err)
	}
	owned, err := st.ObserveQuota(QuotaObservation{AgentID: q.AgentID, EpisodeID: "episode-task-owned", SourceGeneration: "generation-1", ExecutingTurn: "source-task-turn", Owner: QuotaOwner{Kind: "task", TaskID: task.TaskID}, ObservedAt: due.Add(-time.Minute), ResetAt: &due, DueAt: &due, State: QuotaScheduled})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CancelQuota(owned.AgentID, owned.EpisodeID, owned.Revision, ""); err != nil {
		t.Fatal(err)
	}
	if err := st.BindQuotaManualTurn(q.AgentID, "manual-task-generation", "manual-task-turn"); err != nil {
		t.Fatal(err)
	}
	updatedTask, err := st.ReadTask(task.TaskID)
	if err != nil || updatedTask.AssignedGeneration != "manual-task-generation" || updatedTask.ExecutionTurn != "manual-task-turn" {
		t.Fatalf("task assignment after manual bind: %+v err=%v", updatedTask, err)
	}
	if err := st.ResolveQuota(q.AgentID, "manual-task-generation", "manual-task-turn"); err != nil {
		t.Fatal(err)
	}
	resolved, err := st.GetQuota(q.AgentID)
	if err != nil || resolved.EpisodeID != owned.EpisodeID || resolved.State != QuotaContinued {
		t.Fatalf("cancelled episode after exact manual success: %+v err=%v", resolved, err)
	}
}

func TestQuotaClaimCASAndCancellationRace(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(-time.Minute).UTC().Truncate(time.Second)
	q := quotaFixture(t, st, "quota_b", "episode-1", due)
	const contenders = 12
	var wg sync.WaitGroup
	twins := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			twins <- st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, string(rune('a'+i)))
		}(i)
	}
	wg.Wait()
	close(twins)
	twinsWon := 0
	for err := range twins {
		if err == nil {
			twinsWon++
		} else if !errors.Is(err, ErrQuotaConflict) {
			t.Fatalf("claim err=%v", err)
		}
	}
	if twinsWon != 1 {
		t.Fatalf("claims won=%d, want 1", twinsWon)
	}
	claimed, err := st.GetQuota(q.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.CancelQuota(q.AgentID, q.EpisodeID, claimed.Revision-1, claimed.AttemptToken); !errors.Is(err, ErrQuotaConflict) {
		t.Fatalf("stale cancel err=%v", err)
	}
	if err := st.CancelQuota(q.AgentID, q.EpisodeID, claimed.Revision, claimed.AttemptToken); err != nil {
		t.Fatal(err)
	}
	if err := st.MarkQuotaProviderStarted(q.AgentID, q.EpisodeID, claimed.Revision, claimed.AttemptToken, "g2", "t2"); !errors.Is(err, ErrQuotaConflict) {
		t.Fatalf("stale callback err=%v", err)
	}
}

func TestQuotaAdmissionAndCancelShareRevisionCAS(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(-time.Minute).UTC().Truncate(time.Second)
	q := quotaFixture(t, st, "quota_cancel_race", "episode", due)
	start := make(chan struct{})
	results := make(chan error, 2)
	go func() { <-start; results <- st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "race-token") }()
	go func() { <-start; results <- st.CancelQuota(q.AgentID, q.EpisodeID, q.Revision, "") }()
	close(start)
	a, b := <-results, <-results
	if (a == nil) == (b == nil) {
		t.Fatalf("expected exactly one CAS winner, claim=%v cancel=%v", a, b)
	}
	if a != nil && !errors.Is(a, ErrQuotaConflict) {
		t.Fatalf("claim error=%v", a)
	}
	if b != nil && !errors.Is(b, ErrQuotaConflict) {
		t.Fatalf("cancel error=%v", b)
	}
}

func TestQuotaRestartRecoverySeparatesUncertainDelivery(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(-time.Minute).UTC().Truncate(time.Second)
	pre := quotaFixture(t, st, "quota_pre", "episode-1", due)
	if err := st.ClaimQuota(pre.AgentID, pre.EpisodeID, pre.Revision, "pre-token"); err != nil {
		t.Fatal(err)
	}
	started := quotaFixture(t, st, "quota_started", "episode-1", due)
	if err := st.ClaimQuota(started.AgentID, started.EpisodeID, started.Revision, "started-token"); err != nil {
		t.Fatal(err)
	}
	current, _ := st.GetQuota(started.AgentID)
	if err := st.MarkQuotaProviderStarted(started.AgentID, started.EpisodeID, current.Revision, current.AttemptToken, "g2", "t2"); err != nil {
		t.Fatal(err)
	}
	if err := st.RecoverQuotaClaims(); err != nil {
		t.Fatal(err)
	}
	pre, _ = st.GetQuota(pre.AgentID)
	started, _ = st.GetQuota(started.AgentID)
	if pre.State != QuotaScheduled || pre.AttemptToken != "" {
		t.Fatalf("pre-effect claim not restored: %+v", pre)
	}
	if started.State != QuotaNeedsAttention || started.ReasonCode != "uncertain_delivery" {
		t.Fatalf("uncertain start replayed: %+v", started)
	}
}

func TestQuotaDueOrderNoticeReceiptAndAgentDeletion(t *testing.T) {
	st, _ := newTestStore(t)
	now := timeNow().UTC().Truncate(time.Second)
	late := quotaFixture(t, st, "quota_late", "ep", now.Add(-time.Minute))
	early := quotaFixture(t, st, "quota_early", "ep", now.Add(-2*time.Minute))
	items, err := st.DueQuotaIntents(now, 2)
	if err != nil || len(items) != 2 || items[0].AgentID != early.AgentID {
		t.Fatalf("due order=%v err=%v", items, err)
	}
	if _, err := st.CancelPendingQuotas(); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetQuota(late.AgentID); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteAgent(late.AgentID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetQuota(late.AgentID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted agent quota retained: %v", err)
	}
	_ = early
}

func TestQuotaDeferralNoticeAndExactTupleClear(t *testing.T) {
	st, _ := newTestStore(t)
	due := timeNow().Add(-time.Minute).UTC().Truncate(time.Second)
	q := quotaFixture(t, st, "quota_flow", "episode", due)
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "token"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(q.AgentID)
	if err := st.DeferQuota(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, due, "busy"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(q.AgentID)
	if err := st.ClaimQuota(q.AgentID, q.EpisodeID, q.Revision, "token-2"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(q.AgentID)
	if err := st.MarkQuotaProviderStarted(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "g2", "t2"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(q.AgentID)
	if err := st.MarkQuotaNotice(q.AgentID, q.EpisodeID, q.Revision, q.AttemptToken, "quota-notice-key"); err != nil {
		t.Fatal(err)
	}
	q, _ = st.GetQuota(q.AgentID)
	if q.NoticeKey != "quota-notice-key" {
		t.Fatalf("notice receipt missing: %+v", q)
	}
	if cleared, err := st.ClearQuota(q.AgentID, q.SourceGeneration, q.ExecutingTurn); err != nil || !cleared {
		t.Fatalf("exact source tuple clear = %v, %v", cleared, err)
	}
}
