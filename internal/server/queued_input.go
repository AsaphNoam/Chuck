package server

import (
	"context"
	"encoding/json"

	"github.com/AsaphNoam/Chuck/internal/runtime"
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
