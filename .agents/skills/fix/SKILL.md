---
name: fix
description: Explicit invocation only. Run only when the user sends `/fix`; do not trigger from a natural-language request.
---

# Fix findings

At startup, inspect `git status` and the existing diff before any repository edit. Use the injected
handoff header when present; otherwise read only **Current position** and **Active
change** in [`HANDOFF.md`](../../../docs/features/HANDOFF.md). Open its **Review findings** section
for the review being fixed, then read the FS/TS items named by those findings,
[`INVARIANTS.md`](../../../docs/features/INVARIANTS.md)'s trigger index and the classes the
findings touch, and workflow
§§2–6, §8, and §10 completely, then follow the fix process.

Use `write-tests` when behavior needs new verification or tests/fixtures materially change;
read [its instructions](../write-tests/SKILL.md) and follow TS-06.R35. Reuse adequate evidence; each added test must
protect a meaningful contract or plausible defect worth its runtime and maintenance cost.

For architectural findings, follow workflow §8's receipt closure: commit verified repairs, then
record their exact commits, finding IDs and focused evidence in the same architectural receipt.
Fixing architectural findings does not require another architectural review for that release.

`$ARGUMENTS` may scope the run to a finding priority or keyword; otherwise choose findings from any
one available review and say which one you took, then handle its **Must fix** items first as the
continuation of its originating change unit (workflow §1.4 and §8). Chronology and other role queues
do not constrain selection. Stop when that unit is closed rather than continuing into another unit.
The unit's §7 **Fix model** recommendation applies to the whole grouped run and reflects its most
difficult open fix; do not split the unit or choose a weaker model for its easier individual findings.
Before closing a medium or difficult unit, run workflow §8's independent check: a fresh-context
subagent on a model at that band or stronger gets the finding, the requirement and the fix diff,
and answers whether the fix holds at the real boundary, what it does to one adjacent path, and
whether the regression test can fail. For an inventory finding, repair the urgent Must-fix items,
then do the inventory and route its result to `docs/ideas.md` instead of patching the rest.
The fix commit
closes that unit; it does not create a new default review unit. Name what is still open when you
close. Name the finding's invariant class and the checks that passed in the commit message; the
handoff keeps no changelog. Update the
relevant specification when a fix changes behavior or fills missing coverage. Close with the handoff
update, commit, and concise human update.
