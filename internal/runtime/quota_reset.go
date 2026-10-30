package runtime

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Reset evidence follows independent terminal quota classification
// (TS-04.R96/R107). Date/time candidates are tokenized anywhere in that exact
// provider failure, without a fixed sentence: 12-hour clocks with optional
// minutes, 24-hour HH:MM, ISO and English month-day dates, IANA zones and
// numeric offsets. The pinned producers' conventions (Claude native 2.1.286's
// `Acn`/`iu`, Codex 0.159.3's UsageLimitReachedError) only resolve what the
// text leaves open: an omitted zone is the provider process zone, and a
// time-only value follows the provider's date rule. Anything unsupported,
// stale, ambiguous or conflicting stays unknown.

const (
	monthExpr = `jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|june?|july?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?`
	dateExpr  = `(\d{4})-(\d{2})-(\d{2})|\b(` + monthExpr + `)\.?\s+(\d{1,2})(?:st|nd|rd|th)?(?:,?\s+(\d{4}))?`
)

var (
	// URLs and host paths carry digits and colons that are never times.
	urlToken = regexp.MustCompile(`(?i)\S*://\S*|\b(?:[a-z0-9\-]+\.)+[a-z]{2,}/\S*`)
	// 12-hour first, so `10:00 PM` keeps its meridiem.
	clockToken = regexp.MustCompile(`(?i)(\d{1,2})(?::(\d{2}))?\s?([ap])\.?m\b|(\d{1,2}):(\d{2})(?::\d{2}(?:\.\d+)?)?`)
	// A parenthesized zone may follow `p.m.`; bare zones may not cross a sentence.
	zoneToken  = regexp.MustCompile(`^(?:\.?\s*\(([^()\s]{1,64})\)|\s*(?:((?:UTC|GMT)[+-]\d{1,2}(?::?\d{2})?|[+-]\d{2}:?\d{2}|UTC|GMT|Z)\b|([A-Za-z]+(?:/[A-Za-z0-9_+\-]+)+)|([A-Z]{2,5})\b))`)
	dateBefore = regexp.MustCompile(`(?i)(?:` + dateExpr + `)(?:[\s,]+(?:at\s+|@\s*)?|T)$`)
	dateAfter  = regexp.MustCompile(`(?i)^(?:\s+(?:on\s+)?|,\s*)(?:` + dateExpr + `)`)
	// A numeric slash date beside a clock needs guessing; elsewhere a bare
	// pair such as `3/5 used` is a usage fraction, not time evidence.
	slashBefore = regexp.MustCompile(`\b\d{1,2}/\d{1,2}(?:/\d{2,4})?(?:[\s,]+(?:at\s+|@\s*)?)$`)
	slashAfter  = regexp.MustCompile(`^(?:\s+(?:on\s+)?|,\s*)\d{1,2}/\d{1,2}(?:/\d{2,4})?\b`)
	usageLabel  = regexp.MustCompile(`(?i)^\s+(?:used|left|remaining|requests?|messages?|tokens?|calls?)\b`)
	// Date or zone qualifiers that no clock consumed, and dates that need
	// guessing (day-first, relative, slash with year), leave the time unknown.
	strayQualifier = regexp.MustCompile(`(?i)` + dateExpr + `|\b\d{1,2}/\d{1,2}/\d{2,4}\b|\b\d{1,2}(?:st|nd|rd|th)?\s+(?:` + monthExpr + `)\b|\btomorrow\b|\b(?:UTC|GMT)\b`)
	ianaToken      = regexp.MustCompile(`[A-Za-z]+(?:/[A-Za-z0-9_+\-]+)+`)
	zoneOffset     = regexp.MustCompile(`^(?:UTC|GMT)?([+-])(\d{1,2})(?::?(\d{2}))?$`)
)

// Explicit credit/plan/spend-cap branches of the same pinned builders. They
// count only when no reset was established: an upgrade alternative printed
// beside a valid reset is not an intervention requirement.
var (
	claudeInterventionText = []string{"spend limit", "out of usage credits", "out of usage ·", "doesn't include usage",
		"disabled for your org", "disabled by your admin", "limit is set to $0", "monthly usage limit",
		"contact your admin to increase it", "raise it at", "shared budget"}
	codexInterventionText = []string{"out of credits", "spend cap"}
)

// codexMidnightGuard rejects Codex time-only text observed just after local
// midnight: the producer may have formatted it against the previous date.
const codexMidnightGuard = 5 * time.Minute

var monthAbbrev = map[string]time.Month{"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12}

// processZone resolves the effective timezone of a provider process from its
// final environment. An inherited absent TZ is Chuck's own local zone; an
// unloadable value is unknown (nil).
func processZone(env []string) *time.Location {
	tz, set := "", false
	for _, kv := range env {
		if v, ok := strings.CutPrefix(kv, "TZ="); ok {
			tz, set = v, true
		}
	}
	if env == nil {
		tz, set = os.LookupEnv("TZ")
	}
	if !set {
		return time.Local
	}
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(strings.TrimPrefix(tz, ":"))
	if err != nil {
		return nil
	}
	return loc
}

// localWall returns the unique instant of a wall-clock minute in loc. A
// daylight-saving gap or fold (repeated wall time) is ambiguous.
func localWall(loc *time.Location, y int, mo time.Month, d, hh, mm int) (time.Time, bool) {
	t := time.Date(y, mo, d, hh, mm, 0, 0, loc)
	if ty, tmo, td := t.Date(); ty != y || tmo != mo || td != d || t.Hour() != hh || t.Minute() != mm {
		return time.Time{}, false
	}
	for delta := 15 * time.Minute; delta <= 3*time.Hour; delta += 15 * time.Minute {
		for _, other := range []time.Time{t.Add(delta), t.Add(-delta)} {
			o := other.In(loc)
			if oy, omo, od := o.Date(); oy == y && omo == mo && od == d && o.Hour() == hh && o.Minute() == mm {
				return time.Time{}, false
			}
		}
	}
	return t, true
}

func hour12(h int, meridiem string) (int, bool) {
	if h < 1 || h > 12 {
		return 0, false
	}
	h %= 12
	if strings.EqualFold(meridiem[:1], "p") {
		h += 12
	}
	return h, true
}

// textResetCandidates resolves every clock mentioned in one terminal provider
// failure to the minute-starts it can denote. The producers truncate seconds,
// so a candidate whose displayed minute is still running is plausible.
// Equivalent repeated mentions are one; distinct mentions conflict. Two
// plausible days of one time-only mention are returned for structured evidence
// to corroborate or the caller to reject. present reports that the text carried
// time evidence at all, so an invalid or conflicting mention is never mistaken
// for absence. A date or zone qualifier no clock consumed is unknown too.
func textResetCandidates(backendType, text string, observed time.Time, zone *time.Location) (cands []time.Time, present bool) {
	text = urlToken.ReplaceAllString(text, " ")
	unconsumed := []byte(text)
	for _, m := range clockToken.FindAllStringSubmatchIndex(text, -1) {
		if !clockBounded(text, m[0], m[1]) {
			continue
		}
		present = true
		got, start, end, ok := resolveClock(backendType, text, m, observed, zone)
		if !ok {
			return nil, true
		}
		if cands != nil && !slices.EqualFunc(cands, got, time.Time.Equal) {
			return nil, true // distinct mentions conflict
		}
		cands = got
		for i := start; i < end; i++ {
			unconsumed[i] = ' '
		}
	}
	if present && strayQualifiers(string(unconsumed)) {
		return nil, true
	}
	return cands, present
}

func strayQualifiers(text string) bool {
	if strayQualifier.MatchString(text) {
		return true
	}
	for _, name := range ianaToken.FindAllString(text, -1) {
		if _, err := time.LoadLocation(name); err == nil {
			return true
		}
	}
	return false
}

// clockBounded rejects digits inside a longer number, version, ratio, offset
// or path; `T` may join an ISO date to its time.
func clockBounded(text string, start, end int) bool {
	isDigit := func(b byte) bool { return b >= '0' && b <= '9' }
	if start > 0 {
		prev := text[start-1]
		iso := (prev == 'T' || prev == 't') && start > 1 && isDigit(text[start-2])
		alnum := isDigit(prev) || prev >= 'a' && prev <= 'z' || prev >= 'A' && prev <= 'Z'
		if !iso && (alnum || strings.IndexByte(".:/-+_", prev) >= 0) {
			return false
		}
	}
	return end == len(text) || !isDigit(text[end]) && text[end] != ':' && text[end] != '/'
}

// resolveClock collects one clock's qualifiers — a date just before it and any
// run of zones and dates after it — and returns the text span they consume.
// Every qualifier must agree: differing dates, or zones denoting different
// instants, conflict.
func resolveClock(backendType, text string, m []int, observed time.Time, zone *time.Location) ([]time.Time, int, int, bool) {
	group := func(i int) string {
		if m[2*i] < 0 {
			return ""
		}
		return text[m[2*i]:m[2*i+1]]
	}
	hh, mm := atoi(group(4)), atoi(group(5))
	if meridiem := group(3); meridiem != "" {
		var ok bool
		if hh, ok = hour12(atoi(group(1)), meridiem); !ok {
			return nil, 0, 0, false
		}
		mm = atoi(group(2))
	}
	if hh > 23 || mm > 59 || slashBefore.MatchString(text[:m[0]]) {
		return nil, 0, 0, false
	}
	start, end := m[0], m[1]
	var dates [][]string
	var zones []*time.Location
	if d := dateBefore.FindStringSubmatchIndex(text[:start]); d != nil {
		dates = append(dates, submatches(text, d))
		start = d[0]
	}
	for {
		rest := text[end:]
		if z := zoneToken.FindStringSubmatch(rest); z != nil {
			loc, ok := parseZone(z[1] + z[2] + z[3] + z[4])
			if !ok {
				return nil, 0, 0, false
			}
			zones, end = append(zones, loc), end+len(z[0])
			continue
		}
		if d := dateAfter.FindStringSubmatch(rest); d != nil {
			dates, end = append(dates, d[1:]), end+len(d[0])
			continue
		}
		if s := slashAfter.FindString(rest); s != "" && !usageLabel.MatchString(rest[len(s):]) {
			return nil, 0, 0, false
		}
		break
	}
	if zones == nil {
		zones = []*time.Location{zone}
	}
	var out []time.Time
	for i, loc := range zones {
		got, ok := resolveWall(backendType, dates, hh, mm, observed, loc)
		if !ok || i > 0 && !slices.EqualFunc(out, got, time.Time.Equal) {
			return nil, 0, 0, false
		}
		out = got
	}
	return out, start, end, true
}

func submatches(text string, idx []int) []string {
	out := make([]string, 0, len(idx)/2-1)
	for i := 2; i < len(idx); i += 2 {
		if idx[i] < 0 {
			out = append(out, "")
		} else {
			out = append(out, text[idx[i]:idx[i+1]])
		}
	}
	return out
}

// resolveWall applies a clock's date in loc, then the provider's convention
// for what the text leaves open: Claude's time-only value is today or tomorrow
// within its 24-hour horizon (both plausible is ambiguous); Codex's is the
// process-local date, and a past one is stale, never rolled forward.
func resolveWall(backendType string, dates [][]string, hh, mm int, observed time.Time, loc *time.Location) ([]time.Time, bool) {
	if loc == nil {
		return nil, false
	}
	local := observed.In(loc)
	plausible := func(t time.Time) bool { return t.Add(time.Minute).After(observed) }
	if len(dates) > 0 {
		y, mo, d, ok := parseDate(dates[0], local)
		if !ok {
			return nil, false
		}
		for _, other := range dates[1:] {
			if oy, omo, od, ok := parseDate(other, local); !ok || oy != y || omo != mo || od != d {
				return nil, false
			}
		}
		t, ok := localWall(loc, y, mo, d, hh, mm)
		if !ok || !plausible(t) {
			return nil, false
		}
		return []time.Time{t.UTC()}, true
	}
	if backendType == "codex-acp" {
		y, mo, d := local.Date()
		if local.Sub(time.Date(y, mo, d, 0, 0, 0, 0, loc)) < codexMidnightGuard {
			return nil, false
		}
		t, ok := localWall(loc, y, mo, d, hh, mm)
		if !ok || !t.After(observed) {
			return nil, false
		}
		return []time.Time{t.UTC()}, true
	}
	var out []time.Time
	for day := 0; day <= 1; day++ {
		y, mo, d := local.AddDate(0, 0, day).Date()
		t, ok := localWall(loc, y, mo, d, hh, mm)
		if !ok {
			return nil, false
		}
		if plausible(t) && !t.After(observed.Add(24*time.Hour)) {
			out = append(out, t.UTC())
		}
	}
	return out, len(out) > 0
}

// parseDate reads dateExpr's groups: ISO year-month-day, or an English month
// and day whose omitted year is the zone's current year.
func parseDate(g []string, local time.Time) (int, time.Month, int, bool) {
	if g[0] != "" {
		mo := atoi(g[1])
		return atoi(g[0]), time.Month(mo), atoi(g[2]), mo >= 1 && mo <= 12
	}
	mo, ok := monthAbbrev[strings.ToLower(g[3][:3])]
	y := local.Year()
	if g[5] != "" {
		y = atoi(g[5])
	}
	return y, mo, atoi(g[4]), ok
}

// parseZone accepts UTC/Z, numeric offsets and IANA names. A bare
// abbreviation such as EST is ambiguous and stays unknown.
func parseZone(name string) (*time.Location, bool) {
	switch name {
	case "Z", "UTC", "GMT":
		return time.UTC, true
	}
	if m := zoneOffset.FindStringSubmatch(name); m != nil {
		h, mins := atoi(m[2]), atoi(m[3])
		if h > 14 || mins > 59 {
			return nil, false
		}
		offset := (h*60 + mins) * 60
		if m[1] == "-" {
			offset = -offset
		}
		return time.FixedZone(name, offset), true
	}
	if !strings.Contains(name, "/") {
		return nil, false
	}
	loc, err := time.LoadLocation(name)
	return loc, err == nil
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// rateLimitEvidence accumulates one turn's rejected Claude windows
// (TS-04.R87/R96): the latest reset among applicable blockers, and whether any
// blocker lacked a usable time or provable applicability.
type rateLimitEvidence struct {
	reset   *time.Time
	unknown bool
}

// observe folds one `usage_update._meta["_claude/rateLimit"]` frame into e.
// Pinned native 2.1.286 states resetsAt is Unix epoch seconds. Ordinary
// five-hour/seven-day windows apply to every model; a model-specific window
// applies only when the frame's `_claude/model` is that family. Overage and
// unrecognized rejected windows cannot be shown applicable, and an applicable
// window without a valid time makes the combined reset unknown.
func (e *rateLimitEvidence) observe(params json.RawMessage) {
	var su struct {
		Update struct {
			SessionUpdate string `json:"sessionUpdate"`
			Meta          struct {
				RateLimit *struct {
					Status        string   `json:"status"`
					ResetsAt      *float64 `json:"resetsAt"`
					RateLimitType string   `json:"rateLimitType"`
				} `json:"_claude/rateLimit"`
				Model string `json:"_claude/model"`
			} `json:"_meta"`
		} `json:"update"`
	}
	if len(params) > maxSessionFailureBytes || json.Unmarshal(params, &su) != nil || su.Update.SessionUpdate != "usage_update" {
		return
	}
	rl := su.Update.Meta.RateLimit
	if rl == nil || rl.Status != "rejected" {
		return
	}
	switch rl.RateLimitType {
	case "five_hour", "seven_day":
	case "seven_day_opus", "seven_day_sonnet":
		model := strings.ToLower(su.Update.Meta.Model)
		if model == "" {
			e.unknown = true
			return
		}
		if !strings.Contains(model, strings.TrimPrefix(rl.RateLimitType, "seven_day_")) {
			return // another model's window does not block this one
		}
	default:
		e.unknown = true
		return
	}
	if rl.ResetsAt == nil {
		e.unknown = true
		return
	}
	secs := *rl.ResetsAt
	if math.IsNaN(secs) || math.IsInf(secs, 0) || secs <= 0 || secs > 1<<40 || secs != math.Trunc(secs) {
		e.unknown = true
		return
	}
	if t := time.Unix(int64(secs), 0).UTC(); e.reset == nil || t.After(*e.reset) {
		e.reset = &t
	}
}

// resolveQuotaReset combines structured evidence (second precision) with
// provider text (minute precision). A structured instant inside a candidate's
// displayed minute corroborates it; conflicting, ambiguous or incomplete
// evidence is unknown. Intervention reports explicit credit/plan/spend-cap
// text, counted only when no reset was established.
func resolveQuotaReset(backendType, text string, structured rateLimitEvidence, observed time.Time, zone *time.Location) (*time.Time, bool) {
	switch backendType {
	case "claude-acp":
	case "codex-acp":
		structured = rateLimitEvidence{} // no verified structured carrier
	default:
		return nil, false
	}
	cands, hasText := textResetCandidates(backendType, text, observed, zone)
	var reset *time.Time
	switch {
	case structured.unknown:
	case structured.reset != nil && hasText:
		for _, c := range cands {
			if !structured.reset.Before(c) && structured.reset.Before(c.Add(time.Minute)) {
				reset = structured.reset
			}
		}
	case structured.reset != nil:
		reset = structured.reset
	case len(cands) == 1:
		reset = &cands[0]
	}
	if reset == nil || !reset.After(observed) {
		return nil, quotaInterventionText(backendType, text)
	}
	return reset, false
}

func quotaInterventionText(backendType, text string) bool {
	phrases := claudeInterventionText
	if backendType == "codex-acp" {
		phrases = codexInterventionText
	}
	for _, p := range phrases {
		if strings.Contains(text, p) {
			return true
		}
	}
	return false
}
