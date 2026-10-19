# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-18 is archived in
[`HANDOFF-through-2026-10-18`](../archive/state/HANDOFF-through-2026-10-18.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.17.0` published 2026-10-18 (Latest): release CI run 39100834882 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from the
  notes. Main CI run 39100833712 on the release commit failed once on
  `TestCanceledTurnsReleaseToolInput` (see Review findings). Notes:
  [`RELEASE-v0.17.0-notes.md`](../archive/state/RELEASE-v0.17.0-notes.md). Everything it settled
  is archived in [`HANDOFF-through-2026-10-18`](../archive/state/HANDOFF-through-2026-10-18.md).
  The shipped `operating-chuck` package now covers peer transcript reads, approved peer Stop,
  task reporting/`$name` skills, Think Tank goal attachments and judge instructions, quota
  reset/check recovery, the provider-handoff history bridge and restart-paused pipeline stages.
- **Smaller owed checks carried over:** no phone component test for the definite-refusal Retry
  rule; live Claude receipt
  for late tool input; rendered check of the withdraw failed-recovery-read tail notice.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Prompt/harness investigation:** the Review findings unit below records confirmed native Codex
  instruction loss (PROMPT-1), a prompt-composition diagnostic gap (PROMPT-2), and separate
  process-policy recommendations. General output-quality regression remains unproven.
- **Known risk (unchanged, uninvestigated):** runtime `SetSessionConfig` holds `as.mu` across its
  provider call while `onNotification` takes `as.mu` for unmatched frames on the synchronous read
  loop, so an adapter notification sent before that reply could stall it.
- **Design prepared/TBD — subscription quota views:** FS-09.R80–R81/A49–A50,
  TS-04.R89/TS-03.R61. Full passive paired snapshots remain TBD for Claude and Codex; evidence is
  in [`ideas.md`](../ideas.md). Not a ready unit. Do not implement a hidden prompt, command
  parser, login side client or new adapter patch.
- **Paused work:** `migrate-internal-actions-from-mcp.md` remains paused on its transport blocker.
- **Source-build Go tests:** run with `CHUCK_RUNTIME_ROOT` unset; an inherited installed bundle
  violates a credential test's source-build assumption. Prefer `GOFLAGS=-p=1` for full closure:
  the parallel image-fix run timed out `TestCodexProberAcceptsNativeLoginWithoutAPIKey` after 2 s;
  five isolated repeats and the full serial matrix passed without code changes.
- **Known flaky checks (uninvestigated):** `internal/server`
  `TestOrdinaryStageAgentStopPausesPipelineRun` intermittently returns 409 “a resume is already in
  progress”. `FileViewer.test.tsx`
  “scrolls to and marks a cited line” failed once in the full suite and passes alone.
  `internal/server` `TestQuotaDispatcherContinuesSameConversationOnce/stopped` failed once
  (“reloaded transcript differs”, FTS5 variant) in the `v0.16.0` closure run; 40 isolated runs,
  30 `-race` runs and a full FTS5 package rerun pass. Range commit `9989fb88` touched ACP update
  routing; whether it relates is uninvestigated.

## Active change

- **File reader formats and path copying** — user-requested JSON formatting, content-sized
  code backgrounds, PNG/JPEG/WebP/GIF previews and right-click Copy path (FS-03.R52/A77,
  TS-03.R40, TS-08 §2.14). Implementation, 45 focused UI tests, 981 browser checks plus 117
  after the reader-only styling adjustment, the presentation matrix and embedded build pass.
  Both Go closure variants, spec checks and embedded build pass. In progress only for full-suite
  closure: bounded UI concurrency passes 965/966 tests, with the unchanged-baseline assertion
  failure below. Approval for its separate one-line correction is pending; product implementation
  is committed in `8857beb3`. Independent source and rendered review completed 2026-10-19:
  no new functional defect; one pre-existing API-comment drift recorded below. Full-suite
  closure and its separate test correction remain open.

## Acceptance gates still owed

- **FS-16.A33 — task reporting under real provider policy:** run a credentialed sequential
  workflow (Claude `/work` → Codex `$review` → Claude `/fix` as tasks); each assignee must read
  its task and record `report_task_result` without supervisor prompting, and a late native
  background continuation must leave the card idle (FS-01.R52).

- **FS-20.A18 — phone group-header New agent:** independent phone-entry browser rendering
  passes, but the stubbed API journey does not prove accepted launches propagate to desktop and
  another phone. The two-device 390px journey for this entry point (including offline disable) is
  owed.

- **Mobile annotation parity independent rendered review (workflow §14.4):** the review's
  `phone-render.mjs --annotation-check` attempt failed to bind `::1:5199` with `EPERM`.
  Approval-capable independent browser execution remains required; the implementer's stubbed
  browser journey does not close this review gate.

- **FS-13.A19–A21 / FS-20.A27 (mobile annotation parity):** paired iPhone and Android native
  long-press selection inside events and tracked files, then Annotate, with the keyboard open
  over the tray (delivery/errors reachable). The stubbed `phone-render.mjs --annotation-check`
  journey uses programmatic ranges and a short viewport, so it cannot close these.

- **FS-03.A71–A75 (workflow §14.4):** independent real-browser review remains partial. The
  2026-10-19 formats review successfully ran the approved loopback harness independently:
  `file-reader-journey.mjs` passed 981/981 checks across live/archive at 1024/1440/2137px in all
  appearances, including slow Reload and the boundary regression sweep. The existing room viewer,
  dashboard/phone navigation and native transcript anchoring still need independent receipts.
  The fixer's earlier `file-reader-journey.mjs` (with a
  review-regression journey: boundary sweep, slow Reload, tab leave/return, agent change) passes
  963/963 post-fix, and its boundary sweep crashes the pre-fix stage (evidence in shared
  resources `roomier-file-reader-review-2026-10-17/render-postfix/`). Check live/archive/room
  surfaces, the dashboard overlay and unchanged phone/dashboard
  navigation across widths/appearances, including slow Reload, resizing across the ~992px
  docking boundary, late diagrams, nested launch failure/retry, focus return and transcript
  visible-event/follow continuity through width reflow (native anchoring is unproven there).

- **FS-06.A29 / TS-04.R94:** bounded credentialed Claude and Codex receipt that `stop_agent`
  waits for Approve once under each provider's most permissive mode. **FS-06.A30:** real-browser
  desktop and paired-phone peer Stop journeys (reload while pending, Deny, Approve, later Resume).

- **FS-01.A7/A8 / TS-02.R48:** live Claude/Codex cross-backend handoff → immediate Stop/wake,
  explicit model → Stop/Resume, restart, completed target turn and native compaction. Regression
  wire capture and fake compaction notice prove Chuck's bridge delivery/lifetime, not effective
  live-provider context adoption. OpenCode/OpenHands handoffs now refuse before teardown pending
  verified model/primer delivery; their existing FS-09.A6 gate remains owed.

- **FS-20.A22–A24 / TS-13.R30:** paired iPhone Home Screen app and Android browser
  background-return journey (stalled stream, outage, revoked pairing). Simulated/browser proof
  passes; the device suspension cause is still unreproduced.

- **FS-03.A67 / TS-04.R92:** live Claude/Codex permission-mode receipt. Claude 0.59.0's mode
  option is source-checked only; the Codex mode shape is unverified.

- **FS-01.A28/A31/A33/A34:** bounded packaged/installed Claude/Codex terminal-quota receipts:
  real reset text/`_claude/rateLimit` wire, actual continuation, and isolated checks proving
  same account/model, Claude non-persistence, Codex `/rename` + archive, the dotted
  `CODEX_CONFIG` `mcp_servers.<name>.enabled:false` overlay and no auto-title turn. Parsers follow
  pinned formatter source; fake-provider/browser proof is not live delivery evidence.
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

### File reader formats and path copying — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Implementation `8857beb3`; independent source/rendered review completed 2026-10-19 against
FS-03.R52/A77, TS-03.R40/R48 and TS-08 §2.14. No new functional or security defect found.
The unit remains open for the findings below; other review units are unchanged.

Verification: 37 focused UI tests pass (`FileViewer.test.tsx`, `webLinks.test.tsx`); focused Go
image/text/limit tests pass with `CHUCK_RUNTIME_ROOT` unset. Approved loopback Chromium execution
passes all 981 `file-reader-journey.mjs` checks across Core, Sky & Grove and Studio and
live/archive at 1024/1440/2137px. Six additional browser checks pass for document-relative Copy
path without a citation suffix, one composed selection menu, focused-menu containment/copy,
JSON citation marking and exact large numeric literals. Screenshots were independently inspected
for compact code, fitted images, formatted JSON and selection menus. New review evidence:
`/tmp/chuck-reader-review-20261019/{report.json,extra-report.json,*.png}`; its bounded extra harness
is `/tmp/chuck-reader-review-extra.mjs`. Browser API responses are stubbed; this does not close
the remaining real-device/room/native-anchoring gates. Earlier implementation evidence remains in
shared resources `file-reader-formats-2026-10-18/` and `file-reader-formats-final-core/`.

Local choices accepted: JSON formatting changes whitespace only, preserves numeric/escape
spelling, and falls back within the specified depth/output bounds; a cited JSON line stays in
Source, and Formatted has no annotation mapping. Images reuse attachment inspection and release
their owned Blob URLs. Copy path extends the existing offer/claim menu and clipboard-error helper.
Invariant sweep: applicable classes §1, §2, §4, §8, §10, §11, §13, §14, §16 and §17 checked;
§10 has the comment drift below. No applicable surface for §3, §5, §6, §7, §9, §12 or §15.

- **Worth fixing** — **pre-existing API-comment drift** (INV §10):
  `ui/src/api/client.ts:373–374` still says file reads are confined to the session working
  directory and cites the superseded containment contract. An engineer using this comment when
  extending the reader would infer a security boundary the route deliberately removed under
  TS-03.R48 / FS-03.R64. Update the comment to say relative paths resolve from the recorded cwd,
  while absolute paths use local process authority; the JSON-only response claim remains valid.
  Comment-only correction; verify against the current route and R48. No runtime change needed.

- **Worth fixing** — **confirmed pre-existing closure blocker** (INV §17):
  `ui/src/components/chat/TranscriptView.test.tsx:349` asserts that no text matches
  `/could not be loaded/` after withdrawal recovery; the unrelated Think Tank synthesis failure
  also matches, so the full UI suite fails (965 passing, one failure with two workers). The same test fails on an
  unchanged `6ee5f646` archive. Scope the assertion to the exact withdrawal recovery error while
  preserving the successful-recovery and single-withdraw assertions. No product behavior change
  is needed. User approval for this unrelated one-line test repair is pending; no edit made.

- **Worth fixing** — **intermittent test failure, cause uninvestigated** (INV §17):
  `ui/src/remote/AgentScreen.test.tsx` “recovers two successive live gaps without losing the
  loaded history” failed once in the default full UI run on “Gap question 4700”; it passes
  alone and in the full two-worker run. No phone gap code was changed. Diagnose timing or fixture
  isolation before changing production behavior or weakening the test.

### Compact Think Tank chat navigation — commit “think tank: compact chat-tab navigation to room and members” — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Awaiting review.

FS-21.R57/A40, TS-08.R126 (TS-08 now Current). `ThinkTankTab` in `RoomTurnNotice.tsx` renders
one `<nav>` per room: room link, then other roster members as `ad-button ad-button-secondary`
router links (`data-state` active/settled; active = running `active_attempts`, aria-label adds
“taking a room turn”); deleted members are “(unavailable)” text; duplicate names add project.
Adopted an interrupted uncommitted implementation; removed its translateY and busy-colour label.
`ThinkTankTab.test.tsx` covers speaker change/clear on refetch. Closure: UI 975/975 (including
the file-reader withdrawal assertion, which passed this run), `GOFLAGS=-p=1 make test`,
`make build`, UI build. 42 Chromium checks pass at 1024/1600/360px-pane in Core, Sky & Grove and
Studio; screenshots inspected. Evidence: `/tmp/chuck-ttnav-review-20261019/`; harness
`/tmp/chuck-ttnav-review.mjs` (stubbed API; needs `delete window.SharedWorker`).

### Phone groups, Jump to latest, runtime buttons — commit “phone: collapsed groups, Jump to latest, Send-shaped runtime buttons” — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Implementation `fe572fcc`; independent source and rendered review completed 2026-10-19 against
FS-20.R43/R47/R54, A15/A19/A26 and TS-08.R73/R121/R125. The unit remains open for PHONE-1 below.

Verification: 87 focused tests pass (`PhoneApp`, `ProjectScreen`, `AgentScreen`), along with
style/presentation checks. Approved loopback Chromium execution of the actual phone entry with
stubbed API data verifies collapsed named/Ungrouped defaults, retained project-scoped expansion,
first-open newest position, scroll-away/Jump/hide, 44px targets and no horizontal overflow at
360/390/430px in Core, Sky & Grove and Studio. Development presentation matrix renders in all
three appearances at 1024px. Representative screenshots independently inspected for group
hierarchy, compact runtime surfaces and the floating action. Evidence and the failing live-growth
receipt: `/tmp/chuck-phone-polish-review-20261019/` (`report.json`, `stream-growth.png`,
`stream-growth-after-scroll.png`, width/theme screenshots); harness:
`/tmp/chuck-phone-polish-review.mjs` (use `--growth-only` for the focused reproduction). This does
not claim paired-device, native keyboard or live-provider acceptance.

Local choices accepted: reuse the existing bounded page-session group store (256 identities),
existing scroll saver and 48px newest threshold; key the agent screen per id; share runtime/jump
inset styling without a new component or persistence mechanism. No new motion.
Invariant sweep: applicable §1, §2, §4, §8, §10, §13, §16 and §17 checked; §8 has PHONE-1 below.
No applicable surface for §3, §5, §6, §7, §9, §11, §12, §14 or §15.

- **Must fix** — PHONE-1 — **Jump to latest misses live transcript growth** (INV §8):
  `ui/src/remote/AgentScreen.tsx:508–523` updates `awayFromLatest` only on a window scroll event.
  A normal incoming reply can grow a short conversation beyond the viewport without moving
  `scrollY`, leaving the person far from the newest message with no Jump action (FS-20.R54).
  Independent browser reproduction: a new-message notification/refetched reply grows document
  height from 844px to 6324px at `scrollY=0`, while the button stays absent; scrolling just 1px
  makes it appear. Recompute visibility after relevant transcript/layout changes as well as
  scrolling, preserving reading-position restoration. Add a regression that receives a reply
  without dispatching a synthetic scroll, checks the button appears, and verifies jumping hides it.
  This is a localized visibility-state fix at the existing scroll seam.

### Agent-created task group placement — commit “tasks: let create_task place its launched agent in a group” — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Implementation `2b7dcb24`; independent source review completed 2026-10-19 against FS-16.R50/A34
and TS-02.R51 (migration 55). New-group launch, trimming, reserved-label refusal and refusal for
an existing target work through the existing task/launch seams. The unit remains open for TG-1
and TG-2 below. Live Chucky receipt (asks for a new group, agent lands in it) remains owed; fake
runtime and tool tests do not claim real-provider adoption.

Verification: `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 go test ./internal/state ./internal/messaging`
passes, as do the focused server tests `TestAnAgentCreatedLaunchTaskPlacesItsAgentInANewGroup`
and `TestAgentCreatedTaskChecksItsLaunchSpecification`. Loopback escalation was needed for the
HTTP test listeners. A temporary Go overlay test, without repository edits, confirms a 64 KiB
group is accepted and reread unchanged from SQLite; source:
`/tmp/chuck-group-review.itGomY/task_http_test.go`.

Local choices accepted: reuse `normalizeGroup`, `insertTaskRowTx`, the durable task launch inputs
and the existing dispatcher/launch request; leave HTTP authoring and Tasks controls unchanged.
Invariant sweep: applicable §1, §2, §3, §7, §8, §9, §10, §11, §14, §15, §16 and §17 checked;
§8/§16 have TG-1 and §10/§11 have TG-2. No applicable surface for §4, §5, §6, §12 or §13.

- **Worth fixing** — TG-1 — **The new group input has no size bound** (INV §8/§16):
  `internal/server/task_handlers.go:742–748` accepts `create_task.group` through
  `normalizeGroup` (`internal/server/groups.go:41–49`), which only trims and rejects the reserved
  label. A tool call containing an oversized label stores it intact in `tasks.group_name`; the
  dispatcher then copies it into the agent identity published to dashboard clients and group
  headings. The temporary test confirms acceptance and durable round-trip of 65,536 characters.
  Existing launch paths share this validation gap, but this change adds an agent-authored ingress.
  Give group labels a documented bound at the shared validator, reject over-limit values with the
  typed validation outcome before task insertion, and test the limit and no-write refusal.

- **Worth fixing** — TG-2 — **HTTP read shape contradicts the specified field exclusion**
  (FS-16.R50; INV §10/§11): `internal/state/tasks.go:130–132` serializes `Group` as `group`, and
  `internal/server/task_handlers.go:332–335,349–358` embeds that task in detail/list responses.
  Reading a grouped task therefore adds `group` to the task HTTP API, despite R50 saying that
  API does not gain the field. HTTP creation and Tasks controls are correctly unchanged. Clarify
  R50 to exclude only HTTP authoring if the read field is intentional, or omit the field from
  the HTTP response projection; cover the chosen serialized contract with a focused test.

### Late tool input CI flake — `bbc8c89c` — **Fix model:** medium — Codex Terra or Claude Opus.

- **Worth fixing** — CI-1: `TestCanceledTurnsReleaseToolInput` flakes in CI (TS-04.R98; INV §17). Main
  CI run 39100833712 failed at `internal/runtime/tool_input_test.go:37` (“turn 2: open inputs =
  0, want the live call”) after the turn-1 cancel; 60/60 local repeats pass. It matters because
  it either reddens main CI or hides a late turn-1 `finishTurn` dropping a live turn-2 call.
  Uninvestigated whether the test injects its frame before turn 2's `setExecTurn` release or the
  product ordering is wrong; route through `/investigate-bug`, then fix with a deterministic
  ordering test.

The `NewAgentModal`/phone gap-test race fix is confirmed remotely (main CI run 39007164597).

Closed reviews through the `v0.17.0` cut are in the archive above.

### Prompt/harness performance investigation — `f54ce47a` — **Fix model:** medium — Codex Terra or Claude Opus.

Report received 2026-10-18, verbatim:

> I have a suspicion that the chuck system prompts and other harnesses are deteriorating agent performance. This shows up in two forms
> 1. Generally worse output
> 2. Overly instructs every task
>
> The former is harder to judge, the latter is likely easier to just look at the system prompts and see if they encourage this kind of behavior. Look into this and get back to me regarding both matters and what can be done about it

Inspected source at `f54ce47a` after the `v0.17.0` release. The report supplies no affected
backend/persona, installed version, representative task/transcript, comparative outputs or logs;
none were captured from live user sessions. General quality regression and the specific intended
meaning of "overly instructs" remain **undetermined**. Findings below separate proven instruction
loss from plausible behavioral effects.

- **Must fix** — PROMPT-1 — **confirmed**: Codex loses native `config.toml`
  `developer_instructions` when Chuck adds its prompt. `internal/runtime/chat.go:149–155,2565–2601`
  preserves only a prior `CODEX_CONFIG.developer_instructions`; it does not preserve the field
  from the native config mirrored by `internal/config/codexprofile.go:36–55`. Normal trigger: a
  user saves developer guidance in native configuration and launches/resumes/switches a Codex
  Chuck agent. Even No persona or awareness off normally still adds the shared-resources prompt,
  making the override nonempty. The 2.1.1 adapter spreads the overlay into `thread/start.config`
  (`scripts/release/node_modules/@agentclientprotocol/codex-acp/dist/index.js:34087–34091,34187–34206`).
  An isolated 0.159.3 app-server check used separate temporary profiles, an ephemeral thread and a
  loopback mock Responses endpoint, with no credentials or real inference: the baseline request
  contained `NATIVE_CONFIG_SENTINEL`; the same native config plus the adapter-equivalent
  `thread/start.config.developer_instructions` contained only `CHUCK_OVERLAY_SENTINEL`.
  This is loss of configured additional instructions, not proof of native base-prompt loss or of
  the reported quality decline. FS-09.R44/A17 and FS-18.R17/R20/A13 require native setup/guidance
  preservation; TS-04.R14/R69 currently specify only the environment-overlay merge, leaving the
  native-config precedence case uncovered (INV §11–§12/§17). Preserve the effective native value,
  including applicable project/profile precedence, before adding Chuck guidance; keep the current
  malformed-overlay refusal and unrelated-key preservation. Do not grow an independent TOML
  resolver from a guessed subset of provider config. Reproduction:
  `internal/runtime/native_prompt_investigation_test.go`, observed failing on current code and
  committed skipped under this finding. Extend the provider-boundary check with native config,
  environment overlay, empty prompt, resume, and refreshed-profile cases when fixing.
- **Worth fixing** — PROMPT-2 — **confirmed diagnostic gap**, behavioral causality
  **undetermined** (no invariant class): `internal/runtime/chat.go:2387–2418` records/hashes only frozen
  `SystemPrompt`; the deliberately runtime-only operating suffix and installed knowledge/tool
  revisions are not identified in that prompt fingerprint. Consequently an archived task cannot
  establish which complete Chuck instruction set accompanied it, or prove provider adoption from
  that hash. This limits diagnosis of this report across releases/resumes (FS-18.R8/R17,
  TS-11.R5/R15/R17; no existing requirement for a complete diagnostic manifest). Add a bounded
  diagnostic manifest of requested layer identities/hashes/sizes, source and provider versions,
  distinguishing requested delivery from verified adoption; preserve frozen snapshot semantics and
  avoid automatic raw prompt/config logging. Pair it with the already-owed FS-18.A12/A13 evidence.

Other conclusions and recommendations (no additional confirmed product defect):

**Confirmed historical defect, already fixed:** before `12a76571` (2026-10-03), Claude chat sent
  a string `_meta.systemPrompt`, replacing Claude Code's coding preset. Current
  `claudeSystemPrompt` sends `{type:preset,preset:claude_code,append:...}` for both new/load;
  the pinned 0.85.1 adapter forwards it additively. Claude terminal uses
  `--append-system-prompt`; Codex uses additional developer instructions, not a base-prompt
  replacement. Fresh/resumed live adoption remains the explicitly open FS-18.A13 gate. Existing
  frozen persona snapshots/customizations are not refreshed by role migration; a fresh conversation
  is needed to compare current defaults without old conversation effects.
**Works as specified; probable pressure toward excess process:** current seeded personas are
  approximately 100–115 words each (`internal/config/seed.go:143–169`); Chucky explicitly asks for
  bounded assignments/context/completion criteria. Shared context/pointer
  (`internal/server/launch.go:426–451`) adds capability/assignment/reporting instructions, with no
  per-turn repetition. The 348-word operating skill is loaded on demand, not inlined; its four
  references total about 4,096 words, also not all automatically loaded. No source evidence of an
  automatic complete-manual injection or duplicated overlay accumulation was found. These rules
  may influence process/style, but word count alone does not establish degraded reasoning.
Repository-specific harness pressure is stronger and independent of the Chuck persona:
  `AGENTS.md:7–19,85–95` mandates role preflight and an explicit delegation assessment;
  `.agents/skills/work/SKILL.md:8–16` requires workflow sections in full; workflow §5–§6 requires
  state commits and a fixed human-update shape; TS-06.R5 applies the full Go closure matrix to
  every product-code change. This is intentional current policy, not an accidental runtime bug.
  A separate policy simplification should scope preflight/testing/reporting to task consequence,
  keep project-specific invariants, and let a small task remain small. Delegated briefs should state
  outcome, needed context, constraints and completion evidence without prescribing routine methods.
Recommend a small repeated, blind comparison on representative question, small-fix, review and
  coordination tasks: native provider; Chuck with an empty custom persona and awareness off; same
  persona with awareness on; relevant seeded persona. Hold executable/model/effort, repository
  guidance, tools where possible, task and fresh checkout fixed; compare correctness, scope creep,
  unnecessary calls/delegation, elapsed work and answer/brief length. No persona omits only persona
  text; awareness off omits only the two shared prompt additions, retaining tool registration,
  installed skills, project/resource/task instructions and history. These are not clean native
  baselines. Repeat a second comparison with repository workflow guidance reduced to isolate that
  layer. FS-18.A12 is qualitative role acceptance, not a general quality benchmark.
Official OpenAI guidance supports this audit direction, but is not evidence of a Chuck-specific
  regression: [Rethinking skills and prompts for GPT-6 Astra](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra)
  recommends narrow skill triggers, progressive disclosure, contextual document reads and revisiting
  procedural scaffolding. Improvements remain hypotheses until measured on the affected workload.

Verification: config migration/no-polling, embedded progressive-reference, Claude additive-preset,
Codex environment-overlay, and server knowledge/awareness lifecycle tests pass. Server tests required
local loopback permission after the sandbox rejected `httptest`'s listener. The new regression
failed for the missing native instruction before being skipped; the separate pinned-native mock
capture proves actual provider precedence. Spec lint, formatting and diff checks pass. Product code
and specifications are unchanged.

### No persona permission prompts investigation — `1303b36e` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Report received 2026-10-18, verbatim:

> no persona keeps asking for permissions, even though my global Claude /codex settings are auto approval

No logs or version supplied; inspected source at `1303b36e` and the reporter's local
`~/.chuck`, `~/.claude/settings.json` and `~/.codex/config.toml` (read-only).

**Works as specified (core behavior), confirmed from code and local config:** Chuck's permission
policy is its own `skip_permissions`, not the provider's native setting (FS-04.R1/R10/R55). No
persona is the canonical `default` role with `skip_permissions: null`, which inherits global
`config.json` `skip_permissions`; the reporter's is `false` (seed default,
`internal/config/seed.go:33`). `resolveSkip` (`internal/server/launch.go:684`) therefore freezes
`false`, and the runtime gate (`internal/runtime/permission.go:46`) forwards every adapter
permission request to the operator. The reporter's other five personas each carry
`skip_permissions: true`, explaining why only No persona prompts. Native settings are not
"auto" either: `~/.claude/settings.json` has no `permissions`/`defaultMode`, and Codex has
`approval_policy = "on-request"` with `approvals_reviewer = "auto_review"` (mirrored into the
Chuck-owned `CODEX_HOME`, but approval requests still arrive at Chuck's gate). Whether
codex-acp honours `approvals_reviewer` is **undetermined** and irrelevant once Chuck's gate applies.

- **Worth fixing** — PERM-1 — **confirmed spec gap** (no invariant class): the inherited global policy has no
  first-party control. `PUT /api/config` accepts `skip_permissions`
  (`internal/server/config_handlers.go:83`), but no Settings surface edits it, and FS-04.R57
  hides the canonical Default from persona management, so No persona's effective policy can only
  be changed by hand-editing `config.json`. The role form's **Inherit global** option
  (`ui/src/features/settings/RoleForm.tsx:104`) names a setting the user cannot see and reads as
  the provider CLI's global settings. Normal trigger: any user who wants No persona agents to
  auto-approve. Fix: specify and add a Settings control for the global Chuck permission policy
  (FS-04.R10), label the role option by that control's name, and cover it with a Settings
  round-trip test. Workaround now: set `"skip_permissions": true` in `~/.chuck/config.json`
  and restart Chuck (no live config reload); it affects new launches only (TS-01.R6). No reproduction test: behavior is as coded.

## Changelog

- **2026-10-19 — Compact Think Tank chat navigation:** the agent-chat Think Tank tab is now a
  compact link row per room, with active speakers slightly bolder and raised; queued for review.

- **2026-10-19 — Reviewed phone header slimmed; unit closed:** implementation `caf4a008`
  matches FS-20.R43 and TS-08.R111/R134; no findings. **Fix model:** trivial/easy — Claude
  Sonnet or Codex Luna (no open fixes). Independent source review, 16 `PhoneApp` tests,
  46 style/presentation checks and 66 approved loopback Chromium checks pass. Actual phone entry
  at 360/390/430px in Core, Sky & Grove and Studio verifies transparent/unframed header,
  Home/This phone navigation, agent-route omission, back navigation and the pending-annotation
  pill sticking at 8px and reopening its sheet. Representative screenshots inspected; development
  matrix renders at 1024px in all appearances. Evidence:
  `/tmp/chuck-header-review-20261019/{report.json,*.png}`; harness:
  `/tmp/chuck-header-review.mjs`. API data is stubbed; no native-device acceptance claim.
  Local choices accepted: route-match reuse, existing agent back chevron and local sticky spacing;
  no new mechanism or motion. Invariant sweep: §8, §10, §13 and §17 checked; no applicable surface
  for §1–§7, §9, §11–§12 or §14–§16. Other units and acceptance gates are unchanged.

- **2026-10-19 — Reviewed agent-created task group placement:** normal launch/refusal paths and
  focused checks pass. Recorded the unbounded group input (TG-1, INV §8/§16) and the HTTP read
  contract/spec mismatch (TG-2, INV §10/§11); both are worth fixing. Unit remains open.

- **2026-10-19 — Reviewed phone groups, Jump to latest and runtime buttons:** focused tests and
  the three-appearance phone-width browser checks pass; recorded PHONE-1 because live reply growth
  leaves Jump hidden until the person scrolls (INV §8). Unit remains open.

- **2026-10-19 — Reviewed file reader formats and path copying:** no new functional defect;
  37 focused UI tests, focused server image/text tests and 987 browser checks pass. Recorded
  pre-existing API-comment drift (INV §10); existing full-suite blockers remain open.

- **2026-10-18 — Fixed breadcrumb sibling-agent switching; unit closed:** each agent page now
  remounts per id so tab and staged runtime start fresh (BREAD-1, INV §1); reduced motion stops
  busy dots (BREAD-2, INV §8); the `+n` menu right-aligns inside the viewport (BREAD-3, INV §8);
  the live check identifies the current agent by id (BREAD-4, INV §17). FS-03.A76, FS-12.A40 and
  TS-08.R138 extended. Three new `ChatPanel` tests and the new render checks fail pre-fix;
  970 UI tests, `breadcrumb-render.mjs` and the real-server `breadcrumb-live-check.mjs` pass
  (evidence: shared resources `breadcrumb-fix-2026-10-18/`).

- **2026-10-18 — Phone polish:** groups start collapsed, chats gain Jump to latest, and the
  model/effort buttons match Send's shape.

- **2026-10-18 — File reader formats and path copying:** JSON Formatted/Source with literal
  preservation, content-sized code backgrounds, validated PNG/JPEG/WebP/GIF previews and Copy
  path. Implementation, focused tests, browser checks, both Go variants and build pass;
  full UI closure is limited by the unchanged transcript assertion above.

- **2026-10-18 — Breadcrumb sibling-agent switching:** the agent page's breadcrumb lists the
  project's running agents as one-click mini cards with a `+n` menu.

- **2026-10-18 — Phone header slimmed:** no header on agent chats; elsewhere just menu, logo and
  “Chuck” over the background.

- **2026-10-18 — Designed breadcrumb sibling-agent switching:** ready change
  `breadcrumb-sibling-agent-switching.md` (FS-03.R98/A76, FS-12.R70/A40, TS-08.R137–R138, all
  planned); FS-12 is now Partial. Same-project, running-only, one-click mini cards in the
  agent-page breadcrumb; no server change.

- **2026-10-18 — Chucky group placement:** `create_task` can name a (new or existing) project group
  for the agent it launches; operating guidance tells agents to use it instead of the CLI.

- **2026-10-18 — Investigated prompt/harness performance:** confirmed native Codex configured
  instruction loss (PROMPT-1), recorded the diagnostic gap (PROMPT-2), separated the historical
  Claude replacement defect and current process-policy hypotheses from unproven quality effects,
  and preserved a skipped reproduction. Fix unit is above; native role/adoption gates remain owed.
- **2026-10-18 — Release `v0.17.0`:** refreshed the shipped `operating-chuck` quota-recovery,
  resume/switch-history and pipeline-interruption guidance; `env -u CHUCK_RUNTIME_ROOT
  GOFLAGS=-p=1 make test`, 944 UI tests and `make dist VERSION=0.17.0` (embedded UI refreshed)
  pass. Handoff epoch archived.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.17.0-notes.md`](../archive/state/RELEASE-v0.17.0-notes.md),
[`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
