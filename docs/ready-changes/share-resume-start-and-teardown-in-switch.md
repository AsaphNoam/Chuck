# Share the resume start and teardown in Switch

**State:** Waiting to start
**Why:** Direct `/design-feature` request on 2026-11-13. Fourth of five single-owner refactors from
the 60-day invariant review: launch and resume each converge on one function, while Switch is
still a hand-written third sequence with partial cleanup.
**Relevant requirements:** TS-01.R43; INV §2, §4, §23. Preserves FS-01 switch, resume and rollback
behavior as specified; no feature requirement changes.

## Outcome

Every path that starts a resumed runtime uses one start step and one teardown, so a failure at
any branch leaves no stale registration or second process.

## Included work

One shared compose-start-persist-or-teardown step for explicit resume, wakes, the Switch target
and the Switch rollback; `teardownAgentRegistration` in place of the hand-written cleanup subsets
in `switch.go` and in the launch and resume composers; stopping an already-started target before
rollback.

Not included: changing Switch's ordering, its API results or `history_handoff`; the lifecycle
claim and archive-lease gates; the Think Tank launch path, whose lifecycle claim was not traced
during design and is worth a separate check.

Evidence read for the design: `handleSwitch` cleans the old registration with
`cleanupMessagingMCP` + `cleanupHookSettings` only, `rollbackSwitch` repeats that subset, the
branch after a successful `registry.Resume` rolls back without stopping the target
(`internal/server/switch.go`), and compose's partial failures hand-roll cleanup
(`launch.go`, `resume.go`).

## How we will know it works

The TS-01.R43 checks: existing launch, resume and switch suites, plus a failure injected at each
branch after registration for launch, resume, Switch target and Switch rollback.

## Waiting on

Nothing.
