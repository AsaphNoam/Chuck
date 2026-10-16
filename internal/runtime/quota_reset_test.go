package runtime

import (
	"encoding/json"
	"testing"
	"time"
)

func mustZone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func utc(s string) time.Time {
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return v
}

// Claude fixtures are the output of the pinned native 2.1.286 `iu`/`Acn`
// formatter logic (QUOTA-RESET-FALLBACK-2026-10-16) run under the named TZ for
// the given observation/reset instants, not handcrafted strings.
func TestClaudeTextResetFollowsPinnedFormatter(t *testing.T) {
	for _, tc := range []struct {
		name, text, observed, want string
	}{
		{"time only today", "You've hit your session limit · resets 9am (America/New_York)", "2026-10-18T08:00:00Z", "2026-10-18T13:00:00Z"},
		{"time only minutes", "You've hit your session limit · resets 2:05pm (America/New_York)", "2026-10-18T08:00:00Z", "2026-10-18T18:05:00Z"},
		{"time only tomorrow within 24h", "You've hit your session limit · resets 10am (America/New_York)", "2026-10-18T20:00:00Z", "2026-10-19T14:00:00Z"},
		{"date without year", "You've hit your weekly limit · resets Oct 25, 5pm (Asia/Tokyo)", "2026-10-18T08:00:00Z", "2026-10-25T08:00:00Z"},
		{"date with year", "You've hit your weekly limit · resets Jan 3, 2027, 6:13pm (Europe/London) · progress saved", "2026-12-31T06:53:20Z", "2027-01-03T18:13:00Z"}, // text has minute precision
		{"daylight-saving fold", "You've hit your session limit · resets 1:30am (America/New_York)", "2026-11-01T04:30:00Z", ""},
		{"daylight-saving gap", "You've hit your session limit · resets 2:30am (America/New_York)", "2026-03-08T05:00:00Z", ""},
		{"stale dated", "You've hit your weekly limit · resets Oct 17, 5pm (Asia/Tokyo)", "2026-10-18T08:00:00Z", ""},
		{"unknown zone", "You've hit your session limit · resets 9am (Mars/Olympus)", "2026-10-18T08:00:00Z", ""},
		{"missing zone", "You've hit your session limit · resets 9am", "2026-10-18T08:00:00Z", ""},
		{"explicit :00 is not producer output", "You've hit your session limit · resets 9:00am (America/New_York)", "2026-10-18T08:00:00Z", ""},
		{"two different clauses", "· resets 9am (America/New_York) · resets 10am (America/New_York)", "2026-10-18T08:00:00Z", ""},
		{"status-line variant without separator", "your session limit resets 9am (America/New_York)", "2026-10-18T08:00:00Z", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := claudeTextReset(tc.text, utc(tc.observed))
			if tc.want == "" {
				if ok {
					t.Fatalf("reset = %s, want unknown", got)
				}
				return
			}
			if !ok || !got.Equal(utc(tc.want)) {
				t.Fatalf("reset = %s/%v, want %s", got, ok, tc.want)
			}
		})
	}
}

// Codex fixtures follow pinned 0.159.3 format_retry_timestamp: "%-I:%M %p" for
// the process-local date, otherwise "%b %-d<ordinal>, %Y %-I:%M %p".
func TestCodexTextResetUsesProcessZoneAndSameDayRule(t *testing.T) {
	ny := mustZone(t, "America/New_York")
	for _, tc := range []struct {
		name, text, observed, want string
		zone                       *time.Location
	}{
		{"same day", "You’ve hit your usage limit. Try again at 3:05 PM.", "2026-10-18T12:00:00Z", "2026-10-18T19:05:00Z", ny},
		{"upgrade alternative with reset", "You’ve hit your usage limit. Upgrade to Plus to continue using Codex (https://chatgpt.com/explore/plus), or try again at 11:59 PM.", "2026-10-18T12:00:00Z", "2026-10-19T03:59:00Z", ny},
		{"dated", "You’ve hit your usage limit. Try again at Oct 21st, 2026 9:00 AM.", "2026-10-18T12:00:00Z", "2026-10-21T13:00:00Z", ny},
		{"same instant in another process zone", "You’ve hit your usage limit. Try again at 7:05 PM.", "2026-10-18T12:00:00Z", "2026-10-18T19:05:00Z", time.UTC},
		{"past time is stale, never tomorrow", "You’ve hit your usage limit. Try again at 7:05 AM.", "2026-10-18T12:00:00Z", "", ny},
		{"just after midnight", "You’ve hit your usage limit. Try again at 11:00 PM.", "2026-10-18T04:02:00Z", "", ny},
		{"wrong ordinal", "You’ve hit your usage limit. Try again at Oct 21nd, 2026 9:00 AM.", "2026-10-18T12:00:00Z", "", ny},
		{"unknown process zone", "You’ve hit your usage limit. Try again at 3:05 PM.", "2026-10-18T12:00:00Z", "", nil},
		{"later", "You’ve hit your usage limit. Try again later.", "2026-10-18T12:00:00Z", "", ny},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := codexTextReset(tc.text, utc(tc.observed), tc.zone)
			if tc.want == "" {
				if ok {
					t.Fatalf("reset = %s, want unknown", got)
				}
				return
			}
			if !ok || !got.Equal(utc(tc.want)) {
				t.Fatalf("reset = %s/%v, want %s", got, ok, tc.want)
			}
		})
	}
}

func TestClaudeRateLimitResetAppliesOnlyRejectedOrdinaryWindows(t *testing.T) {
	frame := func(info map[string]any) json.RawMessage {
		b, _ := json.Marshal(map[string]any{"sessionId": "s", "update": map[string]any{
			"sessionUpdate": "usage_update", "used": 1, "size": 2, "_meta": map[string]any{"_claude/rateLimit": info}}})
		return b
	}
	if got, ok := claudeRateLimitReset(frame(map[string]any{"status": "rejected", "rateLimitType": "five_hour", "resetsAt": 1792328400})); !ok || !got.Equal(utc("2026-10-18T13:00:00Z")) {
		t.Fatalf("rejected five_hour = %s/%v", got, ok)
	}
	for name, info := range map[string]map[string]any{
		"warning":    {"status": "allowed_warning", "rateLimitType": "five_hour", "resetsAt": 1792328400},
		"overage":    {"status": "rejected", "rateLimitType": "overage", "resetsAt": 1792328400},
		"no type":    {"status": "rejected", "resetsAt": 1792328400},
		"fractional": {"status": "rejected", "rateLimitType": "seven_day", "resetsAt": 1792328400.5},
		"millis":     {"status": "rejected", "rateLimitType": "seven_day", "resetsAt": 1792328400000.0 * 1000},
		"missing":    {"status": "rejected", "rateLimitType": "seven_day"},
	} {
		if got, ok := claudeRateLimitReset(frame(info)); ok {
			t.Fatalf("%s accepted reset %s", name, got)
		}
	}
}

func TestResolveQuotaResetCombinesSourcesConservatively(t *testing.T) {
	observed := utc("2026-10-18T08:00:00Z")
	text := "You've hit your session limit · resets 2:05pm (America/New_York)"
	inMinute := utc("2026-10-18T18:05:42Z")
	other := utc("2026-10-18T19:00:00Z")
	if got, _ := resolveQuotaReset("claude-acp", text, &inMinute, observed, nil); got == nil || !got.Equal(inMinute) {
		t.Fatalf("corroborated structured reset = %v, want %s", got, inMinute)
	}
	if got, _ := resolveQuotaReset("claude-acp", text, &other, observed, nil); got != nil {
		t.Fatalf("conflicting sources = %s, want unknown", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", "You've hit your session limit", &other, observed, nil); got == nil || !got.Equal(other) {
		t.Fatalf("structured only = %v", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", "You've hit your session limit", nil, observed, nil); got != nil {
		t.Fatalf("no evidence = %s", got)
	}
	if got, _ := resolveQuotaReset("other", text, &inMinute, observed, nil); got != nil {
		t.Fatalf("unsupported backend = %s", got)
	}
	if _, intervene := resolveQuotaReset("claude-acp", "You've hit your monthly spend limit · ask your admin to raise it at claude.ai/admin-settings/usage", nil, observed, nil); !intervene {
		t.Fatal("spend limit without reset did not require intervention")
	}
	if _, intervene := resolveQuotaReset("codex-acp", "Your workspace is out of credits. Add credits to continue.", nil, observed, time.UTC); !intervene {
		t.Fatal("codex credits without reset did not require intervention")
	}
	if got, intervene := resolveQuotaReset("codex-acp", "You’ve hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 3:05 PM.", nil, utc("2026-10-18T12:00:00Z"), mustZone(t, "America/New_York")); got == nil || intervene {
		t.Fatalf("promotional alternative beside reset = %v/%v, want reset without intervention", got, intervene)
	}
}

func TestProcessZoneFromEnvironment(t *testing.T) {
	if got := processZone([]string{"PATH=/bin"}); got != time.Local {
		t.Fatalf("inherited zone = %v, want Local", got)
	}
	if got := processZone([]string{"TZ=Asia/Tokyo"}); got == nil || got.String() != "Asia/Tokyo" {
		t.Fatalf("TZ override = %v", got)
	}
	if got := processZone([]string{"TZ="}); got != time.UTC {
		t.Fatalf("empty TZ = %v, want UTC", got)
	}
	if got := processZone([]string{"TZ=Nowhere/Invalid"}); got != nil {
		t.Fatalf("invalid TZ = %v, want unknown", got)
	}
}
