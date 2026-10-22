# Core reliability audits: before-fix synthesis

Prepared 2026-10-21 against `3c8b0cec`, after CR-A1–CR-A7 source reviews completed. Two focused
GPT-6 Luna readers reconciled the data/configuration/information and control/result/resource
findings; the parent read the combined findings and checked the specific maintenance, task-claim
and reconnect paths below. This is synthesis and bounded source research, not another complete
audit, failure probe, test run or native/device receipt. No product code or normative spec changed.

[HANDOFF.md](HANDOFF.md#review-findings) owns findings and closure; the
[audit plan](CORE-RELIABILITY-AUDIT-PLAN.md) owns charters and the proposed integration journey.
At entry the seven audits recorded 29 open findings. The dirty messaging handler/tests already
contained CR-A4-1's enum correction; it was not validated or closed here. Four dirty ready-change
documents were also preserved. Recheck current work before fixing any finding.

**Operator follow-up, 2026-10-21:** CR-A4 was picked up by another agent; its worker has now
closed the enum finding in `c515aea3` and the live handoff. That local correction is independent of
the proposed refactors below. Keep its validation/closure with the existing worker; do not
duplicate, reopen or absorb it into these recommendations.

## Conclusion

Local fixes can start. Interruption-safe reindexing warrants a separate preservation-boundary design.
Task runtime ownership through result/yield/continuation needs explicit cleanup/admission contracts
and coordinated fixes in existing seams; a standalone architectural refactor is not demonstrated.
The observable-state fixes also need one connected
recovery proof. The findings do not establish a need for a new global scheduler, event database,
generic action framework or prompt rewrite.

Most governing distinctions are already specified. The recurring implementation error is using an
easy proxy for authority: a rebuilt session row for retained native identity, terminal task state
for released runtime ownership, a checked project/profile for the actual child setup, or a healthy
stream for current state. Repair those consumer boundaries; another broad workflow rule alone would
not resolve them. The current purposeful-test and architectural-review rules already ask for the
independent boundary evidence that was missing.

## Architectural refactor recommendation

### Pickup readiness

- **CR-A1-1/2: design complete; ready as one `/work` unit.** The operator approved partial repair
  on 2026-10-22. The [waiting change](../ready-changes/interruption-safe-reindex.md) links
  FS-05.R43–R45/A26–A27 and TS-02.R56–R58, superseding R9's unsafe maintenance algorithm.
  One transaction per conversation covers deletion, replay and retained-authority restoration;
  failed sources preserve their full prior projections while healthy ones commit. Independent Astra
  challenge passed. Findings stay open until implementation/review; do not launch competing fixes.
- **CR-A1-3: available as an ordinary focused fix.** Its complete-record correction is separable
  from the reindex design; preserve the shared reader/writer contract.
- **CR-A2: available for a fix agent, with a required first design step.** Under workflow §8,
  first record the cleanup authorization/identity contract and continuation admission-before-input
  order, updating the relevant specification before coding. Choose/prove enforcement rather than
  assume existing fields suffice. Fix accounting before or with deferred cleanup. This stays within
  CR-A2; coordinate shared seams with ASF-3 without absorbing its findings or creating a separate
  ownership architecture project. No ready-change file is required merely to start that fix validation/design.
- **CR-A4 is closed.** Other open CR findings remain available through their existing fix units and
  normal validation/specification obligations; this assessment adds no global queue dependency.

**Primary judgment after source checks and an independent Astra challenge: design the reindex
boundary; require bounded task cleanup/continuation design within coordinated fixes, without a
standalone ownership architecture project.** The reindex design is now a waiting implementation
unit; the assessment itself does not start implementation or close findings.

| Area | Recommended scope | Size and assessment |
|---|---|---|
| Reindex transaction/recovery ownership — CR-A1-1/2 | Settled in TS-02.R56–R58: atomic replacement per conversation, failed-source preservation and explicit incomplete result. Existing replay helpers participate in one transaction; no staging/schema split. | Medium-to-large, difficult. Transaction routing spans metadata, events, tracking, cwd lookup, flush and retained-field restoration. The waiting change defines interruption/Resume and mixed-source proof. |
| Task execution ownership — CR-A2-1/2/3/5/9, coordinated with ASF-3 | No standalone architecture project demonstrated. First define cleanup authorization across callbacks/timer/restart and continuation admission-before-input delivery. Then fix accounting, membership and bounded rediscovery in existing paths, extracting only proven shared policy. | Medium repair effort with substantial race/recovery risk. Existing boundaries are useful, but current fields are not proven sufficient unchanged: cleanup provenance is lost through an empty argument and turn capture is best-effort after activation. A broader layer needs a concrete enforcement advantage before it is justified. |

The selected reindex design scopes transactions to one conversation, retaining completed repair
progress across interruption and avoiding archive-wide transaction lifetime. Source failures roll
back locally; database errors stop further repair while preserving prior completed transactions.
Staging adds no required capability for this offline policy. Keep the small complete-record fix
A1-3 independently executable.

For tasks, the ownership theme does not by itself justify another architecture layer. The existing
admission, cleanup and runtime gate are credible repair seams; that is not proof their present data
and APIs enforce every required distinction. Specify the missing cleanup/continuation contract,
coordinate A2-1/2 and ASF-3, and preserve each finding's focused evidence. Add narrowly scoped durable
data only if the required authorization cannot safely be reconstructed; no new task table, scheduler,
state-machine framework or separate broad refactor work item is currently recommended.

Publication/reconnect/deduplication needs coordinated fixes and one convergence proof, but the
current evidence does not warrant replacing the event bus. Workspace/profile findings call for
validating the actual composed execution through current seams. Scan limits/cancellation,
reconciliation eligibility and CR-A4's enum correction also fit existing owners without a new
architectural layer. Sections below describe the constraints for those decisions.

## Independent challenge record

At the operator's request, a GPT-6 Luna reviewer started with no conversation history and an explicit
mandate to reject unnecessary refactoring. It checked HEAD `72515c47`, the current findings,
requirements and source; only four unrelated ready-change documents were dirty. It made no edits
and ran no tests or probes. Its conclusions were:

- **Reindex: approve narrow design; high confidence in the need, medium in the eventual solution.**
  `reindex.go` commits reset before restoring SQLite-only inputs; `Indexer` opens separate
  transactions. The reproduced interruption and missing-source findings establish a real recovery
  boundary problem. They do not select a whole-rebuild transaction or staging design automatically.
- **Tasks: reject a separate refactor; high confidence.** Existing task claims, assignment fields,
  execution identity, lifecycle fences and ASF-3 seams suffice. The defects have concrete local
  causes: two incorrect capacity predicates, lost cleanup-turn identity, unconditional assignee
  clearing, bypassed continuation admission and absent bounded reap rediscovery. The earlier
  proposal supplied no concrete factoring benefit to justify coupling these into another layer.

The parent initially accepted that challenge and withdrew the task-refactor recommendation.
The operator challenged giving a discovery model that degree of architectural authority. The
frontier review below qualifies that conclusion; this record preserves the earlier reasoning
rather than presenting it as the final decision.

## Frontier challenge and primary judgment

A GPT-6 Astra reviewer started with no conversation history, checked `516488c5`, requirements and
actual source, and treated both the original proposal and Luna's verdict as contested arguments.
It made no changes and ran no probes/tests. Its recommendation was narrow reindex preservation
design plus coordinated task repairs with explicit ownership contracts. It found Luna's overall
disposition defensible, but its high-confidence claim that existing task identity/fences suffice
was stronger than the source supports.

The primary agent separately checked the transaction writers, task settlement, turn capture and
atomic stop gate. The resulting judgment is:

- **Reindex needs a dedicated preservation design.** Independent indexer commits make the current
  destructive boundary unsafe. TS-02.R9 itself prescribes process-local read-before-wipe and
  restore-after-replay, so this is also a design-spec gap, not solely code violating adequate rules.
  Revise that algorithm before implementation. Transaction-bound replay is a credible default to
  compare with staging; do not approve either without its failure/resource policy.
- **Tasks need bounded design inside the fixes; a separate ownership project remains unproven.**
  The shared cleanup and `StopGuardedIdle` seams already provide concrete places to enforce the
  promises. However, the timer's empty `endingTurn` loses deferred versus immediate authorization,
  and `recordTaskExecutionTurn` reads/writes turn identity after activation with logged-only errors;
  `SetTaskExecutionTurn` updates only `starting` rows. Neither a shared helper nor an existing field
  alone proves safe retry authorization. Define those distinctions and continuation delivery order
  before coding; demonstrate whether current outcome/pending fields plus the atomic idle gate are
  sufficient before introducing more persistent state.

This judgment is conditional on enforcement, not on how many requirements already describe the
intended behavior. If a concrete task repair cannot route callers through existing admission/cleanup
without duplicating policy or losing authorization, that would justify a narrow structural refactor.
Conversely, a proven mapping from existing durable fields and runtime gates to the cleanup contract
would keep the repair small. No new scheduler or blanket schema split is justified by current evidence.

## 1. Reindex must be one preservation boundary

**Findings:** CR-A1-1/2; CR-A1-3 independently owns the shared reader/writer complete-record fix.
**Size:** medium-to-large, difficult; transaction/executor and replay/error semantics span the indexer.

`internal/index/reindex.go` commits `resetTables` before replay and keeps the latest native ID,
effort and fast setting only in memory. These three inputs are not reproducible from old
`session_meta`. A repair process can die after deletion, while a missing root/log can commit an
empty replacement and report success. This is more serious than stale archive search: the repair
can remove the authority needed to continue development work.

Handle A1-1/2 together around a boundary that preserves prior usable state or publishes a complete
replacement. The 2026-10-22 design chooses **one transaction per conversation, including every
delete, replay, flush, query and retained-field restoration**. The current `Indexer` owns `*sql.DB`
and starts independent transactions (`internal/index/indexer.go`); wrapping only `resetTables`
cannot make replay atomic. TS-02.R56–R58 define the shared transaction seam and failure policy.
No archive-wide transaction/savepoints or durable staging is needed.

Atomicity alone does not fix missing input. Refuse a missing canonical root before mutation. For a
missing/unreadable agent source, preserve its existing usable snapshot or abort the replacement and
report the identity; do not commit its disappearance as an empty rebuild. Protect the three observed
SQLite-only fields rather than declaring every replayable session column new authoritative state.
Check existing retained identities whose entire source directory is missing as well as an absent
log inside a present directory; enumerating the filesystem alone cannot establish complete inputs.

Fix CR-A1-3's reader/writer newline agreement independently. Preserve malformed-middle tolerance;
all readers, replay and writer reopen must agree on the same complete-record prefix. A7's read-cost
work must retain this boundary rather than create a faster reader with different truth.

**Connected evidence:** kill a real repair after reset executes but before the enclosing rebuild
transaction commits; verify prior authority survives, then rerun and consume the result through
native Resume. For another preservation design, kill at its equivalent destructive boundary.
Separately combine a missing source with a healthy conversation.
Check retained overrides, visible failure/preservation and complete-record readback. An ordinary
successful replay, deferred restoration or catch-all error does not establish crash safety.

## 2. Task completion is not runtime release

**Findings:** CR-A2-1/2/3/5/9; coordinate the related liveness findings A2-4/6/7/8 with existing ASF-3.
**Size:** medium, multiple state/server/messaging paths; consolidate existing predicates and transitions.

The result row, reporting turn, runtime claim, assignment membership and capacity slot are different
facts. The current cleanup ticker interprets an absent observed turn as permission for immediate
Stop, while admission treats `finished` as free capacity despite retained ownership. Correcting only
the premature Stop makes the over-admission window last longer. **Close A2-2's accounting together
with, or before, A2-1's deferred cleanup; do not rely on premature killing to enforce capacity.**

Keep created/woke capacity until confirmed release/reap, including terminal pending cleanup, through
one shared claim predicate used by normal and quota admission. Distinguish live agent-report/yield
deferral from immediate person-result/cancel and restart orphan recovery. Every cleanup retry must
retain that distinction and captured execution identity; current registry state cannot replace it.
Preserve borrowed-runtime policy and independently committed results.

Before implementation, define deferred report/yield cleanup, immediate person/cancel cleanup and
restart orphan recovery explicitly. The current empty `endingTurn` sentinel must not grant new
authority to timer retries. `execution_turn` is recorded after activation and can be absent; do not
assume it is a durable report-turn settlement receipt. Existing outcome/pending fields and the
atomic idle-stop gate are a candidate minimal solution, subject to callback-loss, replacement and
restart proof. Introduce a narrowly scoped cleanup mode/identity only if that proof requires it.

A2-3's lost assignee and A2-5's mail/manual wake bypass show the same assignment/runtime distinction
on continuation. Preserve confirmed membership when abandoning a later reservation, and route input
for a waiting assignment through its existing continuation admission. Do not replace these with
untracked Resume, create another assignee, or copy mail's attempted-at-most-once policy to tasks.
Specify the ordering: coalesced continuation and closure checks, capacity admission, confirmed
execution, then input delivery. Retain mail's opportunity until its own attempt boundary.
A2-9 needs bounded rediscovery after transient reap contention rather than another process restart.
Rediscover only abandoned startup obligations; a live in-flight `starting` row is not automatically
an orphan merely because a dispatcher tick sees it.

A2-4/6/7/8 show why a row remaining durable is not sufficient recovery: missing watch reevaluation,
blocked oldest batches, settled error status and cancelled quota ownership can leave valid work
silent forever. Fair sweeps must advance without removing read/in-flight bounds. A2-6 and A2-7 can
compound: a prefix of indefinitely deferred recipients can also hide unrelated work. This is a
combined consequence of existing findings, not another finding or a reason for universal retry.

**Connected evidence:** hold a reporting turn after its accepted result under budget one, tick cleanup
and attempt ordinary/quota admission, then settle/release and admit the next runtime. Also yield,
deliver mail/manual input at full capacity, inject contention/restart and prove the same assignment
eventually continues with its saved context. Coordinate shared `tasks.go`/dispatcher changes with
ASF-3; its context/progression/quota findings retain their own IDs and closure evidence.

## 3. Published state, transport recovery and replay must converge together

**Findings:** CR-A5-1, CR-A7-1/3 and CR-A6-5. **Size:** small-to-medium coordinated fixes across existing
publisher, bus and browser seams; a new event architecture is not justified.

These are three different failure points: an accepted task result emits no source `task_update`;
a bounded SSE queue drops a state/invalidation event while healthy pings continue; and a transcript
refetch followed by the same streamed sequence applies text twice. Publishing more events cannot
repair silent overflow; reconnecting cannot fix a missing publication on a healthy stream; refetch
without sequence deduplication can manufacture duplicate visible content.

The repair contract is: publish the committed source mutation, explicitly recover from transport
loss, and apply reconstructed/live state once. Reuse the existing post-commit publisher, hydration/
invalidation and per-agent sequence seams. Keep bounded buffers and target/request fencing.

**Additional source-backed gap found during synthesis, owned with A7-1 recovery:** desktop
`SseClient.connect`'s `onopen` in `ui/src/api/sse.ts` invalidates pipeline run lists/proposals,
Tasks, rooms and remote-control state, but omits `PIPELINE_QUERY_KEYS.runDetails`. The keys in
`ui/src/api/pipelines.ts` put run-list and run-detail in separate families; `usePipelineRun` has
no polling interval. Both `task_update` and `pipeline_update` normally invalidate open run details.
If their event is lost while the stream disconnects or overflows, reconnect alone can leave an
open detail stale until another invalidation/remount/focus refresh. Agent hydration does not carry
the missing run detail. This is source-confirmed; no browser timing probe ran in this synthesis.

Include open run-detail invalidation in recovery, not just the agent snapshot or list. A7-1's original
bus probe did not prove query-cache convergence. Existing `ui/src/api/sse.test.ts` tests event-specific
invalidation and hydration separately. Phone uses its own hydration revision for Home/open Run
reads; do not assume the desktop cache change fixes or proves its transport behavior.

**Connected evidence:** commit a result with no dependents; lose a task/pipeline/state update while
other-agent traffic and pings continue; trigger the selected overflow recovery and an ordinary
reconnect. Verify Tasks, open run detail, relevant lists/rooms and agent controls converge to current
truth, while a transcript event present in both readback and the later live stream appears once.
Keep each original finding's local rejection/order proofs; one journey does not replace all of them.

## 4. Validate the setup that will actually run

**Findings:** CR-A3-1/2/3 (fixed 2026-10-22), coordinated with ASF-5's existing readiness fixes.
**Size:** medium shared-seam corrections; no new configuration architecture is warranted.

Directory existence is not owned Git identity; current project cwd is not frozen Resume cwd; the
server's preflight profile is not necessarily the child's environment-selected profile. The combined
requirement is to validate the actual checkout, execution cwd and effective child setup before
effects, through existing composition/start/environment seams. Preserve current fresh/frozen rules,
supported unbound environment semantics and the fact that secret environment values are recomposed.
Do not silently move conversations, reset branches, delete foreign contents or copy authentication.

Keep the three different failure oracles: foreign checkout at the expected path; conflicting A/B
project layers on explicit refresh; and redirected Claude environment at the actual child. A
recorded execution contract or passing server preflight alone cannot certify the effective process.

## 5. Keep the remaining remedies narrow and preserve uncertainty

- CR-A6-1's false-success attachment Steer and A6-6's refusal mapping touch ASF-2's acceptance/
  uncertainty seam. Coordinate their error/receipt distinctions: persisted acceptance, definitely
  refused delivery and unknown native consumption cannot collapse into `sent` or automatic replay.
- CR-A6-2's cross-run phone dialog can target the wrong run; treat it as control correctness, with
  the data-loss/wrong-environment/premature-Stop fixes, rather than deferring it as phone polish.
  A6-3's partial archive warning and A6-4's dead host approval remain local truthful-output fixes.
- CR-A7-2's admitted/cancellable reference scan is distinct from A7-4's repeated idle full replay.
  Carry request cancellation and explicit per-owner scan bounds through existing reader seams;
  avoid replaying unchanged/ineligible retained histories without weakening fallback repair or I/O
  errors. No new shared cache/index or global scheduler is needed just to close these findings.
  A7-5's generation tombstones should follow existing generation/journal teardown.
- CR-A4-1 is a local enum/test correction handled by its existing fix worker; the live handoff
  now records it as fixed. Preserve that worker's validation/closure and actual wake/activation
  assertions. It does not repair task admission or demonstrate mail loss.
  CR-A5-2 is specification alignment with already-settled native-load refusal; fix no runtime
  fallback and do not reopen ASW-2's decided behavior.

## Execution and residual limits

These are coordination recommendations, not new selectable work/fix units or authorization to
transfer findings across queues. Current CR units and ASF-2–ASF-6 retain ownership. Prioritize
irreversible data loss, wrong-target/setup effects, interrupted turns and escaped capacity; other
independent local fixes can proceed. Coordinate shared files rather than impose a global repair
order or reopen completed ASW refactors.

After relevant fixes, reuse the audit plan's bounded development journey: delegate evidence-bearing
work, wait/yield, continue the same assignment, report, settle/release and observe the next work;
include one interruption/lost-update boundary. Verify actual authority consumers and native effect
counts, not only helper returns. This is targeted cross-fix closure, not another full architecture
review automatically forced by fixes. Native/device/adoption and matched performance gates remain
unproved unless their own receipts are supplied; nothing here establishes worse model reasoning.
