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
- **Closed entries (2026-10-12/13):** blank phone entry, dev mobile CSS, phone group memory and
  phone chat drafts are archived in
  [`HANDOFF-closed-2026-10-13`](../archive/state/HANDOFF-closed-2026-10-13.md).
- **Reviewed — chat permission modes, project-page menu, hidden file changes; fix pending
  (2026-10-14):** `bac0d40`, `7cd50ff`; one review unit. Two Must-fix findings: an empty legacy
  mode-set response overwrites a different provider-reported mode, and the project background
  menu intercepts right-clicks on buttons. Findings, independent wire/browser evidence and fix
  recommendation below. Focused runtime/CLI/server checks, runtime race checks, 90 UI tests,
  46 presentation/style checks and 15 Chromium journeys pass apart from the intentional failing
  legacy-mode regression probe. Changelog: identified false mode success and form-control menu
  interception (INV §8/§10/§11/§12). Other review units remain unchanged.
- **Closed — projects home redesign (2026-10-14):** `40e414e` plus the fix commit. Unavailable-
  project cards now render their running count; `ProjectDashboard` regression with a missing
  project and running/stopped agents fails before the fix. Changelog: complete unavailable-project
  footers (INV §10).
- **Closed — mobile agent header, composer and model switch (2026-10-14):** `c3b3c70`,
  `73cc75b`, `f15ee50` plus the fix commit. Back is 44×44px and the model switch 44px tall
  (`phone.css`); `phone-render.mjs "" <out.png> --parity-check` now asserts both in the
  nine-case width/appearance matrix and passes. Changelog: restored 44px phone action
  targets (INV §10).
- **Closed — full operator CLI parity (2026-10-14):** `f6e7b2b`, `ff031e4`, `219b4b7` plus the
  fix commit; unit record is in Git history. Help and example now describe a complete replacement, name the omitted-field resets and start
  from `role show --id <role> --json`; no client-side merge. Help regression
  `TestOperatorRoleEditHelpDescribesFullReplacement`; the built-binary real-server journey adds
  full replacement, retained explicit false settings and title-only reset. Changelog: accurate
  role-edit replacement guidance (INV §10). Closure for all three fixes: `make test` (both Go
  variants), `make build` and UI build pass; UI 815/817, the two failures being the known
  `NewAgentModal` flake (reproduced on unchanged `d74fb39`).
- **Closed — chat images and attachments (2026-10-14):** `e8c50f7`, `395c1a8`, `9ae6f59` plus
  the fix commit on `fix/chat-attachments-20261014`. Startup recovery now clears an owned item's
  pending marker without byte validation and isolates per-conversation failures (logged,
  copies preserved), so a corrupt owned blob is only a per-item read error
  (`TestRestartWithCorruptOwnedAttachmentKeepsDeliveryAvailable`). `DeleteAgent` removes owned
  bytes before the identity row (`TestDeleteAgentKeepsOwnershipUntilAttachmentCleanupSucceeds`).
  Local-image Escape routes through `setDialogOpen` (two `AttachmentView` regressions). All four
  regressions fail on the pre-fix code. TS-02.R47 records deletion ordering and per-conversation
  recovery. Closure: `make test` (both Go variants) passes with `GOFLAGS=-p=1`; the
  parallel run failed only the known `TestToolImageFrameBoundsAcrossActualWire` timing flake.
  `make build` and `make embed` pass; UI 816/819, the failures being the known `NewAgentModal`
  flake. Changelog: isolated attachment recovery, retryable conversation deletion and
  Escape modal teardown (INV §4/§7/§15/§16).
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
  No runtime code or test was changed; timing cause remains uninvestigated.

## Active change

None. Full operator CLI parity, projects home redesign and mobile agent header units closed
2026-10-14 with their review fixes. CLI parity and attachment closure notes are archived
in [`HANDOFF-closed-2026-10-13`](../archive/state/HANDOFF-closed-2026-10-13.md).

## Unit notes

- **Chat permission modes, project-page menu, hidden file changes (2026-10-13).** `bac0d40`:
  FS-02.R77/A58 project-page background menu (`CardGrid` `openersRef`); FS-03.R87/A66 diffs
  hidden from desktop/phone agent chats (`hideFileChanges`; Files-tab Diff reveal keeps its diff;
  archive/Think Tank keep inline). Permission modes: FS-03.R88/A67, TS-03.R70, TS-04.R92, FS-22.R8
  inventory. `internal/runtime/session_mode.go` decodes the advertised mode list (mode-category
  config option, legacy `modes` fallback); live-only state under its own `modeMu`, so read-loop
  mode updates never wait on `as.mu`. `GET /modes`, `POST /mode` (lifecycle claim, phone-denied),
  CLI `agent modes|mode`; desktop `PermissionModeToggle`, reread on agent state change.
  Review notes: Claude 0.59.0's mode option is source-checked only; the Codex mode shape and live
  provider receipts are unverified. Pre-existing risk observed, unchanged: `SetSessionConfig`
  holds `as.mu` across its provider call while `onNotification` takes `as.mu` for unmatched frames
  on the synchronous read loop, so an adapter notification sent before that reply could stall it.
  Rendered runtime band: `/private/tmp/chuck-modes-shots`. Changelog: provider-advertised modes as
  live state with bounded copy and lock-free read-loop handling (INV §1/§3/§5/§8/§10/§11/§16).

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

### Chat permission modes, project-page menu and hidden file changes — `bac0d40`, `7cd50ff` — **Fix model:** medium — Codex Terra or Claude Opus.

Reviewed independently 2026-10-14 against FS-02.R77/A58, FS-03.R87–R88/A66–A67,
FS-22.R1/R3/R8, TS-03.R70 and TS-04.R92. Administrative state commit `ada1b49`, unrelated
phone-sheet change `2930250` and existing uncommitted CLI/dashboard/phone fixes are excluded.

- **Must fix** — **an empty legacy mode-set reply overwrites the provider's reported mode
  (INV §8/§11/§12).** Location: `internal/runtime/session_mode.go:234`. A legacy adapter
  advertises Manual and Plan, receives a request for Plan, reports Manual through
  `current_mode_update`, then returns the normal empty `session/set_mode` result `{}`.
  The notification correctly records Manual, but the fallback calls `setCurrentMode(id)`,
  overwriting it with Plan and returning success. Desktop and CLI therefore claim a mode the
  session did not adopt, instead of restoring the reported selection and surfacing the ignored
  change required by FS-03.R88 and TS-04.R92/TS-03.R70. An independent actual-wire fake-provider
  overlay probe fails with `Current:plan` and `err:<nil>` when the required result is
  `Current:default` and `ErrSettingIgnored`. Preserve any provider mode notification received
  during the call before applying an optimistic fallback. A pre/post value comparison is
  insufficient because the notification can report the same mode as before the call. Cover
  both unchanged and changed reported modes, successful legacy sets without notifications,
  and the existing config-option path. This fix has medium difficulty because it coordinates
  the read-loop notification and setter under the existing mode lock.
- **Must fix** — **the scoped-project background menu intercepts button right-clicks
  (INV §10).** Location: `ui/src/features/dashboard/ProjectDashboard.tsx:339`. The exclusion
  selector covers inputs, textareas and selects but omits buttons. Right-clicking the header's
  Think Tank button opens New agent/New think tank and prevents the control's own context-menu
  behavior, violating FS-02.R77's explicit form-control exclusion. Independent Chromium
  reproduces this at 1024px and 1440px in Core, Sky & Grove and Studio. Extend the existing
  exclusion to buttons (including nested button content); add regressions for header controls
  while retaining background, card, dialog and menu behavior. This is a trivial/easy fix.

Other code and specifications agree in both directions. Reusing the header's dialog openers,
shared diff filtering after reasoning placement, explicit archive opt-in, provider-advertised
mode inventory, bounded display copy, a separate mode lock, per-agent lifecycle claims,
phone route denial and live-only mode state are sound local choices. No human decision is needed.

Verification: 90 focused UI tests pass across PermissionModeToggle, TranscriptView,
ScopedProjectDashboard, AgentScreen and VisualMatrix; 46 style/presentation checks pass.
Fresh CLI wire-inventory and live fake-provider server mode-route tests pass with authorized
loopback execution. Runtime tests and focused mode race tests pass. The intentional failing
legacy-mode regression was independently rerun from
`/private/tmp/chuck-legacy-probe/{overlay.json,legacy_mode_probe_test.go,fake-overlay.json}`;
no repository test or product file was edited. Initial sandbox CLI/server/browser attempts
could not bind loopback; bounded approved executions completed those checks.

The actual working-tree UI passes six desktop journeys at 1024/1440px and nine phone journeys
at 360/390/430px across all three appearances. Desktop checks cover advertised radio choices,
keyboard selection, provider refusal and selection restoration, reread after agent state change,
hidden transcript diffs, Files-tab reveal of exactly one requested diff, retained archive diffs,
both background-menu dialog openers, card exclusion, Escape dismissal and horizontal containment.
All six desktop cases additionally reproduce the button-interception finding. Phone checks
confirm no permission-mode control, hidden conversation diffs, exactly one requested Open diff
card and viewport containment. Inspected settled screenshots show readable labels and inline
refusals, coherent runtime grouping and contained phone diffs. Evidence:
`/private/tmp/review-modes-ui.mjs` and `/private/tmp/review-modes-ui/{receipt.json,*.png}`.
Fixtures make no live-provider, physical-phone or tailnet claim; those provider uncertainties
remain as recorded in TS-04.R92 and Unit notes.

Invariant sweep: applicable §1, §2, §3, §4, §5, §7, §13, §14, §16 and §17 have no findings;
§8/§11/§12 have the mode-report finding and §10 has the button-exclusion finding above.
§6, §9 and §15 have no applicable changed surface: no new runtime/interface, durability or
process-liveness primitive, or durable-commit/side-effect ordering change. Product code and
specifications are unchanged. These findings keep this same unit open; every other unit is
unchanged. Commit only this review state.

### Phone group memory and long-press agent actions — `17b73ed` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Closed, no findings.

Reviewed independently 2026-10-13 against FS-20.R47–R48/A19 and TS-08.R121.

Code and specifications agree in both directions. Project/group identity keys, the page-session
256-entry recency cap, reuse of AgentManagement forms and PhoneSheet, and existing remote API
clients/allowlist are sound local choices; none requires a human decision. Gesture timers and the
scroll listener have cancellation and unmount cleanup. Offline permits Open chat and disables
mutations; runtime changes remain running-chat-only and existing confirmation/error rules apply.

Verification: ProjectScreen's 14 tests and 46 presentation/style checks pass. Independent Chromium
journeys pass at 360/390/430px in Core, Sky & Grove and Studio using the current working tree's
actual remote entry. They verify collapse retention across conversation navigation, a native
held touch without navigation, a subsequent ordinary tap, keyboard action launch, refused rename
value retention, refused Stop confirmation with cancellation, stopped Resume/runtime availability,
Escape dismissal and row-focus restoration. Settled screenshots show readable sheets, visible
destructive actions and refusal feedback, consistent styling, and no horizontal overflow.
Fixtures make no live-provider, physical-device or real-tailnet claim. Evidence:
`/private/tmp/review-phone-actions-20261013.mjs` and
`/private/tmp/review-phone-actions-20261013/{receipt.json,*.png}`.

Invariant sweep: applicable §1, §2, §3, §4, §8, §10, §11, §13, §14, §16 and §17 have no findings.
§5, §6, §7, §9, §12 and §15 have no applicable surface in this unit. Product code and
specifications were unchanged. This review closes only this unit and creates no review obligation.

### Phone chat drafts and Theme selection — `79cbbc3` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Closed, no findings.

Reviewed 2026-10-12 and independently verified 2026-10-13. The shared draft seam now synchronizes
accepted cleanup to mounted and reopened composers while its revision guard preserves newer text.
The existing same-composer regression covers newer-draft protection; focused AgentScreen tests
add delayed text-only Send and Steer leave/reopen/accept flows, with 41 AgentScreen tests passing.
No product specification change was needed because the fix restores
FS-03.R36 and FS-20.R49/A20 behavior already specified.

Review evidence: 66 focused UI tests passed (AgentScreen, PhoneSettings, PhoneApp and drafts);
the two focused remote appearance/field-filter server tests passed. Existing PhoneApp fixtures
emit missing `/api/roles` MSW warnings without failing. The working tree's actual remote entry
passed `phone-render.mjs --parity-check` with isolated Chromium at 360/390/430px in all three
appearances; inspected screenshots show a labelled, readable Theme control, adequate touch height
and no horizontal overflow. Receipts: `/tmp/review-phone-parity-20261012*.png`. Fixtures make no
live-provider or real-tailnet claim. Shared draft/config reuse, partial config merge, finite skin
selection and query ownership are sound local choices; none needs a human decision.

Invariant sweep: applicable §1, §2, §3, §8, §10, §11, §13, §14, §15, §16 and §17 have no
findings. §4, §5, §6, §7, §9 and §12 have no applicable surface in this unit. Product code and
specifications were unchanged.

The mobile outage investigation was closed by the user on 2026-10-12 (see Current position).
Closed 2026-10-11 reviews are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.14.0-notes.md`](../archive/state/RELEASE-v0.14.0-notes.md) (unpublished until
pushed) and [`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
