# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-14 is archived in
[`HANDOFF-through-2026-10-14`](../archive/state/HANDOFF-through-2026-10-14.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.15.1` published 2026-10-14 (Latest): release CI run 38513998402 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from the
  notes file. Manual provider gates below remain owed. Pushed tag `v0.15.0` has no
  GitHub Release: its release CI (run 38512698302) failed on the CLI journey test closed below;
  leave that tag in place. `v0.14.0` stays local-only by user decision. `v0.15.1` notes cover
  everything since the last published `v0.13.0`. Range `v0.15.0..v0.15.1` changes nothing
  agent-facing.
- **Closed — operator CLI journey hermeticity (2026-10-14):** `TestOperatorBuiltBinaryRealServerJourney`
  resolved the host's real Claude executable, failing CI and release CI on hosts without it. It now
  points `CLAUDE_CODE_EXECUTABLE`/`CODEX_PATH` at inert stand-ins like `internal/server/main_test.go`.
  Failed before the fix with `HOME` isolated and no provider CLIs on `PATH`; passes after. Full
  `make test` (both Go variants) passes. Changelog: hermetic CLI journey provider selection
  (no invariant class).
- **Repository:** `AsaphNoam/Chuck`; branch `main`. `origin` URL corrected 2026-10-14 (it had a
  stray trailing `.`).
- **Known risk (unchanged, uninvestigated):** runtime `SetSessionConfig` holds `as.mu` across its
  provider call while `onNotification` takes `as.mu` for unmatched frames on the synchronous read
  loop, so an adapter notification sent before that reply could stall it.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61. Full passive paired snapshots remain TBD for Claude and Codex; evidence is
  in [`ideas.md`](../ideas.md). Not a ready unit. Do not implement a hidden prompt, command
  parser, login side client or new adapter patch.
- **Closed — think tank non-leaver filler (2026-10-14):** FS-21.R16/R20/R52–R53/A38, TS-14.R5.
  Any participant can now submit disposition `pass` on a discussion turn (no message; charges one
  turn, advances the checkpoint, publishes a bodiless `pass` marker, stays eligible); discussion
  guidance names it, so following “nothing to add” no longer holds the room. User-confirmed
  product rule: once every eligible participant has passed since the last non-pass entry,
  discussion ends with `nothing_further` (closing skipped, judge runs). The un-skipped
  reproduction failed before with “disposition must be reply, leave or decline_closing” and passes;
  new `TestThinkTankPassAndQuietRotation` (state) and room-page marker test. `make test`, UI suite
  (known `NewAgentModal` flake only), UI and Go build pass; not run through a live provider.
  Changelog: think tank pass disposition (INV §8/§10/§11/§17); quiet-round ending (INV §10/§17).
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Source-build Go tests:** run with `CHUCK_RUNTIME_ROOT` unset; an inherited installed bundle
  violates a credential test's source-build assumption.
- **Known flaky checks:** `internal/server` `TestOrdinaryStageAgentStopPausesPipelineRun`
  intermittently returns 409 “a resume is already in progress” (pre-existing at `74c8e84`).
  `ui` `NewAgentModal.test.tsx` fails 1–2 cases intermittently on unchanged `d3c9307`
  (observed 2026-10-11). Neither is investigated. `FileViewer.test.tsx` "scrolls to and marks a
  cited line" failed once in the full UI suite on 2026-10-14 and passes 3/3 alone.
  Closed 2026-10-14 — image-frame CI timeout: large-frame runtime tests (image frame and both
  10 MiB attachment tests) use `drainTurnWithin` 20 s instead of the fixed 3 s collector; the
  10 MiB image test also failed under `-race` on unchanged code. `onNotification` now peeks each
  `session/update`'s id and kind once (`peekSessionUpdate`) and runs only that kind's decoder:
  JSON decode CPU for the 25 MiB frame fell from 1.17 s to 0.71 s and the subtest from ~2.4 s to
  ~1.9 s. `make test` (both variants, parallel packages) and runtime `-race` pass. Changelog:
  single-peek ACP update routing (INV §16); large-frame test budgets (no invariant class).
- **Closed — phone model and effort split (2026-10-14):** FS-20.R43, TS-08.R111 (`b1bc5284`);
  review found no findings.
- **Reviewed — phone scroll memory, finding open (2026-10-14):** FS-20.R54/A26, TS-08.R125. A bounded
  (50-entry) page-session map keeps each chat's top visible message row (`data-place`) or
  at-newest, and other routes' `scrollY`. First open and left-at-newest open at the newest message.
  86 phone tests (scrollMemory bound test new), tsc, presentation checks, spec check and embed
  pass. `phone-render.mjs --scroll-check` passes (row restored within 0px, newest kept, project
  page restored) and fails with restoration disabled; `--details/--history/--parity-check` pass.
  Full UI suite 823/823 after dependencies were restored. Under load one earlier run failed two
  AgentScreen pagination tests that pass 42/42 alone with and without this change.
  INV: §8/§13 no new class or message; §16 map capped at 50.
  Independent review of `e522eb1a` on current `main`: the existing `--scroll-check` passes,
  but the live-revision/cache-miss journey loses a project position (1000px → 0px); see the
  review finding below. All 101 remote tests pass. The unit stays open. No validated trivial/easy
  findings.

- **Closed — mobile wallpaper restoration (2026-10-14):** FS-20.R43, TS-08.R111 (`33987801`);
  review found no findings.

- **Awaiting review — agent page width and side panels (2026-10-14):** FS-12.R61, TS-08.R112.
  `7a0228e5`, `30204eaf`: card ~80% of the window; conversation and composer 85% of the card
  (7.5% insets, held when panels narrow it). File viewer and annotation tray sit outside the
  card via `AgentStage` portal slots. Chat/archive tests, presentation checks, embed pass;
  full suite fails only the `NewAgentModal` flake. Receipts: `agent-design/runs/eighty/`.

- **Awaiting review — project rooms at the bottom, collapsible (`4440e79e`, 2026-10-14):**
  FS-02.R71/A53, FS-12.R63, FS-21.R44/A33, TS-08.R96. Rooms follow the agent grid; the heading
  is an `aria-expanded` toggle (count stays visible) remembered in `localStorage`
  `chuck.think-tank-rooms-collapsed` (shared with Archive). Grid header lost "Live operations".
  838/841 UI tests (only the `NewAgentModal` flake, 34/34 alone), presentation checks, tsc and
  spec check pass; rendered via `room-render.mjs <out> project` at 1440. Owed: rendered check
  with real agent cards above the rooms and keyboard toggle focus.

- **Awaiting review — conversation-only archive search (2026-10-14):** FS-05.R41/A24
  (`9627fa96`). Projection no longer indexes tool name/args/output, diff path/text or
  permission reason; prompts, attachment names, replies and annotations stay. Old documents keep
  tool text until `chuck reindex`. New archive test fails without the fix; both Go variants and
  build pass. `check-specs` failed only on another session's uncommitted FS-21.R52/R53 citations.

## Active change

None.

## Acceptance gates still owed

- **FS-20.A22–A24 / TS-13.R30:** paired iPhone Home Screen app and Android browser
  background-return journey (stalled stream, outage, revoked pairing). Simulated/browser proof
  passes; the device suspension cause is still unreproduced.

- **FS-03.A67 / TS-04.R92:** live Claude/Codex permission-mode receipt. Claude 0.59.0's mode
  option is source-checked only; the Codex mode shape is unverified.

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

### Review unit — phone scroll memory (`e522eb1a`, reviewed 2026-10-14) — **Fix model:** medium — Codex Terra or Claude Opus.

Code and specification review plus independent Chromium verification on current `main`. The existing
`phone-render.mjs --scroll-check` passes (chat row restored within 0px, newest kept, project at
179px); `npx vitest run src/remote` passes all 101 tests. The review fixture
`/tmp/chuck-scroll-review.mjs` adds 20 active runs, a live
`state_update` while viewing the chat, and a 300ms project/home response delay: the project was
left at 1000px and returned at 0px after its content loaded (exit 1; screenshot
`/tmp/chuck-scroll-review-revision.png`). No local-choice notes needed resolution. Invariant sweep:
§1/§10 expose the restoration lifecycle/wiring defect below; §4/§8/§13/§16 checked with no additional
finding; §17's existing journey misses the live-revision/cache-miss case. No applicable surface for
§2/§3/§5/§6/§7/§9/§11/§12/§14/§15. User authorized all trivial/easy repairs; this finding is medium
and remains for a fix session.

- **Must fix** (confirmed, Chromium reproduction) — `ui/src/remote/PhoneApp.tsx:42–50` restores
  non-chat scroll only when the route changes, before asynchronous page content is ready.
  Normal-use trigger: scroll a long project/run page, open a chat, receive a live state/task/pipeline
  update, then return. ProjectScreen/RunScreen include the connection revision in their query keys;
  on remount the new revision has no cached data, and placeholder data cannot carry across that
  unmount. The browser clamps the saved `scrollY` to the shorter initial page, and the effect does
  not retry after content arrives. This loses the reading position promised by FS-20.R54/A26 and
  TS-08.R125. INV §1/§10/§17. Suggested fix: coordinate non-chat restoration with page data readiness,
  retain the saved target until it can be applied, and cover a delayed response after a live revision
  change in the phone-width browser journey (project and run; Home also uses revision-keyed queries).

Closed 2026-10-12–14 reviews are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
