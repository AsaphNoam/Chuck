package terminal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	rt "github.com/AsaphNoam/Chuck/internal/runtime"
)

// BootstrapCommand is the hidden first argument of the current Chuck
// executable that tmux and iTerm2 views exec to start an owned agent CLI
// (TS-04.R102). It is internal only: never an agent tool or user command.
const BootstrapCommand = "__terminal-bootstrap"

// Launch-record bounds (TS-05.R28). Stricter platform exec limits remain
// ordinary launch failures.
const (
	bootstrapRecordLimit  = 1 << 20
	bootstrapEntryLimit   = 4096
	bootstrapEntryBytes   = 128 << 10
	bootstrapHandshakeMax = 4 << 10
)

// bootstrapHandshakeWait bounds the wait for a bootstrap's report (TS-01.R41).
// A variable only so tests can exercise the timeout.
var bootstrapHandshakeWait = 10 * time.Second

// bootstrapExecutable resolves the current Chuck executable without a PATH
// lookup, so packaged installs bootstrap through their own binary. Tests
// point it at the test binary.
var bootstrapExecutable = os.Executable

// launchRecord is the owner-only, one-use input of one bootstrap. Only its
// path appears in driver command text; secrets stay in the file.
type launchRecord struct {
	Version int      `json:"version"`
	Nonce   string   `json:"nonce"`
	Argv    []string `json:"argv"`
	Env     []string `json:"env"`
	Cwd     string   `json:"cwd"`
}

// bootstrapHandshake is the bootstrap's nonce-bound report of the process it
// is about to exec into: the same PID, dedicated group and controlling TTY.
type bootstrapHandshake struct {
	Nonce   string `json:"nonce"`
	PID     int    `json:"pid"`
	PGID    int    `json:"pgid"`
	BirthUS int64  `json:"birth_us"`
	TTYDev  uint64 `json:"tty_dev"`
	Error   string `json:"error,omitempty"`
}

// ownedLaunch is the parent side of one external-driver launch. Its private
// directory is removed by cleanup on every exit path (TS-05.R28).
type ownedLaunch struct {
	dir   string
	nonce string
}

// prepareOwnedLaunch writes the launch record for spec into a fresh 0700
// directory and returns the shell command text a driver runs in its new view.
// The text names only the executable and the record path.
func prepareOwnedLaunch(spec TabSpec) (*ownedLaunch, string, error) {
	if len(spec.Command) == 0 {
		return nil, "", fmt.Errorf("terminal: empty launch command")
	}
	if err := checkRecordBounds(spec.Command, spec.Env); err != nil {
		return nil, "", err
	}
	exe, err := bootstrapExecutable()
	if err != nil {
		return nil, "", fmt.Errorf("terminal: resolve Chuck executable: %w", err)
	}
	if exe, err = filepath.Abs(exe); err != nil {
		return nil, "", fmt.Errorf("terminal: resolve Chuck executable: %w", err)
	}
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return nil, "", fmt.Errorf("terminal: launch nonce: %w", err)
	}
	dir, err := os.MkdirTemp("", "chuck-term-")
	if err != nil {
		return nil, "", fmt.Errorf("terminal: launch directory: %w", err)
	}
	l := &ownedLaunch{dir: dir, nonce: hex.EncodeToString(raw[:])}
	data, err := json.Marshal(launchRecord{Version: 1, Nonce: l.nonce, Argv: spec.Command, Env: spec.Env, Cwd: spec.Cwd})
	if err == nil && len(data) > bootstrapRecordLimit {
		err = fmt.Errorf("terminal: launch record exceeds %d bytes", bootstrapRecordLimit)
	}
	if err == nil {
		err = writeExclusive(l.recordPath(), data)
	}
	if err != nil {
		l.cleanup()
		return nil, "", fmt.Errorf("terminal: write launch record: %w", err)
	}
	return l, "exec " + shellQuote(exe) + " " + BootstrapCommand + " " + shellQuote(l.recordPath()), nil
}

func checkRecordBounds(argv, env []string) error {
	if len(argv)+len(env) > bootstrapEntryLimit {
		return fmt.Errorf("terminal: launch has more than %d argv/env entries", bootstrapEntryLimit)
	}
	for _, entry := range append(append([]string{}, argv...), env...) {
		if len(entry) > bootstrapEntryBytes {
			return fmt.Errorf("terminal: a launch argv/env entry exceeds %d bytes", bootstrapEntryBytes)
		}
	}
	return nil
}

func (l *ownedLaunch) recordPath() string    { return filepath.Join(l.dir, "launch.json") }
func (l *ownedLaunch) handshakePath() string { return filepath.Join(l.dir, "handshake.json") }

func (l *ownedLaunch) cleanup() {
	if l != nil {
		_ = os.RemoveAll(l.dir)
	}
}

// await waits for this launch's handshake. A missing, foreign, oversized or
// failed report refuses the launch.
func (l *ownedLaunch) await(timeout time.Duration) (bootstrapHandshake, error) {
	deadline := time.Now().Add(timeout)
	for {
		data, err := readOwnerFile(l.handshakePath(), bootstrapHandshakeMax)
		if err == nil {
			var h bootstrapHandshake
			if err := json.Unmarshal(data, &h); err != nil {
				return h, fmt.Errorf("terminal: unreadable bootstrap handshake: %w", err)
			}
			if h.Nonce != l.nonce {
				return h, fmt.Errorf("terminal: bootstrap handshake does not match this launch")
			}
			if h.Error != "" {
				return h, fmt.Errorf("terminal: bootstrap refused: %s", h.Error)
			}
			return h, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return bootstrapHandshake{}, fmt.Errorf("terminal: bootstrap handshake: %w", err)
		}
		if time.Now().After(deadline) {
			return bootstrapHandshake{}, fmt.Errorf("terminal: bootstrap did not report within %s", timeout)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// verifyOwnedProcess independently corroborates a handshake against the OS:
// a live dedicated group leader with the reported birth, the driver's own PID
// when it reports one, and the driver's TTY device (TS-01.R41).
func verifyOwnedProcess(h bootstrapHandshake, driverPID int, ttyPath string) (time.Time, error) {
	if h.PID <= 0 || h.PGID != h.PID || (driverPID != 0 && driverPID != h.PID) {
		return time.Time{}, fmt.Errorf("terminal: bootstrap process %d does not match the driver's view", h.PID)
	}
	if pgid, err := unix.Getpgid(h.PID); err != nil || pgid != h.PID {
		return time.Time{}, fmt.Errorf("terminal: process %d does not lead its own group", h.PID)
	}
	birth := time.UnixMicro(h.BirthUS)
	if h.BirthUS <= 0 || rt.ClassifyProcess(h.PID, birth) != rt.ProcessOwned {
		return time.Time{}, fmt.Errorf("terminal: process %d identity could not be proved", h.PID)
	}
	if ttyPath != "" {
		var st unix.Stat_t
		if err := unix.Stat(ttyPath, &st); err != nil || uint64(st.Rdev) != h.TTYDev {
			return time.Time{}, fmt.Errorf("terminal: process %d is not attached to %s", h.PID, ttyPath)
		}
	}
	return birth, nil
}

// RunBootstrap is the child side, run by the hidden command inside a newly
// owned terminal view. It consumes the record once, proves it leads the
// foreground group of its controlling terminal, reports that proof and execs
// the exact argv/env/cwd. It returns only on failure.
func RunBootstrap(recordPath string) error {
	rec, err := consumeLaunchRecord(recordPath)
	if err != nil {
		return err
	}
	handshake := filepath.Join(filepath.Dir(recordPath), "handshake.json")
	h := bootstrapHandshake{Nonce: rec.Nonce, PID: os.Getpid(), PGID: unix.Getpgrp()}
	if err := proveForeground(&h); err != nil {
		h.Error = err.Error()
		_ = writeHandshake(handshake, h)
		return err
	}
	path, err := lookPathIn(rec.Argv[0], rec.Env)
	if err == nil {
		err = os.Chdir(rec.Cwd)
	}
	if err != nil {
		h.Error = err.Error()
		_ = writeHandshake(handshake, h)
		return err
	}
	if err := writeHandshake(handshake, h); err != nil {
		return err
	}
	return syscall.Exec(path, rec.Argv, rec.Env)
}

func proveForeground(h *bootstrapHandshake) error {
	if h.PGID != h.PID {
		return fmt.Errorf("process %d does not lead its own group", h.PID)
	}
	fg, err := unix.IoctlGetInt(0, unix.TIOCGPGRP)
	if err != nil || fg != h.PGID {
		return fmt.Errorf("process %d is not the foreground group of its terminal", h.PID)
	}
	var st unix.Stat_t
	if err := unix.Fstat(0, &st); err != nil {
		return fmt.Errorf("terminal device unreadable: %w", err)
	}
	h.TTYDev = uint64(st.Rdev)
	birth, err := rt.ProcessStartTime(h.PID)
	if err != nil {
		return fmt.Errorf("process birth unavailable: %w", err)
	}
	h.BirthUS = birth.UnixMicro()
	return nil
}

// consumeLaunchRecord reads a record only from a private directory and an
// owner-only regular file, removing it before use so it cannot be replayed.
func consumeLaunchRecord(path string) (launchRecord, error) {
	var rec launchRecord
	dir := filepath.Dir(path)
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0o700 || !ownedByMe(info) {
		return rec, fmt.Errorf("terminal: launch directory is not private")
	}
	data, err := readOwnerFile(path, bootstrapRecordLimit)
	if err != nil {
		return rec, fmt.Errorf("terminal: launch record: %w", err)
	}
	if err := os.Remove(path); err != nil {
		return rec, fmt.Errorf("terminal: consume launch record: %w", err)
	}
	if err := json.Unmarshal(data, &rec); err != nil {
		return rec, fmt.Errorf("terminal: launch record: %w", err)
	}
	if rec.Version != 1 || rec.Nonce == "" || len(rec.Argv) == 0 || rec.Cwd == "" {
		return rec, fmt.Errorf("terminal: launch record is incomplete")
	}
	return rec, checkRecordBounds(rec.Argv, rec.Env)
}

// readOwnerFile opens without following links and accepts only a regular
// file owned by this user with no group/other access.
func readOwnerFile(path string, limit int64) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 || !ownedByMe(info) {
		return nil, fmt.Errorf("%s is not a private regular file", filepath.Base(path))
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("%s exceeds %d bytes", filepath.Base(path), limit)
	}
	return io.ReadAll(io.LimitReader(f, limit+1))
}

func ownedByMe(info os.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(st.Uid) == os.Getuid()
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// writeHandshake publishes the report atomically so the parent never reads a
// partial file.
func writeHandshake(path string, h bootstrapHandshake) error {
	data, err := json.Marshal(h)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := writeExclusive(tmp, data); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// lookPathIn resolves argv[0] against the record's PATH, not the bootstrap's
// inherited one, so the exec matches the composed environment.
func lookPathIn(name string, env []string) (string, error) {
	if strings.Contains(name, "/") {
		return name, nil
	}
	for i := len(env) - 1; i >= 0; i-- {
		if v, ok := strings.CutPrefix(env[i], "PATH="); ok {
			for _, dir := range filepath.SplitList(v) {
				if dir == "" {
					continue
				}
				candidate := filepath.Join(dir, name)
				if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0 {
					return candidate, nil
				}
			}
			return "", fmt.Errorf("%s not found on the launch PATH", name)
		}
	}
	return exec.LookPath(name)
}
