package server

import (
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

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
