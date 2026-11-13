# Single-owner turn state

**State:** Waiting to start
**Why:** Direct `/design-feature` request on 2026-11-13. Fifth of five single-owner refactors from
the 60-day invariant review: `internal/runtime/chat.go` grew from 1,533 to 3,694 lines in 90 days
and its `agentState` holds about eighty fields behind five locks with no stated order.
**Relevant requirements:** TS-01.R44 (after R42); INV §5, §23. Behavior-preserving; no feature
requirement changes.

## Outcome

The turn in progress has one owner and one lock, and agent status has one writer, so a new turn
feature no longer adds its own race.

## Included work

One turn structure inside `agentState` with one lock and a documented lock order; every status
write, including Stop's, through one function under the status lock.

Not included: splitting `chat.go` into files for its own sake, changes to the ACP wire, and any
behavior change.

Check while here, each unverified: `Start` and `Resume` repeat the same spawn and goroutine
block; `sessionNewParams` and `sessionLoadParams` are near-copies kept in step by convention;
`ChatRuntime.agents` is keyed by agent id alone, so `Stop` and `lookup` do not check generation.

## How we will know it works

The TS-01.R44 checks: both Go test variants, focused `-race` runs over cancel, permission, steer,
held-message and Stop-during-turn interleavings, and the Stop-versus-status-write test.

## Waiting on

Nothing. TS-01.R42 already supplies the typed outcome before the fields around it move.
