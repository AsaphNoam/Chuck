package runtime

import (
	"context"
	"os/exec"
	"syscall"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// Finding 5 regression (chat runtime): Stop on an agent this runtime does not own
// in memory (e.g. after a dashboard restart, where ReconcileStale never re-adopts
// a live PID) must not silently succeed while the CLI keeps running — it must kill
// the recorded process group first. We WriteRunning a row pointing at a live child
// that is NOT in the runtime's agents map, then Stop, and assert the child is dead.
func TestChatStopKillsOrphanedLiveProcess(t *testing.T) {
	store, err := state.Open(t.TempDir())
	if err != nil {
		t.Fatalf("state.Open: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	cmd := exec.Command("sleep", "60")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn sleep: %v", err)
	}
	pgid := cmd.Process.Pid
	t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL); _, _ = cmd.Process.Wait() })
	go func() { _, _ = cmd.Process.Wait() }() // reap so it doesn't linger as a zombie

	const id = "a_chat_orphan"
	if err := store.WriteAgent(state.Agent{
		AgentID: id, Name: "Ghost", Role: "dev", Project: "demo",
		Backend: "claude", Model: "sonnet", Interface: "chat",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("WriteAgent: %v", err)
	}
	if err := store.WriteRunning(state.RunningEntry{
		AgentID: id, PID: pgid, ProcessBirth: requireStartTime(t, pgid), SessionID: "sess-x", Interface: "chat",
		StartedAt: time.Now().UTC(), Generation: "g1",
	}); err != nil {
		t.Fatalf("WriteRunning: %v", err)
	}
	// The surviving process was mid-turn when the dashboard restarted.
	if err := store.BeginConversationTurn(id, "t1", state.TurnOrigin{Generation: "g1", Kind: state.TurnOriginPerson}); err != nil {
		t.Fatalf("BeginConversationTurn: %v", err)
	}
	busySince := time.Now().UTC()
	if err := store.WriteStatus(state.Status{AgentID: id, State: "busy", Detail: "Running tool", BusySince: &busySince}); err != nil {
		t.Fatalf("WriteStatus: %v", err)
	}

	c := NewChatRuntime(store) // empty agents map → the !ok branch
	if err := c.Stop(context.Background(), id); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if _, err := store.ReadRunning(id); err == nil {
		t.Fatal("running row should be gone after orphan Stop")
	}
	if orphanAlive(t, pgid) {
		t.Fatal("orphaned live process survived Stop (silent success bug)")
	}
	assertStoppedSettled(t, store, id, "g1")
}

// assertStoppedSettled checks FS-01.R6's final state after a Stop that reaped a
// recorded process: `done`, no busy timer, and the generation's turn ended.
func assertStoppedSettled(t *testing.T, store *state.Store, id, generation string) {
	t.Helper()
	st, err := store.ReadStatus(id)
	if err != nil {
		t.Fatalf("ReadStatus: %v", err)
	}
	if st.State != "done" || st.BusySince != nil {
		t.Fatalf("status = %q busy_since=%v, want done with no busy timer", st.State, st.BusySince)
	}
	turn, err := store.ReadConversationTurn(id, generation)
	if err != nil {
		t.Fatalf("ReadConversationTurn: %v", err)
	}
	if turn.State != "ended" || turn.Outcome != TurnProcessEnded.storageValue() {
		t.Fatalf("turn = %q/%q, want ended/%s", turn.State, turn.Outcome, TurnProcessEnded.storageValue())
	}
}

func orphanAlive(t *testing.T, pid int) bool {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return syscall.Kill(pid, 0) == nil
}
