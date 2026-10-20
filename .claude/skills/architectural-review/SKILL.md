---
name: architectural-review
description: Review Chuck's combined architecture as a reliable development hub before every release or on explicit /architectural-review. Follow cross-system ownership, recovery, delivery and cost; finding fixes do not trigger another architectural review for that release.
---

# Protect Chuck's core value

Chuck must be dependable for real development. Saved work, faithful context/messages, correct
execution, progressing assignments, truthful results/control and reasonable overhead are its
core value proposition. If these fail silently, Chuck fails its purpose. Let that consequence
guide depth and priority; do not turn this into a style review or catalogue of speculative cases.

At startup, inspect `git status` and the existing diff before any repository edit. Read the handoff
header, live findings/gates, applicable FS/TS and triggered invariants; use
[workflow §§3–7/18](../../../docs/features/AGENT-WORKFLOW.md) for finding/commit/close rules and
[TS-06.R36](../../../docs/specs/tech/TS-06-build-test.md) for the release gate.
The [core audit map](../../../docs/features/CORE-RELIABILITY-AUDIT-PLAN.md) seeds initial coverage;
later runs reuse the prior receipt's capability/authority map and valid evidence.

Review the stable committed next-release candidate, or the explicitly requested scope, using a
fresh reviewer. Release preparation must already be included. Read the prior-release range and
completed fixes, then trace important assumptions into unchanged consumers, repairers, schedulers
and installed runtime artifacts. A correct diff can still break an old consumer or preserve an
unsafe design; a passing helper test can still leave the real boundary unproved.
An explicitly focused audit reports its limits; do not mark the release review complete solely
from that narrower work.

Choose credible scenarios by core impact and uncertainty. State the stimulus/normal condition,
promised response, owner/commit boundary and observed evidence. Reuse established proof; probe
high-value gaps independently at the real store/process/wire/provider boundary. Include progress
and safe recovery as well as preventing duplicate effects. Measure performance/cost claims.
Keep source/fake/native/live limits explicit. Neither every file nor every conceivable race needs
another check; each investigation must earn its cost.

Record confirmed findings as **Must fix** or **Worth fixing** in the live handoff, with one §7 unit-level Fix model;
deduplicate existing owners. Keep uncertain risks and owed external evidence clearly distinguished.
Write the §18 candidate receipt with the capability coverage, revision/runtime facts and finding IDs.
Include existing release-blocking finding IDs as references with their original owners.
Change no product code or normative specs. Commit the receipt, useful bounded evidence and handoff;
close with the core consequences, findings and evidence limits. No-finding reports are not new
default change-review units.

Fixing architectural findings does not require another architectural review for that release.
The fix workflow verifies and closes them, appending exact repair commits/evidence to the same
receipt. The release gate accepts that closure without rerunning this skill. Do not restart the
review merely because a fix commit exists. Unrelated new work needs focused affected-coverage
evidence; it cannot be disguised as a finding fix. The next release gets its own review.
