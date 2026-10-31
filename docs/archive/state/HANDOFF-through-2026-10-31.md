# Chuck — Handoff archive through 2026-10-31 (v0.21.0)

Settled state moved out of the live handoff when v0.21.0 was released. Non-authoritative history;
see the live handoff and specifications for current state.

## Superseded handoff snapshot (pre-v0.21.0)

Settled history through v0.20.0 is archived in
[HANDOFF-through-2026-10-25](./HANDOFF-through-2026-10-25.md) and earlier
[HANDOFF-through-2026-10-24](./HANDOFF-through-2026-10-24.md) and
[HANDOFF-through-2026-10-23](./HANDOFF-through-2026-10-23.md).
Use [AGENT-WORKFLOW.md](../../features/AGENT-WORKFLOW.md) for role rules; archived snapshots are not live queues.

## Current position

- **Published:** [v0.20.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.20.0),
  2026-10-25, tag at `a774d128` (saved file-viewer tabs, project-scoped groups/selectors,
  skip-permissions default, annotation focus). Release CI 40232088716 and main CI 40232088269
  passed; downloaded archive size and SHA256 match the manifest.
  [Notes](./RELEASE-v0.20.0-notes.md).
- **Review state:** quota recognition unit QUOTA-RPC is closed (QUOTA-RPC-1/2/3 fixed
  2026-10-30); its live receipts stay owed below. Optional annotation source recording unit ANN-SOURCE is closed
  (ANN-SOURCE-1/2 fixed 2026-10-30). Pipeline worktrees fix unit
  PIPE-WT is closed (PIPE-WT-1–6 fixed
  2026-10-30); its packaged Claude/Codex launch/resume cwd probe stays owed below. File-viewer
  Mermaid sizing unit MMD-VIEW is closed (MMD-VIEW-1 fixed 2026-10-31); memory fix unit MEM is
  closed (MEM-4 fixed 2026-10-31; MEM-1/2/3/5/6 previously fixed). Summary-only tool activity unit TOOL-LINE is
  closed (TOOL-LINE-1/2 fixed 2026-10-29). Delegation-guidance fix unit
  DELMSG is closed (DELMSG-1 fixed). The v0.20.0 architectural review
  ([receipt](../reviews/architecture/v0.19.0-next.json)) is closed; ARCH-v020-1 fixed
  and verified. Provider-failure fix unit PERR is closed (PERR-1–4 fixed 2026-10-26); the
  frequent-429 field cause remains unmeasured (needs correlated request history if it recurs).
- **Goal-led pipelines:** reviewed 2026-10-30 (GOAL-LED); fix unit closed 2026-10-31 —
  GOAL-LED-1–14 fixed. Acceptance evidence (A61/A64/A67/A68) remains owed below.
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
- **Phone Home, Sky & Grove wallpaper, composer Cancel:** review closed without findings
  2026-10-31 (`b388707f`, harness correction `1465e136`). Independent phone rendering and 75
  focused tests pass; evidence and invariant sweep are in the changelog below.
- **Group launch defaults and command triggers:** GROUP-SKILL reviewed 2026-10-31; fix unit
  closed 2026-10-31 (GROUP-SKILL-1 fixed).
- **Background-task card count:** BGRUN-COUNT reviewed 2026-10-31; fix unit closed
  2026-10-31 (BGRUN-COUNT-1 fixed).
- **Pipelines redesign:** PIPELINE-FIGMA reviewed 2026-10-31; closed without findings.
  Fresh source, presentation and independent browser evidence is in the changelog below.
- **Boring Night skin:** BORING-NIGHT reviewed 2026-10-31; fix unit closed 2026-10-31
  (BORING-NIGHT-1 fixed).
- **Next architectural review:** combined candidate `0d6d877c` reviewed 2026-10-31; fix unit
  ARCH-NEXT closed 2026-10-31 (ARCH-NEXT-1–4 fixed; closure recorded in the
  [receipt](../reviews/architecture/v0.20.0-next.json)). Final release preparation
  (`make dist`, regenerated embed with affected-coverage receipt update, owed gates) remains owed.

## Active change

None. Running background-work indicator unit BGRUN-UI is closed (BGRUN-UI-1/2 fixed 2026-10-31).

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

None.

## Review findings

None.

## Changelog

- **2026-10-31 — Fixed ARCH-NEXT-1–4 (INV §4/§5/§9/§10/§15/§16):** closes the ARCH-NEXT unit.
  1 (registration cleanup, INV §4/§16): MCP requests snapshot their token's registration; a
  session created while Revoke ran is closed (`d5b2d06e`). 2 (persisted intent/recovery, INV
  §5/§9/§15): migration 69 `attachment_receipts.source_agent_id`; original-conversation replays
  find the moved receipt; startup copies attachments the new owner lacks (`119880aa`). 3 (durable
  operation identity, INV §5/§15): `pipeline_runs.recovery_effect_key` names a charged retry until
  `ConfirmTaskStart`, so its needed replacement reuses the round (`2f804a2f`). 4 (wiring/knowledge,
  INV §10): operating references name run worktrees as the checkout-isolation exception
  (`ed1da637`). Reviewer probes adopted as regressions; each failed before its fix. Both full Go
  variants (per package group), check-specs, build and focused `-race` pass on `ed1da637`.
  TS-02/TS-09 notes updated. Owed: release `make dist` and existing native/provider/device gates.

- **2026-10-31 — Reviewed combined architecture since v0.20.0:** fresh independent system review
  plus focused pipeline, workspace and runtime reviews recorded ARCH-NEXT-1–4. Real SDK HTTP,
  real-store/controller, HTTP/fake-provider and startup-recovery probes establish the failures;
  bounded reproducers and evidence are archived with the receipt. Full UI (1,128 tests), UI build,
  spec, knowledge/CLI and focused runtime/index/server/workspace/Git checks pass. Existing full Go
  evidence is reused; final dist/package and native/provider/device gates remain owed. Product and
  normative specs unchanged; unrelated ideas edit preserved. **Fix model:** difficult — Codex Sol.

- **2026-10-31 — Fixed BORING-NIGHT-1 (TS-08.R151, INV §8/§17):** Boring Night mapped
  `--ad-highlight` to a dark fill, which left syntax function/property names (1.48:1 on the
  technical background) and the composer file-drop outline near-invisible. New public token
  `--ad-highlight-foreground` (Core default `var(--ad-highlight)`, other skins unchanged) now
  carries those two foreground uses; Boring Night maps it to amber `#edc585`. The presentation
  test resolves every appearance's token chain and requires 4.5:1 on the technical background; it
  fails with the skin override removed. It also adds Boring Night to the muted-on-canvas check.
  Review evidence: `/private/tmp/boring-night-review/`.

- **2026-10-31 — Added the Boring Night skin (FS-12.R72, TS-08.R151):** the Figma Make
  exploration's "Chuck Night" palette ships as the dark built-in skin `boring-night` — Go write
  set, `BUILT_IN_SKINS`, contract skin list, Settings/phone/visual-matrix options, and
  `styles/skins/boring-night.css` (private `--ad-boring-night-*` palette, `color-scheme: dark`,
  CSS ledger-grid ornament) imported by desktop and phone entries. Config/API, appearance editor
  and visual-matrix tests cover the new id. Closure with `GOFLAGS=-p=1`: `make test` (incl.
  check-specs), `make build`, `(cd ui && npm test)` (105 files / 1128 tests) and
  `git diff --check` pass on the final tree. Browser check against the
  stress fixture showed Settings/Appearance and the Projects dashboard; remaining rendered
  evidence is owed above.

- **2026-10-31 — Fixed MEM-4 (INV §16/§17):** with operator approval, removed the runtime's
  duplicate full-history collection and unused `Transcript` API. Committed notices alone remain
  for quota/peer-Stop deduplication; sequence bookkeeping, live publication, fork-prefix persistence
  and durable history reads are preserved (TS-01.R4/R5/R39). Tests now observe publication or
  durable writes. The retention regression failed before the repair and passes after; both notice
  retry identities, fork history and 14 migrated focused tests pass. `make test` (both Go variants),
  `make build`, focused journal/retention/notice `-race` and `git diff --check` pass on the final
  MEM-4 source. Logs `/private/tmp/chuck-mem4-{test,build,race,focused,before}.log`; full suites
  required loopback-capable execution. Independent integration review found no remaining concern.
  MEM is closed; no open fix findings remain. Field attribution and real-workload heap/RSS savings
  remain unmeasured. Unrelated dirty `docs/ideas.md` is preserved.

- **2026-10-31 — Fixed GROUP-SKILL-1 (INV §8/§10):** `@` autocomplete now publishes file and
  command results as each request answers (participant → file → command order, existing request
  token), so a slow or failed source no longer hides the other (FS-03.R33). Regression: two
  deferred-response cases in `Composer.test.tsx` hold one source and select the other's entry
  (both fail before, pass after). Unit GROUP-SKILL closed.

- **2026-10-31 — Fixed MMD-VIEW-1 (INV §10):** TS-08.R40 now states that the sanitizing seam
  preserves Mermaid's intrinsic SVG width cap while the stylesheet provides full-width containment,
  centering, height bound and overflow scroll — matching `renderers/mermaid.ts` and the existing
  `AssistantText.test.tsx` geometry tests. Specification only; behavior unchanged. Unit MMD-VIEW
  closed. Closure for both: `make check-specs`, `ui npm test` (1128 pass), `npm run build` and
  `git diff --check` pass; logs `/tmp/fixtriv-*.log`.

- **2026-10-31 — Fixed BGRUN-COUNT-1 (INV §5/§8):** `writeStatus` now applies the live
  context/background-task projections inside `statusMu`, so a prompt or activation busy row built
  before a task terminal update no longer restores the stale count; callers no longer pre-apply
  them. TS-04.R104 states the in-lock read and the interleaving check. Regression
  `TestBusyWriteKeepsCurrentBackgroundCount` (fails before, passes after; `-race -count=3` with
  the count test passes). Closure: `make test` (includes `make check-specs`) and `make build`
  pass, `git diff --check` passes; logs `/tmp/bgcount-make-*.log`. Unit BGRUN-COUNT closed.

- **2026-10-31 — Reviewed pipelines redesign (PIPELINE-FIGMA):** `238f2d10`, checked against
  FS-12.R71/A41 and TS-08.R149 with current-source integration at `97b32d9f`. No findings;
  originating unit closed. **Fix model:** none needed. Independent source review and 57 focused
  pipeline tests pass; style checks (47 tests plus presentation contract), `make check-specs`
  and `git diff --check` pass. Fresh `ui/scripts/pipelines-render.mjs` runs pass 192 browser checks
  across Core, Sky & Grove and Studio at 1024/1440: populated routes, long names,
  running/paused/completed/stopped ledger rows with separate state/outcome, Setup→Review,
  pagination, live refresh and empty/loading/error states. Screenshots inspected for hierarchy,
  readability, clipping and control placement are in `/private/tmp/chuck-pipeline-review/`.
  Additional independent browser probe `/private/tmp/pipeline-review-interactions.mjs` passes
  60 checks at 1024 in all three appearances: editor draft retention, long stage titles,
  add/remove, Validate/Save payloads, paused-run question/answer gating and reachable recovery
  controls. Its nine screenshots are in `/private/tmp/pipeline-review-interactions/`.
  INV sweep: §8/§10/§13/§17 inspected without findings; §1–§7/§9/§11–§12/§14–§16 have no changed
  product-contract surface in this presentation unit. Existing query/mutation ownership and
  semantic-token/CSS seams are retained; no new motion mechanism or local-choice decision.
  Browser fixtures establish presentation and navigation, not backend/provider lifecycle.
  Exact Figma target screenshot comparison remains unverified (Make screenshot/source limitation
  recorded by implementation). Only handoff state changed; concurrent ideas/runtime edits preserved.

- **2026-10-31 — Fixed GOAL-LED-11/13 (INV §8/§10/§11):** accepted queued assignment input is
  rehydrated from `GET .../prompt` `queued` and listed apart from the transcript (`QueuedInput`) on
  desktop and phone until a turn delivers it; desktop re-reads on agent state change. Desktop and
  phone run timelines show each non-implicit owner decision with its required-check evidence linked
  to the cited visit, and label decision-only room visits (`ui/src/lib/ownerDecision.ts`).
  TS-09.R71 note updated. Regressions in `Composer`, `AgentScreen`, `RunBrowser`, `RunScreen` tests
  (176 focused pass). Closure: `make check-specs` ok; full UI 1126/1126 (two earlier runs hit only
  the known CardGrid composer-label flake; HEAD UI passed once), `npm run build`, `make embed`,
  `git diff --check` pass. Logs `/tmp/goalled-*.log`. Unit GOAL-LED closed; A61/A64/A67/A68
  acceptance stays owed.

- **2026-10-31 — Reviewed background-task card count:** BGRUN-COUNT-1 records a confirmed
  stale-count overwrite when task completion overlaps a prompt/activation status write. Independent
  browser rendering, focused UI/runtime checks and state tests pass; the controlled fake-ACP
  interleaving probe fails as recorded above. The same unit remains open for its fix.

- **2026-10-31 — Reviewed group defaults and command triggers:** GROUP-SKILL-1 records coupled
  file/command availability under `@` (INV §8/§10); the unit stays open. Focused tests pass
  172/172, a controlled stalled-source probe reproduces the defect, and 29 browser checks pass.
  Evidence and invariant sweep are retained with the finding; product/spec code unchanged.

- **2026-10-31 — Fixed BGRUN-UI-1/2 (INV §8/§10):** per-task Stop state now lives in
  `RunningTasks` above the dismissible popover, so closing and reopening keeps Stopping…/disabled
  Stop, and a refusal that lands while closed shows the retryable error on reopen. New
  `BackgroundTaskList.test.tsx` case failed before the change and passes after. TS-08 is fully
  shipped and now reads Current (spec index row updated); `make check-specs` passes. UI closure:
  `npm test` 1126/1126, `npm run build` and `git diff --check` pass (logs
  `/tmp/bgrun-fix-ui-{test,build}.log`). Unit closed; concurrent GOAL-LED dirty edits preserved
  uncommitted.

- **2026-10-31 — Reviewed phone Home, wallpaper and Cancel:** `b388707f` and `1465e136`
  match FS-20.R13/R43, FS-12.R68 and TS-08.R111; no findings, unit closed, no fix model needed.
  The hidden Home heading, phone-entry-only wallpaper property and reuse of the chat mutation
  handler are sound local choices; terminal agents retain their status-only conversation.
  Independent Chromium idle/busy Home/chat checks pass all three appearances at 360/390/430px;
  `phone-render.mjs --details-check` also passes all appearances. Parent inspected Core busy
  360px, Sky & Grove Home 390px and Studio busy 430px captures. Evidence:
  `/private/tmp/review-phone-{idle,busy}-*.png`. A Chromium Cancel probe posts `target:turn`,
  surfaces a stubbed 500 refusal and measures a 44px target. API stubs establish UI behavior,
  not live cancellation, native-device or provider behavior. Focused AgentScreen/PhoneApp
  tests pass 75/75; presentation checks pass
  (47 checker tests plus contract). Tests exercise FS-derived idle/busy placement and existing
  mutation behavior; rendered checks supply CSS geometry evidence (TS-06.R35; INV §17).
  INV §§8/10/13/17 apply and pass; §§1–7/9/11–12/14–16 have no applicable changed surface.
  `git diff --check` passes; `make check-specs` still fails on the existing TS-08 Partial-status
  error recorded as BGRUN-UI-2. Other review units and concurrent dirty edits are preserved.

- **2026-10-30 — Reviewed running background-work indicator:** BGRUN-UI-1/2 record lost Stop
  state/feedback on popover dismissal (INV §8) and the newly failing TS-08 status check (INV §10).
  Independent desktop/dashboard and phone rendering plus 91 focused tests pass; browser and
  external component probes reproduce the Stop defect. Unit stays open; unrelated dirty work preserved.

- **2026-10-30 — Reviewed file-viewer Mermaid sizing:** browser geometry/appearance checks pass;
  MMD-VIEW-1 records inherited TS-08.R40 sizing drift for a small specification correction.
  `npm run check:styles` passes (47 checks plus presentation contract); 23 focused Mermaid/assistant
  renderer tests pass. `git diff --check` passes. `make check-specs` fails
  on the existing TS-08 Partial-status marker error (unshipped items lack `(planned)`), outside
  this CSS unit; no product or specification edits made. Existing unrelated dirty work preserved.

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
  [`compact-think-tank-chat-navigation`](../../ready-changes/compact-think-tank-chat-navigation.md)
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
  [`optional-annotation-source-recording`](../../ready-changes/optional-annotation-source-recording.md).
  No product code changed; existing active work remains active. `git diff --check` and twin-skill
  comparison pass; `make check-specs` remains blocked by the pre-existing DELMSG-1 line missing
  an invariant tag.

- **2026-10-27 — Running background-work count (FS-02.R83, TS-04.R104, TS-02.R59):** desktop
  agent cards show **N running** from a live status column the runtime republishes on each task
  lifecycle update; stopped/archived agents project zero. The composer popover (FS-03.R107) is
  paused on the no-output decision.

- **2026-10-26 — Designed running background-work indicator:** ready change
  [`running-background-work-indicator`](../../ready-changes/running-background-work-indicator.md)
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
[`RELEASE-v0.11.0-notes.md`](./RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Latest published notes: [v0.20.0](./RELEASE-v0.20.0-notes.md).
Earlier publication/campaign records remain in the archived handoff.
