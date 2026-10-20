# Proposed testing and architectural-review workflow

Designed 2026-10-20 in response to the human's request for concrete implementation locations,
a `write-tests` skill, and an `architectural-review` skill before every release. This builds on
the [defect-origin research](AGENT-SURFACE-META-ANALYSIS.md). Three focused GPT-6 Luna readers
researched the current workflow/testing rules and Chuck's operator/coordination architecture;
the parent checked the specifications, launcher checker and release sequence.

**Implementation status:** the purposeful `write-tests` companion, TS-06.R35 and workflow §17 are
implemented and verified first. The architectural-review/gate sections below remain proposals
until the second sequential skill is implemented. The [core audit plan](CORE-RELIABILITY-AUDIT-PLAN.md)
remains deferred. This is a medium
documentation/tooling change: two skills in both launcher trees, targeted existing-launcher edits,
shared rule changes, and a small release-receipt check with negative tests. It needs no product
architecture refactor.

## Recommended shape and exact homes

| Home | Proposed responsibility |
|---|---|
| `docs/specs/tech/TS-06-build-test.md` | Own the testing policy and evidence levels, extending existing R5–R9; append any new binding requirements without renumbering existing IDs. Own the mechanical architectural-review release requirement. |
| `docs/features/INVARIANTS.md` §17 | Keep the existing independent-oracle rule authoritative. Link/reuse it; do not copy a competing version into every skill. |
| `docs/features/AGENT-WORKFLOW.md` §§2/13 | Require an affected-path trace when a change alters durable information, execution ownership, admission, permissions or delivery. Call the test-authoring companion when relevant. |
| Workflow §§7/8/10 | Review test evidence and unchanged affected consumers; keep confirmed probes; prevent notes from weakening requirements; independently verify consequential fixes within their originating unit. |
| New `write-tests` launcher in `.agents/skills/` and `.claude/skills/` | Turn the shared test policy into a short authoring procedure used during work/fix. It owns no separate feature or test queue. |
| New `architectural-review` launcher in both trees; new workflow section | Inspect the assembled release's cross-system correctness and unchanged affected paths, independently of the author. Allow focused explicit use between releases. |
| Proposed `docs/archive/reviews/architecture/<candidate>.md` receipts | The architecture reviewer owns each versioned capability/authority map, coverage and result. Start from the prior successful receipt's map (this audit plan seeds the first run), update it against current FS/TS contracts, and retain the updated map in the new receipt. FS/TS remain product authority; no separate mutable map or duplicated specification is added. |
| Workflow §16 plus both `release` launchers | Require a current architectural-review result before every release tag/publication; run it after release preparation can change shipped content. |
| `AGENTS.md`, `CLAUDE.md` where applicable | Register the two roles/companions and their read scope; keep routing short. |
| `scripts/check-launcher-contract.sh` and `scripts/check-launcher-contract-test.sh` | Check both launcher trees retain the required routing/gate statements and remain aligned; extend the existing checker rather than create a parallel launcher system. |
| Proposed `scripts/check-architectural-review.sh` plus its tests; release CI | Check that a release-review receipt exists, is successful, belongs to the right release range, and has no unreviewed substantive changes after it. This checks provenance, not the quality of reasoning. |

The product's embedded `internal/agentknowledge/operating-chuck/` is for agents **using** Chuck.
Do not put repository development/testing/release instructions there. The repository skills are
the `.agents`/`.claude` twins; the shared workflow remains their process authority.

## Existing policy versus the actual gap

TS-06 already requires both SQLite test variants, affected UI checks, focused race checks, temporary
homes, deterministic fake providers and explicit live-provider gates. INV §17 already requires an
independent oracle, a meaningful falsifier, realistic failure shapes, and pre-fix regression proof
when practical. Reintroducing those as new ideas would add text without changing the failure mode.

The gap is consistent application: choosing the right level of test, reaching the real failing
boundary, checking what remains after the operation, and having the reviewer challenge the evidence.
Put the short level-selection policy in TS-06; have `write-tests` make the author apply it and
ordinary review check the resulting evidence. No coverage percentage or test-per-function quota.

## Testing rules to put into practice

| What is being claimed | Smallest useful evidence | What that evidence cannot establish alone |
|---|---|---|
| A local decision, parser, bound or transformation is correct | A deterministic unit test with expected values derived from the requirement or external format; include the relevant boundary/negative case. | Correct wiring into launch, persistence, another process or the real provider. |
| State survives a transition or interruption | A package/integration test through the actual store/transcript/transition owner, checking durable before/after state and recoverability. | Native provider history adoption if only a fake peer runs. |
| A protocol field or command reaches a consumer | Exercise actual serialization/argv/process I/O and a consumer-faithful decoder or pinned producer fixture; inspect the received value. | A permissive fake accepting a Chuck-authored field does not prove the provider uses it. |
| A race, retry or partial failure is safe | Control the relevant ordering/failure point, then check durable outcome, ownership and side-effect/submission counts. Use channels/barriers or fake time where appropriate. | Repeating a happy-path test or running `-race` does not supply the missing schedule. |
| Native OS/driver behavior works | An isolated owned real process/PTY/driver test for that platform contract. | This does not prove live-provider context, authentication or task adoption. |
| A provider adopts the intended context/settings/actions | Bounded real-provider acceptance with the required credentials and pinned version. | Source inspection, a permissive protocol double or a working PTY does not close that gate. |
| A script/operator journey or packaged release works | Real CLI/server wire and exit/output/side-effect evidence, or inspection/execution of the actual built artifact. | A copied command table, clean source build or UI stub does not prove the distributed artifact. |

Additional authoring rules:

- Test the externally meaningful behavior, not internal call order, field names or a copy of the
  implementation's decision table. Use existing canonical builders/fixtures where appropriate,
  but derive expected results independently of the helper under test.
- Prefer a few cases that discriminate the guarantee. Add property/fuzz tests when they reveal
  meaningful parser/bound/state-space errors; do not require them everywhere.
- Make a bug regression fail on the old behavior when practical. For a native/race probe, retain
  the minimized useful artifact and promote it into a permanent regression during fix. Do not
  preserve all scratch logs or require a full mutation-testing framework.
- Use controllable synchronization instead of sleep as the proof. Real timeout/OS contracts still
  need bounded real-time tests; do not ban clocks, subprocesses or all timing assertions.
- Keep user state and credentials isolated. Label evidence as automated, rendered, native/live or
  still owed. Record an actual failed acceptance check as a finding under existing workflow §4.
- Keep the existing focused iteration and one applicable full closure matrix. A reversible copy-
  only or presentation-preserving edit does not gain a new testing project from this skill.

## `write-tests`: concrete skill contract

**Trigger:** explicit `/write-tests`, and a companion to `/work` or `/fix` when a behavioral change
needs tests or tests/fixtures are materially changed. Do not load it for routine documentation,
copy or cosmetic work. The existing launchers must explicitly route to it; do not depend on the
model happening to discover it. Reviewers read the shared policy without invoking a writing skill.

**Inputs:** the selected change/finding, governing requirement, affected paths, and current test
helpers. **Authorized work:** tests and necessary fixtures within that selected scope; product
repairs remain under the invoking work/fix role. It never selects another queued feature or
rewrites a requirement to accommodate a passing implementation.

The skill asks the author to name four things before writing a nontrivial test: the promise,
where the expected answer comes from, the production boundary being exercised, and the smallest
case that would expose the wrong behavior. Then write the test, demonstrate its falsifier when
practical, run focused checks, and state the evidence limit. This is one continuous work step,
not another approval round or a mandatory separately delegated test writer.

Example: for conversation-ID retention, use real hook/state/teardown/reindex operations and compare
the retained ID afterward. A unit test showing that a map can store an ID would not establish it.
For ACP configuration, inspect the consumer-decoded request rather than only the producer struct.

The same worker may author tests. Independence concerns the expected authority and observation,
not merely which agent typed the test. A separate agent given the same false model can reproduce
the same bad oracle. Ordinary review checks test quality; consequential fixes receive the fresh
semantic check below.

## Changes inside existing skills

**Work/design review:** for a changed important field/ownership boundary, record a compact trace:
where it originates, where it becomes durable or applied, who consumes it, and who handles failure.
Search the actual consumers, including old maintenance paths. This is not a new whole-system
document for each change. Review-design challenges whether the proposed ordering/recovery is
implementable, not just whether the desired outcome sounds right.

**Review:** challenge one or more load-bearing test claims at the real consumer boundary, selected
by consequence. Inspect new implementation/deviation notes against the surrounding requirements.
Allow small versioned reproduction artifacts in addition to the handoff. No product fix in review.

**Fix:** for repairs involving durable work, execution ownership, authorization or provider delivery,
have a fresh reviewer verify the repaired guarantee and a nearby affected path before closing that
finding. Keep this inside the same unit; do not create an automatic new review unit for every fix.
The reviewer records the evidence and remaining gaps; the fixer still owns repair/closure. Low-risk
fixes keep their existing closure. A new unrelated defect is recorded separately rather than
silently expanding the selected fix.

**Spec accuracy:** an implementation note cannot make a conflicting requirement optional. Identify
the conflict and resolve a substantive product/correctness change under existing §3; ordinary
implementation choices need no extra permission.

## `architectural-review`: a release gate with a different question

**Trigger:** before **every** release, and when explicitly requested between releases. It is
mandatory even for a fixes-only release; repairs can change ownership and recovery just as features
can. Small releases can have a short review, but no release gets an automatic exemption.

Ordinary review asks whether a selected change meets its requirements. Review-design challenges a
proposed design before implementation. This skill asks whether the **combined current system**
still fulfills its core promises after all the changes and fixes are assembled. Its scope follows
cross-system assumptions, including unchanged consumers, rather than stopping at the release diff.

**Inputs:** previous release, candidate product revision and pinned runtime versions, release range,
completed reviews/fixes, known findings/owed gates, and the capability/authority map from the prior
successful architecture receipt (seeded from the core audit plan on first use).
Use a fresh reviewer rather than have the release preparer certify their own work. Discovery can
be delegated; the reviewer integrates the cross-system judgment.

Required work:

1. Reconcile the release changes and existing evidence against the core promises: saved work,
   executable workspace/configuration, context/knowledge, admission/recovery, authoritative results,
   truthful operator control, and resource cost. Show what changed and what proof is reusable.
2. Follow changed assumptions into unchanged callers, repairers, schedulers and release artifacts.
   Examine commit/acknowledgement, replacement and partial-failure boundaries. Explicitly search
   for an implementation note or helper-level claim being mistaken for a stronger guarantee.
3. Challenge the highest-consequence unproved joins with bounded independent probes. Reuse valid
   prior evidence. Include latent risks outside the diff when current architecture/evidence gives
   a concrete reason; do not merely summarize past reports or reread every file indiscriminately.
4. Record coverage, actual source/runtime revisions, probes/results, limitations and findings.
   A pass means no blocking finding in the assessed release, not proof that the whole product is
   bug-free. Preserve all owed native/provider/device gates; an actual failure remains a finding.

**Baseline and subsequent runs:** the first use reconstructs a capability/authority map and reviews
the current core risk concentrations using the existing ASR evidence. Later runs update that map,
challenge the assembled changes and remaining high-value gaps, and reuse unchanged proven paths.
A full source audit of every subsystem on every release would be too expensive and still would
not prove everything. Mandatory execution and explicit coverage prevent it becoming a no-op receipt.

**Findings and finite closure:** use existing severity/fix-model rules. Deduplicate an existing
finding and preserve its owner. New cross-system findings receive one architecture-review unit
with clear responsibility so `/fix` can pick them up. Every Must-fix blocks release under §16.
Fixes receive focused re-verification of their findings and affected joins; do not restart the
entire audit solely because a finding-fix commit landed. Unrelated substantive changes do require
the affected review coverage to be updated. Review writes no product fixes.

## Enforcing the release gate

Put the gate in the shared release workflow, not only in the new skill's description:

1. Begin release readiness as today; retain the existing Must-fix and dirty-tree checks.
2. Refresh the shipped operating knowledge/documentation and build/generated assets required by
   the release. Commit verified preparation so the reviewer has a stable candidate.
3. Run a fresh architectural review of that candidate before tagging. Store its compact receipt
   under a proposed `docs/archive/reviews/architecture/` location; keep open findings/live gates in
   the handoff. The receipt identifies the reviewed commit, prior-release boundary, component
   revisions, covered joins, evidence limits, disposition and any fix re-verification.
4. A proposed check script rejects a missing, blocked, wrong-range or stale receipt. A substantive
   code, spec, runtime, packaging or shipped-knowledge change after the reviewed commit requires
   updated affected coverage. Receipt/handoff/release-note-only commits do not invalidate it.
   Use ancestry plus an explicit small administrative-path allowance, not an exact-HEAD equality
   rule that invalidates the review merely by committing its receipt.
5. Both release launchers call that check before tagging/publishing, and release CI calls it before
   publishing artifacts. CI must fetch enough history. CI cannot assess the review's reasoning;
   mechanical receipt validation complements the independent review and existing artifact checks.

Publishing authorization, version choice and existing manual-gate rules remain unchanged. A failed
receipt check cannot silently downgrade to an owed review. A missing live gate is reported with
its actual existing blocking policy; this proposal does not relabel all historic owed gates as
new defects or claim native acceptance from source-only evidence.

## Implementation and verification sequence

1. Update TS-06/shared workflow and add the two short skill twins plus the routing entries.
2. Update work/fix/review/review-design/release twins and the existing launcher checker/mutation
   tests for the new required routing and release gate. Keep expectations tied to the shared rule.
3. Add the small receipt validator and meaningful fixtures: missing/blocked/wrong range, a valid
   review, a later substantive change, and a later administrative-only commit. Do not test only
   that a sentence appears.
4. Add the release-CI gate. Exercise both skill designs on isolated real cases: a test whose
   helper-only oracle would pass despite broken delivery, and a multi-change candidate with a
   stale consumer outside its diff. Give the evaluator raw artifacts without the intended answer.
5. Run documentation/spec/launcher checks and targeted checker/forward tests; perform applicable
   closure for any implemented tooling. Keep product changes and the deferred audits separate.

No further policy decision is hidden in this design: it recommends automatic companion use of
`write-tests` for relevant work/fix, a fresh reviewer for consequential fixes, and an architectural
review on every release. The human can approve or revise those concrete boundaries before they
become enforced workflow.
