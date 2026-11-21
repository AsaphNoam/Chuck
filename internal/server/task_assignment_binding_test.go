package server

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// TS-10.R64: inject failure in the actual standing-owner write, not the turn
// hook. No frame may cross the provider boundary; retry must establish both
// execution and authority before sending the assignment.
func TestPipelineBindingFailureSendsNoAssignmentAndRetries(t *testing.T) {
	srv, _, promptLog := activationTestServer(t)
	task := createDispatchingStallVisit(t, srv, "pr_atomic", "tk_atomic")
	if _, err := srv.stateStore.DB().Exec(`CREATE TRIGGER fail_assignment_bind BEFORE UPDATE OF standing_agent_id ON pipeline_stage_tasks BEGIN SELECT RAISE(ABORT, 'injected bind failure'); END`); err != nil {
		t.Fatal(err)
	}
	srv.dispatchReadyTasks(context.Background())
	// The failed fresh launch is stopped; the existing exit path interrupts
	// its reservation and pauses the pipeline for a deliberate retry.
	interrupted := waitTaskState(t, srv, task.TaskID, state.TaskInterrupted)
	if interrupted.ExecutionHandle != "" || interrupted.ExecutionTurn != "" {
		t.Fatalf("failed binding committed execution: %+v", interrupted)
	}
	if n := promptCount(t, promptLog); n != 0 {
		t.Fatalf("failed binding sent %d prompts", n)
	}
	visit, err := srv.stateStore.LatestPipelineStageTask("pr_atomic")
	if err != nil || visit.StandingAgentID != "" {
		t.Fatalf("failed binding published authority: %+v, %v", visit, err)
	}
	if _, err := srv.stateStore.DB().Exec(`DROP TRIGGER fail_assignment_bind`); err != nil {
		t.Fatal(err)
	}
	// Taking the lifecycle claim waits for failed-start cleanup to finish.
	deadline := time.Now().Add(3 * time.Second)
	for srv.lifecycleInFlight(interrupted.AssignedAgentID) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if srv.lifecycleInFlight(interrupted.AssignedAgentID) {
		t.Fatal("failed-start cleanup did not finish")
	}
	paused, err := srv.stateStore.ReadPipelineRun("pr_atomic")
	if err != nil || paused.State != "paused" {
		t.Fatalf("failed launch did not pause recoverably: %+v, %v", paused, err)
	}
	if _, err := srv.stateStore.RetryInterruptedPipelineStageTask("pr_atomic", task.TaskID, paused.Revision); err != nil {
		t.Fatal(err)
	}
	running := dispatchUntilTaskState(t, srv, task.TaskID, state.TaskRunning)
	waitPrompts(t, promptLog, 1)
	visit, err = srv.stateStore.LatestPipelineStageTask("pr_atomic")
	run, runErr := srv.stateStore.ReadPipelineRun("pr_atomic")
	if err != nil || runErr != nil || visit.StandingAgentID != running.AssignedAgentID || run.State != "running" {
		t.Fatalf("retry did not establish authority: visit=%+v run=%+v errors=%v/%v", visit, run, err, runErr)
	}
}

// FS-16.R61: creator provenance survives the fresh-command route. The command
// must remain first while the separately labelled context names its author.
func TestFreshDelegatedAssignmentNamesCreatorAfterCommand(t *testing.T) {
	srv, ts, promptLog := activationTestServer(t)
	creator := launchAndWaitIdle(t, ts, "impl", "tmpproj")
	before := promptCount(t, promptLog)
	task, err := srv.CreateAgentTask(messaging.AgentTaskRequest{
		CreatorAgentID: creator, CreatorGeneration: srv.registry.Generation(creator),
		Project: "tmpproj", DisplayName: "review", Instruction: "$review", Role: "impl",
	})
	if err != nil {
		t.Fatal(err)
	}
	running := dispatchUntilTaskState(t, srv, task.TaskID, state.TaskRunning)
	waitPrompts(t, promptLog, before+1)
	raw, err := os.ReadFile(promptLog)
	if err != nil {
		t.Fatal(err)
	}
	text := promptTextAt(t, raw, before)
	facts := assignmentFacts(t, text, "$review\n\n")
	if !strings.Contains(text, "Chuck task assignment from agent "+creator+".\n") || facts["execution_handle"] != running.ExecutionHandle {
		t.Fatalf("fresh assignment lost creator or handle: %q", text)
	}
}
