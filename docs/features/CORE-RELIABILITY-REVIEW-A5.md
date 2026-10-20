# CR-A5 — Supervision and result authority

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`b58708c6`. This maps **A5 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, current callers and existing test assertions;
it performed no audit, failure probes, test execution or product repairs. ASW-1 fixes were landing
during planning; refresh their live ledger and relevant dirty work before starting the review.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Does the right accepted outcome advance exactly the intended work, only after the required
settlement, cleanup and approval? Can interruption or a failed bookkeeping step recover that
progress without repeating successful agent work or mistaking incomplete work for success?

Trace ordinary report acceptance → matching execution settlement → cleanup/progression, and room
staged intent → matching completion/canonical publication → bound task acceptance → progression.
Both paths retain their required cleanup/approval before actual next work. Prioritize false
completion, lost accepted results, stranded progression, wrong-attempt
acceptance, surviving work after closure, and stale cleanup stopping a successor or private turn.
Status text, an idle agent, fluent assistant output or a staged tool receipt cannot substitute for
the domain's acceptance transaction.

A5 owns the task/pipeline/room authority join, including standing owners, managed coordinators,
descendant closure, judge publication and retained progression. Follow Codex and Claude through
shared host/runtime boundaries without repeating general admission, content or provider reviews.
One reviewer integrates ordinary stage and room-backed stage evidence; these are not separate
component audit jobs.

## Read first

Read the current handoff's Current position and Active change, this charter and related findings.
Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a commit-diff
scope. Read requirements before following source:

- FS-14.R61–R78/R81–R86 and TS-09.R35–R49/R51–R56: current task-backed cursor, standing/coordinator
  authority, immutable results, Continue/Retry/Replace/Stop, closure and room-output acceptance.
  Read R87–R89/TS-09.R57–R60 only where standing instructions distinguish owner from descendant.
- FS-16.R3–R4/R16–R18 and its persistent-assignment/stage extensions; TS-10.R7–R10/R12/R15–R19/
  R25–R35/R38: shared result vocabulary, membership, execution handles, lineage, cleanup and
  room-backed task target. A2 owns admission; here establish result and release authority.
- FS-21's contribution, budget, opening, closing and judge contracts, including R48–R55;
  TS-14.R3–R14/R19–R29: attempt/turn/read ownership, successful finalization, concurrent openings,
  pipeline origin, retained judge results and later projection/input extensions. Read status notes:
  planned acceptance evidence is not evidence that a shipped behavior passed a native gate.
- TS-01.R39, TS-02.R52 and TS-04.R100: canonical history and typed control outcomes are distinct.
  Ordinary task cleanup can follow independently committed task authority despite a failed history
  append; honor reporting-turn settlement/tool-reply timing and exact effect guards independently
  of that append's outcome. Ordered turn settlement separately fences a held successor. Room
  publication still requires committed canonical source records/capture. Do not impose one
  storage-success predicate on both domains.
- FS-01.R10 and current shared Resume contracts apply to task continuation. TS-09's older sections
  retain pre-replacement Retry language, and R38 still describes a native-load fallback; compare
  that wording with the later shared refusal contract/current implementation. Do not introduce a
  fresh native conversation as an assumed recovery oracle; record remaining specification tension.
- Read the invariant trigger index and matching bodies, likely §§1/2/4/5/7/9/10/11/15/16/17,
  plus §8 for completion/recovery claims. TS-06.R35 governs proportional evidence; adequate
  existing proof can justify no new test.

Later shipped task-backed requirements govern the current pipeline model; old version-1 shapes
or a legacy method's continued presence do not make it the live execution path. Requirements,
not this charter or old review prose, govern expected behavior.

## Keep the authorities and recovery controls distinct

- An ordinary stage is a durable task assigned to the standing orchestrator. A dedicated
  coordinator is its managed child; descendant/run lineage does not grant stage-report authority.
  A child outcome, idle/exit or group label cannot complete the standing assignment.
- Task start, accepted task result, reporting-turn settlement, runtime release, stage closure and
  terminal run result are separate facts. Required cleanup includes unfinished descendants, not
  merely the reporter. Cleanup retry repeats retained effects/bookkeeping, not accepted work.
- Continue after failure/blocked creates another stage assignment on the same standing identity;
  interrupted Retry uses the same task/current assignee. Explicit Replace changes the owner via a
  fenced replacement assignment and retains useful child work/history. Approval Continue releases
  an approval pause. None rewrites an immutable accepted result.
- A room submission is staged intent. Only matching successful execution/capture finalization
  publishes and charges the appropriate allowance. Missing submission, cancellation, truncation
  or capture failure cannot become a contribution from incidental assistant text.
- Independent openings can run concurrently, each with its own token/head/generation/turn.
  Their bodies/activity remain withheld until ordered barrier or partial-End publication.
  Discussion/closing/judge retain a single floor; singular actor fields cannot describe several
  active openings. Pass, departure and declined closing have their specified distinct accounting.
- A completed room does not establish agreement or goal achievement. An optional judge follows
  discussion completion; pipeline room stages require its published synthesis. Stage `success`
  there means the synthesis was produced and accepted under the contract, not that it passed an
  inferred quality test or that participants agreed.
- Standalone Pause/End settle room-owned participation without stopping unrelated private work;
  room completion/deletion does not itself stop participants or judge. Run Stop additionally owns
  exact bound-room setup/turn/cleanup obligations. Quota continuation creates a linked new attempt,
  never permission to publish a failed attempt's old staged disposition.

## Starting authority and lifetime map

Verify this map against the selected HEAD; callbacks/SSE are hints or projections, not truth.

| Fact / obligation | Authority and producer | Consumer and boundary |
|---|---|---|
| Run definition and current stage | Immutable run template/inputs/runtime snapshot, run revision/cursor and `pipeline_stage_tasks` association | Pipeline manager selects only the current stage task. Live template edits, agent names and stale attempts cannot redefine its work. |
| Assignment/report authority | Task assignee/generation/execution handle and current stage-task binding | Shared acceptance transaction validates caller, expected revision/current task, outcome and declared outputs. Lineage alone never selects the stage branch. |
| Accepted ordinary result/output | Immutable task result/named outputs, source provenance and run projection | Result, release/closure intent and run revision commit together. Values are opaque text, not copied or verified filesystem artifacts; invalid/over-limit reports install no partial success. |
| Runtime/descendant closure | Durable release/yield/cancel intent, effect identity, lineage and run/stage closure fence | Accepted task result/pending-release authorizes guarded cleanup, preserving reporting-turn timing even when terminal history fails. Turn settlement fences successors; room source failure separately prevents publication. Finish only after obligations settle; never cancel newer unrelated work. |
| Progression/approval | Run pending action/revision, unique next-task key and acceptance/cleanup retry state | Reconcile/Continue/Retry perform the declared transition once. The accepted result is not its own progression receipt; a failure must retain who will try the next step. |
| Room attempt and staged intent | Room opportunity, member/role, token, source generation/executing turn, context head/read progress and staged reply | Submission receipt is not publication. Full required view must be delivered under this attempt before submission; retries do not adopt another token or claim model memory from a read watermark. |
| Published room data | Finalization transaction, room entries/activity/checkpoints/allowance, opening staging/barrier | Only the correct successful attempt publishes. Private turns, failed capture and unpublished openings cannot become canonical contributions or advance judging. |
| Retained judge result | Atomic canonical synthesis plus judge-agent-owned `think_tank_results` read projection | Survives room deletion under its own history retention, without injecting a provider assistant event or changing native history/search. Projection identity/anchor is distinct from provider seq and room entry. |
| Room-stage acceptance | Exact run/stage/task/room binding and finalized synthesis entry, pending `accept_room_output` | Trusted host branch accepts that exact synthesis into the common task-result transaction. A published room result alone is not an accepted stage result; failed acceptance retries the same entry, not judging. |
| Terminal shared work result | Immutable normalized `work_results` registration for task or terminal pipeline run | Prerequisite consumers use the correct source. A stage report is not a terminal run result; allowed run deletion cannot erase the retained registration or unrelated task history. |

## Shipped paths to trace

| Path | Current source entry points and question |
|---|---|
| Run start/current task → actual execution | `internal/pipeline/{manager,actions}.go`, `internal/state/pipeline_tasks.go`, `internal/server/task_dispatcher.go` | Run/stage task commits precede launch; the ordinary dispatcher starts/continues it. Old `LaunchStage`/`ContinueStage` methods are not current production-path proof (ASRF-2). |
| Ordinary task or stage report → release → next work | `internal/messaging/{task_tools,pipeline_tools}.go`, `internal/server/{task_handlers,task_dispatcher}.go`, `internal/state/{tasks,pipeline_tasks}.go`, `internal/pipeline/reconcile.go` | Follow actual accepting transaction, captured terminal observation, exact cleanup and unique successor insertion, not merely manager helper state. |
| Coordinator/descendants → stage/run closure | Task lineage/wait/cleanup helpers, `internal/pipeline/{actions,reconcile}.go` | Stage-owner judgment is separate from subordinate outcomes. Follow ordinary create/admit/repair/wake callers far enough to prove closure fences and retained child ownership. |
| Room setup/openings/discussion/closing/judge | `internal/server/{think_tanks,think_tank_capture}.go`, `internal/state/{think_tanks,think_tank_turns,think_tank_activity}.go`, `internal/messaging/think_tank_tools.go` | Trace exact attempt/turn admission, read receipt, staged submission and successful publication, including concurrent opening exceptions. |
| Synthesis publication → agent projection → bound task acceptance | `internal/state/think_tank_results.go`, `internal/state/pipeline_tasks.go`, `internal/pipeline/rooms.go`, `internal/server/think_tanks.go` | Identify each transaction and its retry owner. Agent-history display is a read projection; acceptance requires the exact finalized room entry and current open binding. |
| Stop/Replace/restart/acceptance repair | `internal/pipeline/{actions,reconcile,rooms,manager}.go`, `internal/server/{server,task_dispatcher,pipeline_lifecycle}.go` and room recovery methods | Inspect actual startup ordering/worker overlap, closure claims, pending progression/approval and automatic bounded cleanup; direct recovery-helper calls alone cannot establish it. |

Pipeline rooms use trusted origin and fresh participants/judge in the run project. Standalone room
attachments/judge instructions and private turns have their own contracts; do not infer those inputs
or broader authority from a pipeline binding. Generic task or stage controls cannot bypass the
room-specific recovery path. Pipeline deletion releases a room pin, not its history or agent data.

## Connected searches and useful scenarios

These are facets of one review, not separately queued checks or a required Cartesian test matrix.

| Scenario / search | Evidence that distinguishes authoritative progression from apparent completion |
|---|---|
| Standing owner reports while coordinator/descendants still have work | Compare exact task/handle/run revision, immutable result and output provenance, matching terminal observation, descendant closure and actual next-task execution. Neither a child report nor reporter idle/release alone permits advance. |
| Duplicate/stale report or terminal callback arrives after same-agent reassignment/Replace | Compare captured generation **and executing turn/task identity** at acceptance and effect time. Old observations cannot release a newer task or stop its runtime. Reuse ASW1-6/9 ownership for those known mechanisms. |
| Result commits, then history/index/control bookkeeping or progression fails | Separate durable task authority from canonical room-source requirements. Verify retained next action/cleanup and live or restart reconciliation without reporting again, re-prompting accepted work or publishing unsupported room output. ASRF-1 and ASW1-4 remain existing owners. |
| Success/blocked/failure → cleanup → approval/Continue/Retry/Replace | Compare old/new task and agent identity, frozen assignment/prior outputs and unique cursor transition. Accepted reports remain unchanged; rejected reports remain actionable; an approval or explicit replacement cannot be simulated by ordinary input or a child outcome. |
| Stop commits while descendant creation, setup/start, wait wake or room acceptance is in flight | Inspect closure CAS, lineage selection and retained release effects, then native starts/cancels and final run/shared result. No late child escapes or stopped run accepts old synthesis. Cleanup repair never retries accepted agent work or kills a later private turn. |
| Independent openings overlap; one fails, pauses, ends or restarts | Track all active attempts, withheld bodies/activity, per-member counters and ordered/partial barrier. Completed peers are not re-executed. Singular actor projections cannot silently assign another opening's ownership. |
| Room actor reads/submits, then provider/capture/completion fails or a private successor runs | Compare read head/checkpoint, staged receipt, canonical source/capture and successful finalization. No allowance, contribution or judge result from a failed/uncertain attempt; private/child activity retains origin ownership. Missing submission cannot be replaced with fluent assistant output. |
| Published judge synthesis → stage acceptance fails or exceeds downstream bounds | Compare exact room/entry/attempt/body, retained agent projection, pending action/revision, task result/output and eventual successor. Retry acceptance of that entry only; judge retry is a distinct explicit recovery before successful publication. |
| Discussion ends through cap/pass/departure/End, then judging or quota continuation intervenes | Verify the actual completion reason, finite allowance/read checkpoint and allowed next phase. A resumed linked attempt cannot publish prior staged text. Synthesis success does not become a claim that the development goal was achieved. |
| Completed run/room is deleted after accepted work | Compare immutable retained task/shared results, judge read projection, source-unavailable framing and native transcript before/after. Removal cannot erase prerequisite truth, silently rewrite provider history or fabricate a different source anchor. |

Prefer a few connected disposable-home journeys: ordinary standing stage → accepted result →
interrupted release/progression → recovery → actual successor, plus room publication → delayed
stage acceptance → approval/next ordinary stage. Add focused counterexamples only for distinct
authority/recovery boundaries. Count accepted results, native starts and exact owner/effect IDs,
not merely callbacks or status updates. Do not multiply every provider/outcome/control permutation.

## Reuse these proofs before adding any

These tests were researched, not run or certified. Inspect what fixtures actually establish.

| Seam | Existing starting proof and its limit |
|---|---|
| Stage association/report | `internal/state/pipeline_tasks_test.go`: `TestAcceptPipelineStageTaskResultFencesHandleAndYield`, `TestTaskResultOutputsAreImmutable`, `TestClosedRunFencesEveryTaskWriter`. State transactions do not alone prove actual terminal delivery or successor execution. |
| Ordinary progression/controls | `internal/pipeline/manager_test.go`: `TestStageCleanupCancelsDescendantsBeforeAdvancing`, `TestContinueFailureCreatesAnotherDurableStageTaskForStandingOwner`, `TestReplaceInterruptedStandingOwnerFencesOldTask`. These drive manager/state fixtures, not a complete native report-to-turn-end journey. |
| Restart/cleanup owner | Same file: `TestFailedStartupRecoveryRetainsTheStopFence`, `TestRepairCleanupClaimsTheRevisionBeforeEffects`; `internal/server/task_dispatch_test.go`: `TestRestartInterruptsStageAndPausesRunForRetry`. Missing-report interruption and direct recovery fixtures do not prove restart after an accepted report before progression completes. |
| Terminal result retention | `internal/state/pipelines_test.go`: `TestTerminalPipelineRunsRegisterTheirOutcome`; `internal/state/tasks_test.go`: `TestATaskArmedOnAPipelineRunIsReleasedByItsOutcome`. Retained registration and downstream evaluation are different assertion boundaries. |
| Opening ownership/barrier | `internal/state/think_tank_openings_test.go`: `TestThinkTankConcurrentOpeningsPublishInOrder`, `TestThinkTankRestartFencesConcurrentOpenings`; `internal/server/think_tanks_test.go`: `TestThinkTankOpeningsRunConcurrently`. Concurrency fixtures do not cover every failure/cancellation interleaving. |
| Capture and judge finalization | `internal/server/think_tank_capture_test.go`: `TestThinkTankOpeningActivityIsWithheld`, `TestThinkTankUncommittedSourceFailsAttempt`; `internal/state/think_tanks_test.go`: `TestThinkTankJudgeLifecycle`. The latter proves exact retained judge projection and room/agent deletion split, not unchanged provider transcript bytes. |
| Room-stage acceptance | `internal/pipeline/rooms_test.go`: `TestMixedRoomStageAcceptsExactSynthesisOnce`, `TestRoomStageApprovalWaitsAfterAcceptance`, `TestRoomOutputAcceptanceBudgetSurvivesManagerRestart`, `TestRoomOutputRefusalAndStopFence`. Check which effects are seeded/faked rather than inferred from the test title. |
| Current integrated room path | `internal/server/pipeline_rooms_test.go`: `TestPipelineThinkTankStageEndToEnd`, `TestPipelineStopRoomPreservesPrivateTurnOnSameGeneration`. Fake-provider/server proof does not close live Claude/Codex room adoption or all source/capture races. |

The main questions are the combined current report/publication → settlement/cleanup → progression
path, restart after an accepted result, and stage acceptance of retained synthesis without another
judge turn. Existing proofs narrow those searches; no test-name inventory establishes correctness.
Use domain requirements and committed before/after state as the oracle. Preserve the owed FS-16.A33,
FS-14.A49–A50 and FS-21.A31–A32 native/race evidence and later opening/result gates as applicable;
fake peers and repeated source reading cannot certify them.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md) and live handoff, including concurrent
ASW-1 closure rather than historical header counts. Reuse ASR-E/F/G and
[original joins J3/J4/J5](AGENT-SURFACE-REVIEW.md#mandatory-joins-between-reviews).

- ASW-1 owns existing commit/control-generation failures, notably ASW1-4's forgotten capture
  failure and ASW1-6/9's ordering/stale cleanup identity while open. ASF-1 owns turn/cancellation
  settlement; ASW-3 is closed for process/driver ownership. Add corroboration, not another owner.
- ASF-3 owns ASRF-1's lost post-result pending action, ASRF-2's obsolete executor/wiring proof,
  ASRE-1's lost watched context and ASRG-1's quota-room compensation (plus its other recovery
  findings). ASF-4 owns result bounds. A different current result consumer or closure mechanism
  can be a new A5 finding; old reproductions or absent live receipts cannot.
- A1 owns history/snapshot repair; A2 eligibility/capacity/wake/retry mechanics; A3 executable
  environment; A4 exact content/knowledge delivery. A5 owns which accepted outcome authorizes
  which next work, retaining the cleanup/progression obligation across those seams.
- A6 will own broader operator action/observation truth; A7 sustained cost. Follow their code when
  it breaks A5's authority promise, but do not turn this into a general UI, CLI or performance audit.

Do not introduce another pipeline executor, scheduler, result store, generalized event system,
automatic judge retry or inferred-success rule. A valuable architectural remedy found by review
is a separately designed work item, not implementation during the audit.

## Required review result

Record exact reviewed HEAD, relevant dirty work and provider/adapter source versions. Produce one
concise ledger of report/publication authorities, execution identities, pending actions/cleanup,
current callers and connected scenarios. Distinguish staged, accepted, published, settled and
completed effects; keep hypotheses and unavailable evidence separate from findings.

Record real findings under **CR-A5** in the handoff using workflow §7: current location, credible
trigger, development impact, requirement/invariant and reproducible evidence or precise source-backed
mechanism. Give the unit one Fix model recommendation based on its hardest open fix. Do not treat
intentional explicit recovery or a bounded discussion ending as a defect merely because automatic
retry or a stronger quality promise might be desirable.

Use isolated temporary Chuck/native homes and loopback-only servers; preserve user data and dirty
work. Follow the review skill's allowed reporting rules; change no product code or normative specs.
Close only this unit, retaining findings and honest evidence limits. Do not run other audit units,
fixes or a release architectural-review receipt as part of A5.
