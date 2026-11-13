# CR-A8 — Pipeline pause, answer and recovery ownership

Prepared 2026-11-13 from one focused Sonnet mapping pass and parent integration against
`e8a8a128`. This extends the [core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md) with one
integrated review for a frontier reviewer. The mapping read requirements-adjacent source and test
names; it ran no probes, tests or repairs, and the parent checked only the two leads marked
*(checked)*. [HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

When a pipeline run pauses, stalls, or receives an answer or decision, does exactly one party own
the next move, can that party always make it, and does input typed into an owner's chat reach the
run once, intact, or get refused with no side effect?

Trace pause reason → who may act → accepted input and its durable record → new visit or resumed
task → admission → provider delivery → report acceptance → next pause or progress. Check safety
and progress together: a run that cannot be advanced by any control except Stop is a defect, and
so is input that reaches an agent outside run control after the run refused it.

This review owns the goal-led pause/continue contract written on and after 2026-10-30:
`internal/pipeline/{actions,decisions,reconcile,manager}.go`, `internal/state/pipeline_decisions.go`,
`internal/server/{queued_input,pipeline_stall,pipeline_lifecycle}.go`, the run branch of
`handleMessageAction` (`internal/server/attachments.go`) and the report path
(`internal/server/task_handlers.go`, `internal/messaging/task_tools.go`).

## Read first

The handoff's Current position and Active change, this charter, and the open **PIPE-CHAT**
findings. Use workflow §§3–7 with this subsystem scope in place of a commit-diff scope.

- FS-14.R6, R19–R20, R48, R55, R62–R63, R66, R73–R74, R97, R99–R100 and their acceptance items.
- TS-09.R43, R49, R55, R69–R71; TS-02.R55 for the input receipt; TS-10.R48–R53 for task cleanup,
  waits and continuation.
- FS-03.R48, R90, R99 for held, queued and recovered chat input.
- INV trigger index, then §2, §5, §15, §18, §19, §21, §22, §24.

Requirements govern expected behavior; source comments, PIPE-CHAT and this map are navigation.

## Starting authority map

Verify against the selected HEAD.

| Fact | Authority | Who may end the wait |
|---|---|---|
| Run position and pause reason | `pipeline_runs.state` / `pending_action` / attention reason, written by revision compare-and-swap | see rows below |
| `needs_input`, failure, blocked, interrupted and similar pauses with no pending action | the run row | a person: Continue, Retry, Replace, or a chat answer through `ContinueFromOwner` |
| `await_approval` | the run row | a person through the run page only; chat is excluded |
| `release_stage_task`, `activate_replacement`, `cleanup_run` | pending action | `Reconcile` from turn end, the dispatcher tick and startup |
| Open visit with a silent owner | stall counter on the stage task | the stall detector: nudge, then pause as `owner_not_reporting` |
| A person's answer | the new visit's instruction; for chat, also an `attachment_receipts` row | delivered by the visit's assignment or by `continueQueuedInput` |

## Leads to settle

Each is a question for the review, not a finding. None has a test.

1. *(checked)* A chat answer that fails validation or loses the unlocked revision read in
   `ContinueFromOwner` is logged in `continueRunFromConversation` and the handler falls through to
   ordinary delivery. Does refused run input reach the owner as plain chat while the run stays
   paused and the person sees acceptance?
2. *(checked)* `task_tools.go` maps `ErrPipelineStageConflict` to "This task already has a recorded
   result". Which distinct conditions return that error, and does an owner with a missing required
   output get told it has already reported (INV §21's recorded `stale_assignment` shape)?
3. On owner replacement, the answer lives only in the replaced visit's instruction and the receipt
   move takes only `prepared` rows. Does the successor ever see the answer?
4. `pauseStartupRun` sets `restart_reconcile_failed` and clears the pending action. Is there any
   control other than Stop that leaves that state when the current task finished successfully?
5. `OnStageTaskInterrupted` and `RecoverStageTask` swallow a lost compare-and-swap. Can an owner
   exit racing start confirmation leave a running run with an interrupted task and no sweep?
6. `RecordOwnerStall` increments before the nudge is sent, and resets only at wait, new visit and
   Resume. Can refused nudges or a person's own chat turns pause a healthy run?
7. Resume binds the input receipt before the Resume transaction. What does a retry at the same
   revision do after a failure between them?
8. Continue on an approval or controller-recovery pause accepts input and binds none. Is the
   input dropped silently?
9. Which stall-check early returns (quota-blocked, lifecycle in flight, not yet idle) have no
   later re-evaluation?
10. `BindPipelineStageTaskStandingAgent` runs in a separate transaction from start confirmation
    and its failure is logged at Debug. What rebinds a running stage with no standing agent?

Also sweep the state-changing calls whose errors are discarded or logged at Debug only in
`queued_input.go`, `manager.go` and `task_dispatcher.go`.

## Reuse these proofs before adding any

`internal/server/{pipeline_stall_test,goal_led_input_test,pipeline_continue_control_test,
attachments_test}.go`, `internal/pipeline/{goal_led_test,manager_test}.go`,
`internal/state/{pipeline_decisions_test,pipelines_test}.go`, and the three skipped PIPE-CHAT
reproductions.

## Joins and finding ownership

- **PIPE-CHAT-1 and -3** own the lost durable user event and the non-atomic successor/receipt
  bind. Corroborate them or add a distinct trigger; do not refile the same mechanism. Lead 7 and
  the duplicate-delivery variant of lead 1 are adjacent, not identical. PIPE-CHAT-2 and -4 are
  independent of this review.
- CR-A2 owns task admission and waits; CR-A5 owns which accepted result advances which work.
  This review owns what happens between a pause and the next accepted input.
- The ready changes `one-typed-turn-outcome.md` and `match-sent-prompts-by-request-id.md` touch
  neighboring seams; note any finding they would resolve or complicate.

No new scheduler, workflow engine or receipt framework is proposed. A structural remedy, such as
one pause/continue ownership rule, is a separately designed work item; this review fixes nothing.

## Required review result

Record the reviewed HEAD and relevant dirty work. Produce one ledger of pause states, who can end
each, the durable record of each input path, and which evidence proves both refusal without side
effect and eventual progress. Separate confirmed findings from hypotheses.

Record findings under **CR-A8** in the handoff using workflow §7, with one Fix model
recommendation for the unit. Say whether PIPE-CHAT-1/-3 should be fixed alone or with CR-A8's
findings. Use isolated temporary homes and loopback-only servers; change no product code or
normative specs.
