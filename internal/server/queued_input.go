package server

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// admitConversationInput is shared by ordinary messages and self annotations:
// assignment ownership and run continuation precede any standalone runtime turn.
func (s *Server) admitConversationInput(ctx context.Context, receipt state.AttachmentReceipt, msg runtime.Message) (state.AttachmentReceipt, bool, error) {
	taskID, queued, err := s.stateStore.QueueAssignmentInput(receipt)
	if err != nil {
		return state.AttachmentReceipt{}, false, err
	}
	if queued {
		receipt.Delivery, receipt.AssignmentTaskID = "prepared", taskID
		return receipt, true, nil
	}
	continued, ok := s.continueRunFromConversation(ctx, receipt, msg)
	return continued, ok, nil
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
		s.log.Debug("queued assignment input pass failed", "err", err)
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
		if err != nil || !ok {
			continue
		}
		var msg runtime.Message
		if err := json.Unmarshal([]byte(rec.MessageJSON), &msg); err != nil {
			_ = s.stateStore.SetAttachmentDelivery(rec.AgentID, rec.RequestID, "failed")
			continue
		}
		// Input accepted by a replaced owner carries its attachments to the
		// assignment's new owner (TS-09.R69).
		if err := runtime.CloneQueuedAttachments(s.configStore.Home(), rec, msg); err != nil {
			_ = s.stateStore.SetAttachmentDelivery(rec.AgentID, rec.RequestID, "failed")
			continue
		}
		msg.RequestID = rec.RequestID
		if _, err := s.registry.SendMessageOrHold(context.Background(), rec.AgentID, msg); err != nil {
			// The runtime records its own transition when it touched the
			// receipt; an untouched one is offered back for recovery.
			if current, readErr := s.stateStore.AttachmentReceipt(rec.AgentID, rec.RequestID); readErr == nil && current.Delivery == "prepared" {
				_ = s.stateStore.SetAttachmentDelivery(rec.AgentID, rec.RequestID, "released")
			}
		}
	}
}

// continueRunFromConversation lets a person's message to the owner of a paused
// run answer or resume that run. The message is recorded with the visit that
// carries it: words the continuation already carries are submitted with it,
// and anything else stays prepared for that assignment's turn. It reports
// false, recording nothing, when no paused run awaits this owner (FS-14.R99/R100).
func (s *Server) continueRunFromConversation(ctx context.Context, receipt state.AttachmentReceipt, msg runtime.Message) (state.AttachmentReceipt, bool) {
	if s.pipelineMgr == nil {
		return state.AttachmentReceipt{}, false
	}
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		text = "(their answer is in the attached files)"
	}
	recorded := receipt
	ok, err := s.pipelineMgr.ContinueFromOwner(ctx, receipt.AgentID, text, func(taskID string, textCarried bool) error {
		recorded.AssignmentTaskID = taskID
		if textCarried && len(msg.Attachments) == 0 {
			recorded.Delivery = "submitted"
			if err := s.stateStore.CreateAttachmentReceipt(recorded); err != nil {
				return err
			}
			return s.stateStore.SetAttachmentDelivery(recorded.AgentID, recorded.RequestID, "submitted")
		}
		if textCarried {
			// The visit's continuation carries the words; its turn carries
			// the attachments.
			attachmentsOnly := runtime.Message{Attachments: msg.Attachments}
			data, err := json.Marshal(attachmentsOnly)
			if err != nil {
				return err
			}
			recorded.MessageJSON = string(data)
		}
		recorded.Delivery = "prepared"
		return s.stateStore.BindAssignmentInput(recorded, taskID)
	})
	if err != nil {
		s.log.Warn("continue paused run from conversation", "agent", receipt.AgentID, "err", err)
		return state.AttachmentReceipt{}, false
	}
	return recorded, ok
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
	if ok, err := s.stateStore.InterruptParkedStageTask(task.TaskID); err != nil || !ok {
		return
	}
	s.recoverStageTask(task.TaskID, "owner_unavailable", true)
}
