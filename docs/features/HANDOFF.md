# Chuck — Implementation handoff

Settled history through v0.23.0 preparation is archived in
[HANDOFF-through-2026-11-08](../archive/state/HANDOFF-through-2026-11-08.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archives are not live queues.

## Current position

- **Release:** operator confirmed v0.23.0 and authorized push on 2026-11-08. Main and the
  annotated v0.23.0 tag at `f3f34986` are pushed. Missing v0.22.0 baseline tag was pushed with
  explicit approval; its own workflow `42450809571` is cancelled. v0.23.0 retry `42450625734`
  passed architecture, assembly, transaction/bootstrap and fresh-install checks, then published
  [v0.23.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.23.0).
  [Notes](../archive/state/RELEASE-v0.23.0-notes.md) are published too. Archive/manifest/installer
  assets are present; manifest size and SHA-256 match GitHub's uploaded archive metadata.
- **Architecture:** combined source audit and all quota repairs are reused, as explicitly
  directed by the operator on 2026-11-08. No second architectural review.
  [Receipt](../archive/reviews/architecture/v0.22.0-next.json) includes release preparation.
  Original audit findings are closed; separate publication-time CI findings remain below.
- **Verification:** hosted release assembly and fresh installation pass. Main CI retry
  `42450626202` passes both Go variants, specs, gate tests and vet, but failed one of 1181 UI
  tests (waiting-input pane opening); that fixture race is fixed locally, unconfirmed by a new
  hosted run. Do not claim full clean-clone CI success. Native/provider/device gates below
  remain owed.
- **Installed runtime:** managed current was v0.20.0 at audit; no install or running-dashboard
  update is part of this release preparation. Global PATH adapters remain unpatched.
- **Checks:** unset CHUCK_RUNTIME_ROOT; use GOFLAGS=-p=1 and a writable isolated GOCACHE.
  Full Go suites need loopback-capable execution.

## Active change

None.

Think Tank templates have findings available for fix work below.

## Acceptance gates still owed

- **Think Tank templates (FS-21.A43–A45, TS-14.R33):** independent Chromium on the built
  desktop bundle and isolated fake-ACP server passed load, unavailable-persona repair, long-row
  layout, declined replacement, room instruction disclosure and room save dialog at 1024px/1600px
  in all four appearances. Core also passed keyboard focus restoration, draft save without a goal,
  active/ended room save and increased-ceiling reload. A saved existing-chat row launched as a
  fresh destination chat and the source template remained intact.
  Chromium reproduced TT-TEMPLATE-3 and TT-TEMPLATE-4. The state tests cover source deletion.
  Recheck the affected rendered paths after repair and exercise A45 with fake-provider turns
  through retry, closing and synthesis plus the SSE check in TT-TEMPLATE-1. The focused Go suites
  and 78 component tests passed.

- **File freshness/editing (FS-03.A86–A88):** actual Chromium desktop live/archive/room/Focus
  journeys and the real phone bundle at 360/390/430px pass. Phone SSE/tracked-file admission was
  fixture-controlled; physical-device, real paired reconnect and non-Chromium gates remain open.

- **FS-03.A85 native default browser:** JXA lookup ran but this environment has no configured
  default web browser. On a normal Mac, right-click a self-contained HTML file and select
  **Open in browser**; confirm the original file opens in the default HTTPS browser (even when
  `.html` is associated with an editor). HTTP dispatch, retained sources, remote refusal, native
  argv/timeout/error handling and in-app live/archive/room rendering are automated evidence only.

- **GOAL-LED evidence (FS-14 §6):** A61 run-start-to-notification browser journey; A64 attachment
  variant and live desktop/phone composer check; A67 fake-provider invocation-reason capture; A68
  bounded packaged Claude/Codex automatic-replacement probe (credentials).
- **QUOTA-RPC live receipts (FS-01.A37/A39):** with credentials under a real quota refusal,
  record one bounded Claude and one Codex terminal-RPC receipt (category value, wording) and the
  resulting automatic continuation at reset plus one minute. Fixtures are captured/source-backed only.

- **Pipeline worktree packaged probe (TS-09.R65):** with credentials, start an isolated run from
  the packaged app and confirm a Claude and a Codex stage agent actually run (and resume) in the
  run checkout; fake-provider capture and real-Git fixtures pass.
- **BROWSE-LAT native picker:** JXA activation probe took 0.10s versus AppleScript's 2.29s,
  and a real JXA panel returned `User canceled. (-128)`. Computer use could not target the
  osascript panel. Still verify selection returns a usable absolute path, the panel appears
  over the browser, and click-to-visible-panel timing; activation-only timing is not panel latency.

- **BORING-NIGHT rendered evidence (FS-12.A42):** Settings/Appearance, live switch, reload
  persistence, the project dashboard with busy/idle/error cards, the agent screen, a Go code
  block, a diff fence and a Mermaid diagram were rendered in Chromium (review 2026-10-31). An
  xterm terminal (selection uses `--ad-action-secondary` `#b3d2ea` under `#cedbec` text),
  overlays/permission prompt and the phone Theme select are not yet rendered.

- **FS-03.A80–A82 / FS-20.A28 / FS-21.A42 (file tabs):** physical-phone touch reordering and
  reconnect, non-Chromium engines, and rendered proof of the 500-total/2 MiB limits and the
  storage-unavailable notice (unit-tested only) remain owed.

- **ASF-2 native/device receipts:** deterministic fake-ACP wire counts, HTTP/CLI/component checks,
  and isolated desktop Chromium withdrawal/unknown-recovery journeys establish the repaired input
  boundaries. They do not establish credentialed Claude/Codex consumption timing or a physical
  phone's background/reconnect behavior. Keep those existing native/phone gates open.
- **Earlier carried checks:** phone component proof for definite-refusal Retry, live Claude
  receipt for late tool input, and rendered verification of the withdraw failed-recovery-read
  tail notice remain owed.
- **ASF-1 live setting receipt:** credentialed mid-turn effort/fast application remains unverified.
- **Test limits:** the concurrent closure hit the known quota-dispatch FTS5 transcript flake;
  full suites and five focused repeats passed. The two runtime cancellation tests that failed
  under `-race` were repaired by ARCH-v019-2 (`-race -count=20` passes).
- **Timed starts on a real Mac:** native sleep through a start time and a time-zone change
  across restart are proven only by store/fake-clock tests; no native macOS receipt exists.

- **FS-07.A12 — real tmux/iTerm2 ownership receipts (ASW-3):** neither tmux nor iTerm2 is
  installed on the implementing machine. On a host with both: two terminal agents against an
  already-running tmux server and a real iTerm2 launch; verify each CLI's environment/hook identity,
  native SessionStart → Stop → Resume, Cancel, Stop, viewer detach versus CLI exit, descendants and
  dashboard crash → Stop/Switch. Fake tmux/osascript fixtures (`drivers_test.go`) and the real-PTY
  bootstrap tests do not close this. Also unverified: iTerm2 shells that do not lead their own
  foreground group would now refuse launch (fail-closed) rather than run.

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
  the `_meta.systemPrompt` request, not live adoption). ASR-I below adds a bounded paired
  performance experiment; no native-versus-Chuck quality or latency comparison has run.
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

- **Think Tank templates** (`6451e66b`, `35f951e9`, `3b0f8ff8`, FS-21.R60–R63, TS-14.R30–R33):
  reviewed 2026-11-12. Open for TT-TEMPLATE fixes below and independent fix review. Existing-chat
  names are valid optional launch names. A departed participant remains a roster slot in the
  snapshot, with no departure state. Template rename/delete is outside this change.

## Investigations awaiting evidence

### EFFORT-BUDGET — worker infers a work budget from its own configuration (2026-11-05)

**Report (verbatim):** “agents in a pipeline read the implementation manifest, saw their reasoning
was set to low, and decided to self limit on an imaginary budget, which made them fail the task
because 'the budget ran out'”. No run/agent identity, timestamp, manifest path, version or logs
were supplied. Investigated source at `47268443` on macOS and read-only local Chuck state and
recorded conversations. No matching incident was identified; the report's “implementation
manifest” must not be assumed to mean Chuck's runtime `ExecutionManifest`.
**Clarification (verbatim):** “The worker one the one that saw its own config and made up it's
own additional budget”. The reported initial failure is the worker's self-imposed constraint;
coordinator acceptance and subsequent recovery exhaustion have not been established.

**Expected behavior / classification:** FS-14.R97–R99 authorizes the work needed to meet the goal,
requires the owner to evaluate failed/blocked child evidence, and reserves human attention for
a real unavailable decision, permission, checkpoint or exhausted operation. FS-18.R15/R21 and
TS-11.R24 define runtime facts and requested-delivery metadata; TS-09.R68 explicitly leaves
semantic completion judgment with the owner. Low effort does not change those obligations.
The reported premature stop would violate that completion contract. **Diagnosis: undetermined.**
The absence of an explicit warning against this particular inference does not establish a prompt
defect or justify adding standing context. The earlier probable guidance finding, mandatory fix
classification and model recommendation are withdrawn. No code defect, specification gap or
missing diagnostic at the actual failure point has been demonstrated from the available evidence.

**Evidence and limits:** no effort-to-work-budget conversion was found in the inspected source.
Actual recovery is `pipeline.recovery.used/limit`, default three
(`internal/state/pipeline_decisions.go:27,229–244`), charged on repeat/backward/recovery
transitions (`internal/pipeline/decisions.go:67–87`), independently of effort or report prose.
`internal/state/pipeline_tasks.go:317–388` validates report authority/structure, as specified;
acceptance does not certify the truth of a claimed blocker. The existing
`TestRecoveryExhaustionPausesAndResumeRenewsTheAllowance` and
`TestQuestionPausesAndTheAnswerContinuesTheRun` pass (focused `go test ./internal/pipeline`,
2026-11-05). They verify genuine recovery/pause mechanics, not the reported model behavior.
No behavioral reproduction or skipped regression is claimed. Next evidence needed is the affected
worker's exact assignment, config/manifest tool result, and recorded explanation/report of the
stop. Inspect those before choosing a remedy; determine whether an instruction, misleading
metadata presentation, actual provider limit, or unsupported model inference supplied the alleged
budget. Any proposed change needs a bounded comparison with independently checkable completion
evidence; prompt delivery alone cannot establish improvement. History separates manifest recording
(`90f27319`, expanded `0f89e683`) from goal-led
recovery (`fb410fdb`); it does not establish when the reported inference began. No product code,
specs, live user state or unrelated dirty files changed.

## Review findings

### Think Tank templates — **Fix model:** medium — Codex Terra or Claude Opus.

- **Must fix** TT-TEMPLATE-1 (INV §11/§14/§17; fix complexity: medium) — The first authorized
  room context read returns private participant instructions. Shared room activity compacts that
  tool result, but a valid ACP `tool_call_update` that echoes it becomes a `tool_result` with raw
  content (`internal/runtime/acpmap.go:541–558`). The history publisher sends that event through
  the ordinary `/api/events` stream, including paired phone (`internal/bus/bus.go:158–161`,
  `internal/server/sse.go:59–67`). TS-14.R32 explicitly excludes the field from SSE. Cover the
  real MCP response plus an echoing fake ACP frame, and verify the authorized read retains the
  instructions while shared room activity and ordinary SSE exclude them. No shipped adapter echo
  was observed in this review.
- **Must fix** TT-TEMPLATE-2 (INV §3/§9/§17; fix complexity: medium) — Saving a room or an
  existing-chat draft stores its source room or agent IDs in `intent` on the template row
  (`internal/state/think_tank_templates.go:79–95,214–218,259–260`). The API response omits them,
  but FS-21.R60 and TS-14.R30–R31 keep source identities outside the snapshot and use the agent
  identity for lookup only. Use a fingerprint of the normalized replay request, account for
  already-saved rows, and check stored data as well as exact replay/conflict after source deletion.
- **Must fix** TT-TEMPLATE-3 (INV §1/§5/§8; fix complexity: easy) — A slow template response can
  replace roster or discussion edits entered after requesting it, with no confirmation for
  those edits (`ui/src/features/thinktank/ThinkTankSetupDialog.tsx:142–174`). Clearing to blank
  also returns before advancing the token, so the late response can restore an unwanted template.
  Fence pending loads on superseding setup changes, including blank, or make replacement an
  explicit commit against the current draft (FS-21.R61, TS-14.R33). Chromium reproduced a limit
  edit from 27 reverting to the template's 4. Cover that and a late response after switching to blank.
- **Must fix** TT-TEMPLATE-4 (INV §15; fix complexity: easy) — Both save surfaces mint a new
  command ID each time the operator submits (`ThinkTankSetupDialog.tsx:215–230`,
  `RoomSaveSetup.tsx:15–24`). When a save commits and its response is lost, retry creates a second
  copy. Retain the command for an identical retry and rotate it on a new intent or acknowledged
  save. Chromium reproduced two draft templates after dropping the response to the first committed
  save. Assert one template under the same schedule from both entry points (FS-21.R60, TS-14.R31).
- **Must fix** TT-TEMPLATE-5 (INV §1/§8; fix complexity: easy) — After an existing-chat member
  switches to Terminal following the discussion, Save setup on the ended room reports its launch
  choices are unrecoverable. The query reads the portable fields but rejects the current interface
  (`internal/state/think_tank_templates.go:110–116,156–163`). Keep chat eligibility for a draft
  lookup and project the committed member's recoverable choices for a fresh chat. Exercise this
  after a real retained member switch (FS-21.R60, TS-14.R31).
- **Must fix** TT-TEMPLATE-6 (INV §8/§10; fix complexity: easy) — A template whose model remains
  configured but no longer supports its saved effort or fast mode receives no row warning
  (`ThinkTankSetupDialog.tsx:94–109`). Start proceeds to setup and can fail launch validation
  (`internal/server/launch.go:396–404`). Check current model and chat capabilities for the saved
  choices, mark the affected participant or judge, and make repair explicit before starting
  (FS-21.R61, TS-14.R30/R33).
- **Worth fixing** TT-TEMPLATE-7 (INV §7/§16; fix complexity: easy) — Room Save setup reads full
  room detail inside the template transaction, including attempts and pending input bodies it
  never uses (`internal/state/think_tank_templates.go:148–172`, `internal/state/think_tanks.go:904–951`).
  Read the bounded setup and roster from the same snapshot without materializing unrelated history
  and queued input. Verify room save with a large retained history and pending queue (TS-14.R31).
- **Worth fixing** TT-TEMPLATE-8 (INV §9/§11; fix complexity: medium) — Template schema version
  exists only on the wire. The table stores no version and reads label every retained payload with
  the running build's constant (`internal/state/think_tank_templates.go:87–95,314–323`). Store the
  version or a versioned payload, and verify that v1 data remains decodable across a schema change
  (TS-14.R30).
- **Worth fixing** TT-TEMPLATE-9 (INV §10/§16; fix complexity: easy) — The dropdown silently
  discards any continuation after ten pages of 100 summaries (`ui/src/api/thinkTanks.ts:335–350`).
  Older saved setups remain in SQLite but become unreachable through the desktop modal as new
  templates accumulate. Expose further pages on demand while keeping each fetch bounded and
  preserving draft state (FS-21.R61, TS-14.R31).
- **Worth fixing** TT-TEMPLATE-10 (INV §8/§11; fix complexity: easy) — The instructions field's
  native `maxLength` counts UTF-16 units (`ThinkTankSetupDialog.tsx:328–332`), so input consisting
  of supplementary Unicode characters is cut off at 4,000 instead of the server's 8,000-rune
  limit. Count runes and verify at the accepted and rejected boundaries (FS-21.R63, TS-14.R32).

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).
Latest published notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).

## Changelog

- **2026-11-12 — Fix:** closed the no-raw-ids review (UI-IDS-1–6; INV §2/§7/§8/§10/§16/§17).
  Annotation cards name another-agent recipients or read *Another agent*; non-Git start/fork
  errors and unknown-project diagnostics name the title, not the slug. Prerequisite options and
  peer Stop rows reuse the duplicate-name rule (FS-06.R41 now says so). Push resolves titles only
  for notes it sends, with "an unavailable project" fallback (FS-20.R19/R33). Breadcrumb harness
  asserts no id text (258 Chromium checks pass); phone/run tests seed `/api/projects`. `make
  test`, 1,204 UI tests and the UI build pass. Unrelated `/api/roles` handler gaps remain in
  pipeline and phone-app tests.

- **2026-11-12 — Design:** native quota snapshot and fork from a message are ready as
  [native-quota-snapshot-and-message-fork](../ready-changes/native-quota-snapshot-and-message-fork.md)
  (FS-09.R83/A52 replacing R81/A50, FS-01.R58–R59/A40–A41, TS-04.R109–R111, TS-03.R85–R86,
  TS-02.R65, TS-08.R162). The operator rejected a full move off ACP and approved, for these two
  items only, a typed quota-read patch per packaged adapter and the adapters' existing
  `_meta.jetbrains.air.fork` key; Edit always forks; quota recovery may use the snapshot with the
  shipped behavior as fallback. Pinned-bundle evidence is in the change file; Claude's idle
  usage read and both live receipts remain acceptance gates. Amended the same day at the
  operator's direction: one shared in-memory snapshot per backend quota scope, read automatically
  and shown by every chat of that backend, with no Refresh control or route.

- **2026-11-12 — Design:** agent mail visibility is ready as
  [expandable-agent-mail-activity](../ready-changes/expandable-agent-mail-activity.md)
  (FS-06.R45/A33, TS-04.R108, TS-08.R161, FS-03.R108 pointer; TS-08 is now Partial). The operator
  chose to keep received bodies with the transcript, expand the notice and mail tool lines in
  place, leave agent-facing views unchanged, render bodies as Markdown, and have Chuck record
  sent and read mail itself. Evidence from real transcripts: Codex saves null mail tool results,
  and Claude saves empty send arguments unless a permission request records them.

- **2026-11-12 — Review:** internal-ID display (`178425ba`) remains open for UI-IDS-1–6.
  Five focused component files passed 143 tests with the fixture gaps above. Presentation checks
  and focused Go remote home/push tests passed. Chromium's populated pipeline fixture passed 120
  checks at 1024px and 1440px in Core,
  Sky & Grove and Studio. The breadcrumb harness stopped on its obsolete id-chip assertion.
  Invariant sweep applied classes 2/7/8/10/11/13/14/16/17. Classes 1/3/4/5/6/9/12/15 had no
  applicable surface. The peer Stop identity contract needs reconciliation, as recorded above.

- **2026-11-12 — Fix:** closed the v0.23.0 main-CI annotation replay race (RELEASE-023-CI-1;
  INV §17 tests prove their contract independently). The self-annotation fixture now holds
  `taskSweepMu` across admission and synthetic restart/replay, releasing it before explicit
  dispatch, so asynchronous dispatch cannot consume the input first; replay safety assertion and
  product receipt safety unchanged. Focused test 30/30, `-race` 10/10; `go vet` and both Go
  variants of `./internal/server` pass. A new hosted main CI run must confirm.

- **2026-11-11 — Work:** finished [Think Tank templates](../specs/features/FS-21-think-tanks.md)
  (FS-21.R60–R63/A43–A45, TS-14.R30–R33). `make test`, `make build`, full UI `npm test`
  (1201) and UI build pass; legacy v47 migration fixture gained the members table every real v47
  database has. No rendered browser pass.

- **2026-11-11 — Fix:** closed the v0.23.0 main-CI UI failure (RELEASE-023-CI-2; INV §17
  tests prove their contract independently). CardGrid cases rendered, clicked or delivered a
  waiting transition before the grid's layout read settled; a late read overwrote clicked panes
  or reseeded the transition (correct FS-02.R61 behavior). Cases now wait for the seeded
  layout's gap marker; product unchanged. Base fails 7/42 with a 300 ms delayed layout read,
  fix passes 42/42 delayed and normal; full `npm test` 1195/1195 twice; `make check-specs` ok.
  UI build not rerun (test-only). A new hosted main CI run must confirm.

- **2026-11-11 — Work:** removed internal ids from user-facing text (operator request):
  breadcrumb id chip, run/room/proposal/template code chips, `name || id` and `(id)` / `· id`
  suffixes, project-slug fallbacks (now title, blank while loading, or *Unavailable project*),
  phone `role@project` titles and push bodies. Specs: FS-02.R25/R69/A13, FS-03.R98, FS-12.R61/R70,
  FS-14 builder panel, FS-20.R12/R19, TS-08.R60/R91. `make test`, full `npm test` and UI build;
  CardGrid waiting-input flake (RELEASE-023-CI-2) is independent.

- **2026-11-11 — Design:** Think Tank templates were made
  ready to start. Confirmed project-independent snapshots that load fresh launch settings into the
  creation modal, editable participants, and optional room-specific instructions
  (FS-21.R60–R63/A43–A45, TS-14.R30–R33).
- **2026-11-11 — Fix:** closed desktop chat composer editing (COMPOSER-LIST-1–2; INV §8
  parsed user-facing data / §10 wiring drift). Shift+Enter no longer numbers lines
  inside a fence opened on an unordered item, and an item after an ATX heading continues from
  any start number; paragraph fallback unchanged. Both regressions fail before the fix.
  `make check-specs`, 69 composer/editing tests and the UI build passed; full `npm test` failed
  only the pre-existing CardGrid load case (RELEASE-023-CI-2, also fails without this fix). Both
  desktop entry points share `editNumberedList`; no rendered Chromium rerun.
- **2026-11-11 — Review:** desktop chat composer editing remains open for COMPOSER-LIST-1–2.
  Specification checks, 67 focused tests and the existing Chromium keyboard/pointer journey
  passed. A separate Chromium probe reproduced both failures in full agent chat and the expanded
  dashboard (`/private/tmp/chuck-composer-review-probe.mjs`, fixture API). The app's Markdown
  parser independently confirmed both block contexts. Tests use explicit text/selection and
  request/transcript expectations. No pending local choices. Invariant sweep applied classes
  1/2/3/5/8/10/13/16/17. Classes 4/6/7/9/11/12/14/15 had no applicable surface.
- **2026-11-11 — Review:** closed pipeline stage work status (`c947386b`, FS-14.R39/A17 and
  TS-08.R149) with no findings. Independent Chromium runs exercised the routed fixture at
  1024px and 1440px in all four appearances, including reduced motion and live/archive links.
  Fixture updates confirmed waiting → running → finished in place in Core at 1024px.
  Specification checks and all 10 run-browser tests passed. No fix model is needed.
- **2026-11-09 — Implementation:** finished
  [desktop composer editing](../ready-changes/desktop-chat-composer-editing.md).
  Specification checks, 1,193 UI tests, the production UI build and both rendered Chromium
  journeys passed. Added the invariant classifications required by the spec check to the two
  open CI findings.
