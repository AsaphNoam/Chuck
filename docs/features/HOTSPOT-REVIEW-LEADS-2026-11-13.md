# Unowned leads from the 2026-11-13 hotspot and audit-candidate review

Leads that surfaced while reviewing 60 days of invariant evidence, the October audit round and
three Sonnet code maps at `e8a8a128`, and that no ready change, charter or finding owns. They are
read from source, unreproduced, and unverified by the planner unless marked *(checked)*. This is
navigation for a later review or design, not a findings list and not a queue.

Owned elsewhere and not repeated here: the five single-owner ready changes (TS-01.R42–R44,
TS-08.R163–R164), [CR-A8](CORE-RELIABILITY-REVIEW-A8.md) and
[CR-A9](CORE-RELIABILITY-REVIEW-A9.md).

## Design topic: one transition rule per durable state machine

Not queued. Intended for `/design-feature` after CR-A8 and CR-A9 report.

No transition table exists for tasks, pipeline runs, activations, receipts or quota intents; each
writer spells its own allowed-from set. `thinkTankTransition` with `settleThinkTankTx`
(`internal/state/think_tanks.go`) is the one shared transition function and the model to copy.
Store calls share one connection, so a read-then-write inside one transaction is atomic and many
guards are Go reads rather than `WHERE` clauses; separate store calls are not atomic together.

| State machine | Writers | Guarded on current value in SQL | Rows-affected checked |
|---|---|---|---|
| `tasks.state` | 23 UPDATE, 5 INSERT sites, 1 DELETE | 22 of 23 | 18 of 23 |
| `task_arms.state` | 7 UPDATE | 7 of 7 | 0 of 7 |
| `pipeline_runs.state` | 11 UPDATE | 4 of 11 (`UpdatePipelineRunCAS` is revision-only; 15 callers hold the from-set in Go) | 10 of 11 |
| `activations.state` | 5 UPDATE, 7 DELETE | 5 of 5, 6 of 7 | 3 |
| `attachment_receipts.delivery` | 3 UPDATE | 2 | 0 |
| `annotation_receipts.state` | 4 UPDATE | 1 | 0 |
| `think_tank_members.setup_state` | 6 UPDATE | 5 of 6 | 2 of 6 |
| `think_tank_attempts.state` | 6 UPDATE | 2 in SQL, rest by same-transaction read | 1 |
| `quota_intents.state` | 16 UPDATE | 15 of 16 | 9 of 16 |

Leads, most likely real first:

1. *(checked)* `ResolveQuota` (`internal/state/quota_success.go`) writes `continued` guarded by
   revision only, with no rows check and no room-hold release; its sibling at `quota.go:516`
   requires `claimed` and the attempt tuple. Also CR-A9 lead 5.
2. `CompleteTaskYield` (`internal/state/task_waits.go:302`) writes `waiting` or `ready` guarded
   only by `pending_yield=1`; its `ready` branch lacks `taskRunOpenClause`. `AdmitReadyTask` still
   has the fence.
3. `ConfirmTaskStart` and `BindPipelineStageTaskStandingAgent` are separate transactions
   (`internal/server/task_dispatcher.go:508-524`). Also CR-A8 lead 10.
4. The closed-run rule exists twice with different missing-run behavior: `taskRunOpenClause`
   treats a deleted run as open, creation (`tasks.go:433-436`) as closed. `DeletePipelineRun`
   leaves `pipeline_stage_tasks` and `task_lineage` rows behind.
5. `SetAttachmentDelivery` (`internal/state/attachment_receipts.go:74`) is unguarded with no rows
   check, so `released` can become `submitted`; `server/queued_input.go:74-75` reads then writes
   across separate store calls.
6. `waiting → ready` has three writers with different preconditions (`task_waits.go:89`, `:240`,
   `attachment_receipts.go:197`).
7. `ready` writers without the closed-run fence: `advanceArmedTasks`, `AbandonTaskStart`,
   `FailTaskStart`, `pipeline_tasks.go:936` and `:1170`.
8. Quota `cancelled` has four writers with three from-sets; `HoldQuotaManualTurn` uses
   `state<>'continued'` and can move `cancelled` to `needs_attention`.
9. Three `finished` writers diverge on what they clear (`tasks.go:1831`, `:2218`,
   `pipeline_tasks.go:413`).
10. `SetAnnotationEffect` takes any state unguarded; the vocabulary value `rejected` is never
    written.
11. The judge's member row stays `pending` after a restart-failed judge; `RetryThinkTankJudge`
    inserts a new row.
12. `ClosePipelineStageTask` and `UpdateTaskCAS` have no production caller.

## Runtime turn core, beyond TS-01.R42 and R44

- The runtime records the applied `fast` value after session configuration but no effective model
  or effort (`applySessionConfig`, `internal/runtime/chat.go`); INV §20 asks for effective values
  read back from the executor.
- `sessionNewParams` and `sessionLoadParams` are near-duplicate functions kept in step by
  convention (the INV registry says "kept in lockstep").
- `Start` and `Resume` repeat the same spawn and goroutine block.
- `ChatRuntime.agents` is keyed by agent id alone; `Stop` and `lookup` do not check generation.

## Client stream and input state

- FS-03.R7 says the composer shows the sent text immediately; `Composer.tsx` appends the local
  bubble after `sendPrompt` returns. Either the requirement or the code is stale.
- `transcriptRequestToken` is never reset on reconnect, and `reconciled` is cleared per agent but
  not on stream open (`ui/src/api/sse.ts`).
- The `chuck-recovery-send:{agent}:{request_id}` localStorage key is removed only on send or
  dismiss, not when the recovery is resolved elsewhere.
- The phone keeps its own held and transcript queries and does not use the desktop held store, so
  desktop held-state fixes do not reach it.

## Audit candidates considered and not queued

No defect is asserted for any of these; they were dropped for lack of evidence, not cleared.

- Run workspaces and placement (`internal/{server,state}/pipeline_workspace*.go`, 2026-10-29): the
  CR-A3 question over new code. Not read.
- File writing and opening any local file (`internal/server/filewrite.go`, 2026-11-06): a new
  destructive capability behind the loopback boundary. Not read.
- Updater and installer: deferred twice by the audit plan. `Layout.Rollback` flips `current`
  first by design; an error returned after the first swap is untested.
- Phone lifecycle and revocation: an in-flight attachment download ignores a revoked device
  (`serveAttachmentBytes`, capped by the 10 MiB file limit); the stream-overflow fix CR-A7-1 was
  not assumed to cover the phone's own connection.
- Operator CLI output truthfulness (`internal/cli/operator_*.go`): CR-A6 found two false-success
  findings in a small sample.
- Other batch-limited sweeps: starvation behind a blocked prefix was found twice (CR-A2-6,
  ASRG-5); no search for a third was run.
- Test doubles against captured real frames (`fakeacp`, UI mocks): INV §17 records six escapes.
