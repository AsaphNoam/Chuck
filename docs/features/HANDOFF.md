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
- **Release runner:** `.github/workflows/release.yml` still uses `macos-14`, retired by GitHub
  2026-11-02 (official announcement/inventory verified 2026-11-06). ARCH-POST021-2 below owns
  this existing blocker; supported arm64 replacement (such as `macos-15`) is an operator decision.
- **Main CI:** prior runs 41222233023 and 41223054609 failed on CI-CARDGRID-1; repaired locally,
  awaiting the next clean-clone CI run.
- **Review state:** CI-CARDGRID and TASK-FORM are closed. The v0.21.0 architectural review
  ([receipt](../archive/reviews/architecture/v0.20.0-next.json)) is closed (ARCH-NEXT-1–4 fixed;
  generated embed covered). SELF-ANNOTATIONS is closed (all three findings repaired).
- **Next architecture audit:** combined source candidate `b1574fd9` assessed; ARCH-POST021 below
  has three Must-fix findings and one Worth-fixing finding. The
  [receipt](../archive/reviews/architecture/v0.21.0-next.json) remains incomplete pending committed
  release preparation and its independent affected-coverage evidence; no final package signoff.
- **File editing review:** FILE-EDIT-1–3 are closed: case aliases share save serialization,
  explicit file opens preserve refused edits until acknowledged leave, and stale browser pages
  cannot remove or overwrite newer durable drafts. Full closure passes; device gates below remain.
- **Recovery fix:** RECOVERED-QUEUE-1–3 are closed in `0e98d3c4`. Withdrawal acknowledges only
  durably restored text; successful Steer fences stale reads on desktop/phone. All closure checks pass.
  Fresh queue/normal delivery do not reproduce the broader report; exact attribution is undetermined.
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

None.

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

### ARCH-POST021 — combined candidate after v0.21.0 (2026-11-06) — **Fix model:** medium — Codex Terra or Claude Opus.

[Report and evidence](../archive/reviews/architecture/v0.21.0-next.md),
[receipt](../archive/reviews/architecture/v0.21.0-next.json). Reviewed committed candidate
`b1574fd979e3e5987e7b55b2a32eb21e51b72417`; unrelated dirty ideas preserved. Source audit is
finished, but TS-06.R36 signoff remains incomplete until workflow §16 preparation is committed
and independently checked. Append exact finding repairs/verification to this receipt; no repeat
of unaffected investigations is needed solely for finding fixes.

- **Must fix** — **ARCH-POST021-1:** standalone self-annotations bypass durable input recovery.
  `internal/server/sessions.go:341–369` creates request-bearing `input` only for assignment
  admission, then sends the original id-less `msg` on the standalone fallback. Runtime transitions
  ignore an empty request ID. Withdraw a held self batch and lose its response/fail browser draft
  persistence, or Stop while it is held: no released input receipt remains. Ordinary text controls
  preserve it. FS-13.R37, FS-03.R90/R100 and TS-03.R74 require the shared recovery behavior
  (INV §2/10/11/15).
  Independent HTTP/runtime/SQLite probes fail 3/3 for annotations and pass 3/3 for ordinary text;
  parent independently repeats the withdrawal failure. The source annotation event remains for
  manual copying; this is loss of managed pending-input recovery, not all instruction bytes.
  Preserve the ordinary prompt receipt through standalone annotation delivery and cover
  unacknowledged withdrawal/Stop; check uncertain Steer through the affected seam too.
- **Must fix** — **ARCH-POST021-2:** release assembly still selects a retired hosted runner at
  `.github/workflows/release.yml:17`. This normalizes the existing release-runner blocker above,
  not a newly discovered runtime defect. GitHub's
  [official announcement](https://github.blog/changelog/2026-10-01-github-actions-macos-14-runner-image-retirement/)
  and runner inventory confirm retirement on 2026-11-02; a next tag/manual release cannot obtain
  that configured image to assemble/verify the archive (TS-06.R13–R14/R21; INV §10). Select a supported
  arm64 image, retain the architecture assertion and verify packaging/toolchain assumptions.
  No hosted job was triggered; the operator image decision remains pending.
- **Must fix** — **ARCH-POST021-3:** installed operator guidance denies shipped file editing.
  `internal/agentknowledge/operating-chuck/references/operate-agents.md:10` still categorically
  says the viewer does not write files. FS-03.R111 now permits deliberate desktop source Save;
  R112 also adds browser-local draft contents as an exception to descriptor-only tab persistence.
  An ordinary product question therefore receives stale release-managed interface advice
  (FS-18.R1/R3/R5, TS-11.R1/R8, INV §10). Publication tests pass while distributing that false
  claim. Update the canonical reference precisely, retaining phone/read-only and explicit-save
  limits without implying agent-tool authority; reuse the existing publication check.
- **Worth fixing** — **ARCH-POST021-4:** a pre-save refresh can replace the saved view with older
  bytes. `ui/src/features/fileTabs/useFileFreshness.ts:67–78` checks lifecycle validity after read,
  but not whether Save applied a newer snapshot meanwhile; `FileViewer.tsx:367` applies Save and
  unblocks editing. A deferred refresh for reviewed external content then overwrites the saved
  view (FS-03.R111; TS-08.R153–R154; INV §1/5/8). Real-hook schedule probe fails independently and in the
  parent repeat. Disk remains safe and a later poll normally repairs the display. Fence read
  completion against its starting snapshot generation and retain that schedule as a regression.

## Changelog

- **2026-11-06 — Combined architectural source audit:** reviewed `v0.21.0..b1574fd9` with
  independent file-durability, input-delivery and installed-runtime reviewers; recorded
  ARCH-POST021-1–4 and inert failing probes. Specs, pipeline/knowledge packages, focused real-store
  task/pipeline/quota checks, 111 task/grid/run UI tests, 54 file UI tests and approved focused
  file/HTML/remote-route tests pass. Input reviewer also passes 125 focused recovery/annotation UI
  tests and runtime diagnostics checks. Ordinary-message controls pass while annotation recovery
  probes fail; older-refresh-over-save probe fails in reviewer and parent runs. Prior full
  `f03badd7` closure is reused for unchanged inputs. Required release preparation is absent for
  this candidate, so the receipt stays `review_complete:false`; no final package, native/provider,
  physical-device, new rendered-browser or field-performance claim. Existing owed gates and dirty
  `docs/ideas.md` preserved; no product/spec/generated files changed.

- **2026-11-06 — Review queue reconciled:** removed stale self-annotation and Think Tank setup
  entries after confirming their recorded reviews and fix closures (`36fae1fd`, `6245157b`).
  No eligible review unit remains; no new product review or fix-model recommendation is needed.
  This state-only correction has no applicable surface under INV §§1–17. Unrelated dirty
  `docs/ideas.md` preserved.

- **2026-11-06 — File-editing review fixes closed (INV §§1/5/8/9/10/15):** FILE-EDIT-1
  serializes saves by pinned parent directory identity, preserving no-follow checks while covering
  case aliases; distinct files in that directory also serialize. FILE-EDIT-2 checks/flushes unsafe
  drafts before explicit opens change selection or URL; Cancel preserves text/selection and an
  approved projection avoids a second prompt. FILE-EDIT-3 compares durable lineage/generation,
  base and text inside the IndexedDB transaction before replacement/removal; stale writes retain
  memory without claiming recovery. Original defects reproduced before repair. The retained alias
  regression fails against original code; focused `-race` and the 100-iteration review overlay pass.
  Actual editor/router regression passes; `(cd ui && node scripts/drafts-browser.mjs)`
  against loopback Vite covers stale discard/save/write and
  newer typing across two pages, both newer lineages and generations, with exact reload recovery.
  Closure: `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 GOCACHE=/private/tmp/chuck-file-fix-gocache
  make test` passes both Go suites; full UI suite 1171/1171, build/style checks, generated embed
  (`make -o ui embed`, reusing that UI build), binary build and diff checks pass. Logs:
  `/private/tmp/file-fix-{go-final,ui,ui-build,build,drafts-after,race}.log`.
  Initial Go compile overlapped embed generation; rerun passed after assets settled. Unit closed
  without adding a review unit; native/device/provider gates remain unchanged. Unrelated dirty
  `docs/ideas.md` preserved.

- **2026-11-06 — File-viewer refresh/editing review:** reviewed `bd1330aa^..617b780b`;
  recorded FILE-EDIT-1–3 and retained the unit for fixes. Focused UI suites pass 80/80;
  `make check-specs` and style/presentation checks pass. Focused backend file-viewer/remote-route
  tests including `-race` pass; an independent Go overlay reproduces the case-alias defect in
  three runs, and a temporary integration test reproduces capacity-refused draft loss. A separate
  two-page actual-product Chromium journey reproduces deletion of a newer recoverable draft.
  Independent approved-loopback Chromium on an isolated fake-ACP fixture passes live/archive/
  retained-room/Focus saves, exact disk bytes and unchanged transcript, conflict/base adoption,
  continuous writes, selection-gated refresh, reduced motion and reload recovery, plus all four
  appearances at 1024/1440px. Screenshots inspected for layout/containment, including comparison
  at the desktop floor: `/private/tmp/file-editing-review-20261106`; supplementary harness
  `/private/tmp/file-review-comparison.mjs`. Browser checks do not close native/device/provider
  gates; phone pairing remains unverified. INV §§1/2/4/5/7/8/9/10/11/13/14/15/16/17 apply;
  §§3/6/12 have no changed applicable surface. Changed tests use explicit requirement outcomes
  at component, disk, registered-route and real IndexedDB/browser boundaries; no separate
  TS-06.R35/INV §17 finding. Product/spec/generated files and unrelated dirty `docs/ideas.md`
  are preserved; other review units remain unchanged.

- **2026-11-06 — File-viewer refresh and editing:** finished the approved change, including
  conditional host saves, recoverable drafts and selected-visible quiet refresh. Closure passes;
  review unit added without replacing existing queues. Unrelated `docs/ideas.md` remains dirty.

- **2026-11-06 — Self-annotation fixes (INV §§1/2/8/10/11/15):** SELF-ANNOTATIONS-1–3
  and their originating review unit closed by `242860c1`. Self batches share ordinary assignment
  admission and paused-run continuation; stable input receipts reconcile interrupted acceptance
  without duplicating input. Desktop and phone refresh pending input even after receipt replay.
  A delivered annotation clears only the matching held text, and Steer reports running-turn versus
  new-turn delivery outside the cleared tray. Regression evidence includes pre-fix failures for
  assignment admission, unrelated-held-batch clearing and missing desktop feedback; waiting-task
  startup recovery/retry, paused-run replay, phone feedback and queued-input display pass.
  Checked repair scope `242860c1`: `env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1
  GOCACHE=/private/tmp/chuck-go-cache make test` (both full Go suites, loopback-capable execution),
  `make build`, full UI Vitest (1,158 tests), UI style/presentation checks and build, and
  `git diff --check` pass. Logs: `/tmp/self-annotation-go-closure.log`,
  `/tmp/self-annotation-ui-closure.log`, `/tmp/self-annotation-ui-build.log`.
  Existing native/provider/device gates remain owed. Unrelated `docs/ideas.md` and embedded
  `internal/server/ui/dist/index.html` edits were preserved; embedded UI was not regenerated.

- **2026-11-05 — Think Tank setup layout fixes (INV §§8/13):** TT-SETUP-1/2 closed by
  separating participant identity from controls and constraining the empty picker to its grid
  track. Real-browser renders covered empty candidates and short/long populated names at 1024px
  and 1440px in Core, Sky & Grove, Studio and Boring Night; all 16 had no horizontal overflow,
  and name/model, limit, May leave, ordering and Remove remained distinct. Screenshots:
  `/var/folders/87/yfpx_jjx4rd6yn0965lj9j680000gn/T/tt-setup-render/`. `make check-specs`, full
  UI Vitest (1,150 tests), UI style/presentation checks and build passed; `git diff --check` passed.

- **2026-11-05 — File-viewer refresh/editing design ready:**
  [ready change](../ready-changes/file-viewer-refresh-and-editing.md) waits to start.
  FS-03.R110–R112/A86–A88, TS-03.R80–R81, TS-05.R29 and TS-08.R153–R155 specify settled
  refresh with a small status spinner over retained readable content, desktop source editing and
  browser-local draft/conflict recovery. Phone stays read-only. Operator approved optimistic
  external-writer conflict detection and existing owned writable regular files, refusing symlinks,
  hard links and unsafe metadata replacement. Documentation/spec and twin-skill checks passed;
  no implementation or product validation is claimed. Active change remains unchanged.

- **2026-11-05 — Recovery fix closed (INV §§1/2/5/8/10/11/15/17):** RECOVERED-QUEUE-1–3
  repaired in `0e98d3c4`; this closes the originating review unit without adding a review unit.
  Desktop/phone acknowledge only the exact withdrawn text receipt after durable draft restoration;
  storage/acknowledgement failure retains recovery. Successful Steer clears matching recovery,
  refreshes its snapshot and fences older reads, preserving unrelated recovery and attachments.
  Bounded receipt-transition diagnostics identify queue/replacement/withdrawal/Steer/provider-error
  paths without message bodies, attachment metadata/content or raw provider errors. FS-03.R100/A70/A79,
  TS-03.R74 and TS-02.R55 clarify ownership and diagnostics. Both original skipped UI reproductions
  failed before repair and now pass. Real HTTP/store coverage verifies released-text acknowledgement
  preserves older uncertainty; controlled desktop/phone reads cover late snapshots and failed refresh.
  Focused diagnostics/provider-error/lost-ack checks and diagnostics `-race` pass. Final UI suite
  passes 1,150/1,150; production UI build (style/presentation/type checks), `make test` (untagged
  and SQLite FTS5), `make build`, and diff checks pass for the repair's source scope.
  Logs: `/private/tmp/recovered-{make-test-approved,make-build-approved,ui-closure-test,ui-closure-build}.log`
  and `/private/tmp/recovered-{diagnostics-test,diagnostics-race,withdrawal-test}.log`.
  The sandbox denied loopback for the first Go run; approved execution passes. Generated embedded UI
  and unrelated ideas edits remain untouched. No live-provider/device receipt is claimed, and the
  broader every-queue report remains unattributed. SELF-ANNOTATIONS-1–3 and TT-SETUP-1–2 remain open.

- **2026-11-05 — Think Tank setup / annotation notice review:** reviewed `e2930630` and
  recorded TT-SETUP-1–2; the unit remains open. Focused setup/AnnotationTray/ReaderWorkspace/
  phone AnnotationDelivery tests pass 40/40; spec integrity and style/presentation checks pass.
  Independent approved loopback Chromium rendered the actual project setup dialog in all four
  appearances at 959/960/1024/1440px with 0/2/8 participants. Adding, scrolling to actions and
  Cancel work; rendered overlap and initial horizontal overflow fail the fit claim. Prior-layout
  CSS comparison confirms the populated-row regression. Browser fixtures stub API/SSE; no
  provider or physical-phone receipt is claimed. Evidence and harness:
  `/private/tmp/think-setup-review/` and `/private/tmp/think-setup-review.mjs`.
  The existing development presentation matrix also renders in Core, Sky & Grove and Studio
  at 1024/1440px (`/private/tmp/think-setup-matrix/`).
  INV §§8/10/13/17 apply; §§1–7/9/11/12/14–16 have no changed applicable surface.
  Changed delivery tests retain real component/request assertions; no TS-06.R35/INV §17 finding.
  The removal of successful-send notices matches FS-13.R29/R35; exact CSS dimensions are
  presentation choices under existing layout requirements, not a separate specification defect.
  Product/spec files and concurrent recovery, idea and generated-embed edits are preserved.

- **2026-11-05 — Self-targeted annotation review:** reviewed `17697763`; recorded
  SELF-ANNOTATIONS-1–3 above and left that unit open. Existing self-delivery handler tests pass
  (4/4) with approved loopback execution; the temporary waiting-assignment regression fails on the
  reproduced `sent` versus `queued` behavior. Overlay files:
  `/private/tmp/self-annotation-review-overlay{.json,_test.go}`; repository tests were not edited.
  Focused AnnotationTray/TranscriptView tests pass 52/52, and independent
  ChatPanel/TranscriptView/transcriptStore tests pass 106/106 (overlapping suites).
  `make check-specs` and style/presentation checks pass. INV §§1/2/5/8/10/11/13/14/15/16/17
  apply; §§3/4/6/7/9/12 have no changed applicable surface. Changed tests observe real handler,
  held-message and transcript boundaries against explicit outcomes; no separate TS-06.R35/INV §17
  finding. The existing-button extension introduces no material visual redesign; no rendered or
  live-provider receipt is claimed. Concurrent recovery-fix work, idea edits, generated UI and the
  other review unit are preserved.

- **2026-11-05 — Card-grid fix (INV §§1/17):** CI-CARDGRID-1 closed. A deferred layout read
  reproduced a real transient auto-open before saved collapsed-group preferences arrived.
  Automatic opening now waits for layout settlement (success or reported failure), records
  transitions during loading without replay, and retains the normal later-transition behavior.
  FS-02.R61/A43 and TS-08.R49 clarify this boundary. The original exclusion test waits for its
  own collapsed layout; the deferred regression fails before the repair and checks both the
  transient DOM and the next saved expanded list. CardGrid tests pass 42/42; full UI tests pass
  1,142 with two existing skips; `make check-specs`, production UI build including style and
  presentation checks, and `git diff --check` pass. Logs: `/tmp/chuck-cardgrid-{specs,ui-tests,build}.log`.
  Clean-clone CI has not been rerun. RECOVERED-QUEUE remains open; unrelated ideas/generated
  embed edits are preserved. This fix closes its originating review unit.

- **2026-11-05 — Pipeline stage-work links review closed without findings:** reviewed
  `43821517` against FS-14.R39/A17/A23. Detail-response live/archive routes work before global
  hydration; live state overrides the route in place; missing identity remains non-linking.
  Existing card selectors support the restored links without a material visual redesign.
  Spec integrity passes. Focused RunBrowser/RunWorkspace UI tests pass 23/23 (the workspace suite emits existing
  unhandled `/api/roles` fixture diagnostics). The added regression observes DOM destinations
  and a real store update against explicit requirement outcomes; no TS-06.R35/INV §17 finding.
  Invariant sweep: §§1/2/8/10/11/13/17 apply (live updates, existing routing seams, fallback,
  navigation wiring, route vocabulary, selectors and test oracle); no violation found.
  §§3/4/5/6/7/9/12/14/15/16 have no changed applicable surface.
  **Fix model:** not applicable — no open fixes. Product/spec files, unrelated dirty
  ideas/generated index, and other review units preserved.

- **2026-11-05 — HTML preview review closed without findings:** reviewed `d2a7caad` and
  administrative follow-up `be5505f8` against FS-03.R109/A85, TS-08.R152 and TS-03.R79.
  Independent Chromium journey passes 20/20; extended sweep passes 44/44 across all four
  appearances at 1024/1440px, including docked/Focus menus and live/archive/retained-room
  dispatch. Captures inspected for containment, controls, contrast and menu stacking; reports
  and screenshots: `/private/tmp/html-review-20261105` and
  `/private/tmp/html-review-appearances-20261105`; extended harness:
  `/private/tmp/html-review-extra.mjs`. Focused UI tests pass 89/89; focused Go browser route
  and Darwin argv tests, spec integrity, style/presentation checks and diff checks pass.
  API stubs establish frontend dispatch, while real-handler tests establish backend routing;
  neither closes the native default-browser gate above. Confirmed the local navigation choice:
  the initial CSP restricts loads, while native self-navigation remains sandboxed as specified.
  Legacy-encoded HTML remains refused by the existing UTF-8 file-read authority
  (FS-03.R64/A48, TS-03.R40); this review does not infer new encoding support from R79.
  Invariant sweep: §§1/2/4/8/9/10/11/12/13/14/16/17 apply (reload/menu state, shared helpers,
  listener cleanup, errors, timeouts, caller wiring, messages, native argv, selectors, local
  guard, bounded reads and test oracles); no violation found. §§3/5/6/7/15 have no changed
  applicable surface. Tests observe requirement outcomes at component, HTTP, argv and real
  browser boundaries; no TS-06.R35/INV §17 finding. **Fix model:** not applicable — no open fixes.
  Product/spec files and unrelated dirty ideas/generated index preserved; other queues unchanged.

- **2026-11-05 — Task-form fix (INV §8):** TASK-FORM-1 closed; typed prerequisite/context
  refusals reopen Advanced and retain the draft. The existing rendered request-boundary regression
  now collapses Advanced before an actual-shaped `unusable_source` refusal and checks correction
  fields and draft retention. No behavior beyond FS-16.R40/R45/R59 was added.
  Regression failed before the repair and passes afterward; focused Tasks tests pass 46/46,
  full UI suite passes 1,141 tests (two existing skips), `make check-specs`, UI production build
  (including style/presentation checks), and `git diff --check` pass.
  Closure logs: `/tmp/chuck-task-form-{specs,ui-tests,build}.log`. Unrelated ideas/generated
  embed edits remain preserved. CI-CARDGRID and RECOVERED-QUEUE remain open.

- **2026-11-05 — Quiet task coordination review closed without findings:** reviewed `a674e6bc`
  against FS-16.R49, FS-14.R75–R78 and TS-09.R49, including shared launch/activation guidance,
  dedicated coordinator instructions and bundled references. These reviewed inputs are unchanged
  through `be5505f8`. Invariant sweep: §§2/10 apply (shared composition and documentation/wiring);
  §§1/3–9/11–17 have no changed applicable surface. No tests were added or materially changed;
  existing fake-provider prompt-delivery checks establish transport, not live model adherence.
  Independent focused launch/running/stopped assignment, dedicated-coordinator and assignment-render
  tests pass for `internal/server` and `internal/pipeline` with loopback-capable execution;
  `make check-specs` passes. Logs: `/private/tmp/chuck-quiet-review-{focused,specs}.log`.
  Initial sandbox run could not bind the test HTTP listener; approved rerun passed.
  **Fix model:** none — no open fixes. Live provider adherence remains unverified. Product/spec
  files and unrelated dirty idea/generated UI edits preserved; other review units unchanged.

- **2026-11-05 — HTML preview:** self-contained HTML now renders inside existing file tabs,
  with Source and explicit default-browser context actions. Automated closure passes; native
  browser launch remains unverified in this environment (no configured browser).

- **2026-11-05 — Project chat search feature draft:** operator confirmed non-archived first,
  running before stopped, and opening at the displayed excerpt; requested an established package.
  FS-05.R46–R50/A28–A30 record the planned behavior. Bleve's official APIs support fuzzy/prefix/
  phrase queries and match offsets; the added local-index tradeoff and matching policy await
  confirmation before technical design. No ready change or product code; Active change stays None.

- **2026-11-05 — Project chat search design started:** recorded the requested project-scoped
  search in `ideas.md` under Ideas being defined; checked existing Archive full-text behavior.
  Product scope and the meaning of active remain pending operator confirmation; no technical
  design, ready change or product code. Active change remains None.

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
