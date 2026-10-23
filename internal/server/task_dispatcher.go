package server

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

const (
	// taskDispatchInterval is the dispatcher's ticker. Durable rows are the
	// authority, so a missed notification costs latency, never correctness
	// (TS-10.R2).
	taskDispatchInterval = 2 * time.Second
	// taskDispatchBatch bounds one pass both ways: how many ready rows it reads
	// and how many starts it may have in flight across passes (TS-10.R3).
	taskDispatchBatch = 8
)

// startTaskDispatcher runs the admission loop beside the activation executor,
// in this process and with no second scheduler (TS-10.R1).
func (s *Server) startTaskDispatcher(ctx context.Context) {
	go s.runTaskDispatcher(ctx)
}

func (s *Server) runTaskDispatcher(ctx context.Context) {
	ticker := time.NewTicker(taskDispatchInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.repairQuotaNotices(ctx); err != nil {
				s.log.Error("repair quota continuation notices", "err", err)
			}
			s.dispatchQuotaContinuations(ctx)
			s.reconcileTaskCleanup(ctx)
			if s.pipelineMgr != nil {
				s.pipelineMgr.ReconcilePending(ctx)
			}
			s.reconcileTaskWaits()
			s.resolveDueTimeArms()
			s.processDueSchedules()
			s.recoverStartingTasks(ctx)
			s.dispatchReadyTasks(ctx)
		}
	}
}

// recoverStartingTasks retries the bounded set of reservations that startup
// could not reap. A task-start lock is acquired before the durable row is
// re-read, so an in-process start either finishes first or is prevented from
// beginning while recovery owns the reservation (INV §5, §15).
func (s *Server) recoverStartingTasks(ctx context.Context) {
	s.taskStartMu.Lock()
	retries := make(map[string]string, taskDispatchBatch)
	ids := make([]string, 0, taskDispatchBatch)
	for _, taskID := range s.startupStartRetryOrder {
		if len(ids) == taskDispatchBatch {
			break
		}
		attemptID, ok := s.startupStartRetries[taskID]
		if !ok {
			continue
		}
		retries[taskID] = attemptID
		ids = append(ids, taskID)
	}
	s.taskStartMu.Unlock()
	for _, taskID := range ids {
		attemptID := retries[taskID]
		unlock, ok := s.tryLockTaskStart(taskID)
		if !ok {
			s.taskStartMu.Lock()
			s.rotateStartupStartRetryLocked(taskID)
			s.taskStartMu.Unlock()
			continue
		}
		fresh, err := s.stateStore.ReadTask(taskID)
		retain := err != nil && !errors.Is(err, state.ErrNotFound)
		if err == nil && fresh.State == state.TaskStarting && fresh.StartAttemptID == attemptID {
			retain = s.recoverStartAttempt(ctx, fresh)
		} else if err == nil {
			// The reservation settled or was superseded; it no longer belongs to
			// startup recovery.
			retain = false
		}
		s.taskStartMu.Lock()
		if retain {
			s.rotateStartupStartRetryLocked(taskID)
		} else {
			delete(s.startupStartRetries, taskID)
			s.removeStartupStartRetryLocked(taskID)
		}
		s.taskStartMu.Unlock()
		unlock()
	}
}

// dispatchReadyTasks admits ready work in the order it became ready, up to the
// budget. Admission is serial and the effect is not: the admitting statement is
// one fast write whose order is the promise (FS-16.R7), while a launch takes
// seconds and must not hold the pass behind it.
func (s *Server) dispatchReadyTasks(ctx context.Context) {
	if s.registry == nil {
		return
	}
	s.taskSweepMu.Lock()
	defer s.taskSweepMu.Unlock()
	ready, err := s.stateStore.ReadyTasksAfter(taskDispatchBatch, s.taskSweepAfter)
	if err != nil {
		s.log.Debug("task dispatch list ready failed", "err", err)
		return
	}
	advanced := false
	budget := s.taskConcurrencyBudget()
	for _, task := range ready {
		plan, ok := s.planTaskStart(task)
		if !ok {
			continue
		}
		// Take the worker slot before admitting, so a full worker pool leaves the
		// task untouched and still ready rather than starting-with-nobody-working.
		select {
		case s.taskStartSlots <- struct{}{}:
		default:
			s.taskSweepAfter = state.Task{}
			return
		}
		admitted, ok, err := s.stateStore.AdmitReadyTask(task.TaskID, plan, budget)
		if err != nil {
			s.log.Debug("task admission failed", "task", task.TaskID, "err", err)
		}
		if err != nil || !ok {
			// Not admitted: no capacity, another task holds this agent, or the row
			// is no longer ready. None of those spends an attempt (FS-16.R25).
			<-s.taskStartSlots
			continue
		}
		advanced = true
		go func(task state.Task) {
			defer func() { <-s.taskStartSlots }()
			s.startAdmittedTask(ctx, task)
		}(admitted)
	}
	if len(ready) == 0 || advanced {
		s.taskSweepAfter = state.Task{}
	} else {
		last := ready[len(ready)-1]
		s.taskSweepAfter = state.Task{TaskID: last.TaskID, ReadyAt: last.ReadyAt}
	}
}

// planTaskStart decides what this start would do to a runtime before anything is
// claimed: the agent it acts on, the generation it acts under, and whether it
// creates, wakes, or borrows that runtime (TS-10.R4).
func (s *Server) planTaskStart(task state.Task) (state.TaskReservation, bool) {
	attemptID := "ta_" + mintHookToken()
	// A task that already confirmed an assignee acts on that same agent, whatever
	// its target kind. A retried launch-spec task resumes the agent it created
	// rather than minting a second one, which would fork its transcript and its
	// attached-context membership (FS-16.R23).
	if task.AssignedAgentID != "" {
		return s.planExistingAgentStart(task, attemptID, task.AssignedAgentID)
	}
	switch task.TargetKind {
	case state.TargetLaunch:
		agentID, err := s.stateStore.NewAgentID()
		if err != nil {
			s.log.Debug("mint task agent failed", "task", task.TaskID, "err", err)
			return state.TaskReservation{}, false
		}
		// The generation is the attempt id, as it is for a pipeline stage: one
		// start attempt is exactly one launch generation.
		return state.TaskReservation{
			AttemptID: attemptID, AgentID: agentID, Generation: attemptID,
			Claim: state.ClaimCreated,
		}, true
	default:
		return s.planExistingAgentStart(task, attemptID, task.TargetAgentID)
	}
}

// planExistingAgentStart decides how this start would treat an agent that
// already exists. Borrowing a runtime that is already up brings up no process
// and takes no budget; waking a stopped one does. Eligibility is deliberately
// not decided here: a target that can never be started parks its task, and
// parking is a transition out of starting, so it happens once the reservation
// exists.
func (s *Server) planExistingAgentStart(task state.Task, attemptID, agentID string) (state.TaskReservation, bool) {
	if blocked, err := s.quotaBlocksAgent(agentID); err != nil || blocked {
		return state.TaskReservation{}, false
	}
	claim := state.ClaimWoke
	generation := attemptID
	if _, err := s.stateStore.ReadRunning(agentID); err == nil {
		claim = state.ClaimBorrowed
		generation = s.taskGeneration(agentID)
		if generation == "" {
			s.log.Debug("task target runtime has no local generation", "task", task.TaskID, "agent", agentID)
			return state.TaskReservation{}, false
		}
	} else if !errors.Is(err, state.ErrNotFound) {
		s.log.Debug("task target liveness read failed", "task", task.TaskID, "err", err)
		return state.TaskReservation{}, false
	}
	return state.TaskReservation{
		AttemptID: attemptID, AgentID: agentID, Generation: generation, Claim: claim,
	}, true
}

// startAdmittedTask performs the effect the reservation authorized. A task that
// brings up its own agent launches it; one that targets an existing agent crosses
// into that conversation through the dependency activation kind (FS-16.R6).
func (s *Server) startAdmittedTask(ctx context.Context, task state.Task) {
	unlock := s.lockTaskStart(task.TaskID)
	defer unlock()
	// Admission and the effect are deliberately separated, so cancellation can
	// finish in between. Re-read under the same server claim cancel takes before
	// launching, resuming, or prompting (TS-10.R4, INV §5).
	fresh, err := s.stateStore.ReadTask(task.TaskID)
	if err != nil || fresh.State != state.TaskStarting || fresh.StartAttemptID != task.StartAttemptID {
		return
	}
	task = fresh
	if task.RuntimeClaim == state.ClaimCreated {
		s.startLaunchedTask(ctx, task)
		return
	}
	s.startExistingAgentTask(ctx, task)
}

// lockTaskStart serializes only one task's effect with its cancellation. The
// dispatcher may still launch independent tasks concurrently up to its worker
// and runtime budgets (FS-16.R7, INV §5).
func (s *Server) lockTaskStart(taskID string) func() {
	s.taskStartMu.Lock()
	lock := s.taskStartLocks[taskID]
	if lock == nil {
		lock = &taskStartLock{}
		s.taskStartLocks[taskID] = lock
	}
	lock.refs++
	s.taskStartMu.Unlock()

	lock.mu.Lock()
	return func() {
		lock.mu.Unlock()
		s.taskStartMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(s.taskStartLocks, taskID)
		}
		s.taskStartMu.Unlock()
	}
}

func (s *Server) tryLockTaskStart(taskID string) (func(), bool) {
	s.taskStartMu.Lock()
	lock := s.taskStartLocks[taskID]
	if lock == nil {
		lock = &taskStartLock{}
		s.taskStartLocks[taskID] = lock
	}
	lock.refs++
	s.taskStartMu.Unlock()
	if !lock.mu.TryLock() {
		s.taskStartMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(s.taskStartLocks, taskID)
		}
		s.taskStartMu.Unlock()
		return nil, false
	}
	return func() {
		lock.mu.Unlock()
		s.taskStartMu.Lock()
		lock.refs--
		if lock.refs == 0 {
			delete(s.taskStartLocks, taskID)
		}
		s.taskStartMu.Unlock()
	}, true
}

// startLaunchedTask brings up a new agent for a launch-spec task and then settles
// the row: confirmed into running, given back on a deferral, or spent as one of
// the bounded attempts on a real failure (FS-16.R25, TS-10.R4).
func (s *Server) startLaunchedTask(ctx context.Context, task state.Task) {
	// Launch mints the same agent-keyed registration as resume, so the claim
	// covers the launch and its first assignment prompt (TS-01.R16, INV §4/§5).
	if !s.claimLifecycle(task.AssignedAgentID) {
		s.deferTaskStart(task, "lifecycle busy")
		return
	}
	defer s.releaseLifecycle(task.AssignedAgentID)

	// A known orchestrator never launches without its frozen guidance; the
	// bounded start attempts surface a persistent failure (TS-09.R58).
	instructions, err := s.pipelineOrchestratorInstructions(task)
	if err != nil {
		s.failTaskStart(task, "launch failed: "+err.Error())
		return
	}
	_, ae := s.launchAgent(ctx, launchRequest{
		Role: task.Role, Project: task.Project, Backend: task.Backend, Model: task.Model,
		Effort: task.Effort, Fast: task.Fast, Interface: "chat", Name: task.DisplayName, Group: task.Group,
	}, launchOptions{AgentID: task.AssignedAgentID, Generation: task.AssignedGeneration, OrchestratorInstructions: instructions})
	if ae != nil {
		s.failTaskStart(task, "launch failed: "+ae.Message)
		return
	}
	// The task's instruction is the assignment input FS-00.R14–R15 permits, sent
	// as an ordinary turn exactly as a pipeline stage's assignment is, with the
	// shared reporting paragraph after it (FS-16.R49).
	if err := s.registry.SendPrompt(ctx, task.AssignedAgentID, runtime.LaunchedTaskPrompt(task.Instruction)); err != nil {
		// The claim is already held, so use the unclaimed stop core.
		if stopErr := s.stopStageLocked(ctx, task.AssignedAgentID); stopErr != nil {
			s.log.Debug("stop failed task runtime failed", "agent", task.AssignedAgentID, "err", stopErr)
			// The starting reservation is the only durable ownership of this
			// runtime. Keep it until recovery can stop the runtime (INV §4/§15).
			return
		}
		s.failTaskStart(task, "assignment was not delivered: "+err.Error())
		return
	}
	s.recordTaskExecutionTurn(task)
	s.confirmTaskStart(task)
}

// startExistingAgentTask crosses into an agent's existing conversation with one
// host-owned dependency activation: the same primitive mail uses, with its own
// instruction and its own retry policy. The activation stays actionable until the
// task confirms its start, so a deferral returns both the row and the task rather
// than spending either (TS-10.R5, R6).
func (s *Server) startExistingAgentTask(ctx context.Context, task state.Task) {
	agent, err := s.stateStore.ReadAgent(task.AssignedAgentID)
	if errors.Is(err, state.ErrNotFound) {
		s.parkTaskStart(task, "the target agent no longer exists")
		return
	}
	if err != nil {
		s.deferTaskStart(task, "read target agent")
		return
	}
	// FS-07's messaging boundary: a terminal agent has no model turn to give, so
	// this can never become startable and retrying it is not a repair (FS-16.R2).
	if agent.Interface != "chat" {
		s.parkTaskStart(task, "the target agent runs a terminal interface and cannot execute work")
		return
	}
	if agent.Archived {
		s.parkTaskStart(task, "the target agent is archived")
		return
	}
	activation, err := s.stateStore.EnsurePendingDependencyActivation(task.AssignedAgentID, task.TaskID)
	if err != nil {
		s.log.Debug("ensure dependency activation failed", "task", task.TaskID, "err", err)
		s.deferTaskStart(task, "ensure activation")
		return
	}
	token, claimed, err := s.stateStore.ClaimActivation(state.ActivationKindDependency, activation.ActivationID)
	if err != nil {
		s.log.Debug("claim dependency activation failed", "task", task.TaskID, "err", err)
	}
	if err != nil || !claimed {
		s.deferTaskStart(task, "claim activation")
		return
	}

	running := false
	if _, err := s.stateStore.ReadRunning(task.AssignedAgentID); err == nil {
		running = true
	} else if !errors.Is(err, state.ErrNotFound) {
		s.log.Debug("task target liveness read failed", "task", task.TaskID, "err", err)
		s.settleActivationStart(task, activation, token, false, false)
		return
	}
	// The reservation decides whether finishing stops this runtime, so a target
	// that changed liveness between planning and starting is re-planned on the
	// next pass rather than started under a claim that is now wrong (FS-16.R4).
	if running != (task.RuntimeClaim == state.ClaimBorrowed) {
		s.settleActivationStart(task, activation, token, false, false)
		return
	}
	if running {
		generation := s.taskGeneration(task.AssignedAgentID)
		if generation == "" {
			s.settleActivationStart(task, activation, token, false, false)
			return
		}
		if generation != task.AssignedGeneration {
			updated, ok, err := s.stateStore.SetTaskStartGeneration(task.TaskID, task.StartAttemptID, generation)
			if err != nil || !ok {
				s.log.Debug("record borrowed task generation failed", "task", task.TaskID, "err", err)
				s.settleActivationStart(task, activation, token, false, false)
				return
			}
			task = updated
		}
		attempted, started := s.runActivationTurn(ctx, state.ActivationKindDependency, activation, token)
		if started {
			s.recordTaskExecutionTurn(task)
		}
		s.settleActivationStart(task, activation, token, attempted, started)
		return
	}
	// A stopped target is woken on exactly the terms mail is (FS-16.R19).
	_, ok, ae := s.wakeCandidate(task.AssignedAgentID)
	if ae != nil {
		s.log.Debug("task wake candidacy failed", "task", task.TaskID, "err", ae.Message)
		s.settleActivationStart(task, activation, token, false, false)
		return
	}
	if !ok {
		s.releaseDependencyActivation(activation, token)
		s.parkTaskStart(task, "the target agent cannot be woken for work")
		return
	}
	attempted, wakeErr := s.wakeForActivation(ctx, state.ActivationKindDependency, activation, token)
	if wakeErr == nil {
		generation := s.taskGeneration(task.AssignedAgentID)
		if generation == "" {
			s.log.Debug("woken task runtime has no generation", "task", task.TaskID)
			s.settleActivationStart(task, activation, token, attempted, false)
			return
		}
		if _, ok, err := s.stateStore.SetTaskStartGeneration(task.TaskID, task.StartAttemptID, generation); err != nil || !ok {
			s.log.Debug("record woken task generation failed", "task", task.TaskID, "err", err)
			s.settleActivationStart(task, activation, token, attempted, false)
			return
		}
		task, _ = s.stateStore.ReadTask(task.TaskID)
		s.recordTaskExecutionTurn(task)
	}
	s.settleActivationStart(task, activation, token, attempted, wakeErr == nil)
}

// recordTaskExecutionTurn captures the provider turn while it is still the
// task's active assignment turn. A later cleanup may cancel only this exact
// borrowed turn, never a human follow-up that starts after it settles.
func (s *Server) recordTaskExecutionTurn(task state.Task) {
	budget, err := s.stateStore.CurrentTurnBudget(task.AssignedAgentID, 1)
	if err != nil || budget.TurnID == "" {
		s.log.Debug("read task execution turn", "task", task.TaskID, "err", err)
		return
	}
	if _, ok, err := s.stateStore.SetTaskExecutionTurn(task.TaskID, task.StartAttemptID, task.AssignedGeneration, budget.TurnID); err != nil {
		s.log.Debug("record task execution turn", "task", task.TaskID, "err", err)
	} else if !ok {
		s.log.Debug("task execution turn became stale", "task", task.TaskID)
	}
}

// settleActivationStart resolves the task and its activation together. A turn
// that never crossed the attempt boundary spent nothing: both go back. A turn
// that crossed it consumed the opportunity, so the row is retired either way and
// only the task's fate differs (TS-10.R5, FS-16.R25).
func (s *Server) settleActivationStart(task state.Task, activation state.Activation, token string, attempted, started bool) {
	if !attempted {
		s.releaseDependencyActivation(activation, token)
		s.deferTaskStart(task, "the activation turn did not start")
		return
	}
	if started {
		s.confirmTaskStart(task)
	} else {
		s.failTaskStart(task, "the assignment turn did not complete")
	}
	if err := s.stateStore.RetireActivation(state.ActivationKindDependency, activation.ActivationID, token); err != nil {
		s.log.Debug("retire dependency activation failed", "activation", activation.ActivationID, "err", err)
	}
}

func (s *Server) releaseDependencyActivation(activation state.Activation, token string) {
	if err := s.stateStore.ReleaseActivation(state.ActivationKindDependency, activation.ActivationID, token); err != nil {
		s.log.Debug("release dependency activation failed", "activation", activation.ActivationID, "err", err)
	}
}

// confirmTaskStart is the only transition into running: the assignment is in a
// live runtime that holds it (FS-16.R6, TS-10.R4).
func (s *Server) confirmTaskStart(task state.Task) {
	if confirmed, ok, err := s.stateStore.ConfirmTaskStart(task.TaskID, task.StartAttemptID); err != nil {
		s.log.Debug("confirm task start failed", "task", task.TaskID, "err", err)
	} else if !ok {
		s.log.Debug("task start was settled by someone else", "task", task.TaskID)
	} else {
		// A pipeline association becomes standing-owner authority only after the
		// dispatcher confirmed this exact task/runtime assignment. A coordinator
		// or an unstarted task can therefore never report a stage by lineage alone.
		if err := s.stateStore.BindPipelineStageTaskStandingAgent(confirmed.TaskID, confirmed.AssignedAgentID); err == nil {
			// Binding may have moved the run from queued to running (FS-14.R37).
			if stage, err := s.stateStore.ReadPipelineStageTaskByTask(confirmed.TaskID); err == nil {
				if run, err := s.stateStore.ReadPipelineRun(stage.RunID); err == nil {
					s.publishPipelineRun(run)
				}
			}
		} else if !errors.Is(err, state.ErrNotFound) && !errors.Is(err, state.ErrPipelineStageConflict) {
			s.log.Debug("bind pipeline standing task failed", "task", confirmed.TaskID, "err", err)
		}
		s.publishTaskUpdate(confirmed)
	}
}

// parkTaskStart stops a task whose target can never become startable, spending no
// attempt, because retrying it is not a repair (FS-16.R8, R19).
func (s *Server) parkTaskStart(task state.Task, reason string) {
	if parked, ok, err := s.stateStore.ParkTaskStart(task.TaskID, task.StartAttemptID, reason); err != nil {
		s.log.Debug("park task start failed", "task", task.TaskID, "err", err)
	} else if ok {
		s.publishTaskUpdate(parked)
		s.propagateTaskFailure(parked)
	}
}

// deferTaskStart gives an admitted reservation back without spending an attempt:
// contention is not a failed start (FS-16.R25).
func (s *Server) deferTaskStart(task state.Task, why string) {
	if deferred, ok, err := s.stateStore.AbandonTaskStart(task.TaskID, task.StartAttemptID); err != nil {
		s.log.Debug("abandon task start failed", "task", task.TaskID, "why", why, "err", err)
	} else if ok {
		s.publishTaskUpdate(deferred)
	}
}

// failTaskStart spends one of the bounded attempts and parks the task on the
// last one, so a start that keeps failing surfaces instead of retrying forever
// (FS-16.R8, R25, INV §8).
func (s *Server) failTaskStart(task state.Task, reason string) {
	if failed, ok, err := s.stateStore.FailTaskStart(task.TaskID, task.StartAttemptID, reason); err != nil {
		s.log.Debug("fail task start failed", "task", task.TaskID, "err", err)
	} else if ok {
		s.publishTaskUpdate(failed)
		s.propagateTaskFailure(failed)
	}
}

// taskConcurrencyBudget reads the install-wide budget fresh, so changing it takes
// effect on the next pass without a restart (FS-16.R21).
func (s *Server) taskConcurrencyBudget() int {
	cfg := s.cfg
	if fromDisk, err := s.configStore.ReadConfig(); err == nil {
		cfg = fromDisk
	}
	if cfg.TaskConcurrency <= 0 {
		return config.DefaultTaskConcurrency
	}
	return cfg.TaskConcurrency
}

// dispatchTurnEnd fans one completed turn out to every control plane that waits
// for a turn boundary. Both wait for the same thing and for the same reason —
// an agent that reported a result is still receiving that call's response, so
// nothing may stop it until the turn ends — so this is one dispatch with two
// subscribers rather than two paths (TS-10.R19, TS-09.R9–R11, INV §2).
func (s *Server) dispatchTurnEnd(agentID, generation, turnID string) {
	s.releaseYieldedTask(context.Background(), agentID, generation, turnID)
	s.releaseReportedTask(context.Background(), agentID, generation, turnID)
	if s.pipelineMgr != nil {
		task, _, err := s.stateStore.PipelineStageTaskForAssignee(agentID, generation)
		if err == nil {
			if err := s.pipelineMgr.Reconcile(context.Background(), task.RunID); err != nil {
				s.pipelineMgr.RecordReconcileFailure(task.RunID, err)
				s.log.Warn("pipeline task release reconcile", "task", task.TaskID, "err", err)
			}
		}
	}
}

// releaseYieldedTask applies a durable wait only after the wait tool response's
// turn has ended. Created/woke runtimes release capacity through the ordinary
// stop path; borrowed conversations remain running and only release assignment
// execution ownership.
func (s *Server) releaseYieldedTask(ctx context.Context, agentID, generation, turnID string) {
	task, err := s.stateStore.PendingYieldTask(agentID, generation)
	if errors.Is(err, state.ErrNotFound) {
		return
	}
	if err != nil {
		s.log.Debug("read pending task yield failed", "agent", agentID, "err", err)
		return
	}
	s.finishTaskCleanupAt(ctx, task, turnID)
}

// releaseReportedTask completes the release a recorded result already promised:
// it stops a runtime this task created or woke and leaves a borrowed one alone,
// then drops the claim (FS-16.R4, TS-10.R19). A stop that cannot happen now
// leaves the durable intent standing for recovery to finish, because dropping
// the claim first would abandon a live runtime nothing owns (INV §15).
func (s *Server) releaseReportedTask(ctx context.Context, agentID, generation, turnID string) {
	task, err := s.stateStore.PendingReleaseTask(agentID, generation)
	if errors.Is(err, state.ErrNotFound) {
		return
	}
	if err != nil {
		s.log.Debug("read pending task release failed", "agent", agentID, "err", err)
		return
	}
	s.finishTaskCleanupAt(ctx, task, turnID)
}

// reconcileTaskCleanup is the bounded durable retry pass. A timer only wakes
// this existing dispatcher; rows and their deadlines remain authoritative.
func (s *Server) reconcileTaskCleanup(ctx context.Context) {
	due, err := s.stateStore.DueTaskCleanup(taskDispatchBatch)
	if err != nil {
		s.log.Debug("list due task cleanup", "err", err)
		return
	}
	for _, task := range due {
		s.finishTaskCleanup(ctx, task)
	}
}

// finishTaskCleanup is the sole release/yield effect seam. It serializes an
// effect with cancellation/start for this task and records a retryable failure
// before returning, so no live runtime loses its durable owner (TS-10.R35).
func (s *Server) finishTaskCleanup(ctx context.Context, task state.Task) {
	s.finishTaskCleanupAt(ctx, task, "")
}

// finishTaskCleanupAt is finishTaskCleanup at a turn boundary: endingTurn is
// the observed turn_end's own turn, which may still be settling when the
// owned runtime is stopped; a later turn of that generation keeps it.
func (s *Server) finishTaskCleanupAt(ctx context.Context, task state.Task, endingTurn string) {
	unlock := s.lockTaskStart(task.TaskID)
	defer unlock()
	fresh, err := s.stateStore.ReadTask(task.TaskID)
	if err != nil || (!fresh.PendingRelease && !fresh.PendingYield) {
		return
	}
	task = fresh
	// A busy deferral is not a failure backoff: the turn end it waited for
	// may settle the effect at once.
	busyDeferral := endingTurn != "" && task.CleanupFailureCount == 0
	if task.CleanupNextRetryAt != nil && task.CleanupNextRetryAt.After(time.Now().UTC()) && !busyDeferral {
		return
	}
	phase := "release"
	if task.PendingYield {
		phase = "yield"
	}
	if task.RuntimeClaim == state.ClaimBorrowed && task.Outcome == state.OutcomeCancelled {
		if task.ExecutionTurn == "" {
			s.recordTaskCleanupFailureClassified(task, phase, errors.New("missing guarded task turn for borrowed-runtime cancellation"), true)
			return
		}
		cancelCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		_, err := s.registry.CancelGuarded(cancelCtx, task.AssignedAgentID, task.AssignedGeneration, task.ExecutionTurn)
		cancel()
		if err != nil {
			s.recordTaskCleanupFailureClassified(task, phase, err, isUnsafeTaskCleanupError(err))
			return
		}
	}
	if task.RuntimeClaim == state.ClaimCreated || task.RuntimeClaim == state.ClaimWoke {
		settled, err := s.stopTaskRuntime(ctx, task, endingTurn)
		if err != nil {
			s.recordTaskCleanupFailureClassified(task, phase, err, isUnsafeTaskCleanupError(err))
			return
		}
		if !settled {
			// A turn of the owned generation is running; its turn end, the
			// generation's exit teardown or a later idle retry settles this
			// obligation. Moving the deadline is not a failure; it only keeps a
			// busy runtime from holding the head of the bounded retry batch.
			if err := s.stateStore.DeferTaskCleanup(task.TaskID, time.Now().UTC().Add(taskDispatchInterval)); err != nil {
				s.log.Debug("defer busy task cleanup", "task", task.TaskID, "err", err)
			}
			s.log.Debug("task cleanup waits for its runtime to go idle", "task", task.TaskID, "phase", phase)
			return
		}
	}
	if task.PendingYield {
		settled, err := s.stateStore.CompleteTaskYield(task.TaskID)
		if err != nil {
			s.recordTaskCleanupFailureClassified(task, phase, err, isUnsafeTaskCleanupError(err))
			return
		}
		s.publishTaskUpdate(settled)
		if settled.State == state.TaskReady {
			go s.dispatchReadyTasks(context.Background())
		}
		s.reconcilePipelineForTask(task)
		return
	}
	if err := s.stateStore.CompleteTaskRelease(task.TaskID); err != nil {
		s.recordTaskCleanupFailureClassified(task, phase, err, isUnsafeTaskCleanupError(err))
		return
	}
	// The release bumped the revision and may have cleared a cleanup attention
	// reason; clients holding the pre-release row refetch it (TS-10.R11).
	s.publishTaskUpdate(s.rereadTask(task))
	s.reconcilePipelineForTask(task)
}

// stopTaskRuntime stops the exact generation a task's runtime claim owns. A
// newer generation is never this obligation's to stop. The stop's authority
// comes from the committed intent, not from the caller: an agent-reported
// result or a yield is deferred, so even a timer retry with no observed turn
// stops a live owner only while idle or settling endingTurn; settled=false
// means it is still busy (TS-10.R19/R35/R48, INV §5).
func (s *Server) stopTaskRuntime(ctx context.Context, task state.Task, endingTurn string) (bool, error) {
	deferred := task.PendingYield || task.OutcomeSource == "agent"
	agentID, generation := task.AssignedAgentID, task.AssignedGeneration
	if !s.claimLifecycle(agentID) {
		return false, errors.New("a lifecycle transition is already in progress")
	}
	defer s.releaseLifecycle(agentID)
	switch current := s.registry.Generation(agentID); {
	case current == "" || generation == "":
		return true, s.stopStageLocked(ctx, agentID) // no live owner: reap any orphan
	case current != generation:
		return true, nil
	case !deferred:
		// A person-recorded result or cancel stops its own generation
		// immediately; it has no turn to wait for (R19).
		return true, s.stopStageLocked(ctx, agentID)
	}
	stopped, err := s.registry.StopGuardedIdle(ctx, agentID, generation, endingTurn)
	if err != nil || !stopped {
		return false, err
	}
	s.teardownAgentRegistration(agentID)
	return true, nil
}

// reconcilePipelineForTask re-drives the run a settled cleanup effect belongs to.
// A release completed here has no turn boundary behind it — it can be the retry
// timer finishing a transient stop failure, or startup recovery — so without
// this the run sits in `finishing`/`stopping` with nothing left to wake it
// (TS-09.R42/R43, INV §15).
func (s *Server) reconcilePipelineForTask(task state.Task) {
	if s.pipelineMgr == nil {
		return
	}
	lineage, err := s.stateStore.ReadTaskLineage(task.TaskID)
	if errors.Is(err, state.ErrNotFound) || (err == nil && lineage.PipelineRunID == "") {
		return
	}
	if err != nil {
		s.log.Debug("read cleanup task lineage", "task", task.TaskID, "err", err)
		return
	}
	if err := s.pipelineMgr.Reconcile(context.Background(), lineage.PipelineRunID); err != nil {
		s.pipelineMgr.RecordReconcileFailure(lineage.PipelineRunID, err)
		s.log.Warn("pipeline cleanup reconcile", "task", task.TaskID, "run", lineage.PipelineRunID, "err", err)
	}
}

func (s *Server) recordTaskCleanupFailure(task state.Task, phase string, cause error) {
	s.recordTaskCleanupFailureClassified(task, phase, cause, false)
}

func (s *Server) recordTaskCleanupFailureClassified(task state.Task, phase string, cause error, unsafe bool) {
	effectKey := phase + ":" + task.TaskID + ":" + task.AssignedGeneration
	updated, err := s.stateStore.RecordTaskCleanupFailureClassified(task.TaskID, phase, effectKey, cause, unsafe)
	if err != nil {
		s.log.Warn("record task cleanup failure", "task", task.TaskID, "err", err)
		return
	}
	s.publishTaskUpdate(updated)
	s.log.Debug("task cleanup deferred", "task", task.TaskID, "phase", phase, "err", cause)
}

func isUnsafeTaskCleanupError(err error) bool {
	if errors.Is(err, runtime.ErrNoHandle) {
		return true
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "permission denied") || strings.Contains(text, "uncorroborated")
}

// evaluateTaskResult releases the arms waiting on a task that just recorded its
// outcome. Evaluation reads the durable registration rather than an outcome
// handed to it, so a dropped or repeated notification changes nothing
// (TS-10.R3).
func (s *Server) evaluateTaskResult(taskID string) {
	s.evaluateSourceResult(state.SourceTask, taskID)
	s.notifyTaskWaiters(taskID)
}

// notifyTaskWaiters is the fast path for every actionable watched change — a
// result, an interruption, a park or a deletion (FS-16.R33, TS-10.R27). It is
// only a hint: reconcileTaskWaits rediscovers anything it drops (TS-10.R49).
func (s *Server) notifyTaskWaiters(taskID string) {
	changed, err := s.stateStore.NotifyTaskWaiters(taskID)
	if err != nil {
		s.log.Debug("notify task waiters failed", "task", taskID, "err", err)
		return
	}
	s.publishWokenWaiters(changed)
}

// reconcileTaskWaits wakes a bounded batch of waiters whose saved watches
// already name actionable work, so a crash or a lost hint never strands one.
func (s *Server) reconcileTaskWaits() {
	changed, err := s.stateStore.ReconcileTaskWaits(taskDispatchBatch)
	if err != nil {
		s.log.Debug("reconcile task waits failed", "err", err)
		return
	}
	s.publishWokenWaiters(changed)
}

func (s *Server) publishWokenWaiters(changed []state.Task) {
	for _, task := range changed {
		s.publishTaskUpdate(task)
		if task.State == state.TaskReady {
			go s.dispatchReadyTasks(context.Background())
		}
	}
}

func (s *Server) evaluateSourceResult(sourceKind, sourceID string) {
	if changed, err := s.stateStore.EvaluateSource(sourceKind, sourceID); err != nil {
		s.log.Debug("evaluate work result failed", "source_kind", sourceKind, "source_id", sourceID, "err", err)
	} else {
		for _, task := range changed {
			s.publishTaskUpdate(task)
			s.propagateTaskFailure(task)
		}
	}
}

func (s *Server) propagateTaskFailure(task state.Task) {
	if task.State != state.TaskDependencyFailed {
		return
	}
	changed, err := s.stateStore.MarkSourceUnsatisfiable(state.SourceTask, task.TaskID)
	if err != nil {
		s.log.Debug("propagate task failure", "task", task.TaskID, "err", err)
		return
	}
	for _, dependent := range changed {
		s.publishTaskUpdate(dependent)
		s.propagateTaskFailure(dependent)
	}
	s.notifyTaskWaiters(task.TaskID)
}

// recoverTasks is the one bounded startup sweep over dependent work. It never
// asks whether a pre-crash runtime survived, because the registry starts empty
// and stale reconciliation deliberately never re-adopts a live process, so that
// question is unanswerable by construction. Every `starting` row is resolved from
// what it durably reserved instead (TS-10.R15, FS-16.R17).
//
// Failing as a whole is fatal to startup for the same reason it is for mail
// activations: a task left `starting` is invisible to the dispatcher, which only
// ever lists ready rows, so its work would never run again. A single task's
// failure is isolated and does not abort the sweep (INV §7).
func (s *Server) recoverTasks(ctx context.Context) error {
	if err := s.stateStore.DiscardDependencyActivations(); err != nil {
		return err
	}
	awaiting, err := s.stateStore.DueTaskCleanup(taskDispatchBatch)
	if err != nil {
		return err
	}
	for _, task := range awaiting {
		s.finishTaskCleanup(ctx, task)
	}
	unfinished, err := s.stateStore.TasksInStates(state.TaskStarting, state.TaskRunning, state.TaskWaiting)
	if err != nil {
		return err
	}
	for _, task := range unfinished {
		if task.State == state.TaskWaiting || task.PendingRelease || task.PendingYield {
			continue
		}
		// A room-backed stage task has no runtime of its own: its room holds
		// uncertain work for explicit recovery (TS-10.R38, TS-14.R14).
		if task.TargetKind == state.TargetThinkTank {
			continue
		}
		if task.State == state.TaskRunning {
			quotaHeld, err := s.stateStore.QuotaHoldsTask(task)
			if err != nil {
				return err
			}
			if quotaHeld {
				continue
			}
			// Its agent is an unowned orphan now, which the ordinary reconciliation
			// sweep reaps. Nothing is resumed on a guess and no outcome is invented.
			s.interruptTask(task, "the server restarted while this task was running")
			continue
		}
		if s.recoverStartAttempt(ctx, task) {
			s.rememberStartupStartRetry(task)
		}
	}
	// Arms may have been satisfied by a result committed just before the crash.
	// Re-evaluating is a no-op when nothing changed, because evaluation reads the
	// durable registration rather than an event.
	armed, err := s.stateStore.TasksInStates(state.TaskArmed)
	if err != nil {
		return err
	}
	for _, task := range armed {
		taskID := task.TaskID
		task, err = s.stateStore.ReadTask(taskID)
		if err != nil {
			s.log.Warn("task recovery read arms failed", "task", taskID, "err", err)
			continue
		}
		for _, arm := range task.Arms {
			if arm.Kind != state.ArmWorkResult || arm.State != state.ArmUnsatisfied {
				continue
			}
			s.evaluateSourceResult(arm.SourceKind, arm.SourceID)
		}
	}
	// A watched change committed just before the crash lost its notification.
	// The first batch is rediscovered here; the dispatcher tick continues it.
	s.reconcileTaskWaits()
	return nil
}

// recoverStartAttempt resolves one interrupted start by the runtime claim it
// reserved. A runtime this attempt would have created or woken is reaped and the
// task is started once more within its attempt limit; a runtime it merely
// borrowed is never touched, because it belongs to someone else and this
// feature's promise does not lapse because Chuck restarted. That task becomes
// interrupted, since whether the assignment reached that conversation cannot be
// known and delivering it twice is worse than asking a person (FS-16.R4, R17).
func (s *Server) recoverStartAttempt(ctx context.Context, task state.Task) bool {
	if task.RuntimeClaim == state.ClaimBorrowed {
		s.interruptTask(task, "the server restarted while this task was starting in an existing conversation")
		return false
	}
	if task.AssignedAgentID != "" {
		if err := s.StopStage(ctx, task.AssignedAgentID); err != nil {
			s.log.Warn("reap interrupted task runtime failed", "task", task.TaskID, "err", err)
			// The starting row still owns this runtime. Do not clear that only
			// durable claim after a failed reap; the next recovery pass can try
			// again without admitting a second agent (INV §4/§15).
			return true
		}
	}
	if failed, ok, err := s.stateStore.FailTaskStart(task.TaskID, task.StartAttemptID,
		"the server restarted during this start attempt"); err != nil {
		s.log.Warn("recover start attempt failed", "task", task.TaskID, "err", err)
		return true
	} else if ok {
		s.publishTaskUpdate(failed)
		s.propagateTaskFailure(failed)
	}
	return false
}

func (s *Server) rememberStartupStartRetry(task state.Task) {
	s.taskStartMu.Lock()
	defer s.taskStartMu.Unlock()
	if _, exists := s.startupStartRetries[task.TaskID]; !exists {
		s.startupStartRetries[task.TaskID] = task.StartAttemptID
		s.startupStartRetryOrder = append(s.startupStartRetryOrder, task.TaskID)
	}
}

func (s *Server) rotateStartupStartRetryLocked(taskID string) {
	s.removeStartupStartRetryLocked(taskID)
	s.startupStartRetryOrder = append(s.startupStartRetryOrder, taskID)
}

func (s *Server) removeStartupStartRetryLocked(taskID string) {
	for i, id := range s.startupStartRetryOrder {
		if id == taskID {
			s.startupStartRetryOrder = append(s.startupStartRetryOrder[:i], s.startupStartRetryOrder[i+1:]...)
			return
		}
	}
}

// finishInterruptedRelease completes a stop and release a reporting turn never
// finished, so a recorded result can never leave a task-owned runtime up with
// nothing owning it (TS-10.R19, INV §15).
func (s *Server) finishInterruptedRelease(ctx context.Context, task state.Task) {
	s.finishTaskCleanup(ctx, task)
}

func (s *Server) interruptTask(task state.Task, reason string) {
	if interrupted, ok, err := s.stateStore.InterruptTask(task.TaskID, reason); err != nil {
		s.log.Warn("interrupt task failed", "task", task.TaskID, "err", err)
	} else if ok {
		s.publishTaskUpdate(interrupted)
		s.notifyTaskWaiters(interrupted.TaskID)
		if s.pipelineMgr != nil {
			if err := s.pipelineMgr.OnStageTaskInterrupted(interrupted.TaskID, "restart_interrupted"); err != nil {
				s.log.Warn("pause pipeline after restart interrupted stage task", "task", interrupted.TaskID, "err", err)
			}
		}
	}
}

// interruptTaskOnExit is the runtime-exit half of FS-16.R16: an assignee that
// goes away before recording a result leaves its task interrupted, needing
// attention, with its claim and slot released — never converted into a result.
// A task that already recorded one is untouched, which is what makes the stop
// that follows a report safe.
func (s *Server) interruptTaskOnExit(agentID, generation, cause string) {
	if task, err := s.stateStore.AssignedTask(agentID); err == nil && task.AssignedGeneration == generation {
		held, err := s.stateStore.QuotaHoldsTask(task)
		if err != nil {
			s.log.Error("read quota-held assignment on exit", "agent", agentID, "err", err)
			return
		}
		if held {
			return
		}
	}
	if task, ok, err := s.stateStore.InterruptTaskForAgent(agentID, generation,
		"the assigned agent went away before recording a result: "+cause); err != nil {
		s.log.Warn("interrupt task on agent exit", "agent_id", agentID, "err", err)
	} else if ok {
		s.publishTaskUpdate(task)
		s.notifyTaskWaiters(task.TaskID)
		if s.pipelineMgr != nil {
			if err := s.pipelineMgr.OnStageTaskInterrupted(task.TaskID, "interrupted"); err != nil {
				s.log.Warn("pause pipeline after stage task exit", "task", task.TaskID, "err", err)
			}
		}
		s.log.Info("task interrupted by agent exit", "agent_id", agentID, "cause", cause)
	}
}
