# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-11 is archived in
[`HANDOFF-through-2026-10-11`](../archive/state/HANDOFF-through-2026-10-11.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.14.0` tagged locally 2026-10-11 at the release commit, **unpushed**; pushing
  `main` and the tag needs explicit user authorization, then publication verification (workflow
  §16.6). `v0.13.0` was published 2026-10-10. The user waived the open review units below for
  release; that is not completed review. Manual provider gates below remain owed.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Awaiting review — mobile agent header, composer and model switch:** `c3b3c70`, `73cc75b`,
  `f15ee50`; FS-20.R43, TS-08.R111. New agent alone uses the centered PhoneSheet. The back
  chevron sits beside the persona eyebrow above the agent name; context usage sits between
  identity and tabs; Stop/Resume is a sibling of the tablist. The composer encloses a borderless
  textbox (no blue typing outline) and Send at bottom right. Model/effort is a right-aligned,
  bordered, raised button above the composer that opens the Switch runtime sheet. Final state:
  73 phone tests, 41 presentation/style checks, spec check and `make dist` pass; full UI suite
  790/791 with only the NewAgentModal flake below. `ui/scripts/phone-render.mjs --details-check`
  passes all three appearances at 360/390/430px; receipts `/tmp/mobile-model-20261011-*.png`.
- **Awaiting review — phone group memory and long-press agent actions:** `17b73ed`,
  FS-20.R47–R48/A19, TS-08.R121. Project-scoped page-session group choices and a long-press
  action sheet over shared phone management forms. Receipts: shared project resources
  `mobile-actions-20261011/`. Fixtures make no provider claim.
- **Awaiting review — phone chat drafts and Theme selection:** FS-20.R49–R50/A20–A21,
  TS-08.R122, TS-13.R25. Per-agent browser-local drafts via `components/chat/drafts.ts` and an
  appearance-only `PUT /api/config` for paired phones. Receipt: `phone-render.mjs --parity-check`.
- **Design awaiting scope confirmation — CLI parity:** `docs/ideas.md` proposes full nonvisual
  web parity with composable terminal input/output. No requirements or active change yet.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61. Full passive paired snapshots remain TBD for Claude and Codex; evidence is
  in [`ideas.md`](../ideas.md). Not a ready unit. Do not implement a hidden prompt, command
  parser, login side client or new adapter patch.
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Source-build Go tests:** run with `CHUCK_RUNTIME_ROOT` unset; an inherited installed bundle
  violates a credential test's source-build assumption.
- **Known flaky checks:** `internal/server` `TestOrdinaryStageAgentStopPausesPipelineRun`
  intermittently returns 409 “a resume is already in progress” (pre-existing at `74c8e84`).
  `ui` `NewAgentModal.test.tsx` fails 1–2 cases intermittently on unchanged `d3c9307`
  (observed 2026-10-11). Neither is investigated.

## Active change

None. The v0.14.0 release is cut; the next change starts from a review unit above or a new request.

## Acceptance gates still owed

- **FS-01.A28:** bounded packaged/installed Claude/Codex terminal-quota and owner-activation
  provider receipt. Current Claude SDK 0.3.286's bare numeric reset declaration has no verified
  producer-unit receipt; Codex forwards no typed reset. Both remain unknown and unscheduled.
  Known-reset state/fake-provider/browser proof is not live-provider reset or delivery evidence.
- **FS-14.A50 / FS-21.A31:** bounded packaged Claude/Codex Think Tank stage probe (start → fresh
  participants → judge → accepted output → next stage). Fake-provider E2E and the 318-check render
  pass are not live receipts. **FS-14.A49** restart/race matrix and **FS-21.A32** live Stop-cancel
  race remain planned.
- **FS-21 / TS-06.R33:** credentialed Claude/Codex Think Tank smoke covering room read/submit,
  addressed input, approval/denial, private Send/Steer, native resume, end-only judge, and retained
  exact result. Fake-provider journey and automated/race closure passed; live receipt is owed.
- **FS-10.R25/A14 / TS-02.R41:** supervised disposable-real-home cutover rehearsal with receipt.
- **FS-02.A46 / A27:** real-browser toast and macOS notification clicks; six-tab shared-stream,
  J14 real-browser, and Sky & Grove with Codex capabilities.
- **FS-14.A54:** packaged Claude/Codex probe that a standing owner/dedicated coordinator adopts
  shared orchestrator instructions; fake delivery capture is not a live receipt.
- **FS-18.A12/A13:** six manual role scenarios and role-free follow-ups on pinned Claude/Codex;
  credentialed fresh/resumed Claude native-preset adoption (pinned adapter 0.85.1 only proves
  the `_meta.systemPrompt` request, not live adoption).
- **FS-20.A1/A5/A6/A8:** real tailnet, Android, iPhone, and `pmset -g assertions`; iPhone Home
  Screen PNG touch icon is absent. The combined 390px dashboard/project/agent-management/Files/
  Commands-retired journey remains owed; fast-mode picker and Continue on approval pause remain
  outside the fake-provider coverage.
- **TS-06.R21:** credentialed Claude/Codex login/chat.
- **TS-06.R31 / FS-09.A40/A42/A46/A47/R78 / FS-10.A10–A12:** finite four-combination provider
  smoke (each provider with current Bundle and one current Installed CLI), plus Installed update →
  Refresh → model and missing Installed → Bundle → retry → Installed-with-overrides journeys.
  Claude Installed fresh launch passed 2026-10-05; the broader credentialed matrix is owed.
- **TS-06.R26/R32:** credentialed Codex 2.1.1 receipt for FS-03.A41/A42 and FS-01.A20.
- **TS-06.R32:** two-point Claude/Codex smoke on ACP 0.85.1/2.1.1 and Codex 0.159.3, one notice,
  Steer during a running command, and refused model switch where policy allows. Automated assembly,
  fake-runtime, release-CI assembly, and native-provider proofs passed; live receipts are owed.

## Decisions needing input

- Whether to standardize TS-03.R3–R4 mixed legacy error envelopes or TS-04.R3 provider model-ID
  ownership (compatibility change).
- Whether a pipeline stage paused after failed launch/resume continues withholding **Open agent**,
  matching restart recovery, or permits chat under a wider continuation contract.
- CLI parity product scope (see Current position).

## Design consistency notes

- Next presentation review: decide whether crisp asymmetric technical-surface radii are deliberate
  under FS-12.R52.
- When direct-action work resumes, align TS-01.R25/TS-03.R32 with TS-04.R40 and scope FS-17
  section 6's opening sentence to the planned boundary.
- Injected-steer lifetime edge case needs `/investigate-bug` before `/fix`; FilesTab still copies
  silently through bare `writeText`.
- The phone agent header still repeats the model as text under the name now that the model
  switch shows it above the composer.

## Review findings

### Mobile outage investigation — 2026-10-12 — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Report verbatim: “mobile is unresponsive, even though tailscale shows the Mac is connected
(but chuck isn't)”. Environment: this Mac, `~/.chuck`, configured desktop port 4317,
`remote_enabled=true`, `keep_awake=true`; running build/launch method at failure unknown.
Repository HEAD at investigation: `605ea3e` (local v0.14.0 release). No user-supplied logs;
local `dashboard.log` and live process/tailnet evidence were inspected.

**Confirmed outage condition:** the pidfile names PID 84843, which no longer exists; no desktop
listener answers port 4317. Tailscale reports the Mac and Android phone online but the separate
embedded `chuck` device offline, last seen about four hours earlier. The log records successful
phone requests through 2026-10-11 22:46:22 +03:00 and desktop requests through 22:57:23.745614,
then ends without `dashboard shutting down`, a fatal error, or a panic. The previous orderly
shutdown at 21:56:37 and start at 21:57:01 are logged. No Chuck crash report or matching macOS
termination event was found in the inspected 22:56:50–22:58:10 window. Chuck being stopped
explains its tailnet device being offline (FS-20.R1, TS-13.R1–R2); why it exited is
**undetermined**, with an abrupt exit **probable**, not a confirmed crash. A connected Mac's
independent Tailscale app cannot keep Chuck's embedded node alive. Recovery: start the normal
dashboard detached, then verify desktop health, remote `on`, and the phone; no runtime state was
changed by this investigation.

- **Must fix** — MOBILE-OFFLINE-1 — confirmed defect; contribution to this field symptom
  undetermined; fix complexity trivial/easy. `ui/src/remote/api.ts:24` and
  `ui/src/remote/connection.ts:55,95`: a health fetch has no deadline. When a lost tailnet path
  leaves it pending, initial classification remains `checking`, so PhoneApp stays on
  “Connecting to your Mac…”; the five-second SSE-error timer likewise awaits that unbounded
  classification and can leave an existing view `reconnecting`. FS-20.R23 and INV §8/§9 require an explicit
  unreachable/stale state. Reproduced with a fetch that never answers: after 30 seconds of fake
  time, link is still `checking`, not `unreachable`. This is a generous regression-test ceiling,
  not a new specification deadline. `ui/src/remote/connection.test.ts` contains the reproduced
  assertion, committed skipped under §12. Suggested fix: bound the read-only health probe,
  classify timeout as unreachable, and retain automatic reconnect; do not add retries or replay
  to mutations or put the health timeout on pairing's intentional long poll. Unskip the test and
  cover a stalled probe after SSE failure as well as recovery to connected.
- **Worth fixing** — MOBILE-OFFLINE-2 — confirmed observability gap; exit cause undetermined;
  fix complexity trivial/easy. `internal/cli/dashboard.go:178–207` and
  `internal/server/server.go:534`: the retained log identifies the listening address but not the
  dashboard PID/build/start mode, and a new start silently overwrites a stale pidfile. In this
  outage there is no exit record, so future restart loses the remaining process identity evidence.
  TS-01.R15 supplies the durable diagnostic sink; INV §9 governs process-liveness evidence;
  workflow §12.5 requires a diagnostic finding
  when silence prevents diagnosis. Log bounded, secret-free startup identity and the discovery
  of a dead prior PID before overwriting it, plus handled shutdown signal/error where available.
  Test these records for foreground/detached starts. This can identify an unclean prior exit and
  the affected build; it cannot capture SIGKILL or explain an OS kill from inside the dead process.

Investigation verification: live process/local endpoint/tailnet checks; relevant code/spec trace;
reproduction failed as expected before marking skipped. Focused phone suite: 14 passed, one
reproduction skipped (existing unhandled `/api/roles` mock warnings). `make check-specs` passes.
Product code and specifications are unchanged.

Closed 2026-10-11 reviews (group controls, persona, quota continuation, Think Tank
room page, button redesign, mobile details/composer) are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.14.0-notes.md`](../archive/state/RELEASE-v0.14.0-notes.md) (unpublished until
pushed) and [`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
