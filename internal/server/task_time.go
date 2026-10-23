package server

import (
	"time"

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
