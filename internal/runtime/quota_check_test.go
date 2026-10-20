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
	"unicode/utf8"
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
	return QuotaCheckSpec{BackendType: backendType, ModelID: "claude-sonnet-4-6", Env: append(base, env...), SourceCwd: h.dir,
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
		opts["strictMcpConfig"] != true || opts["model"] != "claude-sonnet-4-6" || len(opts["tools"].([]any)) != 0 || opts["additionalDirectories"] != nil ||
		opts["settingSources"] == nil || len(opts["settingSources"].([]any)) != 0 || opts["settings"] != nil {
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

// FS-01.R51: the pinned adapter loads user/project/local settings unless the
// caller's settingSources replaces them, and those layers carry instructions
// and lifecycle hooks. A check loads none of them and carries only the routing
// keys the original process would have applied.
func TestQuotaCheckClaudeExcludesAmbientSettingsButKeepsRouting(t *testing.T) {
	h := newQuotaCheckHarness(t)
	project := filepath.Join(h.dir, "project")
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(h.dir, ".claude", "CLAUDE.md"), "Always greet the user in French.")
	write(filepath.Join(h.dir, ".claude", "settings.json"), `{"env":{"ANTHROPIC_BASE_URL":"https://gateway.example","SHARED":"user"},
"apiKeyHelper":"~/bin/key","hooks":{"SessionStart":[{"hooks":[{"type":"command","command":"touch /tmp/hooked"}]}]},"enabledPlugins":{"x":true}}`)
	write(filepath.Join(project, "CLAUDE.md"), "Project instructions.")
	write(filepath.Join(project, ".claude", "settings.local.json"), `{"env":{"SHARED":"local"},"permissions":{"allow":["Bash"]}}`)
	spec, _ := h.spec("claude-acp")
	spec.SourceCwd = project
	if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "ready" {
		t.Fatalf("outcome = %+v", out)
	}
	var params struct {
		Meta struct {
			ClaudeCode struct {
				Options map[string]any `json:"options"`
			} `json:"claudeCode"`
		} `json:"_meta"`
	}
	raw, _ := os.ReadFile(h.newDump)
	if err := json.Unmarshal(raw, &params); err != nil {
		t.Fatal(err)
	}
	opts := params.Meta.ClaudeCode.Options
	settings, _ := json.Marshal(opts["settings"])
	if sources, ok := opts["settingSources"].([]any); !ok || len(sources) != 0 ||
		string(settings) != `{"apiKeyHelper":"~/bin/key","env":{"ANTHROPIC_BASE_URL":"https://gateway.example","SHARED":"local"}}` {
		t.Fatalf("session/new options = %s", raw)
	}

	// Routing that would replace the adapter's programmatic model
	// configuration, or an unreadable layer, cannot be shown equivalent.
	h = newQuotaCheckHarness(t)
	write(filepath.Join(h.dir, ".claude", "settings.json"), `{"env":{"ANTHROPIC_BASE_URL":"https://gateway.example"}}`)
	spec, recorded := h.spec("claude-acp", `CLAUDE_MODEL_CONFIG={"availableModels":["sonnet"]}`)
	if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "unsupported" || len(*recorded) != 0 {
		t.Fatalf("model config conflict = %+v", out)
	}
	write(filepath.Join(h.dir, ".claude", "settings.json"), `{not json`)
	spec, recorded = h.spec("claude-acp")
	if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "unsupported" || len(*recorded) != 0 {
		t.Fatalf("unreadable settings = %+v", out)
	}
}

// Q5: a Sonnet turn's quota is not scheduled from an Opus-only window.
func TestQuotaCheckIgnoresAnotherModelsWindow(t *testing.T) {
	h := newQuotaCheckHarness(t)
	rate, _ := json.Marshal(map[string]any{"status": "rejected", "rateLimitType": "seven_day_opus",
		"resetsAt": time.Now().UTC().Add(3 * time.Hour).Unix()})
	failure := `{"id":"check:error","revision":1,"category":"limit","severity":"error","title":"You've hit your limit","actions":[]}`
	spec, _ := h.spec("claude-acp", "FAKEACP_RATE_LIMIT="+string(rate), "FAKEACP_FAILURE_RESULT="+failure)
	if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "quota" || out.ResetAt != nil {
		t.Fatalf("outcome = %+v, want resetless quota", out)
	}
}

// Q7: an explicit spend-cap/credit refusal is an intervention result.
func TestQuotaCheckReportsIntervention(t *testing.T) {
	for backend, title := range map[string]string{
		"claude-acp": "You've hit your monthly spend limit · ask your admin to raise it at claude.ai/admin-settings/usage",
		"codex-acp":  "Your workspace is out of credits. Add credits to continue.",
	} {
		h := newQuotaCheckHarness(t)
		failure, _ := json.Marshal(map[string]any{"id": "check:error", "revision": 1, "category": "limit", "severity": "error", "title": title, "actions": []string{}})
		spec, _ := h.spec(backend, "FAKEACP_FAILURE_RESULT="+string(failure))
		if out := h.c.RunQuotaCheck(context.Background(), spec); out.Outcome != "quota" || out.ResetAt != nil || !out.Intervention {
			t.Fatalf("%s outcome = %+v, want intervention", backend, out)
		}
	}
}

// Q9: the reply capture is a byte cap, clipped on a UTF-8 boundary.
func TestQuotaCheckReplyRespectsByteBound(t *testing.T) {
	p := &quotaCheckProbe{sessionID: "probe"}
	chunk := func(text string) {
		frame, _ := json.Marshal(map[string]any{"sessionId": "probe", "update": map[string]any{"sessionUpdate": "agent_message_chunk",
			"content": map[string]any{"type": "text", "text": text}}})
		p.onNotification("session/update", frame)
	}
	chunk("a")
	chunk(strings.Repeat("界", 4096))
	if got := p.reply.String(); len(got) > maxQuotaCheckReply || len(got) < maxQuotaCheckReply-3 || !utf8.ValidString(got) {
		t.Fatalf("reply retained %d bytes (valid UTF-8: %v)", len(got), utf8.ValidString(got))
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
	if err := json.Unmarshal(overlay, &cfg); err != nil || cfg["mcp_servers.docs.enabled"] != false || cfg["model_reasoning_summary"] != "auto" || cfg["developer_instructions"] != "" {
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

// Q2: servers named only through the overlay, in either the nested or the
// dotted override shape, are disabled; ambiguous names refuse isolation.
func TestCodexCheckIsolationDisablesOverlayServers(t *testing.T) {
	for name, raw := range map[string]string{
		"dotted": `{"mcp_servers.docs.command":"docs-mcp","mcp_servers.docs.enabled":true}`,
		"nested": `{"mcp_servers":{"docs":{"command":"docs-mcp","enabled":true}}}`,
	} {
		env, err := withCodexCheckIsolation([]string{"HOME=" + t.TempDir(), "CODEX_CONFIG=" + raw})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var overlay map[string]any
		if err := json.Unmarshal([]byte(envValue(env, "CODEX_CONFIG")), &overlay); err != nil {
			t.Fatal(err)
		}
		nested, _ := overlay["mcp_servers"].(map[string]any)
		if docs, ok := nested["docs"].(map[string]any); overlay["mcp_servers.docs.enabled"] != false || ok && docs["enabled"] != false {
			t.Fatalf("%s overlay keeps docs enabled: %s", name, envValue(env, "CODEX_CONFIG"))
		}
	}
	for _, raw := range []string{`{"mcp_servers.\"a.b\".enabled":true}`, `{"mcp_servers":"docs"}`} {
		if _, err := withCodexCheckIsolation([]string{"HOME=" + t.TempDir(), "CODEX_CONFIG=" + raw}); err == nil {
			t.Fatalf("ambiguous overlay %s was accepted", raw)
		}
	}
}

// ASRG-4: work instructions never reach a check, with or without MCP servers.
func TestCodexCheckIsolationSuppressesWorkInstructions(t *testing.T) {
	decode := func(env []string) map[string]any {
		var o map[string]any
		if err := json.Unmarshal([]byte(envValue(env, "CODEX_CONFIG")), &o); err != nil {
			t.Fatalf("CODEX_CONFIG: %v (%q)", err, envValue(env, "CODEX_CONFIG"))
		}
		return o
	}
	overlay := `{"model":"gpt-x","model_provider":"corp","developer_instructions":"do the work","model_instructions_file":"/tmp/i.md"}`
	env, err := withCodexCheckIsolation([]string{"HOME=" + t.TempDir(), "CODEX_CONFIG=" + overlay})
	if err != nil {
		t.Fatal(err)
	}
	got := decode(env)
	if got["developer_instructions"] != "" || got["model"] != "gpt-x" || got["model_provider"] != "corp" {
		t.Fatalf("overlay = %v", got)
	}
	if _, ok := got["model_instructions_file"]; ok {
		t.Fatalf("instruction file kept: %v", got)
	}

	// No overlay at all still emits an explicit suppression.
	env, err = withCodexCheckIsolation([]string{"HOME=" + t.TempDir()})
	if err != nil || decode(env)["developer_instructions"] != "" {
		t.Fatalf("no-overlay env = %v err=%v", env, err)
	}

	// A native layer's instructions are suppressed by the overlay; MCP stays disabled.
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.toml"),
		[]byte("developer_instructions = \"native\"\n[mcp_servers.docs]\ncommand = \"docs-mcp\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env, err = withCodexCheckIsolation([]string{"CODEX_HOME=" + home})
	if err != nil {
		t.Fatal(err)
	}
	got = decode(env)
	if got["developer_instructions"] != "" || got["mcp_servers.docs.enabled"] != false {
		t.Fatalf("native overlay = %v", got)
	}

	// Inseparable sources refuse: instruction files, profile-scoped and base
	// instructions, and global home instruction files (TS-04.R97/R101).
	for name, file := range map[string][2]string{
		"file":              {"config.toml", "model_instructions_file = \"/tmp/i.md\"\n"},
		"profile":           {"config.toml", "[profiles.p]\ndeveloper_instructions = \"x\"\n"},
		"base-instructions": {"config.toml", "instructions = 'base work instructions'\n"},
		"AGENTS.md":         {"AGENTS.md", "global work instructions"},
		"AGENTS.override":   {"AGENTS.override.md", "override work instructions"},
	} {
		h := t.TempDir()
		if err := os.WriteFile(filepath.Join(h, file[0]), []byte(file[1]), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := withCodexCheckIsolation([]string{"CODEX_HOME=" + h}); !errors.Is(err, errQuotaCheckUnsupported) {
			t.Fatalf("%s: err = %v", name, err)
		}
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
