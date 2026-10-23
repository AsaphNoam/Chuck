# Timed and recurring task starts

**State:** Waiting to start
**Why:** Operator request on 2026-10-23 (`/design-feature`): "start this at 17:30" and repeating
starts such as "every day at 9 AM" or "every Sunday for 3 weeks".
**Relevant requirements:** FS-16.R51–R58, A35–A40; TS-10.R54–R60 and §3 timed-start shapes;
TS-03 route inventory; INV §1, §5, §7, §9, §15

## Outcome

A task can wait for a clock time as one more prerequisite. A project schedule creates an ordinary
task at each occurrence of a small recurrence rule. Run only on time (default on) decides whether a
time missed while Chuck was off or asleep is skipped or runs once late. Overlapping occurrences
still run. People and agents can create both kinds of start.

## Included work

- `time` arms, with lateness parking through the existing R8 path and a `late_start` marker.
- `task_schedules`: daily, weekdays, weekly days, or every N days/weeks, with times of day and an
  end (never, after N, or after a date). Local wall-clock time follows the daylight-saving rules in
  TS-10.R56.
- Due processing on the existing two-second dispatcher tick: occurrence creation exactly once,
  missed-occurrence coalescing, pause/resume/edit/delete, and auto-pause when a target is unusable.
- HTTP routes, MCP tools (`create_schedule`, `pause_schedule`, `resume_schedule`, `delete_schedule`;
  `start_at` on `create_task`), `schedule_update` SSE, and a desktop Tasks **Schedules** section.
- Not included: cron syntax, sub-daily intervals, webhooks, waking the Mac, scheduled pipeline runs,
  phone schedule UI, an occurrence history ledger, and live time-zone change detection (TS-10 §5).

## How we will know it works

FS-16.A35–A39 in `internal/state`, `internal/server` and `internal/messaging` tests, using the
`timeNow` seam and fixed time zones for daylight-saving cases. FS-16.A40 needs UI tests plus a rendered
Chromium journey in all three skins. Both Go test variants, plus focused `-race` on the dispatcher
and state packages.

## Waiting on

Nothing.
