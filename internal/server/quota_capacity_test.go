package server

import (
	"context"
	"os"
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
		if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=?, execution_handle='existing-execution', started_at=created_at WHERE task_id=?`,
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

	if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET runtime_claim=? WHERE task_id=?`, state.ClaimBorrowed, occupantTask.TaskID); err != nil {
		t.Fatal(err)
	}
	// Capacity is free. A busy deferral is a permitted, non-spending outcome
	// that the due sweep retries (FS-01.A27), so drive the sweep's ticks rather
	// than assume one attempt starts; a hold is terminal and fails at once.
	deadline := time.Now().Add(15 * time.Second)
	for promptCount(t, promptLog) == 0 {
		q, err := s.stateStore.GetQuota(quotaAgent)
		if err != nil {
			t.Fatal(err)
		}
		switch q.State {
		case state.QuotaNeedsAttention:
			t.Fatalf("continuation held instead of starting: %+v", q)
		case state.QuotaDeferred:
			if q.ReasonCode != "busy" || q.DueAt == nil {
				t.Fatalf("continuation deferred for a non-capacity reason: %+v", q)
			}
			if q.Revision != deferred.Revision {
				t.Logf("continuation deferred again after capacity freed: %+v", q)
			}
			if wait := time.Until(*q.DueAt); wait > 0 {
				time.Sleep(wait)
			}
			s.dispatchQuotaContinuations(context.Background())
		}
		if time.Now().After(deadline) {
			t.Fatalf("continuation never reached the provider: %+v", q)
		}
		time.Sleep(10 * time.Millisecond)
	}
	waitPrompts(t, promptLog, 1)

	deadline = time.Now().Add(5 * time.Second)
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
	raw, err := os.ReadFile(promptLog)
	if err != nil {
		t.Fatal(err)
	}
	facts := assignmentFacts(t, firstPromptText(t, raw), "Chuck task continuation of your assignment from the person. Original instruction:\n\n"+quotaTask.Instruction+"\n\n")
	if facts["execution_handle"] != "existing-execution" || quotaTask.ExecutionTurn == "" {
		t.Fatalf("resumed quota execution changed handle or lacked a current turn: facts=%v task=%+v", facts, quotaTask)
	}
	if running, err := s.stateStore.ReadRunning(quotaAgent); err != nil || running.AgentID != quotaAgent {
		t.Fatalf("quota agent did not resume: %+v err=%v", running, err)
	}
}
