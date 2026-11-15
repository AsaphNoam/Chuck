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
// unverifiable or a group that outlived its leader, is retained and never
// adopted (FS-01.R54). Every row is
// visited; the first failure is returned with the agents already reconciled.
func ReconcileStale(s *state.Store) ([]string, error) {
	running, err := s.ListRunning()
	if err != nil {
		return nil, err
	}
	var pruned []string
	var errs []error
	for _, r := range running {
		if ClassifyProcess(r.PID, r.ProcessBirth) != ProcessExited || LeaderlessGroup(r.PID) {
			continue
		}
		slog.Info("runtime: reconciling exited running row", "agent", r.AgentID, "pid", r.PID)
		if err := SettleStoppedRow(s, r, "process exited"); err != nil {
			errs = append(errs, err)
			continue
		}
		pruned = append(pruned, r.AgentID)
	}
	return pruned, errors.Join(errs...)
}

// SettleStoppedRow ends a running row whose process is gone and that no live
// runtime owns (FS-01.R6/R20/R21): it writes `done` and ends the recorded
// generation's open turn in one transaction, then deletes the row. The status is
// written first, so a failed write keeps the row for a retry to settle.
func SettleStoppedRow(s *state.Store, row state.RunningEntry, detail string) error {
	st, err := s.ReadStatus(row.AgentID)
	if err != nil {
		st = state.Status{AgentID: row.AgentID}
	}
	st.State = state.ComposeChatStatus(state.ChatProcessStopped, state.ConversationTurn{}, false)
	st.Detail = detail
	st.LastTrace = "Stop"
	st.BusySince = nil
	end := state.TurnEnd{Generation: row.Generation, Outcome: TurnProcessEnded.storageValue()}
	if err := s.WriteTurnEndStatus(st, end); err != nil {
		return err
	}
	return s.DeleteRunningRetainingSession(row.AgentID)
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
