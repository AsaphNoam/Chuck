package server

import (
	"context"
	"encoding/json"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

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
		rec, ok, err := s.stateStore.ClaimQueuedInputForTurn(queued.AgentID, queued.RequestID)
		if err != nil || !ok {
			continue
		}
		var msg runtime.Message
		if err := json.Unmarshal([]byte(rec.MessageJSON), &msg); err != nil {
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

// recoverStageTask hands an interrupted pipeline owner to the controller's
// bounded automatic recovery (TS-09.R70).
func (s *Server) recoverStageTask(taskID, attention string, replace bool) {
	if s.pipelineMgr == nil {
		return
	}
	handoff := ""
	var receipts []state.AttachmentReceipt
	if replace {
		// Pending accepted input belongs in the replacement's handoff; the
		// old identity will never run this assignment again (TS-09.R69).
		receipts, _ = s.stateStore.QueuedInputForTask(taskID)
		for _, rec := range receipts {
			var msg runtime.Message
			if json.Unmarshal([]byte(rec.MessageJSON), &msg) == nil && msg.Text != "" {
				handoff += "The person sent the previous owner this message, which it never received: " + msg.Text + "\n"
			}
		}
	}
	replaced, err := s.pipelineMgr.RecoverStageTask(taskID, attention, replace, handoff)
	if err != nil {
		s.log.Warn("recover pipeline stage task", "task", taskID, "err", err)
		return
	}
	if replaced && len(receipts) > 0 {
		if err := s.stateStore.MarkQueuedInputHandedOff(receipts); err != nil {
			s.log.Warn("mark queued input handed off", "task", taskID, "err", err)
		}
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
