package runtime

import (
	"bufio"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// spawnGroup starts a dedicated process group leader and reaps it when it
// exits, so a dead leader does not linger as a zombie.
func spawnGroup(t *testing.T, script string) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command("/bin/sh", "-c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn: %v", err)
	}
	pid := cmd.Process.Pid
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	t.Cleanup(func() { _ = syscall.Kill(-pid, syscall.SIGKILL); <-done })
	return cmd, bufio.NewReader(out)
}

func requireStartTime(t *testing.T, pid int) time.Time {
	t.Helper()
	birth := ProcessBirth(pid)
	if birth.IsZero() {
		t.Skip("process birth is unavailable on this platform")
	}
	return birth
}

func TestClassifyProcessRequiresExactBirth(t *testing.T) {
	cmd, _ := spawnGroup(t, "exec sleep 30")
	pid := cmd.Process.Pid
	birth := requireStartTime(t, pid)

	if got := ClassifyProcess(pid, birth); got != ProcessOwned {
		t.Fatalf("exact birth = %v, want owned", got)
	}
	// A different birth means the PID was reused by a later process.
	if got := ClassifyProcess(pid, birth.Add(-time.Hour)); got != ProcessExited {
		t.Fatalf("reused PID = %v, want exited", got)
	}
	// A legacy row has no proof: alive but never authorizes destruction.
	if got := ClassifyProcess(pid, time.Time{}); got != ProcessUnverified {
		t.Fatalf("unknown proof = %v, want unverified", got)
	}
	if got := ClassifyProcess(2147483600, birth); got != ProcessExited {
		t.Fatalf("dead PID = %v, want exited", got)
	}
}

// ASRC-3: a recorded PID now naming an unrelated process must never be killed.
func TestStopRecordedProcessSparesReusedPID(t *testing.T) {
	cmd, _ := spawnGroup(t, "exec sleep 30")
	pid := cmd.Process.Pid
	birth := requireStartTime(t, pid)

	if err := StopRecordedProcess(pid, birth.Add(-time.Hour), 100*time.Millisecond); err != nil {
		t.Fatalf("StopRecordedProcess: %v", err)
	}
	if !PidAlive(pid) {
		t.Fatal("process reusing the recorded PID was signaled")
	}
}

// FS-01.R54: unknown legacy proof refuses with an actionable error and leaves
// the live process alone.
func TestStopRecordedProcessRefusesUnknownProof(t *testing.T) {
	cmd, _ := spawnGroup(t, "exec sleep 30")
	pid := cmd.Process.Pid

	err := StopRecordedProcess(pid, time.Time{}, 100*time.Millisecond)
	if !errors.Is(err, ErrProcessUnverified) || !strings.Contains(err.Error(), strconv.Itoa(pid)) {
		t.Fatalf("err = %v, want unverified refusal naming pid %d", err, pid)
	}
	if !PidAlive(pid) {
		t.Fatal("unverified process was signaled")
	}
}

// ASRC-3/FS-01.R21: a descendant that survives its leader's SIGTERM is still
// in the owned group and must be gone when Stop reports success.
func TestStopRecordedProcessSweepsGroupDescendants(t *testing.T) {
	cmd, out := spawnGroup(t, `(trap "" TERM; exec sleep 30) & echo $!; wait`)
	pid := cmd.Process.Pid
	birth := requireStartTime(t, pid)
	line, err := out.ReadString('\n')
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	child, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("child pid %q: %v", line, err)
	}

	if err := StopRecordedProcess(pid, birth, 2*time.Second); err != nil {
		t.Fatalf("StopRecordedProcess: %v", err)
	}
	if orphanAlive(t, child) {
		t.Fatal("TERM-ignoring descendant survived Stop")
	}
}

func TestReconcileStaleKeepsLiveOrphansAndPrunesReusedPID(t *testing.T) {
	st, err := state.Open(t.TempDir())
	if err != nil {
		t.Fatalf("state.Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	cmd, _ := spawnGroup(t, "exec sleep 30")
	pid := cmd.Process.Pid
	birth := requireStartTime(t, pid)

	rows := map[string]time.Time{
		"a_owned":  birth,
		"a_legacy": {},
		"a_reused": birth.Add(-time.Hour),
	}
	for id, b := range rows {
		if err := st.WriteAgent(state.Agent{AgentID: id, Name: id, Role: "dev", Project: "p",
			Backend: "claude", Model: "m", Interface: "chat", CreatedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
		if err := st.WriteRunning(state.RunningEntry{AgentID: id, PID: pid, ProcessBirth: b,
			Interface: "chat", StartedAt: time.Now().UTC()}); err != nil {
			t.Fatal(err)
		}
	}

	pruned, err := ReconcileStale(st)
	if err != nil || len(pruned) != 1 || pruned[0] != "a_reused" {
		t.Fatalf("ReconcileStale = %v, %v; want only a_reused", pruned, err)
	}
	for _, id := range []string{"a_owned", "a_legacy"} {
		if _, err := st.ReadRunning(id); err != nil {
			t.Fatalf("%s live orphan row removed: %v", id, err)
		}
	}
	if !PidAlive(pid) {
		t.Fatal("reconcile signaled a live process")
	}
}
