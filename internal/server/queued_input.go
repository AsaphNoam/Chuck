package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/pipeline"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// admitConversationInput is shared by ordinary messages and self annotations:
// assignment ownership and run continuation precede any standalone runtime turn.
func (s *Server) admitConversationInput(ctx context.Context, receipt state.AttachmentReceipt, msg runtime.Message) (state.AttachmentReceipt, bool, error) {
	taskID, queued, err := s.stateStore.QueueAssignmentInput(receipt)
	if err != nil {
		return s.replayConversationInput(receipt, err)
	}
	if queued {
		receipt.Delivery, receipt.AssignmentTaskID = "prepared", taskID
		s.logQueuedAssignmentInput(receipt)
		return receipt, true, nil
	}
	return s.continueRunFromConversation(ctx, receipt, msg)
}

// continueQueuedInput moves a person's queued assignment input toward its
// delivery. Waiting assignments resume through their ordinary continuation,
// whose activation turn carries the input; input whose assignment is already
// executing again (its admission finished before the input could join it) is
// handed to the runtime as an ordinary follow-up; input for ended work is
// released to explicit recovery. It runs on acceptance, at turn end and on the
// dispatcher tick, so restart and missed events converge (TS-09.R69/R70).
func (s *Server) continueQueuedInput() {
	prepared, executing, err := s.stateStore.QueuedInputWork()
	if err != nil {
		s.log.Warn("queued assignment input pass failed", "err", err)
		return
	}
	for _, task := range prepared {
		s.publishTaskUpdate(task)
	}
	if len(prepared) > 0 {
		go s.dispatchReadyTasks(context.Background())
	}
	for _, queued := range executing {
		// A busy turn may still yield to wait; its continuation then carries
		// the input. Only an executing assignment with no turn takes it now.
		if !s.registry.Owns(queued.AgentID) {
			continue
		}
		if st, err := s.stateStore.ReadStatus(queued.AgentID); err != nil || st.State != "idle" {
			continue
		}
		rec, ok, err := s.stateStore.ClaimQueuedInputForTurn(queued.SourceAgentID, queued.RequestID)
		if err != nil {
			s.log.Warn("claim queued assignment input", "agent", boundedLogID(queued.SourceAgentID), "request_id", boundedLogID(queued.RequestID), "reason", "claim_failed")
			continue
		}
		if !ok {
			continue
		}
		var msg runtime.Message
		if err := json.Unmarshal([]byte(rec.MessageJSON), &msg); err != nil {
			s.setQueuedInputDelivery(rec, "failed", "message_decode_failed")
			continue
		}
		// Input accepted by a replaced owner carries its attachments to the
		// assignment's new owner (TS-09.R69).
		if err := runtime.CloneQueuedAttachments(s.configStore.Home(), rec, msg); err != nil {
			s.setQueuedInputDelivery(rec, "failed", "attachment_copy_failed")
			continue
		}
		msg.RequestID = rec.RequestID
		if _, err := s.registry.SendMessageOrHold(context.Background(), rec.AgentID, msg); err != nil {
			// The runtime records its own transition when it touched the
			// receipt; an untouched one is offered back for recovery.
			current, readErr := s.stateStore.AttachmentReceipt(rec.AgentID, rec.RequestID)
			if readErr != nil {
				s.log.Warn("read queued input after delivery refusal", "agent", boundedLogID(rec.AgentID), "request_id", boundedLogID(rec.RequestID), "reason", "receipt_read_failed")
			} else if current.Delivery == "prepared" {
				s.setQueuedInputDelivery(rec, "released", "turn_refused")
			}
		}
	}
}

func (s *Server) setQueuedInputDelivery(rec state.AttachmentReceipt, delivery, reason string) {
	if err := s.stateStore.SetAttachmentDelivery(rec.AgentID, rec.RequestID, delivery); err != nil {
		s.log.Warn("record queued input delivery", "agent", boundedLogID(rec.AgentID), "request_id", boundedLogID(rec.RequestID), "delivery", delivery, "reason", "write_failed")
		return
	}
	s.log.Warn("queued input delivery settled", "agent", boundedLogID(rec.AgentID), "request_id", boundedLogID(rec.RequestID), "delivery", delivery, "reason", reason)
}

// continueRunFromConversation lets a person's message to the owner of a paused
// run answer or resume that run. The full message stays prepared for that
// assignment's turn. When no paused run awaits this owner, admission can
// proceed as ordinary chat (FS-14.R99/R100).
func (s *Server) continueRunFromConversation(ctx context.Context, receipt state.AttachmentReceipt, msg runtime.Message) (state.AttachmentReceipt, bool, error) {
	if s.pipelineMgr == nil {
		return state.AttachmentReceipt{}, false, nil
	}
	ok, err := s.pipelineMgr.ContinueFromOwner(ctx, strings.TrimSpace(msg.Text), receipt)
	if err != nil {
		if replay, accepted, replayErr := s.replayConversationInput(receipt, nil); accepted || replayErr != nil {
			return replay, accepted, replayErr
		}
		reason := "store_failure"
		var control *pipeline.ControlError
		if errors.As(err, &control) {
			reason = control.Code
		}
		s.log.Warn("refused paused run input", "agent", boundedLogID(receipt.AgentID),
			"request_id", boundedLogID(receipt.RequestID), "reason", reason)
		return state.AttachmentReceipt{}, false, err
	}
	if !ok {
		// A concurrent copy of this command may already have advanced the run.
		if replay, accepted, err := s.replayConversationInput(receipt, nil); accepted || err != nil {
			return replay, accepted, err
		}
		return state.AttachmentReceipt{}, false, nil
	}
	recorded, err := s.stateStore.AcceptedAttachmentReceipt(receipt.AgentID, receipt.RequestID)
	if err != nil {
		return receipt, true, err
	}
	s.logRunInputTransition(recorded)
	return recorded, true, nil
}

func (s *Server) replayConversationInput(receipt state.AttachmentReceipt, cause error) (state.AttachmentReceipt, bool, error) {
	previous, err := s.stateStore.AcceptedAttachmentReceipt(receipt.AgentID, receipt.RequestID)
	if err != nil {
		return state.AttachmentReceipt{}, false, cause
	}
	if previous.PayloadHash != receipt.PayloadHash {
		return state.AttachmentReceipt{}, false, apiError(runtime.CodeConflict, "request id already belongs to different content")
	}
	return previous, true, nil
}

// logRunInputTransition records which visit accepted a chat Send that continued
// a paused run, matching the runtime's receipt transition log. Identities and
// fixed vocabulary only: no message text, attachment metadata or provider
// errors (TS-02.R55, INV §15).
func (s *Server) logRunInputTransition(receipt state.AttachmentReceipt) {
	runID := ""
	if lineage, err := s.stateStore.ReadTaskLineage(receipt.AssignmentTaskID); err == nil {
		runID = lineage.PipelineRunID
	}
	s.log.Info("conversation input continued paused run", "agent", boundedLogID(receipt.AgentID),
		"request_id", boundedLogID(receipt.RequestID), "run", boundedLogID(runID),
		"visit", boundedLogID(receipt.AssignmentTaskID), "surface", "chat", "delivery", receipt.Delivery)
}

// logQueuedAssignmentInput records that a chat Send was retained for the
// agent's assignment instead of reaching the transcript, with the assignment's
// state, so a report of input shown under the composer names the branch that
// ran. Identities and fixed vocabulary only (INV §15/§22).
func (s *Server) logQueuedAssignmentInput(receipt state.AttachmentReceipt) {
	taskState := "unknown"
	if task, err := s.stateStore.ReadTask(receipt.AssignmentTaskID); err == nil {
		taskState = task.State
	}
	s.log.Info("conversation input queued for assignment", "agent", boundedLogID(receipt.AgentID),
		"request_id", boundedLogID(receipt.RequestID), "task", boundedLogID(receipt.AssignmentTaskID),
		"task_state", taskState)
}

func boundedLogID(id string) string {
	const limit = 128
	if len(id) > limit {
		return id[:limit]
	}
	return id
}

// recoverStageTask hands an interrupted pipeline owner to the controller's
// bounded automatic recovery (TS-09.R70). A replacement takes over the
// person's pending input with its assignment, structure and attachments
// intact, in the same transaction (TS-09.R69).
func (s *Server) recoverStageTask(taskID, attention string, replace bool) {
	if s.pipelineMgr == nil {
		return
	}
	if _, err := s.pipelineMgr.RecoverStageTask(taskID, attention, replace); err != nil {
		s.log.Warn("recover pipeline stage task", "task", taskID, "err", err)
	}
}

// recoverParkedStageTask replaces a pipeline owner whose start or wake failed
// for good: its native session is unusable, so a fresh same-runtime owner
// continues (FS-14.R101).
func (s *Server) recoverParkedStageTask(task state.Task) {
	if task.State != state.TaskDependencyFailed || s.pipelineMgr == nil {
		return
	}
	if ok, err := s.stateStore.InterruptParkedStageTask(task.TaskID); err != nil {
		s.log.Warn("interrupt parked pipeline owner", "task", task.TaskID, "err", err)
		return
	} else if !ok {
		return
	}
	s.recoverStageTask(task.TaskID, "owner_unavailable", true)
}
