package recurrence

import (
	"errors"
	"testing"
	"time"
)

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatalf("load %s: %v", name, err)
	}
	return loc
}

func norm(t *testing.T, r Rule) Rule {
	t.Helper()
	n, err := r.Normalize()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	return n
}

// nextN follows Next n times from `after` and returns the instants.
func nextN(r Rule, after time.Time, loc *time.Location, n int) []time.Time {
	var out []time.Time
	for i := 0; i < n; i++ {
		nx, ok := r.Next(after, loc)
		if !ok {
			break
		}
		out = append(out, nx)
		after = nx
	}
	return out
}

func fmtAll(ts []time.Time, loc *time.Location) []string {
	var out []string
	for _, t := range ts {
		out = append(out, t.In(loc).Format("2006-01-02 15:04 MST"))
	}
	return out
}

func checkSeq(t *testing.T, got []time.Time, loc *time.Location, want ...string) {
	t.Helper()
	g := fmtAll(got, loc)
	if len(g) != len(want) {
		t.Fatalf("got %v, want %v", g, want)
	}
	for i := range want {
		if g[i] != want[i] {
			t.Fatalf("got %v, want %v", g, want)
		}
	}
}

func TestNormalizeRefusals(t *testing.T) {
	base := Rule{StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: "daily", EndKind: "never"}
	cases := []struct {
		name string
		mod  func(r *Rule)
	}{
		{"no times", func(r *Rule) { r.Times = nil }},
		{"malformed time", func(r *Rule) { r.Times = []string{"9:00"} }},
		{"out of range time", func(r *Rule) { r.Times = []string{"24:00"} }},
		{"repeated time", func(r *Rule) { r.Times = []string{"09:00", "09:00"} }},
		{"unknown frequency", func(r *Rule) { r.Frequency = "monthly" }},
		{"weekly no weekdays", func(r *Rule) { r.Frequency = "weekly" }},
		{"every_n_weeks no weekdays", func(r *Rule) { r.Frequency = "every_n_weeks"; r.Interval = 2 }},
		{"unknown weekday", func(r *Rule) { r.Frequency = "weekly"; r.Weekdays = []string{"funday"} }},
		{"duplicate weekday", func(r *Rule) { r.Frequency = "weekly"; r.Weekdays = []string{"mon", "mon"} }},
		{"every_n_days interval 0", func(r *Rule) { r.Frequency = "every_n_days" }},
		{"every_n_weeks interval 0", func(r *Rule) { r.Frequency = "every_n_weeks"; r.Weekdays = []string{"mon"} }},
		{"malformed start", func(r *Rule) { r.StartDate = "2026-1-5" }},
		{"malformed end date", func(r *Rule) { r.EndKind = "until"; r.EndDate = "soon" }},
		{"count zero", func(r *Rule) { r.EndKind = "count" }},
		{"until before start", func(r *Rule) { r.EndKind = "until"; r.EndDate = "2026-10-31" }},
		{"unknown end kind", func(r *Rule) { r.EndKind = "forever" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := base
			c.mod(&r)
			if _, err := r.Normalize(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("err = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestNormalizeCanonicalizes(t *testing.T) {
	got := norm(t, Rule{
		StartDate: "2026-11-01", Times: []string{"17:30", "09:00"}, Frequency: "weekly",
		Weekdays: []string{"sun", "mon"}, Interval: 7, EndKind: "never", EndCount: 4, EndDate: "2030-01-01",
	})
	if got.Times[0] != "09:00" || got.Weekdays[0] != "mon" || got.Interval != 1 || got.EndCount != 0 || got.EndDate != "" {
		t.Fatalf("not canonical: %+v", got)
	}
	d := norm(t, Rule{StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: "daily", Weekdays: []string{"mon"}, EndKind: "never"})
	if len(d.Weekdays) != 0 {
		t.Fatalf("daily kept weekdays: %+v", d)
	}
}

func TestNextWeeklySunday(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-20", Times: []string{"09:00"}, Frequency: "weekly", Weekdays: []string{"sun"}, EndKind: "never"})
	after := time.Date(2026, 10, 20, 0, 0, 0, 0, ny)
	checkSeq(t, nextN(r, after, ny, 3), ny, "2026-10-25 09:00 EDT", "2026-11-01 09:00 EST", "2026-11-08 09:00 EST")
}

func TestNextWeekdaysSkipsWeekend(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-19", Times: []string{"09:00"}, Frequency: "weekdays", EndKind: "never"})
	after := time.Date(2026, 10, 22, 12, 0, 0, 0, ny) // Thursday
	checkSeq(t, nextN(r, after, ny, 3), ny, "2026-10-23 09:00 EDT", "2026-10-26 09:00 EDT", "2026-10-27 09:00 EDT")
}

func TestNextEveryNDays(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-01", Times: []string{"09:00"}, Frequency: "every_n_days", Interval: 3, EndKind: "never"})
	after := time.Date(2026, 9, 1, 0, 0, 0, 0, ny)
	checkSeq(t, nextN(r, after, ny, 3), ny, "2026-10-01 09:00 EDT", "2026-10-04 09:00 EDT", "2026-10-07 09:00 EDT")
	// Mid-sequence resume lands on the grid, not N days after `after`.
	checkSeq(t, nextN(r, time.Date(2026, 10, 5, 0, 0, 0, 0, ny), ny, 1), ny, "2026-10-07 09:00 EDT")
}

func TestNextEveryNWeeks(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	// Start on Wednesday 2026-10-07: Monday of that week is week 0, so Mon 10-05
	// is before the start and skipped; Thu 10-08 is the first occurrence.
	r := norm(t, Rule{StartDate: "2026-10-07", Times: []string{"08:00"}, Frequency: "every_n_weeks", Interval: 2, Weekdays: []string{"thu", "mon"}, EndKind: "never"})
	after := time.Date(2026, 10, 1, 0, 0, 0, 0, ny)
	checkSeq(t, nextN(r, after, ny, 5), ny,
		"2026-10-08 08:00 EDT", "2026-10-19 08:00 EDT", "2026-10-22 08:00 EDT", "2026-11-02 08:00 EST", "2026-11-05 08:00 EST")
}

func TestNextUntilInclusive(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-22", Times: []string{"09:00"}, Frequency: "daily", EndKind: "until", EndDate: "2026-10-24"})
	got := nextN(r, time.Date(2026, 10, 1, 0, 0, 0, 0, ny), ny, 10)
	checkSeq(t, got, ny, "2026-10-22 09:00 EDT", "2026-10-23 09:00 EDT", "2026-10-24 09:00 EDT")
	if _, ok := r.Next(got[2], ny); ok {
		t.Fatal("occurrence after inclusive end date")
	}
}

func TestNextDailyAcrossDST(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-03-01", Times: []string{"09:00"}, Frequency: "daily", EndKind: "never"})
	checkSeq(t, nextN(r, time.Date(2026, 3, 6, 12, 0, 0, 0, ny), ny, 4), ny,
		"2026-03-07 09:00 EST", "2026-03-08 09:00 EDT", "2026-03-09 09:00 EDT", "2026-03-10 09:00 EDT")
	checkSeq(t, nextN(r, time.Date(2026, 10, 30, 12, 0, 0, 0, ny), ny, 4), ny,
		"2026-10-31 09:00 EDT", "2026-11-01 09:00 EST", "2026-11-02 09:00 EST", "2026-11-03 09:00 EST")
}

func TestNextSpringForwardGap(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-03-01", Times: []string{"02:30"}, Frequency: "daily", EndKind: "never"})
	got := nextN(r, time.Date(2026, 3, 7, 0, 0, 0, 0, ny), ny, 3)
	checkSeq(t, got, ny, "2026-03-07 02:30 EST", "2026-03-08 03:00 EDT", "2026-03-09 02:30 EDT")
	if !got[1].Equal(time.Date(2026, 3, 8, 7, 0, 0, 0, time.UTC)) {
		t.Fatalf("gap occurrence = %v, want the transition instant 07:00Z", got[1].UTC())
	}
}

func TestNextFallBackRepeat(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-01", Times: []string{"01:30"}, Frequency: "daily", EndKind: "never"})
	first, ok := r.Next(time.Date(2026, 10, 31, 12, 0, 0, 0, ny), ny)
	if !ok {
		t.Fatal("no occurrence")
	}
	checkSeq(t, []time.Time{first}, ny, "2026-11-01 01:30 EDT")
	second, _ := r.Next(first, ny)
	checkSeq(t, []time.Time{second}, ny, "2026-11-02 01:30 EST")
}

func TestNextLondonDST(t *testing.T) {
	lon := mustLoc(t, "Europe/London")
	r := norm(t, Rule{StartDate: "2026-03-01", Times: []string{"01:30", "09:00"}, Frequency: "daily", EndKind: "never"})
	// 2026-03-29 01:00 GMT jumps to 02:00 BST, so 01:30 is skipped.
	got := nextN(r, time.Date(2026, 3, 28, 12, 0, 0, 0, lon), lon, 3)
	checkSeq(t, got, lon, "2026-03-29 02:00 BST", "2026-03-29 09:00 BST", "2026-03-30 01:30 BST")
}

func TestNextMultipleTimesOrderedAndStrict(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2026-10-20", Times: []string{"17:30", "09:00"}, Frequency: "daily", EndKind: "never"})
	got := nextN(r, time.Date(2026, 10, 22, 8, 0, 0, 0, ny), ny, 4)
	checkSeq(t, got, ny, "2026-10-22 09:00 EDT", "2026-10-22 17:30 EDT", "2026-10-23 09:00 EDT", "2026-10-23 17:30 EDT")
	// Strictly after: asking at an occurrence instant returns the following one.
	nx, _ := r.Next(got[0], ny)
	if !nx.Equal(got[1]) {
		t.Fatalf("Next(occurrence) = %v, want %v", nx, got[1])
	}
}

func TestNextBeforeStartDate(t *testing.T) {
	ny := mustLoc(t, "America/New_York")
	r := norm(t, Rule{StartDate: "2027-01-04", Times: []string{"09:00"}, Frequency: "weekdays", EndKind: "never"})
	checkSeq(t, nextN(r, time.Date(2026, 1, 1, 0, 0, 0, 0, ny), ny, 1), ny, "2027-01-04 09:00 EST")
}

func TestDescribe(t *testing.T) {
	cases := []struct {
		r    Rule
		want string
	}{
		{Rule{StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: "weekly", Weekdays: []string{"sun"}, EndKind: "count", EndCount: 3}, "Every Sunday at 9:00 AM, 3 times"},
		{Rule{StartDate: "2026-11-01", Times: []string{"17:30", "09:00"}, Frequency: "weekdays", EndKind: "never"}, "Weekdays at 9:00 AM and 5:30 PM starting 2026-11-01"},
		{Rule{StartDate: "2026-11-01", Times: []string{"08:00"}, Frequency: "every_n_weeks", Interval: 2, Weekdays: []string{"mon", "thu"}, EndKind: "until", EndDate: "2026-12-31"}, "Every 2 weeks on Mon and Thu at 8:00 AM, until 2026-12-31"},
		{Rule{StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: "daily", EndKind: "never"}, "Daily at 9:00 AM starting 2026-11-01"},
		{Rule{StartDate: "2026-11-01", Times: []string{"09:00"}, Frequency: "every_n_days", Interval: 3, EndKind: "count", EndCount: 1}, "Every 3 days at 9:00 AM, 1 time"},
	}
	for _, c := range cases {
		if got := norm(t, c.r).Describe(); got != c.want {
			t.Errorf("Describe = %q, want %q", got, c.want)
		}
	}
}
