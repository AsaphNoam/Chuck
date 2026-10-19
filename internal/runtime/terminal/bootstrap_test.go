package terminal

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"

	rt "github.com/AsaphNoam/Chuck/internal/runtime"
)

// TestMain lets the test binary stand in for the Chuck executable's hidden
// bootstrap command, and for the fake tmux/osascript drivers.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == BootstrapCommand {
		err := RunBootstrap(os.Args[2])
		fmt.Fprintln(os.Stderr, "bootstrap:", err)
		os.Exit(1)
	}
	if runFakeDriver() {
		return
	}
	os.Exit(m.Run())
}

func useTestBootstrap(t *testing.T) {
	t.Helper()
	old := bootstrapExecutable
	bootstrapExecutable = func() (string, error) { return os.Args[0], nil }
	t.Cleanup(func() { bootstrapExecutable = old })
}

// startInOwnedView runs shell command text the way a driver's new view does:
// a session leader whose controlling terminal is a fresh PTY.
func startInOwnedView(t *testing.T, command string, ctty bool) (*exec.Cmd, string) {
	t.Helper()
	ptmx, tty, err := pty.Open()
	if err != nil {
		t.Fatalf("pty: %v", err)
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: ctty}
	if !ctty {
		// Not this view's terminal: stdin is a pipe, so no foreground proof.
		cmd.Stdin = nil
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start view: %v", err)
	}
	ttyName := tty.Name()
	_ = tty.Close()
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	go func() {
		buf := make([]byte, 4096)
		for {
			if _, err := ptmx.Read(buf); err != nil {
				return
			}
		}
	}()
	t.Cleanup(func() { _ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); <-done; _ = ptmx.Close() })
	return cmd, ttyName
}

// TS-04.R102/TS-05.R28 (ASRB-6): the bootstrap proves its dedicated
// foreground group, reports a nonce-bound handshake and execs the exact
// composed environment — not the launching process's environment — while
// the record is consumed and the handshake binds the same PID and TTY.
func TestBootstrapExecsExactEnvironmentWithProof(t *testing.T) {
	useTestBootstrap(t)
	t.Setenv("CHUCK_PARENT_ONLY", "leak")
	out := filepath.Join(t.TempDir(), "env.txt")
	spec := TabSpec{
		Command: []string{"/bin/sh", "-c", `printf '%s|%s' "$CHUCK_AGENT_ID" "${CHUCK_PARENT_ONLY-unset}" > ` + out + `; sleep 30`},
		Cwd:     t.TempDir(),
		Env:     []string{"CHUCK_AGENT_ID=a_one", "PATH=/usr/bin:/bin"},
	}
	launch, command, err := prepareOwnedLaunch(spec)
	if err != nil {
		t.Fatal(err)
	}
	defer launch.cleanup()
	if strings.Contains(command, "a_one") {
		t.Fatalf("command text exposes the environment: %q", command)
	}
	view, ttyName := startInOwnedView(t, command, true)

	h, err := launch.await(5 * time.Second)
	if err != nil {
		t.Fatalf("await: %v", err)
	}
	birth, err := verifyOwnedProcess(h, view.Process.Pid, ttyName)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if rt.ClassifyProcess(view.Process.Pid, birth) != rt.ProcessOwned {
		t.Fatal("verified birth does not prove the view's process")
	}
	if _, err := os.Stat(launch.recordPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("launch record was not consumed: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if data, err := os.ReadFile(out); err == nil && len(data) > 0 {
			if got := string(data); got != "a_one|unset" {
				t.Fatalf("child environment = %q, want exact composed env", got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bootstrapped CLI never ran")
		}
		time.Sleep(20 * time.Millisecond)
	}
	// A different TTY or PID is not this launch's view.
	if _, err := verifyOwnedProcess(h, view.Process.Pid+1, ttyName); err == nil {
		t.Fatal("handshake accepted for another driver PID")
	}
	if _, err := verifyOwnedProcess(h, 0, "/dev/null"); err == nil {
		t.Fatal("handshake accepted for another terminal")
	}
}

// Without the foreground terminal proof the bootstrap refuses and never
// execs; the parent sees the refusal through the handshake.
func TestBootstrapRefusesWithoutForegroundTerminal(t *testing.T) {
	useTestBootstrap(t)
	marker := filepath.Join(t.TempDir(), "ran")
	launch, command, err := prepareOwnedLaunch(TabSpec{Command: []string{"/usr/bin/touch", marker}, Cwd: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer launch.cleanup()
	startInOwnedView(t, command, false)
	if _, err := launch.await(5 * time.Second); err == nil || !strings.Contains(err.Error(), "foreground") {
		t.Fatalf("await = %v, want foreground refusal", err)
	}
	time.Sleep(100 * time.Millisecond)
	if _, err := os.Stat(marker); err == nil {
		t.Fatal("refused bootstrap still exec'd the CLI")
	}
}

// TS-05.R28: symlinked, group-readable, oversized and replayed records are
// rejected before any secret is used.
func TestBootstrapRejectsUnsafeRecords(t *testing.T) {
	write := func(t *testing.T, mode os.FileMode, data []byte) string {
		dir, err := os.MkdirTemp("", "chuck-term-test-")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })
		path := filepath.Join(dir, "launch.json")
		if err := os.WriteFile(path, data, mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, mode); err != nil {
			t.Fatal(err)
		}
		return path
	}
	valid, _ := json.Marshal(launchRecord{Version: 1, Nonce: "n", Argv: []string{"/bin/true"}, Cwd: "/"})

	if _, err := consumeLaunchRecord(write(t, 0o644, valid)); err == nil {
		t.Fatal("group-readable record accepted")
	}
	link := write(t, 0o600, valid)
	target := link + ".real"
	if err := os.Rename(link, target); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := consumeLaunchRecord(link); err == nil {
		t.Fatal("symlinked record accepted")
	}
	if _, err := consumeLaunchRecord(write(t, 0o600, make([]byte, bootstrapRecordLimit+1))); err == nil {
		t.Fatal("oversized record accepted")
	}
	once := write(t, 0o600, valid)
	if _, err := consumeLaunchRecord(once); err != nil {
		t.Fatalf("valid record: %v", err)
	}
	if _, err := consumeLaunchRecord(once); err == nil {
		t.Fatal("record replayed after consumption")
	}
	if err := checkRecordBounds(make([]string, bootstrapEntryLimit+1), nil); err == nil {
		t.Fatal("too many entries accepted")
	}
	if err := checkRecordBounds([]string{strings.Repeat("x", bootstrapEntryBytes+1)}, nil); err == nil {
		t.Fatal("oversized entry accepted")
	}
}

// A handshake carrying another launch's nonce is not this launch's proof.
func TestOwnedLaunchRejectsForeignHandshake(t *testing.T) {
	useTestBootstrap(t)
	launch, _, err := prepareOwnedLaunch(TabSpec{Command: []string{"/bin/true"}, Cwd: "/"})
	if err != nil {
		t.Fatal(err)
	}
	defer launch.cleanup()
	if err := writeHandshake(launch.handshakePath(), bootstrapHandshake{Nonce: "other", PID: os.Getpid()}); err != nil {
		t.Fatal(err)
	}
	if _, err := launch.await(time.Second); err == nil {
		t.Fatal("foreign handshake accepted")
	}
	launch.cleanup()
	if _, err := os.Stat(launch.dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("launch directory survived cleanup: %v", err)
	}
}
