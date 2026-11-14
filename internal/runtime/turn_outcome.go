package runtime

import (
	"encoding/json"
	"fmt"
)

// TurnOutcome is the terminal interpretation shared by runtime and replay.
type TurnOutcome uint8

const (
	TurnUnknown TurnOutcome = iota
	TurnCompleted
	TurnCancelled
	TurnFailed
	TurnQuotaBlocked
	TurnProviderTruncated
	TurnProcessEnded
	// These provider reasons have distinct existing wire labels. The decoder
	// groups them with truncation and failure for external consumers.
	turnRequestLimit
	turnRefused
)

func (o TurnOutcome) confirmsInlineMail() bool {
	return o == TurnCompleted || o == TurnProviderTruncated
}

func (o TurnOutcome) storageValue() string {
	switch o {
	case TurnCompleted:
		return "completed"
	case TurnCancelled:
		return "cancelled"
	case TurnFailed:
		return "failed"
	case TurnQuotaBlocked:
		return "quota_blocked"
	case TurnProviderTruncated, turnRequestLimit:
		return "provider_truncated"
	case TurnProcessEnded:
		return "process_ended"
	case turnRefused:
		return "refused"
	default:
		return "unknown"
	}
}

func outcomeForReason(reason string) TurnOutcome {
	switch reason {
	case "end_turn":
		return TurnCompleted
	case "cancelled":
		return TurnCancelled
	case "max_tokens":
		return TurnProviderTruncated
	case "max_turn_requests":
		return turnRequestLimit
	case "refusal":
		return turnRefused
	case "error":
		return TurnFailed
	default:
		return TurnUnknown
	}
}

func promptOutcome(raw json.RawMessage) TurnOutcome {
	var result acpPromptResult
	if json.Unmarshal(raw, &result) != nil {
		return TurnUnknown
	}
	return outcomeForReason(result.StopReason)
}

func resolveTurnOutcome(provider TurnOutcome, cancelled, processEnded, failed bool, quota *QuotaEvidence) TurnOutcome {
	switch {
	case cancelled || provider == TurnCancelled:
		return TurnCancelled
	case processEnded:
		return TurnProcessEnded
	case quota != nil:
		return TurnQuotaBlocked
	case failed:
		return TurnFailed
	default:
		return provider
	}
}

func (o TurnOutcome) eventData(reading contextReading, quota *QuotaEvidence) TurnEndData {
	data := TurnEndData{ContextPct: reading.pct, ContextCounts: reading.counts, Quota: quota}
	switch o {
	case TurnCompleted:
		data.StopReason = "end_turn"
	case TurnCancelled:
		data.StopReason = "cancelled"
	case TurnProviderTruncated:
		data.StopReason = "max_tokens"
	case turnRequestLimit:
		data.StopReason = "max_turn_requests"
	case turnRefused:
		data.StopReason = "refusal"
	default:
		data.StopReason = "error"
	}
	data.ProcessEnded = o == TurnProcessEnded
	return data
}

// DecodeTurnEnd interprets a live or retained event while preserving its original
// payload for transcript projection and context accounting.
func DecodeTurnEnd(ev Event) (TurnOutcome, TurnEndData, error) {
	var data TurnEndData
	if ev.Type != EvTurnEnd {
		return TurnUnknown, data, fmt.Errorf("expected %s event, got %s", EvTurnEnd, ev.Type)
	}
	if err := json.Unmarshal(ev.Data, &data); err != nil {
		return TurnUnknown, data, err
	}
	outcome := outcomeForReason(data.StopReason)
	switch {
	case outcome == TurnCancelled:
	case data.ProcessEnded && outcome == TurnFailed:
		outcome = TurnProcessEnded
	case data.Quota != nil:
		outcome = TurnQuotaBlocked
	case outcome == turnRequestLimit:
		outcome = TurnProviderTruncated
	case outcome == turnRefused:
		outcome = TurnFailed
	}
	return outcome, data, nil
}
