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

2. **Visits (R67).** Migration 66: `pipeline_stage_tasks.visit_sequence` (unique per run,
   `nextVisitSequence` in each insert) and `pipeline_runs.current_task_id` (`setRunCurrentTaskTx`),
   backfilled in stage/attempt order. All cursor reads order by visit; `NextPipelineStageAttempt`
   allocates a revisit's attempt. Raw test inserts must supply a distinct `visit_sequence`.

3–4. **Decision contract and execution.** Migration 67 (`decision_json`, `approved_at`, run
   `recovery_rounds_used`, `recovery_effect_key` (unused yet), `owner_stall_turns`, `pause_question`);
   `state/pipeline_decisions.go`; `pipeline/decisions.go`; Continue handles approval, answer, resume.
   Tests: `state/pipeline_decisions_test.go`, `pipeline/goal_led_test.go`.

## Next

5. **Stall recovery and replacement (R70/R69).** Turn end without report/wait drives owner
   continuation; three such turns pause. Automatic visibly fresh same-runtime owner replacement when
   native resume fails, carrying pending queued input. Run-page answers and conversation answers use
   one recorded-input helper.
6. **Surfaces and closure (R71/R72/R103).** HTTP/CLI/desktop/phone show visits, checks, questions,
   held input, recovery, outcome; A61–A68 tests and journeys; TS-06 closure matrix.
