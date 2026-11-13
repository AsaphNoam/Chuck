# One typed turn outcome

**State:** In progress
**Why:** Direct `/design-feature` request on 2026-11-13, after a review of 60 days of invariant
evidence and fix history named `internal/runtime/chat.go` the most-fixed file (34 fix commits) and
the turn outcome its clearest missing owner. First of five single-owner refactors.
**Relevant requirements:** FS-03.R117/A92; TS-01.R42; INV §17, §20.

## Outcome

A turn's ending is decided once and every consumer reads that decision, so a failed or cancelled
turn can no longer look completed to status, quota settlement, inline mail or the server.

## Included work

One resolver in `internal/runtime` called by normal completion, the failed-prompt path and the
transport-closed path; one exported decoder for `turn_end` events used by the server and
transcript consumers; a source scan against new `stop_reason` comparisons.

Not included: any change to the `stop_reason` wire values or stored transcripts, new outcome
kinds visible to people, and the turn-state restructuring of `single-owner-turn-state.md`.

Evidence read for the design: the outcome is rewritten in three places in `completePromptTurn`
(`internal/runtime/quota.go`), synthesized as `{"stopReason":"error"}` for an RPC error
(`chat.go`), built separately in `onTransportClosed`, and compared as a string in
`internal/server/{quota_notice,think_tanks,fork_point}.go` and `internal/transcript/project.go`.

## How we will know it works

FS-03.A92 and the TS-01.R42 checks: resolver table tests, one consumer test per reader from the
same emitted event, the source scan, both Go test variants.

## Waiting on

Nothing.
