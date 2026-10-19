package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// Fake external drivers for subprocess tests (FS-07.A11). The test binary is
// symlinked as `tmux` / `osascript` on PATH. A fake "view" is a detached
// process that owns a PTY and runs the driver's shell command text as a
// session leader with that PTY as its controlling terminal, inheriting the
// fake server's environment like a real tmux server would. These fixtures do
// not prove real tmux/iTerm2 behavior (FS-07.A12).

const fakeViewArg = "__fake-view"

type fakeViewState struct {
	PID int    `json:"pid"`
	TTY string `json:"tty"`
}

// runFakeDriver dispatches the test binary when it is invoked as a fake
// driver binary or view; it reports false for an ordinary test run.
func runFakeDriver() bool {
	if len(os.Args) == 5 && os.Args[1] == fakeViewArg {
		os.Exit(fakeView(os.Args[2], os.Args[3], os.Args[4]))
	}
	switch filepath.Base(os.Args[0]) {
	case "tmux":
		os.Exit(fakeTmux(os.Args[1:]))
	case "osascript":
		os.Exit(fakeOsascript())
	}
	return false
}

// installFakeDrivers puts fake tmux/osascript first on PATH and returns the
// directory holding view state files.
func installFakeDrivers(t *testing.T) string {
	t.Helper()
	bin, dir := t.TempDir(), t.TempDir()
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tmux", "osascript"} {
		if err := os.Symlink(self, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_DRIVER_BIN", self)
	t.Setenv("FAKE_DRIVER_DIR", dir)
	t.Cleanup(func() {
		paths, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		for _, p := range paths {
			if st, err := readFakeView(p); err == nil {
				_ = syscall.Kill(-st.PID, syscall.SIGKILL)
			}
		}
	})
	return dir
}

func fakeView(stateFile, cwd, command string) int {
	ptmx, tty, err := pty.Open()
	if err != nil {
		return 1
	}
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Dir = cwd
	cmd.Stdin, cmd.Stdout, cmd.Stderr = tty, tty, tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		return 1
	}
	name := tty.Name()
	_ = tty.Close()
	data, _ := json.Marshal(fakeViewState{PID: cmd.Process.Pid, TTY: name})
	_ = os.WriteFile(stateFile+".tmp", data, 0o600)
	_ = os.Rename(stateFile+".tmp", stateFile)
	go func() { _, _ = io.Copy(io.Discard, ptmx) }()
	_ = cmd.Wait()
	_ = os.Remove(stateFile)
	return 0
}

// spawnFakeView starts a view with the fake server's own environment, like an
// already-running tmux server: the launching client's environment does not
// reach the pane (ASRB-6).
func spawnFakeView(stateFile, cwd, command string) error {
	cmd := exec.Command(os.Getenv("FAKE_DRIVER_BIN"), fakeViewArg, stateFile, cwd, command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if data, err := os.ReadFile(filepath.Join(os.Getenv("FAKE_DRIVER_DIR"), "server.env")); err == nil {
		cmd.Env = append(strings.Split(strings.TrimSpace(string(data)), "\n"),
			"FAKE_DRIVER_BIN="+os.Getenv("FAKE_DRIVER_BIN"), "FAKE_DRIVER_DIR="+os.Getenv("FAKE_DRIVER_DIR"))
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if _, err := os.Stat(stateFile); err == nil {
			return nil
		}
	}
	return fmt.Errorf("fake view did not start")
}

func readFakeView(path string) (fakeViewState, error) {
	var st fakeViewState
	data, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(data, &st)
	}
	return st, err
}

func closeFakeView(path string) {
	if st, err := readFakeView(path); err == nil {
		_ = syscall.Kill(-st.PID, syscall.SIGHUP)
		_ = syscall.Kill(-st.PID, syscall.SIGKILL)
	}
	_ = os.Remove(path)
}

func fakeTmux(args []string) int {
	dir := os.Getenv("FAKE_DRIVER_DIR")
	flag := func(name string) string {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == name {
				return args[i+1]
			}
		}
		return ""
	}
	if len(args) == 0 {
		return 1
	}
	switch args[0] {
	case "new-session":
		name := flag("-s")
		state := filepath.Join(dir, name+".json")
		if _, err := os.Stat(state); err == nil {
			fmt.Fprintf(os.Stderr, "duplicate session: %s\n", name)
			return 1
		}
		cwd := flag("-c")
		if cwd == "" {
			cwd = "/"
		}
		if err := spawnFakeView(state, cwd, args[len(args)-1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	case "display-message":
		st, err := readFakeView(filepath.Join(dir, flag("-t")+".json"))
		if err != nil {
			fmt.Fprintln(os.Stderr, "can't find session")
			return 1
		}
		switch args[len(args)-1] {
		case "#{pane_tty}":
			fmt.Println(st.TTY)
		case "#{pane_pid}":
			fmt.Println(st.PID)
		}
	case "kill-session":
		closeFakeView(filepath.Join(dir, flag("-t")+".json"))
	}
	return 0
}

// fakeOsascript understands the driver's create-tab and close-window scripts;
// other scripts (appearance, write text) succeed without effect.
func fakeOsascript() int {
	dir := os.Getenv("FAKE_DRIVER_DIR")
	script, _ := io.ReadAll(os.Stdin)
	text := string(script)
	if strings.Contains(text, "create window") {
		var command string
		for _, line := range strings.Split(text, "\n") {
			if line = strings.TrimSpace(line); strings.HasPrefix(line, `write text "`) {
				lit := strings.TrimSuffix(strings.TrimPrefix(line, `write text "`), `"`)
				command = strings.NewReplacer(`\"`, `"`, `\\`, `\`).Replace(lit)
			}
		}
		id := strconv.FormatInt(time.Now().UnixNano(), 10)
		state := filepath.Join(dir, "iterm-"+id+".json")
		if err := spawnFakeView(state, "/", command); err != nil {
			return 1
		}
		st, _ := readFakeView(state)
		fmt.Printf("%s\t%s\t%s\n", st.TTY, id, id)
		return 0
	}
	if i := strings.Index(text, "tell window id "); i >= 0 {
		id := strings.Fields(text[i+len("tell window id "):])[0]
		closeFakeView(filepath.Join(dir, "iterm-"+id+".json"))
	}
	return 0
}
