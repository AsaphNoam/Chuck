package terminal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	rt "github.com/AsaphNoam/Chuck/internal/runtime"
)

// stubUnknownBirth makes a live process's birth unreadable while unknown is
// set, like a transient proof-read failure. It must run before the fixture so
// its restore follows the fixture's StopAll.
func stubUnknownBirth(t *testing.T) *atomic.Bool {
	t.Helper()
	var unknown atomic.Bool
	classifyProcess = func(pid int, birth time.Time) rt.ProcessState {
		if unknown.Load() && rt.PidAlive(pid) {
			return rt.ProcessUnverified
		}
		return rt.ClassifyProcess(pid, birth)
	}
	t.Cleanup(func() { classifyProcess = rt.ClassifyProcess })
	return &unknown
}

func (f *driverFixture) exitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.exits)
}

// ASW3-1: Stop of a view whose group cannot be proved refuses without closing
// the view, deleting the row or releasing ownership; a later proved Stop
// succeeds.
func TestStopRefusesUnprovedGroupAndRetainsOwnership(t *testing.T) {
	for _, driver := range []string{"tmux", "iterm2"} {
		t.Run(driver, func(t *testing.T) {
			unknown := stubUnknownBirth(t)
			f := newDriverFixture(t)
			row := f.launch(t, "a_unproved", driver, filepath.Join(t.TempDir(), "out"))
			views, _ := filepath.Glob(filepath.Join(f.dir, "*.json"))

			unknown.Store(true)
			err := f.r.Stop(context.Background(), "a_unproved")
			if !errors.Is(err, rt.ErrProcessUnverified) {
				t.Fatalf("Stop with unknown proof = %v, want refusal", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !rt.PidAlive(row.PID) {
				t.Fatal("refused Stop ended the process")
			}
			for _, v := range views {
				if _, err := os.Stat(v); err != nil {
					t.Fatalf("refused Stop closed the view: %v", err)
				}
			}
			if _, err := f.store.ReadRunning("a_unproved"); err != nil {
				t.Fatalf("refused Stop deleted the running row: %v", err)
			}
			if _, err := f.r.lookup("a_unproved"); err != nil {
				t.Fatalf("refused Stop released ownership: %v", err)
			}

			unknown.Store(false)
			if err := f.r.Stop(context.Background(), "a_unproved"); err != nil {
				t.Fatalf("retry Stop: %v", err)
			}
			if orphanAlive(row.PID) {
				t.Fatal("retry Stop left the process running")
			}
			if _, err := f.store.ReadRunning("a_unproved"); err == nil {
				t.Fatal("retry Stop kept the running row")
			}
			if n := f.exitCount(); n != 0 {
				t.Fatalf("requested Stop reported %d unsolicited exits", n)
			}
		})
	}
}

// ASW3-2: an unreadable birth is not an exit. The watcher keeps the view
// owned through unknown and recovered proof and reports only a confirmed exit.
func TestUnverifiedViewIsNotPublishedAsExited(t *testing.T) {
	unknown := stubUnknownBirth(t)
	f := newDriverFixture(t)
	row := f.launch(t, "a_unknown", "tmux", filepath.Join(t.TempDir(), "out"))

	unknown.Store(true)
	time.Sleep(10 * exitPollInterval)
	if n := f.exitCount(); n != 0 {
		t.Fatalf("unknown proof published %d exits", n)
	}
	if _, err := f.store.ReadRunning("a_unknown"); err != nil {
		t.Fatalf("unknown proof deleted the running row: %v", err)
	}
	unknown.Store(false)
	time.Sleep(5 * exitPollInterval)
	if n := f.exitCount(); n != 0 {
		t.Fatalf("recovered proof published %d exits", n)
	}

	if err := syscall.Kill(-row.PID, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(5 * time.Second); f.exitCount() != 1; time.Sleep(20 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("confirmed exit was never published")
		}
	}
}

// ASW3-5: an xterm handle is re-proved before Cancel signals it: a reaped
// child or a stale birth must not reach whatever group now has that ID.
func TestCancelReprovesXtermGroupBeforeSignal(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	waited := make(chan struct{})
	go func() { _ = cmd.Wait(); close(waited) }()
	t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL); <-waited })
	birth := rt.ProcessBirth(pid)
	if birth.IsZero() {
		t.Skip("process birth is unavailable on this platform")
	}
	reaped := make(chan struct{})
	close(reaped)

	cases := []struct {
		name   string
		birth  time.Time
		exited chan struct{}
		want   bool
	}{
		{"reaped", birth, reaped, false},
		{"stale birth", birth.Add(-250 * time.Millisecond), make(chan struct{}), false},
	}
	for _, tc := range cases {
		r := New(nil)
		r.agents["a_x"] = &termAgent{agentID: "a_x", tab: &Tab{Driver: "xterm", PGID: pid, Birth: tc.birth, exited: tc.exited}}
		if ok, err := r.Cancel(context.Background(), "a_x"); err != nil || ok != tc.want {
			t.Fatalf("%s: Cancel = %v, %v; want %v", tc.name, ok, err, tc.want)
		}
	}
	select {
	case <-waited:
		t.Fatal("Cancel signaled an unproved group")
	case <-time.After(200 * time.Millisecond):
	}
}
