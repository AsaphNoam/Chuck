package runtime

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AsaphNoam/Chuck/internal/state"
)

// Provider-owned payloads, written as each pinned adapter's passthrough
// returns them (sdk.d.ts rate_limits; Codex RateLimitSnapshot windows).
const (
	claudeQuotaBoth = `{"rate_limits_available":true,"rate_limits":{
		"five_hour":{"utilization":42.5,"resets_at":"2026-11-12T18:00:00+02:00"},
		"seven_day":{"utilization":7,"resets_at":"2026-11-15T09:30:00Z"},
		"seven_day_opus":{"utilization":99,"resets_at":"2026-11-14T00:00:00Z"},
		"extra_usage":{"utilization":3}}}`
	codexQuotaBoth = `{"snapshots":[
		{"limitId":"codex","primary":{"usedPercent":12,"windowDurationMins":300,"resetsAt":1794500000},
		 "secondary":{"usedPercent":61,"windowDurationMins":10080,"resetsAt":1794900000}},
		{"limitId":"codex_other","primary":{"usedPercent":100,"windowDurationMins":300,"resetsAt":1794400000}}]}`
)

func TestNormalizeQuotaReadMapsOnlyIdentifiedWindows(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	stamp := now.Format(time.RFC3339)
	pct := func(v float64) *float64 { return &v }
	cases := []struct {
		name, provider, raw string
		want                state.SubscriptionQuota
	}{
		{"claude both windows; model-specific and extra usage dropped", "claude", claudeQuotaBoth, state.SubscriptionQuota{
			Provider: "claude", Availability: "complete", CheckedAt: stamp,
			Windows: state.SubscriptionQuotaWindows{
				FiveHour: state.SubscriptionQuotaWindow{UsedPercent: pct(42.5), ResetAt: "2026-11-12T16:00:00Z", ObservedAt: stamp},
				Weekly:   state.SubscriptionQuotaWindow{UsedPercent: pct(7), ResetAt: "2026-11-15T09:30:00Z", ObservedAt: stamp},
			}}},
		{"claude without a subscription", "claude", `{"rate_limits_available":false,"rate_limits":null}`, state.SubscriptionQuota{
			Provider: "claude", Availability: "unavailable", CheckedAt: stamp, ReasonCode: state.QuotaReasonNoSubscription}},
		{"claude five-hour only with null values", "claude", `{"rate_limits_available":true,"rate_limits":{"five_hour":{"utilization":null,"resets_at":null}}}`, state.SubscriptionQuota{
			Provider: "claude", Availability: "partial", CheckedAt: stamp,
			Windows: state.SubscriptionQuotaWindows{FiveHour: state.SubscriptionQuotaWindow{ObservedAt: stamp}}}},
		{"claude out-of-range utilization stays unknown", "claude", `{"rate_limits_available":true,"rate_limits":{"seven_day":{"utilization":140,"resets_at":"not a time"}}}`, state.SubscriptionQuota{
			Provider: "claude", Availability: "partial", CheckedAt: stamp,
			Windows: state.SubscriptionQuotaWindows{Weekly: state.SubscriptionQuotaWindow{ObservedAt: stamp}}}},
		{"codex windows by duration; model-specific snapshot ignored", "codex", codexQuotaBoth, state.SubscriptionQuota{
			Provider: "codex", Availability: "complete", CheckedAt: stamp,
			Windows: state.SubscriptionQuotaWindows{
				FiveHour: state.SubscriptionQuotaWindow{UsedPercent: pct(12), ResetAt: time.Unix(1794500000, 0).UTC().Format(time.RFC3339), ObservedAt: stamp},
				Weekly:   state.SubscriptionQuotaWindow{UsedPercent: pct(61), ResetAt: time.Unix(1794900000, 0).UTC().Format(time.RFC3339), ObservedAt: stamp},
			}}},
		{"codex unidentifiable window stays unknown", "codex", `{"snapshots":[{"limitId":"codex","primary":{"usedPercent":5,"windowDurationMins":60,"resetsAt":1794500000}}]}`, state.SubscriptionQuota{
			Provider: "codex", Availability: "unavailable", CheckedAt: stamp}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeQuotaRead(tc.provider, json.RawMessage(tc.raw), now)
			gotJSON, _ := json.Marshal(got)
			wantJSON, _ := json.Marshal(tc.want)
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("normalized\n got %s\nwant %s", gotJSON, wantJSON)
			}
			for _, banned := range []string{"opus", "extra", "credit", "plan", "codex_other"} {
				if strings.Contains(string(gotJSON), banned) {
					t.Fatalf("projection leaks %q: %s", banned, gotJSON)
				}
			}
		})
	}
}

// quotaTest starts one fake agent whose adapter advertises the quota read and
// answers it from a file the test rewrites.
type quotaTest struct {
	c       *ChatRuntime
	spec    LaunchSpec
	result  string
	log     string
	touchMu sync.Mutex
	touched map[string]int
}

func newQuotaTest(t *testing.T, payload string, extraEnv ...string) *quotaTest {
	t.Helper()
	c, spec := newChatTest(t, "stream_text")
	dir := t.TempDir()
	q := &quotaTest{c: c, spec: spec, result: filepath.Join(dir, "quota.json"), log: filepath.Join(dir, "quota.log"), touched: map[string]int{}}
	q.setResult(t, payload)
	q.spec.Env = append(q.spec.Env, "FAKEACP_QUOTA="+q.result, "FAKEACP_QUOTA_LOG="+q.log)
	q.spec.Env = append(q.spec.Env, extraEnv...)
	// An empty settings home gives a shareable routing; without one every agent
	// is its own unsupported scope.
	q.spec.QuotaCheckEnv = []string{"HOME=" + t.TempDir()}
	c.SetStateTouch(func(agentID string) {
		q.touchMu.Lock()
		q.touched[agentID]++
		q.touchMu.Unlock()
	})
	return q
}

func (q *quotaTest) setResult(t *testing.T, payload string) {
	t.Helper()
	if err := os.WriteFile(q.result, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (q *quotaTest) reads() int {
	raw, _ := os.ReadFile(q.log)
	return strings.Count(string(raw), "\n")
}

func (q *quotaTest) start(t *testing.T) *Handle {
	t.Helper()
	h, err := q.c.Start(context.Background(), q.spec)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = q.c.Stop(context.Background(), h.AgentID) })
	return h
}

// waitQuota waits until agentID projects a snapshot satisfying ok.
func waitQuota(t *testing.T, c *ChatRuntime, agentID string, ok func(*state.SubscriptionQuota) bool) *state.SubscriptionQuota {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		got := c.SubscriptionQuota(agentID)
		if got != nil && ok(got) {
			return got
		}
		if time.Now().After(deadline) {
			raw, _ := json.Marshal(got)
			t.Fatalf("quota for %s never settled; last %s", agentID, raw)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func checked(s *state.SubscriptionQuota) bool { return s.CheckedAt != "" }

// TestQuotaSnapshotReadsAtOpenAndTurnEndAndSharesTheScope is FS-09.A52's
// runtime half: one read per session open and per turn end, no transcript
// event, and a stopped sibling on the same backend and routing projects the
// same snapshot while another backend's agent does not.
func TestQuotaSnapshotReadsAtOpenAndTurnEndAndSharesTheScope(t *testing.T) {
	q := newQuotaTest(t, claudeQuotaBoth)
	sibling := state.Agent{AgentID: "a_sibling", Backend: q.spec.Agent.Backend}
	other := state.Agent{AgentID: "a_other", Backend: "codex"}
	q.c.SetQuotaScopeResolver(func(agentID string) (string, string, string, bool) {
		switch agentID {
		case sibling.AgentID:
			return sibling.Backend, "claude-acp", QuotaCheckRouting("claude-acp", q.spec.QuotaCheckEnv, q.spec.Cwd), true
		case other.AgentID:
			return other.Backend, "codex-acp", "", true
		}
		return "", "", "", false
	})
	h := q.start(t)
	first := waitQuota(t, q.c, h.AgentID, checked)
	if first.Availability != "complete" || q.reads() != 1 {
		t.Fatalf("after open: availability %q reads %d, want complete and one read", first.Availability, q.reads())
	}

	ch, unsub, err := q.c.Subscribe(h.AgentID)
	if err != nil {
		t.Fatal(err)
	}
	defer unsub()
	q.setResult(t, `{"rate_limits_available":true,"rate_limits":{"five_hour":{"utilization":50,"resets_at":null},"seven_day":{"utilization":8,"resets_at":null}}}`)
	if err := q.c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
		t.Fatal(err)
	}
	evs := drainTurn(t, ch)
	got := waitQuota(t, q.c, h.AgentID, func(s *state.SubscriptionQuota) bool {
		return s.Windows.FiveHour.UsedPercent != nil && *s.Windows.FiveHour.UsedPercent == 50
	})
	if q.reads() != 2 {
		t.Fatalf("reads = %d after one turn, want 2", q.reads())
	}
	for _, e := range evs {
		if strings.Contains(string(mustJSON(t, e)), "quota") {
			t.Fatalf("quota read produced a transcript event: %+v", e)
		}
	}
	if sib := q.c.SubscriptionQuota(sibling.AgentID); sib == nil || *sib.Windows.FiveHour.UsedPercent != 50 || sib.CheckedAt != got.CheckedAt {
		t.Fatalf("stopped sibling projects %+v, want the shared snapshot", sib)
	}
	if o := q.c.SubscriptionQuota(other.AgentID); o == nil || o.Provider != "codex" || o.CheckedAt != "" || o.Availability != "unavailable" {
		t.Fatalf("other backend projects %+v, want its own never-read codex scope", o)
	}
	q.touchMu.Lock()
	touched := q.touched[h.AgentID]
	q.touchMu.Unlock()
	if touched == 0 {
		t.Fatal("a completed read touched no agent of the scope")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

// TestQuotaSnapshotFailedReadKeepsStaleValues: a failed read keeps the earlier
// windows with its reason; a timed-out one reports read_timeout.
func TestQuotaSnapshotFailedReadKeepsStaleValues(t *testing.T) {
	q := newQuotaTest(t, claudeQuotaBoth)
	h := q.start(t)
	first := waitQuota(t, q.c, h.AgentID, checked)
	q.setResult(t, "not json")
	q.c.refreshQuota(q.c.quota.agents[h.AgentID])
	stale := waitQuota(t, q.c, h.AgentID, func(s *state.SubscriptionQuota) bool { return s.ReasonCode != "" })
	if stale.ReasonCode != state.QuotaReasonReadFailed || stale.CheckedAt != first.CheckedAt ||
		*stale.Windows.FiveHour.UsedPercent != 42.5 {
		t.Fatalf("after failure %+v, want the earlier values with read_failed", stale)
	}
}

func TestQuotaSnapshotReadTimeout(t *testing.T) {
	q := newQuotaTest(t, claudeQuotaBoth, "FAKEACP_QUOTA_HANG=1")
	q.c.quota.timeout = 50 * time.Millisecond
	h := q.start(t)
	got := waitQuota(t, q.c, h.AgentID, func(s *state.SubscriptionQuota) bool { return s.ReasonCode != "" })
	if got.ReasonCode != state.QuotaReasonReadTimeout || got.CheckedAt != "" {
		t.Fatalf("hung read projects %+v, want read_timeout with no values", got)
	}
}

// TestQuotaSnapshotUnadvertisedAdapterNeverReads: an adapter without the
// advertisement reads unsupported and is never called (TS-04.R109).
func TestQuotaSnapshotUnadvertisedAdapterNeverReads(t *testing.T) {
	c, spec := newChatTest(t, "stream_text")
	log := filepath.Join(t.TempDir(), "quota.log")
	spec.Env = append(spec.Env, "FAKEACP_QUOTA_LOG="+log)
	h, err := c.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Stop(context.Background(), h.AgentID) })
	got := c.SubscriptionQuota(h.AgentID)
	if got == nil || got.ReasonCode != state.QuotaReasonUnsupportedAdapter || got.Availability != "unavailable" {
		t.Fatalf("unadvertised adapter projects %+v, want unavailable/unsupported_adapter", got)
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatal("an unadvertised adapter received a quota read")
	}
}

// TestQuotaSnapshotArmsOneReadAfterAReset: a future displayed reset schedules
// exactly one read once it passes.
func TestQuotaSnapshotArmsOneReadAfterAReset(t *testing.T) {
	c := NewChatRuntime(nil)
	key := c.bindQuotaScope("a1", "claude", "claude-acp", "")
	now := time.Now().UTC()
	sc := c.quota.scopes[key]
	sc.snap.Windows.FiveHour.ResetAt = now.Add(time.Hour).Format(time.RFC3339)
	sc.snap.Windows.Weekly.ResetAt = now.Add(-time.Hour).Format(time.RFC3339)
	c.quota.mu.Lock()
	c.armQuotaResetLocked(key, sc, now)
	armed := sc.resetAt
	c.armQuotaResetLocked(key, sc, now)
	same := sc.timer
	c.quota.mu.Unlock()
	defer same.Stop()
	if armed.Format(time.RFC3339) != sc.snap.Windows.FiveHour.ResetAt {
		t.Fatalf("armed %v, want the future five-hour reset", armed)
	}
	if sc.timer != same {
		t.Fatal("re-arming the same reset replaced its timer")
	}
}

// TestExhaustedQuotaResetAppliesTheWindowRule is FS-01.A41's window rule: the
// latest reset among exhausted applicable windows; a foreign model family's
// weekly window is ignored; anything unknown is no evidence.
func TestExhaustedQuotaResetAppliesTheWindowRule(t *testing.T) {
	now := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	soon, later := "2026-11-12T15:00:00Z", "2026-11-15T09:00:00Z"
	claude := func(windows string) json.RawMessage {
		return json.RawMessage(`{"rate_limits_available":true,"rate_limits":{` + windows + `}}`)
	}
	at := func(s string) *time.Time { v, _ := time.Parse(time.RFC3339, s); return &v }
	cases := []struct {
		name, provider, model string
		raw                   json.RawMessage
		want                  *time.Time
	}{
		{"exhausted weekly", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":40,"resets_at":"` + soon + `"},"seven_day":{"utilization":100,"resets_at":"` + later + `"}`), at(later)},
		{"two exhausted use the later", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":100,"resets_at":"` + soon + `"},"seven_day":{"utilization":100,"resets_at":"` + later + `"}`), at(later)},
		{"foreign family weekly ignored", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":100,"resets_at":"` + soon + `"},"seven_day_opus":{"utilization":100,"resets_at":"` + later + `"}`), at(soon)},
		{"own family weekly counts", "claude", "claude-opus-4-7", claude(`"five_hour":{"utilization":100,"resets_at":"` + soon + `"},"seven_day_opus":{"utilization":100,"resets_at":"` + later + `"}`), at(later)},
		{"nothing exhausted", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":99,"resets_at":"` + soon + `"}`), nil},
		{"exhausted without reset", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":100,"resets_at":null}`), nil},
		{"past reset", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":100,"resets_at":"2026-11-12T09:00:00Z"}`), nil},
		{"unidentified exhausted window", "claude", "claude-sonnet-4-6", claude(`"five_hour":{"utilization":100,"resets_at":"` + soon + `"},"mystery":{"utilization":100}`), nil},
		{"no subscription", "claude", "claude-sonnet-4-6", json.RawMessage(`{"rate_limits_available":false,"rate_limits":null}`), nil},
		{"codex weekly", "codex", "gpt-5.5", json.RawMessage(`{"snapshots":[{"limitId":"codex","primary":{"usedPercent":50,"windowDurationMins":300,"resetsAt":1794500000},"secondary":{"usedPercent":100,"windowDurationMins":10080,"resetsAt":1794900000}}]}`), func() *time.Time { v := time.Unix(1794900000, 0).UTC(); return &v }()},
		{"codex unknown duration", "codex", "gpt-5.5", json.RawMessage(`{"snapshots":[{"limitId":"codex","primary":{"usedPercent":100,"windowDurationMins":60,"resetsAt":1794500000}}]}`), nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := exhaustedQuotaReset(tc.provider, tc.raw, tc.model, now)
			if (got == nil) != (tc.want == nil) || got != nil && !got.Equal(*tc.want) {
				t.Fatalf("reset = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestQuotaSettlementUsesSnapshotResetOnlyWithoutFailureReset: a quota turn
// whose failure states no reset takes the snapshot's exhausted reset with
// source snapshot, and the turn still reads once (FS-01.A41, FS-09.A52).
func TestQuotaSettlementUsesSnapshotResetOnlyWithoutFailureReset(t *testing.T) {
	reset := time.Now().UTC().Add(30 * time.Hour).Truncate(time.Second)
	snapshot := `{"rate_limits_available":true,"rate_limits":{"five_hour":{"utilization":20,"resets_at":null},"seven_day":{"utilization":100,"resets_at":"` + reset.Format(time.RFC3339) + `"}}}`
	failure := quotaFailure("limit:error", 1, "limit", "error")
	dir := t.TempDir()
	result, log := filepath.Join(dir, "quota.json"), filepath.Join(dir, "quota.log")
	if err := os.WriteFile(result, []byte(snapshot), 0o600); err != nil {
		t.Fatal(err)
	}
	c, h, ch, _ := startQuotaTestAgent(t, "stream_text", "FAKEACP_QUOTA="+result, "FAKEACP_QUOTA_LOG="+log,
		"FAKEACP_FAILURE_UPDATE="+failure, "FAKEACP_FAILURE_RESULT="+failure)
	waitQuota(t, c, h.AgentID, checked)
	mu, calls := recordQuotaSettlement(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
		t.Fatal(err)
	}
	drainTurn(t, ch)
	mu.Lock()
	got := append([]quotaSettlementCall(nil), *calls...)
	mu.Unlock()
	if len(got) != 1 || got[0].quota == nil || got[0].quota.ResetAt == nil || !got[0].quota.ResetAt.Equal(reset) || got[0].quota.ResetSource != "snapshot" {
		t.Fatalf("settlement = %+v, want snapshot reset %s", got, reset)
	}
	time.Sleep(100 * time.Millisecond) // any stray turn-end refresh would land here
	raw, _ := os.ReadFile(log)
	if n := strings.Count(string(raw), "\n"); n != 2 {
		t.Fatalf("reads = %d, want one at open and one for the quota turn", n)
	}
}

// TestQuotaSettlementWithoutSnapshotEvidenceIsUnchanged: an unadvertised
// adapter leaves the reset unknown exactly as before (FS-01.A41 fallback).
func TestQuotaSettlementWithoutSnapshotEvidenceIsUnchanged(t *testing.T) {
	failure := quotaFailure("limit:error", 1, "limit", "error")
	c, h, ch, _ := startQuotaTestAgent(t, "stream_text", "FAKEACP_FAILURE_UPDATE="+failure, "FAKEACP_FAILURE_RESULT="+failure)
	mu, calls := recordQuotaSettlement(c)
	if err := c.SendPrompt(context.Background(), h.AgentID, "hello"); err != nil {
		t.Fatal(err)
	}
	drainTurn(t, ch)
	mu.Lock()
	defer mu.Unlock()
	if len(*calls) != 1 || (*calls)[0].quota == nil || (*calls)[0].quota.ResetAt != nil || (*calls)[0].quota.ResetSource != "" {
		t.Fatalf("settlement = %+v, want quota evidence with unknown reset", *calls)
	}
}
