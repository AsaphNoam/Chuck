package server

import (
	"testing"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// FS-14.R99/R103, INV §8 — Continue names what it will do for each pause: an
// approval, a required answer, or a resume with a fresh allowance; a running
// run offers nothing and says why.
func TestContinueControlNamesThePauseItResolves(t *testing.T) {
	failed := pipelineStageTaskDetail{State: state.TaskFinished, Result: &pipelineStageResult{Outcome: state.OutcomeFailure}}
	for _, tc := range []struct {
		name     string
		run      state.PipelineRunRecord
		current  pipelineStageTaskDetail
		eligible bool
		reason   string
	}{
		{"running", state.PipelineRunRecord{State: "running"}, failed, false, "Available when the run pauses for approval, a question or recovery."},
		{"approval", state.PipelineRunRecord{State: "paused", PendingAction: "await_approval"}, failed, true, "Approves this step's accepted result; the owner's recorded decision then runs."},
		{"question", state.PipelineRunRecord{State: "paused", AttentionReason: "needs_input"}, failed, true, "Answers the run's question; an answer is required."},
		{"interrupted", state.PipelineRunRecord{State: "paused", AttentionReason: "owner_not_reporting"}, pipelineStageTaskDetail{State: state.TaskInterrupted}, true, "Resumes the interrupted owner with a fresh recovery allowance; instructions are optional and reach it with its resumed turn."},
		{"exhausted", state.PipelineRunRecord{State: "paused", AttentionReason: state.PauseRecoveryExhausted}, failed, true, "Resumes the run with a fresh recovery allowance; instructions are optional."},
	} {
		got := continueControl(tc.run, tc.current)
		if got.Eligible != tc.eligible || got.Reason != tc.reason {
			t.Errorf("%s: %+v, want eligible=%v %q", tc.name, got, tc.eligible, tc.reason)
		}
	}
}
