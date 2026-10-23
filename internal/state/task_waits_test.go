package state

import (
	"errors"
	"testing"
	"time"
)

func TestTaskWaitYieldsAndWakesAfterWatchedResult(t *testing.T) {
	st, _ := newTestStore(t)
	owner := newTask(t, st, "proj", "owner")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, runtime_claim = ?, execution_handle = ? WHERE task_id = ?`, TaskRunning, "a_owner", "g1", ClaimCreated, "exec1", owner.TaskID); err != nil {
		t.Fatal(err)
	}
	childID, err := st.NewTaskID()
	if err != nil {
		t.Fatal(err)
	}
	child, err := st.CreateTask(Task{TaskID: childID, Project: "proj", DisplayName: "child", Instruction: "work", TargetKind: TargetLaunch, Role: "impl", CreatedByKind: "agent", CreatedByAgentID: "a_owner"})
	if err != nil {
		t.Fatal(err)
	}
	changes, waiting, err := st.WaitForTasks("a_owner", "g1", "exec1", []TaskWaitObservation{{TaskID: child.TaskID, AfterRevision: child.Revision}})
	if err != nil || len(changes) != 0 || !waiting.PendingYield {
		t.Fatalf("wait = changes %+v task %+v err %v", changes, waiting, err)
	}
	if _, err := st.NotifyTaskWaiters(child.TaskID); err != nil {
		t.Fatal(err)
	}
	settled, err := st.CompleteTaskYield(owner.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.State != TaskReady || !settled.ContinuationPending || settled.AssignedAgentID != "a_owner" || settled.RuntimeClaim != "" {
		t.Fatalf("settled wait = %+v", settled)
	}
}

// TS-10.R29 / CR-A2-5: external input prepares the same durable assignment
// continuation and does not grant a direct runtime turn.
func TestExternalInputPreparesWaitingAssignmentContinuation(t *testing.T) {
	st, _ := newTestStore(t)
	task := newTask(t, st, "proj", "waiting continuation")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ? WHERE task_id = ?`, TaskWaiting, "a_owner", "g1", task.TaskID); err != nil {
		t.Fatal(err)
	}
	prepared, changed, err := st.PrepareTaskContinuation("a_owner")
	if err != nil || !changed {
		t.Fatalf("PrepareTaskContinuation = %+v, %v, changed=%v", prepared, err, changed)
	}
	if prepared.State != TaskReady || !prepared.ContinuationPending || !prepared.ResumeNeeded || prepared.AssignedAgentID != "a_owner" {
		t.Fatalf("prepared continuation = %+v", prepared)
	}
	if _, changed, err := st.PrepareTaskContinuation("a_owner"); err != nil || changed {
		t.Fatalf("second preparation = changed=%v err=%v, want idempotent no-op", changed, err)
	}
}

func TestTaskCleanupFailureBacksOffWithoutChangingTaskOutcome(t *testing.T) {
	st, _ := newTestStore(t)
	task := newTask(t, st, "proj", "cleanup")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, outcome = ?, pending_release = 1, runtime_claim = ? WHERE task_id = ?`, TaskFinished, OutcomeSuccess, ClaimCreated, task.TaskID); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 9, 13, 9, 0, 0, 0, time.UTC)
	original := timeNow
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = original })
	first, err := st.RecordTaskCleanupFailure(task.TaskID, cleanupPhaseRelease, "release:"+task.TaskID, errors.New("stop timed out"))
	if err != nil {
		t.Fatal(err)
	}
	if first.State != TaskFinished || first.Outcome != OutcomeSuccess || first.CleanupFailureCount != 1 || first.CleanupNextRetryAt == nil || !first.CleanupNextRetryAt.Equal(now.Add(2*time.Second)) {
		t.Fatalf("first cleanup failure = %+v", first)
	}
	if due, err := st.DueTaskCleanup(8); err != nil || len(due) != 0 {
		t.Fatalf("due before retry = %+v, %v", due, err)
	}
	now = now.Add(2 * time.Second)
	second, err := st.RecordTaskCleanupFailure(task.TaskID, cleanupPhaseRelease, "release:"+task.TaskID, errors.New("stop timed out"))
	if err != nil {
		t.Fatal(err)
	}
	if second.CleanupFailureCount != 2 || second.CleanupNextRetryAt == nil || !second.CleanupNextRetryAt.Equal(now.Add(4*time.Second)) {
		t.Fatalf("second cleanup failure = %+v", second)
	}
	if err := st.CompleteTaskRelease(task.TaskID); err != nil {
		t.Fatal(err)
	}
	settled, err := st.ReadTask(task.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if settled.CleanupFailureCount != 0 || settled.CleanupNextRetryAt != nil || settled.CleanupLastError != "" || settled.RuntimeClaim != "" {
		t.Fatalf("settled cleanup = %+v", settled)
	}
}

func TestUnsafeTaskCleanupRequiresExplicitRepair(t *testing.T) {
	st, _ := newTestStore(t)
	task := newTask(t, st, "proj", "unsafe cleanup")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, pending_release = 1, runtime_claim = ? WHERE task_id = ?`, TaskFinished, ClaimBorrowed, task.TaskID); err != nil {
		t.Fatal(err)
	}
	unsafe, err := st.RecordTaskCleanupFailureClassified(task.TaskID, cleanupPhaseRelease, "release:"+task.TaskID, errors.New("permission denied"), true)
	if err != nil {
		t.Fatal(err)
	}
	if !unsafe.CleanupUnsafe || unsafe.CleanupNextRetryAt != nil || unsafe.AttentionReason == "" {
		t.Fatalf("unsafe cleanup = %+v", unsafe)
	}
	if due, err := st.DueTaskCleanup(8); err != nil || len(due) != 0 {
		t.Fatalf("unsafe cleanup was retried automatically: %+v, %v", due, err)
	}
	repaired, err := st.RepairTaskCleanup(task.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.CleanupUnsafe || repaired.CleanupNextRetryAt == nil || repaired.AttentionReason != "" {
		t.Fatalf("repaired cleanup = %+v", repaired)
	}
}

func TestTaskExecutionTurnIsFencedByStartAttemptAndGeneration(t *testing.T) {
	st, _ := newTestStore(t)
	task := newTask(t, st, "proj", "borrowed")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, start_attempt_id = ? WHERE task_id = ?`, TaskStarting, "a_owner", "g1", "attempt1", task.TaskID); err != nil {
		t.Fatal(err)
	}
	updated, ok, err := st.SetTaskExecutionTurn(task.TaskID, "attempt1", "g1", "t_000000000001")
	if err != nil || !ok || updated.ExecutionTurn != "t_000000000001" {
		t.Fatalf("set execution turn = %+v, %v, %v", updated, ok, err)
	}
	if _, ok, err := st.SetTaskExecutionTurn(task.TaskID, "attempt1", "g2", "t_000000000002"); err != nil || ok {
		t.Fatalf("stale execution turn accepted: ok=%v err=%v", ok, err)
	}
}

func TestTaskWaitReturnsAlreadyChangedTaskWithoutYield(t *testing.T) {
	st, _ := newTestStore(t)
	owner := newTask(t, st, "proj", "owner")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, execution_handle = ? WHERE task_id = ?`, TaskRunning, "a_owner", "g1", "exec1", owner.TaskID); err != nil {
		t.Fatal(err)
	}
	childID, _ := st.NewTaskID()
	child, err := st.CreateTask(Task{TaskID: childID, Project: "proj", DisplayName: "child", Instruction: "work", TargetKind: TargetLaunch, Role: "impl", CreatedByKind: "agent", CreatedByAgentID: "a_owner"})
	if err != nil {
		t.Fatal(err)
	}
	changes, current, err := st.WaitForTasks("a_owner", "g1", "exec1", []TaskWaitObservation{{TaskID: child.TaskID, AfterRevision: 0}})
	if err != nil || len(changes) != 1 || current.PendingYield {
		t.Fatalf("changed wait = %+v %+v %v", changes, current, err)
	}
}

func TestTaskWaitRefusesSameAssigneeAndCancellationWinsYield(t *testing.T) {
	st, _ := newTestStore(t)
	owner := newTask(t, st, "proj", "owner")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, runtime_claim = ?, execution_handle = ? WHERE task_id = ?`, TaskRunning, "a_owner", "g1", ClaimBorrowed, "exec1", owner.TaskID); err != nil {
		t.Fatal(err)
	}
	childID, _ := st.NewTaskID()
	child, err := st.CreateTask(Task{TaskID: childID, Project: "proj", DisplayName: "same agent", Instruction: "work", TargetKind: TargetAgent, TargetAgentID: "a_owner", CreatedByKind: "agent", CreatedByAgentID: "a_owner"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.WaitForTasks("a_owner", "g1", "exec1", []TaskWaitObservation{{TaskID: child.TaskID, AfterRevision: child.Revision}}); !errors.Is(err, ErrTaskWaitConflict) {
		t.Fatalf("same-assignee wait error = %v, want wait conflict", err)
	}

	child.TargetAgentID = ""
	if _, err := st.DB().Exec(`UPDATE tasks SET target_agent_id = '' WHERE task_id = ?`, child.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.WaitForTasks("a_owner", "g1", "exec1", []TaskWaitObservation{{TaskID: child.TaskID, AfterRevision: child.Revision}}); err != nil {
		t.Fatal(err)
	}
	finished, err := st.CancelTask(owner.TaskID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.State != TaskFinished || finished.PendingYield {
		t.Fatalf("cancelled pending yield = %+v", finished)
	}
	if _, err := st.CompleteTaskYield(owner.TaskID); !errors.Is(err, ErrTaskWaitConflict) {
		t.Fatalf("yield after cancellation error = %v, want wait conflict", err)
	}
}

// TS-10.R29/R53 (ASRE-1): the watched changes that woke an assignment survive
// wait → child failure/deletion → yield release → re-admission → confirmation
// and a store reopen, so the resumed execution can read which work changed and
// how. Confirmation settles the wake without re-waking the new execution.
func TestContinuationContextSurvivesConfirmationAndRestart(t *testing.T) {
	home := t.TempDir()
	st, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = st.Close() }()
	owner := newTask(t, st, "proj", "owner")
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, runtime_claim = ?, execution_handle = ? WHERE task_id = ?`, TaskRunning, "a_owner", "g1", ClaimCreated, "exec1", owner.TaskID); err != nil {
		t.Fatal(err)
	}
	failed, deleted := newTask(t, st, "proj", "failing child"), newTask(t, st, "proj", "deleted child")
	if _, err := st.DB().Exec(`UPDATE tasks SET created_by_kind = 'agent', created_by_agent_id = 'a_owner' WHERE task_id IN (?, ?)`, failed.TaskID, deleted.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := st.WaitForTasks("a_owner", "g1", "exec1", []TaskWaitObservation{
		{TaskID: failed.TaskID, AfterRevision: failed.Revision}, {TaskID: deleted.TaskID, AfterRevision: deleted.Revision},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_child', assigned_generation = 'gc', runtime_claim = ? WHERE task_id = ?`, TaskRunning, ClaimBorrowed, failed.TaskID); err != nil {
		t.Fatal(err)
	}
	failedDone, err := st.RecordPersonTaskResult(failed.TaskID, TaskResult{Outcome: OutcomeFailure, Summary: "tests broke"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.NotifyTaskWaiters(failed.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CancelTask(deleted.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeleteTask(deleted.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CompleteTaskYield(owner.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := st.AdmitReadyTask(owner.TaskID, TaskReservation{AttemptID: "att2", AgentID: "a_owner", Generation: "g2", Claim: ClaimCreated}, 4); err != nil || !ok {
		t.Fatalf("admit continuation = ok %v err %v", ok, err)
	}

	want := map[string]TaskObservedChange{
		failed.TaskID:  {TaskID: failed.TaskID, AfterRevision: failed.Revision, Revision: failedDone.Revision, State: TaskFinished, Outcome: OutcomeFailure, Summary: "tests broke", Changed: true},
		deleted.TaskID: {TaskID: deleted.TaskID, AfterRevision: deleted.Revision, Outcome: OutcomeCancelled, Summary: "cancelled", Deleted: true, Changed: true},
	}
	check := func(when string, s *Store) {
		t.Helper()
		got, err := s.TaskContinuationContext(owner.TaskID)
		if err != nil {
			t.Fatalf("%s: continuation context: %v", when, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%s: continuation context = %+v, want %d sources", when, got, len(want))
		}
		for _, change := range got {
			if change != want[change.TaskID] {
				t.Fatalf("%s: observed change = %+v, want %+v", when, change, want[change.TaskID])
			}
		}
	}
	check("before confirmation", st)
	if _, ok, err := st.ConfirmTaskStart(owner.TaskID, "att2"); err != nil || !ok {
		t.Fatalf("confirm = ok %v err %v", ok, err)
	}
	check("after confirmation", st)

	// The settled watch must not wake the execution that already observed it.
	if woken, err := st.NotifyTaskWaiters(failed.TaskID); err != nil || len(woken) != 0 {
		t.Fatalf("notify after confirmation woke %+v err %v", woken, err)
	}
	if woken, err := st.ReconcileTaskWaits(8); err != nil || len(woken) != 0 {
		t.Fatalf("reconcile after confirmation woke %+v err %v", woken, err)
	}

	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(home)
	if err != nil {
		t.Fatal(err)
	}
	st = reopened
	check("after reopen", reopened)
	running, err := reopened.ReadTask(owner.TaskID)
	if err != nil || running.State != TaskRunning || running.ResumeNeeded {
		t.Fatalf("resumed owner = %+v err %v", running, err)
	}
}
