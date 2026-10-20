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
