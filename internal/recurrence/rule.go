// Package recurrence is the closed recurrence rule for scheduled task starts:
// a start date, local times of day, a frequency and an end. It is pure; callers
// own persistence, occurrence counting and the clock.
package recurrence

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Frequencies and end kinds a Rule may name.
const (
	FreqDaily      = "daily"
	FreqWeekdays   = "weekdays"
	FreqWeekly     = "weekly"
	FreqEveryNDays = "every_n_days"
	FreqEveryNWeek = "every_n_weeks"

	EndNever = "never"
	EndCount = "count"
	EndUntil = "until"
)

// maxScanDays bounds the day loop in Next. Every frequency jumps straight to
// its next eligible day, so a valid rule finds an occurrence in far fewer steps.
const maxScanDays = 5000

// ErrInvalid is wrapped by every Normalize failure.
var ErrInvalid = errors.New("invalid recurrence rule")

// Rule is persisted and sent over HTTP, so its JSON tags are a contract.
type Rule struct {
	StartDate string   `json:"start_date"`         // YYYY-MM-DD
	Times     []string `json:"times"`              // HH:MM 24h local wall clock
	Frequency string   `json:"frequency"`          // daily|weekdays|weekly|every_n_days|every_n_weeks
	Weekdays  []string `json:"weekdays,omitempty"` // "mon".."sun"; weekly and every_n_weeks
	Interval  int      `json:"interval,omitempty"` // N for every_n_*
	EndKind   string   `json:"end_kind"`           // never|count|until
	EndCount  int      `json:"end_count,omitempty"`
	EndDate   string   `json:"end_date,omitempty"` // YYYY-MM-DD, inclusive
}

var weekdayNames = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

const (
	dateLayout = "2006-01-02"
	timeLayout = "15:04"
)

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil || len(s) != len(dateLayout) {
		return time.Time{}, invalid("date %q must be YYYY-MM-DD", s)
	}
	return t, nil
}

// parseClock returns the hour and minute of a strict HH:MM string.
func parseClock(s string) (hour, minute int, err error) {
	t, perr := time.Parse(timeLayout, s)
	if perr != nil || len(s) != len(timeLayout) {
		return 0, 0, invalid("time %q must be HH:MM", s)
	}
	return t.Hour(), t.Minute(), nil
}

func weekdayIndex(name string) int {
	for i, n := range weekdayNames {
		if n == name {
			return i
		}
	}
	return -1
}

// mondayIndex maps Go's Sunday-first weekday to 0 (Monday) .. 6 (Sunday).
func mondayIndex(d time.Time) int { return (int(d.Weekday()) + 6) % 7 }

// Normalize validates the rule and returns a canonical copy.
func (r Rule) Normalize() (Rule, error) {
	out := Rule{StartDate: r.StartDate, Frequency: r.Frequency, EndKind: r.EndKind}
	start, err := parseDate(r.StartDate)
	if err != nil {
		return Rule{}, err
	}

	if len(r.Times) == 0 {
		return Rule{}, invalid("at least one time is required")
	}
	seen := map[string]bool{}
	for _, s := range r.Times {
		if _, _, err := parseClock(s); err != nil {
			return Rule{}, err
		}
		if seen[s] {
			return Rule{}, invalid("time %q is repeated", s)
		}
		seen[s] = true
		out.Times = append(out.Times, s)
	}
	sort.Strings(out.Times) // zero-padded HH:MM sorts chronologically

	switch r.Frequency {
	case FreqDaily, FreqWeekdays:
		out.Interval = 1
	case FreqEveryNDays:
		if r.Interval < 1 {
			return Rule{}, invalid("interval must be at least 1")
		}
		out.Interval = r.Interval
	case FreqWeekly, FreqEveryNWeek:
		if r.Frequency == FreqWeekly {
			out.Interval = 1
		} else if r.Interval < 1 {
			return Rule{}, invalid("interval must be at least 1")
		} else {
			out.Interval = r.Interval
		}
		if len(r.Weekdays) == 0 {
			return Rule{}, invalid("at least one weekday is required")
		}
		picked := map[int]bool{}
		for _, name := range r.Weekdays {
			i := weekdayIndex(name)
			if i < 0 {
				return Rule{}, invalid("unknown weekday %q", name)
			}
			if picked[i] {
				return Rule{}, invalid("weekday %q is repeated", name)
			}
			picked[i] = true
		}
		for i, name := range weekdayNames {
			if picked[i] {
				out.Weekdays = append(out.Weekdays, name)
			}
		}
	default:
		return Rule{}, invalid("unknown frequency %q", r.Frequency)
	}

	switch r.EndKind {
	case EndNever:
	case EndCount:
		if r.EndCount < 1 {
			return Rule{}, invalid("end count must be at least 1")
		}
		out.EndCount = r.EndCount
	case EndUntil:
		end, err := parseDate(r.EndDate)
		if err != nil {
			return Rule{}, err
		}
		if end.Before(start) {
			return Rule{}, invalid("end date %s is before start date %s", r.EndDate, r.StartDate)
		}
		out.EndDate = r.EndDate
	default:
		return Rule{}, invalid("unknown end kind %q", r.EndKind)
	}
	return out, nil
}

// Next returns the first occurrence strictly after `after`, computed in loc. It
// ignores EndCount; callers count occurrences themselves. The rule must be
// normalized.
func (r Rule) Next(after time.Time, loc *time.Location) (time.Time, bool) {
	at, _, ok := r.NextWall(after, loc)
	return at, ok
}

// WallLayout names one occurrence by its calendar date and rule time, which
// stay the same occurrence in any zone.
const WallLayout = "2006-01-02T15:04"

// NextWall is Next that also names the occurrence in WallLayout.
func (r Rule) NextWall(after time.Time, loc *time.Location) (time.Time, string, bool) {
	start, err := parseDate(r.StartDate)
	if err != nil {
		return time.Time{}, "", false
	}
	var end time.Time
	hasEnd := r.EndKind == EndUntil
	if hasEnd {
		if end, err = parseDate(r.EndDate); err != nil {
			return time.Time{}, "", false
		}
	}
	type clock struct{ h, m int }
	var clocks []clock
	for _, s := range r.Times {
		h, m, err := parseClock(s)
		if err != nil {
			return time.Time{}, "", false
		}
		clocks = append(clocks, clock{h, m})
	}

	// Days are UTC-midnight values standing for calendar dates, so day
	// arithmetic is exact and never touches DST.
	y, mo, d := after.In(loc).Date()
	day := time.Date(y, mo, d, 0, 0, 0, 0, time.UTC)
	if day.Before(start) {
		day = start
	}
	for i := 0; i < maxScanDays; i++ {
		if hasEnd && day.After(end) {
			break
		}
		skip := r.daysToEligible(day, start)
		if skip == 0 {
			for _, c := range clocks {
				if t := wallInstant(day, c.h, c.m, loc); t.After(after) {
					return t, time.Date(day.Year(), day.Month(), day.Day(), c.h, c.m, 0, 0, time.UTC).Format(WallLayout), true
				}
			}
			skip = 1
		}
		day = day.AddDate(0, 0, skip)
	}
	return time.Time{}, "", false
}

// WallAt returns the instant of a WallLayout occurrence in loc, resolving a
// skipped or repeated wall time as Next does.
func WallAt(wall string, loc *time.Location) (time.Time, error) {
	t, err := time.Parse(WallLayout, wall)
	if err != nil {
		return time.Time{}, invalid("occurrence %q must be %s", wall, WallLayout)
	}
	return wallInstant(t, t.Hour(), t.Minute(), loc), nil
}

// daysToEligible is 0 when day (>= start) is an occurrence day, otherwise the
// number of days to the next day worth checking.
func (r Rule) daysToEligible(day, start time.Time) int {
	n := int(day.Sub(start) / (24 * time.Hour))
	interval := max(r.Interval, 1)
	switch r.Frequency {
	case FreqDaily:
		return 0
	case FreqWeekdays:
		if wd := mondayIndex(day); wd >= 5 {
			return 7 - wd
		}
		return 0
	case FreqEveryNDays:
		if rem := n % interval; rem != 0 {
			return interval - rem
		}
		return 0
	}
	// weekly and every_n_weeks: Monday-start weeks counted from the start week.
	wd := mondayIndex(day)
	week := (n + mondayIndex(start)) / 7
	if rem := week % interval; rem != 0 {
		return (interval-rem)*7 - wd // to Monday of the next eligible week
	}
	for _, name := range r.Weekdays {
		if weekdayIndex(name) == wd {
			return 0
		}
	}
	return 1
}

// wallInstant returns the instant of the local wall time h:m on the calendar
// date of day. A repeated wall time resolves to its earlier instant; a skipped
// one resolves to the transition instant, the first valid minute after the gap.
func wallInstant(day time.Time, h, m int, loc *time.Location) time.Time {
	asUTC := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, time.UTC)
	var best, gap time.Time
	found := false
	// Offsets a day either side cover any transition affecting this wall time.
	for _, probe := range []time.Time{asUTC.Add(-24 * time.Hour), asUTC.Add(24 * time.Hour)} {
		_, off := probe.In(loc).Zone()
		cand := asUTC.Add(-time.Duration(off) * time.Second)
		w := cand.In(loc)
		if w.Hour() == h && w.Minute() == m && w.Day() == day.Day() {
			if !found || cand.Before(best) {
				best, found = cand, true
			}
		} else if gap.IsZero() || cand.After(gap) {
			gap = cand
		}
	}
	if found {
		return best
	}
	// In a gap the later candidate lies just past the transition; its zone
	// period starts exactly at it.
	start, _ := gap.In(loc).ZoneBounds()
	return start
}

// Describe returns the rule in short plain English.
func (r Rule) Describe() string {
	interval := max(r.Interval, 1)
	var days []string
	for _, name := range r.Weekdays {
		days = append(days, strings.ToUpper(name[:1])+name[1:])
	}

	var s string
	switch r.Frequency {
	case FreqDaily:
		s = "Daily"
	case FreqWeekdays:
		s = "Weekdays"
	case FreqEveryNDays:
		if interval == 1 {
			s = "Every day"
		} else {
			s = fmt.Sprintf("Every %d days", interval)
		}
	case FreqWeekly, FreqEveryNWeek:
		switch {
		case interval > 1:
			s = fmt.Sprintf("Every %d weeks on %s", interval, joinAnd(days))
		case len(r.Weekdays) == 1:
			s = "Every " + fullWeekday(r.Weekdays[0])
		default:
			s = "Weekly on " + joinAnd(days)
		}
	default:
		s = r.Frequency
	}

	var times []string
	for _, t := range r.Times {
		times = append(times, clock12(t))
	}
	s += " at " + joinAnd(times)

	switch r.EndKind {
	case EndNever:
		s += " starting " + r.StartDate
	case EndCount:
		if r.EndCount == 1 {
			s += ", 1 time"
		} else {
			s += fmt.Sprintf(", %d times", r.EndCount)
		}
	case EndUntil:
		s += ", until " + r.EndDate
	}
	return s
}

func fullWeekday(short string) string {
	if i := weekdayIndex(short); i >= 0 {
		return time.Weekday((i + 1) % 7).String()
	}
	return short
}

// clock12 renders HH:MM as "9:00 AM"; malformed input is returned unchanged.
func clock12(s string) string {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return s
	}
	return t.Format("3:04 PM")
}

func joinAnd(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	}
	return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
}
