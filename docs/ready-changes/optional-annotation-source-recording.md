# Optional source transcript recording for annotation assignments

**State:** Waiting to start
**Why:** Operator request 2026-10-29, scope confirmed in conversation: send unrelated side quests
to another agent without including the current agent in the loop; rename New task to New agent.
**Relevant requirements:** FS-13.R35–R36/A25–A26, TS-03.R75, TS-02.R60, TS-08.R148;
INV §2, INV §8, INV §9, INV §10, INV §11, INV §15.

## Outcome
Desktop and phone annotation assignments offer a default-on **Include in current agent's
transcript** checkbox for existing/new recipients. Unchecked sends the same annotation mail
without a source event. All annotation destinations use **New agent** terminology.

## Included work
Extend the existing request, delivery receipt and shared batch owner; retain per-batch choice
through reload/cancel/retry and reset it for a fresh batch. Self recording and room recording
remain unchanged; rooms receive only the rename. No transcript deletion, task-system rename,
new storage mechanism, new delivery protocol or provider change.

Existing seams verified: `internal/server/sessions.go::handleAnnotations` currently requires
positive `SourceSeq` for accepted replay and appends before every delivery;
`internal/state/annotation_receipts.go::CommitAnnotationMail` already commits mail/activation and
acceptance atomically without requiring a source sequence. `annotationStore`, `annotationBatch`
and `useAnnotationDelivery` already own draft lifecycle, command identity and desktop/phone
delivery. Extend those seams; no adapter capability or workaround is needed.

## How we will know it works
FS-13.A25–A26: focused handler/receipt tests prove unchanged default delivery, absent source events
when unchecked, truthful zero-sequence success, lost-response/restart replay and SQL rollback.
Store/delivery/control tests prove choice lifetime, self behavior, launch cancellation and same-agent
retry. One focused desktop/phone rendered journey verifies checkbox access, destination feedback
and rename, with a room-control check. Run the applicable implementation closure matrix once.

## Waiting on
None.
