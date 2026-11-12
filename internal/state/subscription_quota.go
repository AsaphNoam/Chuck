package state

// SubscriptionQuota is one backend quota scope's normalized provider snapshot as
// every agent of that scope projects it (FS-09.R80/R83, TS-03.R85). It is held
// in memory only and never persisted (TS-04.R109). Missing fields are unknown,
// never zero. The scope key and routing value are never serialized.
type SubscriptionQuota struct {
	// Provider is "claude" or "codex".
	Provider string `json:"provider"`
	// Availability is "complete" (both windows known), "partial" or
	// "unavailable".
	Availability string                   `json:"availability"`
	Windows      SubscriptionQuotaWindows `json:"windows"`
	// CheckedAt is the last completed read of this scope (RFC 3339 UTC), absent
	// before the first one. It is not advanced by a failed read.
	CheckedAt string `json:"checked_at,omitempty"`
	// ReasonCode explains unavailable or stale values: unsupported_adapter,
	// no_subscription, read_failed or read_timeout. A failed read keeps the
	// previous windows, so a reason beside known windows marks them stale.
	ReasonCode string `json:"reason_code,omitempty"`
}

// SubscriptionQuotaWindows holds the only two windows ever exposed.
type SubscriptionQuotaWindows struct {
	FiveHour SubscriptionQuotaWindow `json:"five_hour"`
	Weekly   SubscriptionQuotaWindow `json:"weekly"`
}

// SubscriptionQuotaWindow is one window; each field is individually optional.
type SubscriptionQuotaWindow struct {
	UsedPercent *float64 `json:"used_percent,omitempty"`
	ResetAt     string   `json:"reset_at,omitempty"`
	ObservedAt  string   `json:"observed_at,omitempty"`
}

// Subscription quota reason codes (TS-03.R85).
const (
	QuotaReasonUnsupportedAdapter = "unsupported_adapter"
	QuotaReasonNoSubscription     = "no_subscription"
	QuotaReasonReadFailed         = "read_failed"
	QuotaReasonReadTimeout        = "read_timeout"
)
