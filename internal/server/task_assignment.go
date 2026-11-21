package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/AsaphNoam/Chuck/internal/contextref"
	"github.com/AsaphNoam/Chuck/internal/messaging"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// errTaskTurnStale refuses a task turn whose starting reservation another
// owner already settled, so its assignment is never sent (TS-10.R64, INV §23).
var errTaskTurnStale = errors.New("task start is no longer this execution's")

// prepareTaskTurn builds the current assignment for the task this agent's
// admitted turn executes and returns the binder that confirms it. It reads the
// task when the turn is admitted, never when its wake was queued, and confirms
// only the exact reserved attempt for the live generation and claimed turn
// (FS-16.R60–R61, TS-10.R63–R65). A read or build failure sends nothing.
func (s *Server) prepareTaskTurn(agentID, generation, turnID string, launched bool) (runtime.TaskTurn, error) {
	task, err := s.stateStore.AssignedTask(agentID)
	continuing := !launched && task.State == state.TaskRunning && task.AssignedGeneration == generation &&
		task.ExecutionHandle != "" && !task.PendingRelease && !task.PendingYield
	if err == nil && !continuing && (task.State != state.TaskStarting || task.StartAttemptID == "") {
		err = errTaskTurnStale
	}
	if err != nil {
		s.log.Warn("task turn refused", "agent", agentID, "generation", generation, "turn", turnID, "task", task.TaskID, "op", "read_assignment", "err", err)
		return runtime.TaskTurn{}, err
	}
	context, err := s.messaging.AssignmentContext(task)
	if err != nil {
		s.log.Warn("task turn refused", "agent", agentID, "generation", generation, "task", task.TaskID, "attempt", task.StartAttemptID, "op", "assignment_context", "err", err)
		return runtime.TaskTurn{}, err
	}
	// New executions use the reserved attempt. Another turn of an existing
	// execution keeps its current handle rather than reviving a stale attempt.
	if !continuing {
		context["execution_handle"] = task.StartAttemptID
	}
	prompt, err := renderTaskAssignment(task, context, launched && task.StartedAt == nil)
	if err != nil {
		s.log.Warn("task turn refused", "agent", agentID, "generation", generation, "task", task.TaskID, "attempt", task.StartAttemptID, "op", "render_assignment", "err", err)
		return runtime.TaskTurn{}, err
	}
	confirm := func() error {
		var confirmed state.Task
		var ok bool
		var err error
		if continuing {
			confirmed, ok, err = s.stateStore.ConfirmTaskContinuation(task.TaskID, task.ExecutionHandle, generation, turnID)
		} else {
			confirmed, ok, err = s.stateStore.ConfirmTaskExecution(task.TaskID, task.StartAttemptID, generation, turnID)
		}
		if err == nil && !ok {
			err = errTaskTurnStale
		}
		if err != nil {
			s.log.Warn("task turn refused", "agent", agentID, "generation", generation, "turn", turnID, "task", task.TaskID, "attempt", task.StartAttemptID, "op", "confirm", "err", err)
			return err
		}
		s.publishTaskPipelineOwner(confirmed.TaskID)
		s.publishTaskUpdate(confirmed)
		return nil
	}
	return runtime.TaskTurn{Prompt: prompt, Confirm: confirm}, nil
}

// maxAssignmentBytes is TS-10.R65's finite ceiling on one rendered task turn,
// derived from the admitted limits of every projected field at their worst
// encoding rather than from mail's smaller allowance. Raw instruction text is
// at most 4 bytes per rune; JSON-escaped text at most 6 (`<`, ` `,
// control characters). Code-owned values with no authoring limit (ids,
// states, attention reasons) get a stated per-value allowance. A turn over the
// ceiling is refused, never clipped.
const (
	maxEscapedBytesPerRune = 6
	codeOwnedValueBytes    = 1000 * maxEscapedBytesPerRune
	assignmentFrameBytes   = 4096 // headings, labels, keys and delimiters
	maxAttachmentBytes     = 2*codeOwnedValueBytes + (contextref.MaxLabelRunes+contextref.MaxDescriptionRunes)*maxEscapedBytesPerRune
	maxObservedChangeBytes = 4*codeOwnedValueBytes + state.MaxResultSummaryRunes*maxEscapedBytesPerRune
	maxPipelineStepBytes   = 2*codeOwnedValueBytes + (pipelineStepTitleRunes+pipelineObjectiveSummaryRunes)*maxEscapedBytesPerRune
	maxPipelineBytes       = 6*codeOwnedValueBytes + state.MaxPipelineSteps*(maxPipelineStepBytes+2*codeOwnedValueBytes) +
		len(messaging.PipelineDecisionContract)*maxEscapedBytesPerRune
	maxAssignmentBytes = assignmentFrameBytes + maxTaskInstructionRunes*utf8.UTFMax +
		maxTaskNameRunes*maxEscapedBytesPerRune + 2*codeOwnedValueBytes +
		maxTaskAttachments*maxAttachmentBytes + state.MaxTaskWaitSources*maxObservedChangeBytes + maxPipelineBytes

	// The pipeline plan projection's own clips (state.PipelineSteps) and the
	// template title limit.
	pipelineStepTitleRunes        = 120
	pipelineObjectiveSummaryRunes = 240
)

// errAssignmentTooLarge refuses a rendered turn over maxAssignmentBytes.
var errAssignmentTooLarge = errors.New("assignment exceeds its derived size ceiling")

// renderTaskAssignment is the one place a task's provider text is composed
// (INV §21). A task's own first launch keeps the instruction first and verbatim,
// so a leading command or skill the author chose reaches the provider as
// written. Every other route opens with a short attributed heading, so a
// previously delivered instruction is task content, never a provider-native
// command run again. The context block carries facts, not procedure.
func renderTaskAssignment(task state.Task, context map[string]any, fresh bool) (string, error) {
	block, err := json.Marshal(context)
	if err != nil {
		return "", fmt.Errorf("encode assignment context: %w", err)
	}
	contextText := "Chuck assignment context:\n" + string(block)
	if fresh {
		contextText = "Chuck task assignment from " + taskAuthor(task) + ".\n" + contextText
	}
	text := task.Instruction + "\n\n" + contextText
	if !fresh {
		heading := "Chuck task assignment from " + taskAuthor(task) + ". Instruction:"
		if task.StartedAt != nil {
			heading = "Chuck task continuation of your assignment from " + taskAuthor(task) + ". Original instruction:"
		}
		text = heading + "\n\n" + text
	}
	if len(text) > maxAssignmentBytes {
		return "", fmt.Errorf("%w: %d bytes", errAssignmentTooLarge, len(text))
	}
	return text, nil
}

// taskAuthor attributes assigned work to who created it; agent-authored work is
// assigned work, not the person's approval (FS-16.R61).
func taskAuthor(task state.Task) string {
	switch task.CreatedByKind {
	case "agent":
		if task.CreatedByAgentID != "" {
			return "agent " + task.CreatedByAgentID
		}
		return "an agent"
	case "person":
		return "the person"
	case "pipeline", "pipeline_coordinator":
		return "a Chuck pipeline"
	}
	return "Chuck"
}
