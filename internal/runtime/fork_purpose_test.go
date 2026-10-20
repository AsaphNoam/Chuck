package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TS-04.R101 / FS-01.A35: the pinned Claude fork answers only an id it never
// registers. The clone opens it through session/load with this generation's
// MCP/directories before configuration and its first prompt.
func TestClaudeForkOpensTheReturnedSessionBeforeConfiguration(t *testing.T) {
	c, spec, dir := forkSpec(t, "FAKEACP_CAPS=1", "FAKEACP_FORK_UNREGISTERED=1")
	loadDump := filepath.Join(dir, "load.json")
	promptLog := filepath.Join(dir, "prompts.log")
	spec.Env = append(spec.Env, "FAKEACP_LOAD_DUMP="+loadDump, "FAKEACP_PROMPT_LOG="+promptLog)
	spec.Effort = "high"
	spec.MCPServers = []MCPServerSpec{{Name: "chuck", URL: "http://127.0.0.1:1/mcp"}}
	ctx := context.Background()
	h, err := c.Start(ctx, spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop(ctx, h.AgentID) })
	if h.SessionID != "fake-fork-1" {
		t.Fatalf("session = %q", h.SessionID)
	}
	raw, err := os.ReadFile(loadDump)
	if err != nil {
		t.Fatalf("fork was not opened by session/load: %v", err)
	}
	var load struct {
		SessionID  string           `json:"sessionId"`
		MCPServers []map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(raw, &load); err != nil || load.SessionID != "fake-fork-1" || len(load.MCPServers) != 1 {
		t.Fatalf("load params = %s", raw)
	}
	if err := c.SendPrompt(ctx, h.AgentID, "first clone prompt"); err != nil {
		t.Fatalf("SendPrompt: %v", err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		if b, _ := os.ReadFile(promptLog); len(b) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("first prompt never reached a registered session")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// A failed open deletes the forked provider session and creates nothing.
func TestClaudeForkOpenFailureDeletesTheFork(t *testing.T) {
	c, spec, dir := forkSpec(t, "FAKEACP_CAPS=1", "FAKEACP_FORK_UNREGISTERED=1",
		`FAKEACP_LOAD_REJECT_DATA={"message":"load refused"}`)
	if _, err := c.Start(context.Background(), spec); err == nil {
		t.Fatal("expected the fork open failure")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "delete.json"))
	if err != nil || !strings.Contains(string(raw), "fake-fork-1") {
		t.Fatalf("delete params = %s (%v)", raw, err)
	}
	if _, err := c.store.ReadRunning(spec.Agent.AgentID); err == nil {
		t.Fatal("failed fork left a running row")
	}
}

// TS-04.R65/R101 (ASRB-3): a Codex fork's process overlay keeps the native
// setup and carries no newly composed persona or bridge; a fresh start of the
// same spec does carry them.
func TestCodexForkProcessCarriesNoNewInstructions(t *testing.T) {
	const overlay = `{"model":"gpt-5.5","developer_instructions":"NATIVE_SENTINEL"}`
	run := func(fork bool) string {
		c, spec, dir := forkSpec(t, "FAKEACP_CAPS=1", "CODEX_CONFIG="+overlay)
		dump := filepath.Join(dir, "codex_config")
		spec.Env = append(spec.Env, "FAKEACP_FORK_ENV_DUMP="+dump, "FAKEACP_NEW_RAW_DUMP="+dump)
		spec.BackendType, spec.ModelID = "codex-acp", "gpt-5.5"
		spec.SystemPrompt, spec.HistoryPrimer = "SYSTEM_SENTINEL", "PRIMER_SENTINEL"
		if !fork {
			spec.Fork = nil
		}
		h, err := c.Start(context.Background(), spec)
		if err != nil {
			t.Fatalf("Start(fork=%t): %v", fork, err)
		}
		_ = c.Stop(context.Background(), h.AgentID)
		name := dump
		if !fork {
			name = dump + ".codex_config"
		}
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return string(raw)
	}
	if got := run(true); got != overlay {
		t.Fatalf("fork CODEX_CONFIG = %s, want unchanged native overlay", got)
	}
	if got := run(false); !strings.Contains(got, "SYSTEM_SENTINEL") || !strings.Contains(got, "NATIVE_SENTINEL") {
		t.Fatalf("fresh CODEX_CONFIG = %s, want native then Chuck guidance", got)
	}
}
