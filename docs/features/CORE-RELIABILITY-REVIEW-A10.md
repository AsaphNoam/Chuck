# CR-A10 — Durable state transitions have one rule

Prepared 2026-11-13 from one focused Sonnet mapping pass and parent integration against
`e8a8a128`. One integrated review for a frontier reviewer, extending the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). The mapping read every status-writing
statement it could find; it ran no probes or tests, and the parent checked only the lead marked
*(checked)*. [HANDOFF.md](HANDOFF.md#review-queue) owns live status. Run after CR-A8 and CR-A9.

## Question and responsibility

Can each durable status column move only along transitions that are stated in one place, and does
every writer notice when it lost the race?

For each state machine below, confirm or refute the leads, then say which machines need a shared
transition function. `thinkTankTransition` with `settleThinkTankTx`
(`internal/state/think_tanks.go`) is the one existing example. The review's second output is a
recommendation for `/design-feature`: which machines to convert first and what the rule must
express. It fixes nothing itself.

## Read first

The handoff's Current position and Active change, this charter, and the CR-A8 and CR-A9 results.
TS-02 for the tables; TS-10.R1–R6, R17–R19, R43, R48–R53 for task and quota compare-and-swap
rules; TS-09.R43 and R69–R70 for the run cursor. INV §2, §5, §15, §19, §23.

Store calls share one connection, so a read-then-write inside one transaction is atomic and many
guards are Go reads rather than `WHERE` clauses. Separate store calls are not atomic together.

## Writer inventory

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

## Leads to settle

Questions for the review, not findings. Most likely real first.

1. *(checked)* `ResolveQuota` (`internal/state/quota_success.go`) writes `continued` guarded by
   revision only, with no rows check and no room-hold release; its sibling at `quota.go:516`
   requires `claimed` and the attempt tuple. Shared with CR-A9 lead 5.
2. `CompleteTaskYield` (`internal/state/task_waits.go:302`) writes `waiting` or `ready` guarded
   only by `pending_yield=1`; its `ready` branch lacks `taskRunOpenClause`.
3. `ConfirmTaskStart` and `BindPipelineStageTaskStandingAgent` are separate transactions
   (`internal/server/task_dispatcher.go:508-524`). Shared with CR-A8 lead 10.
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
13. The runtime records the applied `fast` value after session configuration but no effective
    model or effort (`applySessionConfig`, `internal/runtime/chat.go`). Does anything persisted or
    displayed as the current model rest on the requested value alone (INV §20)?

## Joins and finding ownership

CR-A8 owns the pipeline pause contract and CR-A9 the room lifecycle; where a lead here is shared,
the earlier review's finding stands and this review only adds the structural recommendation.
CR-A2 fixed admission and wait findings on 2026-10-22; corroborate or add a distinct trigger.

## Required review result

Record the reviewed HEAD. Per state machine: confirmed findings, refuted leads, and whether a
shared transition function is warranted. Record findings under **CR-A10** in the handoff using
workflow §7 with one Fix model recommendation. Change no product code or normative specs.
