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
- **Closed — mobile CSS missing in dev mode (2026-10-12):** dev-only phone asset graph refresh,
  404s for missing `/assets/` paths, and service-worker invalid-cache recovery implemented. The
  reproductions were confirmed failing before the fix, then unskipped and passed; five worker
  regressions, 800 UI tests, `make test` (both Go variants), `make build`, and `make dist` pass.
  On-device cache recovery remains unverified. Changelog: fixed derived asset state, error amplification, and dev wiring
  drift (INV §1/§7/§10).
- **Closed mobile outage investigation — 2026-10-12:** the user confirmed that an unrelated
  terminal crash stopped Chuck. The separate embedded tailnet device going offline while the
  Mac stayed connected is expected. Investigation `e0489b5` is closed at the user's request;
  incidental timeout/diagnostic findings are withdrawn from the fix queue, not implemented,
  and the skipped reproduction test is removed. No product or runtime changes.
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
- **Reviewed — phone chat drafts and Theme selection, fix open:** `79cbbc3`,
  FS-20.R49–R50/A20–A21, TS-08.R122, TS-13.R25. The 2026-10-12 review reproduced accepted
  Send cleanup missing a composer reopened while delivery was pending (see Review findings).
  Theme selection and the narrow paired-phone config route passed review. This same unit remains
  open for the draft fix; the other available review units are unchanged.
- **Design ready — full operator CLI parity (2026-10-12):**
  [`full-operator-cli-parity.md`](../ready-changes/full-operator-cli-parity.md) waits to start.
  FS-22.R1–R12/A1–A9, TS-01.R38, TS-03.R64–R69 and TS-06.R34 specify full nonvisual web
  operation coverage, legacy compatibility, readable/JSON output, file/stdin input, explicit
  consent and bounded reconnect-safe transcript follow. Scope and CLI contract are approved.
  Attachment A8 waits for the separate in-progress attachment unit; other families may start
  independently. Changelog: planned boundary recovery, shared client, lifecycle cleanup, honest
  results, consent and finite independent verification (INV §1/§2/§4/§7/§8/§9/§10/§11/§14/§15/§16/§17).
  Spec lint, design-feature/UX launcher twin comparisons and diff checks pass.
  No CLI product code; the active attachment change remains separate.
- **Design ready — chat images and attachments (2026-10-12):**
  [`chat-images-and-file-attachments.md`](../ready-changes/chat-images-and-file-attachments.md)
  waits to start. FS-03.R81–R86/A62–A65; TS-02.R46–R47, TS-03.R62–R63,
  TS-04.R90–R91, TS-05.R26, TS-08.R123 and TS-13.R26 cover desktop/phone picker/drop/paste,
  thumbnails/modal, native images, other files through copied host paths and ordinary tools,
  browser drafts, immutable local history and compatible multipart Send/Steer with retry receipts.
  Product and technical choices approved; additive API judgment delegated by the operator.
  Changelog: planned ownership/cleanup, bounded admission and honest delivery across history,
  runtime and device boundaries (INV §4/§8/§10/§11/§14/§15/§16). No product code; live provider
  receipts remain implementation closure gates, not evidence already obtained.
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

**In progress — chat images and file attachments:**
[`chat-images-and-file-attachments.md`](../ready-changes/chat-images-and-file-attachments.md).
FS-03.R81–R86/A62–A65 and the named attachment TS requirements remain planned until closure.
Storage, runtime and UI slices are delegated with distinct ownership; HTTP admission, durable
receipts and history/lifecycle integration remain under the parent. The working tree was clean
at selection. **Verified storage checkpoint:** immutable owner-only blobs/manifests, per-file
validation, safe reads, independent clone copies, provisional-marker startup cleanup and
conversation deletion; SQLite request receipts release held copies for explicit recovery after
restart. Full `internal/attachment` and `internal/state` tests pass. Limits apply per message,
not per conversation. **Verified runtime/API checkpoint (2026-10-13):** native image/file content
assembly, held/Steer outcomes, durable-before-wire acceptance, assistant/tool/child image events,
clone ownership, multipart and guarded binary reads pass focused integration tests. Full runtime
and transcript suites pass; focused runtime races pass. Migration 51 preserves checkpoint-50
databases and separates pre-wire `prepared` recovery from ambiguous post-wire `unknown` outcomes.
State tests pass, including the older minimal migration fixture and receipt preservation.
The full server run hit `TestPipelineOrchestratorInstructionsReachStandingAndDedicatedOnce`'s
409 Stop/resume race; its focused rerun passed. Final closure remains owed.
Codex 2.1.1/0.159.3 live image, copied-file and attachment-Steer receipts pass; fixtures/receipts:
`/private/tmp/chuck-live-attachment-evidence`. An initial tiny-image interpretation mismatch was
resolved with an independently verified larger fixture. Claude live checks are running with native
auth and isolated Chuck/project state. UI is uncommitted: focused checks pass, but rendered/source
validation found Escape dismissal, missing sent thumbnails and shared draft lifecycle gaps; those
repairs are in progress. Next: close rendered/reload/recovery checks, Claude gates and final matrix.

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

### Phone chat drafts and Theme selection — `79cbbc3` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Reviewed 2026-10-12; fix open.

- **Must fix** — Accepted Send leaves submitted text in a reopened composer (INV §1).
  `ui/src/remote/AgentScreen.tsx:207,413–423` restores the draft only on mount and clears local
  text only in the component that submitted it. Send, navigate away while delivery is pending,
  then reopen the same agent before acceptance: the reopened composer restores the submitted
  draft, but acceptance clears only storage and the old unmounted component. The visible text
  remains available to send again. This violates FS-03.R36 and FS-20.R49/A20's accepted-draft
  cleanup. Synchronize the mounted composer with accepted cleanup through the shared draft seam,
  preserving revision protection for newer text. Add a delayed Send/Steer leave/reopen/accept
  regression case. Independent Chromium reproduction retained `submitted before navigation`
  in the textbox after the storage entry became absent; evidence:
  `/tmp/review-phone-boundary-20261012.mjs` and `/tmp/review-phone-boundary-20261012.png`.

Review evidence: 66 focused UI tests passed (AgentScreen, PhoneSettings, PhoneApp and drafts);
the two focused remote appearance/field-filter server tests passed. Existing PhoneApp fixtures
emit missing `/api/roles` MSW warnings without failing. The working tree's actual remote entry
passed `phone-render.mjs --parity-check` with isolated Chromium at 360/390/430px in all three
appearances; inspected screenshots show a labelled, readable Theme control, adequate touch height
and no horizontal overflow. Receipts: `/tmp/review-phone-parity-20261012*.png`. Fixtures make no
live-provider or real-tailnet claim. Shared draft/config reuse, partial config merge, finite skin
selection and query ownership are sound local choices; none needs a human decision.

Invariant sweep: §1 has the finding above. Applicable §2, §3, §8, §10, §11, §13, §14, §15,
§16 and §17 have no additional findings. §4, §5, §6, §7, §9 and §12 have no applicable surface
in this unit. Product code and specifications were unchanged.

The mobile outage investigation was closed by the user on 2026-10-12 (see Current position).
Closed 2026-10-11 reviews are in the archive above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.14.0-notes.md`](../archive/state/RELEASE-v0.14.0-notes.md) (unpublished until
pushed) and [`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
