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
- **Agent-surface review backlog:** ASR-A (`4fbeb316`), ASR-B and ASR-C (`6b5de19b`), and
  ASR-D (`c12c4251`), ASR-E (`5eb574d9`), ASR-F (`b8157f13`), ASR-G (`fa092f63`) and
  ASR-H (`d49a8f86`)
  have been audited and have open findings; their units remain under
  [Review queue](#review-queue). ASR-I's whole-system source synthesis is complete below. The
  [charters](AGENT-SURFACE-REVIEW.md) combine interacting mechanisms and require cross-system
  joins plus a final synthesis/performance review. All original 25 research areas remain in the
  [search catalogue](AGENT-SURFACE-SEARCH-CATALOGUE.md), not as separate tasks. Their
  evidence and findings are below; all 25 topics and seven joins have been reconciled, with
  native/device adoption and comparative performance still unproved. Other findings and live gates
  remain independent.
- **Designed agent-surface work/fix units:** [AGENT-SURFACE-FIX-UNITS.md](AGENT-SURFACE-FIX-UNITS.md)
  assigns the original 41 findings once: three `/work` refactors (ASW-1 event commit, ASW-2 frozen
  native execution, ASW-3 process/driver ownership) and six consolidated `/fix` batches below.
  The human explicitly authorized these cross-review batches; for this campaign select the new
  unit, not all findings from its old review. Planned FS/TS requirements, acceptance tests,
  migration/security bounds and shared-file coordination are designed. Prioritize unsafe process
  control, ambiguous replay and the config read-loop stall. ASW-1 is reviewed with seven Must-fix
  findings and one documentation finding open below; ASW-3's implementation
  is reviewed with five findings open below; other units remain waiting. The
  [systemic assessment](AGENT-SURFACE-FIX-STRATEGY.md) retains the rationale.
- **Prompt/harness investigation:** ASR-A reopens PROMPT-1 for ancestor-project Codex instruction
  loss; its earlier fix covers user/exact-cwd configuration. The original unit below retains the
  prompt-composition diagnostic gap (PROMPT-2) and separate process-policy recommendations.
  General output-quality regression remains unproven.
- **Former known risk confirmed:** the `SetSessionConfig`/`as.mu` read-loop stall is now ASRC-1
  under Review findings.
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

- **ASW-2 in progress** (2026-10-20) —
  [`preserve-native-execution-contract.md`](../ready-changes/preserve-native-execution-contract.md).
  Slices: (1) versioned execution contract (TS-02.R53) through launch/resume/switch/rollback/fork
  with empty-override normalization — ASRA-1/2; (2) unsupported root/profile preflight — ASRA-3;
  (3) fork purpose: Codex fork omits Chuck overlay, Claude fork loads the returned ID — ASRB-2/3;
  (4) quota-check instruction sanitization — ASRG-4; (5) Codex ancestor project layers —
  PROMPT-1; (6) Steer artifact proof — ASRB-8; (7) bounded execution manifest — PROMPT-2.
  Slices 4 and 5 are delegated to isolated worktrees and integrated here.
  Slice 1 done: TS-02.R53 contract + `continuedContract`; ASRA-1/ASRA-2 fixed pending review
  (`internal/server/contract_test.go`, `internal/index/contract_test.go`).
  Slice 4 done: ASRG-4 fixed pending review (Codex check blanks instructions; refuses
  instruction-file/profile sources; live adoption of the blank override unproved).
  Slice 5 done: PROMPT-1 ancestor variant fixed pending review (`project_root_markers` chain,
  default `.git`; pinned binary strings + reviewer probe; nearer/custom-marker cases fixture-only).
  Slice 3 done: ASRB-2/ASRB-3 fixed pending review (fakeacp `FAKEACP_FORK_UNREGISTERED` mirrors
  the pinned Claude fork; `fork_purpose_test.go`); credentialed clone receipts still owed.
  Slice 6 delegated (Steer artifact proof). Next: slice 2 (root/profile preflight), then 7.
- ASW-1 (conversation event commits) reviewed 2026-10-20; ASW1-1–8 below keep that
  same unit open for fixes. The original sequence/order/reconciliation reproductions are corrected.
- ASW-3 (process and terminal-driver ownership) reviewed 2026-10-19; five Must-fix findings
  keep the unit open (Review findings). FS-07.A12 real tmux/iTerm2 receipts are owed.

## Acceptance gates still owed

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

## Agent-surface work and fix queue

**ASW-1 and ASW-3 reviewed with findings open; ASW-2 in progress; all other units waiting.** The complete [unit designs and primary finding map](AGENT-SURFACE-FIX-UNITS.md)
are the selection authority for this campaign. Three substantive refactors are ready in
[`docs/ready-changes/`](../ready-changes/README.md):

| Unit | Invocation | Outcome |
|---|---|---|
| ASW-1 | reviewed 2026-10-20 — findings open (ASW1-1–8) | One canonical commit boundary, ordered control observations and truthful history failure |
| ASW-2 | `/work preserve-native-execution-contract` | Frozen effective native selection and explicit fresh/resume/fork/check preparation |
| ASW-3 | reviewed 2026-10-19 — five findings open (see Review findings) | Process/terminal ownership and native terminal identity; incomplete fail-closed control and launch-hook ordering |

The following cross-review §8 batches are explicitly authorized by the human's consolidation
request. Their exact scope, required specs, regressions and closure rules are in the linked design.
Keep original finding IDs and evidence until each closes; do not close an originating review just
because one new batch finished. Original review aliases must respect this map and must not
reimplement an ASW refactor as a local fix. The existing launcher/workflow remain otherwise unchanged.

| Fix unit | Invocation / responsibility | Owned findings | Status / fix model |
|---|---|---|---|
| [ASF-1](AGENT-SURFACE-FIX-UNITS.md#asf-1--turn-and-control-settlement) | `/fix ASF-1` — settings, status, permission and cancellation settlement | ASRC-1, ASRC-2, ASRC-5, ASRG-3 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-2](AGENT-SURFACE-FIX-UNITS.md#asf-2--input-acceptance-and-recovery) | `/fix ASF-2` — annotation receipts, drafts, exact withdraw and uncertain Steer | ASRD-4, ASRH-1, ASRH-2, ASRH-4 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-3](AGENT-SURFACE-FIX-UNITS.md#asf-3--durable-continuation-and-controller-recovery) | `/fix ASF-3` — task/pipeline/quota recovery and obsolete executor removal | ASRE-1, ASRF-1, ASRF-2, ASRG-1, ASRG-2, ASRG-5 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-4](AGENT-SURFACE-FIX-UNITS.md#asf-4--retention-and-result-bounds) | `/fix ASF-4` — native-child retention and task-result bounds | ASRC-6, ASRE-2 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-5](AGENT-SURFACE-FIX-UNITS.md#asf-5--project-readiness-and-resource-permissions) | `/fix ASF-5` — checkout readiness and restrictive resource modes | ASRA-5, ASRA-6 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-6](AGENT-SURFACE-FIX-UNITS.md#asf-6--cli-archive-selection) | `/fix ASF-6` — certain CLI resume/new selection | ASRH-3 | Waiting — medium, Codex Terra or Claude Opus |

## Review queue

Nine integrated reviews are sized for frontier reviewers to follow interacting mechanisms and
whole workflows. These are explicitly requested subsystem audits, not reviews of queue commits.
ASR-I's source synthesis is complete; the eight domain units below remain open on their findings.
Their remediation is now assigned to the work/fix units above; these rows retain review provenance,
not a second competing implementation queue.
Read the selected [charter and mandatory joins](AGENT-SURFACE-REVIEW.md), then its retained
[search topics](AGENT-SURFACE-SEARCH-CATALOGUE.md); inspect the complete current implementation and
callers, not only a commit diff. Numeric ASR-01–ASR-25 aliases now select their whole containing
lettered unit, not a separate miniature review. The plan maps every original topic to one group.

Any row is selectable independently. Mark only the selected row **in progress**; findings keep
that same integrated unit open. Record topic coverage, interacting scenarios, findings and evidence
limits together. Remove a row when its review is finished, retaining findings/live gates and a
changelog record. ASR-I can start early, but its final synthesis must reconcile A–H evidence and
J1–J7 before the campaign is described as reviewed. Do not select fixes as part of this backlog.

| Unit | Responsibility | Status |
|---|---|---|
| [ASR-A](AGENT-SURFACE-REVIEW.md#asr-a--instructions-configuration-and-execution-environment) | Instructions, configuration and execution environment | reviewed — findings open (`4fbeb316`) |
| [ASR-B](AGENT-SURFACE-REVIEW.md#asr-b--codex-and-claude-native-integration-fidelity) | Codex and Claude native integration fidelity | reviewed — findings open (`6b5de19b`) |
| [ASR-C](AGENT-SURFACE-REVIEW.md#asr-c--runtime-execution-concurrency-and-control) | Runtime execution, concurrency and control | reviewed — findings open (`6b5de19b`) |
| [ASR-D](AGENT-SURFACE-REVIEW.md#asr-d--conversation-input-and-history-fidelity) | Conversation, input and history fidelity | reviewed — findings open (`c12c4251`) |
| [ASR-E](AGENT-SURFACE-REVIEW.md#asr-e--agent-tools-communication-and-durable-delegation) | Agent tools, communication and durable delegation | reviewed — findings open (`5eb574d9`) |
| [ASR-F](AGENT-SURFACE-REVIEW.md#asr-f--pipelines-think-tanks-and-orchestration) | Pipelines, Think Tanks and orchestration | reviewed — findings open (`b8157f13`) |
| [ASR-G](AGENT-SURFACE-REVIEW.md#asr-g--failure-quota-and-continuation-recovery) | Failure, quota and continuation recovery | reviewed — findings open (`fa092f63`) |
| [ASR-H](AGENT-SURFACE-REVIEW.md#asr-h--human-intent-across-desktop-phone-cli-and-http) | Human intent across desktop, phone, CLI and HTTP | reviewed — findings open (`d49a8f86`) |

**Implementation review units (independent of the table above):**

- **ASW-1 — conversation event commits** (`3f746c2a`, `69ef5951`, `1843470f`, `57af7f61`,
  closure `43da6bec`) — **reviewed; findings open**. Reviewed `3618df3e..43da6bec`;
  ASW1-1–8 below retain this same unit for `/fix`. Original ASRD-1/2/5 reproductions are corrected;
  ASRD-3's history suppression is corrected but its recovery/control obligations remain incomplete.
  ASRD-4 stays with ASF-2. Concurrent ASW-2 edits were excluded by finishing the probes in
  `/private/tmp/chuck-asw1-review-43da6bec`.

- **ASW-3 — process and terminal-driver ownership** (`2609e903`, `3e7df682`, `e2cf34f8`,
  `9a0bf119`, closure `3618df3e`) — **reviewed; findings open**. Reviewed
  `a3bd6015..3618df3e` in an isolated snapshot; ongoing ASW-1 work is outside the unit.
  ASW3-1–5 below retain this same unit for `/fix`; original ASRB-1 and ASRC-3 remain partially
  resolved. FS-07.A12 real-driver receipts remain separately owed.

## Review findings

The `NewAgentModal`/phone gap-test race fix is confirmed remotely (main CI run 39007164597).

Closed reviews through the `v0.17.0` cut are in the archive above.

### ASW-1 — Conversation event commits — reviewed `43da6bec` — **Fix model:** difficult — Codex Sol.

Seven Must-fix findings and one Worth-fixing documentation finding keep this unit open. Fix scope
is assessed independently below; the hardest fixes coordinate conversation/generation ownership
and durable failure recovery. No product code or specifications were changed. Locations refer to
the reviewed snapshot, before concurrent ASW-2 work.

- **Must fix** — ASW1-1 — **History refusal silently discards an accepted held message**
  (INV §1/§5/§8/§15). `internal/runtime/chat.go:959–963,1234–1262,1276–1285`: Send accepts a
  follow-up while the current turn is healthy; a later append/sync failure latches the journal.
  `finishTurn` removes the hold before the successor checks `Admit`. If storage still fails, the
  follow-up is never submitted, its text-only hold is gone, and its error event is suppressed by
  the same broken journal. The composer already cleared its draft on acceptance. FS-03.R99/A78
  require recoverable input ownership. Preserve/restore the exact hold on definite history refusal
  and report outside the transcript, without replacing a newer hold or replaying consumed input.
  **Fix scope: medium.** `TestReviewHeldSurvivesJournalRefusal` reproduces `held=""` with only
  the first native prompt submitted.
- **Must fix** — ASW1-2 — **The failed-permission notification exposes no approval controls**
  (INV §8/§10/§11/§15). `internal/server/server.go:424–427`, `internal/bus/bus.go:153–172`,
  `ui/src/api/sse.ts:213–237`, `internal/server/sessions.go:391–411`: an unsaved permission
  request stays pending in the runtime, but only a notification is published. Toast/notification
  handling merely opens the conversation; desktop controls require a transcript event and phone
  controls use the canonical window's `pending_permission`, which is null. Reload cannot discover
  the request either. This violates FS-03.R99/TS-03.R73's reachable approvals. Add a bounded,
  generation-scoped non-history pending-control projection, including resolutions and reload,
  while keeping failed records out of retained history. **Fix scope: medium.**
  `TestReviewUnsavedPermissionStillHasApprovalSurface` injects request-append failure and confirms
  live `tc_p` plus delivered `permission_required`, but no windowed pending request.
- **Must fix** — ASW1-3 — **Exit during persistence setup leaves an ownerless live writer**
  (INV §4/§5/§9/§16). `internal/runtime/chat.go:2318–2331,2337–2353`: `Journal.Open` attaches
  before `UpsertSessionMeta`, but `journalAttachment` is assigned only after that write. An exit
  in this window runs `closePersistence` with attachment zero, skips Close and releases the
  generation reference. The attached writer keeps the journal alive with zero references and no
  closer; later inactive annotations/notices return `ErrJournalLive` until replacement/restart.
  TS-01.R39/TS-02.R52 require generation-fenced teardown here. Publish attachment ownership with
  lifecycle state and close a writer whose owner already exited. **Fix scope: medium.**
  `TestReviewExitDuringPersistenceAttachClosesJournal` blocks metadata indexing after Open,
  runs exit teardown, releases indexing, and reproduces inactive-append refusal.
- **Must fix** — ASW1-4 — **Failed room settlement forgets its exact capture failure**
  (INV §4/§5/§7/§9/§15). `internal/server/think_tank_capture.go:63–71,91–95`,
  `internal/server/think_tanks.go:466–491`: terminal handling deletes a failed capture before
  `FailThinkTankAttempt`. If that write and the fallback room-hold write fail, nothing retains the
  exact failure for live retry. Restoring SQLite leaves the attempt running; terminal retry now
  sees no failed capture and can finalize its staged contribution. TS-01.R39/TS-04.R100 require
  monotonic failure and an exact recovery obligation; generic restart recovery is insufficient.
  Retain the exact attempt failure until its idempotent failure/hold transition commits and fence
  finalization on it; never replay activity appends. **Fix scope: difficult.**
  `TestReviewFailedRoomSettlementRetainsFailure` aborts both writes with SQLite triggers, restores
  them and retries the terminal event: the raw attempt row becomes `finalized`.
- **Must fix** — ASW1-5 — **Recovery ignores and clean close erases reserved sequences**
  (INV §1/§7/§9/§11/§15). `internal/runtime/journal.go:128–145,186–190,323–328`: health stores
  `ReservedSeq`, but opening advances allocation only from the complete log tail. A failed/torn
  append followed by restart can assign its reserved identity to the gap notice. Clean Close
  while degraded also writes a zero reservation, violating TS-02.R52. Preserve the watermark
  through degradation/close and advance allocation beyond it before recovery records. **Fix scope:
  trivial/easy.** `TestReviewRestartHonorsReservedSequence` seeds reservation 42 and reproduces
  a recovery notice at sequence 1.
- **Must fix** — ASW1-6 — **Control ordering ends at a generation boundary**
  (INV §1/§2/§4/§5/§15/§16). `internal/runtime/chat.go:288,2132–2135`,
  `internal/runtime/control.go:25–35`: the FIFO lives on `agentState`, so Stop/Resume or Switch
  creates another dispatcher while old callbacks may still drain. A newer control observation can
  run before an earlier one settles, with multiple sinks/queue budgets for the same conversation.
  TS-01.R39 requires one per-conversation dispatcher. Share ownership/lifetime with the
  conversation and retain captured generation checks. **Fix scope: difficult.**
  `TestReviewControlOrderSurvivesGenerationReplacement` blocks the old observation and confirms
  the new generation's callback runs concurrently on the same journal.
- **Must fix** — ASW1-7 — **Cancellation removes the control queue's admission bound**
  (INV §4/§5/§16). `internal/runtime/control.go:41–60`, `internal/runtime/chat.go:2146–2156`:
  after capacity fills, cancellation makes `reserve` return false, yet every late emission is
  appended unreserved. Buffered provider output/permission continuations can retain arbitrary
  pending events behind a slow sink. TS-01 §5 calls this deviation bounded but supplies no other
  bound or durable cancellation obligation, contrary to R39's 256 pending plus one in flight.
  Keep admission bounded through shutdown and retain required cleanup through its exact domain
  authority. **Fix scope: medium.** `TestReviewCancelledControlRemainsBounded` reproduces 599
  pending observations against limit 256.
- **Worth fixing** — ASW1-8 — **The spec describes a different inactive-writer fence** (INV §10).
  `docs/specs/tech/TS-01-architecture.md:530–531` requires joining the lifecycle claim and
  rechecking ownership; `Registry.AppendHistory`/`Journal.CommitDetached` use the journal mutex
  and live-writer recheck instead. This lock is sufficient for atomic append-versus-attach and
  avoids a second lifecycle lock; the local choice is accepted for sequence serialization.
  Update normative wording/deviation so the handoff is not its only explanation. **Fix scope:
  trivial/easy.**

Local-choice outcomes: a separate health table is justified because the open marker precedes a
new session row; conservative unclean-stop gap notices and offline index repair are reasonable.
The absent banner is covered by the refused-Send surface, but does not resolve ASW1-1/2. Gap
notices temporarily lagging `as.seq` do not reuse allocation and appear in canonical HTTP reads.
Unreserved enqueue is rejected under ASW1-7; the inactive fence needs ASW1-8's spec correction.

Invariant sweep: §1/§2/§4/§5/§7/§8/§9/§10/§11/§14/§15/§16/§17 apply and were checked; existing HTTP
guards and owner-only persistence remain wired. §3 has no one-shot persisted field/form surface,
§6 adds no external runtime/driver, §12 changes no external CLI invocation, and §13 changes no
markup/selectors. History/control interfaces and terminal persistence retain existing contracts.
The seven reproductions expose missing failure/lifecycle coverage in FS-03.A78 (INV §17).

Evidence: clean-entry untagged runtime/server/state/transcript/bus suites passed (server needed
approved loopback access after sandbox `::1` refusal); focused journal/control race tests passed.
After ASW-2 edits appeared, all seven probes were reproduced against isolated `43da6bec` using
`/private/tmp/chuck-asw1-review-overlay.json` and `chuck-asw1-review-server-overlay.json`.
Focused existing runtime/server checks pass on that snapshot untagged and `sqlite_fts5`.
The room retry probe deliberately repeats terminal settlement to test monotonicity; no live-provider
duplicate-terminal receipt is claimed. Native/provider/device gates remain owed. No sibling fixes
or product/spec edits were selected.

### ASW-3 — Process and terminal-driver ownership — reviewed `3618df3e` — **Fix model:** difficult — Codex Sol.

Five confirmed Must-fix findings keep this unit open. Fix sizing is independent of severity:
ASW3-1/3/5 are medium, ASW3-2 is easy, and ASW3-4 is difficult because leaderless-group recovery
must preserve ownership proof instead of blindly signaling a numeric PGID. All locations below
refer to the reviewed snapshot, before ASW-1's later journal changes.

- **Must fix** — ASW3-1 — **Terminal Stop destroys the view after ownership verification fails**
  (INV §4/§5/§9/§15). `internal/runtime/terminal/terminal.go:369–409,553–560`:
  `terminate` discards `StopRecordedProcess`'s error, then Stop calls tmux `kill-session` or iTerm2
  window close, deletes the running row and reports success. An unreadable live process birth
  therefore bypasses FS-01.R54 / FS-07.R25 / TS-05.R28's fail-closed refusal. The runtime has
  already removed its agent and set `stopped`; `Registry.Stop` (`registry.go:519–535`) also drops
  its claim before calling the runtime. Propagate the refusal and preserve recoverable ownership,
  row and registration until control succeeds; do not just add a return after ownership is lost.
  Actual Stop probe with a test-owned external child and unknown proof returns nil and kills the
  child through CloseTab. Regress unknown proof at Stop, retained ownership and a later safe retry.
- **Must fix** — ASW3-2 — **An unverified external process is published as exited**
  (INV §1/§4/§7/§9). `internal/runtime/terminal/terminal.go:220–237,508–530`:
  `exitSignal` closes for every classification except `ProcessOwned`, including
  `ProcessUnverified`. A transient failure to read a live tmux/iTerm2 process's birth triggers
  watcher teardown, deletes its row and registration and frees the registry slot while the CLI
  is alive. This violates FS-01.R54 / FS-07.R25 / TS-01.R41; only confirmed exit may run exit
  cleanup. Retain the unknown process and recheck/surface its control refusal. The actual watcher
  probe reports `runningErr=state: not found`, `processAlive=true` and an exit callback after
  clearing the test child's birth proof. Regress unknown → recovered proof separately from exit.
- **Must fix** — ASW3-3 / ASRB-1 remainder — **SessionStart can precede the row that authenticates it**
  (INV §1/§4/§5/§11/§15/§17). `internal/runtime/terminal/terminal.go:246–269,296–319` starts
  the CLI before opening persistence and writing `running`; `state.Manager.ApplyHook`
  (`manager.go:96–103`) rejects SessionStart without that row. `_post.sh:48–53` swallows the
  response with no retry. A fast terminal CLI can lose its sole native ID; Stop retains an empty
  ID and later Resume refuses instead of reopening the conversation (FS-01.R54, FS-07.R25,
  TS-02.R54). Establish reliable hook admission before provider execution, with rollback, or
  bounded reliable delivery. A fake driver calling the real Manager during the actual StartTab
  path observes `state: token mismatch`, then Runtime.Start still succeeds. Existing accepted-hook
  persistence tests pass but pre-seed the row. Regress early SessionStart → Stop/restart → Resume.
- **Must fix** — ASW3-4 / ASRC-3 remainder — **Already leaderless groups escape orphan teardown**
  (INV §4/§9/§10). `internal/runtime/procstart.go:107–111` returns success immediately when the
  recorded leader is gone; `runtime/reconcile.go:26–37` also prunes that row without accounting
  for group members. After a dashboard crash, a CLI/adapter leader can exit while its tool child
  remains in the dedicated group. Stop/Archive/Switch can then clear the record and proceed while
  that descendant survives (FS-01.R21/R54, TS-01.R41/TS-02.R54). The actual
  `TestASW3LeaderlessGroupActualStopRecordedProcess` fails: a reaped shell leader's background
  sleep remains alive after StopRecordedProcess returns nil. The existing descendant regression
  starts with a live leader. Preserve sufficient group continuity/ownership evidence or refuse
  uncertain cleanup; an unconditional kill of the old numeric PGID would reintroduce reuse risk.
- **Must fix** — ASW3-5 — **A nonnil xterm exit channel bypasses process proof even after reaping**
  (INV §4/§5/§9). `internal/runtime/terminal/terminal.go:352–355,555–574` treats channel
  existence as continuing ownership and sends SIGINT/TERM/KILL without birth verification.
  `xterm.go:55–61` reaps the child before closing that channel; watcher/persistence cleanup can
  leave the old handle reachable in the meantime. Cancel/Stop during that window can signal a
  reused process group, contrary to FS-01.R54 / FS-07.R25 / TS-05.R28. Check settled exit state
  and corroborate identity immediately before signals; preserve descendant containment. An actual
  Runtime.Cancel probe with a closed exit channel and stale birth delivers SIGINT to a separate
  test-owned group. No unowned process or forced OS PID reuse was used.

Original-finding reconciliation: **ASRA-4, ASRB-4–7, ASRC-4 and ASRC-7 closed** against their
original evidence. Setup cancellation/deadline uses the shared owned group and bounded drain;
registry exit-before-commit and old-generation callbacks are fenced; startup/periodic exit outcomes
agree; external driver exits, process proof, exact child env and same-title identities have source,
fixture and PTY evidence. ASRB-1's accepted-hook durability is repaired but early delivery remains
ASW3-3; ASRC-3's reused-PID reaping is repaired but leaderless containment remains ASW3-4. These
remainders retain their original IDs and have ASW-3 as their one fix owner.

Local choices accepted: migration 56's column is an explicit schema-level proof version with no
legacy backfill; the dedicated leader PID also names PGID; retaining a chat running native ID during
orphan deletion preserves the same identity; installation still controls offered drivers while
per-launch proof gates success; unprovable iTerm2 foreground groups refuse launch. These do not
close FS-07.A12. The shared bootstrap extends existing drivers and the current executable; no
additional daemon/provider service was introduced. Terminal contract checklist: persistence and
composed launch spec are shared; PTY drain/viewer detach remain separate; messaging stays excluded;
turn hooks remain the existing producer; reconcile/teardown failures are recorded above.

Invariant sweep: §1/§2/§4–§12/§14–§17 checked across lifecycle, migration, hooks, bootstrap,
controls, bounded records and tests; findings identify their matching classes. No applicable §3
one-shot persistence/form surface or §13 UI/CSS surface. Independent verification and probe paths
are recorded in the changelog below; no product code/specs changed. Real tmux/iTerm2/provider
receipts remain unclaimed.

### ASR-A — Instructions, configuration and execution environment — reviewed `4fbeb316` — **Fix model:** difficult — Codex Sol.

Source/profile delivery must agree with native authentication,
session isolation and frozen continuation choices; the other fixes mostly extend existing seams.
Audit complete; findings keep this same unit open. No product code or specifications changed.
Source mode: repository source plus installed, lockfile-pinned packages: Claude ACP 0.85.1,
Claude Agent SDK 0.3.286, Codex ACP 2.1.1 and Codex CLI 0.159.3. Tree was clean at entry.

- **Must fix** — ASRA-1 — **Source model overrides are lost on continuation** (INV §1/§2/§3/§11).
  `internal/server/config_sources.go:586–605`, `resume.go:321–328,519`,
  `switch.go:237–239,478`: a bound launch with `overrides.model:"native-sentinel"` sends that
  model, but stores the catalog key (`sonnet`) on the identity. Ordinary/wake resume and an
  identity-preserving switch rebuild `ModelID` from the catalog; their only federation correction
  handles native inheritance, so the override silently becomes `sonnet`. An explicit empty model
  override also emits no model initially but records `native_inherited:false`, causing the same
  change on continuation. This violates FS-08.R10/R18/R19 and TS-07.R5/R6. Restore the frozen
  effective choice through a shared resolver, preserving later explicit model switches and explicit
  refresh. Regress nonempty and empty overrides through fresh → Stop/wake/Resume, same-identity
  switch and rollback with changed catalog/source defaults. Temporary composer probes confirm
  both mechanisms; `/tmp/asr02_nonempty_test.go` captures the actual fresh/resume mismatch.
- **Must fix** — ASRA-2 — **Empty effort overrides bypass the declared default** (INV §2/§11).
  `internal/server/config_sources.go:604–605`, `launch.go:550–560`: a binding with
  `overrides.effort:""` reaches `resolveEffort` as a present override and returns empty instead
  of falling through to the model's `default_effort`. This changes reset-to-inherit behavior and
  freezes a different effort from the documented precedence (FS-08.R10/R31, TS-01.R12,
  TS-07.R14). Normalize empty resets consistently at the existing source/resolution seam; test
  empty versus absent versus nonempty override, with a declared default and explicit request.
  The isolated resolver probe observes empty with a `medium` model default.
- **Must fix** — PROMPT-1 **reopened, ancestor-project variant** — **Codex guidance is still
  replaced from nested working directories** (INV §2/§10/§12/§17).
  `internal/configsource/codex.go:200–221`, `internal/runtime/chat.go:2610–2620`: the merge reads
  only `<cwd>/.codex/config.toml`; pinned Codex also loads ancestor project layers between a
  nested cwd and repository root. With native user text `USER_SENTINEL`, repository-root text
  `PROJECT_SENTINEL` and cwd `repo/packages/app`, native `config/read` selects the project text;
  Chuck instead emits `USER_SENTINEL\n\nCHUCK_SENTINEL`, replacing and losing the project text.
  This violates FS-09.R32/FS-18.R17's preservation contract and TS-07.R2/TS-04.R14. Resolve the
  actual supported native layer chain before adding Chuck's text, retaining bounded reads and
  native precedence. Regress a nested cwd with ancestor and nearer overrides against the pinned
  native boundary, beyond the existing exact-cwd fixture. Evidence: synthetic, credential-free
  `/tmp/chuck-asra-native-config.mjs` and `/tmp/chuck-asra-native-config_test.go`; no model turn.
  The earlier user/exact-cwd correction remains valid; this is the same finding identity.
- **Must fix** — ASRA-3 — **An accepted custom source/profile does not select the child's setup**
  (INV §2/§10/§11/§12). `internal/server/config_sources.go:569–605`, `launch.go:724–737`,
  `internal/config/codexprofile.go:83–92,180–202`: the supported source API can bind a distinct
  Codex root and named profile, resolve/freeze `bound-profile-sentinel`, and omit ACP model as
  native-inherited. Child composition transfers neither root nor profile, and the private home
  refresh copies the ambient personal source (`personal-sentinel`) instead. The agent therefore
  receives another setup/model while its provenance describes the selected binding (FS-08.R3,
  R12/R16–R20/R32, TS-07.R2/R5/R6). The same missing transfer seam affects custom Claude roots;
  the deterministic end-to-end composer/mirror probe covers Codex. Align supported source
  selection with native launch and isolated-home ownership, or explicitly refuse unsupported
  selections before accepting a launch; do not claim success from resolver bytes alone. Test
  distinct personal/bound roots and named profiles across both providers and continuation, with
  native payload verification. Evidence: `/tmp/asr02_codex_root_test.go`, using only fixtures.
- **Worth fixing** — ASRA-5 — **Resource validation widens existing owner permissions**
  (INV §10/§14). `internal/config/resources.go:53–89`: an existing resource directory set to
  `0500` is chmodded to `0700` before its writability probe and launch is allowed. FS-11.R9
  requires an existing unreadable/unwritable target to fail actionably. Tightening group/world
  access under TS-02.R13/TS-05.R13 does not require adding owner write permission. Preserve that
  distinction at the shared helper and test restrictive owner modes as well as permissive umasks;
  `/tmp/chuck-asra-resource-mode_test.go` confirms the mode change. No resource contents were read.
- **Must fix** — ASRA-6 — **A second start bypasses unfinished checkout setup**
  (INV §1/§5/§15). `internal/server/worktree.go:164–189,215`: after one caller recreates a
  missing owned checkout and enters setup, another caller sees the newly existing directory and
  returns before taking the recreation claim. Its agent can start while dependencies/bootstrap
  work is still being installed, despite the first caller holding that claim (FS-19.R3/R7,
  TS-12.R4/R5). A real temporary Git-worktree probe with a 600 ms setup returns the second
  success in about 120 microseconds while the first setup is unfinished. Extend the existing
  per-project readiness seam so every start encountering an in-progress recreation observes its
  completion/warning, including when the directory has already appeared. Regress two starts
  arriving on opposite sides of directory creation, with setup success/failure/cancellation;
  `/tmp/chuck-asr-probe/recreate_setup_race_probe_test.go` preserves the reproduction.

Effective contract and lifetime map (J1, including J5's execution context):

| Layer/choice | Fresh process and native delivery | Continuation/lifetime |
|---|---|---|
| Project/persona/resources and operator-authored pipeline guidance | One base composer freezes ordered prompt, cwd, user/resource add-dirs and resolved permissions/awareness | Resume/switch retain the base; new manual/task/pipeline/room identities use the same launch service |
| Shared Chuck context and installed skill | One runtime-only suffix; conditional verified skill path/env/root; awareness off omits shared text; installation failure suppresses skill advertisement | Rebuilt once per process from frozen base and current startup availability; never added to session prompt/add-dirs metadata |
| Native Claude guidance | Chat new/load request native `claude_code` preset plus append; terminal uses append argv; cwd and directory grants reach SDK/CLI | Native fork deliberately sends no new prompt (TS-04.R65); provider-owned prompt snapshots remain the live FS-18.A13 gate |
| Native Codex guidance/setup | Private-home refresh, then developer-instruction merge into `CODEX_CONFIG`; ACP new/load omit unsupported systemPrompt | New/load share the path; ancestor guidance and selected custom root/profile fail as recorded above |
| Model/effort/fast | Explicit → binding override → documented native/catalog fallback; chat applies model then effort then fast, checking advertisement/readback | Model override/reset defects above; frozen effort and applied fast normally reused; explicit model switch remains explicit until refresh |
| Environment and registrations | Ambient → backend → model → hook/resources → reserved private Codex home/provider executable; skill env is runtime-only | Environment values are recomposed, only key names persisted; hook/MCP identity is fresh and generation-scoped, provider executable reselected per start |
| Checkout and setup | Selected expanded cwd; ownership-based recreation and setup before start; setup failure is intentionally a warning | Resume/switch keep frozen cwd; stable resources follow project id; task/pipeline/room launches and pipeline continuation use the shared composers |
| Isolated quota check | Separate temporary cwd, no work prompt/resources/coordination; explicit settings/MCP isolation | Existing check-selection and isolation tests prove Chuck payloads, not account/model adoption |

Coverage ledger: catalogue **01** — all three prompt composers/callers, knowledge publication and
degradation, exact role migrations, awareness opt-out, orchestration/task/room additions and both
provider prompt seams inspected. Current seeds are about 100–115 words each; the 348-word operating
skill progressively routes four references. Shared standing text encourages reporting/dependencies;
repository `AGENTS.md`/role skills add separate mandatory process. No prompt-size→quality causal
claim, duplicated overlay accumulation or new installation/migration defect established.
Catalogue **02** — selection/federation/freshness/false-empty handling, source manager, native
home/profile and fresh/resume/wake/refresh/clone/switch/rollback paths inspected. Source-invalid
launches fail closed; mirrored generations remain display state. Source override and native delivery
failures are above. Legacy profile-table helper fixtures alone do not prove pinned native validity:
Codex 0.159.3 rejects legacy `profile`/`[profiles.*]` selection and directs callers to named config
files/`--profile`. Catalogue **25** — cwd/resource/grant parity, owned checkout fork/recreation,
Git errors/dirty deletion checks, setup warnings/output bounds and process cancellation inspected;
setup containment and concurrent readiness are defective as above. No external checkout deletion
was established.

Local-choice outcomes: retain runtime-only versus frozen prompt fields, progressive knowledge,
warning-only failed setup, and explicit ownership/consented deletion. An edited owned project's cwd
can diverge from its retained ownership row; the probe confirms this, but ownership intentionally
identifies the old managed checkout and deletion revalidates it and requires explicit consent.
No unconsented-deletion finding is asserted. Terminal `SkipPerms` currently produces identical
argv/env for true/false; FS-07 does not specify that mapping and retains its native-flag acceptance
gate, so terminal policy application remains an explicit evidence/spec limitation rather than a
claimed pass. No new prompt on native fork is required by TS-04.R65.

Existing evidence reconciled: PROMPT-2 remains under its original unit below; the frozen-only
fingerprint limitation is unchanged. CI-1's per-prompt fake cancellation and three repeats of both
late-tool/cancel regressions pass. `SetSessionConfig` still holds `as.mu` over a provider call;
the existing notification-lock hypothesis is not closed or duplicated here. Live persona/native
prompt, federation, terminal, worktree usability and provider/device gates remain owed; no actual
Claude/Codex model call, credentialed acceptance, rendered UI or measured quality comparison ran.

Verification: focused server federation/knowledge/orchestration/resources/worktree tests pass with
approved loopback execution after the sandbox rejected `httptest` binding. Focused runtime new/load,
native-instruction merge, terminal argv and quota isolation tests pass; config, agentknowledge,
configsource and Git-worktree packages pass. Temporary Go overlays preserve product/spec files;
their probes observe defects rather than claim acceptance. Consolidated reproduction:
`GOCACHE=/tmp/chuck-gocache-asr02 GOFLAGS=-p=1 go test -overlay=/tmp/chuck-asra-review-overlay.json
./internal/server ./internal/config ./internal/runtime -run '^(TestASR02.*|TestASRA.*|TestProbeSetupCancellationWaitsForDescendant|TestProbeRecreateSecondStartBypassesSetup)$' -count=1 -v`;
log: `/tmp/chuck-asra-review-probes.log`. The native fixture probe separately uses stdio-only
`config/read`. Invariant sweep: §1–§5, §7–§12 and §14–§17 checked; §6 (new interface/runtime) and
§13 (UI selectors) have no applicable new surface. Fixes and sibling reviews were not selected.

### ASR-B — Codex and Claude native integration fidelity — reviewed `6b5de19b` — **Fix model:** medium — Codex Terra or Claude Opus.

Audit complete; ASRB-1/2/3/8 keep this unit open. ASW-3's review closed ASRB-4–7;
its new terminal control findings are recorded on ASW-3 above. Product code/specifications unchanged.
Fix sizing is independent of severity: ASRB-3 is easy; ASRB-1/2/8 are medium.
Source mode: repository source plus locally installed lockfile packages, Claude ACP
0.85.1 / Agent SDK 0.3.286 and Codex ACP 2.1.1 / CLI 0.159.3. Assembly pins Node 22.22.0 and
Codex's patched component `2.1.1+chuck.1`; local test Node is 22.23.1. No native model turn ran.
The pre-existing dirty ASR-C row and its concurrent administrative closure were preserved;
`07f201a3` changes review state, not the inspected implementation. Three bounded read-only audits
covered Codex selection/profile, Claude negotiation and terminal drivers; findings were re-read
and integrated here.

- **Must fix** — ASRB-1 — **Terminal Stop loses the native conversation needed by Resume**
  (INV §1/§2/§4/§5/§11/§15/§17). **Partially resolved by ASW-3, reviewed 2026-10-19.**
  Accepted SessionStart now updates `sessions.last_session_id` transactionally; Stop/orphan
  deletion retains it and empty-ID Resume refuses. The early-hook race still loses native
  identity: see **ASW3-3** above for current location, actual-code reproduction and fix/test scope.
  That remainder stays owned by ASW-3; do not implement a second fix from this originating unit.
  No real-CLI receipt.
- **Must fix** — ASRB-2 — **Claude Clone never opens the forked conversation**
  (INV §1/§6/§11/§12/§17). `internal/runtime/chat.go:430–467`, `fork.go:34–68`:
  pinned Claude ACP's `unstable_forkSession` performs an SDK persistence fork and returns only
  `{sessionId}` without registering `this.sessions[id]` (`claude-agent-acp/dist/acp-agent.js:1405–1412`,
  `fork-session.js:107–113`). Chuck immediately applies settings: an explicit model/effort fails
  on the absent advertisement; with neither, startup succeeds but the first prompt fails
  `Session not found` (`acp-agent.js:1815–1821,5419–5423`). The fork operation also ignores the
  supplied MCP/additional directories. This violates FS-01.R36 and TS-04.R47/R65. Open the new
  fork through the supported load path with fresh scoped MCP/directories before configuration,
  or withhold unsupported fork; retain compensating delete on failure. Regress the pinned bare
  fork response, registration, settings, first prompt and cleanup. The fake masks this by returning
  config options and immediately activating the fork (`testdata/fakeacp/main.go:329–347`).
  Pinned source establishes the failure; no credentialed clone receipt is claimed.
- **Must fix** — ASRB-3 — **Codex Clone reapplies a newly composed prompt and history primer**
  (INV §1/§2/§11/§12). `internal/runtime/chat.go:160–178,2571–2623`,
  `internal/server/launch.go:417`: even with `spec.Fork`, process start merges
  `StartSystemPrompt()` into `CODEX_CONFIG.developer_instructions`. The pinned adapter parses
  that overlay and passes it in `threadFork.config` (`codex-acp/dist/index.js:40302–40323,
  33447–33461,34187–34206`). Thus cloning after a persona change can replace inherited instructions
  with current ones, and a retained bridge is reapplied, contrary to TS-04.R65's explicit no-new-
  prompt/primer rule. Skip the Chuck addition for a native fork while preserving native setup;
  regress the actual process environment, not just the correctly prompt-free fork RPC. An
  isolated helper probe observes both `SYSTEM_SENTINEL` and `PRIMER_SENTINEL` in that overlay;
  no duplicate visible transcript or model-quality effect is asserted.
- **Must fix** — ASRB-8 — **Source Codex Steer exposes an incompatible detached-turn path**
  (INV §1/§10/§11/§12/§17). `internal/backend/adapter.go:423–432`,
  `internal/runtime/chat.go:791–808,844–877`, `scripts/release/assemble.sh:68–83`:
  documented source launches use PATH's unpatched `codex-acp`, while only release assembly adds
  the required `promptRequired` patch. Pinned unpatched 2.1.1 still advertises steering, drops
  the request's idle opt-in and starts a detached turn if the active turn just finished
  (`codex-acp/dist/index.js:38522–38538,38590–38635,38668–38683`). Chuck accepts
  `startedNewTurn` but has no ordinary prompt completion owner for that turn, violating
  FS-03.R56/TS-04.R51. No-retry of already-consumed content is correct and must remain. Supply
  a verified source adapter path or withhold the unproven feature; regress an idle-race against
  the unpatched pinned boundary and assert busy/cancel/one terminal outcome. Existing legacy
  tests prove classification only, expressly not successful fallback. No live provider race ran.

Provider/path realization and evidence ledger:

| Catalogue | Traced paths and result |
|---|---|
| **03 — selection/package/auth/catalog** | Shared resolver covers fresh/wake/resume/switch/rollback/task/pipeline/clone, direct Claude terminal, version/login/readiness. Installed overrides/discovery and explicit Bundle stay separate; no silent fallback established. Private Node/adapters/native paths, exact pins, patch hashes, single Codex package and manifest/wrapper agreement inspected. Resolver, probe, auth-target, refresh and release-layout fixtures support selection, not credentials or real packaging acceptance. Source Steer differs as ASRB-8. |
| **04 — Codex native contract** | Private-home refresh and reserved environment across start/load; overlay to native config, ordered model→effort→fast, Steer, fork and capability/source contracts traced into pinned source. ASRB-3/8; ASR-A's PROMPT-1/custom-root findings retain their identities. Native account/home/model adoption, handoff/compaction and packaged optional-feature receipts remain owed. |
| **05 — Claude native contract** | New/load share preset-plus-append metadata, MCP/directories, model metadata and ordered configuration; provider model refusal fails without bare/default retry. Advertisements/readback remain adapter evidence. Fork differs from load (ASRB-2). Terminal flags are composed; native-preset snapshots, modes, effective settings and terminal flag/hook adoption remain live gates. |
| **21 — terminal/PTY/drivers** | Launch/resume/status hooks, prompt/resize, multi/no/slow-viewer drain and disconnect, quoting/FIFO env, process exit/Stop/restart and switch callers inspected. Default PTY has bounded scrollback and nonblocking subscribers; closing a viewer unsubscribes. Codex/other terminal backends, terminal MCP mail/ACP permissions/Steer/fast are explicitly unavailable. ASRB-1/4/5/6/7; no UI driver picker or tab cap is claimed beyond FS-07's existing deviations. |

J1 challenge: selection/configuration payloads agree at the shared resolver and ordered option
seam, subject to ASR-A's frozen/source defects and live adoption limits. J3 challenge: Claude fork
does not install fresh scoped MCP authority until opened; tmux may lose hook identity via env.
J5 challenge: terminal native identity and both providers' fork paths break intended context
continuity as above. J4 terminal chain: default PTY output/viewer detach does not stop work;
native failure/crash closes its process-backed runtime, but its durable native id is lost.
Optional drivers lack complete lifetime/containment. Terminal has no chat quota dispatcher or
automatic continuation contract; retained native conversation must precede any supported recovery.
J2/J6 controls were traced through server lifecycle claims, settings partial-success persistence,
switch preflight/rollback and failed native load without fresh-session fallback. ASRC-1/2/3/4/5
retain their existing identities; no duplicate findings are created here.

Local-choice outcomes: retain additive Claude preset metadata, model→effort→fast ordering,
required-setting fail-closed/fast fail-open behavior, native-load refusal preserving history,
explicit terminal capability boundaries and existing supported-driver/UI deviations. No redundant
configuration or compatibility registry is proposed. Invariant sweep: §1–§12 and §14–§17 have
applicable configuration, runtime, driver, protocol, package or test surfaces and were checked;
§13 has no applicable markup/style change. No additional security, resource-bound or test-oracle
defect established beyond the findings and existing ASR-A/C records.

Verification: focused backend/runtime session, Claude/fast/model/capability/Steer/fork tests,
Codex profile/auth tests and the complete terminal package pass. Root independently passed
provider resolver/version bounds, source-aware server composer/error tests, live config partial
success, mode routes, explicit-model continuity, native-load loss refusal, switch rollback,
unsupported handoff and fake clone; approved loopback execution was required after sandbox bind
denials. Credential-probe fake-endpoint tests pass with that approval; release wrapper/layout/
archive-context fixtures, `make check-specs` and `git diff --check` pass. These are focused checks,
not a new release/full closure claim.
Consolidated defect probes pass against the current implementation:
`env -u CHUCK_RUNTIME_ROOT GOCACHE=/tmp/chuck-gocache-asrb GOFLAGS=-p=1 go test
-overlay=/tmp/asrb-probes-overlay.json ./internal/runtime/terminal ./internal/server
./internal/runtime -run '^(TestOverlayTerminalNativeIDDoesNotReachSnapshot|TestASRBTmuxTitlesCollideAcrossAgentIDs|TestASRBZeroPIDPruneLeavesRuntimeClaim|TestCodexForkStillInjectsPromptOverlay)$' -count=1 -v`.
They use temporary Go overlays/fixture homes, exercise actual state/helpers and confirm defects;
they do not prove native adoption. No credentialed provider, iTerm/tmux integration, device,
rendered-browser or quality/performance receipt ran. All named live gates remain owed.

### ASR-C — Runtime execution, concurrency and control — reviewed `6b5de19b` — **Fix model:** difficult — Codex Sol.

The leaderless-group containment remainder needs ownership machinery (ASW3-4); status settlement
coordinates runtime/server state and other fixes extend existing seams. ASW-3's review closed
ASRC-4/7; ASRC-3 is partially resolved. Audit complete; findings keep this same unit open. No product code
or specifications changed. Source mode: repository source; real-adapter shapes read from the
release lockfile packages (Claude ACP 0.85.1; Codex ACP 2.1.1 is a native binary, not inspected).
Bounded discovery for catalogue 06/11/12 was delegated and every reported finding re-read in code.

- **Must fix** — ASRC-1 — **A live effort/fast change stalls the read loop when any frame precedes
  its reply** (INV §5/§15/§16). `internal/runtime/chat.go:1752–1795` holds `as.mu` across
  `session/set_config_option`; `onNotification` takes `as.mu` first in `scopeFor`
  (`subagent.go:79`), so the "mode updates never take mu" comment (`chat.go:1850`) is false. FS-03.R45/R47
  allow applying during a running turn; one streamed chunk before the reply blocks the reader,
  so the reply is never read. The HTTP handler passes `r.Context()` with no server or UI timeout
  while holding the lifecycle claim (`session_config.go:33,83`): the turn freezes, permission
  requests are unread, and Stop returns 409 until the browser aborts. The late success reply is then
  dropped ("response for unknown request id"), so Chuck persists and reports the old value while the
  provider runs the new one. Probe: `/tmp/chuck-asrc/{setconfig_probe_test.go,fakeacp_main.go,overlay.json}`
  (`GOFLAGS="-p=1 -overlay=/tmp/chuck-asrc/overlay.json"`): control applies in 0 s; chunk-before-reply
  fails at the 2 s deadline. Snapshot identity/advertisement under the lock, call unlocked
  (as `SetSessionMode` does under the lifecycle claim), merge the returned option list after;
  regress an interleaved update mid-turn plus the late-reply/persisted-value agreement.
- **Must fix** — ASRC-2 — **Status writes are not owned by the turn or the open permission gate**
  (INV §5/§1). `chat.go:2438–2459` writes `busy` for any tool call/result while `turnActive`,
  ignoring `as.pending`; `updateStatus` (`chat.go:2480`) is an unlocked read-modify-write.
  (a) A parallel or child tool frame after a permission request replaces `waiting_input` with
  `busy`, so the actionable card treatment (FS-02.R3/R22) disappears while a human gate is open.
  (b) A native background continuation frame (FS-01.R52) racing `finishTurn` — idle is written
  before `turn_end` persists and `turnActive` clears (`chat.go:1158–1172`) — leaves the card `busy`
  with no owning turn until the next turn. Delegated fake-ACP probes in `/tmp/asrc/probe_test.go`
  (overlay `/tmp/asrc/overlay.json`): (a) deterministic; (b) 299/300 under a 40-frame flood,
  0/300 without (real-world rate lower). Derive status from turn and pending ownership in one
  critical section; regress both interleavings.
- **Must fix** — ASRC-3 — **Orphan reaping kills by bare PID liveness** (INV §9/§4).
  **Partially resolved by ASW-3, reviewed 2026-10-19.** Recorded kernel birth and shared
  classification now protect orphan signals and reused-PID reconciliation; existing unknown/
  reused-PID regressions pass. The original descendant-containment promise still fails when the
  leader exited before teardown: see **ASW3-4** for the actual-code failure and ownership constraint.
  That remainder stays owned by ASW-3; do not add a parallel reaper in this originating unit.
- **Worth fixing** — ASRC-5 — **ACP permission prompts can remain actionable after settlement**
  (INV §5/§8). `internal/runtime/permission.go:76–83` registers a pending request before
  publishing it, and the `published` gate (`host_permission.go:54`) covers host requests only:
  Cancel/Stop/timeout in that window persists `permission_resolved` before `permission_request`
  and leaves `waiting_input`. Separately, approve/deny with no matching provider option
  (`permission.go:99–108`) answers cancelled, emits only an error and no `permission_resolved`,
  leaving a live card that returns 409. Delegated probes `/tmp/asrc/probe1_test.go`,
  `probe3_test.go`. Extend the publication gate to ACP requests and route the no-option branch
  through `resolvePending`.
- **Worth fixing** — ASRC-6 — **Per-generation retention never frees completed native work**
  (INV §16). `subagent.go:148–154` caps `as.children` at 256 and never deletes a terminal child,
  so in a long-lived orchestrator child 257+ loses all output and its permission requests are
  silently cancelled (`permission.go:32–36`); `as.toolNames` (`chat.go:2442`) grows per tool call.
  TS-04.R63 bounds children to the active generation, not lifetime count. Delegated probe
  `/tmp/asrc/probe_test.go`: 300 sequential children → 256 retained, child 299 dropped.
  Evict terminal children (keeping a bounded tombstone for late frames) and prune tool names
  with tool inputs.

J2 (lead) result — one conversation under concurrent control: Send/hold, withdraw, Steer
promotion and turn-end successor transfer decide under `as.mu` with single-owner hand-off; Stop
and crash release held/steer receipts; held input is intentionally lost on dashboard restart
(FS-03.R48/R49). Cancel escalation is turn-sequence fenced; `completePromptTurn` ignores stale
turns; late tool input is released at turn boundaries. ASRC-1 breaks J2 while a setting changes;
ASRC-2/5 break status/approval truth around permission waits and late native output.
Challenges to siblings: J6/ASR-H — a promoted held message without `request_id` is restored after a
non-refused ambiguous steering error (`chat.go:707–717`) and could be resent if the adapter
consumed it (UI always sends `request_id`; CLI path unchecked). J4/ASR-G — crash `turn_end`
dispatch reads `registry.Generation` after `onExit` cleared it (`server/server.go:426`), so
`dispatchTurnEnd` runs with an empty generation instead of `ev.Generation`; release consequences
unverified.

Coverage ledger: catalogue **06** — launch/Start/registry/reconcile/state transitions with
per-step failure handling; concurrent same-agent launches are atomic; late N exit is
generation-fenced; restart never adopts. **08** — claim/hold/withdraw/Steer/fallback/activation
admission and successor transfer read in full; deterministic count/order covered by existing
tests, no new defect beyond ASRC-1/2. **09** — Stop, cancel, escalation, shutdown, orphan stop;
ASRC-3/4. **10** — transport framing, pending routing, oversized frames end the session (36 MiB),
malformed frames are skipped (a malformed reply leaves only its caller's ctx to release it),
unknown inbound methods get -32601; SetSessionConfig hypothesis reproduced (ASRC-1); launch-time
configuration holds no `as.mu`. Event seq is allocated under `as.mu` but persisted/published
outside it; the writer and readers tolerate reordering, so no impact was established.
**11** — skip-permissions/registered-tool paths cannot resolve host peer-Stop records; single-winner
`takePending`; host decisions durable before release; ASRC-5. Unconfirmed: a model change does not
refresh the mode list from the `set_config_option` reply; selecting `allow_always` when no
`allow_once` exists follows techspec §5.3. **12** — child scoping, reasoning, failure reasons,
stop-reason mapping and replay gating hold; ASRC-2/6. Unconfirmed: open background tasks/children
get no terminal record on Stop/crash (display-only); initial `tool_call` content is not mapped.

Existing evidence reconciled: CI-1 closure holds — `go test -race ./internal/runtime -run
'TestCanceledTurnsReleaseToolInput|TestCancel|TestSteer|TestQueue|TestTurnOwnership|TestHeld|TestSetSessionConfig|TestPermission' -count=3`
passes (`GOCACHE=/tmp/chuck-gocache-asrc GOFLAGS=-p=1`, `CHUCK_RUNTIME_ROOT` unset). PROMPT-2 is
untouched by this unit. No real provider, credentialed run, rendered UI or device ran; peer-Stop
and native mode-adoption live gates remain owed. Invariant sweep: §1, §2, §4, §5, §8, §9, §10,
§11, §15, §16 and §17 applicable and checked; §3 (no persisted one-shot field), §6 (no new
interface), §7 (no record repair), §12 (no new external invocation), §13 (no UI markup) and §14
(no new HTTP surface) have no applicable surface. Fixes and sibling reviews were not selected.

### ASR-D — Conversation, input and history fidelity — reviewed `c12c4251` — **Fix model:** medium — Codex Terra or Claude Opus.

ASRD-4 remains open here and belongs to ASF-2. ASRD-1/2/5's original reproductions are corrected
and independently reviewed in ASW-1; ASRD-3's history suppression is corrected, with remaining
recovery/control obligations tracked only as ASW1-1–7 above. The retained audit evidence below
describes the original snapshot. No product code or specifications changed during this audit.
Tree clean at entry. Source mode: repository source at
`c12c425109a7bdd5682451f02afe958477319e7a`; release lock pins Claude ACP 0.85.1 / Agent SDK
0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. No credentialed provider run was performed.

- **Must fix** — ASRD-4 — **Retrying an annotation after a lost response delivers it twice**
  (INV §2/§11/§15). `ui/src/components/chat/annotationDelivery.ts:76–94`,
  `ui/src/lib/annotations.ts:69–76`, `internal/server/sessions.go:232–262`: delivery inserts
  reserved-user mail (or starts a self prompt) before the success response, but the request carries
  no stable delivery identity or receipt. If that response is lost, the preserved tray's Retry
  inserts another source event and another actionable mail/prompt. The isolated handler probe
  discards the first successful response and retries the same body: both return 202 and the target
  has two mail rows. This defeats FS-13.R3/R5's recoverable batch behavior and INV §15's harmless
  replay rule. Extend the existing receipt/idempotency pattern across desktop/phone delivery and
  launched-recipient retry; retain the deliberate new attempt after a definite pre-delivery
  failure. Room annotations already carry a command id and atomically deduplicate their mail.
  Regress lost responses for mail and self targets, including a reload and New task delivery.
  Evidence: `/tmp/asrd_annotation_overlay.json`, `TestASRDAnnotationLostResponseRetryDuplicates`.

Connected evidence ledger and J5 (context in delegated work):

| Retained topic | Whole-chain coverage and result |
|---|---|
| 07 — Lifecycle/history | Traced ordinary/wake/restart resume, load replay gate, clone prefix/artifact copying, switch/rollback, bounded primer and native-conversation bridge/compaction lifetime. ASRD-1 breaks successful handoff history; ASRB-1/2/3 retain terminal/Claude/Codex fork failures under their existing identities. ASRA-1 remains the frozen model-override finding. |
| 13 — Durability/replay | Traced append/sync → index/counters/rollups → hub/bus/SSE → transcript refetch/window/reconciliation. Atomic snapshot+subscribe and bounded slow-client fan-out close the subscription gap; source and UI tests cover refetch/folding. ASRD-1/2/3/5 break identity, order, durability and fallback truth. Torn tails and oversized/malformed records use the shared tolerant reader. |
| 14 — Input/bytes | Traced Composer through exact ACP text/image/file blocks, immutable SHA/size/ownership validation, request receipts, holds/Steer/Stop/restart recovery and clone-owned copies. File mentions and advertised skill/command names are intentionally verbatim text (FS-03.R31–R33); binary files become resource links/read instructions, never decoded prompt text. No additional attachment defect found. |
| 15 — Authorized context/annotations | Traced token-derived share/read identity, exact source spans, canonical grants, per-page authorization, cursor/rune bounds, tombstones and peer-read fixed prefixes. Task creation rechecks attach authority in its transaction; confirmed assignee membership grants its own references across Stop/terminal completion without copying unrelated grants. Annotations preserve captured excerpts/anchors without rereading disk; ASRD-4 breaks retry delivery and ASRD-1/2 affect recording. |

Task launch passes the verbatim assignment plus reporting guidance through the ordinary input seam;
existing-agent/wake/dependency continuations use host-owned activation and `get_assigned_task`, not
a global context scan. Pipeline successor/replacement context follows explicit task attachments
and work-derived membership. Room access rechecks membership and current attempt; capture matches
generation+turn, omits private prompts/prose, and projects Chuck tool calls/results as receipts.
Goal attachment paths remain ordinary provider file access under FS-21.R54/TS-05.R26, and judge
instructions are judge-only. Relative annotation anchors remain point-in-time source locations,
not directory grants. Native loads request the saved provider session while suppressing replay into Chuck;
cross-provider handoffs deliver bounded semantic text with omission markers, not reconstructed
multimodal history. These traces join intake, correct checkout, authority, native submission and
readback; they do not establish that a live provider consumed each supplied context.

Verification: focused attachment/image runtime and server tests and Composer's 41 tests pass;
context/peer/assignment MCP tests pass with approved loopback execution. Full `-race` package tests
for `internal/contextref`, `transcript`, `index` and `bus` pass. The FTS5 focused server/MCP
annotation, reconciliation, transcript-window, handoff and task-attachment checks pass; the full
FTS5 context-service package also passes. UI
transcript/SSE tests pass 50/50 with presentation checks. Temporary probes above reproduce defects;
the switch sequence regression intentionally fails. `CHUCK_RUNTIME_ROOT` was unset and tests used
isolated temporary homes/fake ACP. No provider credentials, rendered UI or devices were exercised;
FS-01.A7/A8, TS-02.R48 and existing task/room/native/device receipts remain owed.

Existing evidence reconciled: PROMPT-2's frozen-only diagnostics remain unchanged; CI-1's earlier
closure is retained, with no new cancellation-double evidence against it. SetSessionConfig still
holds `as.mu` across the config RPC (ASRC-1); its read-loop stall is not duplicated here.
Invariant sweep: §1–5, §7–12 and §14–17 applicable and checked; §6 has no new interface/runtime
contract to add and §13 has no styling/class-name change. No local implementation choice or human
decision was introduced. Sibling reviews and fixes were not selected.

### ASR-E — Agent tools, communication and durable delegation — reviewed `5eb574d9` — **Fix model:** medium — Codex Terra or Claude Opus.

The continuation fix joins durable wait settlement, assignment projection and MCP reads; report
bounds extend existing shared validation. Audit complete; findings keep this same unit open.
Source mode: current repository source at `5eb574d9`, initially clean. Release lockfile: Claude
ACP 0.85.1 / Agent SDK 0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. Three bounded read-only audits
covered tools/authority, peer delivery and tasks; the lead reconciled interactions and independently
reproduced both findings. No product code or specifications changed.

- **Must fix** — ASRE-1 — **A resumed assignment loses the watched changes that woke it**
  (INV §1/§10/§11/§15). `internal/messaging/task_tools.go:28–66`,
  `internal/state/tasks.go:1425–1442`, `internal/runtime/activation_kinds.go:55–58`:
  after `wait_for_tasks` yields and a child reports, the dependency activation tells the parent
  to read `get_assigned_task`, but that result contains only the original instruction, identity,
  state, execution handle and attachments. It exposes neither `observed_changes` nor the retained
  result/continuation context required by TS-10.R29/§3 and FS-16.R30–R34. Confirmation deletes
  the watches as though the new execution had observed them, without retaining those changes.
  The parent must reconstruct the wait from conversation history and individually inspect children;
  the promised durable continuation context is absent. A serialized MCP probe runs wait → child
  failure → notify → yield release → re-admit/confirm → assignment read and gets no child id,
  revision or outcome. Preserve a bounded authorized continuation snapshot through confirmation
  and expose the watched changes/results through the existing assignment read; test early/late
  reads, multiple/deleted sources, restart and scope isolation. Evidence:
  `/tmp/asre_assignment_probe_test.go`, `/tmp/asre_assignment_overlay.json`,
  `TestASREAssignmentExposesWatchedResultAfterContinuation` (intentionally failing).
- **Must fix** — ASRE-2 — **Task report outputs bypass the result size limits**
  (INV §2/§11/§16). `internal/state/tasks.go:1677–1733,2116–2125`,
  `internal/state/pipeline_tasks.go:224–285`: reporting validates summary/details but never
  bounds the output map's values or aggregate result. The shared insert helper checks only that
  names are nonblank; ordinary reports store arbitrary output keys/values, and stage reports
  check declarations/required values without the generic size checks. TS-10.R33 requires the
  pipeline text limit (64,000 runes per value) and a 256 KiB aggregate result limit. Serialized
  ordinary-task MCP probes accept and persist both one 64,001-character output and five
  individually legal 60,000-character outputs (300,000 bytes), marking each task finished/success.
  A large generated artifact can therefore become immutable retained result data and be loaded
  whole by task reads/lists and downstream consumers. Extend shared report validation to named
  outputs and aggregate encoded size before committing any result, with a typed non-mutating
  refusal. Regress individual and aggregate boundaries, Unicode/encoding overhead, and ordinary
  versus stage reports. Evidence: the same temporary overlay's
  `TestASRETaskReportRejectsOversizedOutputs` (both cases intentionally failing).

Connected evidence ledger and J3 (authority through native execution):

| Retained topic | Whole-chain coverage and result |
|---|---|
| 16 — Tools/hooks/authority/results | Traced token → caller identity → scoped action → committed effect → text/structured response, MCP/hook setup/revoke, owner-only artifacts and bounded hook network failure. Fresh/resume/wake/switch/rollback/crash use shared registration and teardown. Tool discovery, schemas, refusal classification and structured/text parity are producer-checked; paused direct actions are not advertised as shipped. |
| 17 — Peer mail/delivery/budgets | Traced resolver → transactional insert/budget/activation → lifecycle claim → attempted boundary → native prompt → receipt/read settlement. Waking mail coalesces; deferred mail creates no turn and retains unread content. Whole-message/byte/budget bounds, waking-first ordering, stable ids, ambiguous delivery recovery and restart activation retirement are covered. No additional mail defect established. |
| 18 — Tasks/delegation | Traced human/agent create and frozen backend/model/effort/fast/group → durable capacity/assignment claim → launch or existing/wake activation → read/report/wait → terminal/yield cleanup and continuation/restart. Creator/managed-run scope, generation/handle checks, dependency failure/repair, late reporting, and Chucky's bounded guidance/group placement were inspected. ASRE-1/2 break assignment context and retained-result bounds. |

Peer Stop's provider permission exemption admits only the MCP call. Its separate host waiter
requires operator Approve once, captures requester generation/turn and target generation/PID/start,
and admits one effect before the ordinary quota/lifecycle Stop path. Cancellation/deny/timeout,
requester replacement and target replacement refuse or invalidate the request. Durable decisions
precede release; result envelopes distinguish unchanged/stopped/unknown effects and forbid automatic
retry. Task creator/assignee and room membership never resolve that host gate. This source/test
trace does not prove native providers preserve the gate under their most permissive modes.

J4/J5 challenge: task reports commit outcomes/release intent before downstream readiness; runtime
release waits for the reporting turn and stops only created/woke ownership, while borrowed cancel
is generation/turn guarded. Pending yield and cleanup reconciliation retain ownership through
restart; stopped/waiting work uses ordinary capacity and closure admission. The suspected missing
closure predicate in `CompleteTaskYield` is not a finding: admission refuses a closed run and
durable run cleanup cancels the transient ready/waiting row. Ordinary reports' optional execution
handle is the explicit TS-10.R31 compatibility contract, not a new defect. Mail/task inputs use
the common bounded prompt seam; task attachments retain assignee-derived grants. Room and pipeline
tools share caller/result authority and ordinary launch/continuation seams; their supervisory state
machines remain separate audits. Clone/fork gets a new registration and no transferred assignment;
terminal execution has no usable coordination/task tools; isolated quota checks omit them.

Verification: full messaging, hooks, state and agent-knowledge packages pass. Focused runtime/server
host-approval, peer Stop, registration, mail activation, task dispatch/control/report/recovery
tests pass; focused state/runtime race checks and FTS5 state/MCP/server task/mail/Stop checks pass.
Spec lint and diff checks pass. Initial sandbox loopback denials were resolved
with approved isolated loopback execution. Temporary probes above fail against the reviewed source
and preserve the serialized failing responses. No provider credentials, model turns, UI or devices
were exercised: FS-16.A33/A34, FS-06.A29/A30 and existing native task/room adoption gates remain owed.

Existing findings retain their identities: ASRC-1 can stall the runtime read loop and ASRC-2 can
hide permission-wait status; ASRD-2/3 affect event order/durability, including tool/approval history.
CI-1's cancellation-double closure is not contradicted; PROMPT-2 still limits complete-instruction
diagnosis and provides no causal quality result. Invariant sweep: §1–5, §7–12 and §14–17 applicable
and checked; §6 has no new interface/runtime and §13 has no markup/style surface. No local
implementation choice or human decision was introduced; sibling reviews/fixes remain unselected.

### ASR-F — Pipelines, Think Tanks and orchestration — reviewed `b8157f13` — **Fix model:** medium — Codex Terra or Claude Opus.

Recovery needs coordinated controller/dispatcher changes at the existing durable-action seam;
removing the obsolete execution methods and updating their tests is localized. Audit complete;
findings keep this same unit open. Source mode: current repository at `b8157f13`, clean at entry.
Release lockfile pins Claude ACP 0.85.1 / Agent SDK 0.3.286 and Codex ACP 2.1.1 / CLI 0.159.3;
this audit exercised source and isolated fake-provider tests, not installed/native provider behavior.
Two bounded readers traced ordinary stages and room lifecycle; the lead integrated the ownership,
interruption and context joins and independently reproduced the recovery failure. Product code and
specifications unchanged.

- **Must fix** — ASRF-1 — **A failed post-result transition loses its recovery owner**
  (INV §7/§8/§9/§15). `internal/pipeline/manager.go:754–761,792–823`,
  `internal/server/task_dispatcher.go:575–600`, `internal/pipeline/reconcile.go:142–185`:
  after a stage result and runtime release commit, startup reconciliation can fail while creating
  the next stage or recording the final/approval boundary. `pauseStartupRun` then replaces
  `finishing/release_stage_task` with `paused/""`. The successful current task permits neither
  Continue nor Retry, cleanup repair is ineligible, and further reconciliation/restart skips the
  lost action. A temporary store/transition failure can therefore strand an otherwise completed
  stage permanently. Related live-path source tracing shows that the final cleanup callback only
  logs a controller error after clearing the task's pending release; the cleanup ticker no longer
  selects that task, and the room-output sweep does not retry ordinary-stage transitions.
  FS-14.R66/R71/R77 and TS-09.R41–R43 require retained, automatically retryable bookkeeping and
  valid recovery. Preserve the durable action until progression commits and give failed controller
  actions bounded retry/recovery ownership through the existing dispatcher; retain the current
  stopping fence and never rerun accepted stage work. Regress transient next-task/final/approval
  writes both at runtime and restart, including Stop racing recovery and exhausted retry attention.
  Evidence: `/tmp/asrf_startup_probe_test.go`, `/tmp/asrf_overlay.json`,
  `TestASRFStartupPreservesAcceptedStageProgression`. The probe accepts success, settles release,
  injects an abort at next-stage insertion, runs Startup, removes the fault and reconciles again:
  `paused`, empty pending action, one stage, Continue/Retry refused and cleanup repair false.
  The injected case fails in both Go variants; the same fixture without the fault passes and
  creates the queued second stage. The runtime callback gap is source-backed, not a separate
  provider-failure receipt.
- **Worth fixing** — ASRF-2 — **The removed stage executor survives in production source and tests**
  (INV §2/§10/§17). `internal/server/pipeline_lifecycle.go:236–355,382–384`,
  `internal/pipeline/manager_types.go:101–104`, `internal/server/pipeline_fast_test.go:15–55`,
  `internal/server/pipeline_handlers_test.go:833–852`: a tree-wide caller audit finds no production
  caller of `LaunchStage` or `ContinueStage`; `IsRunning` serves only the latter. Yet the public
  lifecycle interface still requires the old launch/resume machinery, and tests use it as evidence
  for pipeline fast-mode/group behavior without creating a run or dispatching a stage task. The
  group test even asserts the superseded stage-group contract. TS-09.R35/R46 requires task-owned
  execution and removal of those paths; TS-12.R4 still describes them as active. Keeping this code
  duplicates lifecycle policy and preserves acceptance tests that cannot detect a broken real
  pipeline-to-task handoff. Remove the unused methods/interface obligations, retain `StopStage`
  used by task cleanup and legacy-runtime reaping, and exercise actual run → stage task → dispatcher
  behavior for frozen/applied fast mode and current run grouping (FS-14.R72/R74). Update the stale
  lifecycle tests/docs at the same seam. Ordinary-task fast-mode tests already cover the shared
  dispatcher, so the missing proof is the pipeline handoff, not all fast-mode behavior.

Connected evidence ledger (catalogue topics 19 and 20):

| Retained topic | State, authority and effect trace |
|---|---|
| 19 — Ordered stages and standing orchestration | Frozen template/runtime/input snapshot → unique stage task and optional managed coordinator child → ordinary task capacity/identity claim → launch or same-identity resume → assignment/report/wait → immutable stage result and closure → release/cancel descendants → approval/next task/final result. Read Start/Continue/Retry/Replace/Stop, dispatcher, report transaction, lineage projections, cleanup and startup. A coordinator's child result cannot substitute for the standing stage report; replacement retains coordinator binding and predecessor work. ASRF-1 breaks recovery after accepted work; ASRF-2 retains the old executor. |
| 20 — Rooms and judge capture | Room/stage binding and reserved new identities commit before setup launches. Independent openings use bounded concurrent starts and withheld publication; discussion uses one floor. Read receipt/head/token plus exact generation/executing turn bind staged reply to successful completion. Completion atomically publishes the judge entry and retained result; trusted stage acceptance binds that exact entry and named output once, followed by ordinary cleanup/approval/progression. Failed/oversized/staged replies never become accepted synthesis. Inspected setup/turn/judge retry, ceilings/pass/leave, End/Pause, capture/privacy, queued input, restart and output-acceptance sweep. |

J3 authority: token-bound membership and task/handle/generation authority remain separate. Only the
standing stage task reports ordinary stage success; room success uses the trusted bound finalized
synthesis. Neither scope grants host peer-Stop approval. Room mutation/admission guards recheck
pipeline openness in their write transaction; Stop commits closure before cancellation/teardown.
Claimed setup may settle registration, then only its captured idle generation is stopped; an active
room turn uses guarded generation/turn cancellation. Completed room release preserves private work.

J4 interruption: traced quota failure → preserved task/room owner → exact continuation claim → new
room attempt/token or ordinary task activation → actual completion. Pause/End/Stop and stage closure
revalidate before delivery; quota cannot synthesize a report or successor. Startup task recovery and
room recovery fence uncertain work, retain withheld/published history, and recover finalized output
acceptance without rejudging. ASRF-1 is the missing post-result controller recovery. The recorded
`TestOrdinaryStageAgentStopPausesPipelineRun` 409 did not reproduce in 25 isolated repeats or the
focused suites; its intermittent cause remains unproven and the existing note remains live.

J5 context: ordinary stages receive bounded persisted assignment, prior-result source ids, named
inputs and coordinator identity; shared operator instructions are frozen once in run-created
orchestrator prompts and retained by ordinary resume/switch composition. Room stage context uses
the same paged read/receipt seam for participants and judge. Public capture excludes private prompts
and projects internal Chuck calls/results to action receipts; source generation/turn remains sealed
before held/private successors start. Native provider context adoption remains unproved here.
Clone/fork creates no stage/room assignment; switch/rollback reuse the shared conversation lifecycle.
Supervisory participants are chat-only; terminal and isolated quota-check processes cannot be stage
assignees or room actors. Existing ASRA-1 configuration-continuation, ASRC-1/2 runtime/permission,
ASRD-2/3 event-order/durability and ASRE-1/2 task-context/output-bound findings retain their identities
and apply through these shared seams; this audit does not close them. PROMPT-2 and CI-1's recorded
limits/closure are unchanged. No measured agent-quality or harness-cost conclusion is claimed.

Verification: focused pipeline/state/server tests pass in both untagged and `sqlite_fts5` builds;
focused pipeline/state/server race tests pass. Server runs used approved isolated loopback listeners
after the readers' sandbox listener denials. The temporary failure probe above is intentionally red;
its no-fault control passes. Spec lint and diff checks pass. No browser/device/provider/model turns
were exercised: FS-14.A49/A50/A54, FS-21.A31/A32 and the existing real-provider/permission/resume
gates remain owed. No visual implementation changed and no rendered gate is closed by this audit.
Invariant sweep: §1–5, §7–11 and §14–17 apply to lifecycle, frozen state, claims, reads, wire data,
authority, bounded capture and verification; all were checked, with findings tagged above.
§6 has no new runtime/driver contract, §12 has no supervisor-owned external-CLI invocation and
§13 has no selected markup/style change. No local implementation choice or human decision was
introduced; sibling review/fix queues remain unchanged.

### ASR-G — Failure, quota and continuation recovery — reviewed `fa092f63` — **Fix model:** medium — Codex Terra or Claude Opus.

Audit complete; findings keep this same unit open. The fixes extend existing admission,
turn-ownership, room-abandonment and bounded-sweep seams; coordinated state/regression work is
needed, without a new scheduler or provider client. Source snapshot `fa092f63`; tree clean at
entry. Profiles: lockfile-pinned Claude ACP 0.85.1, Agent SDK 0.3.286/native 2.1.286, Codex ACP
2.1.1 and Codex 0.159.3. Two bounded read-only audits were independently integrated; no product
code or specification changed.

- **Must fix** — ASRG-1 — **Quota room startup failure leaves a phantom running opportunity**
  (INV §2/§4/§15). `internal/server/quota_dispatch.go:154–180,216–236`,
  `internal/runtime/chat.go:1644–1696`: the quota activation's `before` callback commits a linked
  room attempt and starts capture, then prompt preparation or busy-status persistence can fail
  before any provider prompt. The quota caller only holds the quota row; it never abandons that
  exact attempt/capture. The room retains a running turn with no provider to finish it, obstructing
  further recovery (FS-01.R42/R44, TS-10.R42, TS-14.R6). Ordinary room dispatch already has the
  compensating `abandon` path (`think_tanks.go:262–297`); extend that seam to quota continuation
  for running and resumed agents. The temporary server probe injects a failure only into the
  activation's status write: zero provider prompts, quota needs-attention, one running room attempt.
  Regress prompt-preparation/status/start failure and assert exact attempt/capture cleanup with
  no published contribution, charged allowance or replacement work.
- **Must fix** — ASRG-2 — **Unknown-reset checks spend allowance before task capacity is available**
  (INV §2/§5/§15). `internal/server/quota_check.go:194–251`,
  `quota_dispatch.go:24–38`, `internal/state/quota_checks.go:31–45`: check admission validates
  policy, the episode, agent/project and provider, then charges a check without the original
  work's lifecycle/owner/capacity admission. Mutex keys are serialization, not eligibility.
  A stopped borrowed task with a full task runtime budget therefore spends one of its six
  checks and can issue a paid probe even though its continuation must defer. The deterministic
  server probe observes `admitted=true, checks_started=1` at a budget of one already occupied
  by another assignment. This violates FS-01.R49/R50 and TS-10.R45/R46's no-check-on-capacity
  rule. Share a side-effect-free eligibility check with continuation, revalidate under admission,
  and defer without charging/reserving a runtime or changing owner state. Regress this capacity
  case plus lifecycle contention and current task/room ownership; accepted owner controls must
  still cancel the exact episode. Do not infer extra paused-room failures from source alone:
  successful existing controls normally invalidate those schedules.
- **Must fix** — ASRG-3 — **Cancel can settle a late result as successful quota recovery**
  (INV §5/§11/§15). `internal/runtime/permission.go:190–220`, `quota.go:80–90,147–174`:
  Cancel claims an active turn and sends the native notification without retaining a cancelled
  marker. A same-turn `end_turn` result arriving afterward still qualifies as success, calls
  successful quota settlement and clears the runtime quota block. FS-03.R9 requires the claimed
  cancellation to terminate with `cancelled`; provider completion/cancel races must not falsely
  confirm continuation success. The temporary runtime probe calls Cancel, then supplies the
  same turn's late completion, and observes `successful:true`. This proves host state behavior,
  not a captured native race. Record/fence cancellation under the turn gate and revalidate at
  settlement; test the terminal reason, quota receipt/block and inline-mail behavior. Held
  follow-ups remain allowed after a cancelled turn under FS-03.R49. This is separate from
  ASRC-2's late status writes and CI-1's already-fixed sticky fake cancellation.
- **Must fix** — ASRG-4 — **An isolated Codex check retains explicit work instructions**
  (INV §2/§10/§12). `internal/runtime/quota_check.go:501–575`: isolation reads the inherited
  `CODEX_CONFIG` but changes only MCP entries; with no servers it returns the environment
  unchanged. A backend/model overlay containing `developer_instructions` therefore carries
  that work guidance into the supposedly minimal check process. The sentinel overlay probe
  confirms preservation. This violates FS-01.R51 and TS-04.R97's omission of work instructions;
  a check can be steered away from its fixed request and needlessly consume allowance/usage.
  Sanitize instruction-bearing configuration while retaining proved authentication/routing,
  or refuse unsupported isolation. Regress existing overlay and native-config instruction
  sources with routing preserved and MCP disabled. The current test only asserts no generated
  instruction key for a fixture that supplied none; no live provider adoption is claimed here.
- **Worth fixing** — ASRG-5 — **Older cleanup failures starve later probe cleanup indefinitely**
  (INV §7/§9/§16). `internal/state/quota_checks.go:122–139`,
  `internal/server/quota_check.go:256–289`: the database always selects the oldest eight
  references, then the server skips in-flight/cooldown entries or leaves failed attempts at the
  same ordering position. Eight persistently unresolved references can therefore prevent a ninth
  healthy cleanup from ever being selected, keeping its conversation undeletable and further
  checking blocked (TS-02.R50, TS-10.R47). The nine-reference probe repeats three sweeps and
  leaves the ninth untouched; freeing one earlier batch slot lets the same cleanup succeed.
  Keep bounded work but advance fairly over skipped/failed rows, with per-reference backoff and
  no unrelated-history scan. Regress more than one batch with permanent failures/cooldowns
  ahead of a cleanable session.

Failure/admission/retry ownership map and catalogue 22 evidence ledger:

| Boundary | Authority, durable state and recovery result |
|---|---|
| Provider/startup/model/transport failure | Shared selected-source launch/resume/switch composition and typed provider errors own preflight; no automatic model/source/permission substitution or prompt replay. Ordinary and activation prompt completions share terminal failure decoding and settlement. Crash/Stop teardown and restart reconciliation own process loss. Existing ASRA-1/3, ASRB provider/terminal findings and ASRC-1/2/3 remain applicable. |
| Terminal quota → automatic work | Complete bounded AIR policy plus executing root generation/turn establishes quota; warnings/retry/new-session/context limits and arbitrary prose/429 do not. Source-owned observation and owner pause commit before turn-end consumers; held input, mail/task activations and room scheduling stay blocked. Reset parsing uses only applicable failure/rate-limit evidence; reset and due are distinct, with a 60-second buffer. |
| Due intent → isolated check | Indexed due order, revision/token CAS and eight shared slots; unknown resets use one-hour checks with a six-check chain allowance, intervention errors remain manual. Probe uses temporary cwd, separate ACP process, fixed request, no Chuck identity/event sink or work payload. Claude supplies only routing settings with non-persistence; Codex renames before its model request and archives the exact native session. ASRG-2/4 break admission/isolation; same-account/model adoption and retention remain live gates. |
| Check → original-work continuation | Session reference commits before prompting; result follows cleanup. A ready check only authorizes the same chain and selection, never a result, notice or quota clearance. Token invalidation cancels the check context; stale completion cannot authorize continuation. Ready-selection mismatch/restart holds; unknown renewed quota after a ready check holds instead of looping. ASRG-5 affects cleanup fairness. |
| Continuation → task/stage/room | Ordinary activation or native-session resume preserves logical agent/history; task assignment/capacity and pipeline lineage remain authoritative. Room admission creates a linked attempt with fresh read/submission authority and preserves the failed staged disposition as history. Accepted results/opportunities are not replayed. ASRG-1 breaks pre-provider room compensation; ASRG-3 breaks cancellation settlement. |
| Human control / crash / storage | Successful Send/Steer/Stop/lifecycle and owner controls serialize with quota admission and cancel only captured episodes; refused controls retain them. Global disable cancels unstarted checks/continuations, and startup holds uncertain delivery or interrupted checks without replay. Source progress/success receipts precede an idempotent fsynced Chuck notice; repair never re-prompts, and storage failure holds. Temporary fault probes cover the new gaps; existing quota control/notice/restart tests pass. |

J4 lead conclusion: quota preserves the same task/stage/room owner before generic failure/restart
handling, then uses existing activation/resume and exact execution tuples. ASRG-1/2/3 qualify that
conclusion; ASRF-1's accepted-result transition recovery retains its identity. Stop, provider
refusal and transport loss do not synthesize success or release a successor from an accepted stale
task report. Parent run controls include borrowed-task and room lineage. Claude terminal/viewer
detach uses the PTY lifetime, not chat quota automation; terminal quota checks/owned room-task
activation are unsupported. The previously recorded terminal native-id/driver recovery defects
under ASR-B remain open. Fresh/ordinary/wake launch, task activation, pipeline and room paths were
traced; clone/fork creates no quota-owned task/room assignment, and switch/rollback uses the
shared explicit lifecycle path rather than quota replay.

J1/J6 challenge: check fingerprints cover the live frozen or stopped resume-derived selection;
accepted manual supersession invalidates the pending source episode. ASRA-1/3's continuation
configuration defects retain their identities, and native account/config adoption is not proved
by a fingerprint. TS-04.R96 supersedes R87's earlier no-parser/unverified-unit exclusions: the
existing pinned producer/formatter receipt establishes supported semantics, not a live reset or
continuation. Missing, ambiguous, stale and unsupported windows remain unknown; passive paired
quota views remain planned. PROMPT-2's diagnostic limit and CI-1's closure are unchanged. No
agent-quality/performance inference is made.

Verification: focused server quota, room-stage and resume-failure tests pass; runtime/state quota
and CI-1 regressions pass under `-race` (`CHUCK_RUNTIME_ROOT` unset, `GOFLAGS=-p=1`). The five
required-behavior probes intentionally fail on current source; the cleanup batch-slot control
passes. Probe files are `/private/tmp/chuck_asrg_server_probe_test.go` with
`chuck_asrg_server_overlay.json`, and `/private/tmp/chuck_asrg_probe_test.go` with
`chuck_asrg_overlay.json`; Go overlays leave repository tests unchanged. Server probes ran with
approved isolated loopback listeners after the sandbox bind denial. Focused server/state quota
tests also pass under `-race -tags sqlite_fts5`. No live provider, device or browser journey was run:
FS-01.A28/A31/A33/A34 and existing terminal/handoff/native-adoption gates remain owed.

Invariant sweep: §1–5, §7–12 and §14–17 apply to lifecycle, composition, persistence, teardown,
admission, recovery reads, projections, protocols, CLI isolation, authority, effects, bounds and
tests; all were checked, with findings tagged above. §6 has no new runtime/driver interface;
the existing chat/terminal/check capability boundaries were traced. §13 has no selected
markup/style change; quota banner selectors exist. No visual gate is closed, no new local
implementation choice or human decision was introduced, and sibling queues are unchanged.

### ASR-H — Human intent across desktop, phone, CLI and HTTP — reviewed `d49a8f86` — **Fix model:** medium — Codex Terra or Claude Opus.

Audit complete; four Must-fix findings keep this same unit open. The hardest fixes coordinate the
existing hold/Steer and client recovery contracts; no new scheduler or provider client is needed.
Source snapshot `d49a8f86`; tree clean at entry. Repository source and release-lock profiles:
Claude ACP 0.85.1 / Agent SDK 0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. Three bounded discovery
audits were integrated; the findings below were independently checked against source and probes.
No product code or specifications changed.

- **Must fix** — ASRH-1 — **A successful text Steer erases a newer desktop draft**
  (INV §1/§5). `ui/src/components/chat/Composer.tsx:290–305`: while text-only Steer awaits its
  response, the textarea remains editable. Success unconditionally clears both the live textarea
  and `discardChatDraft(agentId)`, deleting text typed after submission; ordinary Send and
  attachment Steer already check the submitted draft revision. This violates FS-03.R36/A19.
  The delayed-success component probe types a second draft before resolving the request and
  observes an empty textarea instead. Capture/recheck the draft revision and current owner before
  clearing text or changing visible feedback; regress same-agent edits and navigation away/back.
  Fix complexity: easy; localized to the existing composer ownership seam.
- **Must fix** — ASRH-2 — **Stale Withdraw removes one queued message but restores another**
  (INV §1/§5/§11). `ui/src/lib/heldMessage.ts:24–32`,
  `internal/server/sessions.go:84–102`, `internal/runtime/chat.go:571–583`: the browser saves its
  cached held text A, then parameterless DELETE removes whatever is currently held. If another
  device has replaced A with B, the successful response contains no withdrawn identity/content;
  the browser restores A while B has been removed. A turn ending before DELETE similarly lets
  a no-op success restore text already delivered. FS-03.R90 requires returning the withdrawn
  message for editing, not an obsolete cached message. The actual server probe confirms B is
  removed with a content-free response; the UI probe confirms A is restored. Make withdrawal
  and recovery refer to the same authoritative hold (conditional identity/version or an atomic
  returned message), preserving the documented idempotent no-hold behavior. Regress two-device
  replacement and turn-end races, text/attachment combinations, and overlapping withdrawals.
  Fix complexity: medium; coordinate the shared runtime/API and both clients.
- **Must fix** — ASRH-3 — **Bare CLI launch treats an uncertain archive lookup as no match**
  (INV §7/§8/§11/§16). `internal/cli/launch.go:133–159`, `internal/cli/resume.go:85–117`:
  the legacy `role@project` form silently falls through to a new launch after any lookup error.
  Its successful lookup also reads only the first 200 inactive project rows, ignores `total`,
  and filters role/name locally. An older match beyond that page is missed; one apparent match
  can be selected despite a second match on a later page. This violates FS-01.R11's exact match
  rule and FS-22.R1's legacy behavior preservation, creating a new conversation or selecting one
  without establishing the required uniqueness. Both isolated CLI probes observe a new-agent
  POST after a 500 or a valid 200-row page with `total:201` and the match on page two. Surface
  lookup failure; use bounded pagination or require explicit selection/`--new` when completeness
  cannot be established. Regress errors, late matching names/roles and two matches across pages.
  Fix complexity: medium; preserve exact legacy selection with bounded, complete evidence.
- **Must fix** — ASRH-4 — **Interrupted promoted Steer can automatically repeat consumed input**
  (INV §5/§11/§15). `internal/runtime/chat.go:682–717,843–870,1191–1222`: an empty text Steer
  takes the held text, but a cancelled HTTP/CLI request context can end its wait after native
  consumption and before acknowledgement. With no attachment/request receipt, the ambiguous
  error restores the message to the live hold; ordinary turn completion then submits it again
  automatically. No human retry is needed. FS-03.R50 and TS-03.R39/R41 preserve the deliberate
  Steer operation; an unconfirmed acknowledgement cannot authorize another consumption. The
  controlled fake provider applies an injected Steer, withholds its success reply, then sends it
  late after context cancellation. Its wire logs show one Steer and a second ordinary prompt
  containing the same text. Restore automatic holding only after definite no-consumption;
  preserve ambiguous input for explicit recovery without releasing it as a queued successor.
  Regress context loss before/after consumption, late acknowledgement, original turn-end races
  and definite refusal, with provider submission counts and durable transcript observations.
  Fix complexity: medium; coordinate existing Steer/hold ownership and visible recovery.

Catalogue 23 action map and connected evidence ledger:

| Action / entry point | Supported contract and traced outcome |
|---|---|
| Launch | Desktop, CLI and local HTTP share session launch/composition. Phone uses the same handler with chat-only, secret-free runtime choices and the narrow field allowlist; terminal/worktree/permission-bypass choices remain excluded. Desktop defaults, explicit false/config merges, fixed project/group and refusal retention were checked. ASRH-3 affects legacy CLI auto-selection; explicit new launch does not use that lookup. |
| Send / hold / withdraw / Steer / attachments | All four entry points reach shared runtime turn ownership; Send holds behind a busy turn, Steer requires advertisement and preserves its reported injected/new-turn outcome. Attachment bytes, request identity, admission, receipt replay, Stop/restart recovery and explicit recovered sends were traced into ACP and durable records. ASRH-1/2/4 break draft/withdraw/ambiguous recovery. Attachment receipts do not promise idempotency for ordinary text-only retries; that proposed broader finding was rejected. |
| Stop / Cancel / Resume / switch | All supported entry points use shared lifecycle, frozen resume/wake and typed refusal. Phone runtime changes remain chat-only; local desktop/CLI/HTTP own terminal/interface changes. Shared task/pipeline/room/quota admission remains authoritative; reconnect never replays a mutation. Existing ASR-A/B/C/G lifecycle/provider findings retain their identities. |
| Settings / permissions | Desktop, CLI and local HTTP expose live effort/fast and provider mode choices; phone exposes effort/fast but denies provider-mode routes. All use advertised capabilities and shared apply/persist behavior. Ordinary tool/host peer-Stop decisions preserve exact pending identities and single-winner settlement; failed decisions remain visible. ASRC-1's config read-loop stall and ASRC-2/5's status/permission ordering remain open. |
| Annotation / file / navigation | Desktop and phone capture event/diff/file excerpts and anchors before instruction focus changes. Shared source-token delivery, source-scoped bounded drafts, launched-recipient retry and stale read fences were traced; file paths/excerpts remain point-in-time context rather than new grants. CLI/local HTTP use the same validated delivery/read boundaries. Phone tracked-file authority, terminal exclusions and newer drafts were checked. ASRD-4 remains the lost-annotation-response duplicate; it is not duplicated here. |
| Recovery / observation | Phone generation/hydration gates retain cached views, text/files and route ownership while disabling mutations; one foreground owner bounds health/SSE retries. Desktop SSE and selected-agent panels fence stale reads. CLI follow subscribes before catch-up, preserves durable sequence order and exits on Ctrl-C/EPIPE without Stop/Cancel. Fake-clock/component and built-CLI journeys pass; real paired-device suspension/native selection remains unproved. |

J6 lead result: accepted attachment retries preserve their receipt, recovery reads do not repeat
withdrawal, and reconnect refreshes current truth without automatic sends. ASRH-2/4 expose missing
ownership/acknowledgement boundaries; ASRH-1 loses newer intent and ASRH-3 changes CLI intent on an
incomplete read. J2/J4 challenges retain the existing permission/status, quota and owner-admission
findings. ASRC's promoted-Steer hypothesis is confirmed as ASRH-4; its statement that the UI always
sends `request_id` is corrected here: `api/client.ts` omits it for text-only Send/Steer, and ordinary
CLI text also has no receipt. This finding concerns automatic re-consumption, not a universal text
retry guarantee. PROMPT-2's diagnostic limit and CI-1's prior closure remain unchanged.

Verification: 198 focused UI tests pass across Composer, permission modes, annotation tray, drafts,
launch, agent store, desktop SSE, phone agent/recovery and pipeline controls; presentation checks
also pass. Focused server action/remote/mode checks pass normally and under `-race -tags sqlite_fts5`;
attachment admission/receipt tests and runtime permission/hold/Steer/CI-1 checks pass under `-race`.
The CLI built-server/follow/legacy-filter journeys pass under `-race`. `CHUCK_RUNTIME_ROOT` was
unset; temporary homes/fake ACP were used. Sandbox loopback denial was resolved by approved
isolated execution. New failure probes intentionally expose the missing behavior: two CLI
assertions, desktop newer-draft assertion and consumed-Steer assertion; stale-withdraw probes
characterize the inconsistent server/client outcome. Evidence: `/private/tmp/operator_probe_test.go`
with `operator_overlay.json`, `chuck_asrh_composer_probe.test.tsx` (temporarily placed beside
Composer to run Vitest, then removed), `chuck_asrh_server_probe_test.go` with
`chuck_asrh_server_overlay.json`, and `chuck_asrh_steer_probe_test.go` with
`chuck_asrh_steer_overlay.json`/`chuck_asrh_fakeacp_main.go`. No live provider, browser rendering,
tailnet or paired device ran; existing independent-render/native anchoring, annotation selection,
permission-mode/peer-Stop, handoff and phone background-return gates remain owed. Fake provider
consumption models the lost acknowledgement; it is not a captured Claude/Codex race.

Invariant sweep: §1–5, §7–12 and §14–17 apply and were checked across lifecycle, forms, registration,
atomic control, reads, truthful feedback, persistence, wiring, protocols, native boundaries,
authority, effects, bounds and tests. §6 has no new runtime/interface contract; existing capability
exclusions were checked. §13 has no selected markup/style change, and presentation checks pass;
no visual or device gate is closed. No local implementation choice or human decision was introduced.
Sibling reviews/fixes remain unchanged; the subsequent whole-system synthesis is below.

### ASR-I — Whole-system synthesis and agent performance — reviewed `8e03e709` — **Fix model:** difficult — Codex Sol.

Source synthesis complete; no new independent defect or duplicate finding is added. The
recommendation reflects the hardest linked open fixes (ASR-A/B's native contract and driver
ownership), not a new fix unit. Existing finding identities, severities and individual fix-unit
recommendations remain authoritative. Their fixes need not precede this review's closure, but
the source audit does not close any native, device or behavioral acceptance gate.

Source snapshot `8e03e7091633f717b13447f1191cd30bf7d993d2`, clean at entry. The entire range
`4fbeb316..8e03e709` changes only this handoff: no intervening product/spec edit invalidates A–H's
probes. Source mode is repository implementation plus the retained pinned-native traces, not an
installed-provider execution. Pins remain Claude ACP 0.85.1 / Agent SDK 0.3.286 and Codex ACP
2.1.1 / CLI 0.159.3; release assembly applies `2.1.1+chuck.1`, unlike the source PATH adapter.
Three bounded read-only discoveries were integrated with direct checks of the callers and
failure owners below. Product code and specifications were unchanged.

All retained catalogue topics are accounted for:

| Topics / domain | Reconciled evidence and remaining risk |
|---|---|
| 01/02/25 — A | Prompt/native guidance, frozen selection and checkout readiness compose through the shared launch/resume/switch seams. ASRA-1–6 and reopened PROMPT-1 remain; no prompt accumulation or unconsented checkout deletion is established. |
| 03/04/05/21 — B | Native new/load/fork, installed/bundled selection and PTY/external drivers retain ASRB-1–8. Native fork is not ordinary load; source Codex Steer is not the patched release path. Terminal conversation/process/environment identity cannot be inferred from a chat success. |
| 06/08/09/10/11/12 — C | Runtime claims, held/steered input, RPC callbacks, permission settlement, cancellation and native background events retain ASRC-1–7. CI-1's corrected test double remains closed; passing cancellation tests do not settle the distinct late-success quota issue. |
| 07/13/14/15 — D | Native/history bridge, durable/live ordering, exact input bytes, scoped reads and annotation retry retain ASRD-1–5. Attachment recovery has its own receipt; neither a visible transcript nor a copied bridge proves native consumption. |
| 16/17/18 — E | Token/host authority, mail activation, assignment/report/wait and release retain ASRE-1/2. Peer Stop's host decision is independent of permissive provider settings; live adoption of that boundary remains owed. |
| 19/20 — F | Ordinary stage-task progression and room/judge result capture retain ASRF-1/2, with E/G defects applying through shared services. Obsolete stage-executor tests do not prove the current dispatcher handoff. |
| 22 — G | Quota evidence, check isolation/admission, source-owner continuation and cleanup retain ASRG-1–5. Passive quota snapshots remain planned; missing live receipts do not establish a parser/provider failure. |
| 23 — H | Desktop/phone/CLI/local HTTP intent and recovery retain ASRH-1–4 and ASRD-4. Reconnect refreshes state without replaying mutations; physical-device suspension and native anchoring remain open. |
| 24 — I | Frozen diagnostics, revision/adoption limits, combined failure costs and the paired experiment below were assessed. PROMPT-2 keeps its original owner; no measured general quality or excess-process conclusion is supported. |

Whole-journey assurance ledger (J1–J7):

| Join / interacting journey | Boundary, failure/recovery owner and conclusion |
|---|---|
| J1 — selected contract across fresh/resume/wake/switch/rollback | Frozen base plus runtime-only overlay reaches the provider start seam; model/root/profile/native ancestor defects remain ASRA-1/3/PROMPT-1. `applyKnowledgeOverlay` (`launch.go:434–454`) is rebuilt per process, not every turn. Clone/fork has its separate B2/B3 contract. Account/model/preset adoption remains a native gate. |
| J2 — manual multimodal queue/Steer + permission wait + Stop/resume | Runtime holds retain exact descriptors; Stop releases attachment receipts, and restart makes held/prepared copies explicitly recoverable (`chat.go:1264–1284`, `state/attachment_receipts.go:44–65`). ASRC-1/2/5 break read-loop/status/permission truth; ASRH-1/2 break draft/withdraw ownership. ASRH-4 (`chat.go:682–717,843–870`) restores an ambiguously consumed text Steer to automatic holding. These are existing failures, not a second attachment-loss finding. |
| J3 — Chucky mail/task/report + host approval + late continuation | Durable mail/assignment claims precede native activation; reports commit before cleanup/readiness and old generations cannot authorize current work. Peer Stop still requires its separate host waiter. ASRE-1 loses the watched changes before resumed assignment read; ASRE-2 allows oversized retained outputs. Claude fork's missing loaded session and tmux's stale environment weaken native registration (B2/B6). Late-turn guards have fake/source proof, with ASRC-2's final-status race and FS-01.R52/FS-16.A33 live gates retained. |
| J4 — ordinary/room stage + quota/Stop/restart | Task/run closure precedes cancellation; room capture binds generation/turn/attempt and accepted judge output, not arbitrary private prose. `Manager.Startup` → `pauseStartupRun` clears a failed post-result action (`manager.go:754–761,792–825`, ASRF-1). Quota room admission lacks ordinary-start compensation after creating its attempt (`quota_dispatch.go:154–237`, ASRG-1); G2–G5 retain check admission, cancellation settlement, isolation and cleanup defects. Terminal viewer detach preserves PTY work; terminal quota automation is unsupported and B1/B4/B5 recovery remains defective. |
| J5 — context through delegation/handoff/compaction | Assignment references/grants and room read receipts stay scoped; private room input does not become public judge prose. The bounded bridge persists separately through immediate Stop/wake and compaction, but switch marker append opens a second writer after target Resume (`primer.go:65–81`, `switch.go:266–278`, ASRD-1). D2/D3 affect order/durability and B1–B3 affect terminal/fork context. A4/A6 additionally invalidate assumptions that a recreated checkout is ready. Actual live context adoption is still unproved. |
| J6 — accepted-but-lost desktop/phone/CLI actions | Attachment request hash/receipt precedes native admission (`attachments.go:453–541`); identical explicit retry returns stored acceptance. Text-only sends have no universal idempotency promise. CLI reports uncertainty without automatic replay; phone recovery only rehydrates current truth. ASRD-4 annotation retry and ASRH-4 automatic held replay remain proven duplicate-effect paths; H1/H2/H3 retain lost/stale/incorrect human intent. Phone agent routes are keyed by identity (`PhoneApp.tsx:128`), so an old component's completion is not evidence of a new cross-agent draft bug. |
| J7 — cumulative harness cost and quality | Wrong native/frozen instructions (A/B), repeated input/annotation effects (H4/D4), blocked read loop (C1), lost continuation context (E1), stranded recovery (F1/G1), and unbounded retention (C6/D5/E2) are supported mechanisms for wrong work, extra work or delay. Their incidence, usage and output-quality effects have not been compared with native operation. Persona/repository process pressure is a hypothesis, not an additional defect. |

Conflicting earlier statements are resolved explicitly. ASRC's suggestion that transcript
reordering was tolerated is superseded by ASRD-2's scheduled durable-order probe. ASRC's claim
that the UI always supplies a request id is superseded by ASRH-4: ordinary desktop/CLI text has
none. A's intended no-work quota-check isolation is qualified by ASRG-4's inherited explicit
Codex instructions. CI-1's sticky fake-cancellation bug stays closed and does not explain G3's
late same-turn success after Cancel. PROMPT-1's user/exact-cwd fix remains valid, while its
ancestor-project variant stays open. These qualifications preserve the original finding IDs.

The crash-turn generation lookup noted in ASRC is not promoted to a new failure: the exit hook
already interrupts unreported task/room ownership (`server.go:463–464`), while reported/yielded
cleanup has a durable retry pass (`task_dispatcher.go:466–528`). No loss or duplicate effect
beyond the retained findings was established. The intermittent ordinary-stage Stop 409 remains
unresolved; isolated passing runs do not close that note. No contradictory product requirement
was silently selected; ASRC-7's outcome drift is now closed by ASW-3; ASRF-2 remains open.

Diagnostic sufficiency and bounded performance experiment:

`runtimeMeta` (`chat.go:2387–2418`) retains frozen prompt/hash and requested settings, but not
runtime suffix/skill/tool/package identities or provider adoption (PROMPT-2). Event generation
and turn are in-process identities excluded from durable JSON (`event.go:44–52`). Live
`busy_since`, transcript events and provider usage do not constitute an archived per-turn
latency/retry/native-call manifest. Missing correlation is an evidence limit, not an invented
requirement for a telemetry platform.

Start with three randomized paired repetitions for each of four fixed task classes (question,
small fix, known-defect/non-defect review, coordination), separately for Claude and Codex:
native baseline versus Chuck with empty persona/awareness off, at most 48 task executions.
Hold repository commit, executable/model/effort/permissions, guidance, tools/context, input
bytes and fresh/resumed state equal within pairs. Use disposable worktrees and no publishing.
Equivalent coordination tools are required in the baseline, or report that cell as confounded.

If that first pass identifies a signal, allow a separate capped 24-execution ablation using
representative tasks to vary awareness/persona and repository guidance independently. An
awareness-off Chuck session still has tools, installed skills, project/native instructions
and task-specific context; it is not the native baseline. Retain native fresh/resume adoption
receipts separately under FS-18.A13; complete FS-18.A12's six qualitative scenarios as written.

Capture redacted complete requested layer hashes/revisions/sizes, executable/provider source,
observed versus requested settings, generation/turn, submission/tool/delegation/retry counts,
available usage, elapsed wall time and answer/brief length outside production instrumentation.
Blind correctness/coverage/scope/unnecessary-work scoring to condition, randomize order and
report paired deltas and variance. Separate provider failures, harness defects and repository
process mandates. This small sample cannot establish universal quality or absence of harm.

No credentialed model turn, paired benchmark, rendered browser or physical-device run occurred.
The live gates above remain explicit; source/fake success is not substituted for them. No prompt
rewrite, benchmark service, production instrumentation or refactor was selected. Local-choice
outcome: retain the current frozen/runtime split and progressive skill delivery; native contract,
recovery and diagnostic fixes remain independently selectable under their existing units.

Invariant sweep: §1–5, §7–12 and §14–17 apply across the lifecycle, instruction/config, authority,
durability, recovery, protocol, bounds and evidence joins, and were reconciled with the tagged
findings above. §6 has no new runtime/interface to enroll; existing chat/terminal/fork/check
capability exclusions were traced. §13 has no selected markup/style change; source selectors and
presentation checks do not close a rendered/device gate.

Verification: runtime held/Steer/attachment/late-tool checks pass under `-race -count=3`;
CLI uncertainty/receipt checks pass under `-race`. FTS5 server receipt/recovery, stopped-mail
activation, immediate-resume bridge, explicit-model resume and permission-deny checks pass under
`-race`. The sandbox denied the server fixtures' IPv6 loopback bind; approved isolated execution
then passed. Source-build tests unset `CHUCK_RUNTIME_ROOT`, with `GOCACHE=/tmp/chuck-gocache-asri`
and `GOFLAGS=-p=1`. Composer/AgentScreen/PhoneApp pass 113/113 tests and presentation checks;
jsdom scroll and unmatched fixture warnings do not constitute browser/device proof. Spec lint
and diff checks pass. These existing positive tests do not exercise or overturn the retained
intentionally failing probes.

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

PROMPT-1's user/exact-cwd correction is shipped; ASR-A above reopens the same identity for
ancestor-project guidance. PROMPT-2 remains owned here.

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

- **2026-10-20 — Reviewed ASW-1 conversation event commits:** audited through `43da6bec`,
  excluding concurrent ASW-2 edits. Seven independent failure/lifecycle probes reproduce seven
  Must-fix findings; one Worth-fixing item aligns the inactive-writer spec with its accepted
  journal lock. Original sequence/order/reconciliation corrections are verified. ASW1-1–8 retain
  the same unit for fixes; passing existing suites, race and FTS5 checks do not cover those failures.
- **2026-10-20 — Implemented ASW-1 conversation event commits:** one ref-counted journal per
  conversation serializes allocation, append/sync, index and ordered history publication for every
  writer (live events, annotations, notices, switch markers, fork prefixes); control observations
  carry the commit outcome through a bounded ordered queue; history health is persisted outside the
  log (migration 57), repaired with a durable gap notice and gates new work; the reconcile sweep
  reads only the canonical log. Fixes ASRD-1/2/3/5 pending review; FS-03.R99/A78, TS-01.R39,
  TS-02.R52, TS-03.R73 and TS-04.R100 shipped with recorded deviations.
- **2026-10-19 — Reviewed process and terminal-driver ownership (ASW-3):** audited
  `a3bd6015..3618df3e` in `/tmp/chuck-asw3-review-20261019`, independently of ongoing ASW-1
  work. Five Must-fix findings keep this unit open: failed-proof Stop, false unknown-proof exit,
  early SessionStart loss, leaderless descendants and stale xterm signaling. Seven original
  findings closed (ASRA-4, ASRB-4–7, ASRC-4/7); ASRB-1/ASRC-3 are partial and remain assigned to
  ASW-3. Fix difficulty is driven by safe leaderless containment. No product code/spec edits.
  Both Go variants pass for runtime, terminal and state; focused registry/process/bootstrap/
  external-driver/state `-race` checks pass. Full untagged server/CLI and focused FTS5
  setup/launch/identity/orphan/control regressions pass with approved loopback execution after
  the sandbox denied local listeners. `make check-specs` and diff whitespace checks pass.
  Actual-code probes in
  `/tmp/chuck-asw3-probe/internal/runtime/{asw3_probe_test.go,terminal/asw3_probe_test.go,
  terminal/early_session_probe_test.go,terminal/unverified_probe_test.go}` independently confirm
  the five failures; the leaderless regression intentionally fails on the reviewed code.
  Probe helper `/tmp/asw3-sig-helper` catches SIGINT in its own test process group; cleanup signals
  target only test-created groups. Canonical probe invocation: `env -u CHUCK_RUNTIME_ROOT
  GOCACHE=/tmp/chuck-asw3-gocache GOFLAGS=-p=1 go test ./internal/runtime/... -run
  'TestASW3|TestProbe' -count=1 -v` from the probe directory. Real tmux/iTerm2/provider receipts
  remain FS-07.A12; no rendered UI gate applies to this backend-only diff.

- **2026-10-19 — Implemented ASW-3 process and terminal-driver ownership:** running rows record
  the leader's kernel birth (migration 56) and every orphan signal, peer Stop, startup reconcile
  and sweep share one proof check (`runtime.ClassifyProcess`/`StopRecordedProcess`); unproved live
  PIDs refuse with a 409 and confirmed death writes FS-01.R20's outcome. Setup commands are
  contained (`runtime.ContainCommand`, TS-12.R11). Registry launch/resume fence exit-before-commit;
  manual launch and Clone hold the lifecycle claim. Terminal SessionStart makes the native ID
  durable and empty-ID Resume refuses. tmux/iTerm2 exec a hidden bootstrap with a private launch
  record and nonce-bound handshake, unique session handles, 1 s exit polling and proof-checked
  Cancel/Stop. Fixes ASRA-4, ASRB-1, ASRB-4–7, ASRC-3/4/7 pending review. Closure: `make test`
  (both variants), `make build`, `-race` on runtime registry/proof and terminal packages,
  `make check-specs`; each regression was confirmed failing on the pre-fix code. No real
  tmux/iTerm2 or credentialed CLI run (neither driver installed): FS-07.A12 owed.

- **2026-10-19 — Designed agent-surface remediation:** formalized three bounded structural
  refactors as ready changes with planned FS/TS requirements, forward migration, security,
  native-proof and acceptance obligations. Consolidated the other 19 findings into six explicitly
  authorized cross-review fix batches; all 41 open IDs have exactly one primary owner in
  [AGENT-SURFACE-FIX-UNITS.md](AGENT-SURFACE-FIX-UNITS.md). Preserved original review evidence,
  native/device gates and closure identities. No product code, provider turn or fix closure.
  Spec/launcher lint, finding assignment, document links/anchors and diff checks pass.

- **2026-10-19 — Assessed systemic fix strategy:** compared all completed ASR findings and
  inspected the linked execution, process-control, event-sink and recovery seams. Identified three
  structural refactor candidates and three coordinated fix areas, prioritized process-control,
  replay and liveness risks, and strengthened ASRD-3's control-consumer impact without a duplicate
  finding. Rejected an invalid
  same-stage wait composition; retained the valid broader workflow as unprobed. Recommendations
  are in [AGENT-SURFACE-FIX-STRATEGY.md](AGENT-SURFACE-FIX-STRATEGY.md). No product/spec edits,
  provider turns or new end-to-end failure reproduction; existing review evidence remains explicit.
  Spec lint, report/handoff link/anchor/source/finding references and diff checks pass.

- **2026-10-19 — Reviewed whole-system synthesis and agent performance (ASR-I).** Reconciled
  all 25 topics and J1–J7 against unchanged source, resolved conflicting ordering/receipt/isolation
  claims, and retained existing finding identities. No new independent defect or product/spec
  change. Focused runtime/server/CLI race checks, 113 UI tests and presentation/spec/diff checks
  pass. Recorded a bounded paired experiment; native adoption, physical-device and comparative
  performance gates stay open. Removed the completed synthesis row; A–H remain open on findings.

- **2026-10-19 — Reviewed human actions across desktop, phone, CLI and HTTP (ASR-H):** audited
  `d49a8f86` through shared runtime/context/recovery effects. Four Must-fix findings cover erased
  newer Steer drafts, stale Withdraw restoring the wrong message, uncertain/truncated CLI resume
  selection and automatic replay after an unconfirmed consumed Steer. Focused UI, Go/race/FTS5
  and isolated CLI journeys pass; temporary probes reproduce the gaps. Native/device/rendered
  gates remain owed. Product code/specs unchanged.

- **2026-10-19 — Reviewed failure, quota and continuation recovery (ASR-G):** audited current
  implementation at `fa092f63` across provider failures, isolated checks, owner admission,
  interrupted task/stage/room recovery and manual controls. Four Must-fix findings cover phantom
  room attempts, premature check charging, cancelled-turn success and retained Codex check
  instructions; one Worth-fixing finding covers cleanup starvation. Five temporary overlay probes
  reproduce the gaps; the cleanup control, focused quota/recovery tests, race checks, SQLite FTS
  variant and specification checks pass. Live provider/device evidence remains owed. Product
  code/specs unchanged.

- **2026-10-19 — Reviewed pipelines, Think Tanks and orchestration (ASR-F):** audited current
  implementation at `b8157f13`, including task/room authority, Stop/quota/restart and private-context
  joins. One Must-fix finding covers lost post-result transition recovery; one Worth-fixing finding
  covers obsolete stage-executor code and misleading acceptance tests. A temporary fault probe
  reproduces the stranded run in both Go variants and its no-fault control passes. Focused Go,
  race, FTS5 and spec checks pass; the known Stop 409 did not reproduce in 25 repeats. Existing
  provider/device/restart acceptance gates remain owed. Product code/specs unchanged.

- **2026-10-19 — Reviewed agent tools, communication and durable delegation (ASR-E):** audited
  current implementation at `5eb574d9`, including host/provider approval, durable mail and task
  lifecycle/context joins. Two Must-fix findings cover missing watched-result continuation context
  and unbounded task-result outputs. Serialized MCP probes reproduce both; focused integration,
  race, FTS5 and spec checks pass. Live provider/task/device receipts remain owed. Product code/specs
  unchanged.

- **2026-10-19 — Reviewed conversation, input and history fidelity (ASR-D):** audited current
  implementation at `c12c4251`, including delegated context and intake/native/durable/readback
  joins. Five Must-fix findings cover duplicate/out-of-order transcript sequences, live output
  after failed persistence, annotation replay delivery and reconciliation artifact/size bounds.
  Isolated overlay probes reproduce the defects; focused Go, race, FTS5, MCP and UI checks pass
  as scoped above. Native provider/device acceptance remains owed. Product code/specs unchanged.

- **2026-10-19 — Reviewed Codex and Claude native integration fidelity (ASR-B):** audited
  `6b5de19b` against pinned adapter/native contracts. Eight Must-fix findings cover terminal
  conversation retention, Claude/Codex fork realization, optional-driver lifetime/environment/
  identity, and unpatched source Codex steering. Independent source traces, temporary defect
  probes and focused resolver/runtime/server/auth/release tests support the findings; live
  provider/driver acceptance remains owed. Product code/specs unchanged.

- **2026-10-19 — Reviewed runtime execution, concurrency and control (ASR-C):** audited the
  current runtime at `6b5de19b`. Confirmed the SetSessionConfig read-loop stall (ASRC-1); three
  Must-fix and four Worth-fixing findings remain on this unit. Fake-ACP overlay probes and focused
  `-race` runtime tests pass as scoped above; no live provider run. Product code/specs unchanged.

- **2026-10-19 — Reviewed instructions, configuration and execution environment (ASR-A):**
  audited current implementation at `4fbeb316`, including native delivery and checkout readiness.
  Six Must-fix findings (including reopened PROMPT-1) and one Worth-fixing permission/spec mismatch
  remain on this unit; eight sibling audits remain waiting.
  Deterministic fixture/Go-overlay probes, pinned native stdio configuration reads, focused server
  lifecycle tests with approved loopback execution, and relevant package tests pass as scoped above;
  live model adoption and performance comparison are unclaimed. Product code/specs unchanged.

- **2026-10-19 — Consolidated agent-surface reviews:** replaced 25 boundary-level queue items
  with nine integrated reviews for frontier agents. Preserved every original search topic and
  added explicit cross-system joins, connected failure scenarios, coverage accounting and a final
  whole-system/performance synthesis. Original numeric IDs resolve to their containing review.
  Planning only; no substantive review, product/spec change or provider run.
  Spec checks, source/spec/link/anchor checks and queue consistency pass; all 25 original topic
  bodies are preserved unchanged as research, each mapped once, with seven required system joins.

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
