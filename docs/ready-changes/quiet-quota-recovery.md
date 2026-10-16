# Recover quota-interrupted work without noisy wake-ups

**State:** In progress (2026-10-17; slice plan in the handoff's Active change)
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

## Implementation progress

1. **Done** — reset evidence (FS-01.R48, TS-04.R96): `internal/runtime/quota_reset.go` parses
   the pinned Claude `· resets … (Zone)` clause and Codex `try again at …` suffix (process zone
   from the child env `TZ`, else Chuck's local zone), decodes rejected `_claude/rateLimit` epoch
   seconds, corroborates text/structured within the displayed minute, and flags
   credit/spend-cap text without a reset as `QuotaEvidence.Intervention`. RPC prompt errors pass
   their message/`data.message` too. `settleQuotaTurn` schedules `reset + state.QuotaResetBuffer`
   (60 s); the state validator requires that buffer. Fixtures follow the pinned formatters
   (Claude `iu` logic re-run under TZ; Codex `error.rs` 676–836 at rust-v0.159.3).
2. Next — state (TS-02.R50, TS-10.R45–R46): forward migration adding chain id, `next_action`,
   `checks_started`, check token/session/cleanup receipts; check scheduling/charging/result
   APIs; renewed-failure chain retention; migration raises a known due to reset + 60 s.
3. Runtime isolated check (TS-04.R97): separate adapter process via `spawnCmd`/`startCmd`,
   `clientCapabilitiesFor(offered)`, `applySessionConfig`; Claude `_meta.claudeCode.options`
   `persistSession:false, tools:[], strictMcpConfig:true`; Codex `/rename Chuck quota check`
   then `session/delete`, `CODEX_CONFIG` `mcp_servers.<name>.enabled:false`. Build the server
   spec like `composeResumeSpecContext` (`internal/server/resume.go`) without
   `composeResumeSpec`'s MCP/hook registration.
4. Dispatcher wiring plus restart/cleanup recovery (TS-10.R45–R47).
5. API/UI (TS-03.R72, FS-04.R59): projection `next_action`/`checks_remaining`, reason codes,
   banner/Settings copy, the FS-01.R50 notice sentence, CLI decoder.

Owed at closure: full serial Go matrix (`GOFLAGS=-p=1 make test`), tagged build, UI tests/build.

## Waiting on

None for implementation. Live provider/isolation/retention receipts remain acceptance gates before
claiming coverage for each provider/profile; they were not run during design.
