package server

import (
	"context"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ASRG-2 / FS-01.R49: checking cannot spend allowance or convert a borrowed
// assignment while its continuation would be blocked by runtime capacity.
func TestQuotaCheckDefersAtTaskCapacity(t *testing.T) {
	s, ts, log := activationTestServer(t)
	cfg, err := s.configStore.ReadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.TaskConcurrency = 1
	if err := s.configStore.WriteConfig(cfg); err != nil {
		t.Fatal(err)
	}
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	gen := s.registry.Generation(id)
	if ae := s.stopAgent(context.Background(), id); ae != nil {
		t.Fatal(ae)
	}
	other := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	quotaTask := newAgentTask(t, s, id, "interrupted work")
	occupant := newAgentTask(t, s, other, "occupy runtime budget")
	for _, row := range []struct{ task, agent, generation, claim string }{
		{quotaTask.TaskID, id, gen, state.ClaimBorrowed},
		{occupant.TaskID, other, s.registry.Generation(other), state.ClaimWoke},
	} {
		if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET state=?, assigned_agent_id=?, assigned_generation=?, runtime_claim=? WHERE task_id=?`, state.TaskRunning, row.agent, row.generation, row.claim, row.task); err != nil {
			t.Fatal(err)
		}
	}
	q, err := s.stateStore.ObserveQuota(state.QuotaObservation{AgentID: id, EpisodeID: "capacity-check", SourceGeneration: gen,
		ExecutingTurn: "source", Owner: state.QuotaOwner{Kind: "task", TaskID: quotaTask.TaskID},
		ObservedAt: time.Now().UTC().Add(-61 * time.Minute), State: state.QuotaUnknownReset, Check: true})
	if err != nil {
		t.Fatal(err)
	}
	admit := func(q state.QuotaIntent) bool {
		t.Helper()
		claimed, ok := s.claimDueQuota(q)
		if !ok {
			t.Fatal("claim due check")
		}
		spec, err := s.quotaCheckSpec(id)
		_, admitted := s.admitQuotaCheck(claimed, spec, err)
		return admitted
	}
	if admit(q) {
		t.Fatal("capacity-blocked check admitted")
	}
	q, err = s.stateStore.GetQuota(id)
	if err != nil || q.ChecksStarted != 0 || q.State != state.QuotaDeferred || q.CheckSession != "" {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
	task, err := s.stateStore.ReadTask(quotaTask.TaskID)
	if err != nil || task.RuntimeClaim != state.ClaimBorrowed {
		t.Fatalf("task=%+v err=%v", task, err)
	}
	if n := promptCount(t, log); n != 0 {
		t.Fatalf("provider prompts=%d", n)
	}
	// The same assignment becomes eligible after the occupying reservation is
	// released, while the check still leaves the borrowed claim untouched.
	if _, err := s.stateStore.DB().Exec(`UPDATE tasks SET runtime_claim=? WHERE task_id=?`, state.ClaimBorrowed, occupant.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.stateStore.DB().Exec(`UPDATE quota_intents SET due_at=? WHERE agent_id=?`, time.Now().UTC().Add(-time.Second).Format(time.RFC3339), id); err != nil {
		t.Fatal(err)
	}
	q, _ = s.stateStore.GetQuota(id)
	if !admit(q) {
		t.Fatal("check refused after capacity opened")
	}
	task, err = s.stateStore.ReadTask(quotaTask.TaskID)
	if err != nil || task.RuntimeClaim != state.ClaimBorrowed {
		t.Fatalf("check reserved runtime: %+v err=%v", task, err)
	}
}

func TestQuotaCheckDefersDuringLifecycleAction(t *testing.T) {
	s, ts, log := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	q := observeUnknownQuota(t, s, id)
	q, ok := s.claimDueQuota(q)
	if !ok {
		t.Fatal("claim due check")
	}
	spec, err := s.quotaCheckSpec(id)
	if !s.claimLifecycle(id) {
		t.Fatal("claim lifecycle")
	}
	defer s.releaseLifecycle(id)
	if _, admitted := s.admitQuotaCheck(q, spec, err); admitted {
		t.Fatal("check admitted during lifecycle action")
	}
	q, err = s.stateStore.GetQuota(id)
	if err != nil || q.State != state.QuotaDeferred || q.ChecksStarted != 0 || promptCount(t, log) != 0 {
		t.Fatalf("quota=%+v err=%v", q, err)
	}
}
