package hooks

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func TestInstallWritesExecutableScripts(t *testing.T) {
	home := t.TempDir()
	if err := Install(home); err != nil {
		t.Fatalf("Install: %v", err)
	}
	for _, name := range allScripts {
		p := filepath.Join(Dir(home), name)
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing script %q: %v", name, err)
		}
		if runtime.GOOS != "windows" && fi.Mode().Perm()&0o100 == 0 {
			t.Fatalf("script %q not executable: %v", name, fi.Mode())
		}
	}
	// Idempotent re-install (overwrite in place).
	if err := Install(home); err != nil {
		t.Fatalf("re-Install: %v", err)
	}
}

func TestClaudeSettingsMapsHookKeysToScripts(t *testing.T) {
	home := "/home/u/.chuck"
	hookMap := map[string]string{
		"SessionStart": "SessionStart",
		"PreToolUse":   "PreToolUse",
		"Stop":         "Stop",
	}
	settings := ClaudeSettings(home, hookMap)
	block, ok := settings["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("settings.hooks missing/wrong type: %#v", settings)
	}
	if len(block) != len(hookMap) {
		t.Fatalf("hooks block has %d keys, want %d", len(block), len(hookMap))
	}
	// PreToolUse → [{hooks:[{type:command, command: <pre-tool-use.sh path>}]}]
	entry := block["PreToolUse"].([]any)[0].(map[string]any)
	cmd := entry["hooks"].([]any)[0].(map[string]any)
	if cmd["type"] != "command" {
		t.Fatalf("hook type = %v, want command", cmd["type"])
	}
	want := ScriptPath(home, "PreToolUse")
	if cmd["command"] != want {
		t.Fatalf("command = %v, want %v", cmd["command"], want)
	}
}

func TestWriteAgentSettings(t *testing.T) {
	home := t.TempDir()
	path, err := WriteAgentSettings(home, "a_123", map[string]any{"hooks": map[string]any{}})
	if err != nil {
		t.Fatalf("WriteAgentSettings: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file not written: %v", err)
	}
	if !strings.HasPrefix(path, filepath.Join(Dir(home), "agents")) {
		t.Fatalf("settings path = %q, want under hooks/agents", path)
	}
}

// TestInterfaceGate runs the installed _post.sh with shimmed curl+jq so it is
// hermetic. The gate (§4.3) must suppress the POST for a chat agent on a covered
// event, and emit it for a terminal agent.
func TestInterfaceGate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sh hook scripts")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("no POSIX sh")
	}
	home := t.TempDir()
	if err := Install(home); err != nil {
		t.Fatalf("Install: %v", err)
	}
	post := filepath.Join(Dir(home), "_post.sh")

	// Shim dir on PATH: curl records each invocation; jq echoes a fixed body so
	// the script doesn't depend on a real jq being installed.
	shim := t.TempDir()
	curlLog := filepath.Join(t.TempDir(), "curl.log")
	writeShim(t, filepath.Join(shim, "curl"), "#!/bin/sh\necho called >> \"$CURL_LOG\"\n")
	writeShim(t, filepath.Join(shim, "jq"), "#!/bin/sh\necho '{}'\n")

	run := func(iface, event string) bool {
		_ = os.Remove(curlLog)
		cmd := exec.Command(post, event, "busy", "detail=x")
		cmd.Env = append(os.Environ(),
			"PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"),
			"CURL_LOG="+curlLog,
			"CHUCK_AGENT_ID=a_1",
			"CHUCK_HOOK_URL=http://127.0.0.1:9/api/hook",
			"CHUCK_HOOK_TOKEN=tok",
			"CHUCK_INTERFACE="+iface,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("_post.sh %s/%s: %v\n%s", iface, event, err, out)
		}
		_, err := os.Stat(curlLog)
		return err == nil // curl invoked iff the log exists
	}

	// Chat agent on a covered event → gate suppresses the POST.
	if run("chat", "PreToolUse") {
		t.Fatalf("chat PreToolUse POSTed; gate should suppress it")
	}
	// Terminal agent on the same event → POST fires.
	if !run("terminal", "PreToolUse") {
		t.Fatalf("terminal PreToolUse did not POST")
	}
}

// ASW3-3 / FS-07.R25: a SessionStart posted before the launching server has
// written its authenticating running row (401) is retried until admitted;
// other events and other rejections are posted once.
func TestSessionStartRetriesUntilAdmitted(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX sh hook scripts")
	}
	if _, err := exec.LookPath("curl"); err != nil {
		t.Skip("no curl")
	}
	home := t.TempDir()
	if err := Install(home); err != nil {
		t.Fatalf("Install: %v", err)
	}
	post := filepath.Join(Dir(home), "_post.sh")
	shim := t.TempDir()
	writeShim(t, filepath.Join(shim, "jq"), "#!/bin/sh\necho '{}'\n")

	var mu sync.Mutex
	var requests, admitAfter int
	reject := http.StatusUnauthorized
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests++
		if requests <= admitAfter {
			w.WriteHeader(reject)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	run := func(event string, rejected, status int) int {
		mu.Lock()
		requests, admitAfter, reject = 0, rejected, status
		mu.Unlock()
		cmd := exec.Command(post, event, "idle")
		cmd.Env = append(os.Environ(),
			"PATH="+shim+string(os.PathListSeparator)+os.Getenv("PATH"),
			"CHUCK_AGENT_ID=a_1", "CHUCK_HOOK_URL="+srv.URL, "CHUCK_HOOK_TOKEN=tok",
			"CHUCK_INTERFACE=terminal",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("_post.sh %s: %v\n%s", event, err, out)
		}
		mu.Lock()
		defer mu.Unlock()
		return requests
	}

	if n := run("SessionStart", 2, http.StatusUnauthorized); n != 3 {
		t.Fatalf("SessionStart before its row: %d posts, want admitted on the third", n)
	}
	if n := run("SessionStart", 100, http.StatusUnauthorized); n != 6 {
		t.Fatalf("never-admitted SessionStart: %d posts, want bounded at 6", n)
	}
	if n := run("SessionStart", 1, http.StatusBadRequest); n != 1 {
		t.Fatalf("rejected SessionStart: %d posts, want 1", n)
	}
	if n := run("PreToolUse", 1, http.StatusUnauthorized); n != 1 {
		t.Fatalf("PreToolUse: %d posts, want 1", n)
	}
}

func writeShim(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write shim %q: %v", path, err)
	}
}
