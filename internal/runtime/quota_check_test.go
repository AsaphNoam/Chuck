package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type quotaCheckHarness struct {
	c        *ChatRuntime
	dir      string
	prompts  string
	newDump  string
	closeLog string
	delLog   string
}

func newQuotaCheckHarness(t *testing.T) quotaCheckHarness {
	t.Helper()
	c, _ := newChatTest(t, "stream_text")
	dir := t.TempDir()
	return quotaCheckHarness{c: c, dir: dir, prompts: filepath.Join(dir, "prompts.log"), newDump: filepath.Join(dir, "new.json"),
		closeLog: filepath.Join(dir, "close.json"), delLog: filepath.Join(dir, "delete.json")}
}

func (h quotaCheckHarness) spec(backendType string, env ...string) (QuotaCheckSpec, *[]string) {
	recorded := &[]string{}
	base := []string{"HOME=" + h.dir, "FAKEACP_SCENARIO=stream_text", "FAKEACP_CAPS=1", "FAKEACP_PROMPT_LOG=" + h.prompts,
		"FAKEACP_NEW_RAW_DUMP=" + h.newDump, "FAKEACP_CLOSE_LOG=" + h.closeLog, "FAKEACP_DELETE_LOG=" + h.delLog}
	return QuotaCheckSpec{BackendType: backendType, ModelID: "claude-sonnet-4-6", Env: append(base, env...),
		OnSessionCreated: func(id string) error {
			// The probe session is durable before any prompt crosses the wire.
			if _, err := os.Stat(h.prompts); err == nil {
				return errors.New("prompt sent before session was recorded")
			}
			*recorded = append(*recorded, id)
			return nil
		}}, recorded
}

// TS-04.R97 / FS-01.A34: a Claude check is a separate non-persistent,
// tool-free session with no MCP or work instructions, closed after its reply.
func TestQuotaCheckClaudeIsolatedReadyAndClosed(t *testing.T) {
	h := newQuotaCheckHarness(t)
	spec, recorded := h.spec("claude-acp")
	out := h.c.RunQuotaCheck(context.Background(), spec)
	if out.Outcome != "ready" || !out.Cleaned || len(*recorded) != 1 || out.SessionID != (*recorded)[0] {
		t.Fatalf("outcome = %+v recorded=%v", out, *recorded)
	}
	var params struct {
		Cwd        string          `json:"cwd"`
		MCPServers json.RawMessage `json:"mcpServers"`
		Meta       struct {
			SystemPrompt json.RawMessage `json:"systemPrompt"`
			ClaudeCode   struct {
				Options map[string]any `json:"options"`
			} `json:"claudeCode"`
		} `json:"_meta"`
	}
	raw, _ := os.ReadFile(h.newDump)
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	opts := params.Meta.ClaudeCode.Options
	if string(params.MCPServers) != "[]" || params.Meta.SystemPrompt != nil || opts["persistSession"] != false ||
		opts["strictMcpConfig"] != true || opts["model"] != "claude-sonnet-4-6" || len(opts["tools"].([]any)) != 0 || opts["additionalDirectories"] != nil {
		t.Fatalf("session/new = %s", raw)
	}
	if _, err := os.Stat(params.Cwd); !os.IsNotExist(err) {
		t.Fatalf("temporary cwd %q survived: %v", params.Cwd, err)
	}
	if closed, _ := os.ReadFile(h.closeLog); !strings.Contains(string(closed), out.SessionID) {
		t.Fatalf("session/close = %s", closed)
	}
	prompts, _ := os.ReadFile(h.prompts)
	if strings.Count(string(prompts), "\n") != 1 || !strings.Contains(string(prompts), quotaCheckPrompt) {
		t.Fatalf("prompts = %s", prompts)
	}
}

func TestQuotaCheckClaudeQuotaCarriesVerifiedReset(t *testing.T) {
	h := newQuotaCheckHarness(t)
	reset := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Minute)
	if reset.Minute() == 0 {
		reset = reset.Add(time.Minute)
	}
	clause := strings.NewReplacer("PM", "pm", "AM", "am").Replace(reset.Format("Jan 2, 2006, 3:04PM"))
	failure, _ := json.Marshal(map[string]any{"id": "check:error", "revision": 1, "category": "limit", "severity": "error",
		"title": "You've hit your weekly limit · resets " + clause + " (UTC)", "actions": []string{}})
	spec, _ := h.spec("claude-acp", "FAKEACP_FAILURE_RESULT="+string(failure))
	out := h.c.RunQuotaCheck(context.Background(), spec)
	if out.Outcome != "quota" || out.ResetAt == nil || !out.ResetAt.Equal(reset) || out.ResetSource != "text" || !out.Cleaned {
		t.Fatalf("outcome = %+v, want quota with reset %s", out, reset)
	}
}

// Codex checks disable every configured MCP server through the process
// overlay, take an explicit name before the probe and archive the session.
func TestQuotaCheckCodexRenamesDisablesMCPAndArchives(t *testing.T) {
	h := newQuotaCheckHarness(t)
	if err := os.MkdirAll(filepath.Join(h.dir, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(h.dir, ".codex", "config.toml"), []byte("[mcp_servers.docs]\ncommand = \"docs-mcp\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	spec, _ := h.spec("codex-acp", `CODEX_CONFIG={"model_reasoning_summary":"auto"}`)
	spec.ModelID = ""
	out := h.c.RunQuotaCheck(context.Background(), spec)
	if out.Outcome != "ready" || !out.Cleaned {
		t.Fatalf("outcome = %+v", out)
	}
	overlay, _ := os.ReadFile(h.newDump + ".codex_config")
	var cfg map[string]any
	if err := json.Unmarshal(overlay, &cfg); err != nil || cfg["mcp_servers.docs.enabled"] != false || cfg["model_reasoning_summary"] != "auto" || cfg["developer_instructions"] != nil {
		t.Fatalf("CODEX_CONFIG = %s", overlay)
	}
	prompts, _ := os.ReadFile(h.prompts)
	lines := strings.Split(strings.TrimSpace(string(prompts)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "/rename "+quotaCheckName) || !strings.Contains(lines[1], quotaCheckPrompt) {
		t.Fatalf("prompts = %q", lines)
	}
	if deleted, _ := os.ReadFile(h.delLog); !strings.Contains(string(deleted), out.SessionID) {
		t.Fatalf("session/delete = %s", deleted)
	}
}

func TestQuotaCheckRefusesUncorroboratedNameOrMissingDisposal(t *testing.T) {
	h := newQuotaCheckHarness(t)
	spec, _ := h.spec("codex-acp", "FAKEACP_RENAME_SILENT=1")
	out := h.c.RunQuotaCheck(context.Background(), spec)
	prompts, _ := os.ReadFile(h.prompts)
	if out.Outcome != "unsupported" || strings.Contains(string(prompts), quotaCheckPrompt) {
		t.Fatalf("silent rename = %+v prompts=%s", out, prompts)
	}
	if deleted, _ := os.ReadFile(h.delLog); !strings.Contains(string(deleted), out.SessionID) {
		t.Fatalf("refused probe was not archived: %s", deleted)
	}

	h = newQuotaCheckHarness(t)
	spec, recorded := h.spec("claude-acp")
	spec.Env = append(spec.Env, "FAKEACP_CAPS=")
	if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "unsupported" || len(*recorded) != 0 {
		t.Fatalf("unadvertised lifecycle = %+v", out)
	}

	h = newQuotaCheckHarness(t)
	spec, _ = h.spec("claude-acp")
	spec.OnSessionCreated = func(string) error { return errors.New("store unavailable") }
	out = h.c.RunQuotaCheck(context.Background(), spec)
	if _, err := os.Stat(h.prompts); out.Outcome != "failed" || err == nil {
		t.Fatalf("unrecorded session = %+v, prompt sent: %v", out, err == nil)
	}
	if closed, _ := os.ReadFile(h.closeLog); !strings.Contains(string(closed), out.SessionID) {
		t.Fatalf("unrecorded session not disposed: %s", closed)
	}
}

func TestQuotaCheckCleanupRetriesExactCodexSession(t *testing.T) {
	h := newQuotaCheckHarness(t)
	spec, _ := h.spec("codex-acp")
	if err := h.c.CleanupQuotaCheckSession(context.Background(), spec, "probe-exact"); err != nil {
		t.Fatal(err)
	}
	if deleted, _ := os.ReadFile(h.delLog); !strings.Contains(string(deleted), `"probe-exact"`) {
		t.Fatalf("session/delete = %s", deleted)
	}
	if _, err := os.Stat(h.prompts); err == nil {
		t.Fatal("cleanup prompted the probe")
	}
	if err := h.c.CleanupQuotaCheckSession(context.Background(), QuotaCheckSpec{BackendType: "claude-acp"}, "gone"); err != nil {
		t.Fatalf("claude cleanup = %v, want no-op for a non-persistent session", err)
	}
}
