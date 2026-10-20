---
name: write-tests
description: Author or materially change Chuck tests and fixtures; automatically accompanies work/fix when behavior needs new verification, and supports explicit /write-tests. Use purposeful, proportional evidence; not coverage-only expansion or routine copy, styling and documentation edits.
---

# Write tests that earn their cost

Protect a meaningful behavior or plausible defect with the cheapest reliable evidence that reaches
the relevant boundary. Test count and coverage percentage are not objectives.

Within work/fix, reuse the selected scope, already-loaded state and parent commit/closure process.
Standalone, inspect `git status` and the existing diff before editing, read the handoff header and
the requested behavior/finding; do not choose another queued unit. Read
[TS-06.R35](../../../docs/specs/tech/TS-06-build-test.md),
[INV §17](../../../docs/features/INVARIANTS.md#17-tests-must-prove-their-contract-independently),
and [workflow §17](../../../docs/features/AGENT-WORKFLOW.md#17-write-purposeful-tests).
Open only the governing requirements and existing helpers needed for this test.

Before adding a nontrivial test, briefly identify the promised behavior, plausible defect,
independent expected-answer authority, production boundary and smallest distinguishing case or
schedule. A helper-only assertion cannot prove a storage, delivery or lifecycle promise.

Check existing tests first. Reuse or extend an adequate case; add nothing if the guarantee is
already established and the change introduces no distinct risk. Add a boundary/negative/property
case only when it protects a real contract or credible failure, not every conceivable input.
Avoid test-per-function rules, exhaustive combinations, duplicate happy paths at several layers,
private-structure snapshots and irrelevant call-order assertions. Counts/order are appropriate
when the contract itself forbids duplicate submissions or side effects.

Choose the smallest adequate level using TS-06.R35. Prefer observable outcomes and retained state.
For wire behavior inspect the consumer-decoded value; doubles must reproduce relevant real
constraints and failures. Control a race's actual ordering with channels/barriers or suitable fake
time. Use real processes, storage or bounded timing when the guarantee depends on them. Isolate
user data; no credentials or external provider cost is implicitly authorized by this skill.

For regressions, demonstrate failure on the old behavior when practical, then retain the focused
case. Do not weaken useful tests, rewrite the requirement, add broad product abstractions or fix
unrelated bugs to obtain green. Product repairs stay under the parent work/fix scope.

Run focused checks during authoring. The parent runs its applicable closure once; standalone test
changes follow workflow §§2–6. Report what the test proves and any native/live gate still owed.
Stop adding cases when remaining ones provide no distinct confidence worth their cost.
