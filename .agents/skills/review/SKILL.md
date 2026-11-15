---
name: review
description: Explicit invocation only. Run only when the user sends `/review`; do not trigger from a natural-language request.
---

# Review the next substantive change

At startup, inspect `git status` and the existing diff before any repository edit. Use the injected
handoff header when present; otherwise read only **Current position** and **Active
change** in [`HANDOFF.md`](../../../docs/features/HANDOFF.md). Then read the FS/TS items named by the
handoff or request,
[`INVARIANTS.md`](../../../docs/features/INVARIANTS.md)'s trigger index — always, regardless of what
the handoff names — and workflow §§3–7 completely. Check whether code matches the specifications and
whether the specifications cover the code across the handoff's eligible change unit.

Then check the diff against that index and tag each finding with its class number; open a class
body only where the trigger matches.

For materially changed tests, apply TS-06.R35 and INV §17: challenge the meaningful contract,
independent oracle, real boundary and added confidence relative to cost. Do not demand more
tests merely for coverage.

Classify every fix independently from severity, then record exactly one §7 **Fix model**
recommendation for the selected review unit at the level of its most difficult open fix, using
workflow §7's bands. Record it once with the unit, not on individual findings, and repeat it in the
human update.

When every finding in the unit is trivial/easy, fix the unit in this session: read workflow §2 and
§8, repair each finding with its regression test, run the applicable closure, and commit the repair
with the review state, naming each finding and fix in the commit message. Record a unit with any
medium or difficult finding whole and leave it for `/fix`; do not split it or shade a
classification down. When the same subsystem already had findings fixed in the last 30 days, add
the inventory finding workflow §7 describes.

The human may name a commit/range in `$ARGUMENTS`; otherwise choose any available review unit named
by the handoff and say which one you took (workflow §1.4 and §7). Administrative commits never become
default review units: skip review reports, finding-fix commits, release records, and handoff,
archive, or queue-only commits when advancing the selected unit's review state. Chronology does not
constrain selection, and other role queues do not block review. A no-finding review closes its unit
without creating another review obligation; if no eligible unit is pending, report that and do not
make an empty commit.
Outside that trivial/easy repair, do not change product code or specs. Record every finding and
local-choice outcome, make the required state commit, and close with the concise human update. Findings keep that same unit open; they do not
create a second review unit.
