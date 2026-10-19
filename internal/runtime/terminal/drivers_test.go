package terminal

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	rt "github.com/AsaphNoam/Chuck/internal/runtime"
	"github.com/AsaphNoam/Chuck/internal/state"
)

// driverFixture is a terminal runtime whose external drivers are the fake
// tmux/osascript binaries and whose bootstrap is the test binary.
type driverFixture struct {
	r     *Runtime
	store *state.Store
	dir   string
	mu    sync.Mutex
	exits []string
}

func newDriverFixture(t *testing.T) *driverFixture {
	t.Helper()
	useTestBootstrap(t)
	f := &driverFixture{dir: installFakeDrivers(t)}
	store, err := state.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	f.store = store
	f.r = New(store)
	f.r.SetInitialIdleDelay(time.Millisecond)
	f.r.SetOnExit(func(id, gen string) {
		f.mu.Lock()
		f.exits = append(f.exits, id+"/"+gen)
		f.mu.Unlock()
	})
	old := exitPollInterval
	exitPollInterval = 20 * time.Millisecond
	t.Cleanup(func() { exitPollInterval = old; f.r.StopAll(context.Background()) })
	return f
}

// launch starts one agent named like every other (same title) whose CLI
// writes its own CHUCK_AGENT_ID to out and then stays alive.
func (f *driverFixture) launch(t *testing.T, id, driver, out string) state.RunningEntry {
	t.Helper()
	agent := state.Agent{AgentID: id, Name: "Same", Role: "dev", Project: "p", Backend: "claude",
		Model: "m", Interface: "terminal", CreatedAt: time.Now().UTC()}
	if err := f.store.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	f.r.SetCommand("/bin/sh", "-c", `printf %s "$CHUCK_AGENT_ID" > "$OUT_FILE"; exec sleep 30`)
	spec := rt.LaunchSpec{Agent: agent, Generation: "gen-" + id, Driver: driver, Cwd: t.TempDir(),
		Env: []string{"CHUCK_AGENT_ID=" + id, "OUT_FILE=" + out, "PATH=/usr/bin:/bin"}}
	if _, err := f.r.Start(context.Background(), spec); err != nil {
		t.Fatalf("%s launch %s: %v", driver, id, err)
	}
	row, err := f.store.ReadRunning(id)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func waitFile(t *testing.T, path string) string {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
			return string(data)
		}
	}
	t.Fatalf("%s was never written", path)
	return ""
}

// FS-07.A11 (ASRB-6/7): two same-title tmux agents get distinct sessions and
// each CLI sees exactly its own composed environment, even though the tmux
// server's environment names a different agent.
func TestTmuxDriverOwnsSameTitleAgentsWithExactEnv(t *testing.T) {
	f := newDriverFixture(t)
	serverEnv := "CHUCK_AGENT_ID=stale-server-agent\nPATH=/usr/bin:/bin\n"
	if err := os.WriteFile(filepath.Join(f.dir, "server.env"), []byte(serverEnv), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	one := f.launch(t, "a_one", "tmux", filepath.Join(dir, "one"))
	two := f.launch(t, "a_two", "tmux", filepath.Join(dir, "two"))

	if one.DriverIDs["tmux_session"] == two.DriverIDs["tmux_session"] {
		t.Fatalf("same-title agents share tmux session %q", one.DriverIDs["tmux_session"])
	}
	for _, row := range []state.RunningEntry{one, two} {
		if row.PID <= 0 || row.ProcessBirth.IsZero() || rt.ClassifyProcess(row.PID, row.ProcessBirth) != rt.ProcessOwned {
			t.Fatalf("%s has no proved process: %+v", row.AgentID, row)
		}
		if strings.Contains(row.DriverIDs["tmux_session"], "gen-") {
			t.Fatalf("session name exposes the generation: %q", row.DriverIDs["tmux_session"])
		}
	}
	if got := waitFile(t, filepath.Join(dir, "one")); got != "a_one" {
		t.Fatalf("first CLI saw CHUCK_AGENT_ID=%q", got)
	}
	if got := waitFile(t, filepath.Join(dir, "two")); got != "a_two" {
		t.Fatalf("second CLI saw CHUCK_AGENT_ID=%q", got)
	}
}

// ASRB-4/ASRB-5: an external view's exit is observed and releases ownership
// through the generation-scoped exit path, for tmux and iTerm2 alike. iTerm2
// launches carry a proved process identity instead of PID 0.
func TestExternalDriverExitReleasesOwnership(t *testing.T) {
	for _, driver := range []string{"tmux", "iterm2"} {
		t.Run(driver, func(t *testing.T) {
			f := newDriverFixture(t)
			row := f.launch(t, "a_exit", driver, filepath.Join(t.TempDir(), "out"))
			if row.PID <= 0 || row.ProcessBirth.IsZero() {
				t.Fatalf("%s launch has no process identity: %+v", driver, row)
			}
			if err := syscall.Kill(-row.PID, syscall.SIGKILL); err != nil {
				t.Fatal(err)
			}
			for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(20 * time.Millisecond) {
				f.mu.Lock()
				exits := append([]string(nil), f.exits...)
				f.mu.Unlock()
				if len(exits) == 1 && exits[0] == "a_exit/gen-a_exit" {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("exit never released ownership: %v", exits)
				}
			}
			if _, err := f.store.ReadRunning("a_exit"); err == nil {
				t.Fatal("running row survived the exit")
			}
			if st, _ := f.store.ReadStatus("a_exit"); st.State != "done" {
				t.Fatalf("status = %q, want done", st.State)
			}
		})
	}
}

// Stop of an external view terminates the proved group; Cancel reaches it.
func TestExternalDriverStopTerminatesProvedGroup(t *testing.T) {
	f := newDriverFixture(t)
	row := f.launch(t, "a_stop", "iterm2", filepath.Join(t.TempDir(), "out"))
	if ok, err := f.r.Cancel(context.Background(), "a_stop"); err != nil || !ok {
		t.Fatalf("Cancel = %v, %v; want delivered to the proved group", ok, err)
	}
	if err := f.r.Stop(context.Background(), "a_stop"); err != nil {
		t.Fatal(err)
	}
	if orphanAlive(row.PID) {
		t.Fatal("Stop left the external view's process running")
	}
}

// FS-07.R25: a view whose bootstrap never reports refuses the launch,
// closes only that view and records nothing.
func TestExternalDriverHandshakeFailureRefusesLaunch(t *testing.T) {
	f := newDriverFixture(t)
	// A view that stays up but never reports a handshake.
	silent := filepath.Join(t.TempDir(), "silent-bootstrap")
	if err := os.WriteFile(silent, []byte("#!/bin/sh\nexec sleep 30\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	bootstrapExecutable = func() (string, error) { return silent, nil }
	old := bootstrapHandshakeWait
	bootstrapHandshakeWait = 300 * time.Millisecond
	t.Cleanup(func() { bootstrapHandshakeWait = old })
	agent := state.Agent{AgentID: "a_fail", Name: "Same", Role: "dev", Project: "p", Backend: "claude",
		Model: "m", Interface: "terminal", CreatedAt: time.Now().UTC()}
	if err := f.store.WriteAgent(agent); err != nil {
		t.Fatal(err)
	}
	f.r.SetCommand("/bin/sh", "-c", "sleep 30")
	_, err := f.r.Start(context.Background(), rt.LaunchSpec{Agent: agent, Generation: "g", Driver: "tmux", Cwd: t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "did not report") {
		t.Fatalf("launch without a bootstrap handshake = %v, want handshake timeout", err)
	}
	if _, err := f.store.ReadRunning("a_fail"); err == nil {
		t.Fatal("refused launch wrote a running row")
	}
	if views, _ := filepath.Glob(filepath.Join(f.dir, "*.json")); len(views) != 0 {
		t.Fatalf("refused launch left its view open: %v", views)
	}
}

func orphanAlive(pid int) bool {
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		if syscall.Kill(pid, 0) != nil {
			return false
		}
	}
	return syscall.Kill(pid, 0) == nil
}
