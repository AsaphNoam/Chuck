# Chuck handoff — settled through 2026-10-14

Archived at the `v0.15.0` release cut. Non-authoritative history; the live state is
[`HANDOFF.md`](../../features/HANDOFF.md).

## Current position (settled)

- **Release:** `v0.14.0` tagged locally 2026-10-11 at the release commit, **unpushed**; pushing
  `main` and the tag needs explicit user authorization, then publication verification (workflow
  §16.6). `v0.13.0` was published 2026-10-10. The user waived the open review units below for
  release; that is not completed review. Manual provider gates below remain owed.
- **Closed entries (2026-10-12/13):** blank phone entry, dev mobile CSS, phone group memory and
  phone chat drafts are archived in
  [`HANDOFF-closed-2026-10-13`](HANDOFF-closed-2026-10-13.md).
- **Closed — chat permission modes, project-page menu, hidden file changes (2026-10-14):**
  `bac0d40`, `7cd50ff` plus fixes `a27f8d2`, `96d828f`. Empty legacy replies preserve provider
  updates, including unchanged values; project background menus exclude buttons and nested
  content. Actual-wire mode and header-control regressions failed before their fixes and pass
  afterward. Changelog: restored effective-mode reporting and form-control exclusions
  (INV §8/§10/§11/§12). Both findings are closed; combined verification is recorded below.
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
  fix `e8ffaf6`, merged into `main`. Startup recovery now clears an owned item's
  pending marker without byte validation and isolates per-conversation failures (logged,
  copies preserved), so a corrupt owned blob is only a per-item read error
  (`TestRestartWithCorruptOwnedAttachmentKeepsDeliveryAvailable`). `DeleteAgent` removes owned
  bytes before the identity row (`TestDeleteAgentKeepsOwnershipUntilAttachmentCleanupSucceeds`).
  Local-image Escape routes through `setDialogOpen` (two `AttachmentView` regressions). All four
  regressions fail on the pre-fix code. TS-02.R47 records deletion ordering and per-conversation
  recovery. Both Must-fix items and the Worth-fixing item are closed.
  Changelog: isolated attachment recovery, retryable conversation deletion and
  Escape modal teardown (INV §4/§7/§15/§16).
- **Combined fix integration (2026-10-14):** permission modes, project menu and attachment
  branches merged locally into `main` (`ce06af5`, `b237c61`, `9ec87bd`); no push. Shared handoff
  entries reconciled and embedded UI regenerated from the combined source with `make embed`.
  `env -u CHUCK_RUNTIME_ROOT GOCACHE=/private/tmp/chuck-gocache GOFLAGS=-p=1 make test`
  passes both full Go variants; `make build`, UI build, all 821 UI tests in 89 files with two
  workers, all 46 presentation/style checks and focused permission-mode race tests pass.
  Logs: `/private/tmp/chuck-merged-{go,ui,embed,build,race}-20261014.log`. No browser,
  live-provider or physical-phone journeys were rerun for these fixes; previously owed gates
  remain below. Closure creates no new review unit.

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


## Review findings


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


## Settled closure moved from the live handoff — 2026-10-14

- **Closed — phone connection recovery (2026-10-14):** FS-20.R51–R53/A25, TS-13.R27–R29.
  One owner in `ui/src/remote/connection.ts`: ping-fed 25 s deadline, serialized health+stream
  attempts, 5–60 s backoff, hidden pause; initial page stays `checking` until hydration.
  `ReplaceForm` unkeyed; chat re-anchors after refresh. `connection.test.ts` (13, fake clock),
  `phone-render.mjs --recovery-check` (fails without the anchor fix), full UI suite (known
  `NewAgentModal` flake only), `make test`, embed and build pass. Device gates owed below.
  Review (`45dfe9a4`, `ca1fd120`) findings closed: `pause` now moves `connected` to
  `reconnecting` when it closes a hydrated source, so page-cache restore keeps actions gated
  until the replacement hydrates (TS-13.R28/R30 updated; new delayed-health regression fails
  without the fix; 14 connection tests). The harness finding was false: `phone-render.mjs`
  already sets exit code 1 when the conversation is missing (verified with a deliberately
  missing agent: screenshot written, exit 1); no change. `--recovery-check`, `make test`, embed,
  build pass; full UI suite fails only the `NewAgentModal` flake.
  Changelog: phone heartbeat recovery (INV §1/§4/§5/§16); recovery review (INV §1/§4/§8/§17);
  page-cache action gate (INV §1/§4/§8); harness exit finding false (INV §17, no change).

## Settled at the `v0.16.0` release cut — 2026-10-14

- **Release before this cut:** `v0.15.1` published 2026-10-14 (release CI run 38513998402).
  Pushed tag `v0.15.0` has no GitHub Release (CI run 38512698302 failed on the CLI journey test);
  `v0.14.0` stays local-only by user decision. `origin` URL corrected 2026-10-14.
- **Closed — operator CLI journey hermeticity:** `TestOperatorBuiltBinaryRealServerJourney` points
  `CLAUDE_CODE_EXECUTABLE`/`CODEX_PATH` at inert stand-ins; failed before with no provider CLIs on
  `PATH`, passes after.
- **Closed — think tank non-leaver filler:** FS-21.R16/R20/R52–R53/A38, TS-14.R5. Discussion-turn
  `pass` disposition; quiet round ends with `nothing_further`. `TestThinkTankPassAndQuietRotation`
  and room-page marker test. Not run through a live provider.
- **Closed — image-frame CI timeout:** large-frame runtime tests use `drainTurnWithin` 20 s;
  `onNotification` routes via one `peekSessionUpdate` (25 MiB frame decode 1.17 s → 0.71 s).
- **Closed — phone model and effort split** (FS-20.R43, TS-08.R111, `b1bc5284`) and **mobile
  wallpaper restoration** (`33987801`): reviews found no findings.
- **Closed — phone scroll memory:** FS-20.R54/A26, TS-08.R125 (`e522eb1a`, `f5f86dde`). Restore
  re-applies saved `scrollY` after live-update reload; `phone-render.mjs --scroll-check` covers it.
- **Closed — agent page width and side panels:** FS-12.R61/A33, TS-08.R112 (`7a0228e5`,
  `30204eaf`, `06d6f177`). `chat-cleanup-render.mjs --panels` regression added.
- **Closed — project rooms at the bottom, collapsible:** FS-02.R71/A53, FS-05.R39, FS-21.R44,
  TS-08.R96 (`4440e79e`, `f61a5280`).
- **Closed — conversation-only archive search:** FS-05.R41/A24 (`9627fa96`); no review findings.
- **Release notes:** [`RELEASE-v0.16.0-notes.md`](RELEASE-v0.16.0-notes.md).
