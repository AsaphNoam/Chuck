package server

import (
	"errors"
	"time"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// startAtLocalLayout is the offset-free wall-clock form a person types; it is
// read in the server process's zone (TS-10.R56).
const startAtLocalLayout = "2006-01-02T15:04"

// parseStartAt resolves a one-time start to an absolute instant once, at the
// boundary: RFC 3339 keeps its own offset and the local form uses time.Local.
func parseStartAt(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC(), true
	}
	if t, err := time.ParseInLocation(startAtLocalLayout, raw, time.Local); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

// startTimeArm turns the optional start_at/run_only_on_time pair shared by HTTP
// create, re-arm, and the agent tool into the task's time arm (FS-16.R51–R52).
// Run only on time defaults to on. Whether the instant is still in the future
// is decided by the write transaction, against the same clock that resolves it.
func startTimeArm(startAt string, runOnlyOnTime *bool) (*state.TaskArm, *runtime.APIError) {
	if startAt == "" {
		return nil, nil
	}
	due, ok := parseStartAt(startAt)
	if !ok {
		return nil, apiError(runtime.CodeValidation, "start_at must be RFC 3339 or YYYY-MM-DDTHH:MM")
	}
	onTimeOnly := runOnlyOnTime == nil || *runOnlyOnTime
	return &state.TaskArm{Kind: state.ArmTime, DueAt: &due, OnTimeOnly: onTimeOnly}, nil
}

// resolveDueTimeArms is the tick's due-work step for start times (TS-10.R54). A
// task parked as a missed start propagates like any other parked prerequisite.
func (s *Server) resolveDueTimeArms() {
	changed, err := s.stateStore.ResolveDueTimeArms(taskDispatchBatch)
	if err != nil {
		s.log.Error("resolve due start times", "err", err)
		return
	}
	for _, task := range changed {
		s.publishTaskUpdate(task)
		if task.State == state.TaskDependencyFailed {
			s.propagateTaskFailure(task)
		}
	}
}

// scheduleUnavailable is the due transaction's target check (TS-10.R59): the
// same eligibility task creation enforces, read when the occurrence comes due
// rather than fanned out when an agent or project is archived.
func (s *Server) scheduleUnavailable(sched state.TaskSchedule) string {
	// The archive gate passes a missing project; task creation refuses it.
	if _, err := s.configStore.ReadProject(sched.Project); errors.Is(err, config.ErrNotFound) {
		return "the project was deleted"
	}
	if ae := s.projectArchiveGate(sched.Project, ""); ae != nil && ae.Code == runtime.CodeProjectArchived {
		return "the project is archived"
	}
	if sched.TargetKind != state.TargetAgent {
		return ""
	}
	agent, err := s.stateStore.ReadAgent(sched.TargetAgentID)
	switch {
	case errors.Is(err, state.ErrNotFound):
		return "the target agent was deleted"
	case err != nil:
		// Unknown is not unavailable: the occurrence task parks on its own if
		// the agent really cannot run it.
		s.log.Error("read schedule target", "schedule", sched.ScheduleID, "err", err)
		return ""
	case agent.Archived:
		return "the target agent is archived"
	}
	return ""
}

// processDueSchedules is the tick's due-work step for schedules (TS-10.R54,
// R57). Occurrences are computed in the process's zone (TS-10.R56).
func (s *Server) processDueSchedules() {
	outcomes, err := s.stateStore.ProcessDueSchedules(taskDispatchBatch, time.Local, s.scheduleUnavailable)
	if err != nil {
		s.log.Error("process due schedules", "err", err)
	}
	for _, outcome := range outcomes {
		if outcome.Task != nil {
			s.publishTaskUpdate(*outcome.Task)
		}
		s.publishScheduleUpdate(outcome.Schedule)
	}
}

// publishScheduleUpdate follows each committed schedule change with the
// bounded fields a client needs to decide whether to refetch (TS-10.R60).
func (s *Server) publishScheduleUpdate(sched state.TaskSchedule) {
	s.eventBus.Publish("schedule_update", nil, map[string]any{
		"schedule_id":  sched.ScheduleID,
		"project":      sched.Project,
		"revision":     sched.Revision,
		"state":        sched.State,
		"pause_reason": sched.PauseReason,
	})
}
