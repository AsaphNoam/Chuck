# Workflow skill research and validation — 2026-10-20

## Purposeful test authoring

Two GPT-6 Luna researchers examined current Chuck test/workflow policy and primary engineering
sources before implementation. The parent read skill-creator, verified representative sources,
and integrated a short companion with TS-06.R35 and existing INV §17. The human's constraint is
explicit: tests consume development, runtime and maintenance capacity; each must earn its cost.

Research used:

- [Google: test sizes](https://testing.googleblog.com/2010/12/test-sizes.html): scope/isolation and
  dependencies affect speed and consistency. Google's historic taxonomy/time limits are not
  adopted as mandatory local rules.
- [Google: change-detector tests](https://testing.googleblog.com/2015/01/testing-on-toilet-change-detector-tests.html):
  behavior should drive assertions rather than snapshots of incidental implementation.
- [Google: know your test doubles](https://testing.googleblog.com/2013/07/testing-on-the-toilet-know-your-test-doubles.html)
  and [Fowler: mocks and stubs](https://martinfowler.com/articles/mocksArentStubs.html): state and
  interaction checks have different coupling tradeoffs. Chuck retains interaction assertions when
  the interaction itself matters, such as duplicate/forbidden native submissions.

The implementation asks for meaningful incremental confidence, an independent expected authority,
the real failing boundary and a small falsifier. It rejects coverage quotas, routine duplicate
layers and speculative matrices. It adds no mandatory test-author agent or new role queue.

An independent GPT-6 Luna evaluator received the new skill and a small isolated Go store fixture,
without the intended solution. Its request was to cover repair preserving a native Resume target,
and separately assess a prose-only help edit. It added **one** test through Save → Repair → Read
and ResumeTarget, using explicit contract values. The regression intentionally failed because the
fixture's Repair discarded identity; source was left unchanged. It added no test for the prose edit.
The parent inspected the artifact: `/private/tmp/chuck-write-tests-eval/session_repair_test.go`.
This checks useful level selection and restraint, not live-provider behavior or universal skill quality.

Skill-creator's validator, launcher routing/twin checks and their missing-routing/companion mutations,
`make check-specs`, link/reference checks and `git diff --check` pass. PyYAML for skill validation
was installed only under `/private/tmp/chuck-skill-validation`; no project dependency changed.

## Architectural review for the reliable development hub

Only after committing the testing skill did two GPT-6 Luna researchers investigate architecture
evaluation practice and the repository's release/fix mechanics. The human clarified the controlling
rule: architecture review finds items → fix closes them → release must not force another
architectural review for that release. That overrides the earlier proposal's ambiguous freshness rule.

Primary sources informing the review method:

- [SEI: ATAM](https://sei.cmu.edu/library/atam-method-for-architecture-evaluation/): evaluate the
  architectural choices and tradeoffs that affect important quality attributes.
- [SEI: Quality Attribute Workshop](https://www.sei.cmu.edu/asset_files/factsheet/2018_010_001_513488.pdf):
  use concrete scenarios with stimulus, environment and response; prioritize consequential cases.
- [AWS: review process](https://docs.aws.amazon.com/wellarchitected/latest/framework/the-review-process.html):
  keep reviews lightweight, go deeper on uncertainty, and prioritize actions by customer and daily
  operating impact. Chuck does not adopt cloud-specific infrastructure requirements or a formal
  ATAM workshop bureaucracy.

The skill starts with Chuck's core purpose: saved work, faithful delivery, progress, correct
execution, truthful results/control and reasonable overhead. It follows assumptions into unchanged
consumers and repair paths; speculative matrices and style changes do not count as core findings.
The release gate verifies declared provenance, not architectural reasoning. Fixes append exact
commits and focused evidence to the original receipt; no review rerun is required for remediation.

An independent GPT-6 Luna evaluator used the skill against a small committed synthetic hub with
a passing dispatch-helper test. It found a real startup liveness gap: persisted ready work was
never dispatched after restart. It wrote a bounded failing startup probe and scoped report without
editing production modules. Artifacts: `/private/tmp/chuck-architecture-eval/REVIEW-REPORT.md` and
`startup-probe.test.mjs`. This is fixture evidence, not a newly claimed Chuck defect.

Eight focused real-Git gate cases pass. They prove a completed review plus exact verified finding
fixes succeeds without another architectural review, while missing/incomplete review, unresolved
findings, live Must-fix entries, unrelated unrecorded changes and invalid provenance fail. Tag CI
excludes the tag being published from the prior-release boundary. Independent Luna implementation
review confirmed the no-loop path and found no material issue. Skill/twin/mutation, syntax,
documentation/reference and diff checks pass. No real release receipt or core audit was produced.
