package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// Continue records run-page input through the same transaction as chat input.
func (m *Manager) Continue(ctx context.Context, runID string, expectedRevision int64, input string) (RunDetail, error) {
	text := strings.TrimSpace(input)
	var receipt *state.AttachmentReceipt
	if text != "" {
		message, _ := json.Marshal(map[string]string{"text": text})
		requestID := fmt.Sprintf("run_%s_%d", runID, expectedRevision)
		receipt = &state.AttachmentReceipt{RequestID: requestID, PayloadHash: requestID, MessageJSON: string(message), Action: "prompt"}
	}
	return m.continueRun(ctx, runID, expectedRevision, text, receipt)
}

// ContinueFromOwner lets a person's message in a paused run owner's
// conversation continue that run exactly as the run page would: an answer to
// its question, or instructions that resume it. It reports false, changing
// nothing, when no paused run awaits this owner's input; approval checkpoints
// are never continued this way (TS-09.R69, FS-14.R99/R100).
func (m *Manager) ContinueFromOwner(ctx context.Context, text string, receipt state.AttachmentReceipt) (bool, error) {
	runs, err := m.store.ListActivePipelineRuns()
	if err != nil {
		return false, err
	}
	for _, run := range runs {
		if run.State != "paused" || run.PendingAction != "" && run.PendingAction != pendingRepairRoomContext {
			continue
		}
		current, found, err := m.currentStageTask(run.RunID)
		if err != nil {
			return false, err
		}
		if !found || current.RoomID != "" || current.StandingAgentID != receipt.AgentID {
			continue
		}
		_, err = m.continueRun(ctx, run.RunID, run.Revision, text, &receipt)
		return err == nil, err
	}
	return false, nil
}

func (m *Manager) continueRun(ctx context.Context, runID string, expectedRevision int64, input string, receipt *state.AttachmentReceipt) (RunDetail, error) {
	fromConversation := receipt != nil && receipt.AgentID != ""
	unlock := m.lockRun(runID)
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		unlock()
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		unlock()
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before continuing")
	}
	if current, found, stageErr := m.currentStageTask(runID); stageErr == nil && found {
		if fromConversation && receipt.AgentID != current.StandingAgentID {
			unlock()
			return RunDetail{}, controlError("revision_conflict", "the run's owner changed; refresh before continuing")
		}
		detail, detailErr := m.Detail(runID)
		if detailErr != nil {
			unlock()
			return RunDetail{}, detailErr
		}
		task, taskErr := m.store.ReadTask(current.TaskID)
		if taskErr != nil {
			unlock()
			return RunDetail{}, taskErr
		}
		stage, found := stageByID(detail.Template, current.StageID)
		if !found {
			unlock()
			return RunDetail{}, controlError("invalid_state", "current stage is missing")
		}
		kind := ContinueKindFor(run, task.State, task.Outcome, current.RoomID != "")
		if current.RoomID != "" && kind == ContinueUnavailable {
			unlock()
			return RunDetail{}, errRoomRecovery
		}
		if receipt != nil && receipt.AgentID == "" {
			receipt.AgentID = current.StandingAgentID
			if receipt.AgentID == "" && kind != ContinueApproval && kind != ContinueController {
				unlock()
				return RunDetail{}, controlError("invalid_state", "the run has no owner to receive instructions")
			}
		}
		if receipt != nil {
			receipt.OriginTaskID = current.TaskID
			receipt.OriginQuestion = run.PauseQuestion
		}
		var continueErr error
		resumeController := false
		switch {
		case utf8.RuneCountInString(input) > MaxValueRunes:
			continueErr = validationError("continuation input is too long", []Diagnostic{{Field: "input", Code: "invalid", Message: "input must fit the pipeline value limit"}})
		case receipt != nil && (kind == ContinueController || kind == ContinueApproval):
			continueErr = validationError("this control does not accept instructions", []Diagnostic{{Field: "input", Code: "invalid", Message: "clear the instructions before continuing"}})
		case kind == ContinueController:
			// Resume after repair retries the retained controller action once
			// with a fresh failure allowance; nothing already done repeats.
			restored := run.State
			action := ControllerActionForContinue(run, task.State, task.Outcome)
			if action == "release_stage_task" {
				restored = "finishing"
			}
			var resumed state.PipelineRunRecord
			if resumed, continueErr = m.store.UpdatePipelineRunCAS(run.RunID, run.Revision, state.PipelineRunUpdate{
				State: restored, PendingAction: action, CurrentStageID: run.CurrentStageID,
				CurrentAttemptID: run.CurrentAttemptID, CurrentAgentID: run.CurrentAgentID,
			}); continueErr == nil {
				m.publish(resumed)
				resumeController = true
			}
		case kind == ContinueApproval:
			// Approval is the explicit checkpoint action; the recorded decision
			// then runs exactly as reported (TS-09.R68).
			if continueErr = m.store.ApproveStageVisit(current.TaskID); continueErr == nil {
				continueErr = m.executeStageDecision(run, detail, current, stage, task)
			}
		case kind == ContinueInterrupted:
			// Resume continues the same owner on its interrupted visit with a
			// fresh allowance; Retry and Replace stay available (TS-09.R70/R71).
			var resumed state.PipelineRunRecord
			if resumed, continueErr = m.store.ResumeInterruptedPipelineStageTask(runID, current.TaskID, run.Revision, receipt); continueErr == nil {
				m.publish(resumed)
			}
		case kind == ContinueAnswer:
			// The answer continues the same goal on a new visit (FS-14.R99).
			if receipt == nil {
				continueErr = validationError("an answer is required", []Diagnostic{{Field: "input", Code: "invalid", Message: "answer the run's question: " + run.PauseQuestion}})
				break
			}
			continueErr = m.createVisit(run, detail, current, stage, visitOptions{Continuation: "You asked the person: " + run.PauseQuestion + "\nTheir answer accompanies this assignment as a user message.", DecisionSource: current.DecisionSourceTaskID, Input: receipt})
		case kind == ContinueResume || kind == ContinueRoomContext:
			// Resume after repair needs no invented text, and grants a fresh
			// allowance exactly once under this revision (TS-09.R70).
			text := "The person resumed this run after the pause (" + run.AttentionReason + "). Re-check the current state before repeating any action."
			if kind == ContinueRoomContext {
				text = roomContextContinuation(run)
			}
			continueErr = m.createVisit(run, detail, current, stage, visitOptions{Continuation: text, RenewAllowance: true, DecisionSource: current.DecisionSourceTaskID, Input: receipt})
		default:
			continueErr = controlError("invalid_state", "continue is not valid for the current run state")
		}
		unlock()
		if continueErr != nil {
			return RunDetail{}, continueErr
		}
		if resumeController {
			if err := m.Reconcile(ctx, runID); err != nil {
				m.RecordReconcileFailure(runID, err)
			}
		}
		if fromConversation {
			// The caller already has the committed input identity. A later
			// detail read must not turn its acceptance into a refusal.
			return RunDetail{}, nil
		}
		return m.Detail(runID)
	} else if stageErr != nil {
		unlock()
		return RunDetail{}, stageErr
	}
	unlock()
	return RunDetail{}, controlError("invalid_state", "run has no durable stage task")
}

func (m *Manager) Retry(ctx context.Context, runID string, expectedRevision int64) (RunDetail, error) {
	unlock := m.lockRun(runID)
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		unlock()
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		unlock()
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before retrying")
	}
	// Retry output acceptance reuses the exact published synthesis; it never
	// reruns the judge or ordinary stage execution (TS-09.R53).
	if run.State == "paused" && run.PendingAction == pendingAcceptRoomOutput {
		unlock()
		if err := m.acceptRoomOutput(ctx, runID, true); err != nil {
			return RunDetail{}, err
		}
		return m.Detail(runID)
	}
	if current, found, stageErr := m.currentStageTask(runID); stageErr == nil && found {
		if current.RoomID != "" {
			unlock()
			return RunDetail{}, errRoomRecovery
		}
		updated, retryErr := m.store.RetryInterruptedPipelineStageTask(runID, current.TaskID, expectedRevision)
		if retryErr != nil {
			unlock()
			if errors.Is(retryErr, state.ErrPipelineStageConflict) {
				return RunDetail{}, controlError("invalid_state", "retry is not valid for the current stage task")
			}
			return RunDetail{}, retryErr
		}
		m.publish(updated)
		unlock()
		return m.Detail(runID)
	} else if stageErr != nil {
		unlock()
		return RunDetail{}, stageErr
	}
	unlock()
	return RunDetail{}, controlError("invalid_state", "run has no durable stage task")
}

func (m *Manager) Replace(ctx context.Context, runID string, expectedRevision int64, runtime RuntimeAssignment) (RunDetail, error) {
	unlock := m.lockRun(runID)
	defer unlock()
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before replacing the standing owner")
	}
	detail, err := m.Detail(runID)
	if err != nil {
		return RunDetail{}, err
	}
	current, found, err := m.currentStageTask(runID)
	if err != nil || !found {
		return RunDetail{}, controlError("invalid_state", "there is no current stage task to replace")
	}
	if current.RoomID != "" {
		return RunDetail{}, errRoomRecovery
	}
	task, err := m.store.ReadTask(current.TaskID)
	if err != nil {
		return RunDetail{}, err
	}
	stage, found := stageByID(detail.Template, current.StageID)
	if !found || run.State != "paused" || task.State != state.TaskInterrupted || runtime.Backend == "" || runtime.Model == "" {
		return RunDetail{}, controlError("invalid_state", "replacement requires an interrupted standing-owner task and a valid runtime")
	}
	if m.lifecycle != nil {
		if err := m.lifecycle.ValidateStage(ctx, StageExecution{StageID: stage.ID, StageTitle: stage.Title, Role: detail.Template.OrchestratorRole, Project: run.Project, Backend: runtime.Backend, Model: runtime.Model, Effort: runtime.Effort, Fast: runtime.Fast}); err != nil {
			return RunDetail{}, validationError("replacement runtime is unavailable", []Diagnostic{{Field: "orchestrator", Code: "unavailable", Message: err.Error()}})
		}
	}
	if err := m.replaceOwner(run, detail, current, stage, runtime, false); err != nil {
		return RunDetail{}, err
	}
	return m.Detail(runID)
}

// replaceOwner issues a visibly fresh standing owner for an interrupted visit:
// a new identity with the given runtime and a bounded evidence handoff, the
// old identity, history and child work retained (TS-09.R41/R70). An automatic
// replacement spends a round atomically; the person's pending input moves to
// the successor assignment in the same transaction.
func (m *Manager) replaceOwner(run state.PipelineRunRecord, detail RunDetail, current state.PipelineStageTask, stage Stage, runtime RuntimeAssignment, charge bool) error {
	if current.DecisionSourceTaskID != "" {
		stage = decisionStage(stage)
	}
	id, err := m.store.NewTaskID()
	if err != nil {
		return err
	}
	// A replacement keeps the predecessor's coordinator binding, so its assignment
	// names that same child rather than minting a second one (TS-09.R37/R41).
	var replacementCoordinator *coordinatorHandoff
	if current.CoordinatorTaskID != "" {
		replacementCoordinator = &coordinatorHandoff{TaskID: current.CoordinatorTaskID, Role: stage.DedicatedRole, Objective: stage.Objective}
	}
	prior, err := m.priorStageResults(run.RunID, "")
	if err != nil {
		return err
	}
	continuation := "Replacement standing owner: inspect the retained stage work and continue from durable results. Messages the person sent the previous owner, if any, follow as their own messages."
	instruction, digest := renderAssignment(run, detail.Template, stage, detail.Values, assignmentContext{
		Coordinator: replacementCoordinator, PriorResults: prior, Continuation: continuation,
	})
	// A revisited step may already have later attempts than the current one.
	attempt, err := m.store.NextPipelineStageAttempt(run.RunID, current.StageIndex)
	if err != nil {
		return err
	}
	prepared, err := m.store.PreparePipelineStageReplacement(state.CreatePipelineStageTaskParams{RunID: run.RunID, ExpectedRevision: run.Revision, StageIndex: current.StageIndex, AttemptNumber: attempt, StageID: stage.ID, AssignmentDigest: digest, ParentTaskID: current.TaskID, OutputValues: stageOutputValues(stage), ChargeRound: charge, Task: state.Task{TaskID: id, Project: run.Project, DisplayName: stage.Title, Instruction: instruction, TargetKind: state.TargetLaunch, Role: detail.Template.OrchestratorRole, Backend: runtime.Backend, Model: runtime.Model, Effort: runtime.Effort, Fast: runtime.Fast, CreatedByKind: "pipeline"}}, current.TaskID)
	if err != nil {
		if errors.Is(err, state.ErrPipelineStageConflict) {
			return controlError("invalid_state", "replacement lost to a newer stage state")
		}
		return err
	}
	updated, err := m.store.ActivatePipelineStageReplacement(run.RunID, id, prepared.Revision)
	if err != nil {
		return err
	}
	m.publish(updated)
	return nil
}

func (m *Manager) Stop(ctx context.Context, runID string, expectedRevision int64) (RunDetail, error) {
	unlock := m.lockRun(runID)
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		unlock()
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		unlock()
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before stopping")
	}
	if run.State == "completed" || run.State == "stopped" {
		unlock()
		return m.Detail(runID)
	}
	if current, found, stageErr := m.currentStageTask(runID); stageErr == nil && found {
		updated, stopErr := m.store.UpdatePipelineRunCAS(runID, run.Revision, state.PipelineRunUpdate{
			State: "stopping", PendingAction: "cleanup_run", CurrentStageID: run.CurrentStageID,
			CurrentAgentID: current.StandingAgentID, FinalOutcome: "", MarkRoomTeardown: current.RoomID != "",
		})
		if stopErr == nil {
			m.publish(updated)
		}
		unlock()
		if stopErr != nil {
			return RunDetail{}, stopErr
		}
		// The committed cleanup cursor owns room closure and replays it after
		// transient failure or restart.
		if err := m.Reconcile(ctx, runID); err != nil {
			slog.Warn("pipeline: stop cleanup", "run", runID, "err", err)
		}
		return m.Detail(runID)
	} else if stageErr != nil {
		unlock()
		return RunDetail{}, stageErr
	}
	unlock()
	return RunDetail{}, controlError("invalid_state", "run has no durable stage task")
}

// FinishStopCleanup commits the terminal run outcome only after every lineage
// member is terminal and its runtime release has settled.
func (m *Manager) FinishStopCleanup(runID string, expectedRevision int64) (RunDetail, error) {
	unlock := m.lockRun(runID)
	defer unlock()
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before finishing cleanup")
	}
	if run.State != "stopping" || run.PendingAction != "cleanup_run" {
		return RunDetail{}, controlError("invalid_state", "run cleanup is not pending")
	}
	current, found, err := m.currentStageTask(runID)
	if err != nil {
		return RunDetail{}, err
	}
	if found && current.RoomID != "" && m.lifecycle != nil {
		if err := m.lifecycle.StopRoom(context.Background(), current.RoomID); err != nil {
			return RunDetail{}, err
		}
	}
	settled, err := m.settleCleanup(runID, "", "")
	if err != nil {
		return RunDetail{}, err
	}
	if settled {
		if settled, err = m.roomSettled(runID); err != nil {
			return RunDetail{}, err
		}
	}
	if !settled {
		return RunDetail{}, controlError("cleanup_pending", "run cleanup still has unfinished task effects")
	}
	stopped, err := m.store.UpdatePipelineRunCAS(runID, run.Revision, state.PipelineRunUpdate{
		State: "stopped", PendingAction: "", CurrentStageID: run.CurrentStageID,
		CurrentAgentID: "", FinalOutcome: state.OutcomeCancelled,
	})
	if err != nil {
		return RunDetail{}, err
	}
	m.publish(stopped)
	m.notify(stopped, "completed")
	return m.Detail(runID)
}

// RepairCleanup claims the submitted run revision before repair runs the
// member cleanup effects, so of two concurrent or stale requests only the one
// holding the current revision performs any effect; the other sees
// revision_conflict having touched nothing (FS-20.R24, INV §5/§15).
func (m *Manager) RepairCleanup(ctx context.Context, runID string, expectedRevision int64, repair func() error) (RunDetail, error) {
	unlock := m.lockRun(runID)
	run, err := m.store.ReadPipelineRun(runID)
	if err != nil {
		unlock()
		return RunDetail{}, err
	}
	if run.Revision != expectedRevision {
		unlock()
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before repairing cleanup")
	}
	// Stage completion retains cleanup exactly as Stop does, so `finishing` needs
	// the same repair route: without it a stage whose cleanup failed persistently
	// had no operator action at all (TS-09.R42).
	if !cleanupRepairable(run) {
		unlock()
		return RunDetail{}, controlError("invalid_state", "cleanup repair is not valid for the current run state")
	}
	claimed, err := m.store.UpdatePipelineRunCAS(runID, run.Revision, state.PipelineRunUpdate{
		State: run.State, PendingAction: run.PendingAction, CurrentStageID: run.CurrentStageID,
		CurrentAttemptID: run.CurrentAttemptID, CurrentAgentID: run.CurrentAgentID,
		AttentionReason: run.AttentionReason, FinalOutcome: run.FinalOutcome,
	})
	unlock()
	if errors.Is(err, state.ErrPipelineConflict) {
		return RunDetail{}, controlError("revision_conflict", "run changed; refresh before repairing cleanup")
	}
	if err != nil {
		return RunDetail{}, err
	}
	m.publish(claimed)
	if err := repair(); err != nil {
		return RunDetail{}, err
	}
	if err := m.Reconcile(ctx, runID); err != nil {
		return RunDetail{}, err
	}
	return m.Detail(runID)
}

// CleanupRepairable reports whether a run is holding retained cleanup that the
// operator-facing repair control can retry. Both the stopping run's cleanup and
// a completing stage's release are retained the same way, so both expose the
// same recovery action (TS-09.R42, FS-14.R44).
func CleanupRepairable(state, pendingAction string) bool {
	return (state == "stopping" && pendingAction == "cleanup_run") ||
		(state == "finishing" && pendingAction == "release_stage_task")
}

func cleanupRepairable(run state.PipelineRunRecord) bool {
	return CleanupRepairable(run.State, run.PendingAction)
}

// StopProject uses the ordinary durable stop path for every non-terminal run
// owned by a project. Project archival calls it before archiving stage agents.
func (m *Manager) StopProject(ctx context.Context, project string) error {
	runs, err := m.store.ListActivePipelineRuns()
	if err != nil {
		return err
	}
	for _, run := range runs {
		if run.Project != project {
			continue
		}
		if _, err := m.Stop(ctx, run.RunID, run.Revision); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) Delete(runID string) error {
	return m.store.DeletePipelineRun(runID)
}
