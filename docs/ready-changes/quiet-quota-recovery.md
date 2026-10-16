# Recover quota-interrupted work without noisy wake-ups

**State:** Waiting to start
**Why:** Direct `/design-feature` request on 2026-10-16. The operator confirmed reset extraction
and bounded recovery, requested checks outside the interrupted conversation, and approved named
Codex native archive records for isolated probes.
**Relevant requirements:** FS-01.R48–R51/A31–A34, FS-04.R59/A38; TS-04.R96–R97,
TS-10.R45–R47, TS-02.R50, TS-03.R72. Existing ownership/notice foundation: FS-01.R40–R45,
TS-04.R86–R88, TS-10.R39–R44, TS-02.R45, TS-03.R60. INV §1–§5, §7–§12, §15–§17.

## Outcome

Chuck uses applicable verified structured resets, then verified provider error text, to schedule
the same interrupted work after a one-minute buffer. When reset time stays unknown, at most six
hourly isolated target-model checks can authorize its normal continuation. Failed checks do not
wake or append to the original native conversation or Chuck transcript.

## Included work

Extend existing quota intent/dispatcher, failure normalization, owner admission and cancellation;
do not introduce another scheduler, task, Chuck agent or native provider integration. Use bounded
separate ACP sessions with the same effective provider/account/model: Claude non-persistent,
Codex explicitly named then archived through its existing delete operation. Suppress automatic
title-model calls, omit work/coordination context, isolate configured servers without changing
authentication, and keep exact-session cleanup separate from original-work delivery receipts.
Only an actual continuation produces the honest new Chuck notice. A successful small probe can
still be followed by quota on the larger conversation; that mismatch holds rather than loops.

Old unknown records stay unscheduled; explicit false, cancelled work, frozen permissions and
normal task/pipeline/room eligibility remain authoritative. No passive quota dashboard, `/usage`
scraper, API-key substitution, provider login side client, adapter patch or arbitrary date parser.

## How we will know it works

FS-01.A31–A34 and FS-04.A38 cover provider source/wire evidence, timezone/date/DST/rounding,
isolated same-profile check and native retention, all owner paths, six-check allowance across
renewed failures/restart, cancellation/cleanup/delivery races, and desktop/phone state/notice
journeys. Run the applicable TS-06 closure matrix during implementation. Native provider support
requires the bounded packaged/supported-installed live receipts in A28/A34; source inspection and
synthetic helpers are not substitutes.

[Investigation and pinned evidence](../features/QUOTA-RESET-FALLBACK-2026-10-16.md) records the
actual negotiated AIR/RPC carriers, Chuck's discarded text/data, independent Claude epoch-unit
receipt, different Claude/Codex date rules, lack of a common passive ACP check, Codex archive-only
deletion and the supported process config override. Reverify profiles on adapter/native updates;
unsupported versions/isolation remain manual, with no wake-probe fallback in the original chat.

## Waiting on

None for implementation. Live provider/isolation/retention receipts remain acceptance gates before
claiming coverage for each provider/profile; they were not run during design.
