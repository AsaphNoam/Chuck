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
  available after mailbox expiry and archive reload. Turn completion detached the records from
  their tool calls. Recheck completed, archived and phone activity after repair. The physical
  paired-phone check remains owed.

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

- **MAIL-DETAIL — expandable agent mail in transcript activity (`a44abd23`).** Independent review
  found defects. Fix the grouped findings below before closing the unit.

**Core reliability audit units (added 2026-11-13):** explicitly requested subsystem reviews of the
complete current implementation and callers, not of a commit. Read the charter, refresh fix
ownership and record the reviewed HEAD. Intended for a frontier reviewer; the charters' leads are
questions, not findings.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A8](CORE-RELIABILITY-REVIEW-A8.md) | Pipeline pause, answer and recovery ownership: who can end each pause, refused input with no side effect, answers through replacement, stall detection | waiting for review; run first |
| [CR-A9](CORE-RELIABILITY-REVIEW-A9.md) | Think Tank lifecycle and its quota join: fenced member launch, requested Pause/End under quota, restart recovery | waiting for review |

PIPE-CHAT-1 and -3 sit on CR-A8's contract; CR-A8's result says whether they are fixed alone or
with its findings. PIPE-CHAT-2 and -4 are independent.

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

### Expandable agent mail (MAIL-DETAIL) — **Fix model:** medium — Codex Terra or Claude Opus.

- **Must fix** MAIL-DETAIL-1 (INV §1/§2/§10/§17, easy) — Completing a turn separates sent and read
  notices from their calls. `turnActivity.ts:117–135` hides tool calls and results while leaving the
  notices in the reading path. `TurnList.tsx:67–72` groups each segment independently, leaving bare
  tool lines and separate **Chuck mail** runs. Reproduced in Chromium on live completion, the
  authenticated phone view and Archive. Keep outgoing mail in the corresponding tool activity and
  keep received notices in the reading path. Check the same disclosure before and after turn
  completion and after reload (FS-06.R45/A33, TS-08.R161).
- **Must fix** MAIL-DETAIL-2 (INV §5/§8/§11/§17, medium) — `toolRun.tsx:45–74` pairs records by tool
  name and first free call. Deny one send and approve the next in the same run. The denied call
  never reaches MCP, so the successful send's recipient and body attach to the denied line. Two
  parallel sends whose records arrive out of call order also swap their details. Both schedules
  were reproduced against production grouping. The matcher additionally accepts `args.tool` from
  an unspecified server. Associate records using trustworthy call evidence and leave ambiguous
  records standalone. Align TS-08.R161's FIFO rule with FS-06.R45/A33. Cover denied, reversed and
  missing records and an unrelated tool with the same argument.
- **Must fix** MAIL-DETAIL-3 (INV §8/§11/§16/§17, easy) — Refused sends capture raw subject and
  recipient fields (`internal/messaging/tools.go:154–163`). `internal/runtime/event.go:257–273`
  can only drop bodies, so oversized headers or a refusal quoting a long address can still exceed
  64 KiB. The real MCP and transcript API reproduced 71,909 bytes from a 70 KiB rejected subject,
  even with `truncated` set. Bound rejected fields and the quoted display error, then verify the
  final serialized byte limit while preserving the original tool refusal (TS-04.R108).
- **Worth fixing** MAIL-DETAIL-4 (INV §10/§14, easy) — TS-08.R161 promises mail disclosure in Think
  Tank activity. `internal/server/think_tank_capture.go:32–36` excludes notices. TS-14.R10
  protects private mail from room storage, REST and SSE. Reconcile the presentation spec with that
  boundary and verify room activity remains a receipt. The ordinary participant transcript can
  still show its own mail detail.
- **Worth fixing** MAIL-DETAIL-5 (INV §8/§13, easy) — The new Mail disclosure inherits the text-size
  `tool-toggle` hit area (`MailDetail.tsx:20`, `agent.css:1240`). At 390px on a touch viewport the
  received-mail button measures about 43 × 17 CSS pixels. Give the new disclosure a 44px touch
  target while keeping its compact visible label, and verify by tap on the phone (FS-12.R68/A38).

The legacy server aliases and standalone label are reasonable compatibility and display choices.
Empty reads display **No messages returned**. Pairing still needs call identity.
Focused Go and UI tests, the built UI and its style gates passed. The invariant sweep found no new
registration lifecycle, runtime interface, or external CLI invocation (§4/§6/§12).

### Blocked pipeline chat and withdrawn-message recovery (PIPE-CHAT) — **Fix model:** difficult — Codex Sol.

Field report, 2026-11-12. The affected run, conversation, build and provider are unidentified.
The report supplied no request log or provider receipt. The local dashboard has no paused run,
and its retained run and input metadata did not identify the reported submission.

> /investigate-bug messages sent though the chat to a pipeline orchestrator when it's blocked sometimes disappear, I think chuck treats them like tool calls (as if they came from the pipeline continue/send message box) because when it needed my explicit approval the agent said it came through as a system call through chuck so the approval was rejected. Also I have the Withdraw message box on but empty:Withdrawn, but the message and its attachments could not be loaded. Load withdrawn message (button doesn't do antyhing)

The paused-run chat branch came from `376add97`. The shared recovery reader took on background
reads in `0e98d3c4`. The HTTP reproduction reports the answer sent without a durable user event.
Both withdrawal reproductions fail on current code. All three are committed skipped for the fix.

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
- **Must fix** PIPE-CHAT-2 — confirmed, trivial/easy. `ui/src/lib/heldMessage.ts` marks any failed
  prompt read as a failed withdrawal recovery, including `ChatPanel.tsx` hydration for a stopped
  chat. `internal/server/sessions.go` returns 404 when a stopped chat has neither a live hold nor
  queued or recoverable input. `TranscriptView.tsx` then displays a withdrawal that never occurred.
  The retry issues another GET, but a repeated failure leaves the same alert and control. Tie the
  notice to a confirmed withdrawal and show the outcome of retry. Both failure cases were reproduced
  in `TranscriptView.test.tsx`; a transient failure with a successful retry is already covered.
  FS-03.R90/A70, TS-08.R56, INV §1/§8/§11.
- **Must fix** PIPE-CHAT-3 — confirmed by source, difficult. `internal/pipeline/actions.go` creates
  and publishes a successor in `internal/pipeline/decisions.go` before its separate receipt bind.
  Admission can observe the visit before that bind. A bind error after the visit commits is logged
  and converted to ordinary standalone delivery in `internal/server/queued_input.go`, even though
  the run has already advanced. Commit the successor and input receipt together before publication.
  Inject a bind failure and race admission/retry against acceptance, verifying that no input is
  lost, misrouted or delivered twice. TS-02.R55, TS-09.R69, INV §5/§15.
- **Worth fixing** PIPE-CHAT-4 — confirmed observability gap, trivial/easy. The paused-run branch in
  `internal/server/queued_input.go` bypasses the runtime's receipt transition logging. Request logs
  alone cannot distinguish a chat Send accepted into a new visit, a chat turn, or input still queued.
  Record bounded request/agent/run/visit identities, input surface and delivery transition, so the
  operator can trace acceptance to provider submission without storing message text, attachment
  metadata or raw provider errors in logs. TS-02.R55, TS-09.R69, INV §8/§11/§15.

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
