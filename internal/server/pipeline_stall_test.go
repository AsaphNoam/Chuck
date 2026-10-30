package server

import (
	"context"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

const stallSnapshot = `{"version":2,"title":"x","orchestrator_role":"impl","inputs":[],"stages":[{"id":"stage","title":"Stage","objective":"work","inputs":[],"outputs":[]}]}`

// TS-09.R70 — background work defers the turn-end stall check rather than
// ending it: once that work ends, the missed-event sweep continues the idle
// owner instead of leaving the run active indefinitely.
func TestStallCheckDeferredByBackgroundWorkResumesWhenItEnds(t *testing.T) {
	srv := testServer(t, true)
	if err := srv.stateStore.WriteAgent(state.Agent{AgentID: "a_bg", Interface: "chat"}); err != nil {
		t.Fatal(err)
	}
	task := createPipelineSnapshotTask(t, srv, "pr_bg", "tk_bg", stallSnapshot)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = 'a_bg', assigned_generation = 'g1', execution_handle = 'ta_bg', started_at = created_at WHERE task_id = ?`, state.TaskRunning, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, "a_bg"); err != nil {
		t.Fatal(err)
	}
	stalls := func() int {
		var n int
		if err := srv.stateStore.DB().QueryRow(`SELECT owner_stall_turns FROM pipeline_runs WHERE run_id = 'pr_bg'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	if err := srv.stateStore.WriteStatus(state.Status{AgentID: "a_bg", State: "idle", BackgroundRunning: 1, UpdatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	srv.nudgeStalledOwner("a_bg", "g1")
	srv.recoverMissedStageStalls()
	if got := stalls(); got != 0 {
		t.Fatalf("stall counted while background work runs: %d", got)
	}
	if err := srv.stateStore.WriteStatus(state.Status{AgentID: "a_bg", State: "idle", BackgroundRunning: 0, UpdatedAt: 2}); err != nil {
		t.Fatal(err)
	}
	srv.recoverMissedStageStalls()
	if got := stalls(); got != 1 {
		t.Fatalf("owner left idle after background work ended: stall turns = %d", got)
	}
	// The deferred check is consumed once, not repeated by every sweep.
	srv.recoverMissedStageStalls()
	if got := stalls(); got != 1 {
		t.Fatalf("sweep repeated the stall check: %d", got)
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
func TestStalledOwnerIsContinuedThenPaused(t *testing.T) {
	srv, ts, promptLog := activationTestServer(t)
	id := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	generation := srv.taskGeneration(id)
	task := createPipelineSnapshotTask(t, srv, "pr_stall", "tk_stall", `{"version":2,"title":"x","orchestrator_role":"impl","inputs":[],"stages":[{"id":"stage","title":"Stage","objective":"work","inputs":[],"outputs":[]}]}`)
	if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, assigned_agent_id = ?, assigned_generation = ?, execution_handle = 'ta_stall', started_at = created_at WHERE task_id = ?`, state.TaskRunning, id, generation, task.TaskID); err != nil {
		t.Fatal(err)
	}
	if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, id); err != nil {
		t.Fatal(err)
	}
	// The first stalled turn end; every continuation turn's own end then
	// drives the next check through the ordinary turn-end hook.
	srv.nudgeStalledOwner(id, generation)
	deadline := time.Now().Add(10 * time.Second)
	var run state.PipelineRunRecord
	for time.Now().Before(deadline) {
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
	if got := promptCount(t, promptLog); got != state.MaxOwnerStallTurns-1 {
		t.Fatalf("continuation prompts = %d, want %d", got, state.MaxOwnerStallTurns-1)
	}
}
