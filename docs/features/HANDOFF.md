# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-14 is archived in
[`HANDOFF-through-2026-10-14`](../archive/state/HANDOFF-through-2026-10-14.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Finished — task completion in sequential skill workflows fixes (2026-10-17):** fix unit
  `task-completion-workflows-2026-10-17` (TC-1–TC-4) closed; see changelog. FS-16.R49/A33,
  FS-01.R52, TS-04.R99. Owed: FS-16.A33 credentialed Codex `$review` assignee that reads and
  reports its task under real provider policy (Acceptance gates still owed).

- **Finished — mobile annotation parity fixes (2026-10-17):** fix unit
  `mobile-annotation-parity-2026-10-17` (MA-1–MA-3) closed; see changelog. FS-13.R28–R31,
  FS-20.R55, TS-08.R133–R136, TS-13.R31. Owed: independent rendered review and device gates
  (Acceptance gates still owed).

- **Finished — roomier file reader fixes (2026-10-17):** fix unit
  `roomier-file-reader-2026-10-17` (RFR-1–RFR-5) closed; see changelog. FS-03.R92–R97/A71–A75,
  TS-08.R127–R132. Owed: independent rendered review (Acceptance gates still owed).

- **Awaiting fixes — late tool input event (2026-10-17):** unit
  `late-tool-input-2026-10-17`, `a77bb3b6`; review complete, LTI-1/LTI-2 open (two P2).
  Findings and fix routing below. TS-04.R98; live Claude receipt remains owed.

- **Finished — quiet quota recovery fixes (2026-10-17):** fix unit
  `quiet-quota-recovery-2026-10-17` (Q1–Q9) closed on `d3ea0f8f^..9fa9bc81`; see changelog.
  Owed: live FS-01.A28/A31/A33/A34 receipts, A32 full owner/race matrix, A33 real-browser
  cancel/reload/disable journeys.

- **Finished — peer transcript reads and approved Stop fixes (2026-10-18):** fix unit
  `peer-transcripts-approved-stop-2026-10-16` (PS-1–PS-6) closed; see changelog. FS-06.R39–R44,
  TS-04.R93–R95, TS-02.R49. Owed: real Claude/Codex host-gate receipt, physical-device and
  per-kind (task/pipeline/Think Tank/terminal) Stop parity gates (Acceptance gates still owed).

- **Finished — desktop default-persona toggle (2026-10-16):** `33226f23`, FS-04.R58/A37,
  reviewed with no findings; no open fixes (evidence in `/tmp/persona-review-20261016/`).

- **Awaiting fixes — phone New agent from a group header (reviewed 2026-10-17):** unit
  `phone-group-new-agent-2026-10-16`, implementation `7ff852ac`; PGN-1 open (P2).
  FS-20.R46/A18. Independent rendered review passed all appearances at 360/390/430px;
  the group-header plus remains enabled offline. See Review findings for evidence and the
  remaining two-device acceptance gate. Fix model: trivial/easy — Claude Sonnet or Codex Luna.

- **Finished — withdraw returns the held message (2026-10-16, fixes 2026-10-17):** FS-03.R90/A70.
  Implementation `2cf48ba2`; review findings W1–W3 closed (see changelog). Withdraw from
  the desktop composer, the pending tail, or the phone writes a text-only held message back into
  that agent's draft (`restoreChatDraft` in `ui/src/components/chat/drafts.ts`; withdrawn text
  first, then any draft typed since). Attachment holds refresh R84's recovery offer instead.
  Review accepted the join-with-blank-line rule for a non-empty composer as specified and reversible.
  Fix closure: 902 UI tests, UI build and the serial Go matrix pass. Embedded `dist` was not
  refreshed. The failed-recovery-read tail notice reuses the held-tail classes; no rendered check.

- **Finished — Think Tank setup upgrade (2026-10-15):** goal attachments, judge synthesis
  instructions, model/effort display, Add new agent placement and Core selection highlight in every
  skin. Review 2026-10-16's four findings are closed: startup recovery resolves committed room
  blobs by command identity, long names stay within the setup dialog, setup actions use ordinary
  buttons, and synthesis instructions are documented on the operator display and detail wire.

- **Finished — agent context continuity (2026-10-14):** C1–C9 closed locally. Instructions,
  tool evidence and bounded recent history survive handoffs; the separate bridge survives native
  resume/restart. Native-load failures refuse replacement; unverified provider handoffs refuse
  before teardown. Full serial tests, tagged build and focused race checks pass. Live provider
  receipts remain gates below. Original evidence and closure are in
  [`INVESTIGATION-agent-continuity-2026-10-14`](../archive/state/INVESTIGATION-agent-continuity-2026-10-14.md).
- **Finished — stopped-agent image refusal (2026-10-14):** unknown image capability survives old
  snapshots; explicit false blocks image Send with a visible reason. Fix unit closed.
- **Release:** `v0.16.0` published 2026-10-14 (Latest): release CI run 38593397312 passed and
  the GitHub Release carries the archive, `install.sh` and `manifest.json`; body set from
  [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md). Manual provider gates
  below remain owed.
- **Main CI UI test races fixed locally (CI unconfirmed):** `NewAgentModal` tests now wait for an
  enabled Launch and mock `/api/config`; the phone successive-gap test has a 20 s timeout. Full
  UI suite passes locally; remote CI confirmation remains owed. Range `v0.15.1..v0.16.0`: the shipped
  `operating-chuck` Think Tank reference now names the discussion `pass` disposition and the
  quiet-round ending; nothing else agent-facing.
- **Repository:** `AsaphNoam/Chuck`; branch `main`.
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
- **Mail receipt assertion race fixed (2026-10-15):**
  `TestMailActivationStartsIdleAgentWithoutMutatingMailProvenance` now waits for the inline
  receipt instead of reading it as soon as the prompt file appears (it failed 1/10 isolated before
  the mail-notice change and 4/10 after; 30/30 pass across both variants now).

## Active change

None.

## Acceptance gates still owed

- **FS-16.A33 — task reporting under real provider policy:** run a credentialed sequential
  workflow (Claude `/work` → Codex `$review` → Claude `/fix` as tasks); each assignee must read
  its task and record `report_task_result` without supervisor prompting, and a late native
  background continuation must leave the card idle (FS-01.R52).

- **FS-20.A18 — phone group-header New agent:** independent phone-entry browser rendering
  passes, but the stubbed API journey does not prove accepted launches propagate to desktop and
  another phone. Retain the two-device 390px journey for this entry point after PGN-1 is fixed.

- **Mobile annotation parity independent rendered review (workflow §14.4):** the review's
  `phone-render.mjs --annotation-check` attempt failed to bind `::1:5199` with `EPERM`.
  Approval-capable independent browser execution remains required; the implementer's stubbed
  browser journey does not close this review gate.

- **FS-13.A19–A21 / FS-20.A27 (mobile annotation parity):** paired iPhone and Android native
  long-press selection inside events and tracked files, then Annotate, with the keyboard open
  over the tray (delivery/errors reachable). The stubbed `phone-render.mjs --annotation-check`
  journey uses programmatic ranges and a short viewport, so it cannot close these.

- **FS-03.A71–A75 (workflow §14.4):** independent real-browser review of the file reader; the
  review's loopback harness was sandbox-blocked and the implementer's rendered checks do not
  count. The fixer's run binds loopback fine: `file-reader-journey.mjs` (now with a
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

### Review 2026-10-17 — phone New agent from a group header — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Selected unit: `phone-group-new-agent-2026-10-16`, implementation `7ff852ac`; checked the
current form extraction in `041fa5c1` for preservation of this behavior. Governing requirements:
FS-20.R23/R35/R43/R46/A18, FS-12.R68 and TS-08.R111. No product code or specs changed.

- **Must fix** — **PGN-1 / P2 (INV §8/§10): disable the group-header launch entry offline.**
  `ui/src/remote/ProjectScreen.tsx:54` renders `.phone-group-new` without `disabled={offline}`.
  With the phone in `unreachable`, the banner says actions are off and Stop/Archive are disabled,
  but the plus still opens New agent. Create itself is disabled, so no launch is sent; the new
  entry still violates FS-20.R46/A18's explicit offline contract. Add the existing offline guard
  to this button and cover named/Ungrouped buttons while disconnected (disabled, no dialog),
  then reconnect (enabled, correct preselection). Fix complexity: trivial/easy, localized UI guard
  and regression test. Browser probe `/tmp/phone-group-review-20261017.mjs` explicitly sets the
  connection store to `unreachable` and reports `plusDisabled:false,stopDisabled:true,
  opensOffline:true,createDisabled:true`; screenshot
  `/tmp/groupnew-review-20261017-offline-dialog.png`.

Checks: 15/15 `ProjectScreen.test.tsx` tests; `npm run check:styles` (46 script tests and
presentation contract) passed. Approved loopback/Chromium execution of
`phone-render.mjs '' /tmp/groupnew-review-20261017.png --details-check` passed all three
appearances at 360/390/430px, including 44px plus targets and collapsed dialogs; inspected
group-header screenshots for Core, Sky & Grove and Studio and the settled named-group dialog.
Evidence `/tmp/groupnew-review-20261017-*-group-header-*.png` and
`/tmp/groupnew-review-20261017-group-dialog.png`. Initial sandbox `::1:5199` EPERM was resolved
by approved bounded execution. Stubbed API/browser state is not a two-device receipt (gate above).

Local choice accepted: the header passes only an initial group to the same launch form;
ordinary New agent resets it to blank, and Advanced still starts collapsed. Invariant sweep:
§1/§2 satisfied by fresh form state/shared builder; §8/§10 have PGN-1; §13 selector exists and
rendered sizing passes; §17 tests use explicit group/launch expectations. No applicable diff
surface for §3–§7, §9, §11–§12 or §14–§16. Other role queues are unchanged.

State validation: the isolated new review entry passes `check-specs.sh --file`; `git diff
--check` passes. Full `make check-specs` fails on six pre-existing task-completion investigation
errors (four malformed finding prefixes, a superseded unit missing its model recommendation,
and its absolute report link); all offending text is present in the starting HEAD. Concurrent
task-completion product/spec edits were preserved and excluded from this review commit.

### Review 2026-10-17 — late tool input event — **Fix model:** medium — Codex Terra or Claude Opus.

Selected unit: `late-tool-input-2026-10-17`, implementation `a77bb3b6`; reviewed against
TS-04.R98, FS-03.R2, FS-05.R15/R17/R38/R41 and TS-14.R10. Two findings remain; no product
code or specifications changed.

- **Must fix** — **LTI-1 / P2 (INV §1/§16/§17): canceled calls retain full input bodies.**
  `internal/runtime/chat.go:2630–2634,2656–2657` retains each call's title plus complete args
  with no item/byte cap and deletes it only on a tool result. A streamed call interrupted by
  Cancel can end its turn without a terminal tool update; neither turn completion nor the next
  turn releases that entry. A probe through the real cancellation path retains 1, then 2, then
  3 bodies after three canceled turns. Long-lived sessions can accumulate large edit bodies.
  Bound admission and release calls when their owning execution ends, preserving valid late
  child/background ownership rather than indiscriminately clearing live calls. Cover canceled
  and failed turns, repeated turns, normal result cleanup, and bounded active input retention.
  TS-04.R98 / INV §16. Probe: `/tmp/late-tool-review-probe_test.go`, run via
  `/tmp/late-tool-review-overlay.json`; all three zero-retention assertions fail.
- **Must fix** — **LTI-2 / P2 (INV §2/§8/§17): later edits to an already tracked file disappear.**
  `internal/index/indexer.go:450–456` deduplicates late file input by file path rather than
  call identity: `ON CONFLICT(agent_id, path) DO NOTHING`. Two distinct calls, each initially
  empty and later naming `a.go`, leave `edit_count=1,last_seq=2` instead of `2,4`;
  `last_ts` also stays stale, so Files ordering is wrong. This is normal streamed Claude
  edit traffic when no separate diff updates the row. Preserve one edit per distinct call
  while deduplicating that call's subsequent input updates, using the existing tracking seam
  where possible. Test distinct calls to the same path alongside repeated updates of one call.
  FS-05.R15/R17/R38 and TS-04.R98. Independent observational probe:
  `/private/tmp/late_update_probe_test.go` via `/private/tmp/late_update_overlay.json`.

**Verification:** complete untagged index/transcript/contextref package suites pass; focused
runtime/Think Tank/update tests pass, and the focused five-package check passes with
`sqlite_fts5`. Transcript-store and phone AgentScreen suites pass (71 tests). The runtime
lifetime probe fails as described; the independent index probe confirms the incorrect stored row.
The full Go/UI matrix and live Claude receipt were not run; the live receipt remains owed.
No material layout or styling changed in this unit, so workflow §14.4 is not triggered.

**Invariant sweep / local choices:** §1 and §16 fail on LTI-1; §2/§8 expose LTI-2, with §17
regression gaps recorded for both. §5 locking, §7 read/error handling, §9 SQLite behavior,
§10 consumer wiring, §11 event identity/scoping/search exclusion, §13 existing selectors,
§14 existing route guards and §15 capture-before-publication have no additional finding.
§6's event-consumer sweep covers durable projection, context links/share boundaries, index,
Think Tank capture/commands, desktop/phone folding and CLI permission-only status; the runtime
checklist has no new runtime to admit. Classes §3/§4/§12 have no applicable changed surface.
Accepted local choices: append-only updates, byte-identical pair deduplication, title/name fallback,
and dropping orphan UI updates. Bounded phone windows re-fold joined pages, so loading an older
call recovers its retained update. Chuck-tool input remains withheld in the exercised room/share
paths. No speculative multi-path-per-call finding was recorded.


### Usability review 2026-10-14 — stopped-agent image wake (FS-03.A64) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Run report: [`usability-review-run-2026-10-14`](../archive/reviews/usability-review-run-2026-10-14.md).

- **Worth fixing** — **J7/A64 stale Retry after a definite image refusal:** `ui/src/components/chat/Composer.tsx`
  ~L140–168 sets the unconfirmed-send `retry` state for every failed send, so a stopped agent whose
  fresh peer refuses an image (422, nothing consumed) shows “Previous message was not confirmed.
  Retry previous send” beside the refusal. Retry repeats the same 422 with no visible change and the
  notice survives reload. It misstates delivery and invites a dead loop (FS-03.A64 “visibly
  refuses”). Test: a definite 4xx refusal keeps the draft and error but offers no unconfirmed-send
  Retry. (INV §8)

The recovered stopped-agent image implementation unit itself remains closed; live
provider/device acceptance gates and unrelated known risks above remain open.

Closed reviews through the `v0.16.0` cut are in the archive above.

## Changelog

- **2026-10-18 — Reviewer completion investigation (2026-10-17) closed:** the continued trace
  confirms the cause. Reviewer `a_123a8f`'s `/review` instruction put Codex into native review
  mode, and its review child thread (`source: subagent "review"`) ran with
  `approval_policy: never`. The reviewer's parent thread stayed `on-request`/`auto_review`, from
  the user's Codex config mirrored by `codexprofile.go`. Chuck sets no approval policy. The
  17:00Z success came from an ordinary parent turn, not from a policy change. The earlier
  "never → on-request" reading was wrong. Both findings are already closed as TC-1/TC-4
  (`4ddedff1`); no fix is open. Residual: the FS-16.A33 live `$review` receipt is still owed, and
  Codex source for the `never` child was not read. Artifact:
  `~/.chuck/project-resources/agentdeck-20261007t221535z/investigation-reviewer-completion-2026-10-17/INVESTIGATION.md`.
- **2026-10-18 — Peer transcript reads and approved Stop fixes (PS-1–PS-6) closed:**
  PS-1 (INV §9, PID liveness is not identity): `peerExecution` refuses a live orphan unless
  `runtime.ProcessIsRecorded` shows its OS start time (darwin sysctl / linux `/proc`) predates the
  running row's `started_at` + 2 s; unknown start refuses. PS-2 (INV §15): host decisions resolve
  through `emitChecked(required)`, and `permission_resolved` now syncs; an unrecorded decision
  publishes cancelled, settles the waiter cancelled and returns `ErrPermissionNotRecorded`.
  PS-3 (INV §5, generation token + ordered publication): settlers of a host request wait on its
  `published` channel; `hostVoidSeq` (bumped by Cancel) replaces per-approval `voided`, closing
  the take-then-park window too. PS-4 (INV §4/§15): requester crash records cancelled for each
  open host request before the fatal turn end. PS-5 (INV §7): the trailing omitted-record marker
  applies whenever it follows `since_seq`. PS-6 (INV §9/§11): peer cursors `p2` carry a 128-bit
  SHA-256 fingerprint of the prefix, computed in the same scan via `ReadOptions.Tee`; old `p1`
  cursors read as `invalid_cursor`. TS-04.R93/R94, TS-02.R49 updated. Regressions in
  `host_permission_test.go`, `contextref/peer_test.go`, `peer_stop_test.go`, each confirmed
  failing on the prior code; focused `-race` runtime/server passes.

- **2026-10-17 — Task completion in sequential skill workflows fixes (TC-1–TC-4) closed:**
  TC-2 (INV §2, one shared composition): `runtime.LaunchedTaskPrompt` sends a launched task's
  instruction verbatim, then a code-owned paragraph naming `get_assigned_task` and requiring
  `report_task_result`; the dependency activation (existing/woken/retried/continued assignees)
  carries the same `taskReportContract`. TC-1 (INV §12, provider command variance): keeping the
  instruction first preserves deliberate commands, so `create_task`'s description and
  `coordinate-work.md` tell authors to invoke Codex skills as `$name`, never native `/review`
  (whose review mode cannot call Chuck tools). Pinned codex-acp `parseCommand` intercepts only a
  leading `/name` in the first block and passes `$`-names through; the investigation's
  `command-routing.cjs` probe confirmed it. TC-3 (INV §1, derived state across turn end):
  `applyEventStatus` writes tool busy/done status only while `turnActive`; late root tools are
  still transcribed (FS-01.R52). `finishTurn`'s idle-before-release window is unchanged and
  pre-existing. Regression `TestLateRootToolsAfterTurnEndKeepIdle` replaces the skipped
  investigation test. TC-4 (INV §8/§11, bounded protocol error): a contentless `failed`
  `tool_call_update` takes `rawOutput.error.message`, trimmed and clipped to 2,000 runes, as the
  result `error` the shared ToolResult renderer already shows (TS-04.R99,
  `TestMapFailedToolCallUpdateKeepsRawOutputReason`). Investigation artifact:
  `~/.chuck/project-resources/agentdeck-20261007t221535z/investigation-task-completion-2026-10-17/`.
- **2026-10-17 — Mobile annotation parity fixes (MA-1–MA-3) closed:** review of
  `be1dd80d^..9c2206e6` found three P2s, all fixed. MA-1 (INV §5, check-then-act): the phone
  launch sheet ignores Close/Escape/Cancel while `NewAgentForm` reports a pending launch
  (`onBusyChange`), so one batch cannot launch twice. MA-2 (INV §1, derived state across a send):
  acknowledgement calls the new `annotationStore.discardSent`, removing only the submitted draft
  snapshot, so a capture made mid-send survives (desktop shares the fix). MA-3 (INV §1):
  `useAnnotationDelivery` binds callbacks to a per-source token that is live only while that
  owner is mounted; a late launch from a changed/unmounted owner sends nothing, a late delivery
  still clears its acknowledged snapshot but neither updates state nor navigates. FS-13.R29/R30
  and TS-08.R135 updated. New `ui/src/remote/AnnotationDelivery.test.tsx` (6 tests, from the
  review probes; 5 fail on the pre-fix code). Checks: 942 UI tests, UI build, full serial
  `make test` and `make build`. Embedded `dist` not refreshed.

- **2026-10-17 — Mobile annotation parity implemented:** phone rows get a quiet Annotate action
  (in-event selection read on press, else whole event); tracked files open in the shared
  `FileViewerView` with Annotate selection (Source line range, Rendered path only; Reload keeps
  the tracked path key); one tray owner above Chat/Files opens a `PhoneSheet` with sticky
  header/footer; New task preflights then opens the extracted `remote/NewAgentForm.tsx`
  (source-prefilled, active-project chooser), retries delivery to the same launched agent and
  navigates only after acknowledgement; phone rows render `AnnotationCard`. Desktop and phone
  share `components/chat/annotationDelivery.ts`; acknowledged delivery now also ends a desktop
  launched-agent retry. `PhoneSheet` restores focus when unmounted. Checks: 936 UI tests,
  typecheck, `check:styles`, `phone-render.mjs --annotation-check` (9 theme×width journeys,
  screenshots `/tmp/mobile-annot-20261017-annotate-*.png`) and `--details-check`, new remote
  chain Go test, full serial `make test` (both variants) and `make dist`. Review notes: the browser
  journey stubs the API rather than running a fake provider; INV §13 — new `.phone-annotate-*`
  and `.phone-annotatable` selectors defined, `.phone-annotate` removed. Pre-existing:
  `gofmt -l` flags `internal/server/remote_routes.go`.

- **2026-10-17 — Mobile annotation parity ready:** operator confirmed the draft and requested the
  same functionality as Mac. `mobile-annotation-parity.md` (since finished)
  is Waiting to start; FS-13.R28–R31/A19–A21, FS-20.R55/A27, TS-08.R133–R136 and TS-13.R31
  remain planned. Shared capture/delivery, a phone tray accessible from Chat/Files, overall
  instruction, configurable preflighted launch with same-agent retry, and live/replayed cards
  extend existing seams. Source idea promoted; remote file authority/lifetime unchanged; no phone
  archive/room surface. Implementation and browser/device acceptance remain owed; no product code
  changed or active change selected. Independent source/contract consistency check found no scope
  conflict; documented the launch form's optional project chooser and post-delivery navigation seam.
  Spec, twin-skill/launcher and whitespace checks pass.

- **2026-10-17 — Mobile annotation scope drafted:** phone diff annotations already exist. Planned
  FS-13.R28–R29/A19–A20 and FS-20.R55/A27 add transcript excerpt/whole-event capture,
  tracked-file text selections, a shared reachable batch and overall instruction, retaining existing
  targets, browser-local draft lifetime and remote file limits. Source idea is under Ideas being
  defined. Awaiting product confirmation before technical design; no ready change or product edits.
  Spec, twin-skill/launcher and whitespace checks pass.

- **2026-10-17 — Roomier file reader fixes (RFR-1–RFR-5 closed):** unit
  `roomier-file-reader-2026-10-17` closed. RFR-1 (INV §8/§17): `allocateReader` counts the
  dividers its own allocation shows, so docking near ~992px no longer loops; TS-08.R128
  clarified. RFR-2 (INV §1/§8): `ReaderWorkspace` clears its stage occupancy on unmount.
  RFR-3 (INV §1/§8): a new file drops the old snapshot in the same render, so its cited line
  waits for it. RFR-4 (INV §1/§8): only a loaded document records scroll positions. RFR-5
  (INV §1): `AgentStage` takes a `source` (agent id) and resets requests when it changes. New
  regressions in `readerLayout.test.ts`, `ReaderWorkspace.test.tsx` (975–993px open and
  resize, tab leave/return, agent change) and `FileViewer.test.tsx` (short→cited file, slow
  Reload Source/Rendered, shortened snapshot, failure/retry) fail before and pass after. Full UI
  930/930 (the review's TranscriptView failure did not recur), UI build, style/presentation
  checks, serial `make test` and `make build` pass. The review's state commit was made first
  (`6f13725a`). Embedded `dist` not refreshed; independent rendered review still owed.

- **2026-10-17 — Late tool input reviewed:** only `a77bb3b6`; LTI-1/LTI-2 keep
  `late-tool-input-2026-10-17` open for fixes. Canceled calls retain unbounded input bodies;
  distinct streamed calls to the same file lose edit-count/last-touch updates. Independent
  probes confirm both. Focused Go checks (including FTS5), complete index/transcript/contextref
  tests and 71 UI tests pass; live Claude receipt remains owed. No product/spec edits.


- **2026-10-17 — Roomier file reader reviewed:** only `efbf9d03..ad8fb87e`; RFR-1–RFR-5
  (one P1, four P2) keep `roomier-file-reader-2026-10-17` open for fixes. Six independent
  probes reproduce the five defects; cited-file navigation passes on the old viewer.
  Build/presentation checks pass; UI 920/921 with the same baseline fixture failure. Independent
  rendered acceptance is permission-blocked, explicitly retained above. Report/evidence in
  shared resources `roomier-file-reader-review-2026-10-17/`. No product/spec fixes or other
  queue changes; no publication/release.

- **2026-10-17 — Roomier file reader implemented (awaiting review):** FS-03.R92–R97/A71–A75,
  TS-08.R127–R132; ready-change file removed. Contained file Markdown (shared with the room
  viewer); `AgentStage.tsx` allocates file/conversation/tray widths from one `ResizeObserver`
  with two accessible dividers (`readerLayout.ts`); `ReaderWorkspace.tsx` keeps `useFileReader`
  and `useAnnotationDelivery` above the adjacent/Focus file branch; focused menus mount through
  `MenuPortalContext`. Evidence: 921 UI tests, UI build, presentation checks,
  `ui/scripts/file-reader-journey.mjs` (957 checks: 3 appearances × 1024/1440/2137 ×
  live/archived + nested launch), `chat-cleanup-render.mjs --panels`, `room-render.mjs <out>
  file`. The implementation's overlay, native transcript anchoring and one-frame allocation
  notes were assessed in the independent review above; remaining rendered gates are retained
  there and the detailed outcomes are in its report. Embedded `dist` not refreshed.

- **2026-10-17 — File reader design ready:** operator approved the resizable split reader,
  optional Focus file modal, shared annotations and page-lifetime sizing, adding an explicit
  requirement against accidental Chinese characters in authored English UI. Waiting unit:
  `docs/ready-changes/roomier-file-reader.md`; FS-03.R92–R97/A71–A75 and TS-08.R127–R132.
  Technical design extends existing workspace/reader/tray/Radix seams, retaining read and
  delivery owners across presentation changes and containing nested menus inside the modal.
  The idea was promoted; implementation has not started and Active change remains None.
  Incumbent overflow/spacing evidence and official reference links are preserved in the unit.
  Composition sketch: `.chuck-dev/design/file-reader.html` (not product code or an acceptance
  receipt). Spec, twin-skill/launcher and whitespace checks pass; added authored text contains no
  Chinese characters. No product-code change; implementation acceptance and rendered gates remain owed.

- **2026-10-17 — TOOLRUN-03 closed (investigation unit closed):** (INV §1/§6/§11/§16) installed
  `claude-agent-acp` starts a streamed call with `{}` and sends `rawInput`/`title` later in
  status-less `tool_call_update`s. New durable `tool_call_update` event (TS-04.R98):
  `acpmap.go` emits it for non-empty input/title; `agentState.admitToolInput` fills the call's
  name, drops repeats and updates for no open call, and forgets the call at its result. Consumers:
  `transcript.ProjectEvent` (primer/context links; still no search text), contextref render and
  share-span skip (own share input never enters its span), indexer files (no double count) and one
  command row per call, plus the ACP `execute` kind as a command (FS-05.R38), Think Tank capture
  (Chuck tool input withheld, now also when only the title names the tool) and command list. UI
  `appendRenderedEvent` folds it into the call; the call row shows the adapter title (FS-03.R2).
  Full `make test` (both variants, `-p=1`), `make build`, `-race ./internal/runtime`, 909 UI tests
  and UI build pass. Embedded `dist` not refreshed; no live Claude receipt.

- **2026-10-17 — Tool-run fixes (TOOLRUN-01/02 closed; 03 open):** TOOLRUN-01 (INV §2/§8)
  `groupTranscriptRows` keeps an answered `permission_request` whose `tool_call_id` matches a call
  in the run inside that run, so the approved result no longer renders alone on desktop or phone;
  unanswered/unrelated requests still end it (FS-03.R26/A11 amended; un-skipped `toolRun.test.ts`
  plus TranscriptView and AgentScreen cases, all failing before the fix). TOOLRUN-02 (INV §8/§11)
  `ToolResult` shows joined text when every block is ACP-wrapped text, JSON otherwise (FS-03.R2
  amended). 907 UI tests, UI build, spec check pass; no Go changed, Go matrix not rerun; embedded
  `dist` not refreshed.

- **2026-10-17 — Withdraw fixes (W1–W3 closed):** closes the FS-03.R90 unit. W1 (INV §1/§8)
  `drafts.ts` keeps the page's last composer text in memory and `restoreChatDraft` merges with it,
  so unsaved typing survives a storage failure (desktop and phone tests). W2 (INV §7/§8/§10) a
  failed recovery GET after a completed DELETE marks `heldStore.recoveryUnreadByAgent`; the
  transcript tail shows it with "Load withdrawn message", which only re-reads
  (`readHeldRecovery`). W3 (INV §11/§15) `withdrawHeldMessage` shares one in-flight call per agent
  across composer and pending tail. R90/A70 amended. `src/test/setup.ts` resets the memory copy.

- **2026-10-17 — Quiet quota recovery fixes (Q1–Q9 closed):** fix unit
  `quiet-quota-recovery-2026-10-17` closes the original unit. Q1 (INV §2/§12/§17) Claude checks
  send `settingSources:[]` plus only routing keys from user/project/local settings as `settings`;
  unreadable layers or a `CLAUDE_MODEL_CONFIG` clash are unsupported. Pinned adapter 0.85.1 with a
  stub CLI received `--setting-sources=`, `--settings {routing}`, `--no-session-persistence`
  (shared resources `quiet-quota-fix-2026-10-17/`); native CLI honoring is not executed. Q2
  (§11/§12/§17) dotted and nested overlay servers are disabled; ambiguous names refuse. Q3
  (§1/§2/§5/§12/§15) checks reuse the live original's launch-frozen selection
  (`LaunchSpec.QuotaCheckEnv`), a ready check's continuation revalidates it (`check_unsupported`
  when changed or forgotten by restart), and cleanup uses the creating selection. Q4/Q6
  (§11/§12/§17) same-minute and conflicting text stay unknown. Q5 (§1/§11/§12/§17)
  `rateLimitEvidence` keeps model/overage applicability and untimed blockers. Q7 (§2/§10/§11/§17)
  check interventions hold `unknown_reset` with no due. Q8 (§4/§5/§9/§15) deletion refuses pending
  probe cleanup. Q9 (§16/§17) byte-bounded reply. Specs: TS-02.R50, TS-04.R96/R97, TS-10.R45.
  Reviewer probes now pass except the server intervention probe, which injects its failure after
  launch (checks now use the launch-frozen environment; the same case with the failure set at
  launch passes). Serial `make test` (both variants, check-specs) and `make build` pass; focused
  `-race` quota check/cleanup/delete runs pass (server 6/6 after resolving cleanup before the
  ready result, which closed a test-visible ordering window). No UI change.

- **2026-10-17 — Quiet quota recovery reviewed:** `d3ea0f8f^..9fa9bc81`, including the first
  implementation commit. Review stage complete; Q1–Q9 (7 Must fix, 2 Worth fixing) keep the same
  unit open for fixes. Both documented local choices were assessed. Focused Go/UI/render checks
  pass; eight independent overlay probes fail on the confirmed defects. Report/evidence are in
  shared project resources. Live receipts remain owed; no product/spec changes or next stage.

- **2026-10-17 — Quiet quota recovery implemented:** FS-01.R48–R51/A31–A33 (A32 deterministic),
  FS-04.R59/A38, TS-04.R96–R97, TS-10.R45–R47, TS-02.R50, TS-03.R72 in five slices
  (`d3ea0f8f`..HEAD). TS-10 and FS-04 are now Current. Serial `make test` (both variants), `make build`, 897 UI tests, UI build, focused `-race` and the quota render journey pass. Live A28/A31/A33/A34
  receipts remain owed.

- **2026-10-16 — Quiet quota recovery design ready:** operator confirmed the direction, requested
  checks outside the interrupted conversation and accepted native Codex archive probe records.
  [`quiet-quota-recovery.md`](../ready-changes/quiet-quota-recovery.md) is Waiting to start;
  FS-01.R48–R51/A31–A34, FS-04.R59/A38, TS-04.R96–R97, TS-10.R45–R47, TS-02.R50, TS-03.R72 remain
  planned. Existing quota scheduling/claims/owner recovery gain structured/text reset evidence
  and at most six isolated hourly checks, without failed chat wake-ups or a second native
  integration. [Evidence](QUOTA-RESET-FALLBACK-2026-10-16.md) records passive-read limits, exact
  provider lifetime semantics, title suppression and the pinned native MCP-disable contract.
  A small successful check cannot guarantee the larger continuation fits; that mismatch holds.
  Source idea promoted; no product code, no active change, no live provider receipt.
  Spec, twin-skill/launcher and whitespace checks pass; a focused independent consistency check
  found no remaining material blocker after the helper contract and source/provider identity were clarified.

- **2026-10-16 — Quota reset fallback investigated:** recorded pinned adapter AIR/RPC carriers,
  Chuck's text/data loss, native Claude timestamp-unit evidence and Codex local-date formatting
  in [the evidence note](QUOTA-RESET-FALLBACK-2026-10-16.md). Drafted planned FS-01.R48–R50/A31–A33
  and FS-04.R59/A38; scope awaits confirmation before TS/ready-change work. Recommended existing
  recovery with structured → verified-text → bounded unknown-reset retries; no product code.
  Synthetic helper/source receipts are distinct from FS-01.A28/A33's owed live-provider gate.
  Spec checks, twin-skill/launcher checks, whitespace checks, actual pinned helper serialization
  replay and six extracted-native-formatter assertions pass.

- **2026-10-16 — Peer transcript reads and approved Stop implemented:** five slices from the
  ready change; FS-06.R39–R44, FS-03.R91, FS-15.R18/A8, FS-17.R22/A13 and TS-04.R93–R95,
  TS-05.R27, TS-02.R49, TS-03.R71, TS-11.R23 shipped; FS-15 is now Current. Ready-change file
  removed; FS-06.A30/A32 and the live provider receipt remain owed.

- **2026-10-16 — Compact Think Tank chat navigation ready:** operator approved only links to other
  members and the room, with subtle bold/raised active-room-speaker emphasis. FS-21.R57/A40 and
  TS-08.R126 are planned in
  [`compact-think-tank-chat-navigation.md`](../ready-changes/compact-think-tank-chat-navigation.md).
  Reuse existing router/query/button styling; rendered all-appearance acceptance is owed during
  implementation. No product edits; active peer transcript/Stop work remains independent.
- **2026-10-16 — Peer transcript/Stop design ready:** the operator retained existing local-process
  trust. `peer-transcripts-and-approved-stop.md` (since implemented and removed)
  was Waiting to start: FS-06.R39–R44/A28–A32, FS-03.R91, FS-15.R18/A8, FS-17.R22/A13;
  TS-04.R93–R95, TS-05.R27, TS-02.R49, TS-03.R71 and TS-11.R23. Direct reads use shared bounded
  transcript rendering; Stop uses a host-owned, once-only human decision independent of provider
  auto-approval, bound to requester turn/generation and target execution, through existing Stop
  lifecycle claims. Audit distinguishes approval from actual/uncertain effect; pending approvals
  never replay after restart. All new requirements remain planned. Selected idea portion promoted;
  peer Resume/launch remains separate. No product code changed and no active change was selected.
  Documentation closure: spec checks, launcher/twin-skill contract and whitespace checks pass.

- **2026-10-16 — Peer authority scope confirmed:** direct transcript reads across projects,
  including stopped/archived history, and Stop requests for any peer. The operator requires
  explicit user approval for every effective peer Stop. Draft feature requirements are
  FS-06.R39–R44/A28–A32, FS-03.R91, FS-15.R18/A8; no product code changed. Current native
  permissions auto-approve registered Chuck tools, so this needs a host-owned gate independent of
  provider permission behavior. TS-05.R3 still trusts local processes, which can call the existing
  Stop API; awaiting whether to retain that boundary or require shell/API bypass prevention before
  completing technical design and creating a ready change. Idea remains under Ideas being defined.

- **2026-10-16 — Peer transcript reading and stopping design:** selected the read/stop portion of
  `ideas.md`'s broader-agent-authority entry. Current Chuck tools have grant-based context reads
  and agent discovery, but no direct peer transcript read or Stop action. Proposed explicit,
  bounded reads of Chuck's recorded conversation and attributed Stop through existing lifecycle
  behavior. Awaiting the operator's read scope (cross-project/stopped/archived) and stop scope
  (cross-project/managed/terminal versus narrower authority). Feature scope is unconfirmed;
  technical design has not started, no ready change exists, and no product code changed.

- **2026-10-16 — Desktop default persona:** added the Settings → Roles toggle through existing
  partial config updates; no checked persona selects No persona. Implementation `33226f23` is
  available for review. Full UI and both Go variants, builds and three-appearance browser checks pass.

- **2026-10-16 — Withdrawal review:** `2cf48ba2` remains open for live-draft loss on storage
  failure, hidden attachment recovery after a failed read, and duplicated text from overlapping
  withdrawals. Focused UI tests pass; all three failure paths reproduce. Blank-line merge accepted.
- **2026-10-16 — Withdraw returns the held message (FS-03.R90/A70):** desktop and phone Withdraw
  now put a text-only held message back in the composer; attachment holds show the recovery offer.
- **2026-10-16 — Think Tank setup review fixes (INV §7/§8/§10/§15):** startup recovery now
  resolves committed room attachments by command identity while pruning unknown provisional blobs;
  long names remain bounded in setup chips, setup actions use ordinary buttons, and the operator
  display/detail wire coverage for synthesis instructions is explicit. The four findings from the
  2026-10-16 review are fixed. Focused recovery and UI checks, tagged build and UI build pass.
  Rendered long-name checks pass in all skins at 1024/1440; final outlined action treatment is
  confirmed in Core at 1024. The authorized archive test-fixture repair (INV §17) supplies an
  empty Think Tank results response by default; judge-result tests retain their explicit overrides.
  All 885 UI tests pass, including all eight archive-page tests. Full Go matrix passes in both
  variants (`env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 make test`). Verification closure is complete.

- **2026-10-15 — Think Tank setup upgrade (FS-21.R54–R56/A39, TS-14.R29, FS-03.R28):** setup
  takes goal files (chat limits) stored under `attachment.Store` key `tt-<command_id>` and
  offered on every member's first `read_think_tank` as `attachments` paths; judge-only
  `judge_instructions` (migration 53 adds both columns); model · effort on rows, judge and picker
  options; **+ Add new agent…** above the existing-agent picker. Text selection uses Core's
  highlight in every skin. Archived agents were already excluded from the picker (verified
  against live data; now under test). Review 2026-10-16: the crash-pruning note was wrong and
  became a **Must fix**; images-as-paths (R54 “readable copies”) and the v47 fixture stub table are
  sound and closed. Classes 1–6, 9, 11–14, 16–17: no violation found.

- **2026-10-15 — J3/J6/J11 transcript and stopped-state fixes; review closed (INV §8/§13):** the
  transcript stays pinned to the bottom while late rows (diagrams) settle, and only scrolling toward
  older content leaves it (FS-03.R3/A68). A stopped terminal agent opens no terminal socket, says it
  is stopped and offers Resume in the header (FS-07.R24/A10). An agent stopped on error reads
  “Error” on card and header with its detail (never blank) and a restart hint (FS-02.R79/A60,
  FS-03.R89/A69). FS-14.R44 now records R80's Setup → Review supersession. This closes the
  2026-10-15 review unit. Not covered: the phone agent screen still reads “Stopped” for an errored
  agent; Think Tanks, phone remote control, project resources and config-source binding still
  have no usability journey.

- **2026-10-15 — J9/J11 save validation fixes (INV §2/§3/§8):** a malformed pipeline template id now
  reaches the shared validate-before-write path and returns 422 with Validate's diagnostics (new
  FS-14.R90/A56); update with a malformed id is 404. Catalog save refuses a blank backend display
  name through the create path's shared check, and the editor keeps the draft (new FS-09.R82/A51).
  A catalog already holding a blank name still loads. Regressions failed before the fix; focused
  Go (both variants) and UI tests pass. Full closure owed at run end.

- **2026-10-15 — J2/J8/J13/J16 UI fixes (INV §7/§8):** the onboarding Project step holds on a
  `cwd_not_found` warning with Continue anyway; OpenCode/OpenHands guidance names the `opencode`/
  `openhands` command that must be on PATH (the repo documents no install URL — supply one in
  FS-09.R27 to link it); an inactive tray source defaults to an available target; launch, resume
  and switch toast a recreated worktree checkout; the archive page distinguishes unknown id (no
  Resume/Restore), empty transcript and failed read with Retry (new FS-05.R42/A25). Each
  regression failed before its fix; focused UI tests pass. Full closure owed at run end.

- **2026-10-15 — J5/J12 card order fix (INV §1/§3):** the agent stream no longer appends to or
  prunes the stored manual order, the grid saves only a layout that differs from what it read or
  last saved, and unplaced agents sort by creation time then id (new FS-02.R78/A59). The new grid
  regression failed before the fix; focused grid/store/SSE tests and the typecheck pass. A loaded
  layout naming an unknown pane id still saves once (A32). Full closure is owed at run end.

- **2026-10-15 — J10/J13 mail-turn visibility (INV §8):** decided under new FS-06.R38: any turn that
  receives mail in its prompt records one notice row naming each sender and subject (no body), so
  peer-mail and tray-to-agent turns show their cause. The extended activation test fails without
  the notice; runtime package tests pass. Full closure is owed at the end of this fix run.

- **2026-10-15 — J15/J16 task view fixes (INV §1):** deleting a task now publishes a `task_update`
  removal so every open view, including the dashboard attention count, refetches. A finished task
  shows unmet prerequisites and unfired signals as history, without a live wait or signal control.
  Both regressions failed before the fix; focused Go and Tasks UI tests pass. Full closure is owed
  at the end of this review's fix run.

- **2026-10-15 — J13 new-task annotation delivery fix (INV §1/§8/§10/§11/§14/§15/§16/§17):** the tray now runs
  the server's annotation and mail-size validators before opening launch. Once launched, a failed
  delivery names that agent and retries to it instead of launching another; changing source agents
  resets the retry state. The bounded preflight has no transcript or mail side effects and is
  available through the same paired-phone route policy as annotation delivery. Full serial Go
  tests in both variants, all 856 UI tests, Go/UI builds, spec and diff checks pass. The October 15
  review remains open for its other findings.

- **2026-10-15 — J14 pipeline restart fix (INV §1/§2/§17):** restart task recovery interrupted a
  stage task without projecting that transition onto its run. It now uses the existing stage
  interruption seam with a restart reason. The regression failed before the fix; afterward the
  run pauses, offers an effective Retry, and the rendered page withholds Open agent and names the
  restart cause. Full serial Go tests in both variants, all 854 UI tests, Go/UI builds, spec and
  diff checks pass. The October 15 review remains open for its other findings.

- **2026-10-15 — J10 mail state fix (INV §1/§11/§17):** the send callback that refreshes recipient
  unread mail and sender Sent pulse ran only for waking messages. It now runs after every successful
  insert and passes the wake choice so deferred mail does not queue activation. A real MCP-to-bus
  test failed before the fix for deferred sends and passes for both waking and deferred sends; the
  existing read/clear check also passes. Full serial Go tests in both variants, Go build, spec and
  diff checks pass. The October 15 review remains open for its other findings.

- **2026-10-15 — J3 live sent-message fix (INV §11/§17):** a prompt response can arrive after the
  durable user event, reply, and turn divider. The composer now checks for that delivered prompt
  before adding a local echo; repeated identical sends remain distinct. The old behavior failed
  the new late-response regression. Text and attachment send regressions pass. Full Go matrix
  (serial, including `sqlite_fts5`), all 853 UI tests, Go/UI builds, and spec checks pass. The
  2026-10-15 review stays open for its other findings, including the separate J3 reload-scroll issue.

- **2026-10-14 — usability review (J1, FS-03.A64 desktop wake):** real tagged binary, fresh home,
  fake peer as `claude-agent-acp` on PATH, Chromium. J1 and the accept/explicit-refusal variants
  pass; one Worth-fixing stale Retry after refusal recorded above; phone wake not exercised.

- **2026-10-14 — stopped-agent image fix (INV §1/§2/§8/§11/§17):** confirmed the recovered
  regression failed before editing; preserved unknown image advertisements across persistence
  and shared read projections, retained explicit refusal and wake-time validation, and surfaced
  the desktop disabled-Send reason. Added serialization, fresh-wake acceptance/refusal and both
  composer regressions. `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 make test`, all 850 UI tests,
  `make dist` and diff checks pass. No findings remain in this unit; existing manual gates remain.
- **2026-10-14 — context continuity fix (INV §1/§2/§3/§4/§5/§7/§8/§11/§12/§16/§17):** closed
  C1–C9 with canonical bounded primers, durable conversation bridges, explicit model provenance,
  atomic partial-switch composition, fail-closed restore and honest provider capability/refusal.
  `make test` passes both variants serially; tagged build and focused lifecycle race checks pass.
  Live provider verification and the mail-test assertion race remain.

## Blocked on human

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
