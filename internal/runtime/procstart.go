package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

var errProcessStartUnknown = errors.New("runtime: process start time unavailable")

// ErrProcessUnverified marks a live PID whose identity as the recorded owned
// process cannot be proved, so no destructive action may target it (FS-01.R54).
var ErrProcessUnverified = errors.New("runtime: process identity is unverifiable")

// ProcessUnverifiedError is the actionable refusal for an unproved live PID.
type ProcessUnverifiedError struct {
	PID int
	// Leaderless: the recorded leader is gone but its group still has members.
	Leaderless bool
}

func (e *ProcessUnverifiedError) Error() string {
	if e.Leaderless {
		return fmt.Sprintf("process group %d lost its leader but still has running members that cannot be verified as this agent's; stop them manually (kill -- -%d), then retry", e.PID, e.PID)
	}
	return fmt.Sprintf("process %d is still running but cannot be verified as this agent's; stop it manually, then retry", e.PID)
}

func (e *ProcessUnverifiedError) Unwrap() error { return ErrProcessUnverified }

// ProcessState classifies a recorded owned process against the live OS.
type ProcessState int

const (
	// ProcessExited: no live process carries the recorded identity.
	ProcessExited ProcessState = iota
	// ProcessOwned: the recorded group leader is alive with its exact birth.
	ProcessOwned
	// ProcessUnverified: a live PID whose identity cannot be proved (unknown
	// legacy proof or unreadable birth). It is retained, never signaled.
	ProcessUnverified
)

// ClassifyProcess compares a recorded leader PID and kernel birth with the
// live process table (TS-02.R54, INV §9). Births are compared exactly at the
// stored microsecond precision, which both readers report without loss
// (Darwin's kernel microseconds; Linux's boot second plus 10 ms ticks). A live
// PID with any other birth was reused after the recorded process died.
func ClassifyProcess(pid int, birth time.Time) ProcessState {
	if !PidAlive(pid) {
		return ProcessExited
	}
	if birth.IsZero() {
		return ProcessUnverified
	}
	started, err := ProcessStartTime(pid)
	if err != nil {
		return ProcessUnverified
	}
	if started.UnixMicro() != birth.UnixMicro() {
		return ProcessExited
	}
	return ProcessOwned
}

// LeaderlessGroup reports that the recorded leader PID is gone while a process
// group with that ID still has members. Their continuity with the recorded
// group cannot be proved once the leader is gone, so callers retain the record
// and refuse cleanup rather than signal a possibly reused group (FS-01.R54).
func LeaderlessGroup(pid int) bool {
	if pid <= 0 || PidAlive(pid) {
		return false
	}
	err := syscall.Kill(-pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// ProcessBirth captures a freshly spawned child's kernel birth while the
// caller still holds it unreaped, so the PID cannot have been reused. Zero
// means the platform cannot report it, which later refuses destruction.
func ProcessBirth(pid int) time.Time {
	if pid <= 0 {
		return time.Time{}
	}
	t, err := ProcessStartTime(pid)
	if err != nil {
		return time.Time{}
	}
	return t
}

// groupDrainBound limits the wait for a leaderless group's members to vanish
// after the final SIGKILL, and an owned command's wait for its inherited
// output pipes after termination (TS-12.R11).
const groupDrainBound = time.Second

// ContainCommand gives an exec.CommandContext child a dedicated process group
// whose whole membership is SIGKILLed when the context ends, and bounds the
// wait for output pipes held open by descendants (TS-01.R41, TS-12.R11). The
// unreaped leader's PID names the group, so the signal cannot reach a reused PID.
func ContainCommand(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.Setpgid = true
	cmd.Cancel = func() error {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
			return err
		}
		return os.ErrProcessDone
	}
	cmd.WaitDelay = groupDrainBound
}

// StopRecordedProcess ends a recorded owned process group whose leader this
// server does not hold (FS-01.R21/R54). Identity is re-proved immediately
// before each signal; there is no atomic identity-and-signal primitive on
// Darwin. It returns nil once the leader is gone and the dedicated group has
// been swept, and a ProcessUnverifiedError without signaling an unproved PID
// or a group whose leader was already gone before this call.
func StopRecordedProcess(pid int, birth time.Time, grace time.Duration) error {
	switch ClassifyProcess(pid, birth) {
	case ProcessExited:
		if LeaderlessGroup(pid) {
			return &ProcessUnverifiedError{PID: pid, Leaderless: true}
		}
		return nil
	case ProcessUnverified:
		return &ProcessUnverifiedError{PID: pid}
	}
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) && ClassifyProcess(pid, birth) == ProcessOwned {
		time.Sleep(50 * time.Millisecond)
	}
	switch ClassifyProcess(pid, birth) {
	case ProcessOwned:
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		for i := 0; i < 20 && ClassifyProcess(pid, birth) == ProcessOwned; i++ {
			time.Sleep(50 * time.Millisecond)
		}
	case ProcessUnverified:
		return &ProcessUnverifiedError{PID: pid}
	}
	if PidAlive(pid) {
		// The PID now names a later process; its group is not ours to sweep.
		return nil
	}
	// The leader is gone. A group ID is not reassigned while any member
	// remains, so this reaches only descendants left in the owned group.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	for end := time.Now().Add(groupDrainBound); time.Now().Before(end) && syscall.Kill(-pid, 0) == nil; {
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}
