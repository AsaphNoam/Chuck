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
