# Chuck — Handoff archive through 2026-10-24 (v0.19.0)

Settled state moved out of the live handoff when v0.19.0 was released. Non-authoritative history;
see the live handoff and specifications for current state.

## Superseded current position (v0.18.1)

- **Published:** [v0.18.1](https://github.com/AsaphNoam/Chuck/releases/tag/v0.18.1),
  2026-10-23, source `dc979057`; release CI 39959315460 passed. Published archive size and
  SHA256 match the manifest. Main CI 39959316204 failed twice; REL-CI-1 is repaired locally
  by `1b998b24`, with both full Go variants passing. The repair has not been pushed.
- **Released repair:** fixes-only `v0.18.1`. Product repair `69082b39` refuses unreadable or
  malformed dashboard PID files before offline reindex opens SQLite. Approved test-only corrections
  are `001ba8b5`. The operator approved the version, 23 unpushed commits and annotated tag.
- **Review state:** ASW/ASF and CR-A1–A7 are closed. The combined next-release architecture review
  covers `eb56b0c2` with exact later coverage/repair updates in
  [its receipt](../archive/reviews/architecture/v0.18.0-next.json).
  ARCH-1 and REL-VERIFY-1 are verified closed; no second architecture review is needed.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`. Clean release verification uses `/private/tmp/chuck-release-v0.18.1`.
- **Future work:** direct-action migration remains paused on its transport blocker; passive quota
  snapshots remain TBD. No future implementation is selected by this release.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Existing cancellation-race/native/device limitations remain in the owed gates. Prior failing
  bootstrap/recovery fixture runs are retained in the release notes, not represented as passed.

## Closed review findings

ARCH-1 is repaired by `69082b39`; REL-VERIFY-1 is closed by the explicitly approved
test corrections `001ba8b5`. Both full Go variants pass. Original reports and prior failed runs
remain in the archive/release notes; exact repair and affected coverage are in the
[architecture receipt](../archive/reviews/architecture/v0.18.0-next.json).
ARCH-v019-1/2 (v0.19.0 candidate review) are closed by `4b215814` and `bdaa43e9`; closure
evidence is in [their receipt](../archive/reviews/architecture/v0.18.1-next.json).

## Settled changelog

- **2026-10-24 — Fix: v0.19.0 architecture findings closed (ARCH-v019-1 INV §2, ARCH-v019-2
  INV §17):** a deleted project's schedule now pauses with "the project was deleted" when due
  and resume refuses it (FS-16.R57, TS-10.R59 updated); the new server test fails on the old
  code. The two cancel tests wait for the turn gate after `turn_end`; `-race -count=20` went
  from 38/40 failures to passing, unraced `-count=100` passes. The old line-37 CI symptom predates
  the fake-peer cancel scoping fix. `make test` (both variants) and `make build` pass. No new
  review unit; the release gate is unblocked.
- **2026-10-24 — Architecture review: v0.19.0 candidate `c1979ed3`:** complete; receipt
  `v0.18.1-next.json`. ARCH-v019-1 (Must fix, deleted project keeps schedules running) and
  ARCH-v019-2 (Worth fixing, pre-existing intermittent cancel/turn_end CI tests) were reported.
  Schedule crash/outage/migration/cost probes found no duplicate or lost occurrence.
- **2026-10-24 — Fix: timed and recurring task starts closed (TTS-1 INV §1, TTS-2 INV §7/§8/§11,
  TTS-3 INV §10):** schedules store their pending occurrence's date and rule time (migration 62)
  and rebase it to the process zone on the first due pass, so a restart in another zone runs at
  that zone's wall time without replay (TS-10.R56 updated). Failed schedule reads show error/retry
  or stale cached data instead of zero attention. Re-arm can set a new start time with Run only on
  time. New Go zone tests (west/east) and four UI read-failure tests fail on the old code; the
  re-arm test fails without its fix. `make test` (both variants), 1006 UI tests and UI build pass.
  No native macOS zone change or live-provider receipt is claimed.

- **2026-10-23 — REL-CI-1 closed (INV §5/§17):** `1b998b24` settles pending permissions
  before cooperative cancellation, serializes concurrent cancellations and rechecks the captured
  turn before notification. The strengthened wire-order regression fails on the original code;
  controlled publication/claim barriers replace its scheduling sleep. Three focused permission/
  cancellation runs and ten focused race runs pass. `make test` passes both full variants with
  approved loopback access; the first sandboxed attempt was stopped after loopback bind failures.
  `make build` and diff checks pass. FS-03.A5 records the coverage. REL-CI is closed; TTS-1–3
  remain open. No live-provider receipt or new CI run is claimed.
- **2026-10-23 — v0.18.1 published:** explicit operator approval covered the 23-commit push
  and tag at `dc979057`. Release CI 39959315460 passed; archive, manifest and installer are
  attached, and downloaded archive checksum/size match. Main CI failed twice on REL-CI-1;
  the failure is retained as an open finding. Manual acceptance gates remain owed.
- **2026-10-23 — Implementation:** timed and recurring task starts shipped (FS-16.R51–R58/A35–A40,
  TS-10.R54–R60): clock-time arms with Run only on time, closed-rule schedules that create
  ordinary tasks, HTTP routes, agent tools and the desktop Schedules section. FS-16 and TS-10 are
  now Current. Waiting for review.

- **2026-10-23 — Review:** timed and recurring task starts remain open for TTS-1–3: timezone
  restart scheduling, silent schedule-read failures, and missing time re-arm controls. Focused
  automated checks pass; the timezone regression reproduces. Fix routing is recorded with the unit.

- **2026-10-23 — Release preparation:** ARCH-1 repair `69082b39`, exact post-review coverage,
  truthful failed-check records and approved fixture corrections `001ba8b5` are committed.
  Both full Go variants, focused repeats and static checks pass; ARCH-1 (INV §7/§9) and
  REL-VERIFY-1 (INV §17) are closed. Settled campaign history moved out of the live handoff.
  No publication or manual acceptance closure is claimed.
