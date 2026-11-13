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

**In progress:** [Single-owner turn state](../ready-changes/single-owner-turn-state.md).
Turn fields now share one structure under the existing turn lock. Chat status writes share one
transaction and Stop closes it for that launch. The regression reproduces a late completion
overwriting `done` when the Stop fence is removed. Run the focused race checks, both Go test
variants and the build before queuing review.

## Acceptance gates still owed

- **Quota snapshot (FS-09.A49/A52):** one bounded credentialed read each on packaged Claude and
  Codex, compared with the provider's own usage display (window identity, percentage, reset
  instant). Real `get_usage` with `skipBehaviors` and real `account/rateLimits/read` shapes are
  unverified. Rendered 2026-11-12 in independent Chromium (working-tree UI, real binary, isolated
  home, fake ACP): the complete state at 1024px, 1440px and the 390px phone page in Core and
  Sky & Grove showed no overflow, a visible focus ring, keyboard opening and no request on
  open. Stale, partial and unavailable states were not rendered. The phone page was loaded
  without real pairing. After QUOTA-FORK-7 the phone summary measured 390×44 in Chromium
  (isolated markup with the real stylesheets, not the full phone page). The Claude adapter now
  forwards only window members; the `model_scoped` decode is fixture-verified only.
- **Fork from a message (FS-01.A40):** one bounded credentialed fork each on packaged Claude and
  Codex proving the new provider context ends at the chosen turn; Codex must also accept the
  proof-only `session/load` of its fork. Rendered 2026-11-12 in the same run at 1280px in Core:
  enabled and disabled Edit and Fork from here, both confirmations, navigation, Edit's unsent
  text and the unchanged source passed. Keyboard-only menu use, Sky & Grove and the dashboard
  pane were not rendered. Re-render the disabled Edit on a message sent mid-turn and a copied
  turn's reason (QUOTA-FORK-2/4 are component/API-tested only).
- **Mail detail (FS-06.A33):** independent Chromium used the built UI, an isolated Go server,
  fake ACP, real MCP send/read and fixture tool-call events. The unfinished send disclosures
  rendered Markdown and a refusal at 1024px in Core. Keyboard expansion and Markdown layout
  also passed at 1024px/1600px in Sky & Grove.
  The authenticated 390px phone view expanded received and read mail. Received bodies remained
  available after mailbox expiry and archive reload. The completion detachment is repaired and
  covered by projection tests (2026-11-13). Re-render completed, archived and phone activity
  end to end. The physical paired-phone check, including tapping the 44px Mail button, remains
  owed.

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

- **Shared desktop and phone input delivery** — ready for review (TS-08.R164). Both clients
  share send, Steer, retry, attachment completion, withdrawal and the recovery offer. Desktop
  projects the prompt into its transcript store; phone waits for sequenced events. The full
  UI suite (1261 tests), UI build and 390px Chromium harness passed. The harness uses stubbed
  responses. On reconnect, desktop refetch advances the transcript token and clears the
  reconciliation marker for open agents. Phone still uses its held query. Review the lifetime
  of browser retry IDs when another client settles the recovery offer.

- **Shared resume start and Switch rollback** — ready for review at `95d4f621` (TS-01.R43).
  Explicit resume, wakes and both Switch starts share registration cleanup and applied identity
  persistence. Switch stops a target whose final identity write failed before starting rollback.
  The build, specification checks, both Go variants and focused race checks passed. The first
  untagged run hit a pipeline Stop 409. Ten focused repetitions and the full rerun passed.
  Both new Switch persistence regressions fail against the original source.

- **One typed turn outcome** — ready for review. Runtime status, quota settlement, inline mail
  and event readers use the resolved outcome. See FS-03.R117/A92 and TS-01.R42.
  Both Go variants, the build, and focused race checks passed. The fork fixture now waits for the
  retained turn boundary and passed eight tagged repetitions. The known deferred-assignment
  lifecycle test failed once in the untagged matrix, then passed five focused runs and the full
  untagged rerun.

- **Match sent prompts by request ID** — ready for review. Desktop transcript reconciliation and
  held follow-up delivery use the request ID. See FS-03.R118/A93 and TS-08.R163.
  Verification on the implementation diff: `make check-specs`, UI suite (1,251 tests),
  `npm run build` and `git diff --check` passed.

**Core reliability audit units (added 2026-11-13):** explicitly requested subsystem reviews of the
complete current implementation and callers, not of a commit. Read the charter, refresh fix
ownership and record the reviewed HEAD. Intended for a frontier reviewer; the charters' leads are
questions, not findings.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A8](CORE-RELIABILITY-REVIEW-A8.md) | Pipeline pause, answer and recovery ownership: who can end each pause, refused input with no side effect, answers through replacement, stall detection | reviewed at `35f33a07`; findings open under [Review findings](#review-findings) |
| [CR-A9](CORE-RELIABILITY-REVIEW-A9.md) | Think Tank lifecycle and its quota join: fenced member launch, requested Pause/End under quota, restart recovery | waiting for review |
| [CR-A10](CORE-RELIABILITY-REVIEW-A10.md) | Durable state transitions have one rule: status-writer inventory, unguarded and divergent writers, recommendation for a transition-rule design | waiting for review; run after CR-A8 and CR-A9 |

CR-A8's result: fix PIPE-CHAT-1 and -3 together with CR-A8-3, -4 and -7 as one input-record
change. CR-A8-1, -2, -5, -6 and -8 are independent of it and can be fixed first.

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

### Blocked pipeline chat and withdrawn-message recovery (PIPE-CHAT) — **Fix model:** difficult — Codex Sol.

Field report, 2026-11-12. The affected run, conversation, build and provider are unidentified.
The report supplied no request log or provider receipt. The local dashboard has no paused run,
and its retained run and input metadata did not identify the reported submission.

> /investigate-bug messages sent though the chat to a pipeline orchestrator when it's blocked sometimes disappear, I think chuck treats them like tool calls (as if they came from the pipeline continue/send message box) because when it needed my explicit approval the agent said it came through as a system call through chuck so the approval was rejected. Also I have the Withdraw message box on but empty:Withdrawn, but the message and its attachments could not be loaded. Load withdrawn message (button doesn't do antyhing)

The paused-run chat branch came from `376add97`. The HTTP reproduction reports the answer sent
without a durable user event; it is committed skipped for the fix. PIPE-CHAT-2 and -4 are fixed.
CR-A8 reviewed the contract: fix 1 and 3 in one change with CR-A8-3, -4 and -7 below.

- **Must fix** PIPE-CHAT-1 — confirmed, difficult. Sending an answer in the owner's chat during a
  `needs_input` pause or a recoverable blocked visit enters the same continuation branch as the
  run page. `internal/pipeline/actions.go` embeds the words in the next assignment, and
  `internal/pipeline/assignment.go` clips that instruction under both the continuation and total
  assignment limits. `internal/messaging/task_tools.go` exposes it through `get_assigned_task`.
  `internal/server/queued_input.go` marks text-only input submitted before provider delivery, or
  strips its text from the queued payload when attachments exist. The optimistic text bubble in
  `ui/src/components/chat/Composer.tsx` has no durable user event to rehydrate. Preserve the full
  accepted chat payload with its user provenance through assignment delivery, and verify the wire
  payload, delivery status, replay, and transcript after reload for text and attachments. Keep
  pipeline checkpoint and pending provider permission decisions on their explicit controls.
  The reported approval refusal remains undetermined. FS-03.R99, FS-14.R99–R100/A63–A64,
  TS-09.R69/R71, INV §2/§11/§15.
- **Must fix** PIPE-CHAT-3 — confirmed by source, difficult. `internal/pipeline/actions.go` creates
  and publishes a successor in `internal/pipeline/decisions.go` before its separate receipt bind.
  Admission can observe the visit before that bind. A bind error after the visit commits is logged
  and converted to ordinary standalone delivery in `internal/server/queued_input.go`, even though
  the run has already advanced. Commit the successor and input receipt together before publication.
  Inject a bind failure and race admission/retry against acceptance, verifying that no input is
  lost, misrouted or delivered twice. TS-02.R55, TS-09.R69, INV §5/§15.

### CR-A8 — Pipeline pause, answer and recovery ownership — **Fix model:** difficult — Codex Sol.

Subsystem review of the complete implementation at `35f33a07` ([charter](CORE-RELIABILITY-REVIEW-A8.md)).
Dirty work at review time was documentation only (this file's CR-A10 queue row and `docs/ideas.md`).
"Probe" means a throwaway test against the real store and manager, run and then deleted; "source"
means read and traced, not executed. No product code, test or specification changed. The level is
set by CR-A8-3, which shares PIPE-CHAT-1's remedy; every other finding is medium or easier.

**Pause ledger.** Run states are `pipeline_runs.state` / `pending_action` / `attention_reason`.

| Pause | Who can end it | Evidence of progress | Gap |
|---|---|---|---|
| `paused` / none / `needs_input` | person: run-page answer or owner chat | `TestQuestionPausesAndTheAnswerContinuesTheRun`, `TestConversationAnswerContinuesQuestionPause` | PIPE-CHAT-1, CR-A8-3, CR-A8-4 |
| `paused` / none, task interrupted (`interrupted`, `owner_not_reporting`, `restart_interrupted`, `owner_unavailable`) | person: Resume, Retry, Replace, owner chat | `TestStalledOwnerIsContinuedThenPaused`, `TestInterruptedOwnerRecoversAutomaticallyWithinTheAllowance` | CR-A8-7 |
| `paused` / none / `recovery_exhausted`, `required_check_missing`, `approval_required`, or a failed/blocked result | person: Resume or owner chat | `TestRecoveryExhaustionPausesAndResumeRenewsTheAllowance` | CR-A8-3, CR-A8-4 |
| `paused` / `await_approval` | person, run page only; chat is skipped | `TestApprovalGateHoldsCompletionUntilApproved` | input sent with the API call is dropped (CR-A8-7) |
| `paused` / retained action / `controller_recovery_failed` | person: Resume | `TestPersistentControllerFailurePausesAndResumes` | a restart destroys it (CR-A8-1) |
| `paused` / none / `restart_reconcile_failed`, or the room-context refusal text from `createVisit` | nobody but Stop when the current task finished successfully or is an armed successor | none | CR-A8-1 |
| `finishing` / `release_stage_task`, `stopping` / `cleanup_run`, `paused` / `activate_replacement` | `Reconcile` from turn end, the dispatcher tick and startup; Repair cleanup | `TestStartupRetainsAcceptedStageCursorAfterControllerFailure`, `TestFailedStartupRecoveryRetainsTheStopFence` | startup failure of `activate_replacement` falls into CR-A8-1 |
| `queued`/`running` with an open visit and a silent owner | stall detector, in memory and event-driven | `TestStalledOwnerIsContinuedThenPaused`, `TestStallCheckDeferredByBackgroundWorkResumesWhenItEnds` | CR-A8-5, CR-A8-6 |
| `queued`/`running`, owner in `wait_for_tasks` | watched child changes; no deadline | not reviewed here | lead 13 belongs to CR-A2 |

**Input records.** A chat message for a waiting assignment is one `prepared` receipt bound to the
task, and it follows a replacement (`TestQueuedAttachmentFollowsReplacementOwner`). A chat answer to
a paused run is a receipt marked `submitted` at bind, its words living only in the new visit's
instruction. A run-page answer or Resume instruction has no receipt when the visit carries the
words. Refusal without side effect is proven only for an empty answer on the run page; no test
covers a refused chat answer.

**Confirmed findings.**

- **Must fix** CR-A8-1 — probe, medium. `Manager.Startup` (`internal/pipeline/manager.go:816-841`)
  reconciles every run with a pending action and retains the action only for `finishing`. A run
  already held as `paused` / `release_stage_task` / `controller_recovery_failed` whose cause
  persists across a restart reaches `pauseStartupRun`, which clears the action. Before the restart
  Resume retried it; afterwards Continue, Retry and Replace all answer `invalid_state` and the
  dispatcher lists nothing, with the accepted result unadvanced. By source, not probed, the same
  Stop-only state follows a startup failure of `activate_replacement` and the room-context pause in
  `internal/pipeline/decisions.go:206`. `continueControl`
  (`internal/server/pipeline_projection.go:429`) still offers Continue for all three. Keep the
  retained action through startup, give every projected pause a legal Continue, and test each
  pause reason against the manager rather than the projection. TS-09.R42/R43, FS-14.R99/R101,
  INV §8/§18.
- **Must fix** CR-A8-2 — probe, medium. `AcceptPipelineStageTaskResult`
  (`internal/state/pipeline_tasks.go:341-369`) returns `ErrPipelineStageConflict` for a missing
  required output, an undeclared output name, a wrong execution handle, a pending wait, the stop
  fence and a revision race. `internal/messaging/task_tools.go:215` reports all of them as
  `already_reported`, "This task already has a recorded result", and logs nothing. An owner that
  omits one output is told it is finished, ends its turn, and the run pauses as
  `owner_not_reporting` after three silent turns. Return a typed refusal per condition naming the
  correction, log it, and test message against condition. TS-09.R68, INV §21/§22.
- **Must fix** CR-A8-3 — probe, difficult. `replaceOwner` (`internal/pipeline/actions.go:297`)
  writes a fixed continuation and `PreparePipelineStageReplacement` moves only `prepared` receipts.
  An answer or Resume instruction carried by a visit whose owner is replaced before acting, for
  example a wake that fails for good, reaches the successor in neither its instruction nor a
  receipt. The successor's text promises the person's messages "follow as their own messages".
  Same root as PIPE-CHAT-1: the visit instruction is the input's only home. Give accepted run input
  one durable record that the replacement transaction carries. TS-09.R69/R70, INV §2/§19.
- **Must fix** CR-A8-4 — source, easy. `continueRunFromConversation`
  (`internal/server/queued_input.go:117-120`) logs any `ContinueFromOwner` error and reports "not
  continued", so `handleMessageAction` falls through to ordinary delivery, waking a stopped owner.
  An answer over the 64,000-character limit, one that loses the revision check to a simultaneous
  run-page action, or one hit by a store error is delivered as plain chat and shown as sent while
  the run stays paused. The annotation route (`internal/server/sessions.go:353`) shares it. Return
  the refusal to the caller before any write; the bind-failure variant stays with PIPE-CHAT-3.
  FS-14.R100, INV §19/§24.
- **Must fix** CR-A8-5 — source, medium. `OnStageTaskInterrupted`
  (`internal/pipeline/manager.go:218-245`) reads the run outside the run lock and returns nil on a
  lost compare-and-swap; `RecoverStageTask` (`internal/pipeline/decisions.go:257`) returns
  `(false, nil)`. Unlocked writers of the revision include start confirmation
  (`BindPipelineStageTaskStandingAgent`), report acceptance and controller-failure records. The
  loser leaves an interrupted task under a `queued`/`running` run; no sweep covers that pair, and
  Continue, Retry and Replace all require `paused`. `RecoverStageTask` also pauses without
  publishing or notifying when its next step errors. INV §18 names this exact shape. Take the lock,
  re-read and decide again, and add the durable sweep. Occurrence needs an exit racing a revision
  write; not reproduced. INV §5/§18.
- **Worth fixing** CR-A8-6 — source, medium. The stall detector (`internal/server/pipeline_stall.go`)
  counts before the nudge is delivered, logs a refused nudge at Debug and never re-evaluates it,
  and keeps deferred checks only in memory. Its lifecycle-in-flight and not-yet-idle returns have
  no later check, and a background-end event racing the tick can count one stall twice. An idle
  owner with an open visit can stay `running` until a restart. A person's own chat turns do count,
  which the requirement's wording permits. Drive the check from durable state on the tick.
  TS-09.R70, INV §18.
- **Worth fixing** CR-A8-7 — source, easy. Resume instructions on a `controller_recovery_failed`
  pause, where the run page offers the box, and input sent with an approval are accepted and
  dropped (`internal/pipeline/actions.go:115-135`). Resume of an interrupted visit binds the
  receipt before the Resume transaction (`:143-147`); after a failure between them the run-page
  retry at the same revision hits the receipt's uniqueness and a chat retry reports acceptance
  with the run still paused. Refuse or carry the first; commit the second with PIPE-CHAT-3.
  TS-09.R69, INV §15/§19.
- **Worth fixing** CR-A8-8 — source, easy. `confirmTaskStart` (`internal/server/task_dispatcher.go:516-525`)
  swallows a standing-agent bind conflict and logs other failures at Debug; nothing rebinds. The
  run stays `queued`, a dedicated coordinator stays armed, and owner chat is not matched to the
  run. Queued-input release, delivery-state writes and the retained-action records in
  `queued_input.go`, `manager.go` and `task_dispatcher.go` discard or Debug-log their errors the
  same way. Log ownership transitions at Warn and retry the bind from the tick. INV §18/§22.

**Hypotheses, not filed.** Steer skips `admitConversationInput`, and `SteerMessage` claims a turn on
an idle agent, so a Steer that lands after a turn ended reaches a paused or waiting owner outside
run control (lead 11). A follow-up held during the reporting turn is sent before the pause commits.
Released queued input is offered through the recovery card but its release is not logged (lead 12).
Lead 13 has no wait deadline and belongs to CR-A2. The stall interrupt skipping
`notifyTaskWaiters` has no effect: nothing can wait on a stage task.

**Invariant sweep.** Findings touch §2, §5, §8, §15, §18, §19, §21, §22 and §24. §7 was checked
(`ListActivePipelineRuns` skips an unreadable run and continues) with no finding. §1, §3, §4, §6,
§9–§14, §16, §17, §20 and §23 have no surface in this scope; no test changed. The ready changes
`one-typed-turn-outcome` and `match-sent-prompts-by-request-id` resolve none of these; the typed
outcome is the natural input for CR-A8-6.

## Blocked on human

- **First chat setup:** review the planned flow in [Ideas being defined](../ideas.md#ideas-being-defined).
  Confirm whether missing installed CLIs should require one click to use the Chuck bundle and
  whether native configuration linking should remain explicit. The feature draft is in FS-04.R62–R65,
  FS-09.R84 and FS-10.R27. Technical design follows scope confirmation.

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

- **2026-11-13 — Review:** CR-A8 reviewed at `35f33a07`. Eight findings recorded, five Must fix;
  three confirmed by throwaway probes (a restart destroying a held controller action, six report
  refusals answered as `already_reported`, an answer lost on owner replacement). The unit stays
  open at difficult — Codex Sol. PIPE-CHAT-1 and -3 are fixed with CR-A8-3, -4 and -7.

- **2026-11-13 — Design:** drafted first-chat setup with automatic Claude/Codex readiness and one
  provider action in Settings. Scope confirmation is pending.

- **2026-11-13 — Fix:** closed PIPE-CHAT-2 and -4; PIPE-CHAT-1 and -3 stay open on CR-A8, so
  the unit stays open. A failed prompt read (e.g. hydrating a stopped chat) no longer claims a
  withdrawal; only reads after a confirmed withdrawal can, and a failed retry shows a new outcome
  (INV §8/§11). Chat input that continues a paused run now logs request/agent/run/visit, surface
  and delivery without text (INV §15). Both reproductions failed before the fix. Spec deltas:
  FS-03.R90/A70, TS-02.R55. At this fix's tree, `make test` (both Go variants), `make build`,
  UI `npm test` (1246 passed) and UI build pass; focused `-race` of the paused-run answer tests
  passes.

- **2026-11-13 — Fix:** closed MAIL-DETAIL-1–5 and its unit. Sent/read records now stay with
  their calls when a turn completes (INV §1/§2). Records attach only to the one call the run's order
  proves, never FIFO; ambiguous parallel records stay standalone. Codex `args.tool` needs a
  messaging `args.server` (INV §11/§17). Mail detail bounds header and error fields before bodies,
  holding 64 KiB for refused sends (INV §16). TS-08.R161 now matches TS-14.R10, and a capture test
  proves room activity drops mail detail (INV §10). The Mail button is 44px on coarse pointers
  (INV §13). Its emulated-phone Chromium check measured 44×44, tapped, and stayed compact on
  desktop. New tests failed before the fix. Spec deltas: TS-08.R161, TS-04.R108. At this fix's
  tree, `make test` passes both Go variants. UI `npm test` passes 1244 tests with 2 skipped. UI
  build and `make build` pass. A first `make test` run failed only
  `TestDeferredAssignmentResumesOriginalAgent/retry_failure` ("lifecycle transition already in
  progress"). That lifecycle test is outside this diff; it passed 5/5 alone and on the full rerun.
  Watch it for flakiness. Owed: end-to-end re-render and physical phone tap (A33 gate above).

- **2026-11-13 — Review planning:** queued two subsystem reviews, CR-A8 (pipeline pause, answer and
  recovery) and CR-A9 (Think Tank lifecycle and quota join), chosen from the October audit's scope
  and fix history plus fresh Sonnet code maps at `e8a8a128`. Charters carry ten and eleven
  unreproduced leads; four were checked against source by the planner. A third candidate, one
  transition rule per durable state machine, is a design topic for after these reviews and is not
  queued. No product code, specification or finding changed.

- **2026-11-13 — Design:** five single-owner refactors from a review of 60 days of invariant
  evidence and fix history are waiting to start as separate ready changes: typed turn outcome
  (FS-03.R117/A92, TS-01.R42), request-id prompt echo (FS-03.R118/A93, TS-08.R163), shared
  desktop/phone input delivery (TS-08.R164), shared resume start and teardown in Switch
  (TS-01.R43), single-owner turn state (TS-01.R44, after R42). All planned; TS-08 is now Partial.
  Operator decisions: five changes not one; the phone gains no local echo; delivery records for
  the launch assignment prompt and stall nudge are out of scope. No product code changed.
  `make check-specs`, twin-skill comparison and `git diff --check` pass. Not traced: whether the
  Think Tank launch path holds a lifecycle claim.

- **2026-11-12 — Fix:** closed the QUOTA-FORK review (QUOTA-FORK-1–7; INV
  §2/§5/§7/§8/§10/§11/§16/§17). One Go decoder reads the SDK's Claude usage shape
  (`model_scoped` array, unknown members); an undecodable success is a failed read keeping
  earlier windows; `model_scoped` windows count by family. The Claude adapter patch forwards only
  window members (new pinned hash, still `+chuck.2`, unreleased). Only admission from `scheduled`
  pre-checks the snapshot. Edit on a mid-turn message is disabled with its reason (API refuses
  likewise); copied turns say so. First-read failure wording, phone summary 44px, A40/A41/A49/A52
  coverage stated. Both Go variants (`go test ./...`, `-tags sqlite_fts5`), `make check-specs`,
  `make build`, 1,240 UI tests and the UI build pass; one earlier full run timed out
  `TestQuotaStoppedBorrowedTaskDefersUntilRuntimeCapacity` (bypasses the changed dispatcher;
  5/5 alone and green in the rerun — load-sensitive, watch it); the patched
  bundle contract tests pass against the pinned 0.85.1 input. Not run: `make dist`, hosted CI.

- **2026-11-12 — Fix:** closed ANNOTATION-MAIL-HEADER-1 (INV §8/§17). Single-message mail
  notices show the sender once; batch attribution and expansion remain intact. The saved and
  expandable regression cases failed before repair and pass afterward. `make check-specs`, UI
  tests (1,230 passed, 2 skipped), and UI build pass.

- **2026-11-12 — Work:** shipped expandable agent mail (FS-06.R45/A33, TS-04.R108, TS-08.R161;
  INV §2/§8/§10/§11/§13/§15/§16). Received inline mail keeps a clipped copy in its notice; sends
  (including refusals) and reads append a display-only notice to the caller's live chat
  transcript; the UI attaches them to the matching tool line. Agent-facing projections are
  unchanged (notice metadata disposition). `make test`, 1,220 UI tests and the UI build pass.

- **2026-11-12 — Fix:** closed the Think Tank templates review (TT-TEMPLATE-1–10; INV
  §1/§3/§5/§7/§8/§9/§10/§11/§14/§15/§16/§17). ACP mapping strips echoed
  `participant_instructions` before transcript/history/SSE. Migration 73 stores template version
  and replaces source ids in intent with a request fingerprint (TS-14.R30–R32 updated). Room
  save reads roster only and accepts members switched to Terminal. UI fences late template loads
  (edits and blank), retains save command ids for lost-response retry, flags unsupported saved
  effort/fast, pages templates on demand, and counts instruction runes. `make test`, 1,211 UI
  tests and the UI build pass. TT-5 is exercised at the state layer, not the switch route.

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
  expandable-agent-mail-activity
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
