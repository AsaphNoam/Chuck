# Chuck — Implementation handoff

Settled history through v0.20.0 is archived in
[HANDOFF-through-2026-10-25](../archive/state/HANDOFF-through-2026-10-25.md) and earlier
[HANDOFF-through-2026-10-24](../archive/state/HANDOFF-through-2026-10-24.md) and
[HANDOFF-through-2026-10-23](../archive/state/HANDOFF-through-2026-10-23.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archived snapshots are not live queues.

## Current position

- **Published:** [v0.20.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.20.0),
  2026-10-25, tag at `a774d128` (saved file-viewer tabs, project-scoped groups/selectors,
  skip-permissions default, annotation focus). Release CI 40232088716 and main CI 40232088269
  passed; downloaded archive size and SHA256 match the manifest.
  [Notes](../archive/state/RELEASE-v0.20.0-notes.md).
- **Review state:** quota recognition unit QUOTA-RPC is closed (QUOTA-RPC-1/2/3 fixed
  2026-10-30); its live receipts stay owed below. Optional annotation source recording unit ANN-SOURCE is closed
  (ANN-SOURCE-1/2 fixed 2026-10-30). Pipeline worktrees fix unit
  PIPE-WT is closed (PIPE-WT-1–6 fixed
  2026-10-30); its packaged Claude/Codex launch/resume cwd probe stays owed below. Review unit
  MMD-VIEW (file-viewer Mermaid sizing) is available; memory fix unit MEM has one open **Worth fixing**
  finding (MEM-4), blocked on a decision below; summary-only tool activity unit TOOL-LINE is
  closed (TOOL-LINE-1/2 fixed 2026-10-29); MEM-1/2/3/5/6 are fixed. Delegation-guidance fix unit
  DELMSG is closed (DELMSG-1 fixed). The v0.20.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.19.0-next.json)) is closed; ARCH-v020-1 fixed
  and verified. Provider-failure fix unit PERR is closed (PERR-1–4 fixed 2026-10-26); the
  frequent-429 field cause remains unmeasured (needs correlated request history if it recurs).
- **Goal-led pipelines:** reviewed 2026-10-30 (GOAL-LED); fix run in progress — GOAL-LED-1–10, 12 and 14
  fixed; GOAL-LED-11/13 (queued-input and decision/evidence display) stay on the same unit below. Acceptance
  evidence remains owed below.
- **Waiting-prompt investigation:** WAIT-PROMPT-1/2 are fixed by goal-led slice 1 (input to a
  waiting assignee is queued and carried by its continuation); WAIT-PROMPT-3 is fixed by coalescing
  phone streaming refresh signals into 250 ms windows. The investigation unit is closed.
- **Permission-save investigation:** PERM-SAVE-1 is repaired; the reported unchecked toggle's
  field cause remains undetermined pending affected server evidence.
- **Mobile annotation investigation:** MOB-ANN-1 is fixed with optional source recording: both
  remote annotation allowlists admit the bundled client's `command_id` and `omit_source_record`.
  Authenticated gateway delivery/preflight and same-command replay pass.
- **Browse latency:** BROWSE-LAT-1 repaired with fixed JXA script invocation; native selection,
  foreground placement and click-to-panel timing remain owed below.
- **Claude effort fix:** CEF-1 is closed: the shared starter now declares `xhigh`.
  Existing catalogs and add-only sync retain their effort metadata; live model support is unverified.
- **Compact Think Tank navigation:** TTNAV-1/2/3 are closed by fix commit `1cf8fc9d`; the
  originating unit is complete. DELMSG-1 is closed; FS-18.A12's pinned-provider guidance-adherence
  scenarios remain owed.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`, pushed through the v0.20.0 release record.
  Clean release verification uses a detached worktree at `/private/tmp/chuck-release-<version>`.
- **Future work:** direct-action migration remains paused on its transport blocker; passive
  quota snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Native/device/credentialed limitations remain in the owed gates.
- **Awaiting review — phone Home, Sky & Grove wallpaper, composer Cancel (2026-10-29):**
  FS-20.R43. Home drops its tagline (hidden "Home" h1 kept), project rows are separate cards with
  a 12px gap, Sky & Grove's ring layer is off on the phone via `--ad-phone-wallpaper-rings`, and
  Cancel (aria "Cancel turn") sits beside Attach only while busy, replacing `.phone-controls`.
  129 phone tests (one new), 1069 UI tests, presentation checks and embed pass. New
  `ui/scripts/phone-skins-render.mjs [--busy]` checks all appearances at 360/390/430px (failed on
  the old tree); screenshots `/tmp/skins-{after,busy}-*.png`. `phone-render.mjs --details-check`
  was stale since `fe572fcc` (border moved to `::before`; groups start collapsed); fixed and
  passing in all appearances. INV §13/§17.

## Active change

**Running background-work indicator** —
[`running-background-work-indicator`](../ready-changes/running-background-work-indicator.md),
**resumable**. Operator decision 2026-10-29: option (a). Each popover row is named by its
related tool call's adapter title and shows state and elapsed time. The jump only locates where
the command started; no output is shown or copied (TS-04.R64). Real transcripts show no output
reaches the tool call: Codex results are `content: null` and Claude returns only a
"running in background, output in <tmp file>" notice. Tool detail is now gone from transcripts
(TOOL-LINE, closed); popover rows take their name from `toolLine(call).label`
(`ui/src/components/chat/toolLine.ts`).

- **Done (slice 1):** the runtime publishes each generation's open task count into
  `status.background_running` (migration 63; `agentState.applyLive` on every status write,
  `republishLive` after each task spawn/terminal update), `AgentState.background_running`
  projects only with a running row, and the desktop `AgentCard` shows **N running**
  (TS-04.R104, TS-02.R59, FS-02.R83/A64 now shipped). Tests:
  `TestBackgroundRunningCountIsPublished`, `TestBackgroundRunningProjectsOnlyWhileRunning`,
  `AgentCard.test.tsx`. Checks on this tree: `go test ./...`, `-tags sqlite_fts5`, focused
  `-race` task tests, `ui` `npm test` (1060) and `npm run build`, `make build` passed.
  The earlier DELMSG-1 missing-INV-tag lint failure is repaired by CEF closure bookkeeping.
- **Next (after the decision):** FS-03.R107/TS-08.R146 — remove `BackgroundTaskList`, add the
  composer `RunningTasks` indicator/popover from `collectTasks`, phone render check, FS-03.A83's
  fake-ACP scenario, `make embed`, the 1024/1440px render check.

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

- **MEM-4 live transcript bound:** the only full-history reader of `agentState.transcript` is
  `Runtime.Transcript`/`Registry.Transcript`, which has no product caller (12 test call sites in
  `internal/runtime` and `internal/server`); notice dedupe needs notices only. Runtime cannot read
  the durable log (`internal/transcript` imports `runtime`). Recommended: retain notices only,
  drop `Transcript` from the `Runtime` interface, and move tests to subscription recorders or
  durable reads (medium; ~12 test rewrites). Alternative: keep full history only for
  non-durable generations (smaller, but leaves a test-only mode in product code).

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

- **PIPELINE-FIGMA** (2026-10-29, UI): Figma Make pipelines overhaul, FS-12.R71/A41,
  TS-08.R149. Shared header/tabs, light ruled run ledger with separate state/outcome,
  numbered template library, light overview/timeline, stage rail/editor and start-dialog
  finish. Existing real actions/routes remain; prototype-only controls are excluded.
  Pipeline focused tests pass 42/42; final integrated UI suite passes 1068 with 1 skipped. Styles,
  presentation contracts, type/build and `make embed` pass; `git diff --check` passes.
  `make check-specs` retains only the pre-existing DELMSG-1 missing invariant-tag failure.
  Independent browser evidence: `ui/scripts/pipelines-render.mjs`, all appearances at
  1024/1440, populated runs/templates/detail/editor and Setup→Review dialog, pagination,
  event refresh and internal stage-rail overflow (120 checks); empty/loading/error each pass
  24 checks, 192 total. Independent rendered critique has no remaining material findings.
  Screenshots in shared
  resources `pipelines-final/`. Figma MCP supplies source but does not support Make screenshots;
  its workspace source is truncated, so exact target screenshot comparison is unverified.
  Pre-existing generated index saved as shared `pipelines-preexisting-index.html` before
  regenerating the embedded output from integrated source.

- **GROUP-SKILL** (2026-10-29, UI): group-background New agent seeds the editable Group
  field without leaking into later openings; `@` offers advertised commands alongside files
  and room participants, while `/` and the existing `#` open commands. FS-02.R77/A58,
  FS-03.R33/A17. Group regression failed before the fix and all 87 related focused tests
  pass; composer (51) and Think Tank page (32) pass. Full UI suite: 1068 passed, 1 skipped;
  UI styles, presentation contracts, type check and production build pass. The worktree's
  dependency symlink caused TS2742; local dependency copies resolve it. Both edited feature
  specs pass scoped checks; full spec check retains the pre-existing DELMSG-1 missing INV tag
  failure. Merged into main; combined-checkout production build and `make -o ui embed` pass.
  Generated index remains with the pre-existing dirty embedded UI owned by the concurrent
  pipelines work; no pipelines source or handoff edits were committed with this unit.

- **BGRUN-COUNT** (2026-10-27, small, Go + UI): live running background-task count —
  migration 63, `internal/runtime/{chat,tasks}.go` status writes, `internal/state` projection,
  `AgentCard` badge.

- **MMD-VIEW** (2026-10-26, small, UI CSS only): file-viewer Mermaid diagrams fill the reader width
  and center instead of collapsing to 300×150 — `ui/src/styles/features/agent.css`.

## Review findings

### GOAL-LED — Goal-led unattended pipelines (2026-10-30) — **Fix model:** difficult — Codex Sol.

Reviewed implementation `715bbcb2^..af2c9495`, including surrounding callers, FS-14.R97–R103,
TS-09.R66–R72, TS-10.R29/R35, changed persistence/API/chat requirements and test contracts.
The same unit stays open for grouped fixes; product code and specifications were not changed.

- **Must fix** — GOAL-LED-11 (P2): accepted queued input disappears from supervision until delivery.
  `ui/src/components/chat/Composer.tsx:155–186` clears the draft without a held/transcript row;
  only a transient notice remains. `internal/server/sessions.go:46–81` exposes runtime-held or
  recovery input, and `internal/state/attachment_receipts.go:49` excludes prepared assignment
  receipts. After navigation/restart, pending text and attachments cannot be inspected. Rehydrate
  the durable queued state on desktop/phone without falsely adding delivered transcript events.
  No data loss is established by this finding. TS-09.R69/R71; INV §8/§10. Fix complexity: medium.

- **Must fix** — GOAL-LED-13 (P2): run supervision hides accepted decision/evidence provenance.
  `ui/src/remote/RunScreen.tsx:84–95` omits task.decision entirely, hiding deviations and reasons.
  `ui/src/features/pipelines/RunBrowser.tsx:187` shows action/reason but not required_checks;
  neither view identifies the successful visit selected as completion evidence. Existing outputs
  do not show that mapping when older successful evidence is selected after revisits. Independently
  rendered completion fixtures confirmed both omissions. Expose the chosen decision and evidence
  visits on desktop/phone with inspectable result links. FS-14.R103; TS-09.R71; INV §10/§11.
  Fix complexity: trivial/easy.

Review evidence: eight external regression probes exposed room false completion, controller
exhaustion, missing selected input, unselected-room rejection, replacement attachment loss,
startup exhaustion, retry input release and conversation-answer non-continuation. Temporary Go
overlay and probe sources are saved in shared project resources (`goal-led-review-all-overlay.json`);
the parent reran all eight together and confirmed the expected failures in
`goal-led-review-regressions.log`. `make check-specs` and `git diff --check` pass.
temporary originals remain at `/tmp/goal-led-review-overlay.json` and
`/private/tmp/goal_led_review_overlay.json`. Existing focused decision/visit tests pass; fresh full
pipeline/state/server/messaging/runtime package tests pass (server/messaging required approved
loopback execution; the initial sandbox bind failure was environmental). Delegated UI checks:
68/68 focused tests; the existing rendered harness
passes 120 checks in all appearances at 1024/1440. Additional interrupted/completed desktop 1024
and phone 390 fixtures in Core/Sky & Grove reproduce GOAL-LED-12/13 with no overflow; parent
inspected the phone refusal screenshot. Evidence `/private/tmp/goal-led-surface-probe.mjs`,
`/tmp/goal-led-surfaces-render/`, `/tmp/goal-led-{phone,desktop}-*-{interrupted,completed}.png`.
No live-provider, physical-phone or full run-start-to-notification acceptance is claimed; existing
A61/A64/A67/A68 gates remain open. Existing tests use useful real store/controller/HTTP boundaries
but their happy paths do not establish these missing contracts (TS-06.R35; INV §17).

Invariant sweep: §§1/2/3/4/5/7/8/9/10/11/13/14/15/16/17 have applicable lifecycle, persistence,
receipt, protocol, UI, handler or test surfaces and were assessed; findings are tagged above.
§6 has no new runtime/interface, and §12 has no new external CLI invocation/flag surface.
No independent violation found in applicable §§3/4/5/7/13/14. The interrupted-input refusal and
room deviation are implementation choices requiring correction above, not accepted spec exceptions.

### MEM — Memory retention investigation (2026-10-26) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report, verbatim:** “I have a suspicion that Chuck is leaking memory, do a thorough
investigation and see if you can find anything”. No workload, measured growth, affected process,
version, or diagnostic log was supplied. Investigated checkout `43c214d1` (v0.20.0 plus release
records), macOS arm64. This establishes code defects, not which defect caused the reported suspicion.
The optional question about the affected process/workload had no answer at closure.

- **Worth fixing** — MEM-4, confirmed: live chat keeps an unbounded duplicate of all events.
  `internal/runtime/chat.go:295,2318–2322` retains every committed event in `agentState.transcript`;
  completed turns never trim it. This is growth for the lifetime of a running generation, distinct
  from post-Stop leakage. `Transcript` at `:1469–1479` additionally copies the event slice on reads.
  A disposable runtime exercise of 20 completed turns × 100 × 8 KiB deltas retained 2,020 events
  and 16,408,840 payload bytes. These are payload counts, not heap/RSS measurements. Browser reads
  already use durable transcripts; notice deduplication also scans this slice, so simply deleting
  it would break another consumer. INV §16; TS-01.R5/TS-02 transcript authority. Bound the runtime
  projection and preserve transcript-read and notice-idempotence contracts through the existing
  durable authority. Fix complexity: medium. Current retention seam is `69ef5951`, but full-history
  accumulation existed earlier; no exact regression release is asserted. Open on the decision
  under **Decisions needing input**.

Field attribution of the user's suspected growth remains undetermined (no heap/browser snapshot
or workload was supplied); disposable probes remain under `/private/tmp/chuck-memory-*`.

## Changelog

- **2026-10-30 — Fixed GOAL-LED-2/3/7/12/14 (INV §1/§2/§8/§10/§11/§15):** queued input stays bound
  to its assignment through start retries and moves whole (text and attachments) to an automatic
  replacement, its attachments cloned for the new owner at delivery; the clipped text handoff is
  gone. Run-page Continue and a message to a paused run's owner share one continuation: answers and
  resume instructions continue the same run once (never an approval), and interrupted-visit Resume
  now accepts instructions on desktop and phone. Operating guidance describes automatic recovery
  and the decision-only room handoff. `GET .../prompt` now returns retained `queued` input (UI
  display pending, GOAL-LED-11). Regressions: `TestQueuedInputSurvivesStartRetry`,
  `TestQueuedAttachmentFollowsReplacementOwner`, `TestConversationAnswerContinuesQuestionPause`,
  extended `TestInterruptedOwnerRecoversAutomaticallyWithinTheAllowance`.
- **2026-10-30 — Fixed GOAL-LED-6/8/9 (INV §2/§9/§10):** restart start exhaustion now reaches
  the common parked-owner recovery; a stall check deferred by background work is rechecked when
  that work ends (`EvBackgroundTaskState`) and by the dispatcher's bounded sweep, which also heals
  parked owners; after ten controller failures a release pauses `controller_recovery_failed`
  holding its intent, automatic retry stops, and Continue retries once. Regressions:
  `TestStallCheckDeferredByBackgroundWorkResumesWhenItEnds`, `TestParkedOwnerAfterRestartIsRecovered`,
  `TestPersistentControllerFailurePausesAndResumes`; state/pipeline packages pass.
- **2026-10-30 — Fixed GOAL-LED-1/4/5/10 (INV §8/§10/§11/§15/§17):** an accepted room synthesis
  now goes, after cleanup and approval, to one decision-only owner visit (migration 68
  `decision_source_task_id`) that cannot be required-check evidence; acceptance validates only the
  selected destination's required inputs and room context; each activation freezes `input_refs`
  (value key and producing task/room), projected in run detail. TS-09 §5's room deviation is
  removed. Regressions: `TestFinalRoomSynthesisWaitsForTheOwnersDecision`,
  `TestRoomOutputOverflowRefusesOnlyTheSelectedRoom`, `TestDecisionValidatesOnlyTheSelectedDestination`,
  extended mixed-room/approval tests. `go test` state/pipeline/messaging and server
  pipeline/room filters pass (`/tmp/gl-s1*.log`).
- **2026-10-30 — Fixed WAIT-PROMPT-3 (INV §1/§4/§16/§17):** every streamed `new_message`
  previously changed the phone's transcript and held-prompt query keys. The connection owner now
  coalesces known-agent signals with one shared fixed 250 ms timer, including the final update,
  and clears pending work on stream teardown. TS-13.R29 records the bound. Fake-clock regressions
  failed before repair (20 refreshes in 500 ms); the HTTP/component probe also failed before repair
  (21 transcript GETs for 20 messages), and now proves bounded transcript/prompt reads and final
  fetched text rendering. `make check-specs`, UI `npm test` (1122), `make embed` (presentation checks
  and production build), and `git diff --check` pass. Logs `/tmp/wait-prompt-{before,http-before,ui,embed}.log`.
  This closes WAIT-PROMPT; GOAL-LED and the MEM-4 decision remain open. No physical-phone
  performance or battery measurement is claimed; existing native/device gates remain owed.

- **2026-10-30 — Reviewed goal-led pipelines (GOAL-LED):** thirteen Must-fix findings and one
  Worth-fixing finding keep the same unit open. Independent probes show false room completion,
  incomplete routing/input/recovery contracts, and hidden supervision evidence; rendered phone
  checks reproduce the rejected Resume instructions. Findings, fix sizing and verification limits
  are above; product code and specifications remain unchanged.

- **2026-10-30 — Implemented goal-led pipelines (GOAL-LED):** queued waiting-assignment input,
  chronological visits, owner decisions, required checks, per-visit approval, bounded automatic
  recovery/replacement and desktop/phone/CLI question/recovery surfaces. `make test` passed both
  Go suites (`/tmp/glp-closure-test.log`); `make dist` passed, including styles, presentation,
  TypeScript, UI build, regenerated embed and Go binary (`/tmp/glp-dist-final.log`). The build
  emitted a nonfatal sandbox module-cache warning. Final UI suite: 104 files, 1119 tests pass
  (`/tmp/glp-ui-closure-final.log`); `make check-specs` and `git diff --check` pass.
  Room-decision deviation and remaining A61/A64/A67/A68 evidence stay explicit in FS-14 §6
  and the acceptance gates above. Retired the completed ready change and temporary plan.
- **2026-10-30 — Fixed QUOTA-RPC-1/2/3 (INV §11/§12, §17):** every date/zone qualifier around
  a reset clock is consumed and must agree; conflicting or unconsumed qualifiers stay unknown
  (QUOTA-RPC-1). Terminal RPC errors are stamped on transport decode and reset text resolves
  against that arrival, not settlement (QUOTA-RPC-2). Slash dates count only beside a clock, so
  `3/5 used` no longer vetoes a reset (QUOTA-RPC-3). TS-04.R107 states these rules. New cases in
  `TestTextResetExtractsWithoutFixedSentence` and
  `TestTerminalRPCResetResolvesAtArrivalNotSettlement` fail on the old code. Closure: `make test`
  (both variants), `make build`, focused runtime `-race`, `git diff --check`. Unit closed;
  FS-01.A37/A39 live receipts remain owed.

- **2026-10-30 — Reviewed QUOTA-RPC:** two Must-fix findings cover dropped/conflicting reset
  qualifiers and settlement-time reinterpretation; one Worth-fixing finding covers unrelated
  usage fractions suppressing valid resets. Findings and independent evidence are above;
  product code/specifications unchanged, same unit remains open for fix.

- **2026-10-30 — Implemented quota recognition and session-reset recovery (QUOTA-RPC):** a
  provider terminal quota RPC without AIR metadata now enters recovery; reset times are
  extracted regardless of surrounding wording. The `a_a7e830` spend/session-reset error now
  schedules continuation at 22:00 Asia/Jerusalem plus one minute. No migration, field or UI
  change. Checks recorded under Review queue; live receipts owed.

- **2026-10-30 — Defined goal-led unattended pipelines:**
  `goal-led-unattended-pipelines` was waiting
  to start; FS-14.R97–R103/A61–A68 and TS-09.R66–R72 remain planned. Confirmed: useful existing-owner
  judgment with the decision in its existing result; programmatic requirements, advancement and
  mechanical recovery; ordinary-plan steps with explicit required checks; durable mid-run input;
  automatic visibly fresh same-runtime owner replacement; three repair/revisit rounds followed by
  a recoverable pause, never failure merely because allowance was exhausted. Resume/instructions
  can continue the same run with preserved work. New runs of existing version-2 templates receive
  the behavior directly: optional marker/defaults plus additive visit backfill, without a legacy
  engine, adoption flow or destructive reset. The idea is promoted; implementation and the unrelated
  active change are untouched. WAIT-PROMPT-1/2 remain open until repaired; WAIT-PROMPT-3 is excluded.
  Verification: `make check-specs`, twin-skill comparison and `git diff --check` pass. Behavioral
  tests/provider/browser evidence remain planned; no product tests were run for this design-only unit.

- **2026-10-30 — Closed ANN-SOURCE:** ANN-SOURCE-1 (INV §1/§11/§15), repaired in `a9bb424a`,
  retains exact launch payload/command/choice through retry/reload and clears only acknowledged
  drafts. Extended deferred-preflight regression fails before repair in both reload modes and now
  proves identical retry, one modeled receipt effect and the next draft preserved/default checked.
  ANN-SOURCE-2 (INV §8/§10): styled feedback stays with the workspace outside the tray side slot,
  including Focus file. Final embedded 1024px browser checks restore card width to 832px and side
  width to zero for normal/long recipients, with visible wrapping feedback; screenshot independently
  inspected. Final source checks: `make check-specs`, full UI suite (1,117), `make embed` (includes
  UI build and 47 presentation script checks), `git diff --check` pass. Evidence: shared resources
  `ann-source-fix/` logs and `ann-source-fix-layout.{json,mjs}` / `ann-source-fix-layout-{1,2}.png`.
  Browser responses are stubbed; unchanged real-handler persistence/rollback/restart and phone/room
  evidence remains in `optional-annotation-verification/`, `optional-annotation-rendered/` and
  `ann-source-review/`. Native-device/provider gates remain owed as before. No ANN-SOURCE finding
  remains; other units stay open. Added missing invariant metadata to pre-existing WAIT-PROMPT-1
  so global spec lint passes; its finding and product behavior remain open/unchanged.

- **2026-10-30 — Reviewed optional annotation source recording:** ANN-SOURCE stays open for
  a launch-snapshot retry duplication bug (INV §1/§11/§15) and success feedback retaining desktop
  side-panel width (INV §8/§10). Independent focused tests and desktop/phone/room journeys pass;
  controlled retry and rendered geometry probes establish the two missing cases.

- **2026-10-30 — Optional annotation source recording finished:** peer/new-agent assignments
  can omit the source transcript record while preserving delivery and retry identity. Desktop
  and phone share the persisted choice and launch snapshot; room controls gain the **New agent**
  rename only. MOB-ANN-1's rejected phone command identity is repaired (INV §10/§11/§14).
  Complete verification and evidence are recorded under review unit ANN-SOURCE.

- **2026-10-30 — Fixed pipeline worktree review findings (PIPE-WT, unit closed):**
  PIPE-WT-1/2 (INV §5/§9/§16): removal matches every running agent by canonical checkout path
  (`RunningCheckouts`, no 256-session page), and each claimed start records its checkout in
  `resolveStartCheckout` until its lifecycle claim ends; `beginCheckoutRemoval` refuses new
  starts during removal. PIPE-WT-3 (INV §2/§12): `verifyRunCheckout` now includes exact Git
  registration and runs after recreation too. PIPE-WT-4 (INV §2/§10): launched standing owners and
  replacements take `placeRunTaskTx`'s frozen group. PIPE-WT-5 (INV §1/§10): pipeline updates and
  reconnect invalidate workspace list/detail caches. PIPE-WT-6 (INV §10/§11): session responses and
  AgentState project `workspace` from the retained record; workspace changes republish bound
  agents. TS-12.R15 clarifies path matching. Tests: `TestPipelineRunWorkspaceRemovalSeesEveryUser`,
  `...AgentMetadata`, registration case in `...RecoveryAndTombstone`, replacement/room-first
  group assertions, SSE workspace invalidation. Clone metadata relies on the shared session
  `workspace_id` projection rather than a separate clone fixture. Checks on this tree: `make test`
  (both variants), focused `-race` `TestPipelineRunWorkspace*`, `ui` `npm test` (1108) and
  `npm run build`, `make build`, `git diff --check` passed.
- **2026-10-30 — Simplified quota recovery design:** operator approved direct continuation at
  the extracted provider recovery time plus one minute for Claude and Codex. FS-01.R57/A39,
  TS-04.R107 and TS-10.R62 replace the unshipped preliminary-check/session-hint design; terminal
  RPC recognition remains FS-01.R55/TS-04.R105. Retired FS-01.R56/A38, TS-02.R61, TS-03.R76,
  TS-04.R106, TS-08.R150 and TS-10.R61 without reusing IDs. The
  ready change `repair-quota-recognition-and-session-reset-recovery` (implemented the same day)
  was **Waiting to start**, using existing recovery fields, scheduler, projection and UI;
  no new migration, preliminary check or product code. Extraction tolerates wording changes
  while retaining date/timezone ambiguity checks and existing bounded retry/control rules.

- **2026-10-30 — Reviewed pipeline worktrees (PIPE-WT):** six Must-fix findings remain in the
  same unit (INV §1/§2/§5/§9/§10/§11/§12/§16): cross-binding and truncated removal checks,
  missing Git registration validation, fresh-owner group loss, stale workspace caches and partial
  agent metadata. Independent focused suites and Core/Sky & Grove desktop/phone browser checks
  pass; distinguishing external probes fail as recorded above. No product or specification edits.

- **2026-10-29 — Designed quota recognition and session-reset recovery:** operator confirmed
  FS-01.R55–R56/A37–A38; prepared
  `repair-quota-recognition-and-session-reset-recovery`, **Waiting to start**. TS-04.R105–R106 normalize terminal RPC quota evidence and a separately
  scoped session-reset hint; TS-02.R61/TS-03.R76/TS-08.R150/TS-10.R61 define persistence,
  projection and one isolated check before same-work continuation. Removed the promoted idea.
  No product code changed; no existing interruption was rearmed. Provider source receipts are
  in the change; live provider admission remains an explicit acceptance gate.

- **2026-10-29 — Closed TOOL-LINE (TOOL-LINE-1 INV §10/§17, TOOL-LINE-2 INV §10):** each
  transcript view (desktop, phone, Think Tank activity) now builds one `resultsByCall` map over
  its whole nested list, child activity included, and passes it to every `groupTranscriptRows`
  call, so a call split from its result by a visible row in a completed turn still reads
  **failed: …**. Desktop and phone regressions (`folds a result separated from its call by a
  visible row`) failed on the old code and pass. TS-08.R147 states the wiring; FS-03.R26 now
  says the run reveals call lines with folded results and permission chips, without the stale
  result-row and supersedes-R25 text. `make check-specs` and `npm run build` pass; `npm test`
  passes 1091/1092 — the one failure, `CardGrid` "cycles composer focus across group sections"
  (cannot find label `Composer a_1`), is an intermittent pre-existing flake that also fails 3/4
  in isolation on the pre-fix tree. The originating unit is complete.
- **2026-10-29 — Repaired PERM-SAVE-1 (INV §8/§11/§17):** the permissions editor now checks a
  successful response against the requested policy, keeps the returned server value, and shows a
  readable error when the value did not change. The previously skipped HTTP 200 unchanged-policy
  regression failed before the repair and passes with the repair. FS-04.R60/A39 now specify this
  response behavior. `make check-specs` and the focused component suite pass. Isolated UI closure
  with only the PERM-SAVE changes applied passes all 1,090 tests and `npm run build` (including
  style and presentation checks). Logs: `/private/tmp/perm-save-check-specs.log` and
  `/private/tmp/perm-save-snapshot-ui-{test,build}.log`. The full UI run on the shared checkout
  also exposed failures in concurrent tool-summary edits (`AgentScreen.tsx` references undefined
  `toolResults`; build also reports missing `results` and unused `resultsByCall`); those files were
  preserved. The affected server's field cause remains undetermined.

- **2026-10-29 — Repaired BROWSE-LAT-1 (INV §12):** directory Browse now uses a fixed JXA
  script through direct `/usr/bin/osascript` argv, avoiding AppleScript's activation stall.
  The command regression, existing picker contract tests, `make test` (standard and
  `sqlite_fts5`), `make build`, and `git diff --check` pass. Activation probe: AppleScript 2.29s,
  JXA 0.10s; real JXA cancellation returned `-128`. Native selection/foreground/panel timing
  remain owed above. No product requirement changed. Logs: `/private/tmp/chuck-browse-{focused,
  make-test,make-build}.log`. Existing tool-summary UI edits were preserved.

- **2026-10-29 — Closed DELMSG-1 (INV §10):** shared runtime context and the embedded
  `operating-chuck` guidance now favor concise communication with detail scaled to recipient needs,
  uncertainty and consequences. They explain when delegation helps and leave assignment detail to
  judgment and applicable skills. FS-18/TS-11 and FS-18.A12 now describe this contract and its simple
  and richer-context scenarios. `make check-specs`, embedded-skill publication checks, focused
  overlay tests, `make test` (both standard and `sqlite_fts5` suites), `make build`, and
  `git diff --check` pass. Logs: `/private/tmp/chuck-delmsg-{make-test,make-build,overlay-approved}.log`.
  No pinned-provider adherence evaluation ran; FS-18.A12 remains an open acceptance gate. Existing
  unrelated UI edits remain uncommitted.

- **2026-10-29 — CEF-1 repaired (INV §10):** the shared Claude starter declares
  `low, medium, high, xhigh, max` for all four aliases; `medium` and `sonnet` defaults remain.
  FS-09.R46/A19 document it. Starter/fresh-disk-seed regression passes and fails with the old
  seed via Go overlay. Saved catalogs and add-only sync remain unchanged; compatible existing
  models can declare `xhigh` in Settings → Backends. No live-provider support claim.
  Verification: `make check-specs`, both `go test ./...` and `go test -tags sqlite_fts5 ./...`,
  `make build`, focused config tests and `git diff --check` pass. Go checks used
  `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 GOCACHE=/private/tmp/chuck-cef-gocache`; full suites
  needed permitted loopback access after sandbox-restricted attempts failed. Logs:
  `/private/tmp/chuck-cef-{go-test-approved,go-fts-approved,specs-final,build,regression}.log`.
  Added DELMSG-1's missing invariant tag as lint bookkeeping; the finding was closed separately.
  CEF's originating unit is closed; no new review unit. Other fix units remain open.

- **2026-10-29 — Approved pipeline worktrees and descendant placement:** technical/lifecycle
  contract confirmed, including additive API, retained workspace management after run deletion,
  explicit-removal tombstone and desktop cleanup. Created the
  `pipeline-worktrees-and-descendant-placement` ready change (shipped 2026-10-29, review unit
  PIPE-WT) and removed its idea entry. Planned FS-14.R91–R96/A57–A60,
  FS-19.R12/A8, TS-09.R61–R65 and TS-12.R12–R17 agree; no product code or active-change update.
  `git diff --check` and design-feature twin comparison pass. `make check-specs` reports only
  the pre-existing DELMSG-1 missing invariant tag. Supersedes the pending checkpoint below.

- **2026-10-29 — Defined pipeline worktrees and recursive group placement (awaiting technical
  confirmation; INV §§2/5/8/9/11/12/14/15/16/17):** operator confirmed optional run-owned checkout
  under the original project, initial run-group placement preserving manual moves, refusal of
  existing-agent targets in another checkout, and an end-of-run removal offer. Planned
  FS-14.R91–R96/A57–A60, FS-19.R12/A8, TS-09.R61–R65 and TS-12.R12–R17 cover recursive task/room
  launch binding, frozen resume/switch cwd, ownership and consented removal. Idea remains in
  `docs/ideas.md`; no ready change or product code. Awaiting confirmation of additive API,
  retained-workspace list after run deletion, explicit-removal tombstone and desktop cleanup.
  `git diff --check` and design-feature twin comparison pass; spec lint reports only the existing
  DELMSG-1 missing invariant tag. Active work is unchanged.

- **2026-10-29 — Fixed compact Think Tank navigation (INV §§8/13/17):** closed TTNAV-1/2/3
  from review `addb1f12` for the existing `compact-think-tank-chat-navigation` unit. Fix commit
  `1cf8fc9d` constrains long unavailable names, reserves active-link width to prevent flex reflow,
  and makes project-qualified labels unique against literal names. Focused UI tests pass 52/52;
  full UI suite passes 1061 with 1 skipped; style/presentation checks and UI build pass. The
  supplied Chromium harness passes 108/108 across all three appearances, 1024/1600px desktop,
  360px narrow layout and a 320–600px wrapping sweep. `git diff --check` passes. `make check-specs`
  remains blocked only by pre-existing DELMSG-1 missing INV tag. No other TTNAV findings remain.

- **2026-10-29 — Reviewed compact Think Tank navigation:** recorded TTNAV-1/2 (**Must fix**)
  and TTNAV-3 (**Worth fixing**) against the same originating unit; prior dead-CSS finding remains
  closed. Fresh focused tests/styles pass; rendered receipts reproduce the three scoped issues.
  Fixes belong to the next stage. Only handoff state changed; other queues and the dirty generated
  UI index are preserved.

- **2026-10-29 — Reconciled interrupted compact Think Tank navigation work:** originating
  [`compact-think-tank-chat-navigation`](../ready-changes/compact-think-tank-chat-navigation.md)
  is already finished (FS-21.R57/A40, TS-08.R126). Design `11245b90`; substantive implementation
  range `467bf836^..467bf836`; existing review `d46bd2a7` and dead-CSS fix `c85f94fb` are separate
  closure, with finished status retained by `88adeecd`. The implementation checkpoint records
  UI 975/975, Go tests/build, UI build and 42 Chromium checks; review records independent 42/42
  rendered checks, and fix closure records UI 976/976, style/build/spec passes. Browser receipts
  are historical: their `/tmp` artifacts are no longer present. At `02b919bc`, focused
  `ThinkTankTab.test.tsx`/`ChatPanel.test.tsx` pass 51/51, `npm run check:styles` passes 47/47
  plus presentation contracts, and `git diff --check` passes. `make check-specs` still fails only
  on the pre-existing DELMSG-1 missing invariant tag. No product/spec change, replacement unit
  or reopened review queue; subsequent pipeline stages own review/fix. Active background-work
  implementation and the dirty generated UI index are preserved.

- **2026-10-29 — Shipped summary-only tool activity (TOOL-LINE):** one-line tool labels
  (runtime description, `Ran <program>`, short title, `Used a tool`), trimmed failure reasons,
  no argument/output bodies (FS-03.R108/A84, TS-08.R147). Awaiting review.

- **2026-10-29 — Annotation assignment design ready:** operator confirmed the default-on
  source-transcript checkbox and **New agent** terminology. FS-13.R35–R36/A25–A26,
  TS-03.R75, TS-02.R60 and TS-08.R148 define the waiting change
  [`optional-annotation-source-recording`](../ready-changes/optional-annotation-source-recording.md).
  No product code changed; existing active work remains active. `git diff --check` and twin-skill
  comparison pass; `make check-specs` remains blocked by the pre-existing DELMSG-1 line missing
  an invariant tag.

- **2026-10-27 — Running background-work count (FS-02.R83, TS-04.R104, TS-02.R59):** desktop
  agent cards show **N running** from a live status column the runtime republishes on each task
  lifecycle update; stopped/archived agents project zero. The composer popover (FS-03.R107) is
  paused on the no-output decision.

- **2026-10-26 — Designed running background-work indicator:** ready change
  [`running-background-work-indicator`](../ready-changes/running-background-work-indicator.md)
  replaces the transcript Background tasks list with a desktop composer **N running**
  indicator/popover and a desktop card count (FS-03.R107/A83, FS-02.R83/A64, TS-08.R146,
  TS-04.R104, TS-02.R59). Waiting to start.

- **2026-10-26 — File-viewer Mermaid sizing (FS-03.R37):** the viewer's shrink-wrapped
  (`fit-content`) `pre` left Mermaid's `width="100%"` SVG no definite width, so it fell back to
  300×150 at the left edge. A `pre:has(.mermaid-diagram)` now takes full width and the figure
  centers the SVG. `check:styles` passed; a Chromium layout probe measured 300×150 before and
  full width after.

- **2026-10-26 — Provider-failure fixes (INV §8/§11/§15/§16):** PERR-1 a prompt result carrying a
  current error-severity AIR failure (e.g. Codex exhausted 429 retries) ends `error` with one
  visible transcript error and error status, not `end_turn`; PERR-2 it settles unsuccessful
  (claimed continuation holds `non_quota_failure`) and confirms no inline mail; PERR-3 one safe
  correlated `provider turn failed` log (TS-04.R85); PERR-4 new/forked Codex sessions are named
  via local `/rename` before the first prompt, removing the background title-model turn
  (new TS-04.R103). TS-04.R86 extended. Each regression failed before its fix; `make test`
  (untagged + FTS5) and `make build` passed on the fix tree. No generic Chuck
  retry loop exists; the field 429 frequency cause stays undetermined (cross-conversation bursts
  and quota-recovery waves remain documented, unpaced risks).

- **2026-10-26 — Provider-failure investigation:** confirmed silent terminal non-quota failures
  and false success settlement; recorded PERR-1/2 plus missing terminal-result diagnostics PERR-3
  and one skipped reproduction. Expanded comparison adds PERR-4's unused background title-model
  request and records cross-conversation burst/recovery/mail patterns. No generic request-retry loop
  or increased retry setting found; frequent-429 cause remains undetermined without correlated
  request history. Product behavior unchanged.

- **2026-10-26 — Memory fixes (INV §4/§16):** MEM-1 revoking a launch token closes its MCP
  transport sessions (TS-04.R7); MEM-2 archive pages release history with their last reader and
  fence late fetches; MEM-3 journal close flushes a stopped turn's pending search text (TS-02.R16);
  MEM-5 draft-index eviction drops cached text (FS-03); MEM-6 ended children release open
  reasoning spans (TS-04.R63). Each regression failed before its fix. MEM-4 open on a decision.

- **2026-10-26 — Memory investigation:** confirmed MCP/session, archive, stopped-turn index,
  live transcript, draft-cache and child-span retention findings; recorded one MEM fix unit and
  three skipped reproductions. Product behavior unchanged; actual field attribution remains open.

- **2026-10-25 — v0.20.0 published:** `main` and annotated tag `v0.20.0` at `a774d128` were
  pushed (29 commits) on the operator's instruction. Untagged and FTS5 Go suites, full UI suite
  (103 files/1055 tests), `make dist VERSION=0.20.0` and the architectural release gate passed.
  Release CI 40232088716 and main CI 40232088269 passed; archive size/SHA256 match the manifest.
  Settled history moved to the 2026-10-25 archive. Manual gates remain owed.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest published notes: [v0.20.0](../archive/state/RELEASE-v0.20.0-notes.md).
Earlier publication/campaign records remain in the archived handoff.
