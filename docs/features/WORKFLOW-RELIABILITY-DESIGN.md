# Workflow reliability — implemented skills and rules

Implemented sequentially on 2026-10-20 after human approval: `write-tests` first, then
`architectural-review`. Each used skill-creator, separate GPT-6 Luna research, primary engineering
sources and an independent isolated forward test. The
[research and validation record](../archive/reviews/workflow-skills-2026-10-20.md) retains sources
and evidence. This replaces the earlier proposal; its history remains in Git.

The [core audit plan](CORE-RELIABILITY-AUDIT-PLAN.md) is recorded and deferred for human expansion
and selection. Creating these skills did not run the real Chuck audits or a release review.

## Where the rules live

| Home | Responsibility |
|---|---|
| TS-06.R35 | Test value, cost and appropriate evidence level; INV §17 remains the independent-oracle authority. |
| Workflow §17 and twinned `write-tests` skills | Apply the test policy during relevant work/fix or explicit test work, without a new queue/author/approval cycle. |
| Work/fix/review launchers | Route authors to the companion when relevant; reviewers challenge test value and evidence rather than demand more tests. |
| TS-06.R36 and workflow §18 | Architectural review's core purpose, release requirement, bounded scope, evidence/receipt and no-loop closure. |
| Twinned `architectural-review` skills | Inspect combined-system assumptions, unchanged consumers and realistic high-value scenarios through a fresh reviewer. |
| Workflow §§8/16 plus fix/release launchers | Record verified architectural finding fixes and release from that same completed review. |
| `scripts/check-architectural-review.mjs` | Validate committed review/fix provenance, open findings and post-review coverage. |
| CI and release CI | Exercise real-Git gate fixtures; release CI fetches full history and enforces the gate before assembly/publication. |
| Existing launcher checker/mutations | Guard twin alignment, conditional test routing, mandatory release review and no-rerun-after-fixes routing. |

Repository launchers live in `.agents/skills/` and `.claude/skills/`. Product-managed
`internal/agentknowledge/operating-chuck/` remains guidance for agents using Chuck; development
workflow instructions were not added there.

## Purposeful tests

Each test must protect a meaningful behavior or plausible defect worth its development, runtime
and maintenance cost. Before a nontrivial addition, identify the promised behavior, independent
expected-answer authority, real boundary and smallest distinguishing case/schedule. Existing
adequate evidence can justify adding nothing. Choose the smallest adequate evidence, not a
helper-only test for a storage/process/delivery promise.

Unit tests cover local decisions and transformations. Integration tests observe actual stores,
wire/consumer boundaries and transitions. Real OS/driver and live-provider claims require their
respective evidence. Relevant races are controlled, not hoped for through repeated green runs.
No quotas, exhaustive matrices, duplicate happy-path layers or irrelevant structure assertions.
The existing focused iteration and one applicable closure matrix remain.

## Architectural review protects the development hub

Its purpose is Chuck's core value: recoverable work, faithful instructions/messages, progressing
assignments, correct execution/workspaces, truthful results/control and reasonable overhead.
Without those guarantees Chuck is not a reliable hub for real development. Cosmetic preferences
and implausible edge cases do not displace these consequences.

One completed review is required before every release, including fixes-only releases. It follows
important assumptions into unchanged consumers, maintenance, scheduling and installed artifacts.
Depth scales with consequence and uncertainty; reusable evidence reduces work. Source/fake checks
do not certify native/provider adoption, and performance claims require measured workloads.

The canonical JSON receipt is
`docs/archive/reviews/architecture/<previous-tag>-next.json` (`initial-next.json` without a prior
release). Workflow §18 defines its schema. It keeps the original reviewed commit and coverage,
references findings owned in the live handoff, and accepts exact verified update commits.

## Human's controlling no-loop rule

**Architectural review → findings → fix and verification → recorded closure → release.**

Fixing findings does **not** require another architectural review for that release. The fixer
commits the repair, then records its exact commits, original finding IDs and focused verification
in the same receipt. The gate accepts that remediation without moving the original review anchor
or rerunning the skill. Multiple repair commits can share a bounded evidence entry.

Unrelated new work needs explicit affected-coverage evidence rather than being disguised as a fix.
A later release has its own review. Receipt/handoff/release-note-only commits do not invalidate
themselves. Missing/incomplete reviews, unresolved Must-fix findings and unrecorded substantive
changes block release; CI validates declarations/provenance rather than judging review quality.

This delivery does not impose a new general reviewer on every ordinary fix. Earlier broader
workflow proposals about design traces or additional fix-review policy are not silently made
binding by these two skills. Existing publishing authorization and native/live gate policy remain.

## Verification

- Skill-creator validation and twin/routing/mutation checks.
- Test skill forward-check: one useful failing repair/Resume regression, no prose-only test.
- Architecture skill forward-check: a startup integration stall found despite a passing local
  dispatch test, with a bounded failing probe; only synthetic fixture state was used.
- Eight focused real-Git gate cases, including successful review → fix → release without rerun,
  unresolved findings, missing review, unrelated changes, tag CI and invalid provenance.
- Independent Luna gate/closure review; documentation/reference, syntax and diff checks.

No fabricated passing receipt was created for Chuck's next release. Its actual architectural
review remains a future explicitly selected/release-triggered task.
