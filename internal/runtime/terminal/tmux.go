package terminal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// tmuxDriver runs the CLI inside a named, detached tmux session (§2.1) so the
// user can reattach outside the browser. Prompts are delivered with `send-keys`;
// the controlling tty and pane pid are read back with `display-message`. The
// same PTY-bridge UI can attach to the session, or the user attaches manually.
type tmuxDriver struct{}

func (tmuxDriver) Name() string { return "tmux" }

// StartTab creates a detached session whose pane execs the owned bootstrap,
// then accepts it only after the bootstrap's handshake matches the pane's PID
// and TTY (TS-04.R102). The pane receives exactly the record's environment,
// never the tmux server's global or session environment (ASRB-6), and the
// session name is the unique machine handle, not the title (ASRB-7).
func (tmuxDriver) StartTab(spec TabSpec) (*Tab, error) {
	launch, command, err := prepareOwnedLaunch(spec)
	if err != nil {
		return nil, err
	}
	defer launch.cleanup()
	name := spec.tmuxSession()
	args := []string{"new-session", "-d", "-s", name}
	if spec.Cwd != "" {
		args = append(args, "-c", spec.Cwd)
	}
	// The tmux client keeps Chuck's own environment: agent secrets must not
	// seed a new tmux server's global environment.
	args = append(args, command)
	if out, err := exec.Command("tmux", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("terminal: tmux new-session: %w: %s", err, strings.TrimSpace(string(out)))
	}
	tab := &Tab{Driver: "tmux", tmuxName: name, IDs: map[string]string{"tmux_session": name}}
	fail := func(err error) (*Tab, error) {
		_ = exec.Command("tmux", "kill-session", "-t", name).Run()
		return nil, err
	}
	h, err := launch.await(bootstrapHandshakeWait)
	if err != nil {
		return fail(err)
	}
	tty, terr := tmuxDisplay(name, "#{pane_tty}")
	pidStr, perr := tmuxDisplay(name, "#{pane_pid}")
	pid, aerr := strconv.Atoi(strings.TrimSpace(pidStr))
	if terr != nil || perr != nil || aerr != nil || pid <= 0 {
		return fail(fmt.Errorf("terminal: tmux pane identity unavailable"))
	}
	birth, err := verifyOwnedProcess(h, pid, tty)
	if err != nil {
		return fail(err)
	}
	tab.TTY, tab.PGID, tab.Birth = tty, pid, birth
	if spec.Title != "" {
		_ = exec.Command("tmux", "rename-window", "-t", name, spec.Title).Run()
	}
	return tab, nil
}

func (tmuxDriver) WriteText(tab *Tab, text string) error {
	if tab == nil || tab.tmuxName == "" {
		return fmt.Errorf("terminal: no tmux session for tab")
	}
	// Two send-keys: the literal text, then Enter, so the line editor submits it.
	if err := exec.Command("tmux", "send-keys", "-t", tab.tmuxName, "-l", text).Run(); err != nil {
		return fmt.Errorf("terminal: tmux send-keys: %w", err)
	}
	return exec.Command("tmux", "send-keys", "-t", tab.tmuxName, "Enter").Run()
}

// CloseTab kills the tmux session, terminating the CLI inside it.
func (tmuxDriver) CloseTab(tab *Tab) error {
	if tab == nil || tab.tmuxName == "" {
		return nil
	}
	return exec.Command("tmux", "kill-session", "-t", tab.tmuxName).Run()
}

// tmuxSession names the session from the unique agent/generation handle; two
// agents with the same title never collide (ASRB-7). Handles are already
// tmux-safe ([a-z0-9_-]); anything else is replaced defensively.
func (s TabSpec) tmuxSession() string {
	base := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			return r
		default:
			return '-'
		}
	}, s.Handle)
	if base == "" {
		base = "agent"
	}
	return "chuck-" + base
}

func tmuxDisplay(session, fmtStr string) (string, error) {
	out, err := exec.Command("tmux", "display-message", "-p", "-t", session, fmtStr).Output()
	if err != nil {
		return "", fmt.Errorf("terminal: tmux display-message: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// shellJoin renders argv as a single shell-quoted command line for tmux.
func shellJoin(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = shellQuote(a)
	}
	return strings.Join(parts, " ")
}

// shellQuote single-quotes a shell argument, escaping embedded single quotes.
func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n\"'\\$`*?[]{}()|&;<>~#") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
