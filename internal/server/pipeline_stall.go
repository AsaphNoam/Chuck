package server

import (
	"errors"

	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// ownerStallPrompt continues a pipeline owner whose turn ended with its stage
// visit still open and neither a report nor a wait recorded (TS-09.R70).
const ownerStallPrompt = "Your pipeline assignment is still open, but your last turn ended without report_task_result or wait_for_tasks. " +
	"Call get_assigned_task, then continue the work, wait for child tasks with wait_for_tasks, or finish with report_task_result (adding pipeline_decision when needed). " +
	"If only the person can unblock you, report with pipeline_decision action needs_input and the concrete question."

func (s *Server) nudgeStalledOwner(agentID, generation string) {
	if s.pipelineMgr == nil || s.ownerStallNudgeOff {
		return
	}
	visit, task, err := s.stateStore.PipelineStageTaskForAssignee(agentID, generation)
	if errors.Is(err, state.ErrNotFound) || err == nil && (task.State != state.TaskRunning || task.PendingYield || task.PendingRelease) {
		return
	}
	if err != nil {
		s.log.Warn("read pipeline owner for stall check", "agent", agentID, "generation", generation, "err", err)
		return
	}
	// A settled turn that failed while the process lives is as stalled as an
	// idle one; the runtime gate below still refuses a stopped, stopping or
	// quota-held owner (FS-14.R101).
	st, err := s.stateStore.ReadStatus(agentID)
	if err != nil || st.State != "idle" && st.State != "error" || st.BackgroundRunning > 0 {
		return
	}
	if blocked, err := s.quotaBlocksAgent(agentID); err != nil || blocked {
		return
	}
	if !s.claimLifecycle(agentID) {
		return
	}

	// The runtime gate prevents a concurrent check or a person's Send from
	// starting a turn between counting the settled turn and claiming its nudge.
	// A refused nudge can be retried against the same durable turn key.
	var interrupted state.Task
	err = s.registry.ContinueSettledChat(agentID, generation, func(turnID string) (string, error) {
		_, count, err := s.stateStore.RecordOwnerStall(task.TaskID, generation, turnID)
		if err != nil {
			return "", err
		}
		if count < state.MaxOwnerStallTurns {
			return ownerStallPrompt, nil
		}
		interrupted, _, err = s.stateStore.InterruptPipelineOwner(task.TaskID, generation,
			"the run owner ended several turns without reporting or waiting", "owner_not_reporting")
		return "", err
	})
	s.releaseLifecycle(agentID)
	if err != nil && !errors.Is(err, state.ErrNotFound) && !errors.Is(err, runtime.ErrNoHandle) {
		s.log.Warn("continue stalled pipeline owner", "agent", agentID, "generation", generation, "run", visit.RunID, "task", task.TaskID, "reason", "continuation_failed")
	}
	if interrupted.TaskID != "" {
		s.publishTaskUpdate(interrupted)
		s.pauseInterruptedOwner(task.TaskID, "owner_not_reporting")
	}
}

func (s *Server) pauseInterruptedOwner(taskID, attention string) {
	if err := s.pipelineMgr.OnStageTaskInterrupted(taskID, attention); err != nil {
		s.log.Warn("pause pipeline for interrupted owner", "task", taskID, "reason", attention, "err", err)
	}
}

// recoverMissedStageStalls rechecks current visits from durable state. A
// busy owner, quota sleep, child wait or active background command cannot
// claim another turn, and an explicit owner stop is never restarted.
func (s *Server) recoverMissedStageStalls() {
	owners, err := s.stateStore.PipelineOwnersNeedingRecovery(s.pipelineRecoveryAfter, taskDispatchBatch)
	if err != nil {
		s.log.Warn("list pipeline owners needing recovery", "err", err)
		return
	}
	if len(owners) == taskDispatchBatch {
		s.pipelineRecoveryAfter = owners[len(owners)-1].TaskID
	} else {
		s.pipelineRecoveryAfter = ""
	}
	for _, owner := range owners {
		switch owner.Action {
		case "check_stall":
			s.nudgeStalledOwner(owner.AgentID, owner.Generation)
		case "bind":
			s.bindPipelineOwner(owner.TaskID, owner.AgentID, owner.Generation)
		case "parked":
			s.recoverParkedStageTask(state.Task{TaskID: owner.TaskID, State: state.TaskDependencyFailed})
		case "owner_unavailable", "restart_interrupted":
			s.recoverStageTask(owner.TaskID, owner.Action, owner.Action == "owner_unavailable")
		case "unexpected_exit":
			s.recoverStageTask(owner.TaskID, "interrupted", false)
		default:
			s.pauseInterruptedOwner(owner.TaskID, owner.Action)
		}
	}
}
