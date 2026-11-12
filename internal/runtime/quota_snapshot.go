package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// quotaReadMethod is the packaged adapters' negotiated quota passthrough,
// advertised at initialize as _meta.chuck.quotaRead.version == 1 (TS-04.R109).
const quotaReadMethod = "_chuck/quota/read"

// defaultQuotaReadTimeout bounds one read; there is no retry (TS-04.R109).
const defaultQuotaReadTimeout = 10 * time.Second

// QuotaScopeResolver names a stopped agent's quota scope from durable records:
// its configured backend, backend type and launch routing. ok is false when
// the scope cannot be resolved; the agent then projects no snapshot.
type QuotaScopeResolver func(agentID string) (backendID, backendType, routing string, ok bool)

// quotaStore is the one in-memory snapshot per backend quota scope
// (FS-09.R83). It survives no server restart and is never persisted.
type quotaStore struct {
	mu       sync.Mutex
	scopes   map[string]*quotaScope
	agents   map[string]string // agentID -> scope key, kept after stop
	resolver QuotaScopeResolver
	timeout  time.Duration
}

type quotaScope struct {
	snap state.SubscriptionQuota
	// call is the read in flight, joined by concurrent triggers; last is the
	// latest successful read, kept raw for recovery evidence (TS-04.R111).
	call    *quotaCall
	last    *quotaCall
	resetAt time.Time // armed post-reset read, zero when none
	timer   *time.Timer
}

func newQuotaStore() *quotaStore {
	return &quotaStore{scopes: map[string]*quotaScope{}, agents: map[string]string{},
		timeout: defaultQuotaReadTimeout}
}

// quotaProvider maps a backend type to the projected provider name, "" for a
// backend without a subscription view.
func quotaProvider(backendType string) string {
	switch backendType {
	case "claude-acp":
		return "claude"
	case "codex-acp":
		return "codex"
	}
	return ""
}

// quotaScopeKey is the backend id plus R97's frozen routing value; an
// unsupported routing is its own unshared scope (TS-04.R109).
func quotaScopeKey(agentID, backendID, routing string) string {
	if routing == quotaRoutingUnsupported {
		return backendID + "\x00" + routing + "\x00" + agentID
	}
	return backendID + "\x00" + routing
}

// SetQuotaScopeResolver installs the stopped-agent scope lookup. Set before serving.
func (c *ChatRuntime) SetQuotaScopeResolver(fn QuotaScopeResolver) {
	c.quota.mu.Lock()
	c.quota.resolver = fn
	c.quota.mu.Unlock()
}

// bindQuotaScope records a launching agent's scope and returns its key, or ""
// when its backend has no subscription view.
func (c *ChatRuntime) bindQuotaScope(agentID, backendID, backendType, routing string) string {
	provider := quotaProvider(backendType)
	if provider == "" {
		return ""
	}
	key := quotaScopeKey(agentID, backendID, routing)
	q := c.quota
	q.mu.Lock()
	defer q.mu.Unlock()
	q.agents[agentID] = key
	if q.scopes[key] == nil {
		q.scopes[key] = &quotaScope{snap: state.SubscriptionQuota{Provider: provider, Availability: "unavailable"}}
	}
	return key
}

// SubscriptionQuota returns agentID's scope snapshot for projection, or nil
// when its backend has no subscription view. It never calls the provider.
func (c *ChatRuntime) SubscriptionQuota(agentID string) *state.SubscriptionQuota {
	q := c.quota
	q.mu.Lock()
	key, ok := q.agents[agentID]
	resolver := q.resolver
	q.mu.Unlock()
	if !ok {
		if resolver == nil {
			return nil
		}
		backendID, backendType, routing, resolved := resolver(agentID)
		if !resolved {
			return nil
		}
		key = c.bindQuotaScope(agentID, backendID, backendType, routing)
	}
	if key == "" {
		return nil
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	sc := q.scopes[key]
	if sc == nil {
		return nil
	}
	out := sc.snap
	return &out
}

// decodeQuotaReadSupport reads _meta.chuck.quotaRead.version == 1; anything
// else is unsupported, never an error (TS-04.R109).
func decodeQuotaReadSupport(initRes json.RawMessage) bool {
	var body struct {
		Meta struct {
			Chuck struct {
				QuotaRead struct {
					Version int `json:"version"`
				} `json:"quotaRead"`
			} `json:"chuck"`
		} `json:"_meta"`
	}
	if json.Unmarshal(initRes, &body) != nil {
		return false
	}
	return body.Meta.Chuck.QuotaRead.Version == 1
}

// quotaSessionOpened runs after a session open completes: a non-advertising
// adapter marks a never-read scope unsupported; an advertising one reads.
func (c *ChatRuntime) quotaSessionOpened(as *agentState) {
	if as.quotaScope == "" {
		return
	}
	if !as.quotaRead {
		q := c.quota
		q.mu.Lock()
		if sc := q.scopes[as.quotaScope]; sc != nil && sc.snap.CheckedAt == "" && sc.snap.ReasonCode == "" {
			sc.snap.ReasonCode = state.QuotaReasonUnsupportedAdapter
		}
		q.mu.Unlock()
		c.touchState(as.agentID)
		return
	}
	c.refreshQuota(as.quotaScope)
}

// refreshQuota issues one read for key through any live advertising session of
// the scope. A trigger while a read is in flight joins it; with no live
// session nothing is started and the snapshot ages (FS-09.R83).
func (c *ChatRuntime) refreshQuota(key string) {
	if key == "" {
		return
	}
	if reader := c.quotaReader(key); reader != nil {
		c.startQuotaCall(key, reader)
	}
}

// quotaCall is one bounded read of a scope; done closes when it settles.
type quotaCall struct {
	done chan struct{}
	raw  json.RawMessage
	err  error
	at   time.Time
}

// startQuotaCall returns the scope's in-flight read, or starts one through
// reader. Nil when the scope is unknown.
func (c *ChatRuntime) startQuotaCall(key string, reader *agentState) *quotaCall {
	q := c.quota
	q.mu.Lock()
	defer q.mu.Unlock()
	sc := q.scopes[key]
	if sc == nil {
		return nil
	}
	if sc.call != nil {
		return sc.call
	}
	call := &quotaCall{done: make(chan struct{})}
	sc.call = call
	go c.readQuota(key, sc.snap.Provider, reader, q.timeout, call)
	return call
}

// quotaReader picks a live, advertising session of the scope.
func (c *ChatRuntime) quotaReader(key string) *agentState {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, as := range c.agents {
		if as.quotaScope == key && as.quotaRead && as.transport != nil {
			return as
		}
	}
	return nil
}

func (c *ChatRuntime) readQuota(key, provider string, as *agentState, timeout time.Duration, call *quotaCall) {
	as.mu.Lock()
	sessionID := as.sessionID
	as.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	res, err := as.transport.Call(ctx, quotaReadMethod, map[string]any{"sessionId": sessionID})
	cancel()
	now := time.Now().UTC()
	call.raw, call.err, call.at = res, err, now
	defer close(call.done)

	q := c.quota
	q.mu.Lock()
	sc := q.scopes[key]
	sc.call = nil
	var snap state.SubscriptionQuota
	ok := false
	if err == nil {
		snap, ok = normalizeQuotaRead(provider, res, now)
	}
	if !ok {
		// A failed or undecodable read keeps the previous observation; the
		// reason marks it stale (FS-09.R83).
		sc.snap.ReasonCode = state.QuotaReasonReadFailed
		if errors.Is(err, context.DeadlineExceeded) {
			sc.snap.ReasonCode = state.QuotaReasonReadTimeout
		}
	} else {
		sc.last = call
		sc.snap = snap
		c.armQuotaResetLocked(key, sc, now)
	}
	var agents []string
	for agentID, k := range q.agents {
		if k == key {
			agents = append(agents, agentID)
		}
	}
	q.mu.Unlock()
	for _, agentID := range agents {
		c.touchState(agentID)
	}
}

// armQuotaResetLocked schedules one read at the earliest future displayed
// reset plus the recovery buffer (FS-09.R83). Caller holds q.mu.
func (c *ChatRuntime) armQuotaResetLocked(key string, sc *quotaScope, now time.Time) {
	var next time.Time
	for _, w := range []state.SubscriptionQuotaWindow{sc.snap.Windows.FiveHour, sc.snap.Windows.Weekly} {
		t, err := time.Parse(time.RFC3339, w.ResetAt)
		if err != nil || !t.After(now) {
			continue
		}
		if next.IsZero() || t.Before(next) {
			next = t
		}
	}
	if next.IsZero() || next.Equal(sc.resetAt) {
		return
	}
	if sc.timer != nil {
		sc.timer.Stop()
	}
	sc.resetAt = next
	sc.timer = time.AfterFunc(next.Add(state.QuotaResetBuffer).Sub(now), func() {
		c.quota.mu.Lock()
		if s := c.quota.scopes[key]; s != nil && s.resetAt.Equal(next) {
			s.resetAt, s.timer = time.Time{}, nil
		}
		c.quota.mu.Unlock()
		c.refreshQuota(key)
	})
}

// normalizeQuotaRead maps one successful passthrough to the two projected
// windows under TS-04.R89/R109. Unverifiable values stay unknown. ok is false
// when the payload cannot be decoded; the caller treats that as a failed read.
func normalizeQuotaRead(provider string, raw json.RawMessage, now time.Time) (state.SubscriptionQuota, bool) {
	out := state.SubscriptionQuota{Provider: provider, CheckedAt: now.Format(time.RFC3339)}
	observed := false
	switch provider {
	case "claude":
		body, err := decodeClaudeQuota(raw)
		if err != nil {
			return out, false
		}
		if !body.Available {
			out.ReasonCode = state.QuotaReasonNoSubscription
			break
		}
		out.Windows.FiveHour, observed = body.Windows["five_hour"].window(now, observed)
		var weekly bool
		out.Windows.Weekly, weekly = body.Windows["seven_day"].window(now, false)
		observed = observed || weekly
	case "codex":
		var body struct {
			Snapshots []struct {
				LimitID   string                   `json:"limitId"`
				Primary   *codexQuotaWindowPayload `json:"primary"`
				Secondary *codexQuotaWindowPayload `json:"secondary"`
			} `json:"snapshots"`
		}
		if json.Unmarshal(raw, &body) != nil {
			return out, false
		}
		// Only the account's main snapshot (listed first); a model-specific
		// limit never substitutes for the all-model windows.
		if len(body.Snapshots) == 0 {
			break
		}
		main := body.Snapshots[0]
		for _, w := range []*codexQuotaWindowPayload{main.Primary, main.Secondary} {
			if w == nil || w.WindowDurationMins == nil {
				continue
			}
			var slot *state.SubscriptionQuotaWindow
			switch *w.WindowDurationMins {
			case 300:
				slot = &out.Windows.FiveHour
			case 10080:
				slot = &out.Windows.Weekly
			default:
				continue
			}
			*slot = w.window(now)
			observed = true
		}
	}
	fiveKnown := out.Windows.FiveHour.UsedPercent != nil
	weekKnown := out.Windows.Weekly.UsedPercent != nil
	switch {
	case fiveKnown && weekKnown:
		out.Availability = "complete"
	case observed:
		out.Availability = "partial"
	default:
		out.Availability = "unavailable"
	}
	return out, true
}

type claudeQuotaWindowPayload struct {
	Utilization *float64 `json:"utilization"`
	ResetsAt    *string  `json:"resets_at"`
}

// claudeQuota is one decoded Claude passthrough (sdk.d.ts rate_limits): named
// window objects plus the model_scoped list of per-model weekly windows.
type claudeQuota struct {
	Available   bool
	Windows     map[string]*claudeQuotaWindowPayload
	ModelScoped []struct {
		DisplayName string `json:"display_name"`
		claudeQuotaWindowPayload
	}
}

// claudeNamedWindows are the SDK's declared window members; a malformed one
// fails the read. Other members are kept only when they decode as a window, so
// an exhausted unidentified window still voids recovery evidence.
var claudeNamedWindows = map[string]bool{
	"five_hour": true, "seven_day": true, "seven_day_oauth_apps": true,
	"seven_day_opus": true, "seven_day_sonnet": true,
}

func decodeClaudeQuota(raw json.RawMessage) (claudeQuota, error) {
	var top struct {
		Available  bool                       `json:"rate_limits_available"`
		RateLimits map[string]json.RawMessage `json:"rate_limits"`
	}
	if err := json.Unmarshal(raw, &top); err != nil {
		return claudeQuota{}, err
	}
	out := claudeQuota{Available: top.Available, Windows: map[string]*claudeQuotaWindowPayload{}}
	for name, v := range top.RateLimits {
		if name == "model_scoped" {
			if err := json.Unmarshal(v, &out.ModelScoped); err != nil {
				return claudeQuota{}, err
			}
			continue
		}
		var w *claudeQuotaWindowPayload
		if err := json.Unmarshal(v, &w); err != nil {
			if claudeNamedWindows[name] {
				return claudeQuota{}, err
			}
			continue
		}
		out.Windows[name] = w
	}
	return out, nil
}

// window decodes Claude's 0..100 utilization and ISO 8601 reset.
func (p *claudeQuotaWindowPayload) window(now time.Time, prior bool) (state.SubscriptionQuotaWindow, bool) {
	if p == nil {
		return state.SubscriptionQuotaWindow{}, prior
	}
	w := state.SubscriptionQuotaWindow{ObservedAt: now.Format(time.RFC3339)}
	if u := p.Utilization; u != nil && *u >= 0 && *u <= 100 && !math.IsNaN(*u) {
		v := *u
		w.UsedPercent = &v
	}
	if p.ResetsAt != nil {
		if t, err := time.Parse(time.RFC3339, *p.ResetsAt); err == nil {
			w.ResetAt = t.UTC().Format(time.RFC3339)
		}
	}
	return w, true
}

type codexQuotaWindowPayload struct {
	UsedPercent        *float64 `json:"usedPercent"`
	WindowDurationMins *int64   `json:"windowDurationMins"`
	ResetsAt           *int64   `json:"resetsAt"`
}

// window decodes Codex's integer usedPercent and epoch-second resetsAt.
func (p *codexQuotaWindowPayload) window(now time.Time) state.SubscriptionQuotaWindow {
	w := state.SubscriptionQuotaWindow{ObservedAt: now.Format(time.RFC3339)}
	if u := p.UsedPercent; u != nil && *u >= 0 && *u <= 100 {
		v := *u
		w.UsedPercent = &v
	}
	if r := p.ResetsAt; r != nil && *r > 0 {
		w.ResetAt = time.Unix(*r, 0).UTC().Format(time.RFC3339)
	}
	return w
}

// quotaSnapshotReset is TS-04.R111's optional evidence: the latest reset among
// the scope's exhausted windows applicable to modelID, or nil for "no snapshot
// evidence". It reads through via when it advertises the read, else through
// any live session of the scope; with none, only a stored read made after
// since counts. No process is started.
func (c *ChatRuntime) quotaSnapshotReset(key string, via *agentState, modelID string, since time.Time) *time.Time {
	if key == "" {
		return nil
	}
	reader := via
	if reader == nil || !reader.quotaRead {
		reader = c.quotaReader(key)
	}
	var raw json.RawMessage
	if reader != nil {
		call := c.startQuotaCall(key, reader)
		if call == nil {
			return nil
		}
		<-call.done // bounded by the read timeout
		if call.err != nil {
			return nil
		}
		raw = call.raw
	} else {
		c.quota.mu.Lock()
		if sc := c.quota.scopes[key]; sc != nil && sc.last != nil && sc.last.at.After(since) {
			raw = sc.last.raw
		}
		c.quota.mu.Unlock()
		if raw == nil {
			return nil
		}
	}
	return exhaustedQuotaReset(quotaProviderOfKey(c, key), raw, modelID, time.Now().UTC())
}

func quotaProviderOfKey(c *ChatRuntime, key string) string {
	c.quota.mu.Lock()
	defer c.quota.mu.Unlock()
	if sc := c.quota.scopes[key]; sc != nil {
		return sc.snap.Provider
	}
	return ""
}

// QuotaSnapshotReset is quotaSnapshotReset for agentID's scope, for the
// recovery pre-check (TS-04.R111).
func (c *ChatRuntime) QuotaSnapshotReset(agentID, modelID string, since time.Time) *time.Time {
	c.quota.mu.Lock()
	key := c.quota.agents[agentID]
	c.quota.mu.Unlock()
	return c.quotaSnapshotReset(key, nil, modelID, since)
}

// exhaustedQuotaReset applies R96's window rule to one raw read: ordinary
// windows apply to every model, a Claude model-specific weekly window only to
// that family. Any applicable exhausted window without a future reset, or an
// exhausted window that cannot be identified, makes the evidence unknown.
func exhaustedQuotaReset(provider string, raw json.RawMessage, modelID string, now time.Time) *time.Time {
	var latest *time.Time
	exhausted := func(used *float64, reset time.Time, known bool) bool {
		if used == nil || *used < 100 {
			return true
		}
		if !known || !reset.After(now) {
			return false
		}
		if latest == nil || reset.After(*latest) {
			latest = &reset
		}
		return true
	}
	switch provider {
	case "claude":
		body, err := decodeClaudeQuota(raw)
		if err != nil || !body.Available {
			return nil
		}
		model := strings.ToLower(modelID)
		// A family window counts only when the model id names its family; an
		// exhausted window whose family cannot be read is unknown.
		family := func(name string, w *claudeQuotaWindowPayload) (applies, unknown bool) {
			if name == "" {
				return false, w.Utilization != nil && *w.Utilization >= 100
			}
			return model != "" && strings.Contains(model, name), false
		}
		var applicable []*claudeQuotaWindowPayload
		for name, w := range body.Windows {
			if w == nil {
				continue
			}
			applies, unknown := true, false
			switch name {
			case "five_hour", "seven_day":
			case "seven_day_opus", "seven_day_sonnet":
				applies, unknown = family(strings.TrimPrefix(name, "seven_day_"), w)
			default:
				applies, unknown = family("", w)
			}
			if unknown {
				return nil
			}
			if applies {
				applicable = append(applicable, w)
			}
		}
		for i := range body.ModelScoped {
			w := &body.ModelScoped[i]
			applies, unknown := family(strings.ToLower(strings.TrimSpace(w.DisplayName)), &w.claudeQuotaWindowPayload)
			if unknown {
				return nil
			}
			if applies {
				applicable = append(applicable, &w.claudeQuotaWindowPayload)
			}
		}
		for _, w := range applicable {
			var reset time.Time
			known := false
			if w.ResetsAt != nil {
				if t, err := time.Parse(time.RFC3339, *w.ResetsAt); err == nil {
					reset, known = t.UTC(), true
				}
			}
			if !exhausted(w.Utilization, reset, known) {
				return nil
			}
		}
	case "codex":
		var body struct {
			Snapshots []struct {
				Primary   *codexQuotaWindowPayload `json:"primary"`
				Secondary *codexQuotaWindowPayload `json:"secondary"`
			} `json:"snapshots"`
		}
		if json.Unmarshal(raw, &body) != nil || len(body.Snapshots) == 0 {
			return nil
		}
		for _, w := range []*codexQuotaWindowPayload{body.Snapshots[0].Primary, body.Snapshots[0].Secondary} {
			if w == nil || w.UsedPercent == nil || *w.UsedPercent < 100 {
				continue
			}
			if w.WindowDurationMins == nil || (*w.WindowDurationMins != 300 && *w.WindowDurationMins != 10080) {
				return nil
			}
			var reset time.Time
			known := w.ResetsAt != nil && *w.ResetsAt > 0
			if known {
				reset = time.Unix(*w.ResetsAt, 0).UTC()
			}
			if !exhausted(w.UsedPercent, reset, known) {
				return nil
			}
		}
	}
	return latest
}
