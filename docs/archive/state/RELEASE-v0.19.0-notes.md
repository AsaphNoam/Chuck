# Chuck v0.19.0

Status: [published](https://github.com/AsaphNoam/Chuck/releases/tag/v0.19.0) on 2026-10-24,
annotated tag at `308ad23f` after the operator approved the version and asked to release.
Previous release: v0.18.1. This is a minor release: it adds timed and recurring task starts.

## What's new

- **Start a task at a clock time.** A task can wait for a local date and time, alone or together
  with its other prerequisites ("at 17:30 and after task X succeeds"). **Run only on time** is on
  by default: if Chuck was off or the Mac slept through the time, the task parks as missed for
  re-arm or cancel instead of starting late. Turn it off to let it start late once.
- **Recurring schedules.** A schedule creates one ordinary task per occurrence: daily, weekdays,
  chosen weekdays, or every N days/weeks at one or more local times, ending never, after N
  occurrences or on a date. Missed occurrences never create a burst of catch-up tasks. Schedules
  survive restart, sleep and time-zone changes without running twice.
- **Schedules section in the Tasks view** shows each rule in words, the next and last occurrence
  and its state, with Create, Edit, Pause, Resume and Delete. Schedules are also managed over the
  local HTTP API and by agents through scoped tools; an agent can only change schedules it created.
- A schedule whose target agent or project is archived or deleted pauses itself
  visibly and counts toward Tasks needs-attention, instead of quietly creating tasks.

## Fixes and polish

- Cancelling a turn settles pending permission requests first, so permission prompts no longer
  outlive the cancelled turn.
- Annotation instruction fields focus when they open and use the regular text font.
- The shipped `operating-chuck` guidance now tells agents to use start times and schedules
  instead of sleeping, polling or re-creating tasks.

## Verification and limits

`make test` (both Go variants) and `make build` pass after the final product fix; the full UI
suite (1006 tests) and UI build pass on unchanged UI source. `make dist VERSION=0.19.0` passes on a
clean worktree at `308ad23f`, reproducing the committed embedded UI. The architecture review
([receipt](../reviews/architecture/v0.18.1-next.json)) found one Must fix (a deleted project's
schedules kept running) and one Worth fixing (two intermittently failing cancel tests); both are
fixed and verified, and the release gate passed without a second review.

Two review units ship without change review at the operator's choice: annotation input
font/focus and workflow verification cleanup. Native macOS sleep/zone-change behavior,
credentialed Claude/Codex checks, physical-phone behavior and real terminal-driver ownership
remain owed manual gates; automated and fake-provider checks do not establish them.

## Publication receipts

[Release CI 40104625973](https://github.com/AsaphNoam/Chuck/actions/runs/40104625973) and
[main CI 40104624461](https://github.com/AsaphNoam/Chuck/actions/runs/40104624461) passed.
Archive, installer and manifest are attached. The downloaded
`chuck-0.19.0-darwin-arm64.tar.gz` matches manifest size `325146703` and SHA256
`661ccc8b38d779faf6ad4f2b07e5cb9d57d813e843a620261fb30d7b11f89c10`.

Concurrent uncommitted file-viewer tabs design edits remain outside the published source.
