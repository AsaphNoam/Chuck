# Chuck — Implementation handoff

Settled campaign/review history through 2026-10-23 is archived in
[HANDOFF-through-2026-10-23](../archive/state/HANDOFF-through-2026-10-23.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archived snapshots are not live queues.

## Current position

- **Published:** [v0.18.1](https://github.com/AsaphNoam/Chuck/releases/tag/v0.18.1),
  2026-10-23, source `dc979057`; release CI 39959315460 passed. Published archive size and
  SHA256 match the manifest. Main CI 39959316204 failed twice; REL-CI-1 is repaired locally
  by `1b998b24`, with both full Go variants passing. The repair has not been pushed.
- **Released repair:** fixes-only `v0.18.1`. Product repair `69082b39` refuses unreadable or
  malformed dashboard PID files before offline reindex opens SQLite. Approved test-only corrections
  are `001ba8b5`. The operator approved the version, 23 unpushed commits and annotated tag.
- **Review state:** ASW/ASF and CR-A1–A7 are closed. The combined next-release architecture review
  covers `eb56b0c2` with exact later coverage/repair updates in
  [its receipt](../archive/reviews/architecture/v0.18.0-next.json).
  ARCH-1 and REL-VERIFY-1 are verified closed; no second architecture review is needed.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`. Clean release verification uses `/private/tmp/chuck-release-v0.18.1`.
- **Future work:** direct-action migration remains paused on its transport blocker; passive quota
  snapshots remain TBD. No future implementation is selected by this release.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Existing cancellation-race/native/device limitations remain in the owed gates. Prior failing
  bootstrap/recovery fixture runs are retained in the release notes, not represented as passed.

## Active change

- No change in progress. The timed and recurring task starts unit is closed: TTS-1–3 are fixed
  and verified (see changelog); the fix creates no new review unit.
- Workflow verification cleanup is finished and waiting for review in the separate unit below.

## Acceptance gates still owed

- **ASF-2 native/device receipts:** deterministic fake-ACP wire counts, HTTP/CLI/component checks,
  and isolated desktop Chromium withdrawal/unknown-recovery journeys establish the repaired input
  boundaries. They do not establish credentialed Claude/Codex consumption timing or a physical
  phone's background/reconnect behavior. Keep those existing native/phone gates open.
- **Earlier carried checks:** phone component proof for definite-refusal Retry, live Claude
  receipt for late tool input, and rendered verification of the withdraw failed-recovery-read
  tail notice remain owed.
- **ASF-1 live setting receipt:** credentialed mid-turn effort/fast application remains unverified.
- **Test limits:** the concurrent closure hit the known quota-dispatch FTS5 transcript flake;
  this release's full suites and five focused repeats pass. Two known runtime cancellation tests
  fail under `-race` on the earlier baseline as recorded by ASF-1; no repair is claimed here.

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

- **Annotation input font and focus** (FS-13.R33/A23), waiting for review. Desktop agent
  and room instruction fields focus on mount; instruction and overall fields use the regular
  text font at 1rem. Focus regression and full UI suite pass (99 files, 1001 tests).
  Spec checks, production build (including presentation/style checks), and diff checks pass.
  Native macOS keyboard verification remains unclaimed. Existing unrelated design edits are
  preserved outside this unit's commit.

- **Workflow verification cleanup** (TS-06.R5, TS-08.R17), waiting for review. UI-only changes
  no longer require full Go suites; local checks follow the changed source/tooling, valid checks
  can be reused across roles, fix runs close once, and successful logs stay compact. `npm test`
  runs Vitest directly; `prebuild` retains the style gate. Entry points: workflow §§2/8/16,
  mirrored release skills, `ui/package.json` and the presentation checker. Verification on this
  unit's exact changed scope: `make check-specs` passes; focused `npm test --
  src/presentation/VisualMatrix.test.tsx --reporter=dot` passes 8 tests; `npm run build` passes
  Stylelint, 47 script tests, presentation audit, TypeScript and Vite. The new regression fails
  against the pre-change checker for its obsolete `pretest` requirement. Release skill metadata
  and twin parity pass via Node YAML; the Python validator is unavailable without PyYAML.
  CI retains full coverage; no full product suites or live-provider checks were needed here.

## Review findings

ARCH-1 is repaired by `69082b39`; REL-VERIFY-1 is closed by the explicitly approved
test corrections `001ba8b5`. Both full Go variants pass. Original reports and prior failed runs
remain in the archive/release notes; exact repair and affected coverage are in the
[architecture receipt](../archive/reviews/architecture/v0.18.0-next.json).

## Changelog

- **2026-10-24 — Fix: timed and recurring task starts closed (TTS-1 INV §1, TTS-2 INV §7/§8/§11,
  TTS-3 INV §10):** schedules store their pending occurrence's date and rule time (migration 62)
  and rebase it to the process zone on the first due pass, so a restart in another zone runs at
  that zone's wall time without replay (TS-10.R56 updated). Failed schedule reads show error/retry
  or stale cached data instead of zero attention. Re-arm can set a new start time with Run only on
  time. New Go zone tests (west/east) and four UI read-failure tests fail on the old code; the
  re-arm test fails without its fix. `make test` (both variants), 1006 UI tests and UI build pass.
  No native macOS zone change or live-provider receipt is claimed.

- **2026-10-23 — Workflow verification cleanup:** scoped local checks, removed duplicate UI
  style runs, allowed reuse of unchanged passing evidence, and clarified one completion pass per
  fix unit plus compact test logs. Focused command/build and spec/launcher checks pass; waiting
  for review. Existing product findings and manual gates are unchanged.

- **2026-10-23 — REL-CI-1 closed (INV §5/§17):** `1b998b24` settles pending permissions
  before cooperative cancellation, serializes concurrent cancellations and rechecks the captured
  turn before notification. The strengthened wire-order regression fails on the original code;
  controlled publication/claim barriers replace its scheduling sleep. Three focused permission/
  cancellation runs and ten focused race runs pass. `make test` passes both full variants with
  approved loopback access; the first sandboxed attempt was stopped after loopback bind failures.
  `make build` and diff checks pass. FS-03.A5 records the coverage. REL-CI is closed; TTS-1–3
  remain open. No live-provider receipt or new CI run is claimed.
- **2026-10-23 — v0.18.1 published:** explicit operator approval covered the 23-commit push
  and tag at `dc979057`. Release CI 39959315460 passed; archive, manifest and installer are
  attached, and downloaded archive checksum/size match. Main CI failed twice on REL-CI-1;
  the failure is retained as an open finding. Manual acceptance gates remain owed.
- **2026-10-23 — Implementation:** timed and recurring task starts shipped (FS-16.R51–R58/A35–A40,
  TS-10.R54–R60): clock-time arms with Run only on time, closed-rule schedules that create
  ordinary tasks, HTTP routes, agent tools and the desktop Schedules section. FS-16 and TS-10 are
  now Current. Waiting for review.

- **2026-10-23 — Review:** timed and recurring task starts remain open for TTS-1–3: timezone
  restart scheduling, silent schedule-read failures, and missing time re-arm controls. Focused
  automated checks pass; the timezone regression reproduces. Fix routing is recorded with the unit.

- **2026-10-23 — Release preparation:** ARCH-1 repair `69082b39`, exact post-review coverage,
  truthful failed-check records and approved fixture corrections `001ba8b5` are committed.
  Both full Go variants, focused repeats and static checks pass; ARCH-1 (INV §7/§9) and
  REL-VERIFY-1 (INV §17) are closed. Settled campaign history moved out of the live handoff.
  No publication or manual acceptance closure is claimed.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest published notes: [v0.18.1](../archive/state/RELEASE-v0.18.1-notes.md).
Earlier publication/campaign records remain in the archived handoff.
