# Agent-surface findings — systemic fix strategy

Assessment date: 2026-10-19. Source snapshot: `678beec060bc3c29e106fd7ae5f077351bbdecb4`.
Inputs are the completed ASR-A–I records in [HANDOFF.md](HANDOFF.md#review-findings).
The range `4fbeb316..678beec0` changes only that handoff, so intervening product changes do not
invalidate the reviews' source evidence. This assessment adds targeted source traces and fix
recommendations; it does not claim a new provider/device run or an implemented refactor.

**Formalized 2026-10-19:** the human requested implementation-ready refactor designs and consolidated
fix batches. [AGENT-SURFACE-FIX-UNITS.md](AGENT-SURFACE-FIX-UNITS.md) now assigns all 41 findings
to three ready architectural changes and six fix units, with normative planned requirements,
acceptance and closure rules. The assessment below records the reasoning that led to those scopes;
its proposal language is historical. No implementation or finding closure has occurred.

## Main conclusion

Several findings are symptoms of shared ownership defects and should be fixed as coordinated,
bounded changes. A wholesale runtime rewrite, a new workflow engine, or event-sourcing every store
is not justified by this evidence. The existing architecture already calls for shared composition,
generation ownership, single-writer conversation history and durable retry obligations. The code
does not consistently make those guarantees unavoidable at its boundaries.

The important combined risk is **loss of control-state integrity**: an agent may receive a different
working contract than Chuck records, consume input that Chuck later requeues, or finish work while
Chuck loses the evidence or next recovery owner. These are concrete ways to make a capable model
do wrong, duplicate or stalled work. They take priority over tuning persona wording or attributing
a general quality regression to the model.

ASR-I already reconciled the review coverage and known contradictions. The additional judgment
here is which fixes need a common owner, which should stay local, which consequences deserve
earlier triage, and which tempting cross-system fixes would introduce new problems.

## Urgency and broader consequences

| Priority | Established findings | Combined consequence and immediate fix direction |
|---|---|---|
| First: safe process identity and control | ASRC-3; ASRB-4/5/6/7; ASRA-4 | Stop/Switch/Archive can kill an unrelated reused PID; external terminal execution may lack a controllable process/exit identity, or receive stale launch environment and hook identity. This is host/agent isolation and control safety, not a presentation issue. Corroborate ownership before destructive signals, contain descendants, and refuse a driver/operation whose required control or per-child environment contract cannot be established. Actual cross-agent tmux contamination is source/contract-backed, not a captured incident. |
| First: no automatic replay of possibly consumed work | ASRH-4; ASRD-4; ASRH-2 | One lost acknowledgement can produce a second native input without any deliberate new request, or restore the wrong queued input. Repeated instructions can cause repeated edits, commands or external actions; those downstream effects depend on the task and were not reproduced here. Preserve uncertain consumption for explicit recovery, deduplicate annotation retries, and withdraw the authoritative held message. |
| First: keep Stop and permissions usable during settings changes | ASRC-1, compounded by ASRC-2/5 | An ordinary in-turn effort/fast change can block the read loop and hold the lifecycle claim, leaving output/approval frames unread and Stop returning 409. The immediate unlocked-call correction should not wait for a larger state-machine refactor. |
| High: trustworthy event settlement and recovery | ASRD-1/2/3; ASRF-1; ASRG-1/3 | Sequence collisions, out-of-order/non-durable events and lost post-result actions can make visible/native/room/task state disagree. A successful native turn or task result is not proof that its controller has a recoverable next step. Fix accepted-work recovery without rerunning already accepted stage work. |
| High: preserve the actual native contract | ASRA-1/2/3, PROMPT-1; ASRB-2/3/8; ASRG-4 | “Same model/persona/session” in Chuck is not sufficient evidence of the same native setup after resume/fork/check. Wrong instructions, model or source selection can masquerade as model-quality degradation. Validate native execution choices before drawing persona-level causal conclusions. |

These priorities strengthen the impact assessment of existing findings. They are not new duplicate
finding IDs, an assertion of an observed incident, or permission to disable a feature without the
appropriate fix/product decision.

## Where a wider refactor is justified

The sizes below are relative scope estimates, not implementation commitments. Preserve the existing
finding IDs and their regression obligations when coordinating work across units.
The strongest structural refactor cases are 1 (event commit), 2 (execution contract) and 5
(process/driver ownership). Items 3, 4 and 6 are coordinated repairs at existing state/receipt/retry
seams; they do not need separate architectural projects.

### 1. One conversation event-commit boundary

**Group:** ASRD-1/2/3, with ASRD-5's authoritative-reader restriction.

**Size:** medium to large, across runtime emission, active/inactive append callers and server
consumers. This is more than adding a mutex to one Writer.

The transcript writer's mutex is instance-local. Runtime sequence allocation, append and publication
are separate steps; switch markers and inactive annotations can open another writer. Those are three
manifestations of the same missing ownership boundary. Extend the existing per-conversation
persistence seam so one owner coordinates sequence allocation, ordered append and ordered publication
across live events, markers, annotations and notices. Fence inactive append against resume/switch.
Keep reconciliation restricted to the canonical normalized log and bounded reads.

**Additional source-backed consequence:** `emitChecked` in
`internal/runtime/chat.go:2080–2129` calls the hub and external sink after non-required persistence
failure. `internal/server/server.go:412–435` uses that sink for quota progress, room activity,
permission processing, task turn-end release and room finalization. This is broader than output
disappearing after reload: control callbacks are not uniformly backed by successful transcript
persistence. The code path is verified; a new combined storage-failure/automation scenario has not
been executed. This strengthens ASRD-3 rather than creating another finding.

**Do not fix this by blindly suppressing every callback.** Task reports and their pending release
are independently committed in SQLite. `dispatchTurnEnd`
(`internal/server/task_dispatcher.go:466–510`) releases those tasks; room finalization has separate
SQLite capture/result authority. A global transcript-failure gate can delay required cleanup or
leave a room waiting. Distinguish provisional observation, committed conversation history,
provider terminal outcome and independently durable control obligations. Carry persistence outcome
or use explicit control callbacks; each consumer must check its own required durable authority.
Append, sync and index success need distinct treatment where relevant: `persistEvent` ignores its
extra turn-end Sync error and logs index failure while returning true. The concrete
`transcript.Writer.Append` already syncs turn-end records, so this is not evidence that every
turn-end is unsynced; an index failure also does not imply the canonical log is lost.

**Why now:** separate D1/D2/D3 patches can leave the next writer or consumer inconsistent, while a
single broad lock across a reentrant callback can recreate ASRC-1's liveness failure. Preserve
SQLite's existing authority for tasks/rooms; do not make the transcript a replacement workflow DB.

**Exit evidence:** increasing unique durable and observed sequences across switch/annotation/resume;
append/sync/index failures with explicit history state; separately correct task/room cleanup and
safe progression; no lock held across a reentrant provider or controller callback.

### 2. One frozen execution contract with explicit launch purpose

**Group:** ASRA-1/2/3, PROMPT-1, ASRB-3, ASRG-4; ASRB-2 is an adjacent native-session prerequisite.

**Size:** medium to large, across the existing launch/snapshot resolver and provider preparation.
Extend LaunchSpec and current helpers only as needed; do not build another configuration service.

The same information currently has several interpretations: catalogue model identity, effective
provider model, redacted federation provenance, source/profile selection, inherited native guidance
and process-only overlays. Metadata can describe a source the child never uses. Continuation can
reconstruct a different model from the display/catalogue key.

Preserve one authoritative resolved selection, with explicit native-inherit versus explicit value
semantics, through persistence and ordinary continuation. Provider preparation must realize the
chosen native root/profile and supported layer chain or refuse before side effects. Keep authentication
material out of snapshots and diagnostics.

Also make operation purpose explicit at this seam. Fresh/load, native fork and isolated quota check
do **not** have the same instruction contract: normal work supplements native guidance; fork inherits
the source conversation without a new persona/primer; a probe excludes work guidance and authority
while preserving verified routing/account/model selection. Sharing every injection unconditionally
would reproduce B3/G4. Conversely, a generic blacklist of instruction keys is fragile across native
layers and versions.

**Why now:** fixing each wrong field independently leaves three competing meanings of “the agent's
setup.” The required distinction is between frozen user contract and purpose-specific process
realization, retaining the existing frozen/runtime split rather than abandoning it.

**Exit evidence:** actual child selection and native request agree with the snapshot through
fresh→resume/wake→switch/rollback, nested project/profile layers, clone and isolated check. Test
inherit/empty/default explicitly. Observe native consumption separately from adapter readback.
Claude's missing fork load can receive its bounded correction without waiting for this whole refactor.

### 3. Turn-owned terminal decisions and status

**Group:** ASRC-2, ASRG-3 and the settlement/publication portion of ASRC-5.

**Size:** medium, within the existing turn gate and status/permission/completion helpers.

Generation/turn checks already exist, but cancellation intent, provider completion, pending human
permission and status updates do not share one authoritative settlement decision. Native frames can
overwrite a human gate or reassert busy after completion; a late successful same-turn reply can
settle a claimed Cancel as quota success.

Make the terminal/cancellation decision monotonic for that turn and derive status from that outcome
plus current permission ownership. Revalidate provider results before quota/mail/held-successor
effects. Retain the documented difference between Cancel and Stop: Cancel does not prohibit all
future deliberate or already allowed held follow-ups.

**Why now:** C2 and G3 would otherwise acquire separate boolean checks that can drift again.
ASRC-1's locked RPC is an immediate local correction; it does not require replacing the runtime
with an actor system. Keep provider calls outside state locks and merge under a rechecked owner.

**Exit evidence:** cancellation versus late success, parallel child/tool frames during approval,
turn finish versus background output, and current-turn successor transfer have one terminal owner,
truthful status and the correct quota/mail settlement.

### 4. Explicit request identity and acknowledgement uncertainty

**Group:** ASRD-4, ASRH-2/4; ASRH-1 should reuse the existing draft-revision fix locally.

**Size:** medium, coordinated runtime/API/client work using existing receipt patterns.

A turn ID, HTTP request lifetime, held-message version, annotation batch ID and durable acceptance
receipt are different identities. One cannot substitute for another. Extend the attachment/room
command receipt approach where a retryable action needs stable command identity. Annotation retries
must reuse the same identity; a deliberate new action receives a new one. Withdrawal must compare
or atomically return the current authoritative hold rather than revive a cached draft.

Steer additionally needs a definite-no-consumption versus accepted versus unknown outcome. An HTTP
timeout cannot prove the native input was rejected; unknown content must not return to an automatic
successor queue. This property cannot be supplied by a database uniqueness constraint alone.

**Why now:** these are repeated manifestations of ambiguous delivery treated as safe replay.
A shared vocabulary and small receipt/admission seam are justified; a universal transactional
“exactly once” wrapper around native providers is not. Native tools/external effects cannot
automatically be rolled back after a lost response.

**Exit evidence:** lost responses before/after acceptance, reload, two-device replacement,
late acknowledgement and overlapping withdrawal preserve exact content/ownership and intended native
submission counts, without automatic replay of an unknown outcome.

### 5. Enforceable process and driver ownership

**Group:** ASRB-1/4/5/6/7, ASRC-3/4; apply existing containment to ASRA-4.

**Size:** medium to large, concentrated in driver/registry identity, exit/teardown and persisted
native session handling. Immediate unsafe signaling/unsupported-driver containment is smaller.

The common driver interface claims interchangeable execution, while implementations can return
without an exit signal, usable process identity, unique machine name or guaranteed child environment.
Meanwhile hook-discovered native session identity is not always promoted into the durable conversation
snapshot before Stop removes the running row.

Require the needed ownership/control capabilities when accepting a launch: generation-scoped native
process/driver identity, per-child environment, exit notification or corroborated monitoring, supported
cancellation/termination and durable native conversation identity. Use stable agent/generation IDs
for machine handles and display titles only for presentation. Extend the existing generation-scoped
teardown to every confirmed exit. Refuse a driver or operation until its required contract can be met.

**Why now:** fixing tmux naming, PID 0 and retained registry claims independently leaves the same
partially implemented driver considered fully usable. Corroboration before destructive reaping
and whole-process-tree containment are safety fixes, not optional later cleanup.

Do not put worktree readiness into a generic process manager. ASRA-6 needs every start to join the
existing per-project recreation/setup completion barrier, even after the directory appears.
That is a bounded local coordination fix with the same ownership principle.

**Exit evidence:** two concurrent terminal agents get distinct identity/environment, native
SessionStart survives Stop/restart, exit releases the exact generation, reused PID cannot authorize
a kill, and cancellation/deadline contains descendants. Credentialed/iTerm/tmux evidence must be
explicit; a fixture cannot establish actual external driver control.

### 6. Retain recovery obligations until the owning transition commits

**Group:** ASRF-1, ASRG-1/2/5; coordinate ASRE-1 at its separate continuation projection.

**Size:** medium, across existing dispatcher/pipeline/room retry and assignment helpers.

Keep the accepted business result, the runtime cleanup, and the controller's next action as separate
obligations. Completing cleanup must not erase the retry selector for a failed post-result stage
transition. A room attempt prepared before provider submission must be abandoned or retain a
specific recovery owner if startup fails. Eligibility and capacity must be checked before spending
a quota-check allowance, and failed/cooldown cleanup records must not starve later work.

Reuse the durable pending action/retry machinery and ordinary room abandonment path rather than
creating a generic saga or another scheduler. Each domain retains its exact generation/attempt/
Stop fence. Preserve ASRE-1's bounded authorized watched-change snapshot through assignment
confirmation; a wake notification alone is not the continuation payload.

**Why now:** a healthy agent or durable success result cannot repair an action that no controller
will select again. Conversely, rerunning accepted agent work is the wrong repair for failed
bookkeeping. A shared recovery rule is valuable; merging task, pipeline, room and quota states into
one generic state machine would hide their different authority and terminal semantics.

**Exit evidence:** inject faults between accepted result, release, next-stage/final/approval writes
and restart; remove faults and observe recovery without rerunning accepted work. Exercise room
start failure, full capacity, permanent earlier cleanup failures and watched-change continuation.

## Keep these fixes local

- ASRA-2's empty-reset normalization, ASRA-5's owner-mode preservation and ASRA-6's readiness join
  can be small existing-seam changes even when checked in the broader execution-contract matrix.
- ASRB-2's fork load, ASRB-8's source/release capability mismatch and ASRC-1's unlocked configuration
  call need direct corrections and independent native contract evidence; they should not wait for
  an all-provider framework.
- ASRH-1's draft revision check and ASRH-3's complete/error-aware archive lookup are bounded fixes.
- ASRC-6, ASRD-5 and ASRE-2 need correct admission/retention bounds at their own data boundaries,
  not a universal resource-budget service.
- ASRF-2's dead executor should be removed and its tests redirected to the actual task dispatcher.
  Retaining it while adding new common lifecycle code would preserve a misleading parallel policy.
- ASRC-7 is a specification/outcome alignment decision at the existing reconciliation helper.
  It does not justify redesigning startup recovery.

## Further combined conclusions and their confidence

**Verified source-level amplification:** the ASRD-3 external sink drives control callbacks as well
as UI delivery. Both durability-gating everything and treating every live event as durable are
unsafe simplifications. Different consumers must use their actual authoritative record.

**Plausible combined workflow, not an additional proven bug:** a standing stage owner can lose a
managed child's watched result at continuation (ASRE-1), later report its own accepted stage result,
then encounter stranded post-result progression (ASRF-1). Together this weakens both informed agent
recovery and automatic controller recovery. The combined sequence has not been reproduced.
A parent directly watching the same pipeline stage task is **not** a valid demonstration: current
task-wait authorization excludes those stage sources. Do not promote that tempting shortcut into
a new finding.

**Evaluation consequence:** requested settings, persona names and transcript appearance cannot
establish a controlled native-versus-Chuck comparison while A/B/D/H contract and replay defects
are active. Bounded contract/adoption probes are useful now; persona ablations should record/exclude
these correctness failures and first establish equal native configuration and intended submissions.
An aggregate comparison can still measure total harness harm, but cannot attribute it to persona
wording or reduced model reasoning without controlling those mechanisms.

**Testing consequence:** Claude fork's invented fake registration, source Codex's missing release
patch, exact-cwd-only native instruction fixtures and dead-stage-executor tests have one common
problem: passing tests can confirm Chuck's assumptions rather than the executed system. Require
independent pinned-native contract fixtures for the corrected boundary, real dispatcher entry paths,
and bounded authorized live receipts for adoption claims. Promote the temporary deterministic
review probes into durable regression tests when fixing their findings; do not depend on `/tmp`
surviving. This is a test-conformance correction, not a new testing platform.

## Recommended sequencing

1. Contain unsafe process/driver identity and ambiguous automatic replay; unblock configuration
   calls so approval/Stop remain usable. These narrow safety corrections need not wait for refactors.
2. Coordinate event-commit ownership and turn settlement with their controller consumers.
3. Coordinate frozen/native execution selection and explicit fresh/load/fork/check purpose.
4. Repair durable progression/room compensation, continuation payloads and cleanup fairness.
   Local bounds and client defects can proceed independently where their contracts are stable.
5. Close the associated native/provider/device receipts and then interpret the bounded performance
   comparison with known harness faults accounted for.

This is a recommendation, not a scheduled implementation queue or a new set of fix findings.
Existing ASR units and severities remain authoritative; decide the coordinated fix scope before
implementing it, update relevant specs if behavior/architecture changes, and retain each original
regression criterion. No product code/specification was changed, provider called, refactor selected,
or generic quality regression established by this assessment.
