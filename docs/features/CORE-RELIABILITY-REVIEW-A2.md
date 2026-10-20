# CR-A2 — Admission, waiting and recovery

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`bdc13d02`. This maps **A2 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, source paths and existing test assertions;
it performed no audit, failure probes, test execution or product repairs.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Will eligible work make progress under its declared start/retry policy, while cancelled, stale,
ineligible or already-owned work stays stopped? After contention, interruption or cleanup failure,
does every unfinished obligation retain an owner and a reachable recovery path?

Trace committed source fact → readiness/opportunity → assignment/capacity claim → lifecycle/turn
admission → native effect → start confirmation → yield/release → recovery. Look for silent
stranding, duplicate or unauthorized execution, lost capacity and cleanup that affects a newer
conversation. Check both safety and progress: preventing duplicate work by leaving an eligible task
forever invisible is not sufficient. An explicit parked/interrupted state with the prescribed repair
can be correct; neither process existence nor a `starting` row proves confirmed execution.

This review owns task/dependency admission, durable waiting, waking-mail opportunities, quota
continuation and their runtime-claim/recovery joins. Follow Codex and Claude chat through the
shared runtime boundary. Automatic mail/task/quota paths must respect terminal exclusions; adding
terminal automation is outside scope. Inspect neighboring owners wherever needed to establish this
contract, without re-auditing all pipeline, room, provider or content behavior.

## Read first

Read the current handoff's Current position and Active change, this charter and current related
findings. Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a
commit-diff scope. Read requirements before following source:

- FS-16.R2–R9/R16–R18/R21–R26: exclusive assignment, explicit outcomes, typed prerequisites,
  start budget, attempts, Retry/Re-arm and runtime ownership. R30–R38 cover persistent assignment,
  waits, scoped cancellation and managed work; read R49's task reporting boundary as needed.
- TS-10.R1–R6/R9–R10/R15/R17–R19: durable readiness, CAS, distinct claim/effect/confirmation,
  capacity and recovery. R25–R35 cover lineage/closure, waits, yield, continuation and durable
  cleanup; R38–R47 add room-backed targets and quota ownership/check recovery.
- FS-06.R22/R24–R27/R30–R36 and TS-01.R16/R19–R21/R24/R31–R33: wake eligibility, activation
  policy, exclusive lifecycle/turn seams and mail receipt versus opportunity lifetime. Read the
  later shipped inline-mail rules: R30–R36 supersede the older pull-only wording where they differ.
- FS-01.R6–R7/R10/R20–R21/R33–R34/R40–R45/R48–R51 and TS-10.R39–R47: lifecycle, orphan and
  quota recovery. Later unknown-reset checks extend the earlier known-reset-only behavior.
  TS-04.R86–R88/R97 govern provider quota evidence and isolated checks; TS-02.R23/R25/R34/R45/R50
  govern related storage. TS-01.R39/TS-04.R100 govern captured terminal observations at the join.
- Read the invariant trigger index, then matching bodies. Likely classes are
  §§1/2/4/5/7/9/11/15/16/17; include §8 when refusal or repair status matters. TS-06.R35 governs
  proportional evidence: adequate existing proof can justify adding no test.

Requirements govern expected behavior. Source comments, old reviews and this map are navigation.
Do not treat planned passive quota views, paused action transport or a new scheduler as shipped
requirements. Admission review does not require implementing any of them.

## Starting authority and lifetime map

Verify this map against the selected HEAD; apparent state names alone are not the oracle.

| Fact / obligation | Authority and producer | Consumer and lifetime distinction |
|---|---|---|
| Prerequisite truth and task readiness | State-owned `work_results`, task arms and signal-fire mutations in `internal/state/tasks.go` | `EvaluateSource`/arm resolution decides readiness from committed facts; an impossible conjunction parks for the documented repair. Wait-for-change is a different predicate from all-arms-satisfied. |
| Start reservation and assignment | Durable task row/revision, attempt ID, reserved agent/generation, assignee membership and exclusive claim | Admission reserves before effects; confirmation establishes membership/running. Reservation alone grants no attached-context membership. A waiting/continuing assignment retains exclusivity after releasing runtime capacity. |
| Runtime capacity and ownership | Durable created/woke/borrowed task claim | Created/woke runtimes consume the task budget; borrowed runtimes do not. Readiness, assignment exclusivity, capacity, lifecycle claim, turn gate and in-process worker slots are separate constraints. |
| Wait/yield/continuation | Wait sources/revisions/version, execution handle, pending yield/resume/continuation fields | `NotifyTaskWaiters`, turn-end cleanup and re-admission operate on the same assignment. Watches cannot lose a concurrent source change or admit before the old runtime claim is released. |
| Mail opportunity versus content | `activations` mail row/token and independent durable message/read/inline-reservation state | Waking insert and opportunity commit together; deferred insert creates no opportunity. Attempt retirement does not mean content was read, and uncertain content receipt does not re-arm a wake. |
| Dependency opportunity | `activations` dependency row/token with task source ID | Existing-agent task start consumes a task-owned opportunity. The source task, not mail policy, determines retry and confirmed start. |
| Quota interruption and continuation | `quota_intents` episode/chain/revision/token, exact owner/execution tuple, check allowance, selected provider source/profile and start/cleanup receipts | Due dispatch and owner admission authorize the original work. Check success, actual provider start and corroborated continuation are different receipts. A positive check cannot authorize a different source selection. New failure IDs or restart do not replenish a chain's check allowance. |
| Unfinished release/yield/cancel | Durable task intent, effect identity/version, retained runtime claim and cleanup phase/backoff | Cleanup retries only the exact retained effect. A terminal result does not prove Stop/release finished; failure cannot discard ownership or free capacity prematurely. |
| Runtime/process and wake hints | In-process registry/lifecycle/turn owners; bus/channels and worker semaphores | These coordinate live effects, not durable work truth. Restart does not adopt old runtimes. Hints may be lost; name the actual durable rediscovery path. |

Ordinary Think Tank turns use room opportunities/attempts rather than an `activations` table row
for every runtime kind. A2 checks owner admission and quota/recovery joins there; A5 will own the
whole room/result lifecycle. Do not infer a universal persistence model from the activation registry.

## Keep the retry policies distinct

| Opportunity / effect | Before an external attempt | After attempt, ambiguity or restart |
|---|---|---|
| Waking mail | Lost lifecycle/turn claim leaves a pending opportunity; recheck eligible unread waking source | The same attempted opportunity is not automatically replayed, even after failure. New waking mail can create another; durable unread content may appear on a later independently authorized turn. |
| Deferred mail | Saved without wake or continuation | Idle, turn-end, unread backlog and restart cannot promote it to waking mail. Content receipt/recovery remains independent of activation. |
| Task/dependency start or wait continuation | Busy lifecycle, capacity or competing assignment defers without spending a start failure or losing admission order | Remains actionable until confirmed task start under its bounded task policy. Restart distinguishes created/woke reservations from ambiguous borrowed ones; do not promise universal exactly-once execution or automatic retry of uncertainty. |
| Quota continuation / isolated check | Revalidate enablement, exact owner, archive state and task/room capacity before charging or launching | Preserve episode/chain and exact execution identity. Uncorroborated original-work delivery holds for attention; an interrupted check cannot become positive continuation authorization. After probe cleanup, revalidate the exact token/revision/chain, original work and current source selection before admission. |
| Task cleanup | Commit release/yield/cancel intent before Stop/cancel/release | Retry retained cleanup with classified backoff and effect fencing, not the accepted task or report. Unsafe ownership refuses with retained obligation/repair; ordinary contention is not a failed cleanup attempt. |

Do not impose one confirmation boundary on all domains. Task start confirms the recorded runtime
generation and assignment; the task outcome requires a later explicit result. Mail content receipt
requires its own valid provider completion, while quota continuation corroboration may use qualifying
progress from the exact admitted execution. A successful isolated check proves none of those effects.

## Shipped paths to trace

| Path | Entry points and connected implementation |
|---|---|
| Create/repair prerequisites → ready → start | `internal/state/tasks.go`; `internal/server/{task_handlers,task_dispatcher}.go`; `internal/messaging/task_tools.go`. Follow HTTP/tool creation, typed results and project signals, Retry versus Re-arm, and current stage-task creation as a caller. |
| Existing conversation start / continuation | `internal/state/activations.go`; `internal/server/{task_dispatcher,messaging_loops,resume}.go`; `internal/runtime/{tasks,activation_kinds,registry_message,chat}.go`. Follow the real start/confirm boundary beyond the composed request. |
| Wait → source change → yield → re-admit | `internal/state/task_waits.go`, task tool/assignment read and `internal/server/task_dispatcher.go`. Include source deletion/actionable interruption, coalescing and owner closure at effect time. |
| Waking/deferred mail → possible turn | `internal/state/{messages,activations}.go`, `internal/server/messaging_loops.go` and runtime prompt/activation preparation. Follow stopped versus already-running recipients and waiting-task routing. |
| Quota → check/continue → owner admission | `internal/state/{quota,quota_admission,quota_controls,quota_owners,quota_checks}.go`; `internal/server/{quota,quota_dispatch,quota_check,quota_controls}.go`. Trace task and room owner handoff, allowance/start receipts and exact probe cleanup. |
| Turn-end / manual cancellation → retained cleanup | Task release/yield/cancel methods and dispatcher cleanup; runtime captured generation/turn cancellation and process-ownership seams. Observe the effect guard, not just an earlier lookup. |
| Startup and shutdown → automatic recovery | `internal/server/server.go`, mail recovery, `recoverTasks`, quota claim recovery and task/room/pipeline startup callers. Follow actual ordering, failure propagation and loops, not only direct helper calls in tests. |

At this baseline task dispatch, mail execution and quota admission have separate bounded worker
mechanisms. Do not mistake their similar batch sizes for one global agent-concurrency budget.
The task tick revisits ready work/cleanup; result evaluation is event-driven with startup recovery,
not a periodic full graph walk. Establish how each producer's unsettled work is rediscovered.

At the planning baseline, startup launches the reconciliation sweep, recovers quota claims/repairs
notices, then recovers mail and starts its loops, recovers tasks, starts task dispatch, and recovers
Think Tanks. Pipeline startup follows server serving. Task recovery discards dependency activations,
visits due cleanup, resolves unfinished tasks and re-evaluates arms. Re-read this ordering in
`server.go`; launched goroutines can overlap later recovery, so a sequential call list alone does
not prove admission is fenced or quota ownership preserved through generic task/run/room recovery.

## Connected searches and useful scenarios

These are facets of one review, not additional queue units or a required Cartesian test matrix.

| Scenario / search | Evidence that distinguishes correct deferral/recovery from silent failure |
|---|---|
| Several prerequisites or a signal become satisfied while the budget is full, then capacity frees | Read committed source/arm/task revisions, ready order, claim and attempt count. Show that contention spends no failure and oldest eligible work progresses. Check whether an older blocked batch can hide later eligible work; claim only the ordering/fairness the product promises. |
| Source fact commits but evaluation/activation hint is lost or a post-commit step fails | Identify the precise event-driven callback, durable pending marker, bounded sweep or startup step that will revisit it. Exercise the real producer-to-consumer chain when helper-only proofs do not establish liveness. No hint must be required as durable truth. |
| Two tasks, mail and manual input compete for the same running/stopped assignee | Compare assignment/lifecycle/turn claims with native frames; only authorized work starts. Pending dependency work must not inherit mail retirement, and mail cannot bypass a waiting assignment's capacity/closure gate. |
| Watch completion races wait registration or turn-end yield | Compare wait version/source revision, execution handle, pending yield/resume flags and runtime claim. Coalesced changes lead to the same assignment only after cleanup; cancellation wins. Inspect retained assignment context using ASRE-1's owner rather than duplicating it. |
| Server dies after start reservation or external effect but before confirmation | Distinguish created, woke and borrowed ownership. Check actual startup ordering, reaping/attention, retained claim and recalculated capacity; no guessed adoption, second borrowed prompt, or clearing a failed reap. Use one well-chosen real process boundary if seeded recovery tests leave this unproved. |
| Result/cancel/yield commits while Stop or release fails, then storage/process control recovers | Observe exact cleanup identity, guard at effect time, retained slot, retry due time and safe progress. Prove bookkeeping retry does not rerun the task or stop a newer/unrelated turn. Reuse open ASW1-6/9 and ASF-1/3 evidence. |
| Mail wake is attempted and fails; deferred mail and a later new waking message remain | Count provider attempts per opportunity and track distinct message/activation IDs. A later authorized content redelivery is not another attempt of the old wake. Deferred backlog alone must never schedule a turn. |
| Owned work hits quota while task capacity is full or owner cancellation/disable wins | Compare original task/room owner, episode/chain/revision/token, effective source/profile selection, checks charged, native effects and confirmed continuation. A positive check for a different effective source/profile cannot authorize continuation. Check eligible deferral and no allowance reset; stale completion cannot revive cancelled work. Existing ASRG-1/2/3/5 retain ownership of their known mechanisms. |
| Multiple unsettled records survive a failed cleanup or restart | Check per-record isolation, bounded due selection and progress for healthy eligible work behind failures. A fatal whole-recovery failure must prevent normal startup; a bad record must not erase other work or silently consume the only recovery owner. |

Build a small number of connected journeys from the actual implementation. Combine a dependency
wait/capacity recovery journey with a contrasting mail/quota policy case where needed. Count real
native submissions and confirmed ownership alongside rows; do not demand a test for every table
cell, provider, ownership kind or arbitrary timing combination.

## Reuse these proofs before adding any

These existing tests were researched, not run or certified. Check their setup and assertions.

| Seam | Existing proof and its limit |
|---|---|
| Readiness and capacity | `internal/state/tasks_test.go`: `TestFanInBecomesReadyOnlyOnTheLastArm`, `TestAdmissionTakesCapacityInTheStatementThatGrantsIt`, `TestConcurrentAdmissionForOneAgentLeavesExactlyOneActive`, `TestOnlyRealStartFailuresSpendTheBoundedAttempts`. Store transitions do not alone prove dispatcher/native effects or dropped-hint recovery. |
| Real dispatcher admission | `internal/server/task_dispatch_test.go`: `TestTheBudgetLimitsRuntimesWithoutNarrowingReadiness`, `TestATaskForARunningAgentActivatesThatConversation`, `TestATaskForAStoppedAgentWakesIt`. Inspect start confirmation and failure coverage, not only happy-path status. |
| Wait/yield | `internal/state/task_waits_test.go`: `TestTaskWaitYieldsAndWakesAfterWatchedResult`, `TestTaskWaitRefusesSameAssigneeAndCancellationWinsYield`; `internal/server/task_dispatch_test.go`: `TestYieldedTaskReleasesThenResumesOnWatchedResult`. The latter reaches a durable continuation-ready request; that alone does not establish its later native execution. |
| Restart and retained cleanup | `internal/server/task_dispatch_test.go`: `TestRestartResolvesUnfinishedTasksFromTheirOwnRows`, `TestDispatcherReconcilesDueTaskCleanupWithoutRetryingWork`; `internal/state/task_waits_test.go`: `TestTaskCleanupFailureBacksOffWithoutChangingTaskOutcome`, `TestUnsafeTaskCleanupRequiresExplicitRepair`. Seeded in-process recovery is not a server crash across a live effect. |
| Mail opportunity | `internal/server/activation_test.go`: `TestCoalescedMailProducesOnePromptAndIsNeverReplayed`, `TestFailedStoppedActivationIsNotRetriedAfterRestart`, `TestMailActivationDefersToALifecycleClaimTakenAfterTheHint`, `TestStartMessagingLoopsFailsWhenRecoveryFails`; `internal/state/messages_test.go`: `TestClaimMailActivationIgnoresDeferredBacklog`. These prove mail policy, not dependency retry. |
| Quota owner/capacity/control | `internal/server/quota_capacity_test.go`: `TestQuotaStoppedBorrowedTaskDefersUntilRuntimeCapacity`; `quota_dispatch_test.go`: `TestQuotaTaskAssignmentSurvivesProviderInterruptionAndRestart`; `quota_controls_test.go`: `TestQuotaControlWrapperPreservesRefusalAndReplacedEpisode`; `quota_check_test.go`: `TestQuotaCheckReadyAuthorizesOneContinuation`, `TestQuotaCheckDisableCancelsBeforeProviderWork`. Inspect whether each exercises pre-check admission or only continuation; ASRG-2 already records that distinction. |

The main unproved joins to resolve are producer-to-dispatch liveness, wait-ready-to-actual-start,
and actual startup across ambiguous effects. This is a research question, not a claim that no other
test covers them. Expected transitions come from the source domain's requirements, not a mock that
copies today's implementation. A fake provider can establish frame identity/count and host fences;
live-provider task-reporting/continuation adoption remains separate evidence.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md) and live handoff. Reuse ASR-E/G,
ASR-I and [original joins J3/J4/J6](AGENT-SURFACE-REVIEW.md#mandatory-joins-between-reviews).

- ASW-1 remains open at this baseline; ASW-2/3 are closed. ASW1-6/9 already own cross-generation
  control ordering and stale terminal cleanup identity. ASF-1 owns turn/cancellation settlement,
  including ASRG-3; do not file the same mechanism under CR-A2.
- ASF-3 owns ASRE-1's lost watched continuation context, ASRF-1's lost post-result action,
  ASRF-2's obsolete executor, ASRG-1's room-start compensation, ASRG-2's check-before-capacity
  and ASRG-5's probe-cleanup starvation. ASF-4 owns result bounds. Corroborate these or add a
  distinct current trigger; do not create another owner or use old failing overlays as new proof.
- A1 owns durable conversation/session truth and repair. A2 uses its continuation seam but owns
  work eligibility and recovery claims. A3 will own actual workspace/config readiness; A4 exact
  content/reference/knowledge delivery. Here distinguish opportunity/start from content receipt.
- A5 will own which accepted task/pipeline/room result advances which work. A2 owns downstream
  readiness/admission and retained runtime obligations; follow result/closure callers far enough
  to prove those boundaries. A6 will own broader action receipts/operator truth; A7 sustained
  cost. Bounded draining that determines whether eligible work ever runs remains an A2 question.

No new scheduler, broker, retry framework, generalized workflow engine, transport migration or
automatic provider prompt is proposed. Any high-value architectural remedy found during review
is a separately designed work item; this review implements no fix.

## Required review result

Record exact reviewed HEAD, relevant dirty work and provider/adapter source versions. Produce one
concise ledger of authorities, distinct retry/confirmation policies, producer/consumer paths and
connected scenarios, showing which evidence proves both safe refusal and eventual progress.
Separate confirmed findings from hypotheses and unavailable live evidence.

Record real findings under **CR-A2** in the handoff using workflow §7: source location, credible
trigger, development impact, requirement/invariant and reproducible evidence or a precise
source-backed mechanism. Give the unit one Fix model recommendation based on its hardest open
fix. A known safe/manual recovery policy is not a defect merely because another policy would retry.

Use isolated temporary Chuck/native homes and loopback-only servers, preserve dirty work/user data
and follow the review skill's allowed reporting rules. Change no product code or normative specs.
Close only this unit when its review is complete, keeping findings and evidence limits accurate.
Do not run A1/A3–A7, fixes or a release architectural-review receipt as part of this review.
