package runtime

import "encoding/json"

const (
	maxSessionFailureBytes = 16 * 1024
	maxSessionFailureID    = 256
	maxSessionFailureText  = 2048
)

// QuotaEvidence is the bounded, typed evidence attached to a provider turn end.
// It deliberately carries no reset time: neither pinned producer gives a
// verified reset timestamp in this failure surface.
type QuotaEvidence struct {
	FailureID string `json:"failure_id"`
	Revision  int64  `json:"revision"`
}

// SessionFailureRecord is the safe correlation subset of an AIR failure. A
// later revision with the same ID can supersede an earlier quota failure.
type SessionFailureRecord struct {
	ID       string
	Revision int64
	Quota    bool
}

type airSessionFailurePayload struct {
	ID       string   `json:"id"`
	Revision int64    `json:"revision"`
	Category string   `json:"category"`
	Severity string   `json:"severity"`
	Title    string   `json:"title"`
	Details  *string  `json:"details"`
	Reason   *string  `json:"reason"`
	Actions  []string `json:"actions"`
}

// decodeSessionFailureRecord recognizes either a prompt result's _meta or the
// session_info_update notification params. It accepts only the complete AIR
// envelope emitted by the pinned Claude and Codex adapters. Quota is the shared
// terminal policy (error + limit + empty actions); a valid later record with the
// same ID can supersede an earlier revision.
func decodeSessionFailureRecord(raw json.RawMessage) (SessionFailureRecord, bool) {
	if len(raw) == 0 || len(raw) > maxSessionFailureBytes {
		return SessionFailureRecord{}, false
	}
	var envelope struct {
		Meta   json.RawMessage `json:"_meta"`
		Update struct {
			SessionUpdate string          `json:"sessionUpdate"`
			Meta          json.RawMessage `json:"_meta"`
		} `json:"update"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return SessionFailureRecord{}, false
	}
	meta := envelope.Meta
	if envelope.Update.SessionUpdate != "" {
		if envelope.Update.SessionUpdate != "session_info_update" {
			return SessionFailureRecord{}, false
		}
		meta = envelope.Update.Meta
	}
	var extensions struct {
		JetBrains struct {
			Air struct {
				Version        int                       `json:"version"`
				SessionFailure *airSessionFailurePayload `json:"sessionFailure"`
			} `json:"air"`
		} `json:"jetbrains"`
	}
	if len(meta) == 0 || json.Unmarshal(meta, &extensions) != nil || extensions.JetBrains.Air.Version < airVersion {
		return SessionFailureRecord{}, false
	}
	f := extensions.JetBrains.Air.SessionFailure
	if f == nil || f.ID == "" || len([]rune(f.ID)) > maxSessionFailureID || f.Revision < 1 ||
		!validSessionFailureCategory(f.Category) || (f.Severity != "error" && f.Severity != "warning") || f.Title == "" || f.Actions == nil {
		return SessionFailureRecord{}, false
	}
	if (f.Details != nil && len([]rune(*f.Details)) > maxSessionFailureText) || (f.Reason != nil && len([]rune(*f.Reason)) > maxSessionFailureText) || len([]rune(f.Title)) > maxSessionFailureText {
		return SessionFailureRecord{}, false
	}
	for _, action := range f.Actions {
		if action != "retry" && action != "login" && action != "new_session" {
			return SessionFailureRecord{}, false
		}
	}
	return SessionFailureRecord{ID: f.ID, Revision: f.Revision, Quota: f.Category == "limit" && f.Severity == "error" && len(f.Actions) == 0}, true
}

func validSessionFailureCategory(category string) bool {
	switch category {
	case "connection", "access", "limit", "request", "service", "unknown":
		return true
	default:
		return false
	}
}

func decodeSessionFailure(raw json.RawMessage) (QuotaEvidence, bool) {
	record, ok := decodeSessionFailureRecord(raw)
	if !ok || !record.Quota {
		return QuotaEvidence{}, false
	}
	return QuotaEvidence{FailureID: record.ID, Revision: record.Revision}, true
}
