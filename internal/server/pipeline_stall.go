package server

import (
	"context"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ownerStallPrompt continues a pipeline owner whose turn ended with its stage
// visit still open and neither a report nor a wait recorded. It is ordinary
// continuation of the same assignment, not a status poll (TS-09.R70).
const ownerStallPrompt = "Your pipeline assignment is still open, but your last turn ended without report_task_result or wait_for_tasks. " +
	"Call get_assigned_task, then continue the work, wait for child tasks with wait_for_tasks, or finish with report_task_result (adding pipeline_decision when needed). " +
	"If only the person can unblock you, report with pipeline_decision action needs_input and the concrete question."

// nudgeStalledOwner gives an owner whose turn ended without a report or wait
// one more ordinary turn, up to state.MaxOwnerStallTurns consecutive turns;
// then the visit is interrupted and the run pauses for the person. Legitimate
// silence — a quota sleep, a running background command, another lifecycle
// transition or a person's own follow-up — is left alone (FS-14.R101).
func (s *Server) nudgeStalledOwner(agentID, generation string) {
	if s.pipelineMgr == nil || s.ownerStallNudgeOff {
		return
	}
	_, task, err := s.stateStore.PipelineStageTaskForAssignee(agentID, generation)
	if err != nil || task.State != state.TaskRunning || task.PendingYield || task.PendingRelease {
		return
	}
	if blocked, err := s.quotaBlocksAgent(agentID); err != nil || blocked {
		return
	}
	if st, err := s.stateStore.ReadStatus(agentID); err != nil || st.State != "idle" || st.BackgroundRunning > 0 {
		return
	}
	if s.lifecycleInFlight(agentID) {
		return
	}
	_, count, err := s.stateStore.RecordOwnerStall(task.TaskID)
	if err != nil {
		return
	}
	if count >= state.MaxOwnerStallTurns {
		interrupted, ok, err := s.stateStore.InterruptTask(task.TaskID, "the run owner ended several turns without reporting or waiting")
		if err != nil || !ok {
			return
		}
		s.publishTaskUpdate(interrupted)
		if err := s.pipelineMgr.OnStageTaskInterrupted(task.TaskID, "owner_not_reporting"); err != nil {
			s.log.Warn("pause pipeline for stalled owner", "task", task.TaskID, "err", err)
		}
		return
	}
	// A person's held follow-up may have started in the meantime; that turn is
	// the owner's continuation, so a refused nudge is simply dropped.
	if err := s.registry.SendPrompt(context.Background(), agentID, ownerStallPrompt); err != nil {
		s.log.Debug("nudge stalled pipeline owner", "agent", agentID, "err", err)
	}
}
