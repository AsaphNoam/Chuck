# Goal-led unattended pipelines — temporary implementation plan

Sequencing only; FS-14.R97–R103/A61–A68 and TS-09.R66–R72 are authoritative. Delete when the
change finishes.

## Done

1. **Queued input for waiting assignments (WAIT-PROMPT-1/2, TS-09.R69 core).** Migration 65
   `attachment_receipts.assignment_task_id`; `QueueAssignmentInput` writes the receipt atomically
   with the open-assignment check → `delivery:"queued"`; the dependency activation claims it
   (`state.TakeQueuedAssignmentInput`, `runtime/message.go takeQueuedAssignmentInput`) and carries
   it in the same provider turn; `server/queued_input.go continueQueuedInput` (tick, turn end,
   acceptance) readies the next continuation, hands input to an idle executing assignee, releases
   ended-work input to recovery. Composer/phone show the queued notice.

## Next

2. **Visits (R67).** Migration adding `pipeline_stage_tasks.visit_sequence` (unique per run) and
   the run's current-task reference; backfill in stage-index/attempt order. `latestStageOrder`,
   `LatestPipelineStageTask`, `PipelineStageTaskForAssignee`, `AcceptedPipelineStageTaskReport`,
   `MarkRoomTeardown`'s latest-room subquery, `priorStageResults`/`lastStandingAgent`, timeline and
   Stop use visit order. A revisit allocates the next attempt number for that stage.
3. **Contract (R66/R68/R71).** Template `required_for_completion`; `get_assigned_task.pipeline`
   (visit, current stage, ≤32-entry step menu, required checks, recovery); optional
   `report_task_result.pipeline_decision` validated inside `AcceptPipelineStageTaskResult` and stored
   with the result/closure intent. Both tool schemas, operating guidance and builder examples change
   together.
4. **Decision execution (R66/R68/R70).** `reconcileTaskStageRelease`/`advanceTaskStage` execute the
   recorded decision: next/revisit, complete (needs required evidence; end of plan needs explicit
   complete), needs_input (paused with the question). Failure/blocked without decision → owner
   recovery assignment. Approval gate still pauses before activation/completion. Round accounting
   (three per allowance, charged on same-stage/backward routes and recovery assignments, effect-key
   idempotent), exhaustion → recoverable pause + notification, Resume/instructions renew once.
5. **Stall recovery and replacement (R70/R69).** Turn end without report/wait drives owner
   continuation; three such turns pause. Automatic visibly fresh same-runtime owner replacement when
   native resume fails, carrying pending queued input. Run-page answers and conversation answers use
   one recorded-input helper.
6. **Surfaces and closure (R71/R72/R103).** HTTP/CLI/desktop/phone show visits, checks, questions,
   held input, recovery, outcome; A61–A68 tests and journeys; TS-06 closure matrix.
