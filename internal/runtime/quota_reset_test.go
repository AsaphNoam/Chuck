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

// incidentText is the terminal RPC message captured for a_a7e830 event 200
// (provider text of event 199), arriving 2026-10-29T16:25:06Z.
const incidentText = "Internal error: You've hit your monthly spend limit · raise it at https://claude.ai/settings/usage?from=cc_cli_limit_message · your session limit resets 10pm (Asia/Jerusalem)"

// Text-only extraction (FS-01.A39, TS-04.R107). Producer fixtures follow the
// pinned formatters (QUOTA-RESET-FALLBACK-2026-10-16): Claude native 2.1.286
// `iu`/`Acn` and Codex 0.159.3 format_retry_timestamp. Alternate wordings
// check that no fixed sentence is needed. Each expected instant is the
// stated wall time converted by hand under the named zone's rules.
func TestTextResetExtractsWithoutFixedSentence(t *testing.T) {
	ny, jerusalem := mustZone(t, "America/New_York"), mustZone(t, "Asia/Jerusalem")
	const claude, codex = "claude-acp", "codex-acp"
	const incident = "2026-10-29T16:25:06Z" // 18:25:06 IST (+02:00)
	for _, tc := range []struct {
		name, backend, text, observed, want string
		zone                                *time.Location
	}{
		// Claude producer forms.
		{"time only today", claude, "You've hit your session limit · resets 9am (America/New_York)", "2026-10-18T08:00:00Z", "2026-10-18T13:00:00Z", nil},
		{"time only minutes", claude, "You've hit your session limit · resets 2:05pm (America/New_York)", "2026-10-18T08:00:00Z", "2026-10-18T18:05:00Z", nil},
		{"time only tomorrow within 24h", claude, "You've hit your session limit · resets 10am (America/New_York)", "2026-10-18T20:00:00Z", "2026-10-19T14:00:00Z", nil},
		{"date without year", claude, "You've hit your weekly limit · resets Oct 25, 5pm (Asia/Tokyo)", "2026-10-18T08:00:00Z", "2026-10-25T08:00:00Z", nil},
		{"date with year", claude, "You've hit your weekly limit · resets Jan 3, 2027, 6:13pm (Europe/London) · progress saved", "2026-12-31T06:53:20Z", "2027-01-03T18:13:00Z", nil},
		{"incident named session reset", claude, incidentText, incident, "2026-10-29T20:00:00Z", nil},
		// Changed wording, punctuation, case and clock forms.
		{"available again", claude, "Quota exhausted; available again at 10pm (Asia/Jerusalem)", incident, "2026-10-29T20:00:00Z", nil},
		{"24-hour retry", claude, "Usage limit reached. Retry at 22:00 (Asia/Jerusalem)", incident, "2026-10-29T20:00:00Z", nil},
		{"upper case with minutes", claude, "USAGE LIMIT REACHED -- RESETS 10:00 PM (Asia/Jerusalem)!", incident, "2026-10-29T20:00:00Z", nil},
		{"dotted meridiem", claude, "Limit hit, resets at 10 p.m. (Asia/Jerusalem)", incident, "2026-10-29T20:00:00Z", nil},
		{"bare IANA zone", claude, "limit reached, back at 22:00 Asia/Jerusalem", incident, "2026-10-29T20:00:00Z", nil},
		{"omitted zone uses process zone", claude, "Usage limit reached; try again at 22:00.", incident, "2026-10-29T20:00:00Z", jerusalem},
		{"numeric offset", claude, "Usage limit reached; retry at 22:00 +02:00", incident, "2026-10-29T20:00:00Z", nil},
		{"UTC offset", claude, "Usage limit reached; retry at 10pm UTC+2", incident, "2026-10-29T20:00:00Z", nil},
		{"ISO with offset", claude, "Quota exhausted until 2026-10-30T09:15:00+02:00", incident, "2026-10-30T07:15:00Z", nil},
		{"ISO with Z", claude, "Quota exhausted until 2026-10-30T07:15Z.", incident, "2026-10-30T07:15:00Z", nil},
		{"long month after zone", claude, "Limit reached. Available at 9:15am (UTC) on October 30", incident, "2026-10-30T09:15:00Z", nil},
		{"long month with year", claude, "resets October 30, 2026 at 9:15 AM (Europe/London)", incident, "2026-10-30T09:15:00Z", nil},
		{"equivalent repeats", claude, "resets 10pm (Asia/Jerusalem) · resets 22:00 (Asia/Jerusalem)", incident, "2026-10-29T20:00:00Z", nil},
		{"slash ratio reads as an unsupported date", claude, "Limit reached (v2.1.286, HTTP 429, 3/5 used, 1:1). See https://claude.ai/x:12:30/usage · resets 10pm (Asia/Jerusalem)", incident, "", nil},
		{"unrelated numbers without ratios", claude, "Limit reached (v2.1.286, HTTP 429, 1,000 tokens). See https://claude.ai/x/12:30/usage · resets 10pm (Asia/Jerusalem)", incident, "2026-10-29T20:00:00Z", nil},
		// Unknown: zones, DST, dates and conflicts.
		{"daylight-saving fold", claude, "You've hit your session limit · resets 1:30am (America/New_York)", "2026-11-01T04:30:00Z", "", nil},
		{"daylight-saving gap", claude, "You've hit your session limit · resets 2:30am (America/New_York)", "2026-03-08T05:00:00Z", "", nil},
		{"stale dated", claude, "You've hit your weekly limit · resets Oct 17, 5pm (Asia/Tokyo)", "2026-10-18T08:00:00Z", "", nil},
		{"unknown zone", claude, "You've hit your session limit · resets 9am (Mars/Olympus)", "2026-10-18T08:00:00Z", "", nil},
		{"abbreviated zone", claude, "You've hit your session limit · resets 9am EST", "2026-10-18T08:00:00Z", "", ny},
		{"missing zone and unknown process zone", claude, "You've hit your session limit · resets 9am", "2026-10-18T08:00:00Z", "", nil},
		{"two different clauses", claude, "· resets 9am (America/New_York) · resets 10am (America/New_York)", "2026-10-18T08:00:00Z", "", nil},
		{"rounded current minute is ambiguous", claude, "You've hit your session limit · resets 9am (UTC)", "2026-10-18T09:00:30Z", "", nil},
		{"current-minute ISO", claude, "Quota exhausted until 2026-10-29T16:25Z", incident, "", nil},
		{"day-first date", claude, "resets 30 Oct 9am (UTC)", incident, "", nil},
		{"numeric slash date", claude, "resets 10/30 9am (UTC)", incident, "", nil},
		{"relative date", claude, "resets tomorrow at 9am (UTC)", incident, "", nil},
		{"invalid hour", claude, "resets 13pm (UTC)", incident, "", nil},
		{"invalid ISO month", claude, "until 2026-13-01T09:00Z", incident, "", nil},
		// Codex producer forms and its same-date convention.
		{"codex same day", codex, "You’ve hit your usage limit. Try again at 3:05 PM.", "2026-10-18T12:00:00Z", "2026-10-18T19:05:00Z", ny},
		{"codex upgrade alternative with reset", codex, "You’ve hit your usage limit. Upgrade to Plus to continue using Codex (https://chatgpt.com/explore/plus), or try again at 11:59 PM.", "2026-10-18T12:00:00Z", "2026-10-19T03:59:00Z", ny},
		{"codex dated", codex, "You’ve hit your usage limit. Try again at Oct 21st, 2026 9:00 AM.", "2026-10-18T12:00:00Z", "2026-10-21T13:00:00Z", ny},
		{"codex dated without ordinal", codex, "Usage limit reached; available again October 21 2026, 09:00", "2026-10-18T12:00:00Z", "2026-10-21T13:00:00Z", ny},
		{"codex same instant in another process zone", codex, "You’ve hit your usage limit. Try again at 7:05 PM.", "2026-10-18T12:00:00Z", "2026-10-18T19:05:00Z", time.UTC},
		{"codex 24-hour retry", codex, "Usage limit reached. Retry at 22:00", "2026-10-18T12:00:00Z", "2026-10-19T02:00:00Z", ny},
		{"codex explicit zone", codex, "quota exhausted - available again at 3:05 pm (UTC).", "2026-10-18T12:00:00Z", "2026-10-18T15:05:00Z", ny},
		{"codex past time is stale, never tomorrow", codex, "You’ve hit your usage limit. Try again at 7:05 AM.", "2026-10-18T12:00:00Z", "", ny},
		{"codex just after midnight", codex, "You’ve hit your usage limit. Try again at 11:00 PM.", "2026-10-18T04:02:00Z", "", ny},
		{"codex daylight-saving gap", codex, "You’ve hit your usage limit. Try again at 2:30 AM.", "2026-03-08T06:00:00Z", "", ny},
		{"codex unknown process zone", codex, "You’ve hit your usage limit. Try again at 3:05 PM.", "2026-10-18T12:00:00Z", "", nil},
		{"codex later", codex, "You’ve hit your usage limit. Try again later.", "2026-10-18T12:00:00Z", "", ny},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := resolveQuotaReset(tc.backend, tc.text, rateLimitEvidence{}, utc(tc.observed), tc.zone)
			if tc.want == "" {
				if got != nil {
					t.Fatalf("reset = %s, want unknown", got)
				}
				return
			}
			if got == nil || !got.Equal(utc(tc.want)) {
				t.Fatalf("reset = %v, want %s", got, tc.want)
			}
		})
	}
}

func rateLimitFrame(info map[string]any, model string) json.RawMessage {
	meta := map[string]any{"_claude/rateLimit": info}
	if model != "" {
		meta["_claude/model"] = model
	}
	b, _ := json.Marshal(map[string]any{"sessionId": "s", "update": map[string]any{
		"sessionUpdate": "usage_update", "used": 1, "size": 2, "_meta": meta}})
	return b
}

func structuredAt(at time.Time) rateLimitEvidence { return rateLimitEvidence{reset: &at} }

func TestRateLimitEvidenceKeepsApplicability(t *testing.T) {
	observe := func(model string, infos ...map[string]any) rateLimitEvidence {
		var e rateLimitEvidence
		for _, info := range infos {
			e.observe(rateLimitFrame(info, model))
		}
		return e
	}
	if e := observe("", map[string]any{"status": "rejected", "rateLimitType": "five_hour", "resetsAt": 1792328400}); e.unknown || e.reset == nil || !e.reset.Equal(utc("2026-10-18T13:00:00Z")) {
		t.Fatalf("rejected five_hour = %+v", e)
	}
	if e := observe("claude-sonnet-4-6",
		map[string]any{"status": "rejected", "rateLimitType": "five_hour", "resetsAt": 1792328400},
		map[string]any{"status": "rejected", "rateLimitType": "seven_day", "resetsAt": 1792332000}); e.unknown || !e.reset.Equal(utc("2026-10-18T14:00:00Z")) {
		t.Fatalf("several applicable blockers = %+v, want the latest", e)
	}
	if e := observe("claude-sonnet-4-6", map[string]any{"status": "rejected", "rateLimitType": "seven_day_sonnet", "resetsAt": 1792328400}); e.unknown || e.reset == nil {
		t.Fatalf("matching model window = %+v", e)
	}
	// Another model's window neither schedules nor blocks this one.
	if e := observe("claude-sonnet-4-6", map[string]any{"status": "rejected", "rateLimitType": "seven_day_opus", "resetsAt": 1792328400}); e.unknown || e.reset != nil {
		t.Fatalf("mismatched model window = %+v", e)
	}
	for name, infos := range map[string][]map[string]any{
		"unstamped model window": {{"status": "rejected", "rateLimitType": "seven_day_opus", "resetsAt": 1792328400}},
		"overage":                {{"status": "rejected", "rateLimitType": "overage", "resetsAt": 1792328400}},
		"overage included":       {{"status": "rejected", "rateLimitType": "seven_day_overage_included", "resetsAt": 1792328400}},
		"no type":                {{"status": "rejected", "resetsAt": 1792328400}},
		"fractional":             {{"status": "rejected", "rateLimitType": "seven_day", "resetsAt": 1792328400.5}},
		"millis":                 {{"status": "rejected", "rateLimitType": "seven_day", "resetsAt": 1792328400000.0 * 1000}},
		"second blocker untimed": {{"status": "rejected", "rateLimitType": "five_hour", "resetsAt": 1792328400}, {"status": "rejected", "rateLimitType": "seven_day"}},
	} {
		if e := observe("", infos...); !e.unknown {
			t.Fatalf("%s = %+v, want unknown", name, e)
		}
	}
	if e := observe("", map[string]any{"status": "allowed_warning", "rateLimitType": "five_hour", "resetsAt": 1792328400}); e.unknown || e.reset != nil {
		t.Fatalf("warning window = %+v, want no evidence", e)
	}
}

func TestResolveQuotaResetCombinesSourcesConservatively(t *testing.T) {
	observed := utc("2026-10-18T08:00:00Z")
	text := "You've hit your session limit · resets 2:05pm (America/New_York)"
	inMinute := utc("2026-10-18T18:05:42Z")
	other := utc("2026-10-18T19:00:00Z")
	if got, _ := resolveQuotaReset("claude-acp", text, structuredAt(inMinute), observed, nil); got == nil || !got.Equal(inMinute) {
		t.Fatalf("corroborated structured reset = %v, want %s", got, inMinute)
	}
	if got, _ := resolveQuotaReset("claude-acp", text, structuredAt(other), observed, nil); got != nil {
		t.Fatalf("conflicting sources = %s, want unknown", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", "You've hit your session limit", structuredAt(other), observed, nil); got == nil || !got.Equal(other) {
		t.Fatalf("structured only = %v", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", text, rateLimitEvidence{reset: &inMinute, unknown: true}, observed, nil); got != nil {
		t.Fatalf("untimed applicable blocker = %s, want unknown", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", "You've hit your session limit", rateLimitEvidence{}, observed, nil); got != nil {
		t.Fatalf("no evidence = %s", got)
	}
	if got, _ := resolveQuotaReset("other", text, structuredAt(inMinute), observed, nil); got != nil {
		t.Fatalf("unsupported backend = %s", got)
	}
	// Two distinct clauses conflict; structured data does not resolve them.
	if got, _ := resolveQuotaReset("claude-acp", "limit · resets 9am (UTC) · resets 10am (UTC)", structuredAt(utc("2026-10-18T09:00:10Z")), observed, nil); got != nil {
		t.Fatalf("conflicting text with structured = %s, want unknown", got)
	}
	// The producer truncates seconds: a 09:00:50 reset observed at 09:00:30
	// prints 9am, which alone could also mean tomorrow.
	rounded := "You've hit your session limit · resets 9am (UTC)"
	if got, _ := resolveQuotaReset("claude-acp", rounded, rateLimitEvidence{}, utc("2026-10-18T09:00:30Z"), nil); got != nil {
		t.Fatalf("rounded current minute = %s, want unknown", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", rounded, structuredAt(utc("2026-10-18T09:00:50Z")), utc("2026-10-18T09:00:30Z"), nil); got == nil || !got.Equal(utc("2026-10-18T09:00:50Z")) {
		t.Fatalf("structured corroboration of the running minute = %v", got)
	}
	if got, _ := resolveQuotaReset("claude-acp", rounded, rateLimitEvidence{}, utc("2026-10-18T09:01:00Z"), nil); got == nil || !got.Equal(utc("2026-10-19T09:00:00Z")) {
		t.Fatalf("legitimate next-day text = %v", got)
	}
	if _, intervene := resolveQuotaReset("claude-acp", "You've hit your monthly spend limit · ask your admin to raise it at claude.ai/admin-settings/usage", rateLimitEvidence{}, observed, nil); !intervene {
		t.Fatal("spend limit without reset did not require intervention")
	}
	if _, intervene := resolveQuotaReset("codex-acp", "Your workspace is out of credits. Add credits to continue.", rateLimitEvidence{}, observed, time.UTC); !intervene {
		t.Fatal("codex credits without reset did not require intervention")
	}
	if got, intervene := resolveQuotaReset("codex-acp", "You’ve hit your usage limit. Visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 3:05 PM.", rateLimitEvidence{}, utc("2026-10-18T12:00:00Z"), mustZone(t, "America/New_York")); got == nil || intervene {
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
