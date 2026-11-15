package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

func TestParkedContinuationReleasesDispatcherSlot(t *testing.T) {
	s, ts, _ := activationTestServer(t)
	agentID := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	generation := s.registry.Generation(agentID)
	task := newAgentTask(t, s, agentID, "continuation")
	reservation := state.TaskReservation{AgentID: agentID, Generation: generation, AttemptID: "initial", Claim: state.ClaimBorrowed}
	if _, ok, err := s.stateStore.AdmitReadyTask(task.TaskID, reservation, 8); err != nil || !ok {
		t.Fatalf("initial admission = %t, %v", ok, err)
	}
	if _, ok, err := s.stateStore.ConfirmTaskStart(task.TaskID, reservation.AttemptID); err != nil || !ok {
		t.Fatalf("initial confirmation = %t, %v", ok, err)
	}
	// Seed the durable waiting boundary. Preparing the continuation and both
	// subsequent admissions go through the normal store and dispatcher paths.
	if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET state='waiting' WHERE task_id=?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := s.stateStore.PrepareTaskContinuation(agentID); err != nil || !ok {
		t.Fatalf("prepare continuation = %t, %v", ok, err)
	}
	if _, err := s.stateStore.DB().Exec(`UPDATE agents SET archived=1 WHERE agent_id=?`, agentID); err != nil {
		t.Fatal(err)
	}
	s.dispatchReadyTasks(context.Background())
	parked := waitTaskState(t, s, task.TaskID, state.TaskDependencyFailed)
	if !strings.Contains(parked.AttentionReason, "archived") {
		t.Fatalf("unexpected park reason: %s", parked.AttentionReason)
	}
	if _, err := s.stateStore.DB().Exec(`UPDATE agents SET archived=0 WHERE agent_id=?`, agentID); err != nil {
		t.Fatal(err)
	}
	next := newAgentTask(t, s, agentID, "next assignment")
	dispatchUntilTaskState(t, s, next.TaskID, state.TaskRunning)
}

func TestFailedManualTurnDoesNotRearmCancelledQuota(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-cancelled")
	body := fmt.Sprintf(`{"target":"quota_continuation","episode_id":%q,"revision":%d}`, q.EpisodeID, q.Revision)
	w := quotaHTTP(t, s, http.MethodPost, "/api/sessions/"+q.AgentID+"/cancel", body)
	if w.Code != http.StatusAccepted {
		t.Fatalf("cancel quota: %d %s", w.Code, w.Body.String())
	}
	if err := s.stateStore.BindQuotaManualTurn(q.AgentID, "generation", "new-turn"); err != nil {
		t.Fatal(err)
	}
	if err := s.settleQuotaTurn(q.AgentID, "generation", "new-turn", nil, false); err != nil {
		t.Fatal(err)
	}
	got, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || got.State != state.QuotaCancelled {
		t.Fatalf("quota after failed manual turn: state=%s err=%v", got.State, err)
	}
}

// EXEC-7 (TS-01.R49, INV §8): a manual turn ends while another owner bumps the
// quota revision. Settlement re-reads and decides again instead of reporting
// the lost compare-and-swap as a storage failure.
func TestFailedManualTurnRedecidesAfterQuotaConflict(t *testing.T) {
	s := testServer(t, true)
	q := seedQuotaHTTP(t, s, "quota-conflict")
	if err := s.stateStore.BindQuotaManualTurn(q.AgentID, "generation", "manual-turn"); err != nil {
		t.Fatal(err)
	}
	// The first hold attempt loses: a concurrent writer bumps the revision and
	// the hold's own update is skipped.
	if _, err := s.stateStore.DB().Exec(`CREATE TABLE conflict_once (n INTEGER);
CREATE TRIGGER quota_conflict_once BEFORE UPDATE ON quota_intents
WHEN NEW.state='needs_attention' AND NOT EXISTS (SELECT 1 FROM conflict_once)
BEGIN
  INSERT INTO conflict_once VALUES (1);
  UPDATE quota_intents SET revision=revision+1 WHERE agent_id=NEW.agent_id;
  SELECT RAISE(IGNORE);
END;`); err != nil {
		t.Fatal(err)
	}
	if err := s.settleQuotaTurn(q.AgentID, "generation", "manual-turn", nil, false); err != nil {
		t.Fatalf("settle after a lost compare-and-swap: %v", err)
	}
	got, err := s.stateStore.GetQuota(q.AgentID)
	if err != nil || got.State != state.QuotaNeedsAttention || got.ReasonCode != "non_quota_failure" {
		t.Fatalf("quota after re-decided manual turn: %+v err=%v", got, err)
	}
}

func TestQuotaDispatchAdvancesPastMissingRoomOwners(t *testing.T) {
	s, ts, log := activationTestServer(t)
	victim := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	observe := func(id, episode string, due time.Time) {
		t.Helper()
		if _, err := s.stateStore.ObserveQuota(state.QuotaObservation{
			AgentID: id, EpisodeID: episode, SourceGeneration: s.registry.Generation(id),
			ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
			ObservedAt: due.Add(-2 * time.Minute), ResetAt: quotaResetFor(due),
			DueAt: &due, State: state.QuotaScheduled,
		}); err != nil {
			t.Fatal(err)
		}
	}
	var orphans []string
	now := time.Now().UTC()
	for i := 0; i < taskDispatchBatch; i++ {
		id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
		orphans = append(orphans, id)
		observe(id, fmt.Sprintf("orphan-%d", i), now.Add(-10*time.Minute))
		if _, err := s.stateStore.DB().Exec(`UPDATE quota_intents SET owner_json = ? WHERE agent_id = ?`,
			`{"kind":"room","room_id":"tt_deleted_room","attempt_id":"att","turn":"discussion"}`, id); err != nil {
			t.Fatal(err)
		}
	}
	observe(victim, "victim", now.Add(-time.Second))
	s.dispatchQuotaContinuations(context.Background())
	s.dispatchQuotaContinuations(context.Background())
	for _, id := range orphans {
		q, err := s.stateStore.GetQuota(id)
		if err != nil || q.State != state.QuotaNeedsAttention || q.ReasonCode != "owner_unavailable" {
			t.Fatalf("missing owner %s: state=%s reason=%s err=%v", id, q.State, q.ReasonCode, err)
		}
	}
	waitPrompts(t, log, 1)
}

func TestQuotaDispatchRotatesPastClaimContention(t *testing.T) {
	s, ts, log := activationTestServer(t)
	victim := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	now := time.Now().UTC().Truncate(time.Second)
	for i := 0; i < taskDispatchBatch; i++ {
		id := fmt.Sprintf("quota_contended_%02d", i)
		seedQuotaHTTP(t, s, id)
		if _, err := s.stateStore.DB().Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, now.Add(-time.Minute).Format(time.RFC3339), id); err != nil {
			t.Fatal(err)
		}
		release, ok := s.claimQuotaActions(quotaAgentKeys([]string{id}))
		if !ok {
			t.Fatalf("claim %s", id)
		}
		t.Cleanup(release)
	}
	due := now.Add(-time.Second)
	if _, err := s.stateStore.ObserveQuota(state.QuotaObservation{
		AgentID: victim, EpisodeID: "victim", SourceGeneration: s.registry.Generation(victim),
		ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "ordinary"},
		ObservedAt: due.Add(-2 * time.Minute), ResetAt: quotaResetFor(due),
		DueAt: &due, State: state.QuotaScheduled,
	}); err != nil {
		t.Fatal(err)
	}
	s.dispatchQuotaContinuations(context.Background())
	s.dispatchQuotaContinuations(context.Background())
	waitPrompts(t, log, 1)
}

func TestRecoverTasksPropagatesAlreadyParkedPrerequisite(t *testing.T) {
	srv := testServer(t, true)
	seedPipelineRun(t, srv, "pr_failed")
	first := newDependentTask(t, srv, "first", state.SourcePipelineRun, "pr_failed")
	second := newDependentTask(t, srv, "second", state.SourceTask, first.TaskID)
	if err := srv.stateStore.RegisterWorkResult(state.WorkResult{
		SourceKind: state.SourcePipelineRun, SourceID: "pr_failed", Outcome: state.OutcomeFailure,
	}); err != nil {
		t.Fatal(err)
	}
	// Simulate the crash window after the first park commits and before its
	// dependents are visited. A parked task has no work result to replay.
	if _, err := srv.stateStore.EvaluateSource(state.SourcePipelineRun, "pr_failed"); err != nil {
		t.Fatal(err)
	}
	if err := srv.recoverTasks(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertDependencyChainParked(t, srv, first, second)
	third := newDependentTask(t, srv, "third", state.SourceTask, second.TaskID)
	if third.State != state.TaskDependencyFailed {
		t.Fatalf("task armed on an already parked prerequisite: state=%s", third.State)
	}
}
