package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

const quotaSettlementError = "Quota recovery state could not be saved. Retry after restoring local storage."

func (c *ChatRuntime) settleTurnQuota(as *agentState, turnID string, outcome TurnOutcome, quota *QuotaEvidence) error {
	c.mu.Lock()
	settle := c.quotaSettlement
	c.mu.Unlock()
	if settle == nil {
		return nil
	}
	return settle(as.agentID, as.generation, turnID, quota, outcome == TurnCompleted)
}

// claudeUsageLimitPrefixes are pinned claude-agent-sdk 0.3.286's
// USAGE_LIMIT_ERROR_PREFIXES: the openings of its synthetic usage-limit errors.
var claudeUsageLimitPrefixes = []string{"You've hit your", "You've reached your", "You're out of usage credits",
	"Your org is out of usage · add funds to continue", "Your org is out of usage · contact your admin",
	"Your seat type doesn't include usage credits", "Your seat type doesn't include usage",
	"Your usage allocation has been disabled by your admin", "Your group's usage limit is set to $0",
	"Fable 5 requires usage credits", "You're out of extra usage", "Your seat type doesn't include extra usage"}

// claudeNonQuotaErrorKinds are the pinned Claude adapter's recognized
// non-quota `errorKind` values; matching text cannot override them. A
// `rate_limit` refusal is absent: the native CLI categorizes its own
// usage-limit refusal that way, so its grammar decides (TS-04.R105).
var claudeNonQuotaErrorKinds = map[string]bool{"authentication_failed": true, "oauth_org_not_allowed": true,
	"verification_required": true, "cloud_credential_error": true, "overloaded": true,
	"invalid_request": true, "model_not_found": true, "max_output_tokens": true, "server_error": true,
	"incomplete_tool_call": true, "worker_shutdown": true, "no_result": true, "transport_lost": true}

// rpcQuotaRefusal classifies a terminal session/prompt RPC error that arrived
// without AIR quota metadata (TS-04.R105). Only verified provider carriers
// count: Claude's `data.errorKind` billing_error/account_on_hold, or, when the
// category is absent, unrecognized or rate_limit, the SDK's usage-limit
// grammar opening the adapter's `Internal error: ` detail (plain throttling
// reads `API Error: …` and stays excluded); Codex's `data.codexErrorInfo`
// usageLimitExceeded. Oversized or malformed carriers stay ordinary errors.
func rpcQuotaRefusal(backendType string, err error) bool {
	var rpcErr *rpcError
	if !errors.As(err, &rpcErr) || rpcErr.Code != -32603 || len(rpcErr.Data) > maxSessionFailureBytes ||
		len(rpcErr.Message) > maxSessionFailureBytes || utf8.RuneCountInString(rpcErr.Message) > maxSessionFailureText {
		return false
	}
	var data struct {
		ErrorKind      string `json:"errorKind"`
		CodexErrorInfo any    `json:"codexErrorInfo"`
	}
	if len(rpcErr.Data) > 0 && json.Unmarshal(rpcErr.Data, &data) != nil {
		return false
	}
	switch backendType {
	case "claude-acp":
		switch {
		case data.ErrorKind == "billing_error" || data.ErrorKind == "account_on_hold":
			return true
		case claudeNonQuotaErrorKinds[data.ErrorKind]:
			return false
		}
		detail, ok := strings.CutPrefix(rpcErr.Message, "Internal error: ")
		if !ok {
			return false
		}
		for _, prefix := range claudeUsageLimitPrefixes {
			if strings.HasPrefix(detail, prefix) {
				return true
			}
		}
	case "codex-acp":
		return data.CodexErrorInfo == "usageLimitExceeded"
	}
	return false
}

// promptErrorArrival is when the transport received a session/prompt RPC
// error. Reset text is relative to that instant, never to later settlement
// (TS-10.R62); errors without a transport stamp fall back to now.
func promptErrorArrival(err error) time.Time {
	var rpcErr *rpcError
	if errors.As(err, &rpcErr) && !rpcErr.arrivedAt.IsZero() {
		return rpcErr.arrivedAt
	}
	return time.Now().UTC()
}

// promptErrorText is the bounded reset-evidence text of a session/prompt RPC
// error: its message and a string data.message (legacy non-AIR carriers).
func promptErrorText(err error) string {
	var rpcErr *rpcError
	if !errors.As(err, &rpcErr) {
		return ""
	}
	text := boundRunes(rpcErr.Message, maxSessionFailureText)
	var data struct {
		Message string `json:"message"`
	}
	if len(rpcErr.Data) <= maxSessionFailureBytes && json.Unmarshal(rpcErr.Data, &data) == nil && data.Message != "" {
		text += "\n" + boundRunes(data.Message, maxSessionFailureText)
	}
	return text
}

func boundRunes(s string, n int) string {
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

// SetQuotaSettlement commits quota state before owner settlement and successor
// admission. The server resolves ownership; the runtime owns the exact turn.
func (c *ChatRuntime) SetQuotaSettlement(fn func(string, string, string, *QuotaEvidence, bool) error) {
	c.mu.Lock()
	c.quotaSettlement = fn
	c.mu.Unlock()
}

func (r *Registry) SetQuotaSettlement(fn func(string, string, string, *QuotaEvidence, bool) error) {
	if r != nil && r.chat != nil {
		r.chat.SetQuotaSettlement(fn)
	}
}

// RunQuotaCheck runs one isolated check on the chat runtime (TS-04.R97).
func (r *Registry) RunQuotaCheck(ctx context.Context, spec QuotaCheckSpec) QuotaCheckOutcome {
	if r == nil || r.chat == nil {
		return QuotaCheckOutcome{Outcome: "unsupported", Cleaned: true}
	}
	return r.chat.RunQuotaCheck(ctx, spec)
}

// CleanupQuotaCheckSession retries exact disposal of a recorded probe.
func (r *Registry) CleanupQuotaCheckSession(ctx context.Context, spec QuotaCheckSpec, sessionID string) error {
	if r == nil || r.chat == nil {
		return errQuotaCheckUnsupported
	}
	return r.chat.CleanupQuotaCheckSession(ctx, spec, sessionID)
}

// SetQuotaManualBind binds ordinary prompt turns to the current active quota
// episode before provider work begins.
func (c *ChatRuntime) SetQuotaManualBind(fn func(string, string, string) error) {
	c.mu.Lock()
	c.quotaManualBind = fn
	c.mu.Unlock()
}

func (r *Registry) SetQuotaManualBind(fn func(string, string, string) error) {
	if r != nil && r.chat != nil {
		r.chat.SetQuotaManualBind(fn)
	}
}

// ClearQuotaBlock releases the in-memory hold after an owner control cancels
// the durable quota episode.
func (c *ChatRuntime) ClearQuotaBlock(agentID, generation, turnID string) {
	as, err := c.lookup(agentID)
	if err != nil {
		return
	}
	as.mu.Lock()
	if !as.quotaBlocked || as.quotaBlockedGeneration != generation || as.quotaBlockedTurnID != turnID {
		as.mu.Unlock()
		return
	}
	as.quotaBlocked = false
	as.quotaBlockedGeneration = ""
	as.quotaBlockedTurnID = ""
	as.mu.Unlock()
}

func (r *Registry) ClearQuotaBlock(agentID, generation, turnID string) {
	if r != nil && r.chat != nil {
		r.chat.ClearQuotaBlock(agentID, generation, turnID)
	}
}

// recoveredByTerminalSuccess reports a terminal end_turn without failure
// metadata. Both pinned providers clear an active streaming failure internally
// after a real model success; that recovery has no clear notification on the
// wire, so this result is the confirmation (TS-04.R86). Ordinary turns and
// isolated checks share it.
func recoveredByTerminalSuccess(provider TurnOutcome, hasResultFailure bool) bool {
	return !hasResultFailure && provider == TurnCompleted
}

// completePromptTurn is shared by user prompts and every owned activation.
// Streaming failure metadata becomes terminal evidence only at completion.
func (c *ChatRuntime) completePromptTurn(as *agentState, turnID, deliveryKey string, result json.RawMessage, promptErr error) {
	provider := promptOutcome(result)
	resultFailure, hasResultFailure := decodeSessionFailureRecord(result)
	as.mu.Lock()
	if as.stopped || as.stopping || !as.turnActive || as.execTurnID != turnID || as.promptSettlement != nil {
		as.mu.Unlock()
		return
	}
	as.promptSettlement = make(chan struct{})
	cancelled := provider == TurnCancelled ||
		as.cancelledTurnSeq != 0 && turnID == fmt.Sprintf("t_%012d", as.cancelledTurnSeq)
	td := TurnEndData{}
	if hasResultFailure &&
		(as.turnFailure == nil ||
			resultFailure.ID == as.turnFailure.ID && resultFailure.Revision >= as.turnFailure.Revision ||
			resultFailure.ID != as.turnFailure.ID && resultFailure.Quota && !as.turnFailure.Quota) {
		if as.turnFailure != nil && as.turnFailure.ID == resultFailure.ID && as.turnFailure.Revision == resultFailure.Revision {
			resultFailure.ObservedAt = as.turnFailure.ObservedAt // first frame of this revision
		}
		as.turnFailure = &resultFailure
		as.turnQuota = nil
		if resultFailure.Quota {
			as.turnQuota = &QuotaEvidence{FailureID: resultFailure.ID, Revision: resultFailure.Revision}
		}
	}
	if !cancelled && promptErr == nil && recoveredByTerminalSuccess(provider, hasResultFailure) {
		as.turnFailure = nil
		as.turnQuota = nil
	}
	// TS-04.R86/FS-01.R43: an accepted Cancel wins over late provider quota
	// evidence, so the cancelled work is never re-armed for continuation.
	if as.caps.SessionFailure && as.turnQuota != nil && !cancelled {
		q := *as.turnQuota
		td.Quota = &q
	}
	backendType := ""
	if as.adapter != nil {
		backendType = as.adapter.Type()
	}
	// TS-04.R105: without AIR evidence for this turn, a verified quota carrier
	// on the terminal RPC is the refusal. A current AIR record (including a
	// newer non-quota or retryable revision) and cancellation keep precedence.
	if td.Quota == nil && as.turnFailure == nil && !cancelled && rpcQuotaRefusal(backendType, promptErr) {
		td.Quota = &QuotaEvidence{FailureID: "rpc:" + turnID, Revision: 1}
	}
	if td.Quota != nil {
		text, observed := promptErrorText(promptErr), promptErrorArrival(promptErr)
		if as.turnFailure != nil {
			text, observed = as.turnFailure.Text+"\n"+text, as.turnFailure.ObservedAt
		}
		td.Quota.ResetAt, td.Quota.Intervention = resolveQuotaReset(backendType, text, as.turnRateLimit, observed, as.processZone)
		if td.Quota.ResetAt != nil {
			td.Quota.ResetSource = "text"
			if backendType == "claude-acp" && as.turnRateLimit.reset != nil {
				td.Quota.ResetSource = "structured"
			}
		}
		as.quotaBlocked = true
		as.quotaBlockedGeneration = as.generation
		as.quotaBlockedTurnID = turnID
	}
	// A prompt result carrying a current error-severity failure is a terminal
	// provider failure (e.g. exhausted 429 retries) despite its end_turn: it is
	// neither a successful completion nor an inline-mail receipt (FS-03.R8).
	var terminal, failed *SessionFailureRecord
	if as.caps.SessionFailure && as.turnFailure != nil && (td.Quota != nil ||
		hasResultFailure && as.turnFailure.Severity == "error" && !cancelled) {
		f := *as.turnFailure
		terminal = &f
		if td.Quota == nil {
			failed = &f
		}
	}
	outcome := resolveTurnOutcome(provider, cancelled, false, promptErr != nil || failed != nil || provider == TurnUnknown, td.Quota)
	as.mu.Unlock()
	// TS-04.R111: with no structured or text reset, one snapshot read on this
	// still-live process may supply the latest exhausted applicable reset. It
	// stands in for this turn end's ordinary refresh, so the turn reads once.
	if td.Quota != nil && td.Quota.ResetAt == nil && !td.Quota.Intervention && as.quotaScope != "" {
		if reset := c.quotaSnapshotReset(as.quotaScope, as, as.quotaSelection.ModelID, time.Now()); reset != nil {
			td.Quota.ResetAt, td.Quota.ResetSource = reset, "snapshot"
		}
		as.mu.Lock()
		as.quotaReadInTurn = true
		as.mu.Unlock()
	}
	if terminal != nil {
		// TS-04.R85: safe correlation only, never provider text.
		slog.Warn("runtime: provider turn failed", "agent", as.agentID, "backend", backendType,
			"generation", as.generation, "turn", turnID, "operation", "session/prompt",
			"failure_id", terminal.ID, "revision", terminal.Revision,
			"category", terminal.Category, "severity", terminal.Severity, "quota", terminal.Quota)
	}
	successful := outcome == TurnCompleted
	settlementErr := c.settleTurnQuota(as, turnID, outcome, td.Quota)
	if settlementErr != nil {
		as.mu.Lock()
		as.quotaBlocked = true
		as.quotaBlockedGeneration = as.generation
		as.quotaBlockedTurnID = turnID
		as.mu.Unlock()
		c.emit(as, EvError, ErrorData{Scope: "internal", Message: quotaSettlementError, Fatal: false})
	}
	c.settleInlineMail(as, deliveryKey, outcome)
	if failed != nil {
		c.emit(as, EvError, ErrorData{Scope: "protocol", Message: failed.Text, Fatal: false})
	}
	if successful && settlementErr == nil {
		as.mu.Lock()
		as.quotaBlocked = false
		as.quotaBlockedGeneration = ""
		as.quotaBlockedTurnID = ""
		as.mu.Unlock()
	}
	nextText, nextTurnID := c.finishTurn(as, turnID, outcome, outcome.eventData(as.settleTurnContext(), td.Quota))
	c.runReservedHeld(as, nextText, nextTurnID)
}
