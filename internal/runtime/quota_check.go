package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/AsaphNoam/Chuck/internal/backend"
	"github.com/AsaphNoam/Chuck/internal/config"
)

// Isolated quota checks (TS-04.R97): one short-lived adapter process with the
// agent's frozen provider/account/model selection, a temporary empty cwd, no
// Chuck agent, hooks, MCP, instructions or event sink. Only a real model reply
// with an error-free completion and successful disposal is `ready`.

const (
	QuotaCheckTimeout      = 60 * time.Second
	QuotaCheckCleanupGrace = 5 * time.Second
	quotaCheckPrompt       = "Reply with exactly OK. Do not use tools."
	quotaCheckName         = "Chuck quota check"
	maxQuotaCheckReply     = 4 * 1024
)

// QuotaCheckSpec is the sanitized frozen provider selection for one check.
// Env is the composed provider environment without hook, resource or
// coordination layers. OnSessionCreated must durably record the probe
// session before any prompt; an error refuses further effects.
type QuotaCheckSpec struct {
	BackendType      string
	ModelID          string
	Effort           string
	Fast             bool
	Env              []string
	OnSessionCreated func(sessionID string) error
}

// QuotaCheckOutcome is the typed check result. Outcome is ready, quota,
// unsupported, failed or timeout; Cleaned is independent of it.
type QuotaCheckOutcome struct {
	Outcome     string
	ResetAt     *time.Time
	ResetSource string
	SessionID   string
	Cleaned     bool
}

type quotaCheckProbe struct {
	mu        sync.Mutex
	sessionID string
	reply     strings.Builder
	tool      bool
	titled    bool
	failure   *SessionFailureRecord
	rateLimit *time.Time
}

func (p *quotaCheckProbe) onNotification(method string, params json.RawMessage) {
	if method != "session/update" {
		return
	}
	sessionID, kind := peekSessionUpdate(params)
	p.mu.Lock()
	defer p.mu.Unlock()
	if sessionID != p.sessionID || p.sessionID == "" {
		return
	}
	if failure, ok := decodeSessionFailureRecord(params); ok {
		if p.failure == nil || failure.ID == p.failure.ID && failure.Revision > p.failure.Revision || failure.Quota && !p.failure.Quota {
			p.failure = &failure
		}
		return
	}
	switch kind {
	case "usage_update":
		if reset, ok := claudeRateLimitReset(params); ok && (p.rateLimit == nil || reset.After(*p.rateLimit)) {
			p.rateLimit = &reset
		}
	case "session_info_update":
		var su struct {
			Update struct {
				Title *string `json:"title"`
			} `json:"update"`
		}
		if json.Unmarshal(params, &su) == nil && su.Update.Title != nil && *su.Update.Title == quotaCheckName {
			p.titled = true
		}
	case "agent_message_chunk":
		var su struct {
			Update struct {
				Content struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"update"`
		}
		if json.Unmarshal(params, &su) == nil && su.Update.Content.Type == "text" && p.reply.Len() < maxQuotaCheckReply {
			p.reply.WriteString(boundRunes(su.Update.Content.Text, maxQuotaCheckReply-p.reply.Len()))
		}
	case "tool_call", "tool_call_update":
		p.tool = true
	}
}

// RunQuotaCheck performs one isolated check. It never touches a registered
// agent, transcript, preview, notification or owner settlement path.
func (c *ChatRuntime) RunQuotaCheck(ctx context.Context, spec QuotaCheckSpec) QuotaCheckOutcome {
	if spec.BackendType != "claude-acp" && spec.BackendType != "codex-acp" || spec.OnSessionCreated == nil {
		return QuotaCheckOutcome{Outcome: "unsupported", Cleaned: true}
	}
	ad, err := c.adapterFor(spec.BackendType)
	if err != nil {
		return QuotaCheckOutcome{Outcome: "unsupported", Cleaned: true}
	}
	cwd, err := os.MkdirTemp("", "chuck-quota-check-")
	if err != nil {
		return QuotaCheckOutcome{Outcome: "failed", Cleaned: true}
	}
	checkCtx, cancel := context.WithTimeout(ctx, QuotaCheckTimeout)
	defer cancel()
	probe := &quotaCheckProbe{}
	as, err := c.startQuotaCheckProcess(ad, spec, cwd, probe)
	if err != nil {
		cleaned := os.RemoveAll(cwd) == nil
		if errors.Is(err, errQuotaCheckUnsupported) {
			return QuotaCheckOutcome{Outcome: "unsupported", Cleaned: cleaned}
		}
		return QuotaCheckOutcome{Outcome: "failed", Cleaned: cleaned}
	}
	out := c.driveQuotaCheck(checkCtx, as, ad, spec, cwd, probe)
	out.Cleaned = c.disposeQuotaCheck(as, spec.BackendType, out.SessionID, cwd)
	if out.Outcome == "ready" && !out.Cleaned {
		out.Outcome = "failed"
	}
	return out
}

var errQuotaCheckUnsupported = errors.New("runtime: quota check isolation unsupported")

func (c *ChatRuntime) startQuotaCheckProcess(ad backend.BackendAdapter, spec QuotaCheckSpec, cwd string, probe *quotaCheckProbe) (*agentState, error) {
	cmd, err := c.spawnCmd(ad, LaunchSpec{BackendType: spec.BackendType, ModelID: spec.ModelID, Cwd: cwd, Env: spec.Env})
	if err != nil {
		return nil, err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	start := cmd.Start
	if spec.BackendType == "codex-acp" {
		start = func() error {
			env, err := withCodexCheckIsolation(cmd.Env)
			if err != nil {
				return err
			}
			cmd.Env = env
			return cmd.Start()
		}
		if home := envValue(cmd.Env, "CODEX_HOME"); home != "" {
			plain := start
			start = func() error { return config.WithRefreshedCodexProfile(home, plain) }
		}
	}
	if err := start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	actx, acancel := context.WithCancel(context.Background())
	as := &agentState{cmd: cmd, pgid: cmd.Process.Pid, stdin: stdin, ctx: actx, cancel: acancel, adapter: ad, stderrDone: make(chan struct{})}
	close(as.stderrDone)
	as.transport = NewTransport(stdin, probe.onNotification, func(req *IncomingRequest) {
		// No permission, elicitation, file or terminal authority is offered.
		if req.Method == "session/request_permission" {
			_ = req.Respond(map[string]any{"outcome": map[string]any{"outcome": "cancelled"}})
			probe.mu.Lock()
			probe.tool = true
			probe.mu.Unlock()
			return
		}
		_ = req.RespondError(-32601, "not supported by a Chuck quota check")
	})
	go func() { _ = as.transport.Run(stdout) }()
	return as, nil
}

func (c *ChatRuntime) driveQuotaCheck(ctx context.Context, as *agentState, ad backend.BackendAdapter, spec QuotaCheckSpec, cwd string, probe *quotaCheckProbe) QuotaCheckOutcome {
	timedOut := func(err error) QuotaCheckOutcome {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
			return QuotaCheckOutcome{Outcome: "timeout"}
		}
		return QuotaCheckOutcome{Outcome: "failed"}
	}
	initRes, err := as.transport.Call(ctx, "initialize", map[string]any{
		"protocolVersion": 1, "clientCapabilities": clientCapabilitiesFor(clientOffer{SessionFailure: true}),
	})
	if err != nil {
		return timedOut(err)
	}
	if checkACPVersion(initRes) != nil || !negotiateCapabilities(initRes, clientOffer{SessionFailure: true}).SessionFailure ||
		!advertisesSessionLifecycle(initRes, spec.BackendType) {
		return QuotaCheckOutcome{Outcome: "unsupported"}
	}
	params := map[string]any{"cwd": cwd, "mcpServers": []any{}}
	if spec.BackendType == "claude-acp" {
		options := map[string]any{"persistSession": false, "tools": []any{}, "strictMcpConfig": true}
		if spec.ModelID != "" {
			options["model"] = spec.ModelID
		}
		params["_meta"] = map[string]any{"claudeCode": map[string]any{"options": options}}
	}
	newRes, err := as.transport.Call(ctx, "session/new", params)
	if err != nil {
		return timedOut(err)
	}
	var sess struct {
		SessionID string `json:"sessionId"`
	}
	if json.Unmarshal(newRes, &sess) != nil || sess.SessionID == "" || len(sess.SessionID) > 256 {
		return QuotaCheckOutcome{Outcome: "failed"}
	}
	out := QuotaCheckOutcome{SessionID: sess.SessionID}
	if err := spec.OnSessionCreated(sess.SessionID); err != nil {
		out.Outcome = "failed"
		return out
	}
	probe.mu.Lock()
	probe.sessionID = sess.SessionID
	probe.mu.Unlock()
	launch := LaunchSpec{BackendType: spec.BackendType, ModelID: spec.ModelID, Effort: spec.Effort, Fast: spec.Fast}
	launch.Agent.Interface = "chat"
	applied, err := applySessionConfig(ctx, as.transport, ad, launch, sess.SessionID, decodeSessionConfigOptions(newRes))
	if err != nil || spec.Fast && !applied.Fast {
		// A refused model/effort/tier would test another quota (TS-04.R97).
		out.Outcome = "unsupported"
		return out
	}
	if spec.BackendType == "codex-acp" {
		// An explicit name suppresses Codex's automatic title-model turn.
		if _, err := as.transport.Call(ctx, "session/prompt", map[string]any{"sessionId": sess.SessionID,
			"prompt": []map[string]any{{"type": "text", "text": "/rename " + quotaCheckName}}}); err != nil {
			out.Outcome = timedOut(err).Outcome
			return out
		}
		// The name update is a notification that can trail the command's reply.
		var titled, tool bool
		for deadline := time.Now().Add(2 * time.Second); ; {
			probe.mu.Lock()
			titled, tool = probe.titled, probe.tool
			probe.mu.Unlock()
			if titled || tool || time.Now().After(deadline) || ctx.Err() != nil {
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
		probe.mu.Lock()
		probe.reply.Reset()
		probe.mu.Unlock()
		if !titled || tool {
			out.Outcome = "unsupported"
			return out
		}
	}
	res, err := as.transport.Call(ctx, "session/prompt", map[string]any{"sessionId": sess.SessionID,
		"prompt": []map[string]any{{"type": "text", "text": quotaCheckPrompt}}})
	if err != nil && ctx.Err() != nil {
		out.Outcome = "timeout"
		return out
	}
	resultFailure, hasResultFailure := decodeSessionFailureRecord(res)
	probe.mu.Lock()
	defer probe.mu.Unlock()
	failure := probe.failure
	if hasResultFailure && (failure == nil || resultFailure.ID == failure.ID && resultFailure.Revision >= failure.Revision || resultFailure.Quota && !failure.Quota) {
		failure = &resultFailure
	}
	if failure != nil && failure.Quota {
		text := failure.Text + "\n" + promptErrorText(err)
		reset, _ := resolveQuotaReset(spec.BackendType, text, probe.rateLimit, failure.ObservedAt, processZone(as.cmd.Env))
		out.Outcome, out.ResetAt = "quota", reset
		if reset != nil {
			out.ResetSource = "text"
			if spec.BackendType == "claude-acp" && probe.rateLimit != nil {
				out.ResetSource = "structured"
			}
		}
		return out
	}
	var plain acpPromptResult
	if err != nil || failure != nil || json.Unmarshal(res, &plain) != nil || plain.StopReason != "end_turn" ||
		probe.tool || strings.TrimSpace(probe.reply.String()) == "" {
		out.Outcome = "failed"
		return out
	}
	out.Outcome = "ready"
	return out
}

// advertisesSessionLifecycle verifies the exact disposal the pinned profile
// needs: Claude close (non-persistent), Codex delete (archive).
func advertisesSessionLifecycle(initRes json.RawMessage, backendType string) bool {
	var body struct {
		AgentCapabilities struct {
			SessionCapabilities map[string]json.RawMessage `json:"sessionCapabilities"`
		} `json:"agentCapabilities"`
	}
	if json.Unmarshal(initRes, &body) != nil {
		return false
	}
	need := "close"
	if backendType == "codex-acp" {
		need = "delete"
	}
	_, ok := body.AgentCapabilities.SessionCapabilities[need]
	return ok
}

// disposeQuotaCheck closes (Claude) or deletes/archives (Codex) the exact probe
// session, then tears down the process group and temporary cwd.
func (c *ChatRuntime) disposeQuotaCheck(as *agentState, backendType, sessionID, cwd string) bool {
	ok := true
	if sessionID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), QuotaCheckCleanupGrace)
		method := "session/close"
		if backendType == "codex-acp" {
			method = "session/delete"
		}
		if _, err := as.transport.Call(ctx, method, map[string]any{"sessionId": sessionID}); err != nil {
			ok = false
		}
		cancel()
	}
	as.shutdown()
	if err := os.RemoveAll(cwd); err != nil {
		ok = false
	}
	return ok
}

// CleanupQuotaCheckSession retries exact disposal of a recorded probe without
// loading or prompting it. Claude probes are non-persistent: once their owning
// process is gone nothing remains to dispose.
func (c *ChatRuntime) CleanupQuotaCheckSession(ctx context.Context, spec QuotaCheckSpec, sessionID string) error {
	if spec.BackendType == "claude-acp" {
		return nil
	}
	if spec.BackendType != "codex-acp" || sessionID == "" {
		return errQuotaCheckUnsupported
	}
	ad, err := c.adapterFor(spec.BackendType)
	if err != nil {
		return err
	}
	cwd, err := os.MkdirTemp("", "chuck-quota-cleanup-")
	if err != nil {
		return err
	}
	as, err := c.startQuotaCheckProcess(ad, spec, cwd, &quotaCheckProbe{})
	if err != nil {
		_ = os.RemoveAll(cwd)
		return err
	}
	callCtx, cancel := context.WithTimeout(ctx, QuotaCheckCleanupGrace)
	defer cancel()
	_, err = as.transport.Call(callCtx, "initialize", map[string]any{"protocolVersion": 1, "clientCapabilities": clientCapabilitiesFor(clientOffer{})})
	if err == nil {
		_, err = as.transport.Call(callCtx, "session/delete", map[string]any{"sessionId": sessionID})
	}
	as.shutdown()
	if rmErr := os.RemoveAll(cwd); err == nil {
		err = rmErr
	}
	return err
}

// withCodexCheckIsolation disables every MCP server known to the effective
// Codex configuration layers through the process CODEX_CONFIG overlay, without
// touching the shared profile or its authentication.
func withCodexCheckIsolation(env []string) ([]string, error) {
	home := envValue(env, "CODEX_HOME")
	if home == "" {
		if h := envValue(env, "HOME"); h != "" {
			home = filepath.Join(h, ".codex")
		}
	}
	if home == "" {
		return nil, errQuotaCheckUnsupported
	}
	overlay := map[string]any{}
	if raw := envValue(env, "CODEX_CONFIG"); raw != "" {
		if json.Unmarshal([]byte(raw), &overlay) != nil || overlay == nil {
			return nil, errQuotaCheckUnsupported
		}
	}
	names := map[string]bool{}
	if servers, ok := overlay["mcp_servers"].(map[string]any); ok {
		for name := range servers {
			names[name] = true
		}
	}
	for _, path := range []string{filepath.Join(home, "config.toml"), filepath.Join(home, "managed_config.toml"),
		"/etc/codex/config.toml", "/etc/codex/managed_config.toml"} {
		raw, err := os.ReadFile(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, errQuotaCheckUnsupported
		}
		var layer struct {
			MCPServers map[string]any `toml:"mcp_servers"`
		}
		if err := toml.Unmarshal(raw, &layer); err != nil {
			return nil, fmt.Errorf("%w: %s", errQuotaCheckUnsupported, filepath.Base(path))
		}
		for name := range layer.MCPServers {
			names[name] = true
		}
	}
	sorted := make([]string, 0, len(names))
	for name := range names {
		if name == "" || strings.ContainsAny(name, ".\"'") {
			return nil, errQuotaCheckUnsupported
		}
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		overlay["mcp_servers."+name+".enabled"] = false
	}
	if len(sorted) == 0 {
		return env, nil
	}
	encoded, err := json.Marshal(overlay)
	if err != nil {
		return nil, err
	}
	return append(stripEnv(env, "CODEX_CONFIG"), "CODEX_CONFIG="+string(encoded)), nil
}
