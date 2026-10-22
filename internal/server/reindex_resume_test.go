package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AsaphNoam/Chuck/internal/config"
	"github.com/AsaphNoam/Chuck/internal/index"
)

// FS-05.A26 / FS-05.R43: a failed offline replay preserves the latest native
// identity and frozen setup so the ordinary Resume path can load the existing
// provider conversation. The damaged tail exercises failed replay; the fake
// ACP load dump is the independent wire oracle.
func TestFailedReindexPreservesNativeResume(t *testing.T) {
	fake := buildFakeACP(t)
	t.Setenv("FAKEACP_SCENARIO", "stream_text")
	t.Setenv("FAKEACP_FAST_OPTION", "fast-mode")
	loadDump := filepath.Join(t.TempDir(), "load.json")
	t.Setenv("FAKEACP_LOAD_DUMP", loadDump)
	newDump := filepath.Join(t.TempDir(), "new.json")
	t.Setenv("FAKEACP_NEW_DUMP", newDump)

	srv := testServer(t, true)
	srv.registry.Chat().SetCommand(fake)
	cwd := t.TempDir()
	addDir := t.TempDir()
	if err := srv.configStore.WriteProject("tmpproj", config.Project{Title: "Tmp", Cwd: cwd, AddDirs: []string{addDir}}); err != nil {
		t.Fatalf("WriteProject: %v", err)
	}
	if err := srv.configStore.WriteRole("impl", config.Role{Title: "Impl", SystemPrompt: "frozen persona", SkipPermissions: boolPtrForReindex(true)}); err != nil {
		t.Fatalf("WriteRole: %v", err)
	}
	if err := srv.configStore.WriteBackends(config.BackendsConfig{Version: 2, Backends: map[string]config.Backend{
		"claude": {Name: "Claude", Type: "claude-acp", Default: true, DefaultModel: "sonnet", Models: map[string]config.Model{
			"sonnet": {Name: "Sonnet", Model: "sonnet", Efforts: []string{"low", "high"}, DefaultEffort: "high", Fast: true},
		}},
	}}); err != nil {
		t.Fatalf("WriteBackends: %v", err)
	}

	ts := httptest.NewServer(srv.routes())
	defer ts.Close()
	t.Cleanup(func() { srv.registry.Shutdown(context.Background()) })
	resp, body := post(t, ts.URL+"/api/sessions", map[string]any{
		"role": "impl", "project": "tmpproj", "backend": "claude", "model": "sonnet", "effort": "high", "fast": true,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("launch status = %d: %s", resp.StatusCode, body)
	}
	var launched sessionResponse
	if err := json.Unmarshal(body, &launched); err != nil {
		t.Fatalf("decode launch: %v", err)
	}
	id := launched.Agent.AgentID
	if id == "" {
		t.Fatal("launch returned empty agent id")
	}
	waitForStatus(t, srv, id, "idle")
	if resp, body = post(t, ts.URL+"/api/sessions/"+id+"/stop", nil); resp.StatusCode != http.StatusOK {
		t.Fatalf("stop status = %d: %s", resp.StatusCode, body)
	}
	frozenBefore, err := srv.stateStore.ReadSession(id)
	if err != nil {
		t.Fatalf("ReadSession before failed reindex: %v", err)
	}
	if !strings.Contains(frozenBefore.SystemPrompt, "frozen persona") || !containsReindexString(frozenBefore.AddDirs, addDir) {
		t.Fatalf("initial frozen setup omitted fixture values: %+v", frozenBefore)
	}
	if err := os.Remove(newDump); err != nil {
		t.Fatalf("remove launch session/new marker: %v", err)
	}

	// This value is learned by the native hook after the transcript metadata was
	// written. Reindex must retain it instead of deriving an older identity.
	const latestNativeID = "native-latest-after-reindex"
	if _, err := srv.stateStore.DB().Exec(`UPDATE sessions SET last_session_id = ?, effort = ?, fast = ? WHERE agent_id = ?`, latestNativeID, "high", true, id); err != nil {
		t.Fatalf("retain native fields: %v", err)
	}
	transcriptPath := filepath.Join(srv.configStore.Home(), "sessions", id, "transcript.ndjson")
	f, err := os.OpenFile(transcriptPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open transcript tail: %v", err)
	}
	// The envelope is valid, but its metadata payload is not. This models a
	// failed replay of a damaged source while the prior session remains usable.
	if _, err := f.WriteString(`{"seq":999,"type":"session_meta","data":"damaged metadata","ts":"2026-10-22T00:00:00Z"}` + "\n"); err != nil {
		t.Fatalf("write damaged tail: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close transcript: %v", err)
	}

	err = index.Reindex(srv.configStore.Home(), srv.stateStore.DB())
	if err == nil || !strings.Contains(err.Error(), id) {
		t.Fatalf("Reindex error = %v, want failed replay naming %s", err, id)
	}
	snap, err := srv.stateStore.ReadSession(id)
	if err != nil {
		t.Fatalf("ReadSession after failed reindex: %v", err)
	}
	if snap.LastSessionID != latestNativeID || snap.Effort != frozenBefore.Effort || !snap.Fast || snap.Cwd != frozenBefore.Cwd || snap.SystemPrompt != frozenBefore.SystemPrompt || snap.SkipPermissions != frozenBefore.SkipPermissions || !sameStrings(snap.AddDirs, frozenBefore.AddDirs) {
		t.Fatalf("preserved snapshot = %+v, before = %+v", snap, frozenBefore)
	}

	resp, body = post(t, ts.URL+"/api/sessions/"+id+"/resume", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resume after failed reindex = %d: %s", resp.StatusCode, body)
	}
	loaded, err := os.ReadFile(loadDump)
	if err != nil {
		t.Fatalf("read session/load dump: %v", err)
	}
	var wire struct {
		SessionID string `json:"sessionId"`
		Cwd       string `json:"cwd"`
		Meta      struct {
			SystemPrompt struct {
				Append string `json:"append"`
			} `json:"systemPrompt"`
			ClaudeCode struct {
				Options struct {
					AdditionalDirectories []string `json:"additionalDirectories"`
				} `json:"options"`
			} `json:"claudeCode"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(loaded, &wire); err != nil {
		t.Fatalf("decode session/load dump: %v", err)
	}
	if wire.SessionID != latestNativeID || wire.Cwd != frozenBefore.Cwd || !strings.HasPrefix(wire.Meta.SystemPrompt.Append, frozenBefore.SystemPrompt) || !sameStrings(wire.Meta.ClaudeCode.Options.AdditionalDirectories, frozenBefore.AddDirs) {
		t.Fatalf("session/load wire lost native identity or frozen setup: %+v", wire)
	}
	if _, err := os.Stat(newDump); !os.IsNotExist(err) {
		t.Fatalf("resume started a fresh session: new dump stat = %v", err)
	}
	if _, err := srv.stateStore.ReadRunning(id); err != nil {
		t.Fatalf("resume did not leave agent running: %v", err)
	}
}

func boolPtrForReindex(v bool) *bool { return &v }

func containsReindexString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
