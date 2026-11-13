# Match sent prompts by request id

**State:** Finished
**Why:** Direct `/design-feature` request on 2026-11-13. Second of five single-owner refactors from
the 60-day invariant review; submitted input is the area with the most recent field reports.
**Relevant requirements:** FS-03.R118/A93 (building on R7, R48); TS-08.R163; INV §17, §19.

## Outcome

A sent prompt renders exactly once, repeated identical prompts stay distinct, and a queued
follow-up clears only for its own delivery.

## Included work

Pair the local echo with its sequenced event by `message_id` / `request_id` in the transcript
store, the composer's already-delivered check and the held-message clear; keep text matching only
for a sequenced event that has no `request_id`.

Not included: new wire fields or stored data (both ids already exist: `UserPromptData.RequestID`
in `internal/runtime/event.go`, `message_id` on the echo in `Composer.tsx`); a local echo on the
phone; delivery records for the launch assignment prompt and the owner stall nudge, which the
operator dropped from scope on 2026-11-13 because the task row and the stall counter already own
them; consolidation of the receipt tables.

Settle while here: FS-03.R7 says the composer shows sent text immediately, but `Composer.tsx`
appends the local bubble after `sendPrompt` returns. Correct whichever is stale.

## How we will know it works

FS-03.A93 and the TS-08.R163 checks: store and composer tests for both arrival orders, repeated
text and the no-id fallback, with fixtures from the real client serializer; the UI test suite.

## Waiting on

Nothing.
