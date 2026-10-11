# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-10 is archived in
[`HANDOFF-through-2026-10-10`](../archive/state/HANDOFF-through-2026-10-10.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.13.0` tagged at `b3ccae0` and published on 2026-10-10 after ROOM-WIDTH-01
  was fixed. User's “go ahead and release” waives pending reviews for this release and authorizes
  pushing all unpushed main commits and the tag. Independent reviews remain available below;
  waiver is not completed review. Both Go variants, go vet, FTS5 build proof, shell syntax and
  `make dist VERSION=0.13.0` passed. Operator package and component pins already match the range.
  README install/update examples now use the renamed repository. 758 UI tests and 41
  style/presentation checks passed.
  Release run `38073052316` and general CI `38073052277` succeeded. Archive, installer and
  checksum manifest are uploaded; manifest size (323869523) and SHA-256 match GitHub's
  archive metadata. Readable notes are published. Manual provider gates below remain owed.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Awaiting review — phone group memory and long-press agent actions:** `17b73ed`,
  FS-20.R47–R48/A19, TS-08.R121. Finished 2026-10-11. Page-session, project-scoped group choices and
  a long-press action sheet over shared phone management forms are implemented.
  789 UI tests, 41 presentation checks, refreshed embed and tagged build pass.
  Both Go variants pass. Nine fixture-driven native-touch Chromium journeys at 360/390/430px
  in all three appearances pass
  (retained collapse, held-press click suppression, refused rename draft, focus and overflow).
  Receipt/harness: shared project resources `mobile-actions-20261011/`; screenshots
  `/tmp/chuck-mobile-actions/`. Run source-build Go tests with `CHUCK_RUNTIME_ROOT`
  unset: the inherited installed bundle violates a credential test's source-build assumption.
  Existing real-device/provider gates remain separate; these fixtures make no provider claim.
- **Review complete — desktop/phone group controls:** FS-02.R72–R76/A54–A57,
  FS-20.R45–R46/A17–A18, TS-03.R56–R58, TS-08.R118, TS-13.R24.
  Range `4a7899b` through `1ec40f4` closed without originating findings on 2026-10-11.
  Source/server comparison and 33 original-snapshot authenticated browser checks passed.
  The later quota continuation Archive publication regression QUOTA-04 is now fixed.
- **Reviewed; fixes verified/closed 2026-10-11 — button redesign:** `0a4bfb9`, FS-12.R68/A38,
  TS-08.R119. Independent specification/code and rendered comparison completed on 2026-10-11;
  BUTTON-01/02 are fixed and verified.
  Review receipts and screenshots: `/tmp/chuck-button-review-20261011/`.
- **Reviewed; fixes verified/closed 2026-10-11 — Think Tank room page:** `815a57e`, FS-12.R64/A36,
  TS-08.R115. Independent specification/code and rendered comparison completed on 2026-10-11,
  including paused/held failure/pipeline, Files viewer and focus. ROOM-PAGE-01/02 are fixed and
  verified. Review receipts and
  screenshots: `/tmp/chuck-think-tank-review-20261011/`.
- **Reviewed; fixes verified/closed 2026-10-11 — mobile details/composer:** `58b63d1`,
  FS-20.R37/R43, TS-08.R111. Independent specification/code and rendered checks completed on
  2026-10-11. Inline preview, compact Manage, unboxed footer and Send touch target passed;
  MOBILE-01 is fixed and verified. Receipts and screenshots:
  `/tmp/chuck-mobile-review-20261011/`.
- **Reviewed; fixes verified/closed 2026-10-11 — No persona and optional persona awareness:**
  [`agents-without-a-persona-by-default.md`](../ready-changes/agents-without-a-persona-by-default.md)
  is Finished. Substantive range `d29a4f5` through `2de7167`. FS-01.R46–R47/A29–A30,
  FS-04.R54–R57/A34–A36, FS-18.R20/A16, TS-02.R43–R44, TS-03.R59, TS-08.R120,
  TS-11.R21–R22. Five-role seed/once-only preference upgrade, Advanced persona choice,
  hidden canonical Default, editor-only awareness and frozen lifecycle/metadata/reindex choice.
  Both Go variants, focused race checks, 771 UI tests, 41 presentation checks and make dist pass.
  88 real Chromium checks pass in all three appearances at 1024px and phone 360/390/430px;
  desktop/phone/onboarding launch/edit keyboard journeys and development matrix were inspected.
  [Verification receipt](../archive/state/PERSONA-DEFAULT-2026-10-10.md); raw screenshots/receipt
  `/tmp/chuck-persona-journey-final3/`. Fake providers make no model-adoption claim; existing
  credentialed provider gates below remain owed. Independent specification/code comparison
  completed on 2026-10-11; PERSONA-01 is fixed and verified.
- **Reviewed; fixes verified/closed 2026-10-11 — quota continuation:**
  [`continue-chats-after-quota-reset.md`](../ready-changes/continue-chats-after-quota-reset.md)
  is Finished. Substantive range `9409787` through `8011691`: FS-01.R40–R45/FS-04.R53,
  TS-02.R45/TS-03.R60, TS-04.R86–R88/TS-10.R39–R44 and the named owner exceptions.
  Default-on policy, exact terminal classification, bounded owner-preserving recovery and receipts,
  explicit Cancel targets, desktop/phone indicators and manual controls are implemented.
  Both Go variants, go vet, focused runtime/state/server races, 782 UI tests, 41 presentation
  checks and tagged build/embed pass. 41 Chromium checks cover three-skin desktop/phone layout,
  stopped unknown-reset indication/settings/cancellation and exact-once notice reload/Archive parity.
  [Verification receipt](../archive/state/QUOTA-CONTINUATION-2026-10-11.md).
  Current providers have no verified reset-unit receipt, so observed interruptions remain unknown
  and unscheduled. Known-reset recovery proof uses state fixtures; FS-01.A28 remains owed.
  Independent specification/code comparison completed on 2026-10-11. QUOTA-01/04 publication
  fixes passed focused HTTP tests, including live/reconnect truth, mixed group members and
  save-again recovery after publication failure. QUOTA-02 is fixed: target agent/owner reservations
  keep unrelated Stop/group actions available during startup. QUOTA-03 is fixed: live/stopped
  notice indexing and receipt confirmation commit together; stopped repair reconciles durable
  transcript counters, including legacy crash orders, and publishes one repaired notice.
  Final both Go variants, tagged build, go vet and focused quota/lifecycle/index races passed.
  No findings remain in this originating unit; FS-01.A28 remains the live-provider gate.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61 prepare Claude and Codex 5-hour/Weekly views. Chat-header disclosure confirmed.
  Full passive paired snapshots/reads remain TBD for both providers: Claude only has partial live
  metadata; full `/usage` and Codex `/status` occupy the provider turn queue and return text.
  Verified gaps and source evidence are in [`ideas.md`](../ideas.md)'s design entry/ACP Wait-list.
  Live Claude percentage/timestamp units also need a verified producer/provider receipt; bare
  numeric declarations are not unit evidence. The full view is not a ready implementation unit.
  Do not implement a hidden prompt, command parser, login side client or new adapter patch.
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Known flaky check:** `internal/server` `TestOrdinaryStageAgentStopPausesPipelineRun`
  intermittently returns 409 “a resume is already in progress” (pre-existing at `74c8e84`);
  synchronization remains separate work.

## Active change

In progress — mobile composer and New agent placement (FS-20.R43, TS-08.R111).
New agent alone uses PhoneSheet's centered variant. The composer encloses top-right model/effort,
borderless textarea and bottom-right Send; scoped textbox styling removes the blue typing outline.
Existing shared button, management, inline-file and other-sheet behavior is preserved.
48 focused phone tests, all 789 UI tests, 41 presentation/style checks and `make dist` pass.
`phone-render.mjs --details-check` verifies all three appearances, centered dialog geometry at
360/390/430px, metadata position, Send containment and absent typing outline. Screenshots:
`/tmp/mobile-composer-20261011-*.png`. Both Go variants are running before final closure.
Previously recorded manual/provider acceptance gates remain owed below.

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

## Review findings

### Desktop/phone group controls — `4a7899b` through `1ec40f4`, reviewed and closed 2026-10-11. **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Independent server audit and root frontend/integration review compared the group requirements in
both directions. No confirmed defect originates in this unit; no group fix is required.
Current-tree focused server group,
identity and remote-route tests passed; six focused UI files passed all 120 tests, and all 41
presentation checks passed. The UI run emitted existing unmatched `/api/roles` fixture warnings;
group work adds no roles request. Current-tree Chromium verified keyboard Create selection,
typing without assignment and refused-launch draft retention in all three appearances, and
rendered the 1024px development matrix. Long labels, section collapse, picker focus/stacking,
confirmation focus and result feedback were independently inspected.

Current-tree authenticated Archive exposed QUOTA-04, introduced by `30c6aea` after this unit.
To establish attribution, exported `1ec40f4` to an isolated temporary checkout, generated its UI
with Vite and `make embed -o ui`, and ran its own loopback-only fake-provider server fixture.
The temporary shared-node_modules symlink caused a TypeScript inferred-declaration path error;
Vite generation and the original snapshot's 41 presentation checks passed. This is not a new
TypeScript closure claim. Its authenticated desktop/phone browser journey passed all 33 checks:
Core, Sky & Grove and Studio; desktop 1024/1440px and phone 360/390/430px; collapsed-header
cross-status drag, bidirectional membership, Cancel focus, cross-project isolation, successful
live Archive removal, individual Restore, legacy-route absence, no overflow or page exceptions.
Receipts/screenshots: `/tmp/chuck-group-original-render-20261011/`; current-tree supplemental
keyboard/matrix evidence: `/tmp/chuck-group-review-20261011/extra/`. Current-tree Archive gate
remains failed under QUOTA-04, rather than being represented by the passing historical receipt.

Invariant sweep: §1–§2, §4–§5, §7–§11 and §13–§17 matched and were checked. §3 has no one-shot
runtime/config field or seeded-config form change; group is durable identity. §6 has no new
interface/runtime/driver; §12 has no changed external-CLI invocation. Shared picker/projection,
member stop/archive services, existing claims, four-worker request lifetime, JSON label transport,
pointer collision and unchanged flat manual order are sound local choices. No unresolved local
choice or product decision. Product code and specifications were unchanged.

### No persona and optional persona awareness — `d29a4f5` through `2de7167`, reviewed 2026-10-11. **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Review verification: focused config/state/index/CLI persona and reindex tests passed in untagged
and `sqlite_fts5` builds; focused server/runtime awareness, API, overlay and clone tests passed
with bounded approved loopback execution. Frozen lifecycle/clone race checks passed. Six focused
UI files passed all 59 tests. `make embed` passed TypeScript/build and all 41 presentation checks.
Fresh Chromium launch/edit/onboarding journeys passed all 85 checks with zero page errors in
Core, Sky & Grove and Studio at desktop 1024px and phone 360/390/430px. Screenshots of compact/
Advanced launch, editor and onboarding states were inspected; no additional material visual
finding. Receipt and screenshots: `/tmp/chuck-persona-review-20261011/receipt.json`.
An independent phone probe reproduced PERSONA-01 after four failed config requests, with no
visible error and disabled Create, then verified explicit Advanced selection enables Create:
`/tmp/chuck-persona-review-20261011/config-failure-receipt.json` and `phone-config-failure.png`;
probe source `/tmp/chuck-persona-review-failure.mjs`. The isolated home/fake provider proves host
behavior, not live native-provider prompt adoption; existing credentialed gates remain owed.

Invariant sweep: §1–§5, §7–§11, §13–§15 and §17 matched and were checked. §6 has no new interface/
runtime/driver; §12 has no changed external-CLI invocation; §16 has no new stream, queue, sweep,
fan-out or retained-state admission surface. No additional confirmed finding. Local choices are
sound: private once-only completion uses the existing atomic writer and records completion last;
occupied customized Default definitions preserve ownership and block automatic selection;
awareness uses one normalized SQLite/metadata value and Clone's internal option under its existing
claim; the existing overlay gates both shared prompt blocks while preserving tools/knowledge
availability. Canonical presentation uses the shared predicate and keeps customized same-id roles
named. No unresolved implementation-local choice. Product code and specifications were unchanged.

### Quota continuation — `9409787` through `8011691`, reviewed and closed 2026-10-11. **Fix model:** medium — Codex Terra or Claude Opus.

All findings are fixed and verified. FS-01.A28 remains owed; fixture proof is not a live-provider
reset/delivery receipt. Fix commits continue the originating unit and create no new review unit.

Fix checkpoint 2026-10-11: QUOTA-01/04 closed (INV §1/§2/§8/§10). Global disable publishes
committed cancellations with an idempotent save-again recovery path; group Archive publishes all
successful members once after quota cancellation. Both new live/reconnect regressions failed
before the fix; focused config/archive checks and state cancellation tests pass. Both full Go
variants, focused state/server race checks and make dist passed at final closure.

QUOTA-02 checkpoint 2026-10-11 (INV §4/§5/§10/§16/§17): manual controls reserve only target
agents/owners, automatic admission includes pipeline parents, and global disable/admission retains
policy ordering. `TestStopDuringWakeConflictsAndKeepsRegistration` failed against the old server
via `/tmp/quota02-before-overlay.json` and passes after the fix; unrelated individual Stop and group
Stop/Archive succeed while same-agent Stop conflicts and registration survives. Focused quota,
lifecycle and owner-deferral tests pass.

QUOTA-03 closure 2026-10-11 (INV §2/§5/§7/§9/§10/§15/§17): the shared indexer commits ordinary
session counters and the quota notice receipt in one transaction after transcript sync. Stopped
repair holds the lifecycle claim through durable counter reconciliation, so an unindexed notice,
an old live index-before-receipt crash, and a later indexed event all retain exact metadata.
Success and progress receipt transitions serialize; publication follows a newly committed receipt,
including an existing pending notice, once. Stopped-metadata and legacy double-count regressions
failed before their corrections.
`TestQuotaNoticeRepairIndexesOnceAcrossAppendCrash`, failed-index/retry publication, and live/stopped
dispatcher metadata checks pass. Final `make test` (both Go variants), `make build`, `make vet`, and
focused server/state/runtime/index quota, lifecycle and index-boundary races pass. UI source was
unchanged by these fixes; concurrent mobile work is excluded from the commits.

Review verification: focused server quota/concurrent-quota suite passed with bounded approved
loopback execution (`GOCACHE=/tmp/chuck-go-cache go test ./internal/server -run
'TestQuota|TestConcurrentQuota' -count=1`). Independent state package and race runs passed; focused
runtime quota/session-failure tests passed. Three isolated Go overlay probes reproduced QUOTA-01,
QUOTA-02 and QUOTA-03 without editing repository tests; probe sources/overlay are
`/tmp/quota_review_probe_test.go` and `/tmp/quota_review_overlay.json`. Fresh current-tree Chromium
rendering passed all 23 checks at 1024px desktop and 390px phone in Core, Sky & Grove and Studio;
screenshots inspected, no retained visual finding. Receipt:
`/tmp/chuck-quota-render-review-20261011/receipt.json`. The phone fixture proves layout, not
authenticated successful quota cancellation. FS-01.A28 remains the previously declared live-provider
gate; no provider-generated known-reset delivery claim was made.

Invariant sweep: classes §1–§5 and §7–§17 matched the diff and were checked; the findings above
identify their matching classes. §6 has no applicable new runtime/interface/driver surface.
Existing lifecycle/registration and external-CLI seams remain in use. Provider classification,
durable ownership/CAS/restart/notice claims, bounded admission, explicit Cancel client wiring,
local/remote guards, and parsed/styled UI surfaces had no additional confirmed findings. No
unresolved implementation-local choice was found. Product code and specifications were unchanged.

### Think Tank room page — `815a57e`, reviewed 2026-10-11. **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Review verification: independent source/specification audit and root integration/render review
checked FS-12.R64/A36, TS-08.R115 and the retained FS-21 room contracts in both directions.
All 25 focused ThinkTankPage tests and all 41 style/presentation checks passed. Approved execution
ran the working tree's embedded production server on loopback with a newly isolated fixture home
and deterministic fake ACP. Chromium inspected all three appearances at 1024/1440px: concurrent
hidden openings, a live turn with held input, paused, failed-turn, ended/synthesis/departed,
long-title/goal, pipeline and failed-judge states. Representative state/read payloads were
injected at the browser boundary; actual message and turn-limit mutations used the real server.
Files dialogs opened the existing viewer with the originating source preserved; Commands listed
attribution. Layout/state/read-only/reduced-motion checks passed without horizontal page overflow.
The main run records 130 passing checks and the 12 focus failures above. Eleven supplemental
checks passed, including entry annotation/excerpt capture, exact opening retry identity,
refused-retry error feedback, modal focus trapping, the focus-loss consequence, source-error
observation, and actual room deletion/navigation while preserving participant sessions. These
observation checks prove the recorded failures; they do not close them. No product-page exception
was observed. The initial reference host needed automatic JSX and a React Router 7-to-6 Link
import adaptation before it rendered; these temporary-host repairs changed no product code.

Retrieved the Figma Make study's ThinkTank component/styles and rendered them in a temporary
minimal host for comparison. Inspected the reference and representative product screenshots;
composition, density, attribution, speaker tints, recovery state salience, goal disclosure and
viewer layering conform apart from the findings above. The development matrix rendered at both
widths in all appearances. Current button construction comes from later `0a4bfb9`; its retry
variant changes are not findings against this unit. No live-provider delivery/reset/synthesis
claim is made, and the existing credentialed gates remain owed.
Receipts/screenshots: `/tmp/chuck-think-tank-review-20261011/`; probe sources:
`/tmp/chuck-think-tank-review-20261011.mjs` and
`/tmp/chuck-think-tank-review-extra-20261011.mjs`.

Invariant sweep: §1/§2/§8/§10/§11/§13/§16/§17 matched and were checked. §3 has no persisted
runtime field or seeded-config write; §4 has no registration artifact; §5 has no concurrent
claim; §6 has no new interface/runtime; §7 has no record repair/recovery iteration; §9 has no
durability/liveness primitive; §12 has no external-CLI invocation; §14 has no request trust or
filesystem-security change; §15 has no external-side-effect ordering change. Existing keyed
room state, shared room vocabulary, sourced file loading, bounded query windows, canonical
annotation/rendering seams and feature-owned semantic palette are sound local choices. The new
dialog's incomplete focus convention and false source-count fallback are rejected above. No
unresolved product decision. Product code and specifications were unchanged.

### Button redesign — `0a4bfb9`, reviewed 2026-10-11. **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Review verification: 782 UI tests, all 41 style/presentation checks and the production TypeScript/
Vite build passed; built entry asset names match the embedded UI. Independently inspected the live
Figma `/study/buttons` reference and current-tree Chromium surfaces in Core, Sky & Grove and Studio.
The repository button sweep reported zero errors/legacy-style findings; a supplemental isolated
fake-provider run covered 57 room, confirmation, launch, onboarding, phone pipeline, pairing and
Settings states, with no page errors, horizontal overflow or phone buttons under 44px. Nine actual
fake-provider permission-card render checks passed at 1024/1440px desktop and 390px phone across
all appearances (`live-permission.json`); phone permission targets measured at least 44px. Twelve
keyboard-focus, primary/destructive-hover and reduced-motion checks passed. Development matrix
and representative room fixtures rendered at 1024/1440px; screenshots were inspected. Receipts,
interaction checks and screenshots are under `/tmp/chuck-button-review-20261011/`; supplemental
probe sources are `/tmp/chuck-button-review-extra.mjs`, `/tmp/chuck-button-review-states.mjs` and
`/tmp/chuck-button-live-permission.mjs`.
The fixture owns temporary CHUCK_HOME state and loopback listeners; no live-provider claim or
production-state mutation was made. Existing credentialed gates remain owed.

Invariant sweep: §2/§3/§8/§10/§13/§17 matched and were checked. §1 has no derived-state transition;
§4–§6 no registration/lifecycle/concurrent claim/new runtime; §7 no repair/read-recovery change;
§9 no durability/liveness change; §11 no wire payload change; §12 no external-CLI invocation
change; §14 no request trust/security boundary change; §15 no side-effect ordering change;
§16 no production admission, retained-state, stream or fan-out change. The shared semantic CSS
construction, public variant update, retired-token removal, navigation selected cues and preserved
event handlers are sound. No unresolved implementation-local choice. The partial sweep is a
diagnostic report, not sole A38 evidence; supplemental routes and report inspection were required.
Product code and specifications were unchanged. Pre-existing persona/quota finding heading and
severity-marker formatting was normalized to the existing handoff checker contract.

### Mobile details/composer — `58b63d1`, reviewed 2026-10-11. **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Review verification: a focused independent source comparison and root integration review checked
FS-20.R37/R43, TS-08.R111 and FS-12's applicable presentation rules in both directions. The
previous handoff's FS-20.R32/TS-08.R116 pointers named diff annotation and desktop cards rather
than this unit; the pointers above are corrected. All 34 AgentScreen tests and all 41
style/presentation checks passed. The focused test run emitted existing unmatched `/api/roles`
fixture warnings; this unit adds no roles request. Current-tree Chromium checks covered Core,
Sky & Grove and Studio at 360/390/430px: inline preview loading, row switching and Close, long
paths/content, compact Manage, refused rename with preserved draft, offline action disabling,
unboxed footer and outlined Send with a visible focus ring and a 44×44px target. The failed-file
state rendered its read error in the selected row. All checked states had no page errors or
horizontal page overflow; action heights remained at least 44px. The 1024px development matrix
rendered in each appearance. Representative screenshots were independently inspected. Evidence:
`/tmp/chuck-mobile-review-20261011/receipt.json`, the supplemental hover receipt and screenshots;
probe source `/tmp/chuck-mobile-review.mjs`. The bounded harness used a loopback-only Vite server
and mocked API/event input against the real phone entry. It made no live-provider, pairing or
production-state claim. No product code or specification changed.

Invariant sweep: §8/§10/§13/§17 matched and were checked. §1 has no lifecycle/derived-state
transition change; §2 no second builder; §3 no persisted/seeded field change; §4 no registration
artifact change; §5 no concurrent claim; §6 no new runtime/interface; §7 no record repair or
recovery change; §9 no durability/liveness change; §11 no wire contract change; §12 no external-CLI
change; §14 no request trust/security change (the existing tracked-file query and server limits
remain in place); §15 no side-effect ordering change; §16 no new admission or retained-state
mechanism. The existing single selected-path query is sound. The feature-local green secondary
choice is rejected as MOBILE-01; compact geometry and inline row placement are sound.

## Changelog

- 2026-10-11: closed QUOTA-02 (INV §4/§5/§10/§16/§17) and QUOTA-03
  (INV §2/§5/§7/§9/§10/§15/§17). Unrelated Stop/group controls stay available during manual
  provider startup; repaired notices retain exact session metadata and publish once after a
  successful index/receipt commit. Both Go variants, tagged build, vet and focused races pass.
  The quota continuation unit is closed; FS-01.A28 remains owed.
- 2026-10-11: closed PERSONA-01 (INV §8/§10), ROOM-PAGE-01/02 (INV §8/§10), BUTTON-01/02
  (INV §2/§8/§10/§13), MOBILE-01 (INV §8/§10/§13), and QUOTA-01/04 (INV §1/§2/§8/§10).
  Both Go variants, focused quota state/server race checks, 786 UI tests, 41 presentation
  checks and make dist passed. Chromium passed 21 room checks at 1024px and 15 phone checks
  at 390px across Core, Sky & Grove and Studio; representative screenshots were inspected.
  Receipts: `/tmp/chuck-trivial-room-fix-check/receipt.json` and
  `/tmp/chuck-trivial-phone-fix-check/receipt.json`. These fixtures use actual UI with mocked
  API data; they make no live-provider claim. QUOTA-02/03 were closed in the subsequent fix above.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
Publication verified; settled pre-release evidence is in the handoff archive.
