package runtime

import (
	"encoding/json"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Reset evidence follows independent terminal quota classification (TS-04.R96).
// Grammars are the pinned producers' verified formatters, not a generic date
// parser: Claude native 2.1.286's rejected-error builder (`Acn`/`iu`, English
// hour/minute with an IANA zone suffix) and Codex 0.159.3's
// UsageLimitReachedError suffix (process-local time, no zone). Anything
// unsupported, stale, ambiguous or conflicting stays unknown.

var (
	claudeResetClause = regexp.MustCompile(`\x{00b7} resets (?:([A-Z][a-z]{2}) (\d{1,2}), (?:(\d{4}), )?)?(\d{1,2})(?::(\d{2}))?(am|pm) \(([A-Za-z0-9_+\-/]+)\)`)
	codexResetClause  = regexp.MustCompile(`[Tt]ry again at (?:([A-Z][a-z]{2}) (\d{1,2})(st|nd|rd|th), (\d{4}) )?(\d{1,2}):(\d{2}) (AM|PM)\.`)
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

var monthAbbrev = map[string]time.Month{"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6,
	"Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12}

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
	if strings.EqualFold(meridiem, "pm") {
		h += 12
	}
	return h, true
}

// uniqueMatch returns the only distinct clause in text; repeated identical
// clauses (title echoed in details) are one candidate.
func uniqueMatch(re *regexp.Regexp, text string) ([]string, bool) {
	all := re.FindAllStringSubmatch(text, -1)
	if len(all) == 0 {
		return nil, false
	}
	for _, m := range all[1:] {
		if m[0] != all[0][0] {
			return nil, false
		}
	}
	return all[0], true
}

// claudeTextCandidates parses the rejected-error reset clause into the
// minute-starts it can denote. The producer truncates seconds, so a candidate
// whose displayed minute is still running is plausible. Time-only text may
// mean today or tomorrow within the producer's 24-hour horizon; two plausible
// days are ambiguous. present reports that the text carried reset evidence at
// all, so a conflicting or unparseable clause is never mistaken for absence.
func claudeTextCandidates(text string, observed time.Time) (cands []time.Time, present bool) {
	if !claudeResetClause.MatchString(text) {
		return nil, false
	}
	m, ok := uniqueMatch(claudeResetClause, text)
	if !ok {
		return nil, true
	}
	loc, err := time.LoadLocation(m[7])
	if err != nil {
		return nil, true
	}
	hh, ok := hour12(atoi(m[4]), m[6])
	if !ok {
		return nil, true
	}
	mm := 0
	if m[5] != "" {
		if mm = atoi(m[5]); mm < 1 || mm > 59 { // the producer omits ":00"
			return nil, true
		}
	}
	plausible := func(t time.Time) bool {
		return t.Add(time.Minute).After(observed) && !t.After(observed.Add(24*time.Hour))
	}
	local := observed.In(loc)
	if m[1] == "" {
		for day := 0; day <= 1; day++ {
			y, mo, d := local.AddDate(0, 0, day).Date()
			t, ok := localWall(loc, y, mo, d, hh, mm)
			if !ok {
				return nil, true
			}
			if plausible(t) {
				cands = append(cands, t.UTC())
			}
		}
		return cands, true
	}
	mo, known := monthAbbrev[m[1]]
	if !known {
		return nil, true
	}
	y := local.Year()
	if m[3] != "" {
		y = atoi(m[3])
	}
	t, ok := localWall(loc, y, mo, atoi(m[2]), hh, mm)
	if !ok || !t.Add(time.Minute).After(observed) {
		return nil, true
	}
	return []time.Time{t.UTC()}, true
}

// claudeTextReset is the text-only reading: exactly one plausible candidate,
// which must lie in the future.
func claudeTextReset(text string, observed time.Time) (time.Time, bool) {
	cands, _ := claudeTextCandidates(text, observed)
	if len(cands) != 1 || !cands[0].After(observed) {
		return time.Time{}, false
	}
	return cands[0], true
}

// codexTextReset parses the `try again at` suffix in the provider process's
// effective zone. A time-only value means today; a past one is stale, never
// rolled to tomorrow.
func codexTextReset(text string, observed time.Time, loc *time.Location) (time.Time, bool) {
	if loc == nil {
		return time.Time{}, false
	}
	m, ok := uniqueMatch(codexResetClause, text)
	if !ok {
		return time.Time{}, false
	}
	hh, ok := hour12(atoi(m[5]), m[7])
	mm := atoi(m[6])
	if !ok || mm > 59 {
		return time.Time{}, false
	}
	local := observed.In(loc)
	y, mo, d := local.Date()
	if m[1] == "" {
		if local.Sub(time.Date(y, mo, d, 0, 0, 0, 0, loc)) < codexMidnightGuard {
			return time.Time{}, false
		}
	} else {
		var known bool
		if mo, known = monthAbbrev[m[1]]; !known {
			return time.Time{}, false
		}
		d, y = atoi(m[2]), atoi(m[4])
		if ordinalSuffix(d) != m[3] {
			return time.Time{}, false
		}
	}
	t, ok := localWall(loc, y, mo, d, hh, mm)
	if !ok || !t.After(observed) {
		return time.Time{}, false
	}
	return t.UTC(), true
}

func ordinalSuffix(day int) string {
	if day%100 >= 11 && day%100 <= 13 {
		return "th"
	}
	switch day % 10 {
	case 1:
		return "st"
	case 2:
		return "nd"
	case 3:
		return "rd"
	}
	return "th"
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
	var cands []time.Time
	var hasText bool
	switch backendType {
	case "claude-acp":
		cands, hasText = claudeTextCandidates(text, observed)
	case "codex-acp":
		hasText = codexResetClause.MatchString(text)
		if t, ok := codexTextReset(text, observed, zone); ok {
			cands = []time.Time{t}
		}
		structured = rateLimitEvidence{} // no verified structured carrier
	default:
		return nil, false
	}
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
