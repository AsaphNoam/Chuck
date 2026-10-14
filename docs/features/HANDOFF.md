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
- **Closed — chat permission modes, project-page menu, hidden file changes (2026-10-14):**
  `bac0d40`, `7cd50ff` plus fixes `a27f8d2`, `96d828f`. Empty legacy replies preserve provider
  updates, including unchanged values; project background menus exclude buttons and nested
  content. Actual-wire mode and header-control regressions failed before their fixes and pass
  afterward. Changelog: restored effective-mode reporting and form-control exclusions
  (INV §8/§10/§11/§12). Integration verification is in progress.
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
- **Reviewed — chat images and attachments; fix pending (2026-10-13):**
  [`chat-images-and-file-attachments.md`](../ready-changes/chat-images-and-file-attachments.md)
  is implemented in the attachment commits beginning `91ae7db`, `e8c50f7`, and `395c1a8`, plus
  the final attachment closure commit. The unrelated CLI design commit `5fd573c` is excluded.
  FS-03.R81–R86/A62–A65; TS-02.R46–R47, TS-03.R62–R63,
  TS-04.R90–R91, TS-05.R26, TS-08.R123 and TS-13.R26 cover desktop/phone picker/drop/paste,
  thumbnails/modal, native images, other files through copied host paths and ordinary tools,
  browser drafts, immutable local history and compatible multipart Send/Steer with retry receipts.
  Changelog: immutable local copies, bounded multipart admission, native image input, ordinary
  file references, durable retry receipts, explicit held recovery and shared browser drafts;
  assistant/local image rendering and keyboard-accessible inspection across history and device
  boundaries (INV §1/§2/§4/§5/§7/§8/§9/§10/§11/§12/§13/§14/§15/§16/§17).
  UI verification passes 805 tests in 88 files. Rendered chat/dashboard/archive and phone checks
  pass, including portrait containment, modal focus, reload and newer-draft recovery. Embedded
  server desktop and authenticated TLS test-phone journeys pass; no physical-phone/tailnet claim.
  Pinned Claude 0.85.1/2.1.286 and Codex 2.1.1/0.159.3 live image, copied-file and attachment-Steer
  receipts pass; `/private/tmp/chuck-live-attachment-evidence` retains evidence and the first
  Codex tiny-image interpretation mismatch. Browser receipts: `/private/tmp/attachment-render`
  and `/private/tmp/chuck-attachment-browser-receipts`. Final `make test` (both Go variants),
  `make dist`, spec/launcher checks and diff checks pass; the embedded UI is refreshed.
  Independent review confirms two Must-fix items: a corrupt retained blob blocks subsequent
  attachment delivery globally after restart, and failed conversation-byte cleanup removes
  durable identity before leaving retained bytes behind. One Worth-fixing item: local-image
  Escape dismissal bypasses object-URL/request cleanup. Findings and evidence below; the same
  unit stays open. Changelog: identified recovery amplification and incomplete deletion/modal
  teardown (INV §4/§7/§15/§16). Other review units are unchanged.
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


### Chat images and file attachments — `e8c50f7`, `395c1a8`, `9ae6f59` — **Fix model:** medium — Codex Terra or Claude Opus.

Reviewed independently 2026-10-13 against FS-03.R81–R86/A62–A65, TS-02.R46–R47,
TS-03.R62–R63, TS-04.R90–R91, TS-05.R26, TS-08.R123 and TS-13.R26.
`91ae7db` starts the administrative state; unrelated CLI design `5fd573c`, later CLI/UI
units and concurrent dashboard/diff-display work are excluded.

- **Must fix** — one corrupt retained blob disables attachment delivery globally after restart
  (INV §7). Locations: `internal/attachment/store.go:238`,
  `internal/server/attachment_recovery.go:74`, `internal/server/attachments.go:454`.
  Startup enumerates durable ownership, then `PruneUnowned` calls `Retain` on each owned
  descriptor. An integrity error or missing blob within a retained attachment directory
  propagates through `recoverAttachmentStorage` into the global `attachmentStartupErr`.
  Every subsequent attachment-bearing Send/Steer receives 500, including valid new uploads;
  text-only conversations remain usable. A temporary actual-server overlay probe retains a
  receipt-owned blob, corrupts it, runs startup recovery and confirms a new valid multipart
  upload is rejected. TS-02.R47 requires missing/corrupt blobs to remain per-item errors;
  FS-03.R85 requires surrounding conversation to remain usable. Preserve the bad item's
  ownership, report its unavailability, and isolate recovery failure so healthy conversations
  and uploads proceed. Add a restart regression with a corrupt owned item and a healthy
  second conversation, checking failed item reads and successful new delivery; do not delete
  retained bytes merely because ownership enumeration or validation fails.

- **Must fix** — failed conversation-byte deletion loses durable ownership before cleanup can
  be retried (INV §4/§15). Location: `internal/state/agents.go:229–234`.
  `DeleteAgent` commits the identity deletion and cascades receipt rows before attempting
  `DeleteConversation`. A filesystem cleanup error returns after the identity is already
  absent. An independent probe retains a blob, makes its own attachment directory read-only
  (`0500`), calls deletion, and observes both `ReadAgent` returning not found and the exact
  original blob remaining readable at its original path. Restoring permissions allows a
  direct internal retry, but no durable identity/receipt schedules or exposes that retry;
  startup preserves retained copies without a pending marker. This affects the owning
  deletion/failed-launch rollback seam; there is currently no public session DELETE route.
  TS-02.R47 and FS-03.R84/A64 require conversation deletion to remove its owned copies.
  Keep durable ownership until cleanup succeeds, or durably retain the cleanup obligation;
  add an injected cleanup-failure regression proving retryability and eventual byte removal.
  Successful deletion must still leave source/project files untouched.

- **Worth fixing** — Escape leaves a local-image fetch/object URL owned by a closed modal
  (INV §4/§16). Location: `ui/src/components/chat/AttachmentView.tsx:178`.
  Both Escape handlers call `setOpen(false)` directly, bypassing `setDialogOpen(false)`'s
  generation invalidation, request abort, object-URL revocation and source reset. A loaded
  local image remains retained while its hidden component is mounted; a pending read may
  complete after dismissal. TS-05.R26 requires owned object URLs to be released on applicable
  modal close. Independent isolated component probes observe zero revocations after Escape
  and one after the Close button. Route both Escape paths through the existing close helper
  and add loaded-image and in-flight-request regressions. No new mechanism is needed.

Specifications otherwise cover the shipped attachment behavior. Sound local choices include
opaque fixed-path ownership, bounded immutable uploads, shared multipart/owned-copy admission,
durable request receipts, negotiated native images versus ordinary uploaded-file reads,
shared draft revisions and retention, and the existing dialog/Markdown seams. Separate image
and Steer capability gates preserve Send. Prepared recovery remains distinct from ambiguous
post-wire outcomes. Historical runtime switching does not resend images. Clone copies have
independent ownership: the apparent runtime-only rollback gap is covered by server
`DeleteAgent`; a final applied-config write failure retains identity and copied transcript,
so it is not an unowned-copy finding. Attachment-only Stop recovery is covered by the
authoritative held/recovery refetch; the legacy text truthiness check alone is not a
confirmed lost-attachment finding.

Verification: 100 focused UI tests across Composer, attachments, TranscriptView, AttachmentView
and phone AgentScreen pass; 46 presentation/style checks pass. Six independent real Chromium
fixture journeys cover desktop 1024px in Core, Sky & Grove and Studio and Core phone
360/390/430px. They verify aspect-preserving previews, wide/tall modal containment, keyboard
focus, all three dismissals, preserved draft text, corrupt-image Retry, inert remote image
references and local reads only after activation. Desktop chat, expanded pane and archive
screenshots retain the quiet strip/primary composer and existing visual language. Harness and
settled receipts: `/private/tmp/review-attachments-render.mjs` and
`/private/tmp/review-attachments-render/{receipt.json,*.png}`; the loopback/Chromium run used
authorized execution after the sandbox refused binding. These fixtures make no live-provider,
physical-phone or real-tailnet claim. Existing pinned-provider receipts and their independent
image/file/Steer oracles were inspected separately, including the retained tiny-image mismatch.
Fresh `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 go test -count=1 ./internal/attachment
./internal/state ./internal/runtime ./internal/transcript ./internal/server` passes all five
packages (`/private/tmp/review-attachments-go.log`), including real fake-provider wire and
server failure/security journeys. Spec, launcher/review-contract and diff checks pass.
Failure probes: `/private/tmp/chuck_attachment_repro_test.go`,
`/private/tmp/chuck-overlay.json`, `/private/tmp/review-attachments-repro.log`;
`env -u CHUCK_RUNTIME_ROOT GOCACHE=/private/tmp/chuck-gocache go test
-overlay=/private/tmp/chuck-overlay.json -count=1 ./internal/server
-run 'TestRepro(CorruptRetainedBlocksRecovery|DeletePermissionFailureLeavesOriginalBytes)'`
passes by asserting the observed failures. Local-modal probes and output:
`/private/tmp/review-localimage-escape.test.tsx`,
`/private/tmp/review-localimage-escape.output.txt` (two passing observations of Escape/Close).

Invariant sweep: applicable §1, §2, §5, §8–§14 and §17 have no additional findings;
§7 has recovery amplification, §4/§15 have deletion ownership ordering and §4/§16 have
local-image teardown. §3 and §6 have no applicable changed surface: no seeded config form or
one-shot persisted configuration field, and no new runtime/interface/driver. Review changes
only this state file; every other role/review unit and concurrent product edits remain unchanged.

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
