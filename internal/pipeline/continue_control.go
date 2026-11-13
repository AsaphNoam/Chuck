package pipeline

import (
	"strings"

	"github.com/AsaphNoam/Chuck/internal/state"
)

type ContinueKind int

const (
	ContinueUnavailable ContinueKind = iota
	ContinueController
	ContinueApproval
	ContinueInterrupted
	ContinueAnswer
	ContinueResume
	ContinueRoomContext
)

const pendingRepairRoomContext = "repair_room_context"

// ContinueKindFor is the shared eligibility rule for execution and projection.
func ContinueKindFor(run state.PipelineRunRecord, taskState, outcome string, room bool) ContinueKind {
	if run.State != "paused" {
		return ContinueUnavailable
	}
	if ControllerActionForContinue(run, taskState, outcome) != "" {
		return ContinueController
	}
	switch run.PendingAction {
	case "await_approval":
		if outcome == state.OutcomeSuccess {
			return ContinueApproval
		}
	case pendingRepairRoomContext:
		if !room {
			return ContinueRoomContext
		}
	case "":
		switch {
		case room:
			return ContinueUnavailable
		case taskState == state.TaskInterrupted:
			return ContinueInterrupted
		case strings.HasPrefix(run.AttentionReason, "stages.") && strings.Contains(run.AttentionReason, ".context: the goal, objective and inputs for this Think Tank stage exceed "):
			// Older versions wrote only the field diagnostic.
			return ContinueRoomContext
		case run.AttentionReason == attentionNeedsInput:
			return ContinueAnswer
		case run.AttentionReason == attentionRecoveryExhausted || outcome == state.OutcomeFailure || outcome == state.OutcomeBlocked ||
			run.AttentionReason == attentionCheckMissing || run.AttentionReason == attentionApprovalMissing:
			return ContinueResume
		}
	}
	return ContinueUnavailable
}

// ControllerActionForContinue returns the action held for a person to retry.
// Older versions erased the cursor at startup. Restore it only when the
// durable task proves which step can safely be replayed.
func ControllerActionForContinue(run state.PipelineRunRecord, taskState, outcome string) string {
	if run.State != "paused" {
		return ""
	}
	if run.AttentionReason == state.PauseControllerRecoveryFailed &&
		(run.PendingAction == "release_stage_task" || run.PendingAction == "activate_replacement") {
		return run.PendingAction
	}
	if run.PendingAction == "" && run.AttentionReason == "restart_reconcile_failed" {
		switch {
		case taskState == state.TaskArmed:
			return "activate_replacement"
		case taskState == state.TaskFinished && (outcome == state.OutcomeSuccess || outcome == state.OutcomeFailure || outcome == state.OutcomeBlocked):
			return "release_stage_task"
		}
	}
	return ""
}

func roomContextContinuation(run state.PipelineRunRecord) string {
	return "The person resumed after a Think Tank stage rejected its context (" + run.AttentionReason +
		"). Use the prior accepted result to correct the produced values or choose a valid next step."
}
