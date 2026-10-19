package runtime

import (
	"errors"
	"log/slog"
	"syscall"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// ReconcileStale removes running rows whose recorded process has exited and
// writes FS-01.R20's `done` / "process exited" outcome. Server start and the
// periodic sweep share it. A live orphan, including one whose identity is
// unverifiable, is retained and never adopted (FS-01.R54). Every row is
// visited; the first failure is returned with the agents already reconciled.
func ReconcileStale(s *state.Store) ([]string, error) {
	running, err := s.ListRunning()
	if err != nil {
		return nil, err
	}
	var pruned []string
	var errs []error
	for _, r := range running {
		if ClassifyProcess(r.PID, r.ProcessBirth) != ProcessExited {
			continue
		}
		slog.Info("runtime: reconciling exited running row", "agent", r.AgentID, "pid", r.PID)
		if err := s.DeleteRunningRetainingSession(r.AgentID); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := MarkProcessExited(s, r.AgentID); err != nil {
			errs = append(errs, err)
		}
		pruned = append(pruned, r.AgentID)
	}
	return pruned, errors.Join(errs...)
}

// MarkProcessExited writes FS-01.R20's outcome for an agent whose process is
// gone without a live owner.
func MarkProcessExited(s *state.Store, agentID string) error {
	st, err := s.ReadStatus(agentID)
	if err != nil {
		st = state.Status{AgentID: agentID}
	}
	st.State = "done"
	st.Detail = "process exited"
	st.LastTrace = "Stop"
	st.BusySince = nil
	return s.WriteStatus(st)
}

// pidAlive reports whether a process group / pid is still alive. signal 0 does no
// signalling but performs the existence + permission check.
func pidAlive(pid int) bool {
	return PidAlive(pid)
}

// PidAlive reports whether a pid is still alive (signal 0: existence + permission
// check, no actual signal delivered). It alone never authorizes a signal;
// destructive callers use ClassifyProcess (INV §9).
func PidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
