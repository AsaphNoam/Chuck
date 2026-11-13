package server

import (
	"context"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-14.R99/R103, INV §8 — Continue names what it will do for each pause: an
// approval, a required answer, or a resume with a fresh allowance; a running
// run offers nothing and says why.
func TestContinueControlNamesThePauseItResolves(t *testing.T) {
	failed := pipelineStageTaskDetail{State: state.TaskFinished, Result: &pipelineStageResult{Outcome: state.OutcomeFailure}}
	succeeded := pipelineStageTaskDetail{State: state.TaskFinished, Result: &pipelineStageResult{Outcome: state.OutcomeSuccess}}
	for _, tc := range []struct {
		name     string
		run      state.PipelineRunRecord
		current  pipelineStageTaskDetail
		eligible bool
		reason   string
	}{
		{"running", state.PipelineRunRecord{State: "running"}, failed, false, "Available when the run pauses for approval, a question or recovery."},
		{"approval", state.PipelineRunRecord{State: "paused", PendingAction: "await_approval"}, succeeded, true, "Approves this step's accepted result; the owner's recorded decision then runs."},
		{"question", state.PipelineRunRecord{State: "paused", AttentionReason: "needs_input"}, failed, true, "Answers the run's question; an answer is required."},
		{"interrupted", state.PipelineRunRecord{State: "paused", AttentionReason: "owner_not_reporting"}, pipelineStageTaskDetail{State: state.TaskInterrupted}, true, "Resumes the interrupted owner with a fresh recovery allowance; instructions are optional and reach it with its resumed turn."},
		{"exhausted", state.PipelineRunRecord{State: "paused", AttentionReason: state.PauseRecoveryExhausted}, failed, true, "Resumes the run with a fresh recovery allowance; instructions are optional."},
		{"unknown pause", state.PipelineRunRecord{State: "paused", AttentionReason: "unrelated"}, succeeded, false, "Nothing is waiting for you."},
		{"replacement held", state.PipelineRunRecord{State: "paused", PendingAction: "activate_replacement", AttentionReason: state.PauseControllerRecoveryFailed}, pipelineStageTaskDetail{State: state.TaskArmed}, true, "Retries the held control step after repair."},
		{"room context", state.PipelineRunRecord{State: "paused", PendingAction: "repair_room_context"}, succeeded, true, "Resumes the producing step so its owner can correct the Think Tank stage context."},
	} {
		got := continueControl(tc.run, tc.current)
		if got.Eligible != tc.eligible || got.Reason != tc.reason {
			t.Errorf("%s: %+v, want eligible=%v %q", tc.name, got, tc.eligible, tc.reason)
		}
	}
}

func TestProjectedContinueMatchesManagerForDurablePauses(t *testing.T) {
	for _, tc := range []struct {
		name, taskState, outcome, action, reason, input string
		eligible                                        bool
	}{
		{name: "approval", taskState: state.TaskFinished, outcome: state.OutcomeSuccess, action: "await_approval", eligible: true},
		{name: "question", taskState: state.TaskFinished, outcome: state.OutcomeSuccess, reason: "needs_input", input: "Use the existing plan", eligible: true},
		{name: "interrupted", taskState: state.TaskInterrupted, reason: "owner_not_reporting", eligible: true},
		{name: "exhausted", taskState: state.TaskFinished, outcome: state.OutcomeFailure, reason: state.PauseRecoveryExhausted, eligible: true},
		{name: "unknown pause", taskState: state.TaskFinished, outcome: state.OutcomeSuccess, reason: "unrelated"},
		{name: "replacement held", taskState: state.TaskArmed, action: "activate_replacement", reason: state.PauseControllerRecoveryFailed, eligible: true},
		{name: "replacement legacy hold", taskState: state.TaskArmed, reason: "restart_reconcile_failed", eligible: true},
		{name: "room context", taskState: state.TaskFinished, outcome: state.OutcomeSuccess, action: "repair_room_context", reason: "stages.room.context: context too large", eligible: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := testServer(t, true)
			const runID = "pr_continue_projection"
			task := createPipelineSnapshotTask(t, srv, runID, "tk_continue_projection", inputSnapshot)
			if err := srv.stateStore.WriteAgent(state.Agent{AgentID: "a_owner", Interface: "chat"}); err != nil {
				t.Fatal(err)
			}
			if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = 'running', assigned_agent_id = 'a_owner' WHERE task_id = ?`, task.TaskID); err != nil {
				t.Fatal(err)
			}
			if err := srv.stateStore.BindPipelineStageTaskStandingAgent(task.TaskID, "a_owner"); err != nil {
				t.Fatal(err)
			}
			if _, err := srv.stateStore.DB().Exec(`UPDATE tasks SET state = ?, outcome = ? WHERE task_id = ?`, tc.taskState, tc.outcome, task.TaskID); err != nil {
				t.Fatal(err)
			}
			if tc.taskState == state.TaskFinished {
				if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_stage_tasks SET state = 'closed' WHERE task_id = ?`, task.TaskID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := srv.stateStore.DB().Exec(`UPDATE pipeline_runs SET state = 'paused', pending_action = ?, attention_reason = ? WHERE run_id = ?`, tc.action, tc.reason, runID); err != nil {
				t.Fatal(err)
			}
			detail, err := srv.pipelineMgr.Detail(runID)
			if err != nil {
				t.Fatal(err)
			}
			_, controls, err := srv.pipelineTaskRunProjection(detail)
			if err != nil || controls.Continue.Eligible != tc.eligible {
				t.Fatalf("Continue projection = %+v, %v, want eligibility %v", controls.Continue, err, tc.eligible)
			}
			_, err = srv.pipelineMgr.Continue(context.Background(), runID, detail.Run.Revision, tc.input)
			if (err == nil) != tc.eligible {
				t.Fatalf("Continue execution = %v, want success %v", err, tc.eligible)
			}
		})
	}
}
