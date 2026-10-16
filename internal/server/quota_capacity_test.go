package server

import (
	"context"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-01.A25/A27: a stopped borrowed task cannot resume past the runtime budget;
// deferral spends neither a provider turn nor a task start attempt.
func TestQuotaStoppedBorrowedTaskDefersUntilRuntimeCapacity(t *testing.T) {
	s, ts, promptLog := activationTestServer(t)
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.TaskConcurrency = 1
	if err := s.configStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}

	quotaAgent := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	quotaGeneration := s.registry.Generation(quotaAgent)
	if ae := s.stopAgent(context.Background(), quotaAgent); ae != nil {
		t.Fatal(ae)
	}
	occupantAgent := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	occupantGeneration := s.registry.Generation(occupantAgent)
	if ae := s.stopAgent(context.Background(), occupantAgent); ae != nil {
		t.Fatal(ae)
	}

	quotaTask := newAgentTask(t, s, quotaAgent, "resume quota-interrupted work")
	occupantTask := newAgentTask(t, s, occupantAgent, "occupy the runtime budget")
	for _, row := range []struct {
		task       state.Task
		agent, gen string
		claim      string
	}{{quotaTask, quotaAgent, quotaGeneration, state.ClaimBorrowed}, {occupantTask, occupantAgent, occupantGeneration, state.ClaimWoke}} {
		if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`,
			state.TaskRunning, row.agent, row.gen, row.claim, row.task.TaskID); err != nil {
			t.Fatal(err)
		}
	}

	reset := time.Now().UTC().Add(-time.Second)
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{
		AgentID: quotaAgent, EpisodeID: "known-reset-capacity-fixture", SourceGeneration: quotaGeneration,
		ExecutingTurn: "interrupted-turn", Owner: state.QuotaOwner{Kind: "task", TaskID: quotaTask.TaskID},
		ObservedAt: reset.Add(-2 * time.Minute), ResetAt: quotaResetFor(reset), DueAt: &reset, State: state.QuotaScheduled,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.startQuotaContinuation(context.Background(), q)

	deferred, err := s.stateStore.GetQuota(quotaAgent)
	if err != nil || deferred.State != state.QuotaDeferred || deferred.ReasonCode != "busy" || deferred.DueAt == nil {
		t.Fatalf("quota was not deferred for capacity: %+v err=%v", deferred, err)
	}
	if got := promptCount(t, promptLog); got != 0 {
		t.Fatalf("provider prompts while runtime budget full = %d", got)
	}
	if _, err := s.stateStore.ReadRunning(quotaAgent); err == nil {
		t.Fatal("quota agent resumed while runtime budget was full")
	}
	quotaTask, err = s.stateStore.ReadTask(quotaTask.TaskID)
	if err != nil || quotaTask.RuntimeClaim != state.ClaimBorrowed || quotaTask.AssignedGeneration != quotaGeneration || quotaTask.StartAttemptCount != 0 {
		t.Fatalf("capacity deferral changed task assignment/attempt: %+v err=%v", quotaTask, err)
	}
	occupantTask, err = s.stateStore.ReadTask(occupantTask.TaskID)
	if err != nil || occupantTask.RuntimeClaim != state.ClaimWoke {
		t.Fatalf("capacity occupant was changed: %+v err=%v", occupantTask, err)
	}

	if wait := time.Until(*deferred.DueAt); wait > 0 {
		time.Sleep(wait)
	}
	deferred, err = s.stateStore.GetQuota(quotaAgent)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET runtime_claim=? WHERE task_id=?`, state.ClaimBorrowed, occupantTask.TaskID); err != nil {
		t.Fatal(err)
	}
	s.startQuotaContinuation(context.Background(), deferred)
	waitPrompts(t, promptLog, 1)

	deadline := time.Now().Add(5 * time.Second)
	for {
		quotaTask, err = s.stateStore.ReadTask(quotaTask.TaskID)
		if err == nil && quotaTask.RuntimeClaim == state.ClaimWoke && quotaTask.AssignedGeneration != quotaGeneration {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("quota task was not resumed in its owned task slot: %+v err=%v", quotaTask, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if quotaTask.AssignedAgentID != quotaAgent || quotaTask.StartAttemptCount != 0 {
		t.Fatalf("quota resume changed task ownership or spent a launch attempt: %+v", quotaTask)
	}
	if running, err := s.stateStore.ReadRunning(quotaAgent); err != nil || running.AgentID != quotaAgent {
		t.Fatalf("quota agent did not resume: %+v err=%v", running, err)
	}
}
