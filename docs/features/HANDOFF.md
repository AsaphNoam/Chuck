# Chuck — Implementation handoff

Settled history through v0.19.0 is archived in
[HANDOFF-through-2026-10-24](../archive/state/HANDOFF-through-2026-10-24.md) and earlier
[HANDOFF-through-2026-10-23](../archive/state/HANDOFF-through-2026-10-23.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archived snapshots are not live queues.

## Current position

- **Published:** [v0.19.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.19.0),
  2026-10-24, tag at `308ad23f` (timed and recurring task starts). Release CI 40104625973 and
  main CI 40104624461 passed; downloaded archive size and SHA256 match the manifest.
  [Notes](../archive/state/RELEASE-v0.19.0-notes.md).
- **Review state:** the v0.19.0 architecture review
  ([receipt](../archive/reviews/architecture/v0.18.1-next.json)) is closed: ARCH-v019-1/2
  fixed and verified. Project-scoped groups/selectors review is closed; PSG-R1/2 repaired.
  Saved file-viewer tabs have four open findings (FVT-R1–R4).
  Annotation font/focus and workflow verification cleanup
  shipped unreviewed at the operator's choice and remain in the review queue.
- **Checkout:** `main`, repository `AsaphNoam/Chuck`, pushed through `308ad23f`. Clean release
  verification uses a detached worktree at `/private/tmp/chuck-release-<version>`.
- **Future work:** saved file-viewer tabs are implemented and awaiting fixes for FVT-R1–R4.
  Direct-action migration remains paused on its transport blocker; passive quota
  snapshots remain TBD.
- **Checks:** unset `CHUCK_RUNTIME_ROOT`; use `GOFLAGS=-p=1` and a writable isolated Go cache.
  Native/device/credentialed limitations remain in the owed gates.

## Active change

- No change in progress. Saved file-viewer tabs reviewed; FVT-R1–R4 await fixes.

## Acceptance gates still owed

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

- **Saved file-viewer tabs** (FS-03.R101–R106/A80–A82, FS-20.R56/A28, FS-21.R59/A42,
  TS-08.R140–R145), reviewed 2026-10-25; awaiting fixes for FVT-R1–R4 below.
  Shared `ui/src/features/fileTabs/`: pure model/codec
  (`chuck.file-tabs.v1`, 50/500/2 MiB admission, no eviction), store (reactive owners,
  non-reactive per-tab views, coalesced saves, inline notice), 4-slot read limiter,
  `FileTabsReader` (keyed per-tab `useFileReader` hosts portalled into the docked slot or Focus
  file), `useFileTabRoute` (desktop/room address projection; first open pushes, POP to a
  file-free address hides; applied once per location) and `useFileTabSet` (phone, no address).
  `lib/fileLinkParams.ts` was folded into `readFileTarget`/`writeFileTarget`. Evidence: UI suite
  (102 files, 1045 tests), build and presentation checks pass; `ui/scripts/file-tabs-journey.mjs`
  passes 612/612 (`/tmp/file-tabs-journey`) and `file-reader-journey.mjs` 999/999 after three
  tab-behaviour edits. The journey first caught three defects, fixed before closure: the active
  tab not revealed after restore, its close control clipped, and dev StrictMode hiding restored
  tabs. Reviewer notes: Focus file's view is portalled from outside the dialog's React tree
  (menus get `MenuPortalContext` explicitly; "Return to chat" no longer uses `Dialog.Close`);
  Reload is ignored while a read is pending; phone link refusal is inline, desktop room-source
  refusal is a toast.
  Independent review covered `56ecd6e4`, `1935914a`, `80613f8b`, `abb54bff`, `0c143ee5`
  and `7f67f533`; administrative `1869ed6f` is outside the unit. Focused model/store tests
  pass (24 tests). Independent Chromium journeys pass desktop main (23 checks), Focus file
  (5), room at 1024px (9), Core phone at 360px (18) and Sky & Grove phone at 390px (18); existing full-suite/build
  evidence is reused.
  Additional keyboard/citation checks use the working UI through a loopback Vite proxy to
  the isolated `TestGroupBrowserFixture` fake-provider server: Enter drops focus to BODY,
  and a repeated line-12 URL restores line 100 instead. Evidence:
  `/private/tmp/chuck-tabs-review-live/` (report and Core/Sky & Grove screenshots),
  `/private/tmp/chuck-tabs-review-{core,render,room,phone,phone360}/` (journey reports/screenshots).
  The pre-existing embedded bundle predates tabs and was not rebuilt or committed.
  Temporary negative route/phone tests reproduce FVT-R1–R3 and were removed after review.
  Accepted local choices: explicit menu portal context in Focus file, coalesced Reload while
  pending, inline phone refusal and desktop room-source toast. Storage denial reports on
  first open while retaining memory operation; no material defect there.
  Invariant sweep: applicable classes 1–4, 7–11, 13–14, 16–17; no applicable surface for
  classes 5–6, 12 or 15. No other violation found. Physical-phone touch/reconnect,
  non-Chromium engines, and rendered 500-total/2 MiB/storage-failure gates remain owed above.
  Generated embed was not committed.

- **Skip permissions by default** (FS-04.R61/A40), waiting for review. Fresh/fallback global
  configuration now enables `skip_permissions`; No persona and all seeded personas inherit it.
  Existing saved global choices, persona overrides and frozen sessions retain their policy.
  The fresh-home/no-clobber regression fails against the old default and passes after the change.
  Three existing CLI/server permission journeys now explicitly select manual prompting.
  Final `go test ./...` and `go test -tags sqlite_fts5 ./...` pass with `GOFLAGS=-p=1`,
  `GOCACHE=/private/tmp/chuck-permission-default-gocache` and approved loopback access.
  `make build`, `make check-specs` and diff checks pass for the changed scope. Logs:
  `/private/tmp/chuck-permission-default-{go-final,fts-final,build,spec}.log`.
  An initial task lifecycle-contention failure passes both its focused recheck and final full
  variants; no lifecycle repair was made. No new credentialed-provider claim is made.

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

### Saved file-viewer tabs — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

Four open findings; same review unit as above.

- **Must fix** — **FVT-R1 (INV §1/§10): late normalization rewrites the address after Back.**
  `ui/src/features/fileTabs/useFileTabRoute.ts:147–150` projects the active tab after a
  normalized read without respecting the route's hidden state. Open a file with a delayed
  read, press Back to the same owner's file-free address, then let the normalized reply
  finish: its still-mounted TabHost calls `resolved`, replacing the address with `?file=`.
  Temporary hook regressions fail for conversation and room owners: expected empty search,
  observed `?file=a.md` and `?file=a.md&source=s1`. This violates FS-03.R106/TS-08.R143's
  promise to keep Back file-free. Preserve descriptor normalization while suppressing
  projection for the hidden reader; cover a delayed read completion after Back on both routes.

- **Must fix** — **FVT-R2 (INV §1/§10): a repeated explicit citation loses to the saved position.**
  `ui/src/features/fileTabs/useFileTabRoute.ts:96` emits a jump only for a new tab or a
  different stored line. Revisit an explicit `?file=a.txt&fileLine=12` URL when the saved tab
  already records line 12 but was subsequently scrolled elsewhere: `useFileReader` receives
  an initial view with jump zero (`FileViewer.tsx:68`) and restores the old anchor instead.
  Browser-confirmed with line 100 saved: scrollTop 1992, cited line 12 above the viewport at
  -1513px. A temporary hook test likewise finds no jump command. FS-03.R101/R103 and
  TS-08.R140/R143 require explicit citations to take precedence. Treat every externally
  applied cited URL as a navigation command; test remount and Forward with the same citation.

- **Must fix** — **FVT-R3 (INV §2/§10): phone nested links drop their cited line.**
  `ui/src/remote/AgentScreen.tsx:333` validates the tracked path, then opens only
  `{ path: link.path }`. Follow a local link to `./other.go:12` inside a tracked file: the right
  file opens but its line is discarded, contrary to FS-03.R101/FS-20.R56/TS-08.R140.
  The real AgentScreen integration reproduces this: the admitted `src/other.go` descriptor
  has `line: undefined`, expected 12. Preserve the parsed line alongside the validated path
  without changing tracked-file authority; extend the existing permitted/refused link test
  to assert the destination citation and subsequent jump.

- **Must fix** — **FVT-R4 (INV §1/§10): keyboard tab selection discards keyboard focus.**
  `ui/src/features/fileTabs/FileTabs.tsx:59–64,93–95` renders the strip inside the selected
  TabHost. Selecting another tab removes the focused strip and mounts another; only closing
  sets the focus-restoration flag. In Chromium at 1024px, focus A, ArrowRight to B, Enter:
  B selects but activeElement becomes BODY, and ArrowLeft can no longer navigate the tabs.
  This breaks FS-03.R102/TS-08.R145's keyboard/roving-focus contract on the shared reader.
  Retain strip focus or deliberately transfer it to the selected tab after activation;
  add an ArrowRight → Enter → ArrowLeft browser regression that checks focus at each step.

Project-scoped groups/selectors PSG-R1/2 closed on 2026-10-25.

Closed findings through v0.19.0 are archived with their receipts.

## Changelog

- **2026-10-25 — Saved file-viewer tabs reviewed:** four Must-fix findings cover Back,
  repeated citations, phone citation preservation and keyboard focus. Storage/model checks
  and focused rendered journeys pass; the same unit remains open for fixes.

- **2026-10-25 — Project-scoped groups/selectors fixes (INV §1/§9/§10):** PSG-R1 flushes
  pending layout on navigation and orders saves before reopening reads; PSG-R2 resets outside
  annotation recipients for fresh agent/room batches without remounting delivery ownership.
  Controlled regressions failed before repairs and pass after; full UI suite passes 102 files /
  1048 tests (`/tmp/psg-ui-test.log`); UI build/style/presentation checks, `make check-specs`
  and diff checks pass (`/tmp/psg-ui-build.log`, `/tmp/psg-specs-final.log`). Browser/device
  gates above remain open. Existing dirty
  embedded index preserved; generated embed is not part of this closure.

- **2026-10-25 — Saved file-viewer tabs implemented:** desktop conversations (live/archive,
  Focus file), rooms and phone Files share one tabbed reader with browser-saved tabs.

- **2026-10-25 — Project-scoped groups/selectors reviewed:** two Must-fix findings: collapse
  persistence on rapid navigation and annotation scope reset on a fresh batch. Independent
  26-check browser journey passes; targeted browser reproductions confirm the gaps. Room,
  desktop and phone rendering checked in Core and Sky & Grove. Same unit remains open for fixes.

- **2026-10-24 — Default permissions:** fresh installs now skip permission prompts for No
  persona and all seeded personas through the inherited global policy. Saved settings and
  explicit overrides survive. Both Go variants, build, spec and diff checks pass; ready for review.

- **2026-10-24 — v0.19.0 published:** the operator approved the version and asked to release;
  `main` and tag `v0.19.0` at `308ad23f` were pushed (23 commits). Release and main CI passed;
  archive checksum/size match. The shipped operator guidance covers start times and schedules.
  Settled history moved to the 2026-10-24 archive. Manual gates remain owed.

- **2026-10-24 — Project-scoped groups and selectors:** typed group names are kept, group
  suggestions and collapse follow the project, and annotation/room pickers start at the current
  project with Include other projects. Browser journey, UI and Go suites pass; ready for review.
- **2026-10-24 — Design ready:** saved file-viewer tabs
  are waiting to start: FS-03.R101–R106/A80–A82, FS-20.R56/A28, FS-21.R59/A42 and
  TS-08.R140–R145. The operator confirmed browser/device persistence, existing phone access/screens,
  desktop rooms, 50/500 caps without eviction and Back hiding without deleting tabs. No product
  code changed. Spec lint, design-feature twin comparison and `git diff --check` pass; rendered
  and actual-device checks are implementation gates, not claimed by this design-only unit.
- **2026-10-23 — Workflow verification cleanup:** scoped local checks, removed duplicate UI
  style runs, allowed reuse of unchanged passing evidence, and clarified one completion pass per
  fix unit plus compact test logs. Focused command/build and spec/launcher checks pass; waiting
  for review. Existing product findings and manual gates are unchanged.

- **2026-10-23 — Design in progress:** file-viewer tabs recorded under Ideas being defined.
  The requested open/close/reorder behavior needs scope confirmation for duplicate files,
  reader lifetime and supported surfaces before technical design. No product code changed.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest published notes: [v0.19.0](../archive/state/RELEASE-v0.19.0-notes.md).
Earlier publication/campaign records remain in the archived handoff.
