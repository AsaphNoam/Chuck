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
- **Open investigation (2026-10-14):** stopped agents last run before image negotiation refuse
  image Sends; see the investigation unit under **Review findings** (ready for `/fix`).
- **Repository:** `AsaphNoam/Chuck`; branch `main`. `origin` URL corrected 2026-10-14 (it had a
  stray trailing `.`).
- **Known risk (unchanged, uninvestigated):** runtime `SetSessionConfig` holds `as.mu` across its
  provider call while `onNotification` takes `as.mu` for unmatched frames on the synchronous read
  loop, so an adapter notification sent before that reply could stall it.
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
  Runtime `TestToolImageFrameBoundsAcrossActualWire` exceeded its unchanged three-second
  collector for 25-MiB frames during CLI closure on 2026-10-13. Isolated tests and full runtime
  retries passed, then the complete matrix passed with package execution serialized (`GOFLAGS=-p=1`).
  No runtime code or test was changed; timing cause remains uninvestigated. On GitHub CI
  (parallel packages) it failed on both 2026-10-14 `main` runs (38512698255, 38513996157), so
  `main` CI is red until it is investigated.

## Active change

None.

## Acceptance gates still owed

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

Closed 2026-10-12–14 reviews are in the archive above.

**Investigation unit — stopped agent refuses an image (2026-10-14, `/investigate-bug`).**
Report (verbatim): “Trying to send a message with an attached image when an agent is stopped gives
an error that this provider doesn't support images.” No version, provider or logs supplied; the
refusal is client-side, so nothing would be logged. Read as current `main` (`v0.15.1`).
**Fix model:** medium — Codex Terra or Claude Opus.

- **Must fix** (confirmed — reproduced) — Stopped agents from before image negotiation refuse images.
  `image_input` joined the frozen capability snapshot in `395c1a8` (2026-10-13). A stopped agent
  whose last handshake ran on an older build has `sessions.runtime_capabilities_json` without that
  key; `state.DecodeRuntimeCapabilities` reads it as false, and both `AttachmentCapabilitiesFor`
  projections (`internal/state/manager.go` state snapshot, `internal/server/launch.go` session read)
  publish `image_input:false`. `ui/src/components/chat/Composer.tsx` (`capabilityError`, Send/Steer
  disabled) and `ui/src/remote/AgentScreen.tsx` then refuse with “This runtime does not accept image
  input…”. The server path is correct: Send → `ErrNoHandle` → `wakeAgent` → resume renegotiates
  `initialize` → the image is delivered (202). Normal-use trigger: upgrade to `v0.15.x`, then attach
  an image to any agent not run since; manual Resume clears it. Violates FS-03.R83 (stopped-agent
  wake carries the message’s attachments) and TS-04.R90 (wake revalidates capability at
  consumption). Spec gap alongside it: TS-01.R35/TS-03.R63 do not say what the projection reports
  when the snapshot has no image advertisement on record — `types.go` calls zero “the answer for old
  rows”, which here turns unknown into a refusal. Suggested fix: decode an absent `image_input` as
  unknown (omit it or make it nullable in `attachment_capabilities`), have both composers allow an
  unknown image and rely on the server’s wake-time revalidation (`attachment_capability` 422 still
  surfaces a real refusal), and record the unknown state in TS-03.R63/TS-01.R35. A genuine frozen
  `false` from a real handshake keeps blocking. Regression test: un-skip
  `internal/server/attachment_wake_repro_test.go`
  `TestStoppedAgentWithPreImageSnapshotDoesNotRefuseImages` (fails today on the `image_input:false`
  projection; its wake Send already returns 202), plus a Composer/AgentScreen test that an unknown
  image capability leaves Send enabled.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
