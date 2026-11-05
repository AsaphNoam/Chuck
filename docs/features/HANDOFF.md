# Chuck — Implementation handoff

Settled history through v0.21.0 is archived in
[HANDOFF-through-2026-10-31](../archive/state/HANDOFF-through-2026-10-31.md); earlier epochs are
linked from there. Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archived snapshots
are not live queues.

## Current position

- **Published:** [v0.21.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.21.0),
  2026-11-01, tag at `d3fda601` (goal-led pipelines, run worktrees, Boring Night skin, one-line
  tool activity, running background-command indicators).
  [Notes](../archive/state/RELEASE-v0.21.0-notes.md), also the release body. Release CI
  41222233162 passed on attempt 2 (attempt 1 was cancelled by GitHub's `macos-14` brownout);
  downloaded archive size 325336221 and SHA256 `36c955dd…f24ac` match the manifest.
- **Release runner:** `.github/workflows/release.yml` still uses `macos-14`, which GitHub retires
  2026-11-02; the next release cannot assemble until it moves to `macos-15` (operator decision
  pending).
- **Main CI:** red on CI-CARDGRID-1 (runs 41222233023 and 41223054609); see Review findings.
- **Review state:** CI-CARDGRID and TASK-FORM are open for `/fix`. The v0.21.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.20.0-next.json)) is closed (ARCH-NEXT-1–4 fixed;
  generated embed covered).
- **Field investigation:** RECOVERED-QUEUE is open for `/fix`: a transient uncertainty snapshot can
  survive successful Steer; text withdrawal also duplicates the restored draft as recovery. Fresh
  queue/normal delivery do not reproduce the broader report; exact attribution is undetermined.
- **Pipeline field investigation:** EFFORT-BUDGET awaits the affected worker's manifest and trace.
  Cause and fix are undetermined; no validated `/fix` unit or prompt change is recommended.
- **Open field causes:** the frequent-429 provider failure and the unchecked permission-save
  toggle remain unmeasured; each needs affected server evidence if it recurs.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`. Clean release verification may use a
  detached worktree at `/private/tmp/chuck-release-<version>`.
- **Future work:** direct-action migration remains paused on its transport blocker; passive
  quota snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Full suites need loopback-capable execution. Native/device/credentialed limitations remain in
  the owed gates.

## Active change

None.

## Acceptance gates still owed

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

- **Quiet task coordination** (2026-11-05): FS-16.R49 / FS-14.R76 / TS-09.R49 — shared task
  guidance keeps routine progress and user-requested adjustments in final results, preserving
  earlier mail for action, coordination risks and decisions with deferred FYIs. Dedicated
  coordinators no longer receive a blanket request to report progress; bundled references agree.
  `internal/runtime/activation_kinds.go`, `internal/pipeline/manager.go` and the operating-chuck
  references. Both Go variants and tagged build passed; live provider adherence remains unverified.

- **Self-targeted annotations behave like chat messages** (2026-11-02): FS-13.R37 / TS-03.R78 —
  Current agent is available for any non-archived source; the handler wakes a stopped source,
  holds behind a running turn, or steers (`target.steer`) when the runtime advertises it; desktop
  tray and phone form offer **Steer** while busy. R23 suppression now pairs each block prompt with the
  nearest earlier unpaired self card (`prompt_seq` lets ChatPanel clear the held tail). Review note: a
  queued batch replaces any held chat message, per FS-03.R48. `internal/server/sessions.go`,
  `ui/src/components/chat/annotationDelivery.ts`, `ui/src/store/transcriptStore.ts`.

- **Pipeline stage-work cards link to their agents** (2026-11-02): restores FS-14.R39/A17/A23
  linking lost in the task-only cutover — each run-page "Stage work" card opens `/agent/{id}` while
  live, `/archive/{id}` once stopped (live store overrides the detail route), and stays a
  non-linking card without an agent identity. `ui/src/features/pipelines/RunBrowser.tsx`.

- **Think Tank setup fit and annotation toast removal** (2026-11-01): setup dialog widens to
  1120px with participants beside the goal at >=960px; the desktop and phone "Annotations sent to
  …" notices and `sentTo` delivery state are gone (FS-13.R29/R35 wording updated).

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

### RECOVERED-QUEUE — recovered messages after queue use (2026-11-05) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “the 'recovered message' feature shows a recovered message whenever you use
the queue (send a message while the agent is mid run) regardless of if you ended up leaving it
there, steering or withdrawing.” Reported version, agent identity, desktop/phone surface and exact
timing were not supplied; no logs were attached. Investigated source at `a674e6bc` on macOS;
read-only local `$CHUCK_HOME` receipt/transcript/dashboard evidence was also inspected, but the
report did not identify one of those conversations as the affected example.
**Clarification (verbatim):** “message I just queued, agter the turn ended.” The reported card
contains the newly queued text, rather than an older message; exact agent/time remain unknown.
The clean natural-turn-end fixture does not reproduce that path.

**Expected behavior / classification:** FS-03.R48/R49/R50 require pending input to remain held,
then become delivered once, including promotion through Steer. R90 restores withdrawn text to the
composer; R100 extends durable explicit recovery to released/uncertain text receipts. These paths
need coherent recovery ownership without dropping genuinely uncertain input. Successful-Steer
stale recovery is a code defect; withdrawal exposes a contract gap between draft restoration and
retained recovery that needs a coordinated fix. The broader claim that every queue creates recovery
is **undetermined**, not established by the available evidence.

- **Must fix** — **Confirmed; code/spec interaction** — **RECOVERED-QUEUE-1: text restored after withdrawal remains offered
  as a separate recovered message.** FS-03.R90/R100; TS-03.R74; INV §1/§8/§11/§15.
  `internal/runtime/chat.go:658` marks the withdrawn receipt `released`;
  `ui/src/lib/heldMessage.ts:25-35` restores its text to the composer but never acknowledges that
  recovery record. `internal/state/attachment_receipts.go:90` selects it on the next GET, and
  `ui/src/components/chat/ChatPanel.tsx:135-146` / `TranscriptView.tsx:235-238` show a separate
  “Message not delivered” / “Send recovered message” offer. Phone has the same split at
  `ui/src/remote/AgentScreen.tsx:908-925` and refreshes after withdrawal. Trigger: queue text,
  successfully withdraw, then refresh the recovery snapshot (or let the current turn finish).
  The composer already contains the returned input; the extra offer survives reload and unrelated
  sends, inviting duplicate submission. Coordinate acknowledgement of the exact withdrawn request
  with successful draft restoration; preserve recovery when the response/draft handoff fails,
  preserve unrelated older uncertain receipts and attachment recovery, and do not clear all recovery
  merely because a queue or Steer succeeded. Regression: real released receipt on successful
  withdrawal, retained restored draft, refreshed snapshot, and one explicit resend decision.

- **Must fix** — **Confirmed; code defect** — **RECOVERED-QUEUE-2: a transient recovery snapshot survives
  successful Steer.** FS-03.R50/R100; TS-03.R74; INV §1/§8/§11. During promoted Steer,
  `internal/runtime/chat.go:795` records `unknown` before awaiting the native reply; GET prompt
  legitimately exposes that uncertainty. A ChatPanel read on turn-end/state change can cache it.
  Successful Steer sets the receipt `submitted` at `chat.go:828`, but
  `ui/src/components/chat/Composer.tsx:335-348` releases held state without refreshing the recovery
  snapshot. `heldStore.ts:46-66` preserves recovery independently. The success notice and a
  “may already have reached the agent” recovery offer then coexist for that same request until
  another qualifying refresh. The real delayed-ack fake-provider probe confirms GET returns
  unknown during the native call and no recovery after success; the actual ChatPanel/Composer
  reproduction fails because the already-loaded card survives that success. Reconcile matching
  recovery with the authoritative successful receipt, guard older in-flight reads, and preserve
  unrelated recovery. Cover turn end during Steer and a delayed read completing after success.

- **Worth fixing** — **Confirmed; observability gap** — **RECOVERED-QUEUE-3: request logs cannot attribute a recovery
  card to a queue transition.** Investigation §12.5; TS-02.R55 (no invariant class).
  Ordinary request logs identify
  method/path/status, while `messageTransition` (`internal/runtime/message.go:38-48`) and
  `SetAttachmentDelivery` (`internal/state/attachment_receipts.go:73-75`) retain no transition
  diagnostic. Current receipts hold only the latest state/timestamp, so the available logs cannot
  establish whether the displayed copy was the just-held input, a replaced/withdrawn receipt, or
  an older provider failure. Add bounded structured diagnostics with agent/request identity,
  operation, delivery transition and reason at acceptance/release/submission/uncertainty; never
  log message bodies or attachment contents. Verify queue, replacement, withdrawal, successful
  Steer and provider-error diagnostics identify the exact receipt without leaking payloads.

**Evidence and limits:** a fresh fake-provider HTTP/runtime matrix confirms no `recovery` while
held, after natural delivery, or after successful Steer; withdrawal returns a `released` recovery,
and replacing a hold retains the older released input. All four probe cases pass their observed
state assertions; an additional delayed-ack Steer case passes. Replacement retention is consistent
with R100 and is not independently labelled a defect. Two actual ChatPanel/Composer reproductions
fail at the intended no-duplicate/no-stale-card assertions, independently rerun and retained skipped
in `ui/src/components/chat/RecoveredQueue.test.tsx` (finding IDs in test comments). All 138 existing
Composer/ChatPanel/TranscriptView tests pass; their mocked HTTP responses do not cover these
handoffs. Backend probes and UI execution logs are under `/private/tmp/queue-recovery-*.log` and
`/private/tmp/recovered-ui-independent.log`; probe source/overlay is in the investigation workspace.
Local persisted evidence has 37 submitted receipts,
six unknown (five dismissed), and two dismissed released receipts; no held receipts at inspection.
Two recent unknown receipts in `a_e9646f` match logged provider content-filter errors at
2026-11-05 08:31:54Z and 08:59:26Z, after canonical user events and provider output. They establish
another source of recovery, not attribution of this report. Unknown recovery itself is required by
R100. History: `2268e279`/`07599b77` extended media receipts to text without adding a withdrawal
draft acknowledgement. No product code, specs, user state, provider sessions or unrelated dirty
files were changed. Exact affected request/timing remains needed to confirm the broader symptom.

### TASK-FORM — Standard task creation form (`17e050fb`) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

- **Must fix** — **TASK-FORM-1: server prerequisite validation leaves correction fields collapsed.**
  INV §8; FS-16.R59/A41 and R40/R45. `ui/src/features/tasks/taskForms.tsx:440` only sets the
  server error, whereas the local validation branch opens Advanced. Enter a missing prerequisite
  ID (or select a source deleted before submission), collapse outer Advanced, then Create task:
  the server's typed `unusable_source` refusal appears below the form while the prerequisite fields
  remain hidden. `internal/state/tasks.go:670` produces this refusal and
  `internal/server/task_handlers.go:445` maps its typed envelope. Open the affected disclosure for
  relevant server validation codes and retain the draft; add a collapsed-submit refusal regression.
  Independent Chromium reproduction with that actual error envelope confirms `advancedOpen: false`
  and `sourceVisible: false`; local missing-outcome validation does reopen Advanced.
  Evidence in shared resources: `task-form-review-20261104-probe/dependency-recovery.json`,
  `dependency-recovery.png`, `local-dependency-recovery.json`, and `task-form-review-probe.mjs`.

**Review evidence (2026-11-04):** reviewed the implementation and administrative follow-up
  `23ef3a34`; this same unit remains open for the finding above. Focused Tasks tests pass 46/46;
  `make check-specs`, `npm run check:styles` (47 checker tests plus presentation contract), and
  `git diff --check` pass. Independent approved loopback Chromium run of the actual source UI
  covers all four appearances at 1024/1440, the development matrix, timed request fields,
  Advanced draft retention and visible API errors. All eight task-form captures inspected without
  clipping or escaped controls. `task-form-review-20261104/report.json` and screenshots retain the
  evidence; API routes are stubbed, so this establishes frontend behavior rather than live dispatch.
  Invariant sweep: §§1/2/8/10/13/17 apply (project/runtime reset, shared builders, recovery,
  task/schedule wiring, selectors and independent request assertions); only §8 has a finding.
  §§3/4/5/6/7/9/11/12/14/15/16 have no changed applicable surface. New tests protect requirement
  outcomes at the rendered component/request boundary; TS-06.R35/INV §17 reveal no separate
  test-value finding. Shared runtime helpers and native details retain the intended local choices.
  Product code/specs and unrelated dirty `docs/ideas.md`/generated UI index are preserved.

### CI-CARDGRID — release v0.21.0 clean-clone CI — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

- **Worth fixing** — **CI-CARDGRID-1: collapsed-section auto-open test fails in CI.** INV §1/§17. `ui/src/components/grid/CardGrid.test.tsx` "opens nothing for
  terminal, cross-project, and collapsed-section agents" fails on GitHub `ubuntu-latest` (main CI
  41222233023 on `d3fda601`, and 40232900275 on handoff-only `43c214d1`) with
  `Composer a_hidden` present, while it passes locally (1/1). The collapsed group's layout likely
  hydrates after the `waiting_input` update on slower runners, so the automatic opening fires for
  a collapsed-section agent. Decide whether this is a test-ordering race (wait for the collapsed
  group before the update) or a product race against FS-02's no-open rule for collapsed sections;
  add a deferred-layout regression either way. Not introduced by the v0.21.0 range. TS-06.R8
  clean-clone CI is red until fixed.

## Changelog

- **2026-11-05 — Pipeline effort/budget investigation, corrected:** the worker reportedly inferred
  an additional budget from its own config. Withdrew the unsupported guidance-gap diagnosis and
  `/fix` recommendation; absence of a specific prohibition does not establish a prompt defect.
  Cause/remedy remain undetermined pending the worker's assignment, manifest and stop explanation.
  Effort metadata and genuine recovery rounds are independent; two existing recovery tests pass
  but do not reproduce this behavior. No product/spec change.

- **2026-11-05 — Queue recovery investigation:** RECOVERED-QUEUE records confirmed withdrawal
  duplication, successful-Steer stale recovery and missing receipt-transition diagnostics. Two UI
  reproductions fail when enabled and are retained skipped; five real-handler/fake-provider probe
  cases pass. Existing focused UI tests (138), TypeScript, spec integrity and diff checks pass.
  Natural queued delivery remains unreproduced; no live mutation/provider call or product/spec
  change. Unrelated ideas and generated UI edits preserved.

- **2026-11-05 — Quiet task coordination:** `make test` (both Go variants and spec checks),
  `make build`, link-target and `git diff --check` passed for this unit; Go commands used
  `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 GOCACHE=/private/tmp/chuck-go-cache`, tests with
  loopback-capable execution. Logs: `/private/tmp/chuck-quiet-coordination-{test,build}.log`.
  Existing prompt-delivery coverage reused; no claim of live pipeline noise reduction. Unrelated
  `docs/ideas.md` and generated UI edits preserved.

- **2026-11-02 — Self-targeted annotations wake, queue and steer:** checks on the working tree:
  `make test` (both Go variants, check-specs), `make build`, full UI vitest (1,131) and UI build,
  `git diff --check`; no rendered browser/phone journey.

- **2026-11-02 — Pipeline stage-work cards link to agents:** checks: check-specs, full UI vitest
  (1,129) and UI build on the working tree; no Go change.
- **2026-11-01 — Think Tank setup fit, no annotation-sent notice:** checks: tsc -b, check:styles,
  check-specs, 517 vitest (chat, remote, thinktank); Playwright render at 1280×800 with three
  participants and a judge shows no dialog scroll.
- **2026-10-31 — Released v0.21.0:** readiness clean (no open units or findings); the
  operating-chuck package already matched the range (goal-led decisions, run worktrees, queued
  owner input and concise coordination guidance landed with their commits); README and pinned
  components unchanged. Reused full Go on `ed1da637` and 1,128 UI tests on `0d6d877c` (inputs
  unchanged). `make dist VERSION=0.21.0` passed (`/private/tmp/chuck-dist-0.21.0.log`); embed
  committed `a3cc6ec9` with receipt coverage. Settled history moved to the 2026-10-31 archive.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest notes: [v0.21.0](../archive/state/RELEASE-v0.21.0-notes.md) (previous [v0.20.0](../archive/state/RELEASE-v0.20.0-notes.md)).
Earlier publication/campaign records remain in the archived handoff.
