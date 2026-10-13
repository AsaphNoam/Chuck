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
- **Closed — blank phone entry on local dev (2026-10-13):** rebuilt the current dev server
  as `/private/tmp/chuck-remote-recovery` and gracefully restarted the pre-fix process;
  PID 18990 serves loopback port 4317 and the existing tailnet address. Live Chromium
  phone-emulation checks show Android pairing and iPhone browser Home Screen instructions;
  all requested JavaScript, CSS and font assets return 200 with no page errors.
  Receipt/screenshots: `/private/tmp/chuck-remote-recovery-{receipt.json,Android.png,iPhone.png}`.
  The fix was already in `0de23db`; no product code/specification change or new review unit.
  Physical-phone reload/pairing remains the operator's next step. Other review units unchanged.
  Changelog: loaded the existing derived-asset refresh fix into the live dev process (INV §1/§10).
- **Closed — mobile CSS missing in dev mode (2026-10-12):** dev-only phone asset graph refresh,
  404s for missing `/assets/` paths, and service-worker invalid-cache recovery implemented. The
  reproductions were confirmed failing before the fix, then unskipped and passed; five worker
  regressions, 800 UI tests, `make test` (both Go variants), `make build`, and `make dist` pass.
  On-device cache recovery remains unverified. Changelog: fixed derived asset state, error amplification, and dev wiring
  drift (INV §1/§7/§10).
- **Reviewed — projects home redesign; fix pending (2026-10-13):** `40e414e`,
  FS-12.R69/A39, TS-08.R124. One Must-fix: unavailable-project cards omit their known running
  count from the required footer. The rest of the composition and existing interactions pass
  22 focused dashboard tests, 46 presentation/style checks and six independent Chromium journeys
  at 1024/1440px in all three appearances. Finding, evidence and fix recommendation below;
  this unit remains open. Other review units are unchanged.
  Changelog: identified incomplete unavailable-project footers (INV §10).
- **Reviewed — mobile agent header, composer and model switch; fix pending (2026-10-13):**
  `c3b3c70`, `73cc75b`, `f15ee50`; FS-20.R43, TS-08.R111. One Must-fix: Back measures
  32×44px and model switch is 36px tall, below FS-12.R68/A38's 44px phone action target.
  Behavior and composition otherwise pass 41 AgentScreen tests, 46 presentation/style checks
  and nine independent Chromium journeys at 360/390/430px in all three appearances.
  Finding, verification and fix recommendation below; this unit remains open. Other review units
  are unchanged. Changelog: identified undersized phone action targets (INV §10).
- **Reviewed and closed — phone group memory and long-press agent actions (2026-10-13):**
  `17b73ed`, FS-20.R47–R48/A19, TS-08.R121. No findings; 14 focused phone tests and 46
  presentation/style checks pass. Nine independent Chromium journeys verify retained group
  choices, touch and keyboard actions, refusals, nested confirmation, stopped availability and
  restored focus at 360/390/430px in all three appearances. Review evidence below; fixtures
  make no provider or real-tailnet claim. Other review units remain unchanged.
- **Reviewed and closed — phone chat drafts and Theme selection (2026-10-13):** `79cbbc3`,
  FS-20.R49–R50/A20–A21, TS-08.R122, TS-13.R25. The 2026-10-12 review reproduced accepted
  Send cleanup missing a composer reopened while delivery was pending. Theme selection and the
  narrow paired-phone config route passed review. Independent text-only AgentScreen regressions
  now verify delayed Send and Steer leave/reopen/accept cleanup through the shared draft seam,
  including newer-draft protection. Changelog: synchronized accepted draft cleanup across
  reopened composers (INV §1). The other review units are unchanged.
- **Reviewed — full operator CLI parity; fix pending (2026-10-13):**
  [`full-operator-cli-parity.md`](../ready-changes/full-operator-cli-parity.md), implementation
  `f6e7b2b`, `ff031e4`, `219b4b7`; final closure record is administrative, not another review unit.
  FS-22.R1–R12/A1–A9, TS-01.R38, TS-03.R64–R69 and TS-06.R34 are implemented.
  All nonvisual command families, explicit JSON/file/stdin input, consent, attachments, status
  and bounded transcript follow are covered. Legacy launch/resume defaults are preserved;
  auto-resume now reads the existing project-scoped archive response.
  Independent wire and real-server built-binary fake-ACP journeys pass, including exact child
  permission IDs, attachment retry after response loss, stale guards and Ctrl-C/EPIPE.
  Focused observer/attachment race tests, binary help, `make build`, and final
  `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 make test` pass (both Go variants and spec checks).
  Changelog: shared client, explicit authority/consent, bounded recovery and lifecycle cleanup,
  honest results and independent verification (INV §1/§2/§4/§7/§8/§9/§10/§11/§14/§15/§16/§17).
  Independent review found one Must-fix: `role edit` help promises partial JSON, while the
  existing server replaces the full definition and clears omitted settings. The real-handler
  probe confirms prompt and explicit permission/awareness override loss. Fresh CLI tests,
  including the built-binary real-server journey, and focused observer/attachment/status race
  checks pass. Finding and fix recommendation below; this unit remains open.
  Changelog: identified misleading role-edit replacement guidance (INV §10).
  No provider/tailnet claim; the completed attachment review unit remains separate.
- **Awaiting review — chat images and attachments (2026-10-13):**
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

None. Full operator CLI parity is finished; independent review found misleading role-edit help,
recorded below, and its fix remains pending.
CLI extends existing session APIs without a runtime registration, new storage or lifecycle owner.
INV §6: persistence/full launch use server composition; fan-out/drain, mailbox eligibility, turn
boundaries, reconciliation and hooks retain server ownership; capabilities use server projections.
Observers close only their requests/streams, join bounded workers and remove failed output
temporaries. No new registration, mailbox or lifecycle store. Loopback test fixtures require
loopback-enabled execution permission and fail rather than silently skip when unavailable.

Chat images and file attachments is complete and available for independent review above.
Final closure: full attachment/state/runtime/transcript and focused API/security/race checks pass.
Forward migration 51 preserves checkpoint-50 receipts; pre-wire recovery remains separate from
ambiguous post-wire outcomes. Incoming output images obey 10-image/25-MiB decoded per-frame limits
before storage and bounded event fanout. Session reads and state snapshots share the same attachment
capability projection. Rendered findings in modal dismissal, sent thumbnails and shared draft
lifecycle were repaired and verified. The existing orchestrator-instructions test race reproduced
on baseline `5fd573c` 2/10 times; a test-only lifecycle-drain wait now passes 10/10.
Final shared matrix passes in `/private/tmp/chuck-attachments-closure-test.log`; final embedded
build passes in `/private/tmp/chuck-attachments-final-dist.log`. The preceding concurrent run's
two-second Codex credential fixture timeout did not reproduce in ten focused current or baseline
runs; the complete rerun without concurrent build load passes. Credential code is unchanged.

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

### Full operator CLI parity — `f6e7b2b`, `ff031e4`, `219b4b7` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Reviewed independently 2026-10-13 against FS-22.R1–R12/A1–A9, TS-01.R38,
TS-03.R64–R69 and TS-06.R34. Closure record `e90a15a` and intervening unrelated review,
handoff and UI commits are excluded. The attachment implementation remains a separate unit.

- **Must fix** — role-edit help promises a partial update that clears omitted settings
  (INV §10). Location: `internal/cli/operator_management.go:105` (submission at `:666`).
  `chuck role edit --help` describes "partial role JSON", so an operator changing only a title
  can reasonably submit `{"title":"Reviewer v2"}` with `--id reviewer --request-file`.
  The command forwards that object unchanged to the existing `PUT /api/roles/{role}` handler,
  which replaces the definition rather than merging it. A real-handler probe with a retained
  prompt and explicit `chuck_aware:false` / `skip_permissions:false` confirms a successful
  title-only update clears the prompt and resets both overrides to inheritance/defaults.
  This loses persona instructions and can change subsequent launch permissions. FS-22.R1/R3/R8
  require accurate operation help/input shapes and existing semantics; TS-03.R64 requires the
  existing request contract. Correct the help and example to describe a complete replacement,
  identify the omitted-field resets, and show how to retain existing values from `role show`.
  Do not introduce a client-side read/merge mutation as an incidental fix. Add a focused help
  regression and a real-server role-edit case covering complete replacement, retained explicit
  false settings and the documented reset behavior. No product or architecture decision is needed.

Code and specifications otherwise agree in both directions. The single REST/SSE client,
exact collection lookups, raw JSON/number preservation, server-owned launch and mutation semantics,
explicit guard/receipt forwarding, shared transcript projection and bounded temporary observer
tail are sound local choices. The CLI adds no runtime registration or lifecycle owner: persistence,
full launch composition, fan-out/drain, mailbox eligibility, turn boundaries, reconciliation and
hooks remain server-owned; capabilities use existing server projections. Observer teardown closes
only its own request/stream and joins its worker; file downloads publish only completed bounded
owner-only temporaries. Separate JSON/shortcut-field duplicates are rejected; ordinary repeated
scalar flags retain Cobra's established behavior, which the specifications do not prohibit.

Verification: fresh `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 go test -count=1 ./internal/cli`
passes, including independent family wire/input/output/consent tests and the built-binary
real-server fake-ACP conversation, management, pipeline/task/room, redaction, response-loss retry,
exact child permission, Ctrl-C and EPIPE journeys. Focused `go test -race -count=1` observer,
status and attachment checks pass. The initial sandboxed fresh run could not bind loopback;
both passing runs used authorized loopback-enabled execution. A temporary Go overlay probe
exercises the actual role handler and persisted definition without repository test/code edits:
`/private/tmp/review-cli-role-probe_test.go` and `/private/tmp/review-cli-role-overlay.json`.
The probe confirms the observed loss, rather than claiming the desired preservation passes.
Fixtures make no live-provider, physical-device or real-tailnet claim.

Invariant sweep: applicable §1, §2, §4, §6, §7, §8, §9, §11, §14, §15, §16 and §17 have
no findings; §10 has the help/behavior drift above. §3, §5, §12 and §13 have no applicable
changed surface: no seeded UI form or transient persisted runtime field, runtime check/claim,
new installed-tool invocation, or UI markup/style change. Product code and specifications were
unchanged. This finding keeps the same unit open; every other role/review unit is unchanged.

### Projects home redesign — `40e414e` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Reviewed independently 2026-10-13 against FS-12.R69/A39 and TS-08.R124.

- **Must fix** — **unavailable-project cards omit the running count (INV §10).**
  Location: `ui/src/features/dashboard/ProjectDashboard.tsx:151`. When a non-archived agent
  names a project missing from the configuration, the shared projection retains that agent in
  an unavailable-project card. The new footer renders its agent count but suppresses its running
  count with `!project.unavailable`. FS-12.R69 requires the ruled footer to show both counts,
  and A39 explicitly includes the unavailable state. The independent fixture's `ghost` card has
  one running agent but shows only `1 agent`, while the header includes that agent in its running
  total. This reproduces at both required widths in all three appearances. Render the already
  computed running count for unavailable cards too; add a focused regression with a missing
  project and running/stopped agents, asserting both footer values and the page total.

Other code and specifications agree in both directions. Keeping count derivation in the feature
component over `deriveDashboardProjects`, leaving the phone's distinct `stateSummary` contract
alone, using semantic state tones and the existing project-accent exception, and removing the
unused shadow token are sound local choices. No human decision is needed.

Verification: 21 `ProjectDashboard` tests plus one shared-projection test and 46 presentation/style
checks pass. The shipped `projects-render.mjs` fixture passes all six appearance/width combinations;
a separate Chromium harness exercises the working tree's actual UI in Core, Sky & Grove and Studio
at 1024px and 1440px. It verifies all seven live/stopped labels in attention order, archived-agent
exclusion, running totals and live running-to-stopped updates, long-title wrapping, long-branch
ellipsis without card/page overflow, right-click project/background menus, Rename/Archive/New
project dialogs and cancellation, keyboard focus feedback, and reduced-motion project navigation.
Settled screenshots and an independent visual critique confirm the footer finding and otherwise
readable header, cards, state tones and menus. The deterministic visual matrix was rendered in each
appearance at both widths; the removed token has no remaining source or embedded reference.
Fixtures make no provider or real-home claim. Evidence:
`/private/tmp/review-projects-20261013.mjs` and
`/private/tmp/chuck-projects-review/{receipt.json,*.png}`.

Invariant sweep: applicable §2, §4, §8, §13 and §17 have no findings; §10 has the finding above.
§1, §3, §5, §6, §7, §9, §11, §12, §14, §15 and §16 have no applicable surface in this unit.
Product code and specifications were unchanged. Only this unit's review state was updated.

### Mobile agent header, composer and model switch — `c3b3c70`, `73cc75b`, `f15ee50` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Reviewed independently 2026-10-13 against FS-20.R43, TS-08.R111 and their shared
FS-12.R68/A38 phone action sizing contract. Administrative checkpoints `c2266b2`/`470b53d`
and unrelated CLI design checkpoint `d3c9307` are excluded. Later draft and attachment changes
are separate units; rendered checks exercise the current working tree's actual remote entry.

- **Must fix** — Restore 44px phone hit targets (INV §10). In
  `ui/src/styles/features/phone.css:63`, `.phone-agent-title .phone-agent-back` sets width to
  32px; at `:121`, `.phone-model-switch` sets minimum height to 36px. Opening an agent on a
  phone and tapping Back or the model switch encounters targets below FS-12.R68/A38's required
  44px size, making these common actions harder to hit. Independent Chromium measurement
  confirms Back is 32×44px and model switch is 36px tall at 360/390/430px in Core, Sky & Grove
  and Studio. Preserve the specified compact chevron and raised model surface while extending
  their hit areas to at least 44px. Add rendered assertions for Back width/height and model
  switch height in the same nine-case matrix; existing header/composer geometry checks omit
  these dimensions. No product decision or spec change is required.

Code and specifications otherwise agree in both directions. Reusing detailed ContextBar,
AgentManagement's runtime action, the mutation/error seam, and PhoneSheet's opt-in centered
variant are sound local choices; none needs human input. Refused runtime switching retains the
chosen model and effort, successful switching closes the sheet without clearing the composer
draft, and focus returns to the trigger. Runtime-sheet bottom placement, centered New agent
placement with expanded fields contained in the viewport, live context updates, stopped
Resume availability, offline mutation disabling and Back navigation pass independent checks.
Inspected settled screenshots show the required header hierarchy, borderless typing surface,
Send inside the composer, right-aligned model control, readable refusals and no horizontal
overflow in all three appearances.

Verification: AgentScreen's 41 tests and 46 presentation/style checks pass. Nine independent
Chromium journeys pass the behavior and composition checks above while reporting the undersized
targets. Harness and receipts:
`/private/tmp/review-mobile-header-20261013.mjs` and
`/private/tmp/review-mobile-header-20261013/{receipt.json,*.png}`. Fixtures make no live-provider,
physical-device or real-tailnet claim.

Invariant sweep: applicable §1, §2, §8, §10, §13, §16 and §17 were checked; only §10 has the
finding above. §3–§7, §9, §11, §12, §14 and §15 have no applicable changed surface in this unit.
Product code and specifications are unchanged. The finding keeps this same unit open and
creates no second review obligation; every other role/review unit is unchanged.

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
