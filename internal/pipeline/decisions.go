package pipeline

import (
	"context"
	"errors"
	"fmt"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// Attention reasons a goal-led run pauses with. Each is recoverable: the
// person answers, repairs and resumes the same run (FS-14.R99/R101).
const (
	attentionNeedsInput        = "needs_input"
	attentionRecoveryExhausted = state.PauseRecoveryExhausted
	attentionApprovalMissing   = "approval_required"
	attentionCheckMissing      = "required_check_missing"
)

// visitOptions shape one new visit of a step.
type visitOptions struct {
	// Continuation is bounded context for the owner: why this visit exists.
	Continuation string
	// ChargeRound spends an automatic repair/revisit round; RenewAllowance
	// grants a fresh allowance after an explicit Resume (TS-09.R70).
	ChargeRound, RenewAllowance bool
}

// executeStageDecision runs exactly the decision recorded with the settled
// visit's result, after release and cleanup (TS-09.R66/R68). A success with
// no decision was recorded as the usual next step at acceptance; a failure or
// blocked result without one hands the evidence back to the owner as a
// bounded recovery visit rather than to the person (FS-14.R98).
func (m *Manager) executeStageDecision(run state.PipelineRunRecord, detail RunDetail, current state.PipelineStageTask, stage Stage, task state.Task) error {
	decision, err := m.store.StageDecision(current.TaskID)
	if err != nil {
		return err
	}
	if decision == nil {
		if current.RoomID != "" {
			// A room's synthesis carries no owner judgment; the usual plan
			// continues, and the plan's end completes on accepted evidence.
			if task.Outcome != state.OutcomeSuccess {
				return m.pauseRun(run, stage.ID, task.Outcome, "")
			}
			if current.StageIndex+1 < len(detail.Template.Stages) {
				return m.createVisit(run, detail, current, detail.Template.Stages[current.StageIndex+1], visitOptions{})
			}
			return m.completeRun(run, detail, current, stage, nil)
		}
		return m.createVisit(run, detail, current, stage, visitOptions{ChargeRound: true, Continuation: recoveryContinuation(run, stage, task)})
	}
	switch decision.Action {
	case state.DecisionNext:
		target, found := stageByID(detail.Template, decision.StageID)
		if !found {
			return controlError("invalid_state", "decided step is missing")
		}
		opts := visitOptions{}
		if !decision.Implicit {
			opts.Continuation = "The run owner chose this step: " + decision.Reason
		}
		// Repeating or revisiting a step is repair work and spends a round;
		// forward progress through the plan does not (TS-09.R70).
		opts.ChargeRound = stageIndex(detail.Template, target.ID) <= current.StageIndex
		return m.createVisit(run, detail, current, target, opts)
	case state.DecisionNeedsInput:
		return m.pauseRun(run, stage.ID, attentionNeedsInput, decision.Question)
	case state.DecisionComplete:
		return m.completeRun(run, detail, current, stage, decision.RequiredChecks)
	}
	return controlError("invalid_state", "recorded decision is not executable")
}

// recoveryContinuation is the evidence an owner recovery visit carries.
func recoveryContinuation(run state.PipelineRunRecord, stage Stage, task state.Task) string {
	return clipText(fmt.Sprintf("Your previous visit of step %q ended %s: %s\nRecover the run: repair and report again, choose another step with pipeline_decision, or ask the person with needs_input only if they alone can decide. Automatic recovery rounds used: %d of %d.",
		stage.ID, task.Outcome, task.OutcomeSummary, run.RecoveryRoundsUsed+1, state.PipelineRecoveryRounds), MaxValueRunes)
}

// completeRun ends the run successfully. Required evidence was validated with
// the report; approvals are checked here because an approval gate may commit
// after the report that made it actionable (TS-09.R68).
//
// A room at the plan's end has no owner report, so nil checks take the latest
// accepted evidence for each required step instead.
func (m *Manager) completeRun(run state.PipelineRunRecord, detail RunDetail, current state.PipelineStageTask, stage Stage, checks map[string]string) error {
	if checks == nil {
		evidence, err := m.store.RequiredCheckEvidence(run.RunID)
		if err != nil {
			return err
		}
		checks = evidence
	}
	for _, step := range detail.Template.Stages {
		if !step.ApprovalAfterSuccess && !step.RequiredForCompletion {
			continue
		}
		source := checks[step.ID]
		if source == "" {
			return m.pauseRun(run, stage.ID, attentionCheckMissing, "")
		}
		if !step.ApprovalAfterSuccess {
			continue
		}
		approved, err := m.store.StageVisitApproved(source)
		if err != nil {
			return err
		}
		if !approved {
			return m.pauseRun(run, stage.ID, attentionApprovalMissing, "")
		}
	}
	completed, err := m.store.UpdatePipelineRunCAS(run.RunID, run.Revision, state.PipelineRunUpdate{
		State: "completed", PendingAction: "", CurrentStageID: stage.ID,
		CurrentAgentID: "", FinalOutcome: state.OutcomeSuccess,
	})
	if err == nil {
		m.publish(completed)
		m.notify(completed, "completed")
	}
	return err
}

// pauseRun holds the run recoverably with a specific reason and notifies the
// person; it never publishes a terminal failure (FS-14.R99).
func (m *Manager) pauseRun(run state.PipelineRunRecord, stageID, attention, question string) error {
	paused, err := m.store.UpdatePipelineRunCAS(run.RunID, run.Revision, state.PipelineRunUpdate{
		State: "paused", PendingAction: "", CurrentStageID: stageID, CurrentAgentID: run.CurrentAgentID,
		AttentionReason: attention, PauseQuestion: question,
	})
	if err == nil {
		m.publish(paused)
		m.notify(paused, "needs_attention")
	}
	return err
}

// createVisit starts the next visit of any listed step: the usual next step,
// a repeat, a revisit or an owner recovery visit. It is the one constructor
// for those paths (INV §2). The standing owner of the latest ordinary visit is
// reused; a spent allowance pauses the run instead (TS-09.R67/R70).
func (m *Manager) createVisit(run state.PipelineRunRecord, detail RunDetail, current state.PipelineStageTask, next Stage, opts visitOptions) error {
	index := stageIndex(detail.Template, next.ID)
	if index < 0 {
		return controlError("invalid_state", "step "+next.ID+" is missing")
	}
	attempt, err := m.store.NextPipelineStageAttempt(run.RunID, index)
	if err != nil {
		return err
	}
	taskID, err := m.store.NewTaskID()
	if err != nil {
		return err
	}
	assignment := standingAssignment(detail.Assignments, next.ID)
	coordinator, err := m.coordinatorTask(run.Project, run.Goal, taskID, next, detail.Assignments[next.ID])
	if err != nil {
		return err
	}
	prior, err := m.priorStageResults(run.RunID, "")
	if err != nil {
		return err
	}
	instruction, digest := renderAssignment(run, detail.Template, next, detail.Values, assignmentContext{
		Coordinator: coordinatorContext(coordinator, next), PriorResults: prior, Continuation: opts.Continuation,
	})
	// A room stage has no standing owner, so the owner of the latest ordinary
	// stage is the one reused (TS-09.R52).
	standing, err := m.lastStandingAgent(run.RunID)
	if err != nil {
		return err
	}
	targetKind := state.TargetLaunch
	if standing != "" {
		targetKind = state.TargetAgent
	}
	params := state.CreatePipelineStageTaskParams{
		RunID: run.RunID, ExpectedRevision: run.Revision, StageIndex: index,
		AttemptNumber: attempt, StageID: next.ID, AssignmentDigest: digest, ParentTaskID: current.TaskID,
		OutputValues: stageOutputValues(next), Coordinator: coordinator, ChargeRound: opts.ChargeRound, RenewAllowance: opts.RenewAllowance,
		Task: state.Task{
			TaskID: taskID, Project: run.Project, DisplayName: next.Title, Instruction: instruction,
			TargetKind: targetKind, TargetAgentID: standing, Role: detail.Template.OrchestratorRole,
			Backend: assignment.Backend, Model: assignment.Model, Effort: assignment.Effort, Fast: assignment.Fast,
			CreatedByKind: "pipeline",
		},
	}
	if StageCoordination(next) == CoordinationThinkTank {
		room, err := m.roomStage(context.Background(), run, next, detail.Values, attempt, frozenAssignments(run))
		var refused *ControlError
		if errors.As(err, &refused) && len(refused.Diagnostics) > 0 {
			// Produced values overflowed the room's stage context: hold the run
			// with the field named rather than truncating (TS-09.R56).
			return m.pauseRun(run, current.StageID, clipText(refused.Diagnostics[0].Field+": "+refused.Diagnostics[0].Message, MaxDescriptionRunes), "")
		}
		if err != nil {
			return err
		}
		params = roomStageParams(params, run, next, room)
	}
	_, _, _, err = m.store.CreatePipelineStageTask(params)
	if errors.Is(err, state.ErrRecoveryExhausted) {
		return m.pauseRun(run, current.StageID, attentionRecoveryExhausted, "")
	}
	if err == nil {
		updated, readErr := m.store.ReadPipelineRun(run.RunID)
		if readErr == nil {
			m.publish(updated)
		}
	}
	return err
}

// RecoverStageTask is automatic, bounded recovery for a standing owner whose
// execution was interrupted (process exit, restart) or could not be started:
// the same identity is resumed through ordinary admission, or — when its
// native session cannot be used — a visibly fresh owner with the run's frozen
// runtime takes over with the handoff text. Either spends one round; a spent
// allowance pauses recoverably. Explicitly stopped work never reaches here
// (FS-14.R101, TS-09.R70).
func (m *Manager) RecoverStageTask(taskID, attention string, replace bool, handoff string) (bool, error) {
	visit, err := m.store.ReadPipelineStageTaskByTask(taskID)
	if errors.Is(err, state.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if visit.ExecutionKind != state.StageExecutionAgent {
		return false, m.OnStageTaskInterrupted(taskID, attention)
	}
	unlock := m.lockRun(visit.RunID)
	defer unlock()
	run, err := m.store.ReadPipelineRun(visit.RunID)
	if err != nil {
		return false, err
	}
	current, found, err := m.currentStageTask(run.RunID)
	if err != nil || !found || current.TaskID != taskID || run.State == "completed" || run.State == "stopped" || run.State == "stopping" {
		return false, err
	}
	paused, err := m.store.UpdatePipelineRunCAS(run.RunID, run.Revision, state.PipelineRunUpdate{
		State: "paused", PendingAction: "", CurrentStageID: current.StageID, CurrentAgentID: current.StandingAgentID, AttentionReason: attention,
	})
	if errors.Is(err, state.ErrPipelineConflict) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	detail, err := m.Detail(run.RunID)
	if err != nil {
		return false, err
	}
	stage, ok := stageByID(detail.Template, current.StageID)
	if !ok {
		return false, controlError("invalid_state", "current stage is missing")
	}
	if replace {
		err = m.replaceOwner(paused, detail, current, stage, standingAssignment(detail.Assignments, stage.ID), true, handoff)
	} else {
		var updated state.PipelineRunRecord
		if updated, err = m.store.RecoverInterruptedPipelineStageTask(run.RunID, taskID, paused.Revision); err == nil {
			m.publish(updated)
		}
	}
	if errors.Is(err, state.ErrRecoveryExhausted) {
		return false, m.pauseRun(paused, stage.ID, attentionRecoveryExhausted, "")
	}
	return err == nil, err
}

// stageIndex is the step's place in the frozen plan, -1 when absent.
func stageIndex(t Template, id string) int {
	for i, st := range t.Stages {
		if st.ID == id {
			return i
		}
	}
	return -1
}
