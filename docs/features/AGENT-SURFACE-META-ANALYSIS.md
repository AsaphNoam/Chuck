# Chuck defect origins, review escapes and audit priorities

Research completed 2026-10-20 against `0bebb062` (product source through `9a15afa6`). Six
`gpt-6-luna` agents investigated historical causes, recent refactor causes, workflow history,
test oracles, other backend/data surfaces, and client/security surfaces. The parent checked
representative historical diffs and current source, corrected causal overstatements, and ran a
new isolated reindex probe. Four pre-existing dirty ready-change files were preserved.

This is a discovery report and a workflow-change proposal. It changes no product code, specification,
skill, review policy or release gate. The confirmed new defect is META-1 in
[HANDOFF.md](HANDOFF.md#review-findings). Other prospective audit areas below are hypotheses or
evidence gaps, not newly asserted bugs.

**Subsequent state:** META-1 was fixed in `0cafe370`; its finding/probe descriptions below retain
the discovery snapshot, not a claim that the reproduction still fails. The live handoff owns current
fix status. The human's later reliability priorities are mapped in
[CORE-RELIABILITY-AUDIT-PLAN.md](CORE-RELIABILITY-AUDIT-PLAN.md); that proposed list supersedes the
relative priorities below, without starting another audit.

## Findings

**Both design and implementation contributed.** The older defects include unsafe policies and
incomplete external-contract assumptions. Several recent refactor defects instead depart from
guarantees already explicit in their design. A third recurring cause is integration drift: a field,
owner or requirement acquires new meaning without enrolling its unchanged consumers and repair paths.

The practical failure is that a correct local statement is repeatedly used as evidence for a
larger guarantee it does not establish: a mutex becomes "ordered history"; a live PID becomes
"proof of process ownership"; a persisted field becomes "survives repair"; a fake accepting a
request becomes "the provider consumes it". Each claim needs the actual end-to-end boundary checked.

**Many review cycles do not imply many independent tests of the failing condition.** Some original
defects survived weeks or months. Most ASW findings were found in the first substantive review of
new code on Oct 19–20. Calling all of them long-hidden failures would misrepresent the history.
The ASR subsystem audits and ASW reviews did detect defects; the useful question is what their
successful probes tested that ordinary closure and earlier narrower reviews did not.

No finding count here estimates repository-wide defect density, model quality or reviewer ability.
The samples are deliberately biased toward complicated, suspicious boundaries. Reports and commits
show recorded scope and evidence, not a complete record of a reviewer's reasoning or attention.

## Cause classification and chronology

The categories overlap: **design** means an unsafe/incomplete contract or unproved mechanism;
**implementation** means failure to realize an already specified guarantee; **integration drift**
means old code retains assumptions invalidated by a later change. A missing test explains escape,
but does not by itself identify the original cause.

| Case | Evidence and timing | Cause and escape |
|---|---|---|
| Original orphan signaling, ASRC-3 | Before `8831d893`, FS-01.R21 explicitly said the server checks the live PID, SIGKILLs it and deletes the row. Current proof-based policy was introduced in the ASW design. | An unsafe policy was codified in the design. Reviewing faithful implementation against that rule was insufficient. A separate invariant/OS-identity question was needed. The later leaderless remainder is both an implementation failure of fail-closed retention and an incompletely designed continuity mechanism. |
| Native Codex guidance, PROMPT-1 | `6152c367` on Oct 19 extends FS-09.R32/TS-04.R14 from preserving a caller overlay value to preserving native configuration too; `8e47d056` later adds ancestor layers. | The original integration modeled Chuck's prompt delivery more completely than the provider's instruction sources. Verifying emitted configuration against Chuck's own fake could agree with an incomplete external model. The preservation scope evolved; do not project every later detailed rule backward onto earlier reviewers. |
| Live config deadlock, ASRC-1 | `c640b486` on Sep 8 introduced the lock spanning ACP RPC. Same-day review-fix `8a01ebf5` touched runtime, fake peer and server tests. The Oct 19 probe sends a frame before the reply. | A locking/reader-progress implementation defect. Earlier notification handlers already needed the lock for streamed output; Sep 22's child scoping broadened the dependency. Testing returned values and lifecycle serialization did not test progress with an interleaved frame. |
| Lost post-result recovery, ASRF-1 | Pipeline startup/pause code traces to `5f371f14` on Jul 26. Retryable cleanup guarantees were strengthened by the Sep 12 design (`05512823`) and shipped Sep 13; `1a94a3e5` changed recovery without eliminating this old action-clearing path. | Integration drift: an old error policy outlived the newer durable-recovery contract. The discriminating fault is after a result/release succeeds but before next-stage/final/approval bookkeeping succeeds. Generic restart and Stop tests do not imply this boundary was tested. |
| Uncertain promoted Steer, ASRH-4 | Promoted-held handling dates to `ae3740eb` on Sep 10; attachment changes followed in `395c1a8a` on Oct 13. `6ea0a646` fixed duplicate UI submissions, a different cause. | The host treated missing acknowledgement as evidence of no native consumption. A UI click-deduplication review could not prove that external acknowledgement contract. The useful probe consumes input, loses the reply, then counts later native submissions. |
| Frozen clone permissions, ASW2-6 | Before implementation, TS-01.R40/TS-02.R53 and `8831d893` required source-contract inheritance. Clone instead reconstructed permissions from today's persona/global defaults. | An explicit implementation violation. Passing model/setup inheritance tests was not proof that the whole contract survived. This is not evidence that the product needed another permission policy design. |
| Legacy models, queue bounds, manifests | The prior design forbade catalogue guesses, set a 256+1 queue bound, and required every process's bounded manifest. Implementation notes later described catalogue reconstruction, unreserved canceled enqueue, and chat-only/incomplete manifests (`43da6bec`, `9a15afa6`). | Recorded local choices weakened explicit guarantees. Notes and green tests did not make the contradictory requirements agree. Some closure tests followed the narrowed implementation rather than the original requirement. These were new-code review findings, not months-old escapes. |
| Early SessionStart, ASW3-3 | Original design said to persist the ID at authenticated SessionStart. It did **not** specify an admission barrier before provider execution. Tests pre-seeded the row that authenticates the hook. | A design ordering-mechanism gap plus an integration-test gap. It would be inaccurate to call this violation of an explicit pre-provider handshake requirement: that mechanism was not specified. |
| Delayed cleanup, ASW1-9 | Design required conversation ordering and captured ownership, but actual cleanup escaped into a goroutine and looked up current rather than original execution. Current callback-injection probes reproduce the consequence. | Missing end-to-end ownership construction. A per-conversation FIFO alone cannot order later detached effects. Simply waiting synchronously can self-deadlock when Stop emits a new terminal event. |
| Birth precision, ASW3-6 | ASW design required exact platform-precision identity. Shared code added a one-second tolerance; `TestClassifyProcessRequiresExactBirth` checked only an hour-old mismatch. | An explicit implementation/test gap. An invariant name and a far-away negative test did not prove the actual boundary. The ±250 ms probe discriminates it. |
| Reindex native identity, META-1 | `e2cf34f8` on Oct 19 makes hook-learned identity durable in SQLite. Older `Reindex` still deletes/reconstructs entire session rows from transcripts. The Oct 20 probe loses the learned ID despite successful offline repair. | An authority change was not propagated to an unchanged repairer. This is a newly demonstrated integration failure, not proof that this exact scenario survived all earlier reviews. |

The original ready designs were themselves inspected by focused readers, but the retained history
does not establish a completed independent `/review-design` pass for these three changes. That is
an evidence limit, not proof of a skipped role. More design review would only help if it asks for
an implementable ownership/failure trace; another prose-consistency pass is insufficient.

## Why previous reviews and tests could stay green

1. **Scope was narrower than the system claim.** Ordinary `/review` examines a named change unit
   and sweeps its diff against triggered invariants. The ASR charters explicitly trace complete
   workflows and unchanged callers. These are different review shapes. Reviewing many feature
   diffs does not collectively prove that an old repairer still understands a new authority field.
   ASW-3, ASW-1 and ASW-2 were also reviewed at different isolated snapshots; that correctly protects
   each review's evidence, but does not constitute a final review of their combined behavior.
2. **The oracle shared the implementation's assumption.** Commit `a0dc9127` describes a prior
   concrete example: design emitted a top-level model field, runtime sent it, fake ACP accepted it,
   and the provider read none of it. Every Chuck-authored layer agreed. INV §17 and TS-04.R54
   now require independent producer/schema evidence. The same failure class recurs when checks
   inspect only one instruction key or an earlier manifest input rather than effective delivery.
3. **Tests began after the dangerous transition.** A pre-seeded running row proves hook updates,
   not launch ordering. A live process leader proves one containment case, not a leaderless group.
   One `agentState` proves local queue ordering, not ordering across replacement. A short model ID
   cannot demonstrate the manifest's serialized byte bound. These are test-seam and input choices.
4. **Logical failures are not data races.** `-race`, both SQLite variants and repeated green runs
   do not create notification-before-reply, lost acknowledgement after consumption, or write failure
   after result acceptance unless the fixture supplies that schedule. A correctly synchronized
   program can lose the wrong owner or deadlock. More repetitions of the same oracle do not help.
5. **Repair and negative outcomes lacked closure checks.** “Preserved through Stop” omitted offline
   rebuild; “failed submission” omitted the possibility of native consumption; local queue ordering
   omitted later detached cleanup. Assertions stopped short of checking the user's recoverable
   state after the last irreversible effect or the first failed acknowledgement. The proposed remote
   and updater audits ask analogous questions; those candidates still need reproductions.
6. **The process permits evidence gaps to persist.** Failed live checks must be recorded, while
   unrun checks remain owed. That distinction is correct, but implemented/reviewed is not native-
   verified. The handoff honestly retains real-provider/device/driver gates. Source-proven bugs
   such as queue ownership did not need those credentials and should not be explained away by them.
7. **High-risk fixes need a bounded independent check.** Workflow §8 lets a finding-fix commit close
   its originating unit without another default review. That avoids review ping-pong, but can leave
   substantial repair logic checked only by its fixer. For scale, `1a94a3e5` was a 35-file pipeline
   review-fix commit, not a tiny patch. This is a risk in the current closure policy; available
   records do not prove that a fresh check was absent in every historical instance.

This is not evidence that all prior reviews were ineffective. They found and corrected defects,
CI-1 was traced to a faulty fake cancel model (`d2e59d6f`), and `a9d5e387` made failed live gates
discoverable after an archived-only report was missed. The latest reviews exposed the sampled
construction mistakes. Their successful method is the counterexample to preserve.

## Proposed workflow and skill changes

These are proposals, not applied policy. Keep the shared workflow authoritative; launchers should
point at its requirements rather than duplicate a new long checklist in both skill trees.
Do not require a whole-repository audit, full Cartesian test matrix or extra full-suite run for
every ordinary change. The existing full closure policy is unchanged by this research.

In plain language:

| Proposal | What the agent should do differently | Example |
|---|---|---|
| **Follow the affected paths.** | When changing how important information is stored or used, check the other paths that depend on it, including old ones. | Saving a conversation ID is incomplete if reindex later erases it. |
| **Test the way the promise could fail.** | Choose a test that would expose the suspected loss or error, at the real boundary. | To test safe retry, let the recipient receive the message, lose the acknowledgement, then check that retry does not send it again. |
| **Keep the promised behavior.** | Do not finish a simpler implementation and use a note to excuse the missing guarantee. Resolve a genuine change to the promise explicitly. | “Preserve the original permissions” cannot become “use today's defaults.” |
| **Have another agent check consequential fixes.** | Before closing a fix that can lose work or misdirect an agent, have a fresh reviewer check that fix and one nearby affected path. | After repairing native-ID retention, check both ordinary Resume and the repair path. |

The first three are mostly stronger checks on promises already made. The fourth adds a focused
independent check to consequential fix closure. None requires asking the human about routine
implementation details, another whole-app review, or another full test run for every small fix.

### 1. Enroll owners and consumers when semantics change

**Where:** workflow §2, the ownership hygiene in §13, and a short link from `/work`/`review-design`.
**Trigger:** a changed authority/permission, durable field, async owner, acknowledgement or lifecycle
boundary. **Proposed text:**

> Before implementing a changed ownership or durability boundary, name its authority and identity,
> its commit/acknowledgement point, and who retains recovery after cancellation, replacement or
> partial failure. Enumerate the actual entry paths and consumers, including repair/rebuild,
> restore, migration and deletion where applicable. Check unchanged consumers too. Use one compact
> trace/table for the changed seam; ordinary local changes need no extra artifact.

This would put terminal SessionStart admission, leaderless continuity and reindex enrollment in
the design discussion rather than leaving them as desired outcomes. It asks for a feasible
mechanism, including safe refusal, rather than a new abstraction. Cost: a bounded source search
and roughly one short table on high-risk changes; not another full design document.

### 2. Make closure evidence discriminate the actual failure

**Where:** workflow §§2/7/8, linking existing INV §17; a brief `/review` launcher cue distinguishes
change review from explicitly requested subsystem audit. **Proposed text:**

> For a changed high-risk contract, identify the independent oracle and the real entry/consumer
> seam. Exercise the smallest counterexample at its dangerous boundary: failure before/after
> commit, lost acknowledgement, replacement, or the exact bound as appropriate. Explain why that
> check fails when the claimed guarantee is false. Do not substitute helper agreement or an
> unrelated passing suite. For that changed contract, reconcile required gates as passed, failed or owed.

Promote minimized confirmed probes into permanent regressions during fix. For a nontrivial race or
native-contract probe, keep a small versioned reproducer/patch with the finding instead of relying
only on an OS-temp path; exploratory probes/logs need not be retained. This needs a narrow update
to §7's allowed review artifacts. The META-1 source is retained with this report as an example.
Cost: a few discriminating cases selected by risk, not more repetitions of broad existing tests.

### 3. Stop implementation notes from authorizing weaker guarantees

**Where:** workflow §10 and review's code/spec comparison. **Proposed text:**

> An implementation or deviation note does not override a requirement. A proposed weaker guarantee
> must identify the conflicting requirement and its replacement acceptance evidence; resolve the
> substantive decision under §3 before claiming closure. Otherwise retain the discrepancy as an
> open finding. Review newly added notes against the surrounding normative text.

Use existing authorization for routine implementation choices. This does not add approval for every
mechanism decision; it addresses changed product/security/correctness guarantees. It would reject
“bounded” unreserved enqueue and “each process” narrowed to chat without making every style choice
an approval flow. Cost: focused review of a small semantic delta, no additional test suite.

### 4. Independently verify consequential fixes within their existing unit

**Where:** workflow §8 and both fix launchers' closure sentence. **Proposed text:**

> When a fix changes ownership/lifecycle, authorization, external-protocol delivery or durable
> recovery, keep the originating unit open until a fresh reviewer checks the fix at the affected
> seam, the original reproducer and one adjacent affected path. This is verification within the
> existing unit, not a new default review unit or a full repeat of unrelated acceptance. Ordinary
> low-risk fixes retain current closure. The reviewer records any missing evidence explicitly.

This is a deliberate proposed policy change, not what today's `/fix` already guarantees. It adds
an independent semantic check where the repair itself can reassign or destroy ownership. Scope
it by consequence rather than line count. Cost: one narrow fresh-context pass, without requiring
that reviewer to rerun every unchanged closure check or create an endless new review queue.

These changes preserve the independent role queues and lean read order. They make important claims
reviewable; adding more generic “be thorough” instructions would not address the observed causes.

## How widespread, and where to spend the next audit capacity

**The failure classes extend beyond the reviewed agent boundaries.** INV §17's retained examples
already document historical failures in tool-result classification, mail activation, configuration
I/O, release build tags and server/UI wire agreement. META-1 now supplies a current cross-domain
counterexample. This supports targeted expansion, not a claim that every Chuck subsystem is broken.

| Priority / audit slice | Concrete evidence, uncertainty and best next check | Approximate scope |
|---|---|---|
| **First: archive repair, reindex and durable-state preservation** | **META-1 confirmed:** hook-learned terminal native ID is lost by successful offline reindex. `internal/index/reindex.go:22–35,151–158` clears session rows; `state/manager.go:357–363` writes identity only to SQLite; terminal launch metadata precedes that ID. Also inspect failure after reset: existing bad-agent test checks the good agent, not preservation of the unreadable agent's prior state. Inventory authoritative versus replayable fields and all rebuild/restore paths. | Medium, state/index/CLI plus lifecycle consumers; one coherent audit. |
| **Next: updater/installer activation, rollback and crash recovery** | Source-backed risk: `internal/release/pointers.go:64–91` changes current before previous, while `swapPointer` can return an error after rename. Two individually atomic swaps are not one transaction. `TestRollbackRestoresPrevious` checks successful undoability, not interruption/second-write failure. TS-06.R18 promises failed rollback leaves old current intact. Inject the between-swap and post-rename faults; verify truthful outcome and recoverable pointer pair. No reproduced current defect claimed here. | Medium, few mutation boundaries plus packaging/CLI consumers. |
| **Then: remote revocation through every response/effect path** | Admission locking and SSE cancellation have tests. `remoteDevices.end` cancels contexts, while `serveAttachmentBytes` uses `io.CopyN` without observing them (`attachments.go:59–84`). Probe a slow in-flight transfer during revoke against TS-13.R9's one-second closure rule. Distinguish previously authorized bytes already in flight from a new unauthorized request; no present data-exposure incident was demonstrated. | Small-to-medium, targeted route/stream consumers and real loopback transport. |
| **Then: bounded retrieval under long histories** | `contextref.Service.renderSpan` (`service.go:428–448`) starts a canonical scan for each page; output bounds do not prove scan/I/O bounds. Existing fixed-source pagination tests use small fixtures. Measure bytes/work for late spans and repeated pages as history grows, then judge an explicit workload budget. Per-page memory and authorization are defended; unacceptable latency has not been measured. | Medium, one retrieval pipeline; performance evidence before architectural work. |
| **Finish the existing phone lifecycle receipts** | Connection generations, hydration and fake-clock tests are substantial. FS-20.A22–A24 still need actual iPhone/Android background-return receipts. Spend device capacity on those existing gates rather than another generic source pass claiming OS behavior. Concurrent defaults labels are a low-confidence check to fold into that journey, not a separate large audit. | Bounded real-device journey; depends on devices, not a new subsystem project. |

Lower immediate priority in this sample: consented worktree deletion and ownership/path rechecks,
pairing admission's revoke race, tracked-file/no-follow authorization, shared archive lifecycle
gates, and remote allowlist/denial inventory all have concrete defenses/tests. This is negative
evidence against indiscriminate re-auditing, not exhaustive assurance for those areas.

Do not start with UI styling, every CRUD endpoint, or a generic entire-app audit just because those
areas exist. Start with irreversible effects, authority changes, repair and recovery; select the
next slice based on the first focused probes. Existing ASW/ASF fixes remain useful and independently
selectable. Proposed workflow changes do not retroactively block their start.

## Evidence and limits

Representative checks: historical specs/code at `8831d893` and its parent; `c640b486`, `8a01ebf5`,
`dc6c1932`, `6152c367`, `5f371f14`, `05512823`, `1a94a3e5`, `ae3740eb`, `6ea0a646`,
`a0dc9127`, `a9d5e387`, and ASW implementation/review snapshots. Current files checked include
the role launchers, workflow §§1–8/10/13, TS-06.R5–R9/R18, TS-02.R9/R54, INV §17 and the relevant
source/test seams. No exact bug-escape rate or causal estimate of “AI versus human” is supportable.

META-1 probe: real temporary SQLite store and transcript, real authenticated `Manager.ApplyHook`,
real teardown retention, then actual `index.Reindex`. Before: `native-learned-after-start`; after:
empty. It intentionally fails in both normal and `sqlite_fts5` builds. Reproducer:
[`reindex-native-id_test.go.txt`](../archive/reviews/meta-2026-10-20/reindex-native-id_test.go.txt).
It uses existing index test helpers and can be mapped by Go `-overlay` to a test file in
`internal/index`; the original overlay is `/private/tmp/chuck-meta-20261020-overlay.json`.
Only test-owned state was touched. No credentialed provider, browser/device, real external terminal,
installed application pointers or user archive was mutated; no product fix or full-suite pass is
claimed. Documentation/reference and diff checks apply to this report's closure.
