# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-18 is archived in
[`HANDOFF-through-2026-10-18`](../archive/state/HANDOFF-through-2026-10-18.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** `v0.17.0` published 2026-10-18 (Latest): release CI run 39100834882 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from the
  notes. Main CI run 39100833712 on the release commit failed once on
  `TestCanceledTurnsReleaseToolInput` (test-double flake, fixed 2026-10-19; see Changelog). Notes:
  [`RELEASE-v0.17.0-notes.md`](../archive/state/RELEASE-v0.17.0-notes.md). Everything it settled
  is archived in [`HANDOFF-through-2026-10-18`](../archive/state/HANDOFF-through-2026-10-18.md).
  The shipped `operating-chuck` package now covers peer transcript reads, approved peer Stop,
  task reporting/`$name` skills, Think Tank goal attachments and judge instructions, quota
  reset/check recovery, the provider-handoff history bridge and restart-paused pipeline stages.
- **Smaller owed checks carried over:** no phone component test for the definite-refusal Retry
  rule; live Claude receipt
  for late tool input; rendered check of the withdraw failed-recovery-read tail notice.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
- **Agent-surface review backlog:** 25 user-requested whole-subsystem review units for Codex and
  Claude are **waiting to start** under [Review queue](#review-queue). Select any one per `/review`;
  [AGENT-SURFACE-REVIEW.md](AGENT-SURFACE-REVIEW.md) defines its responsibility, source/spec entry
  points, failure searches and completion evidence. This was research/queue preparation only;
  none of the slices has been reviewed. Existing findings and live gates remain independent.
- **Prompt/harness investigation:** native Codex instruction loss (PROMPT-1) is fixed; the
  Review findings unit below still records a prompt-composition diagnostic gap (PROMPT-2) and
  separate process-policy recommendations. General output-quality regression remains unproven.
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

- None. File reader formats and path copying closed 2026-10-19 (see Changelog); its
  real-device/room/native-anchoring gates were not claimed by the stubbed browser review.

## Acceptance gates still owed

- **FS-16.A33 — task reporting under real provider policy:** run a credentialed sequential
  workflow (Claude `/work` → Codex `$review` → Claude `/fix` as tasks); each assignee must read
  its task and record `report_task_result` without supervisor prompting, and a late native
  background continuation must leave the card idle (FS-01.R52).

- **FS-20.A18 — phone group-header New agent:** independent phone-entry browser rendering
  passes, but the stubbed API journey does not prove accepted launches propagate to desktop and
  another phone. The two-device 390px journey for this entry point (including offline disable) is
  owed.

- **FS-16.A34 — live Chucky group placement:** ask Chucky to delegate work into a new group and
  confirm the launched agent lands in it; fake runtime and tool tests do not claim real-provider
  adoption.

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

## Review queue

These are explicitly requested subsystem audits, not reviews of the queue-preparation commit.
Review the selected item's full owned logic at the HEAD recorded when starting, including callers
and failures, rather than only that HEAD's diff. The [review plan](AGENT-SURFACE-REVIEW.md) owns the
scope and interface boundaries; this table owns live status. Any row is independently available.
On selection mark only that row **in progress**; findings keep that same unit open. Remove a row
only after its review is finished, retaining results/evidence gaps under Review findings, acceptance
gates and the changelog as applicable. Do not choose or perform fixes as part of this backlog.

| Unit | Responsibility | Status |
|---|---|---|
| [ASR-01](AGENT-SURFACE-REVIEW.md#asr-01--instruction-composition-personas-and-operating-knowledge) | Instruction composition, personas and operating knowledge | waiting to start |
| [ASR-02](AGENT-SURFACE-REVIEW.md#asr-02--effective-configuration-federation-and-frozen-launch-values) | Effective configuration, federation and frozen launch values | waiting to start |
| [ASR-03](AGENT-SURFACE-REVIEW.md#asr-03--provider-selection-packaging-authentication-and-model-discovery) | Provider selection, packaging, authentication and model discovery | waiting to start |
| [ASR-04](AGENT-SURFACE-REVIEW.md#asr-04--codex-native-adapter-and-instructionconfiguration-delivery) | Codex native adapter and instruction/configuration delivery | waiting to start |
| [ASR-05](AGENT-SURFACE-REVIEW.md#asr-05--claude-native-adapter-and-instructionconfiguration-delivery) | Claude native adapter and instruction/configuration delivery | waiting to start |
| [ASR-06](AGENT-SURFACE-REVIEW.md#asr-06--fresh-launch-process-registration-crash-and-restart-recovery) | Fresh launch, registration, crash and restart recovery | waiting to start |
| [ASR-07](AGENT-SURFACE-REVIEW.md#asr-07--resume-clone-fork-switch-and-native-history-continuity) | Resume, clone, fork, switch and native history continuity | waiting to start |
| [ASR-08](AGENT-SURFACE-REVIEW.md#asr-08--message-submission-held-input-steer-and-turn-ownership) | Message submission, held input, Steer and turn ownership | waiting to start |
| [ASR-09](AGENT-SURFACE-REVIEW.md#asr-09--stop-cancellation-process-containment-and-teardown) | Stop, cancellation, process containment and teardown | waiting to start |
| [ASR-10](AGENT-SURFACE-REVIEW.md#asr-10--acp-transport-callbacks-locking-and-backpressure) | ACP transport, callbacks, locking and backpressure | waiting to start |
| [ASR-11](AGENT-SURFACE-REVIEW.md#asr-11--permissions-session-modes-and-human-decisions) | Permissions, session modes and human decisions | waiting to start |
| [ASR-12](AGENT-SURFACE-REVIEW.md#asr-12--native-events-tools-child-agents-background-activity-and-context-usage) | Native events, tools, children, background work and context usage | waiting to start |
| [ASR-13](AGENT-SURFACE-REVIEW.md#asr-13--transcript-durability-replay-indexing-and-stream-continuity) | Transcript durability, replay, indexing and stream continuity | waiting to start |
| [ASR-14](AGENT-SURFACE-REVIEW.md#asr-14--submitted-text-mentions-files-images-and-attachment-bytes) | Submitted text, mentions, files, images and attachment bytes | waiting to start |
| [ASR-15](AGENT-SURFACE-REVIEW.md#asr-15--authorized-context-peer-reads-and-annotations) | Authorized context, peer reads and annotations | waiting to start |
| [ASR-16](AGENT-SURFACE-REVIEW.md#asr-16--agent-facing-mcp-hooks-caller-authority-and-result-contracts) | MCP, hooks, caller authority and result contracts | waiting to start |
| [ASR-17](AGENT-SURFACE-REVIEW.md#asr-17--peer-mail-deferred-delivery-waking-and-message-budgets) | Peer mail, deferred delivery, waking and budgets | waiting to start |
| [ASR-18](AGENT-SURFACE-REVIEW.md#asr-18--durable-tasks-waits-reports-and-chucky-delegation) | Durable tasks, waits, reports and Chucky delegation | waiting to start |
| [ASR-19](AGENT-SURFACE-REVIEW.md#asr-19--pipelines-and-standing-orchestration) | Pipelines and standing orchestration | waiting to start |
| [ASR-20](AGENT-SURFACE-REVIEW.md#asr-20--think-tanks-room-turns-private-input-and-judge-results) | Think Tanks, room turns, private input and judge results | waiting to start |
| [ASR-21](AGENT-SURFACE-REVIEW.md#asr-21--claude-interactive-terminal-pty-and-external-drivers) | Claude terminal, PTY and external drivers | waiting to start |
| [ASR-22](AGENT-SURFACE-REVIEW.md#asr-22--provider-failures-quota-admission-and-recovery) | Provider failures, quota admission and recovery | waiting to start |
| [ASR-23](AGENT-SURFACE-REVIEW.md#asr-23--operator-entry-point-parity-and-recovery) | Desktop, phone, CLI and HTTP action parity/recovery | waiting to start |
| [ASR-24](AGENT-SURFACE-REVIEW.md#asr-24--diagnostic-sufficiency-and-measured-agent-performance) | Diagnostics and measured agent performance | waiting to start |
| [ASR-25](AGENT-SURFACE-REVIEW.md#asr-25--project-execution-context-worktrees-and-shared-resources) | Project cwd, worktrees, setup and shared resources | waiting to start |

## Review findings

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

PROMPT-1 (native Codex developer instructions lost) is fixed; see Changelog.

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

## Changelog

- **2026-10-19 — Prepared agent-surface review backlog:** researched shared runtime and native
  Codex/Claude boundaries; queued 25 independent subsystem reviews with ownership, source/spec
  entry points, failure hypotheses and evidence requirements in
  [AGENT-SURFACE-REVIEW.md](AGENT-SURFACE-REVIEW.md). Planning only: no substantive review,
  product/spec change or provider run. Existing findings, gates and dirty code were preserved.
  `make check-specs`, source/spec-reference checks, local link/anchor checks and 25-unit queue
  consistency checks pass.

- **2026-10-19 — Fixed CI-1; late tool input CI flake unit closed** (INV §17 — test double
  diverged from the real peer): product ordering is correct. `fakeacp` closed one process-wide
  `cancelCh` on the first `session/cancel`, so every later permission prompt returned
  `cancelled` immediately and turn 2 of `TestCanceledTurnsReleaseToolInput` could end (releasing
  its input) between the test's frame injection and its count. Cancel is now scoped to the
  in-flight prompt (fresh channel per `session/prompt`). `TestCancelDoesNotEndNextTurn`
  (permission_test.go) fails 3/3 on the old double and passes; the flaky test passes 30/30.
  `TestSteerPromptRequiredRunsAHostOwnedTurn` relied on the sticky cancel to end its held
  successor and now releases it explicitly. Full Go (serial, both variants), focused `-race` and Go
  build pass; remote CI not yet observed.
- **2026-10-19 — Fixed PERM-1; no-persona permission review closed** (no invariant class; INV
  §3/§8/§13 checked): Settings → Roles gains **Default permissions** with a **Skip permission
  prompts** toggle over the global `skip_permissions` (`PermissionPolicyEditor`, same
  optimistic-save/restore pattern as Auto continue). The persona form option and row badge now
  read **Use default permissions** / **Default permissions** instead of "Inherit global".
  FS-04.R60/A39 added, R46 and FS-12's badge text updated. Tests: `PermissionPolicyEditor.test.tsx`
  (round trip across remount, failed-save restore), `RoleForm.test.tsx` option label. Full Go
  (serial), UI suite, UI build and Go build pass. Rendered browser check not run.
- **2026-10-19 — Fixed PROMPT-1; prompt/harness unit stays open on PROMPT-2** (INV §12, §2,
  §17): Codex launches now prepend the instructions Codex would otherwise use — the
  `CODEX_CONFIG` overlay value, else the native `developer_instructions` from the child's
  `CODEX_HOME` (user config, active `[profiles.<profile>]`, launch project `.codex/config.toml`)
  — before Chuck's prompt. `configsource.CodexDeveloperInstructions` follows the existing Codex
  resolver's layer order; the merge moved from `spawnCmd` into `startCmd` inside the
  profile-refresh lock so it reads the generation published for that child. Malformed native
  TOML or a non-string value fails the launch. FS-09.R32 and TS-04.R14 updated. Tests:
  `TestCodexNativeDeveloperInstructionsPreserved` (user, profile, project, overlay precedence,
  empty prompt, resume, malformed), `TestCodexStartUsesRefreshedNativeInstructions`; the
  invalid-overlay test now proves no child starts. Live provider adoption is still the FS-18.A13
  gate. PROMPT-2 remains open: it needs a new requirement and a choice of where a per-start
  manifest lives, since the runtime suffix is deliberately outside the frozen session snapshot.

- **2026-10-19 — Fixed agent-created task group review; group placement unit closed** (INV §8,
  §16, §10, §11): TG-1 — the shared `normalizeGroup` now rejects trimmed labels over 256
  characters with `invalid_group_name` at every launch, identity and `create_task` ingress, before
  any write (TS-03.R56; `TestNormalizeGroup`, oversized case in
  `TestAgentCreatedTaskChecksItsLaunchSpecification`). TG-2 — kept the task HTTP read field,
  consistent with effort/model/backend; FS-16.R50 now excludes only HTTP authoring and the Tasks
  view, and the group test asserts the HTTP detail reports it. Live Chucky receipt moved to
  Acceptance gates. No open findings remain for this unit.

- **2026-10-19 — Fixed file-reader review findings; file reader formats unit closed** (INV §10,
  §17): the file-read API comment now says relative paths resolve from the recorded cwd and
  absolute paths use local process authority (TS-03.R48). The withdrawal-recovery test asserts
  the exact recovery error, not any `could not be loaded` text. The phone two-gap test's flake
  is CPU-load timing: 16 concurrent copies fail 16/16 at the 5 s wait for “Gap question 4700”
  and pass 16/16 with a 20 s wait (test bound 60 s); no product change. Test/comment only, no
  spec change. Closure: UI 976/976, UI build. No open findings remain for this unit.

- **2026-10-19 — Fixed dead Think Tank roster CSS; compact chat navigation unit closed**
  (INV §10/§13): deleted the unreferenced `.think-tank-card-roster` rules and their false
  room-cue comment; no `ui/src` reference remains. CSS-only, no spec change. Closure: style and
  presentation checks, UI 976/976, UI build, spec check. No open findings remain for this unit.

- **2026-10-19 — Fixed PHONE-1; phone groups/Jump/runtime-buttons unit closed** (INV §8): the
  phone chat now re-checks Jump to latest whenever transcript rows change, so a live reply that
  grows the chat without a scroll offers the button (FS-20.R54; no spec change). New `PhoneApp`
  regression fails without the fix; the reviewer's `--growth-only` Chromium reproduction passes.
  Closure: `GOFLAGS=-p=1 make test`, `make build`, UI 976/976, UI build. No open findings remain
  for this unit; paired-device/native keyboard/live-provider acceptance is still not claimed.

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
