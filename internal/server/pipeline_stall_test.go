package server

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

const stallSnapshot = `{"version":2,"title":"x","orchestrator_role":"impl","inputs":[],"stages":[{"id":"stage","title":"Stage","objective":"work","inputs":[],"outputs":[]}]}`

// TS-09.R70 — background work defers the turn-end stall check rather than
// ending it: once that work ends, the missed-event sweep continues the idle
// owner instead of leaving the run active indefinitely.
func TestStallCheckDeferredByBackgroundWorkResumesWhenItEnds(t *testing.T) {
	promptLog := filepath.Join(t.TempDir(), "prompts.log")
	srv, _, owner := taskChatServer(t, map[string]string{"FAKEACP_CAPS": "1", "FAKEACP_PROMPT_LOG": promptLog})
	generation := srv.taskGeneration(owner)
	task := createDispatchingStallVisit(t, srv, "pr_bg", "tk_bg")
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, execution_handle = 'ta_bg', started_at = created_at WHERE task_id = ?`, state.TaskRunning, owner, generation, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindConfirmedPipelineOwner(task.TaskID, owner, generation); err != nil {
		t.Fatal(err)
	}
	stalls := func() int {
		var n int
		if err := srv.stateStore.DB().QueryRow(`SELECT owner_stall_turns FROM pipeline_runs WHERE run_id = 'pr_bg'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	// No turn-end callback is needed to discover the stalled visit.
	srv.recoverMissedStageStalls()
	if got := stalls(); got != 0 {
		t.Fatalf("stall counted while background work runs: %d", got)
	}
	if err := srv.registry.Chat().StopBackgroundTask(context.Background(), owner, "task_1"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) && promptCount(t, promptLog) < 2 {
		srv.recoverMissedStageStalls()
		time.Sleep(10 * time.Millisecond)
	}
	// The nudge can itself settle and trigger the next nudge before this
	// check runs. The contract here is that the deferred nudge gets through.
	if got := promptCount(t, promptLog); got < 2 {
		t.Fatalf("provider prompts = %d, want the original turn and a nudge", got)
	}
}

// TS-09.R70 — exhausting start attempts during restart recovery reaches the
// same parked-owner recovery as a live failure, and the dispatcher sweep heals
// an owner parked before that path existed.
func TestParkedOwnerAfterRestartIsRecovered(t *testing.T) {
	srv := testServer(t, true)
	task := createPipelineSnapshotTask(t, srv, "pr_start", "tk_start", stallSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = 'starting', start_attempt_id = 'last', start_attempt_count = ? WHERE task_id = ?`, state.MaxTaskStartAttempts-1, task.TaskID); err != nil {
		t.Fatal(err)
	}
	task, _ = srv.stateStore.ReadTask(task.TaskID)
	srv.recoverStartAttempt(context.Background(), task)
	assertOwnerRecovered(t, srv, "pr_start", task.TaskID)

	stranded := createPipelineSnapshotTask(t, srv, "pr_stranded", "tk_stranded", stallSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ? WHERE task_id = ?`, state.TaskDependencyFailed, stranded.TaskID); err != nil {
		t.Fatal(err)
	}
	srv.recoverMissedStageStalls()
	assertOwnerRecovered(t, srv, "pr_stranded", stranded.TaskID)
}

func TestPipelineRecoverySweepRespectsInterruptedOwnerDisposition(t *testing.T) {
	srv := testServer(t, true)
	for _, tc := range []struct{ run, kind, want string }{
		{"pr_exit", "unexpected_exit", "queued"},
		{"pr_stop", "interrupted", "paused"},
	} {
		task := createPipelineSnapshotTask(t, srv, tc.run, "tk_"+tc.run, stallSnapshot)
		if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = 'running' WHERE task_id = ?`, task.TaskID); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := srv.stateStore.InterruptPipelineOwner(task.TaskID, "", "owner exited", tc.kind); err != nil || !ok {
			t.Fatalf("interrupt task = %v %v", ok, err)
		}
		srv.recoverMissedStageStalls()
		run, err := srv.stateStore.ReadPipelineRun(tc.run)
		if err != nil || run.State != tc.want {
			t.Fatalf("%s recovery = %+v, %v, want %s", tc.kind, run, err, tc.want)
		}
		if tc.kind == "unexpected_exit" && run.RecoveryRoundsUsed != 1 {
			t.Fatalf("unexpected exit recovery charged %d rounds", run.RecoveryRoundsUsed)
		}
		if tc.kind == "interrupted" && run.AttentionReason != "interrupted" {
			t.Fatalf("explicit stop paused with %q", run.AttentionReason)
		}
	}
}

func TestPipelineOwnerBindFailureRollsBackConfirmation(t *testing.T) {
	srv := testServer(t, true)
	task := createDispatchingStallVisit(t, srv, "pr_bind", "tk_bind")
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = 'starting', start_attempt_id = 'bind', assigned_agent_id = 'a_owner', assigned_generation = 'g_bind' WHERE task_id = ?`, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if _, err := srv.stateStore.DB().Exec(`CREATE TRIGGER fail_pipeline_bind BEFORE UPDATE OF standing_agent_id ON pipeline_stage_tasks BEGIN SELECT RAISE(ABORT, 'injected bind failure'); END;`); err != nil {
		t.Fatal(err)
	}
	task.StartAttemptID = "bind"
	srv.confirmTaskStart(task)
	if current, err := srv.stateStore.ReadTask(task.TaskID); err != nil || current.State != state.TaskStarting || current.ExecutionHandle != "" {
		t.Fatalf("failed binding committed task execution: %+v, %v", current, err)
	}
	if run, err := srv.stateStore.ReadPipelineRun("pr_bind"); err != nil || run.State != "queued" {
		t.Fatalf("after bind failure run = %+v, %v", run, err)
	}
	if _, err := srv.stateStore.DB().Exec(`DROP TRIGGER fail_pipeline_bind`); err != nil {
		t.Fatal(err)
	}
	srv.confirmTaskStart(task)
	run, err := srv.stateStore.ReadPipelineRun("pr_bind")
	visit, visitErr := srv.stateStore.LatestPipelineStageTask("pr_bind")
	if err != nil || visitErr != nil || run.State != "running" || visit.StandingAgentID != "a_owner" {
		t.Fatalf("after sweep run = %+v, %v, owner = %q, %v", run, err, visit.StandingAgentID, visitErr)
	}
}

func createDispatchingStallVisit(t *testing.T, srv *Server, runID, taskID string) state.Task {
	t.Helper()
	task := createPipelineSnapshotTask(t, srv, runID, taskID, stallSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_runs SET pending_action = 'dispatch_stage_task' WHERE run_id = ?`, runID); err != nil {
		t.Fatal(err)
	}
	return task
}

func assertOwnerRecovered(t *testing.T, srv *Server, runID, oldTaskID string) {
	t.Helper()
	run, err := srv.stateStore.ReadPipelineRun(runID)
	if err != nil {
		t.Fatal(err)
	}
	latest, err := srv.stateStore.LatestPipelineStageTask(runID)
	if err != nil {
		t.Fatal(err)
	}
	replaced := latest.TaskID != oldTaskID
	exhausted := run.State == "paused" && run.AttentionReason == state.PauseRecoveryExhausted
	if !replaced && !exhausted {
		after, _ := srv.stateStore.ReadTask(oldTaskID)
		t.Fatalf("parked owner stranded: task=%s run=%s/%s pending=%s", after.State, run.State, run.AttentionReason, run.PendingAction)
	}
}

// FS-14.A65, TS-09.R70 — an owner whose turns end with its stage visit open
// and neither a report nor a wait gets an ordinary continuation turn, and after
// three such turns the visit is interrupted and the run pauses for the person.
// FS-14.R101, TS-09.R70 — an owner turn that ends in a provider failure while
// the process lives settles at `error`, not `idle`; the sweep and the turn-end
// nudge must still continue it and then pause, not leave the run running.
func TestFailedOwnerTurnIsContinuedThenPaused(t *testing.T) {
	t.Setenv("FAKEACP_CAPS", "1")
	t.Setenv("FAKEACP_FAILURE_RESULT", `{"id":"turn-1:error","revision":1,"category":"service","severity":"error","title":"provider failed","actions":[]}`)
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	generation := srv.taskGeneration(id)
	task := createDispatchingStallVisit(t, srv, "pr_stall", "tk_stall")
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, execution_handle = 'ta_stall', started_at = created_at WHERE task_id = ?`, state.TaskRunning, id, generation, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, id); err != nil {
		t.Fatal(err)
	}
	// Hold the claim through the first turn so its turn-end check is skipped
	// and the first continuation has to come from the sweep.
	if !srv.claimLifecycle(id) {
		t.Fatal("could not hold the initial turn-end check")
	}
	if err := srv.registry.SendPrompt(t.Context(), id, "begin assigned work"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, srv, id, "error")
	srv.releaseLifecycle(id)
	srv.recoverMissedStageStalls()
	deadline := time.Now().Add(10 * time.Second)
	var run state.PipelineRunRecord
	for time.Now().Before(deadline) {
		var err error
		if run, err = srv.stateStore.ReadPipelineRun("pr_stall"); err == nil && run.State == "paused" {
			break
		}
		time.Sleep(20 * time.Millisecond)
		srv.recoverMissedStageStalls()
	}
	if run.State != "paused" || run.AttentionReason != "owner_not_reporting" {
		t.Fatalf("run after failed owner turns = %s attention %q, want paused owner_not_reporting", run.State, run.AttentionReason)
	}
	if after, _ := srv.stateStore.ReadTask(task.TaskID); after.State != state.TaskInterrupted {
		t.Fatalf("failed owner visit = %s, want interrupted", after.State)
	}
	if got := promptCount(t, promptLog); got != state.MaxOwnerStallTurns {
		t.Fatalf("assignment plus continuation prompts = %d, want %d", got, state.MaxOwnerStallTurns)
	}
	raw, err := os.ReadFile(promptLog)
	if err != nil {
		t.Fatal(err)
	}
	text := promptTextAt(t, raw, 1)
	assignment, _, _ := strings.Cut(text, "\n\nYour pipeline assignment is still open")
	facts := assignmentFacts(t, assignment, "Chuck task continuation of your assignment from a Chuck pipeline. Original instruction:\n\n"+task.Instruction+"\n\n")
	if facts["task_id"] != task.TaskID || facts["execution_handle"] != "ta_stall" {
		t.Fatalf("stall continuation facts = %v, want task %s with its existing handle", facts, task.TaskID)
	}
	if strings.Contains(text, "Call get_assigned_task") {
		t.Fatalf("stall continuation still requires an assignment lookup: %q", text)
	}
}

func TestStalledOwnerIsContinuedThenPaused(t *testing.T) {
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	generation := srv.taskGeneration(id)
	task := createDispatchingStallVisit(t, srv, "pr_stall", "tk_stall")
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, execution_handle = 'ta_stall', started_at = created_at WHERE task_id = ?`, state.TaskRunning, id, generation, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, id); err != nil {
		t.Fatal(err)
	}
	// A real assigned turn is the first stall. Losing its turn-end observation
	// costs latency only, so run the same bounded pass as the dispatcher tick.
	if !srv.claimLifecycle(id) {
		t.Fatal("could not hold the initial turn-end check")
	}
	if err := srv.registry.SendPrompt(t.Context(), id, "begin assigned work"); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, srv, id, "idle")
	if _, err := srv.stateStore.DB().Exec(`CREATE TRIGGER fail_stall_nudge BEFORE INSERT ON turn_budget BEGIN SELECT RAISE(ABORT, 'injected prompt admission failure'); END;`); err != nil {
		t.Fatal(err)
	}
	srv.releaseLifecycle(id)
	deadline := time.Now().Add(3 * time.Second)
	stalls := 0
	for time.Now().Before(deadline) && stalls == 0 {
		srv.recoverMissedStageStalls()
		if err := srv.stateStore.DB().QueryRow(`SELECT owner_stall_turns FROM pipeline_runs WHERE run_id = 'pr_stall'`).Scan(&stalls); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	for range state.MaxOwnerStallTurns {
		srv.recoverMissedStageStalls()
	}
	if err := srv.stateStore.DB().QueryRow(`SELECT owner_stall_turns FROM pipeline_runs WHERE run_id = 'pr_stall'`).Scan(&stalls); err != nil {
		t.Fatal(err)
	}
	if got := promptCount(t, promptLog); got != 1 || stalls != 1 {
		t.Fatalf("repeated refused nudge saw %d prompts and %d counted turns, want one each", got, stalls)
	}
	if _, err := srv.stateStore.DB().Exec(`DROP TRIGGER fail_stall_nudge`); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(10 * time.Second)
	var run state.PipelineRunRecord
	for time.Now().Before(deadline) {
		srv.recoverMissedStageStalls()
		var err error
		if run, err = srv.stateStore.ReadPipelineRun("pr_stall"); err == nil && run.State == "paused" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if run.State != "paused" || run.AttentionReason != "owner_not_reporting" {
		t.Fatalf("stalled run = %+v, want a recoverable pause", run)
	}
	if after, _ := srv.stateStore.ReadTask(task.TaskID); after.State != state.TaskInterrupted {
		t.Fatalf("stalled visit = %s, want interrupted", after.State)
	}
	// Turns one and two were continued; the third stalled end paused instead.
	if got := promptCount(t, promptLog); got != state.MaxOwnerStallTurns {
		t.Fatalf("assignment plus continuation prompts = %d, want %d", got, state.MaxOwnerStallTurns)
	}
}
