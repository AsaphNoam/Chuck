package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

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

// completePromptTurn is shared by user prompts and every owned activation.
// Streaming failure metadata becomes terminal evidence only at completion.
func (c *ChatRuntime) completePromptTurn(as *agentState, turnID, deliveryKey string, result json.RawMessage, promptErr error) {
	as.mu.Lock()
	current := !as.stopped && as.turnActive && as.execTurnID == turnID
	as.mu.Unlock()
	if !current {
		return
	}
	td, hasPct := mapPromptResult(result)
	var plain acpPromptResult
	if json.Unmarshal(result, &plain) == nil {
		td.StopReason = defaultStr(plain.StopReason, "end_turn")
	}
	td.Quota = nil
	resultFailure, hasResultFailure := decodeSessionFailureRecord(result)
	as.mu.Lock()
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
	// Both pinned providers clear an active quota failure internally after a
	// real model success. That recovery has no clear notification on the wire;
	// a successful terminal result without failure metadata is the confirmation.
	if !hasResultFailure && td.StopReason == "end_turn" && validInlineReceipt(result) {
		as.turnFailure = nil
		as.turnQuota = nil
	}
	if as.caps.SessionFailure && as.turnQuota != nil && td.Quota == nil {
		q := *as.turnQuota
		td.Quota = &q
	}
	if !as.caps.SessionFailure {
		td.Quota = nil
	}
	if td.Quota != nil {
		text, observed := promptErrorText(promptErr), time.Now().UTC()
		if as.turnFailure != nil {
			text, observed = as.turnFailure.Text+"\n"+text, as.turnFailure.ObservedAt
		}
		backendType := ""
		if as.adapter != nil {
			backendType = as.adapter.Type()
		}
		td.Quota.ResetAt, td.Quota.Intervention = resolveQuotaReset(backendType, text, as.turnRateLimitReset, observed, as.processZone)
		if td.Quota.ResetAt != nil {
			td.Quota.ResetSource = "text"
			if backendType == "claude-acp" && as.turnRateLimitReset != nil {
				td.Quota.ResetSource = "structured"
			}
		}
		td.StopReason = "error"
		as.quotaBlocked = true
		as.quotaBlockedGeneration = as.generation
		as.quotaBlockedTurnID = turnID
	}
	as.mu.Unlock()
	c.mu.Lock()
	settle := c.quotaSettlement
	c.mu.Unlock()
	successful := td.StopReason == "end_turn" && validInlineReceipt(result)
	settled := true
	if settle != nil {
		if err := settle(as.agentID, as.generation, turnID, td.Quota, successful); err != nil {
			settled = false
			as.mu.Lock()
			as.quotaBlocked = true
			as.quotaBlockedGeneration = as.generation
			as.quotaBlockedTurnID = turnID
			as.mu.Unlock()
			c.emit(as, EvError, ErrorData{Scope: "internal", Message: "Quota recovery state could not be saved. Retry after restoring local storage.", Fatal: false})
		}
	}
	if td.Quota == nil {
		c.settleInlineMail(as, deliveryKey, result)
	} else {
		c.clearInlineMail(as.agentID, deliveryKey)
	}
	if successful && settled {
		as.mu.Lock()
		as.quotaBlocked = false
		as.quotaBlockedGeneration = ""
		as.quotaBlockedTurnID = ""
		as.mu.Unlock()
	}
	as.settleTurnContext(&td, hasPct)
	nextText, nextTurnID := c.finishTurn(as, turnID, td)
	c.runReservedHeld(as, nextText, nextTurnID)
}
