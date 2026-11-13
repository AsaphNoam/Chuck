# Share input delivery between desktop and phone

**State:** Waiting to start
**Why:** Direct `/design-feature` request on 2026-11-13. Third of five single-owner refactors from
the 60-day invariant review: `ui/src/remote/AgentScreen.tsx` took 21 fix commits in six weeks,
mostly in paths it re-implements from desktop chat.
**Relevant requirements:** TS-08.R164; INV §2, §13, §17. Preserves FS-03 and FS-20 input behavior
as specified; no feature requirement changes.

## Outcome

Send, Steer, retry, withdraw and the recovery offer run the same code on desktop and phone, so a
fix to one is a fix to both.

## Included work

Shared hooks and one recovery-offer component under `ui/src/components/chat`, used by `Composer`,
`TranscriptView` and the phone `AgentScreen`; each client keeps its own state projection and
layout.

Not included: a local echo on the phone (operator decision, 2026-11-13); the phone's own event
connection, transcript window and pagination; the permission card; any visual change.

Easier after `match-sent-prompts-by-request-id.md`, which settles the id the shared send path
carries, but not blocked by it.

## How we will know it works

The TS-08.R164 checks: existing desktop and phone input tests against the shared hooks, one shared
suite over accepted, held, queued, uncertain and refused outcomes for both projections, and the
phone render harness at phone width.

## Waiting on

Nothing.
