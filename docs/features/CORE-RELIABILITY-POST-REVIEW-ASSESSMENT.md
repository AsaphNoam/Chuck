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

## Conclusion

Local fixes can start. Two boundaries deserve coordinated implementation rather than disconnected
patches: interruption-safe reindexing and task runtime ownership through result/yield/continuation.
The observable-state fixes also need one connected recovery proof. The findings do not establish
a need for a new global scheduler, event database, generic action framework or prompt rewrite.

Most governing distinctions are already specified. The recurring implementation error is using an
easy proxy for authority: a rebuilt session row for retained native identity, terminal task state
for released runtime ownership, a checked project/profile for the actual child setup, or a healthy
stream for current state. Repair those consumer boundaries; another broad workflow rule alone would
not resolve them. The current purposeful-test and architectural-review rules already ask for the
independent boundary evidence that was missing.

## 1. Reindex must be one preservation boundary

**Findings:** CR-A1-1/2; CR-A1-3 is the shared complete-record prerequisite.
**Size:** medium-to-large, difficult; transaction/executor and replay/error semantics span the indexer.

`internal/index/reindex.go` commits `resetTables` before replay and keeps the latest native ID,
effort and fast setting only in memory. These three inputs are not reproducible from old
`session_meta`. A repair process can die after deletion, while a missing root/log can commit an
empty replacement and report success. This is more serious than stale archive search: the repair
can remove the authority needed to continue development work.

Handle A1-1/2 together around a boundary that preserves prior usable state or publishes a complete
replacement. One transaction in the existing database is a plausible minimal design **only if every
reset, replay, flush and retained-field restoration uses it**. The current `Indexer` owns `*sql.DB`
and starts independent transactions
(`internal/index/indexer.go`); wrapping only `resetTables` cannot make replay atomic. A transaction-
bound executor would need intentional whole-rebuild failure or per-agent savepoint semantics and
consideration of long write-lock duration. Durable staging is an alternative if those constraints
make a long transaction unsuitable; it adds recovery state and is not automatically preferable.

Atomicity alone does not fix missing input. Refuse a missing canonical root before mutation. For a
missing/unreadable agent source, preserve its existing usable snapshot or abort the replacement and
report the identity; do not commit its disappearance as an empty rebuild. Protect the three observed
SQLite-only fields rather than declaring every replayable session column new authoritative state.

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

A2-3's lost assignee and A2-5's mail/manual wake bypass show the same assignment/runtime distinction
on continuation. Preserve confirmed membership when abandoning a later reservation, and route input
for a waiting assignment through its existing continuation admission. Do not replace these with
untracked Resume, create another assignee, or copy mail's attempted-at-most-once policy to tasks.
A2-9 needs bounded rediscovery after transient reap contention rather than another process restart.

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

**Findings:** CR-A3-1/2/3, coordinated with ASF-5's existing readiness fixes.
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
- CR-A4-1 is a local enum/test correction, already dirty at synthesis entry; preserve actual
  wake/activation assertions. It does not repair task admission or demonstrate mail loss.
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
