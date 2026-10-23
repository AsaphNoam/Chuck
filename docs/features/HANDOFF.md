# Chuck — Implementation handoff

**Live agent state.** Settled state through 2026-10-18 is archived in
[`HANDOFF-through-2026-10-18`](../archive/state/HANDOFF-through-2026-10-18.md).
Follow [`AGENT-WORKFLOW.md`](AGENT-WORKFLOW.md).

## Current position

- **Release:** [v0.18.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.18.0) published
  2026-10-21 from `722282dd`; release CI 39542860225 passed, and the archive, installer and
  checksum manifest are attached. Notes and the one-release operator waiver are in
  [RELEASE-v0.18.0-notes.md](../archive/state/RELEASE-v0.18.0-notes.md).
  Main CI 39542863948 also passed on the tagged source.
- **Operator authorization (2026-10-20):** release and push with pending fix items; skip the
  architecture review for v0.18.0 only. No review receipt or finding closure is claimed.
  The operator also confirmed v0.18.0. All open findings, reviews and manual gates below remain live.
- **Repository:** `AsaphNoam/Chuck`; branch `main`. This release excludes and preserves the
  operator checkout's uncommitted ready-change edits. The concurrent runtime fix is now committed
  and included in the candidate.
- **Runtime campaign:** ASW-1 event commits, ASW-2 frozen native execution and ASW-3 process/terminal
  ownership, ASF-1 turn/control settlement, ASF-2 input acceptance/recovery and ASF-6 CLI selection
  are closed. Select remaining work through
  [AGENT-SURFACE-FIX-UNITS.md](AGENT-SURFACE-FIX-UNITS.md); ASF-3/4/5 are closed.
Their source reviews and open findings remain below; do not reopen completed refactors.
- **Core audits:** CR-A1–CR-A7 are reviewed and closed (no open findings). CR-A2's nine
  admission/recovery findings were fixed on 2026-10-22 (`0f911d74`, `de0ac20a`).
  [CORE-RELIABILITY-AUDIT-PLAN.md](CORE-RELIABILITY-AUDIT-PLAN.md) owns the charters.
  [Before-fix synthesis](CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md) connects preservation,
  task ownership/capacity and observable-state recovery; it adds open run-detail recovery to
  CR-A7-1's acceptance scope (fixed with CR-A7-1).
  Its [refactor recommendation](CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md#architectural-refactor-recommendation)
  now reflects an independent Astra challenge and primary source checks: design a narrow reindex
  preservation boundary; specify task cleanup/continuation contracts within coordinated fixes.
  A separate task-ownership architecture project is not demonstrated. CR-A4 is closed below.
  **Reindex preservation implemented:** FS-05.R43–R45/A26–A27 and TS-02.R56–R58 now use
  per-conversation atomic replay under the approved partial-repair policy. CR-A1-1/2 were
  closed by independent review on 2026-10-22; CR-A1-3 was fixed the same day.
  Workflow reliability skills/gate passed independent implementation review on 2026-10-21;
  no findings remain for that unit. This does not supply a release architectural receipt.
- **Other state:** direct-action migration remains paused on its transport blocker; passive
  subscription quota snapshots remain TBD, not ready work. PROMPT-1/2 and META-1 are closed.
- **Source checks:** unset `CHUCK_RUNTIME_ROOT`; prefer `GOFLAGS=-p=1` for full Go closure.
  Known intermittent tests include bootstrap download contention, ordinary stage Stop/resume,
  quota dispatcher transcript reload and the file reader cited-line scroll. Preserve failures
  as evidence; do not treat isolated reruns as live-provider proof.
- **History:** settled implementation/planning details through this release are in
  [HANDOFF-through-2026-10-20](../archive/state/HANDOFF-through-2026-10-20.md);
  earlier published history remains in the 2026-10-18 archive.

## Active change

- **2026-10-23 — ARCH-1 repair, then release requested:** the offline PID inspection guard now
  refuses read/decode failures before opening SQLite. Malformed/unreadable command regressions
  fail before repair and pass afterward; full clean-candidate release checks and receipt closure
  are next. Proposed patch `v0.18.1`; version and publication confirmation remain pending.
  Uncommitted task-scheduling design edits are preserved and excluded; release verification
  will use a clean committed checkout. The post-review runtime guard commit `6050f2ce` needs
  focused affected-coverage evidence in the existing receipt, not another architectural review.
  Clean verification checkout: `/private/tmp/chuck-release-v0.18.1`, source `69082b39`.
  Distribution build, unchanged generated UI, 995 UI tests, runtime guard race test and vet pass.
  The first no-tag CLI package hit the known bootstrap download-start
  deadline, and three focused repeats passed twice/failed once. Logs: `/private/tmp/chuck-v0.18.1-*`.
  The same no-tag run also failed `TestRestartResolvesUnfinishedTasksFromTheirOwnRows`'s
  immediate-release assertion: the fixture retained a live busy reporting turn, so R19/R48
  correctly deferred cleanup. No test/product edit made. The full FTS5 variant passes; the full
  no-tag retry's server suite passes, but bootstrap readiness fails again. REL-VERIFY-1 remains
  open pending test-only approval and verified closure. No tag/publish. Exact focused repair/
  affected coverage is recorded in the existing receipt; ARCH-1 administrative closure is deferred.

- **Next-release architectural review completed, 2026-10-23:** committed candidate `eb56b0c2`
  since v0.18.0; [receipt](../archive/reviews/architecture/v0.18.0-next.json) and
  [coverage/evidence](../archive/reviews/architecture/v0.18.0-next.md). No Must-fix finding;
  ARCH-1 below is Worth fixing. Existing native/device/terminal gates remain owed. Concurrent
  uncommitted runtime/handoff work was excluded and preserved. Later substantive release
  preparation/work needs exact affected coverage; finding fixes use recorded closure without
  restarting the review.

- ASR-G is closed: ASRG-1 in `63a723e5`, ASRG-2/5 in `8b7db7ed` repair quota room-start
  compensation, eligibility before check allowance and fair bounded cleanup. Both full Go
  variants, tagged build, focused tagged quota/state race checks and documentation/launcher
  checks pass. Original-code overlays fail the room, capacity/lifecycle and ninth-cleanup
  regressions. ASRE-1 closed in `5091d0d3` (ASR-E closed); ASRF-1/2 are closed in
  `473d74dd` with durable controller recovery and task-owned pipeline dispatch evidence.

- ASF-4 is closed: task result outputs are bounded before commit (`91f30979`, ASRE-2) and
  ended native children/finished tool names are freed (`65de12ce`, ASRC-6). ASR-C is closed;
  `TestCanceledTurnsReleaseToolInput` fails under `-race` on
  unmodified HEAD too (turn end publishes before its inputs are released); not addressed here.
- ASF-5 is closed: resource owner-permission preservation (`8db628c7`) and owned-checkout
  setup readiness (`c379b51c`) are repaired. Both full Go variants, tagged build and focused
  config/server race checks pass. No native/device acceptance gate is closed by these repairs.
- CR-A2 is closed: all nine findings are fixed, both complete Go variants and the tagged
  build pass. TS-10.R48–R52 record cleanup, continuation, bounded queue and recovery contracts.
  The four unrelated ready-change edits remain preserved. ASF-6 closed concurrently in its
  own commits.
- ASF-2 is finished in `/private/tmp/chuck-fix-asf-2`, branch `fix/asf-2`, based on
  `31a06a0f`; implementation commits are `2268e279`, `f8338ac5` and `64bd3962`.
  Now integrated with the completed CR-A2 and ASF-6 changes on main. Both Go variants, 995 UI tests, production
  UI/embed/binary builds, focused race checks and desktop Chromium recovery journeys pass.
  Native-provider and physical-phone receipts remain explicitly owed below.
- Previously: interruption-safe reindex is finished and independently reviewed without findings.
  Both Go variants and the build pass. The operator-requested worktree kept the main checkout's
  existing uncommitted ready-change edits separate.
- Other available review/fix units and owed live receipts remain below.

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

## Agent-surface work and fix queue

**All ASW and ASF units are closed.** The complete [unit designs and primary finding map](AGENT-SURFACE-FIX-UNITS.md)
are the selection authority for this campaign. Three substantive refactors are implemented and
reviewed; their remaining findings select the same units for fixes:

| Unit | Invocation | Outcome |
|---|---|---|
| ASW-1 | closed 2026-10-20 — nine findings fixed (Changelog) | One canonical commit boundary, ordered control observations and truthful history failure |
| ASW-2 | closed 2026-10-20 — nine findings fixed, ASW2-1/5 resolved by human decision (Changelog) | Frozen effective native selection and explicit fresh/resume/fork/check preparation |
| ASW-3 | closed 2026-10-20 — six findings fixed (Changelog) | Process/terminal ownership and native terminal identity |

The following cross-review §8 batches are explicitly authorized by the human's consolidation
request. Their exact scope, required specs, regressions and closure rules are in the linked design.
Keep original finding IDs and evidence until each closes; do not close an originating review just
because one new batch finished. Original review aliases must respect this map and must not
reimplement an ASW refactor as a local fix. The existing launcher/workflow remain otherwise unchanged.

| Fix unit | Invocation / responsibility | Owned findings | Status / fix model |
|---|---|---|---|
| [ASF-1](AGENT-SURFACE-FIX-UNITS.md#asf-1--turn-and-control-settlement) | `/fix ASF-1` — settings, status, permission and cancellation settlement | ASRC-1, ASRC-2, ASRC-5, ASRG-3 | Closed 2026-10-20 — `66c6a984` (ASRC-1), `60777c14` (ASRG-3), `ad08b386` (ASRC-2/5) |
| [ASF-2](AGENT-SURFACE-FIX-UNITS.md#asf-2--input-acceptance-and-recovery) | `/fix ASF-2` — annotation receipts, drafts, exact withdraw and uncertain Steer | ASRD-4, ASRH-1, ASRH-2, ASRH-4 | Closed 2026-10-22 — `2268e279`, `f8338ac5`, `64bd3962`; native/device receipts remain owed |
| [ASF-3](AGENT-SURFACE-FIX-UNITS.md#asf-3--durable-continuation-and-controller-recovery) | `/fix ASF-3` — task/pipeline/quota recovery and obsolete executor removal | ASRE-1, ASRF-1, ASRF-2, ASRG-1, ASRG-2, ASRG-5 | Closed 2026-10-23 — `5091d0d3` (ASRE-1), `473d74dd` (ASRF-1/2), `63a723e5` (ASRG-1), `8b7db7ed` (ASRG-2/5) |
| [ASF-4](AGENT-SURFACE-FIX-UNITS.md#asf-4--retention-and-result-bounds) | `/fix ASF-4` — native-child retention and task-result bounds | ASRC-6, ASRE-2 | Closed 2026-10-22 — `91f30979` (ASRE-2), `65de12ce` (ASRC-6) |
| [ASF-5](AGENT-SURFACE-FIX-UNITS.md#asf-5--project-readiness-and-resource-permissions) | Checkout readiness and restrictive resource modes | ASRA-5, ASRA-6 | Closed — `8db628c7`, `c379b51c` |
| [ASF-6](AGENT-SURFACE-FIX-UNITS.md#asf-6--cli-archive-selection) | Certain CLI resume/new selection | ASRH-3 | Finished |

## Review queue

**Core reliability audit units:** explicitly requested subsystem reviews of the complete current
implementation and callers, not reviews of planning commits. Read the selected charter, refresh
current fix ownership and record the reviewed HEAD. CR-A1–CR-A7 are reviewed and closed.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A1](CORE-RELIABILITY-REVIEW-A1.md) | Durable truth and recovery: conversation/session authority through Stop/restart, archive/Restore, interrupted reindex and native continuation | reviewed `da44be64` — closed; A1-1/2 closed by reindex review, A1-3 fixed |
| [CR-A2](CORE-RELIABILITY-REVIEW-A2.md) | Admission, waiting and recovery: task/dependency eligibility, assignment/capacity, mail/quota retry policies, retained cleanup and startup progress | reviewed `f1c59961` — closed; CR-A2-1–9 fixed |
| [CR-A3](CORE-RELIABILITY-REVIEW-A3.md) | Executable workspace/configuration: checkout/setup readiness, shared resources, fresh/frozen selection and actual native cwd/environment/setup | reviewed `7ec20a01` — closed; CR-A3-1/2/3 fixed |
| [CR-A4](CORE-RELIABILITY-REVIEW-A4.md) | Information/operating knowledge: selected content and source identity, authorized reads, mail receipts, managed guidance and actual native delivery | reviewed `7e0e1a0a` — closed; CR-A4-1 fixed |
| [CR-A5](CORE-RELIABILITY-REVIEW-A5.md) | Supervision/result authority: accepted reports, matching execution settlement, descendant closure, judge publication, room-stage acceptance and retained progression | reviewed `785e3f4e` — closed; CR-A5-1/2 fixed |
| [CR-A6](CORE-RELIABILITY-REVIEW-A6.md) | Operator action/observation truth: exact targets/guards, acceptance versus completion, partial/unknown effects, CLI output and current-state recovery | reviewed `3334471d` — closed; CR-A6-1–6 fixed |
| [CR-A7](CORE-RELIABILITY-REVIEW-A7.md) | Sustained capacity/cost: retrieval scan work, accumulated data/views, shared pressure, temporary resource lifetime and unnecessary native work | reviewed `8c837942` — closed; CR-A7-1–5 fixed |

**Reindex implementation review closed, 2026-10-22:** no findings in the interruption-safe
reindex product changes `90caf6b4` / `467b70ba` within `7b77dd25..467b70ba`, checked at
`0ac1c55c`. CR-A1-1/2 are closed (FS-05.R43–R45/A26–A27; TS-02.R56–R58).
Process-kill/reopen/retry and fake-native Resume wire proofs use separate fixtures; no
same-fixture end-to-end process-kill/Resume claim is made. This state-only closure adds no
review obligation. CR-A2's fix closed its cleanup/accounting and continuation-delivery contracts
within the existing seams (TS-10.R48–R52); ASF-3 closed its own findings separately.

The [audit plan](CORE-RELIABILITY-AUDIT-PLAN.md) records all seven charters and the separate proposed
cross-audit integration proof. These subsystem audits do not substitute for the candidate-bound
architectural review before release or start any fix workflow.

**Original agent-surface campaign:**

Nine integrated reviews are sized for frontier reviewers to follow interacting mechanisms and
whole workflows. These are explicitly requested subsystem audits, not reviews of queue commits.
ASR-I's source synthesis is complete; the remaining domain units below retain their open findings.
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

No review rows remain; ASR-A–I are reviewed with no open findings. Owed live receipts stay below.

## Review findings

The `NewAgentModal`/phone gap-test race fix is confirmed remotely (main CI run 39007164597).

Closed reviews through the `v0.17.0` cut are in the archive above.

### v0.18.1 release verification — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

- **Must fix** — **REL-VERIFY-1: release closure is blocked by pre-existing fixture assumptions** (INV §17).
  `internal/cli/installer_test.go:334–343` allows only one second for the bootstrap subprocess
  to reach its download marker; it failed in the initial and retry no-tag suites, while focused
  repeats passed twice/failed once and the tagged CLI suite passed. The installer and test are
  unchanged since v0.18.0. `internal/server/task_dispatch_test.go:952–955` additionally asserted
  immediate release after recovery while the same live reporting turn still owned the runtime;
  TS-10.R19/R48 correctly defer that cleanup. The test never simulates losing the registry.
  Proposed bounded test-only corrections keep the lock contention/final release assertions:
  wait up to 10 seconds for bootstrap readiness, and drive/wait for ordinary deferred cleanup.
  No product regression or repair is claimed. Approval requested under the operator's separate-fix
  instruction; reviewable patch `/private/tmp/chuck-release-test-proposal.patch`. Keep release
  blocked until checks pass. Logs are `/private/tmp/chuck-v0.18.1-{go-closure,go-retry,bootstrap-diagnostic}.log`.

### Next-release architecture — reviewed `eb56b0c2` — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

[Candidate receipt](../archive/reviews/architecture/v0.18.0-next.json) and
[scenario/evidence report](../archive/reviews/architecture/v0.18.0-next.md).

- **Worth fixing** — **ARCH-1: fail closed when repair cannot inspect the dashboard PID file.**
  `internal/cli/reindex.go:14–19` maps corrupt/unreadable PID files to “not running,” so offline
  repair opens a second writer while a dashboard can still be healthy (TS-02.R2/R58; INV §7/§9).
  A disposable real-dashboard process correctly refused reindex with its healthy PID file;
  malformed JSON and mode `000` each changed the command to exit 0/`reindexed archive` while
  `/api/health` still returned 200. This requires a PID-file/filesystem fault; the empty test home
  demonstrated no lost data. Propagate a read/decode error and refuse before opening the repair
  writer, preserving absent/stale-PID behavior. Add focused command-level corruption and
  unreadable-file coverage. No new lock or stronger startup-exclusion guarantee is requested.

### ASR-A — Instructions, configuration and execution environment — reviewed `4fbeb316` — **Fix model:** difficult — Codex Sol.

Source/profile delivery must agree with native authentication,
session isolation and frozen continuation choices; the other fixes mostly extend existing seams.
Audit and finding fixes are closed. The original review changed no product code or specifications.
Source mode: repository source plus installed, lockfile-pinned packages: Claude ACP 0.85.1,
Claude Agent SDK 0.3.286, Codex ACP 2.1.1 and Codex CLI 0.159.3. Tree was clean at entry.

ASRA-1/2 are verified closed by the ASW-2 review: new contracts preserve source model overrides
and native inheritance through continuation, and empty effort resets use the declared default.

PROMPT-1 and ASRA-3 are closed by the ASW-2 fixes (ASW2-7 effective root markers, ASW2-2 frozen
profile refusal; Changelog 2026-10-20).
ASRA-5/6 are fixed (2026-10-22): `8db628c7` preserves existing owner permissions while removing
group/world access (INV §10/§14); `c379b51c` shares the per-project readiness claim through setup
completion, warning propagation and waiter cancellation (INV §1/§5/§15). The `0500` regression
failed before repair. The isolated real-Git overlay probe fails on the original directory-exists
bypass and passes after repair; retained tests cover post-directory arrival, setup warning and
cancelled waiters. Both complete Go variants, tagged build and focused race checks pass.
All ASR-A findings are closed; unrun native-provider gates remain open.

Integration choice: failed Git recreation before a directory exists retains the existing retry
behavior for the next caller. The barrier shares completed setup readiness, not a durable failed
Git attempt; no scheduler or new retry policy was introduced.

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

Audit complete; no findings remain open. ASW-3's review closed ASRB-4–7 and its fixes closed the
ASRB-1 remainder (2026-10-20, Changelog). This section retains the evidence ledger and the owed
live receipts named below.
Source mode: repository source plus locally installed lockfile packages, Claude ACP
0.85.1 / Agent SDK 0.3.286 and Codex ACP 2.1.1 / CLI 0.159.3. Assembly pins Node 22.22.0 and
Codex's patched component `2.1.1+chuck.1`; local test Node is 22.23.1. No native model turn ran.
The pre-existing dirty ASR-C row and its concurrent administrative closure were preserved;
`07f201a3` changes review state, not the inspected implementation. Three bounded read-only audits
covered Codex selection/profile, Claude negotiation and terminal drivers; findings were re-read
and integrated here.

ASRB-2/3/8 are verified closed as original reproductions by the ASW-2 review: Claude opens its
returned fork before settings/prompt with exact-id delete on failure, Codex omits the new prompt
overlay on fork, and unproven source/PATH Codex Steer is withheld while ordinary Send remains
usable. ASW2-6 fixed clone permissions; ASW2-5's Claude fork layers were kept by human decision
(TS-04.R65/R101 amended 2026-10-20). Credentialed
clone/adoption and unpatched idle-Steer race receipts remain owed.

Provider/path realization and evidence ledger:

| Catalogue | Traced paths and result |
|---|---|
| **03 — selection/package/auth/catalog** | Shared resolver covers fresh/wake/resume/switch/rollback/task/pipeline/clone, direct Claude terminal, version/login/readiness. Installed overrides/discovery and explicit Bundle stay separate; no silent fallback established. Private Node/adapters/native paths, exact pins, patch hashes, single Codex package and manifest/wrapper agreement inspected. Resolver, probe, auth-target, refresh and release-layout fixtures support selection, not credentials or real packaging acceptance. ASW-2 now withholds unproven source Steer. |
| **04 — Codex native contract** | Private-home refresh and reserved environment across start/load; overlay to native config, ordered model→effort→fast, Steer, fork and capability/source contracts traced into pinned source. ASW-2 corrects the original fork overlay and source Steer defects; ASW2-7/2 closed ASR-A's PROMPT-1/custom-root remainders. Native account/home/model adoption, handoff/compaction and packaged optional-feature receipts remain owed. |
| **05 — Claude native contract** | New/load share preset-plus-append metadata, MCP/directories, model metadata and ordered configuration; provider model refusal fails without bare/default retry. Advertisements/readback remain adapter evidence. ASW-2 now loads the returned fork id with the source's resume instructions (decided) and keeps source permissions. Terminal flags are composed; native-preset snapshots, modes, effective settings and terminal flag/hook adoption remain live gates. |
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

Status settlement coordinates runtime/server state and other fixes extend existing seams. ASW-3's
review closed ASRC-4/7 and its fixes closed ASRC-3; ASF-1 closed ASRC-1/2/5 (2026-10-20, Changelog); ASF-4 closed ASRC-6 (`65de12ce`, 2026-10-22).
All ASR-C findings are closed; this ledger is retained provenance. The original review changed no product code
or specifications changed. Source mode: repository source; real-adapter shapes read from the
release lockfile packages (Claude ACP 0.85.1; Codex ACP 2.1.1 is a native binary, not inspected).
Bounded discovery for catalogue 06/11/12 was delegated and every reported finding re-read in code.

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

ASRD-4 closed with ASF-2 on 2026-10-22; no ASR-D findings remain. ASRD-1/2/5's original reproductions are corrected
and independently reviewed in ASW-1; ASRD-3's history suppression and its remaining
recovery/control obligations closed with the ASW-1 fixes (Changelog, 2026-10-20). The retained audit evidence below
describes the original snapshot. No product code or specifications changed during this audit.
Tree clean at entry. Source mode: repository source at
`c12c425109a7bdd5682451f02afe958477319e7a`; release lock pins Claude ACP 0.85.1 / Agent SDK
0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. No credentialed provider run was performed.

**Closed ASRD-4 — `2268e279` / `f8338ac5` (INV §2/§7/§11/§15/§16).** Annotation commands now
persist exact source/target/hash, canonical sequence and effect identity before delivery. Peer mail
and its receipt commit together; self delivery claims uncertainty before native submission.
Identical retries reuse the outcome; changed content conflicts, and canonical read failures refuse
rather than append again. Desktop/phone persist the command, target and launched recipient through
remount/reload. Server regressions cover peer/self replay, concurrency, canonical reconciliation and
read failure; annotation store/delivery tests cover stable IDs and recipient recovery. Room command
authority is unchanged. Retained regression coverage is in
`internal/server/annotations_test.go` and the shared annotation UI tests.

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

**Closed.** ASF-4 closed ASRE-2 (`91f30979`, 2026-10-22). ASRE-1 (INV §1/§10/§11/§15) is
repaired in `5091d0d3` (2026-10-23): confirmation no longer deletes the wait set. It stays the
resumed execution's continuation context, at most 64 rows, until the next wait or a terminal
result. `get_assigned_task` returns `observed_changes` with each watched source's id, revisions,
state, outcome and summary. A deleted source shows `deleted` with its retained outcome. Waits
re-wake only waiting or yield-pending work (TS-10.R53, §3). State and MCP regressions cover early
and late reads, multiple and deleted sources, store reopen, no re-wake and caller scoping. The MCP
probe failed before the repair. The view is live: `after_revision`/`changed` let the agent see a
later retry. `continuation_reason` is not projected. No native-provider receipt is claimed. The
ledger below records the original audit.
Historical source mode: current repository source at `5eb574d9`, initially clean. Release lockfile: Claude
ACP 0.85.1 / Agent SDK 0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. Three bounded read-only audits
covered tools/authority, peer delivery and tasks; the lead reconciled interactions and independently
reproduced both findings. No product code or specifications changed.

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

**Closed ASRF-1/2 — `473d74dd` (INV §2/§7/§10/§15/§17).** Post-result cursor actions now retain
their durable owner through startup and live callback faults, retry on the task dispatcher's bounded
schedule with attention after ten failures, and preserve Stop's closure fence. The obsolete direct
stage executor is removed; a real run → stage task → dispatcher test proves frozen/applied fast mode
and run-name grouping. Focused pipeline/state/server tests, both full Go variants, tagged build,
spec lint and diff checks pass. Provider/device receipts remain unchanged.

Historical review evidence: source mode at `b8157f13`, clean at entry.
Release lockfile pins Claude ACP 0.85.1 / Agent SDK 0.3.286 and Codex ACP 2.1.1 / CLI 0.159.3;
this audit exercised source and isolated fake-provider tests, not installed/native provider behavior.
Two bounded readers traced ordinary stages and room lifecycle; the lead integrated the ownership,
interruption and context joins and independently reproduced the recovery failure. Product code and
specifications unchanged.

Connected evidence ledger (catalogue topics 19 and 20):

| Retained topic | State, authority and effect trace |
|---|---|
| 19 — Ordered stages and standing orchestration | Frozen template/runtime/input snapshot → unique stage task and optional managed coordinator child → ordinary task capacity/identity claim → launch or same-identity resume → assignment/report/wait → immutable stage result and closure → release/cancel descendants → approval/next task/final result. Read Start/Continue/Retry/Replace/Stop, dispatcher, report transaction, lineage projections, cleanup and startup. A coordinator's child result cannot substitute for the standing stage report; replacement retains coordinator binding and predecessor work. Durable controller recovery and task-owned dispatch are closed in `473d74dd`. |
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
acceptance without rejudging. The recorded
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

ASRG-1 is repaired in `63a723e5`; ASRG-2/5 in `8b7db7ed` (2026-10-23).
The room path shares exact-attempt startup compensation and its retryable failure owner;
checks share continuation eligibility and atomically revalidate ownership/capacity before
charging, without reserving a task runtime; cleanup advances a bounded agent-key cursor past
in-flight, cooldown and failed rows. TS-14.R6 and TS-10.R45/R47 record these contracts.
Focused quota/state tests and tagged race checks pass. Original-code overlays fail the room,
capacity/lifecycle and ninth-cleanup regressions. Running/resumed startup, failed compensation
retry, changed task/room owners and mixed blocked cleanup rows are covered.
Both full Go variants, tagged build and documentation/launcher checks pass.
ASRG-3 was closed by ASF-1; ASRG-4 was closed with ASW-2. No ASR-G findings remain;
native-provider receipts remain owed. The ledger below records the original source audit
(`fa092f63`), rather than current open defects. ASRE-1 closed in `5091d0d3`.

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

ASRH-1/2/4 closed with ASF-2 and ASRH-3 closed with ASF-6 on 2026-10-22; no ASR-H findings remain. The retained original
audit evidence follows; the repairs use existing hold/Steer and client recovery contracts.
Source snapshot `d49a8f86`; tree clean at entry. Repository source and release-lock profiles:
Claude ACP 0.85.1 / Agent SDK 0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. Three bounded discovery
audits were integrated; the findings below were independently checked against source and probes.
No product code or specifications changed during the original audit.

**Closed ASRH-1 — `2268e279` (INV §1/§5).** Steer clears only its submitted draft revision and
current composer owner, including attachment cleanup/feedback. The delayed-reply/newer-draft and
navigation regressions failed before repair and pass afterward in `Composer.test.tsx`.

**Closed ASRH-2 — `2268e279` / `f8338ac5` / `64bd3962` (INV §1/§5/§11).** Every hold has an
authoritative `held_id`; atomic conditional withdrawal returns the exact removed message or no
message. Desktop/phone restore server-returned content only; stale/no-hold responses cannot revive
cached text. Runtime/server/UI tests cover replacement, stale IDs, exact text/request ownership,
overlapping withdrawal and no-hold recovery. Chromium verified withdrawn text plus a newer draft,
and a stale two-device withdrawal that preserved then exactly restored the replacement.

**Closed ASRH-3 — ASF-6 (`12899812`, `85e7abf2`).** CLI archive lookup now requires complete,
certain selection; its original finding and verification closed independently before this merge.

**Closed ASRH-4 — `2268e279` / `f8338ac5` / `64bd3962` (INV §5/§8/§11/§15).** Text shares the
existing request receipts. Steer records uncertainty before native dispatch; only explicit
pre-consumption refusal restores an automatic hold. Unknown text/media stays available for a
deliberate new Send, with typed `delivery_uncertain`, truthful repeat-risk feedback and no automatic
Steer retry. The lost-acknowledgement fake-ACP regression failed under the original legacy restore
policy and now proves one Steer with no duplicate prompt after late success, both with and without
request IDs. Definite-refusal, stale promotion, stopped/crashed input, HTTP replay/conflict, CLI ID,
and desktop/phone recovery tests pass. Unrelated text cannot dismiss an older uncertain recovery.
Rendered text-only recovery has no page errors. Physical-phone and live-provider receipts remain
owed; deterministic fake consumption is not a native receipt.

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
ownership/acknowledgement boundaries; ASRH-1 loses newer intent. ASF-6 closed ASRH-3's incomplete
CLI lookup. J2/J4 challenges retain the existing permission/status, quota and owner-admission
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
| 19/20 — F | Ordinary stage-task progression and room/judge result capture use durable controller recovery and the shared task dispatcher (`473d74dd`); E/G defects apply through shared services. |
| 22 — G | Quota evidence, check isolation/admission, source-owner continuation and cleanup retain ASRG-1–5. Passive quota snapshots remain planned; missing live receipts do not establish a parser/provider failure. |
| 23 — H | Desktop/phone/CLI/local HTTP intent and recovery retain ASRH-1/2/4 and ASRD-4; ASF-6 closed ASRH-3. Reconnect refreshes state without replaying mutations; physical-device suspension and native anchoring remain open. |
| 24 — I | Frozen diagnostics, revision/adoption limits, combined failure costs and the paired experiment below were assessed. PROMPT-2 keeps its original owner; no measured general quality or excess-process conclusion is supported. |

Whole-journey assurance ledger (J1–J7):

| Join / interacting journey | Boundary, failure/recovery owner and conclusion |
|---|---|
| J1 — selected contract across fresh/resume/wake/switch/rollback | Frozen base plus runtime-only overlay reaches the provider start seam; model/root/profile/native ancestor defects remain ASRA-1/3/PROMPT-1. `applyKnowledgeOverlay` (`launch.go:434–454`) is rebuilt per process, not every turn. Clone/fork has its separate B2/B3 contract. Account/model/preset adoption remains a native gate. |
| J2 — manual multimodal queue/Steer + permission wait + Stop/resume | Runtime holds retain exact descriptors; Stop releases attachment receipts, and restart makes held/prepared copies explicitly recoverable (`chat.go:1264–1284`, `state/attachment_receipts.go:44–65`). ASRC-1/2/5 break read-loop/status/permission truth; ASRH-1/2 break draft/withdraw ownership. ASRH-4 (`chat.go:682–717,843–870`) restores an ambiguously consumed text Steer to automatic holding. These are existing failures, not a second attachment-loss finding. |
| J3 — Chucky mail/task/report + host approval + late continuation | Durable mail/assignment claims precede native activation; reports commit before cleanup/readiness and old generations cannot authorize current work. Peer Stop still requires its separate host waiter. ASRE-1 loses the watched changes before resumed assignment read; ASRE-2 allows oversized retained outputs. Claude fork's missing loaded session and tmux's stale environment weaken native registration (B2/B6). Late-turn guards have fake/source proof, with ASRC-2's final-status race and FS-01.R52/FS-16.A33 live gates retained. |
| J4 — ordinary/room stage + quota/Stop/restart | Task/run closure precedes cancellation; room capture binds generation/turn/attempt and accepted judge output, not arbitrary private prose. Pipeline controller recovery retains accepted post-result actions through startup/live faults and Stop (`473d74dd`). Quota room admission lacks ordinary-start compensation after creating its attempt (`quota_dispatch.go:154–237`, ASRG-1); G2–G5 retain check admission, cancellation settlement, isolation and cleanup defects. Terminal viewer detach preserves PTY work; terminal quota automation is unsupported and B1/B4/B5 recovery remains defective. |
| J5 — context through delegation/handoff/compaction | Assignment references/grants and room read receipts stay scoped; private room input does not become public judge prose. The bounded bridge persists separately through immediate Stop/wake and compaction, but switch marker append opens a second writer after target Resume (`primer.go:65–81`, `switch.go:266–278`, ASRD-1). D2/D3 affect order/durability and B1–B3 affect terminal/fork context. A4/A6 additionally invalidate assumptions that a recreated checkout is ready. Actual live context adoption is still unproved. |
| J6 — accepted-but-lost desktop/phone/CLI actions | Attachment request hash/receipt precedes native admission (`attachments.go:453–541`); identical explicit retry returns stored acceptance. Text-only sends have no universal idempotency promise. CLI reports uncertainty without automatic replay; phone recovery only rehydrates current truth. ASRD-4 annotation retry and ASRH-4 automatic held replay remain proven duplicate-effect paths; H1/H2 retain lost/stale human intent; ASF-6 closed H3. Phone agent routes are keyed by identity (`PhoneApp.tsx:128`), so an old component's completion is not evidence of a new cross-agent draft bug. |
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
was silently selected; ASRC-7's outcome drift is now closed by ASW-3; ASRF-2 is closed in `473d74dd`.

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
ancestor-project guidance; both PROMPT-1 and PROMPT-2's diagnostic gap are closed by the ASW-2
fixes (ASW2-7, ASW2-8–10; Changelog 2026-10-20). FS-18.A12/A13 native adoption/quality evidence
remains separate.

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

- **2026-10-23 — Fix: activations honor a claimed cleanup stop (INV §2/§15, TS-10.R19).**
  `StartActivation` checked only `turnActive`; a cleanup stop that had claimed an idle/error
  runtime still admitted mail/task activations. It now refuses stopping/stopped runtimes like
  `claimTurnLocked`. `TestStartActivationRefusesStoppingRuntime` failed before the fix. Found
  while comparing abandoned CR-A2-6/7 worktrees with main; their remaining content is
  superseded by `0f911d74`. One extra behavior was not ported: releasing a room whose quota
  retry is waiting on a person (`needs_attention`).

- **2026-10-23 — Fix: resumed-assignment continuation context (ASRE-1, INV §1/§10/§11/§15) —
  ASF-3 and ASR-E closed.** `5091d0d3` keeps the wait set through confirmation and projects it
  as `get_assigned_task.observed_changes`, including deleted sources' retained outcomes
  (TS-10.R53). The MCP probe and state continuation/restart regressions failed before the repair.
  Both full Go variants, the build and state/messaging race checks pass. Parallel ASRF/ASRG
  worker branches duplicated work another session had already landed. They were discarded unmerged.

- **2026-10-23 — Architectural review: combined next-release candidate `eb56b0c2`.**
  Three fresh domain readers plus parent integration assessed preservation/workspaces,
  admission/recovery/supervision, information/input/control, bounded overhead and installed runtime.
  Focused store/process/wire/UI tests, release transactions, tagged binary metadata and gate
  fixtures pass. ARCH-1 records the PID inspection failure as Worth fixing; no Must-fix finding
  or data loss was established. Receipt and coverage are in `docs/archive/reviews/architecture/`.
  No product/spec edit, release assembly or provider/device acceptance closure. Concurrent dirty
  runtime/handoff work was preserved and excluded.

- **2026-10-23 — Fix: quota recovery (ASRG-1 INV §2/§4/§15; ASRG-2 INV §2/§5/§15;
  ASRG-5 INV §7/§9/§16).** `63a723e5` shares ordinary/quota room-start compensation;
  `8b7db7ed` shares side-effect-free continuation eligibility, revalidates owner/capacity in
  the check-charge transaction, and rotates bounded cleanup past failed/skipped references.
  Running/resumed status-write faults and compensation retry preserve exact ownership without
  prompts, contributions or completed allowance. Capacity/lifecycle tests spend no check;
  changed task/room ownership is refused at charge; a ninth healthy cleanup progresses behind
  in-flight/cooldown/failing rows. Original-code overlays fail the corresponding regressions.
  Focused quota/state tests, tagged race checks, both full Go variants, tagged build and
  documentation/launcher checks pass. No native-provider receipt is claimed.
  ASRE-1 is excluded and remains with its separate agent; ASRF-1/2 are not selected.

- **2026-10-22 — Fix: retention and result bounds (ASF-4) — ASR-C closed.** ASRE-2 (INV §2/§11/§16,
  `91f30979`): ordinary, person and stage reports share `ValidateAgentTaskResult`, bounding each
  named output to 64,000 runes and the encoded result to 256 KiB with the typed refusal and no
  write. ASRC-6 (INV §16, `65de12ce`): ended children free their live slot behind a 64-entry
  tombstone; tool names are released with their inputs (TS-04.R63 updated). Both regressions
  failed before repair. Focused race checks, both Go variants and the build pass.

- **2026-10-22 — Fix: resource permissions and checkout readiness (ASF-5).** ASRA-5 (INV §10/§14,
  `8db628c7`) preserves owner bits; ASRA-6 (INV §1/§5/§15, `c379b51c`) waits through setup and
  shares its recreation notice/warning without letting waiter cancellation stop the owner.
  Pre-fix filesystem and real-Git probes distinguish both repairs. Both Go variants, tagged
  build and focused race checks pass. ASR-A and ASF-5 close; ASF-3/4 and live/device gates remain.

- **2026-10-22 — Integration: ASF-2 on main (INV §5/§15).** Preserved CR-A2 and ASF-6 closure
  while merging `fix/asf-2`. Existing input receipts replay before the task-continuation admission
  gate; new prompts still require that admission. The real-store HTTP regression now installs a
  waiting assignment between acceptance and retry and verifies the original acceptance survives.
  Focused merge checks, specification checks, regenerated UI and tagged build pass. The four
  pre-existing ready-change edits remain uncommitted and excluded from this merge.
- **2026-10-22 — Fix: admission, waiting and recovery (CR-A2-1–9, INV §1/§2/§3/§4/§5/§7/§8/§9/§10/§11/§14/§15/§16/§17) — CR-A2 closed.**
  Repairs `0f911d74` and `de0ac20a` preserve deferred cleanup authority, outstanding-claim
  capacity and confirmed assignees; wire actionable watch producers and bounded rediscovery;
  route mail/manual wake through assignment admission; use stable queue cursors past blocked
  prefixes; admit later automatic turns after settled failures; supersede exact room quota
  blocks, including needs-attention episodes; and rotate retained startup reap retries.
  Real-store and fake-provider regressions cover held/stale turns, original-agent native resume,
  exit/delete/park/lost-result recovery, stopped/borrowed/full-budget/manual/cancel continuations,
  shrinking backlogs, failed-turn mail/task admission, three room quota states and >8 startup
  retries. Both complete Go variants, tagged build, specification checks and diff checks pass.
  Final full receipts: `/tmp/chuck-cr-a2-{go,fts}-closure.log`. Earlier full runs exposed
  state-only capacity and process-removal-before-durable-release test assumptions; both fixtures
  now await real committed cleanup (five focused release repeats pass). The documented bootstrap
  download-start flake occurred once and passed a focused rerun and final full suites; no installer
  repair is claimed. Existing credentialed/device gates and ASF-2–ASF-5 findings remain owed.
- **2026-10-22 — Fix: certain CLI archive selection (ASF-6 / ASRH-3, INV §7/§8/§11/§16).**
  Bare launch now propagates archive errors and uses complete, validated pagination within
  ten 200-row pages and ten seconds; uncertainty requires explicit `--resume` or `--new`.
  Launch-boundary HTTP regressions cover failures, late unique matches, cross-page duplicates,
  complete absence, lookup bound, malformed pages and explicit new; the HTTP-failure case fails
  against the original launch code through a temporary Go build overlay. Focused CLI tests,
  both Go variants, spec checks and tagged build pass. ASF-6 is closed; ASF-2–ASF-5 and
  ASRH-1/2/4 remain open. Isolated worktree verification preserves concurrent main-checkout work.
- **2026-10-22 — Fix: input acceptance and recovery (ASF-2, INV §1/§2/§5/§7/§8/§11/§15/§16/§17).**
  ASRD-4 and ASRH-1/2/4 are closed by `2268e279`, `f8338ac5` and `64bd3962`; no ASF-2 finding
  remains. Stable annotation/text command receipts prevent duplicate effects, successful replies
  preserve newer drafts, withdrawal restores the exact removed input, and uncertain Steer never
  automatically repeats consumed work. FS-03.R100/A79, FS-13.R32/A22, TS-02.R55 and TS-03.R74
  describe the shipped contract. Both Go variants (`env -u CHUCK_RUNTIME_ROOT GOFLAGS=-p=1 make test`),
  995 UI tests, UI/embed/binary builds, focused runtime/server/state race checks and diff checks pass.
  Chromium screenshots: `/private/tmp/asf2-withdraw.png`, `/private/tmp/asf2-two-device-withdraw.png`,
  `/private/tmp/asf2-uncertain-recovery.png`. The known pipeline Stop journey failed earlier and then
  passed in both full variants; no unrelated pipeline repair is claimed. Native/provider/device gates
  remain above. Work is isolated on `fix/asf-2`; the other checkout is preserved. ASRH-3/ASF-6 and all
  sibling fix queues remain independent; this fix closure adds no new review unit.
- **2026-10-22 — Fix: torn transcript tail readback (CR-A1-3, INV §2/§7/§9/§11) — CR-A1 closed.**
  The shared transcript reader now stops at the last newline, so readback, reconciliation and
  replay consume the same complete-record prefix that writer Open keeps; malformed/oversized
  middle-record skips are unchanged. TS-02.R8 now states the shared boundary. Regression
  `TestReadbackMatchesWriterRecoveryForUnterminatedRecord` failed before the fix (two records
  before Open, one after) and passes; `make test` and `make build` pass. Entry dirty
  ready-change edits are preserved. No CR-A1 findings remain.
- **2026-10-22 — Review: interruption-safe reindex — closed without findings.** Reviewed
  substantive commits `90caf6b4` / `467b70ba` at `0ac1c55c`, including shared live indexer and
  removal callers, CLI diagnostics, source inventory/validation, complete-prefix replay and
  retained-field authority. Two focused independent readers checked transaction routing and
  source failure isolation; the parent checked their conclusions, specifications in both
  directions, tests and native Resume consumer. No new local-choice question or product decision.
  CR-A1-1/2 are closed; CR-A1-3 remains open with the recommendation recorded above. No fix
  model is needed for the closed reindex implementation unit. Unrelated entry dirty work is
  preserved. Concurrent commit `4d437bac` captured this review's handoff receipt; the review's
  final state-only commit corrects its summary and changes only the handoff.
  Independent verification: full `internal/index` / `internal/archive` suites, CLI
  `TestReindexCommandPartialRepairAndRetry`, and server `TestFailedReindexPreservesNativeResume`
  pass in both untagged and `sqlite_fts5` builds (`env -u CHUCK_RUNTIME_ROOT`,
  `GOCACHE=/private/tmp/chuck-cache`, `GOFLAGS=-p=1`, `-count=1`). The server test initially
  failed to bind `::1` under the sandbox; approved loopback reruns pass. Real process kill at
  retained-field restoration, full old-projection comparisons, late storage abort, partial
  repair/retry, annotation/partial-turn compatibility and actual native load serialization
  provide distinct contract evidence under TS-06.R35 / INV §17. No same-fixture kill-to-Resume,
  live-provider adoption, stronger offline exclusion or release architectural receipt is claimed.
  Invariant sweep: §§1/2/4/7/8/9/10/11/15/16/17 apply to replacement state, shared builders,
  transaction/buffer/fixture cleanup, failed-source reads and diagnostics, SQLite durability,
  wiring, metadata meaning, publication order, streaming replay and independent oracles; no
  findings. §§3/5/6/12/13/14 have no applicable surface: no transient persisted field, concurrent
  claim, runtime contract, installed-tool invocation, markup or HTTP/security boundary changed.
  No rendered gate applies. Spec and diff checks pass. This administrative closure adds no
  review obligation and changes no product code or specification.

- **2026-10-22 — Implemented interruption-safe offline reindex.** Each conversation replaces
  its session/search/tracking projections and restores native identity/live settings in one
  transaction. Missing/unusable sources preserve complete prior data; healthy neighbors rebuild;
  storage failures stop further repair. Added complete-projection, late storage failure,
  process-kill/reopen/retry, CLI partial-result/retry and native Resume wire tests. The original
  baseline fails the missing-root and late-invalid-metadata preservation regressions. No schema,
  source format, UI, online repair or live-provider adoption change is claimed.
  Closure: unset `CHUCK_RUNTIME_ROOT`, `GOCACHE=/private/tmp/chuck-cache`, `GOFLAGS=-p=1`;
  `make test` (both Go variants), `make build` and diff checks pass. One parallel tagged run hit
  the already-recorded `TestOrdinaryStageAgentStopPausesPipelineRun` timing flake: Stop returned
  409, "a resume is already in progress" (`pipeline_handlers_test.go:347`). The independent full
  tagged run passed; five focused repeats passed; the final `make test` passed using successful
  test-cache results. No pipeline repair or live-provider proof is claimed.
  After closure, integrated concurrent main commit `44633fbf` at `08470872`, preserving both
  changelog entries. On the combined source, focused reindex/SSE checks pass in both Go variants,
  the full bus package passes, and the build passes; the full matrix receipt above is for the
  pre-integration reindex source. Uncommitted context-reference work in main is excluded.

- **2026-10-22 — Fix: sustained capacity and cost (CR-A7-1 INV §1/§11, CR-A7-2 INV §4/§16,
  CR-A7-3 INV §16, CR-A7-4 INV §7/§16, CR-A7-5 INV §4/§16).**
  CR-A7-4: the periodic reconciliation sweep first asks `Manager.StaleCorrectionCandidates` for
  agents with a running row and a status older than the stale window, and reads only their
  transcripts; the fsnotify path, apply-time recheck and read-error logging are unchanged
  (TS-01.R8). `TestReconcileSweepSkipsIneligibleRetainedHistory` (stopped agent's transcript is a
  FIFO) hung with the filter disabled. Live agents with stale status are still re-read each
  sweep; no unchanged-file skip was added. Closure: `make test` (both Go variants,
  `GOFLAGS=-p=1`), `make build`, 984 UI tests and the UI build pass; bus and SSE overflow tests
  pass under `-race`.
  CR-A7 is closed. Its cost/ownership ledger and evidence limits remain in this file's history at
  `0ac1c55c` and in the review charter.
  CR-A7-2: share/read transcript scans go through `Service.scanTranscript`, which takes one of
  four reference-scan slots (separate from the eight peer slots), refuses excess with transient
  `context_busy`, and stops on request cancellation; MCP handlers now pass their contexts.
  TS-04.R28 and FS-17.R3 record the limit and code. `TestReferenceScansAreAdmittedAndCancellable`
  covers cancelled share/read, full-pool refusal and peer-pool independence; the review probe
  (16 cancelled reads all completing) is the pre-fix evidence. No physical read boundary was
  added: fixed-span and oversized-record edge rendering are unchanged.
  CR-A7-5 (INV §4/§16): `agentState.closePersistence`, which already runs after the generation's
  last emit, now calls `Journal.forgetGeneration`, deleting that generation's control-queue shed
  marker. `TestShedGenerationMarkersEndWithTheirGeneration` (32 shed generations against a full
  FIFO, `-race`) keeps live markers until teardown and retains none after; it drives the journal
  seam, not a real chat-runtime Stop.
  CR-A7-1: a bus subscriber's overflow channel closes at its first dropped event and
  `/api/events` ends that stream instead of pinging on; the browser reconnect resnapshots and now
  also refetches open pipeline run details (TS-03.R9 documents the strategy).
  `TestSSEOverflowEndsStreamAndReconnectCarriesLostState` (stalled writer, lost terminal state,
  257 later events) failed with the overflow case disabled; the reopen run-detail UI test failed
  before. CR-A7-3: `hydrationIds` exists only between `onopen` and the `hydrated` marker; the
  1,000-update retention test failed before. Per-agent transcript tokens are unchanged.

- **2026-10-22 — Fix: operator action and observation truth (CR-A6-1 INV §8/§11, CR-A6-2 INV §1,
  CR-A6-3 INV §8/§10, CR-A6-4 INV §2, CR-A6-5 INV §1/§11, CR-A6-6 INV §8).**
  A refused attachment Steer (receipt `failed`) now answers with the TS-03.R39 refusal envelope on
  first response and retry, so the CLI exits nonzero; receipt `delivery` is `sent` only for
  `submitted` (TS-03.R62). `TestRefusedAttachmentSteerIsNotReportedSent` failed before the fix.
  CR-A6-2 (INV §1): the phone keys `RunScreen` by run, so navigating between runs drops the prior
  run's Stop dialog, input and placeholder data; the new `RunScreen` navigation test failed before.
  CR-A6-3 (INV §8/§10): the human operator row renders `checkout not deleted: <reason>` (and
  `checkout_deleted`) for project archive; covered in `TestOperatorHumanReceiptsAndStoredFailures`.
  CR-A6-6 (INV §8): `steerError` maps history-unavailable and in-flight refusals through Send's
  409 wording; `TestHistoryUnavailableIsADefiniteRefusal` now covers Steer.
  CR-A6-5 (INV §1/§11): the desktop transcript store records the highest applied seq per agent and
  drops a streamed event at or below it; the new reducer test failed before. End-to-end timing
  across the SSE hop was not reproduced.
  CR-A6-4 (INV §2): `chuck agent status` keeps a peer-Stop request only when it is the server's
  live `pending_permission` and otherwise reports coverage incomplete (TS-03 status projection);
  `TestOperatorStatusTrustsServerLivenessForPeerStopRequests` covers dead and live waits.
  Closure: `make test` (both Go variants, `GOFLAGS=-p=1`), `make build`, 982 UI tests and UI build pass.
  CR-A6 is closed. The review's non-finding evidence limits (generic project-archive mid-stop
  error, newest-only unretained permission under `complete`, crash-left ordinary request still
  showing Approve/Deny) remain recorded in its review file, not as open findings.

- **2026-10-22 — Fix: executable workspace and configuration (CR-A3-1 INV §1/§12/§15, CR-A3-2
  INV §1/§3/§10/§11, CR-A3-3 INV §2/§11/§12).** `ensureWorktreeCheckout` now requires an existing
  owned checkout's Git common directory to be the recorded repository, refusing a foreign worktree
  or plain directory without touching it (TS-12.R4). Latest-setup Resume resolves and preflights
  the source against the frozen cwd. `preflightClaudeChildProfile` refuses bound Claude
  launch/resume/switch whose composed `CLAUDE_CONFIG_DIR`/`HOME` selects another root; unbound
  starts are unchanged (TS-07.R19). Regressions `TestOwnedCheckoutPathReuseIsRefused` (real Git,
  frozen Resume), `TestResumeConfigRefreshResolvesFrozenCwd` and
  `TestBoundClaudeChildMustUseThePreflightedRoot` each failed before the fix; federation fixtures
  now pin `CLAUDE_CONFIG_DIR` to their fixture root. `make test` passes with `GOFLAGS=-p=1`.
  CR-A3 is closed. No credentialed Claude run observed native profile adoption; ASRA-5/6 stay
  with ASF-5.

- **2026-10-22 — Completed reindex preservation design:** the operator accepted partial repair.
  FS-05.R43–R45/A26–A27 now define healthy rebuild, failed-source preservation and nonzero partial
  results. Planned TS-02.R56–R58 supersede R9's wipe/restore algorithm with one atomic transaction
  per conversation, complete transaction routing, source inventory/validation and scoped errors.
  A focused Luna supplied source facts; independent clean-context Astra recommended and then
  challenged the concrete boundary without required changes. Created the
  waiting reindex change (implemented above), removed its source idea and
  updated pickup guidance. Spec/twin-skill/diff checks pass. No product code, implementation start
  or CR-A1 finding closure; CR-A1-3 remains separate.

- **2026-10-21 — Started reindex preservation feature design:** explicit `/design-feature` selected
  CR-A1-1/2. Read current archive/CLI/replay behavior and independent frontier assessment; a focused
  GPT-6 Luna reader supplied source/test facts. Drafted planned FS-05.R43–R45/A26–A27 and recorded
  the idea as being defined. The operator is choosing partial-repair versus whole-rebuild failure
  behavior before technical design; no ready change, TS update, product edit or finding closure.

- **2026-10-21 — Fix: supervision publication and resume-spec drift (CR-A5-1 INV §1/§10, CR-A5-2
  INV §10).** Accepted ordinary and stage reports, room-backed stage acceptance (new
  `pipeline.Publisher.PublishTaskUpdate` through the server's single helper) and completed task
  release now publish the source `task_update`. Server report tests and the mixed-room acceptance
  test require the stored id/state/outcome/revision; all three failed before the fix. Release
  publication is source-only (no dedicated test). TS-09.R38/R58, TS-04's resume summary and
  FS-14.R66/A55 now match FS-01.R10's visible load refusal. CR-A5 closes; other core-audit and
  ASF findings remain open. Ready-change edits preserved.

- **2026-10-21 — Clarified core repair pickup readiness:** checked the live queue and workflow §8.
  CR-A1-1/2 remain design-pending rather than implementation-ready; A1-3 remains independently
  actionable. CR-A2 is selectable for fix validation/design with cleanup/continuation enforcement
  specified before code, without creating a separate architecture unit or absorbing ASF-3.
  The assessment/handoff now state that distinction. No product/spec change or new work unit.

- **2026-10-21 — Frontier reassessment of refactor scope:** after the operator challenged relying
  on Luna for architectural judgment, a fresh-context GPT-6 Astra reviewer independently checked
  `516488c5`; the primary agent also checked transaction writers, task turn capture and stop gates.
  The [assessment](CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md#frontier-challenge-and-primary-judgment)
  retains narrow reindex design, including revision of TS-02.R9's unsafe preservation algorithm.
  Task fixes require explicit retry authorization and continuation-delivery contracts; the earlier
  blanket assertion that existing fields suffice is withdrawn. No standalone task architecture
  project is yet justified. No implementation, new execution unit or finding closure occurred.

- **2026-10-21 — Independent challenge of refactor recommendations:** a fresh-context GPT-6 Luna
  reviewer checked `72515c47` and rejected a standalone task-ownership refactor: the existing
  architecture already represents the required claims/assignments/cleanup, and focused coordinated
  fixes have less coupling risk. It approved narrow reindex preservation-boundary design while
  leaving the transaction/staging choice open. The parent accepted the challenge and revised the
  [assessment](CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md#independent-challenge-record).
  No new execution unit, product/spec edit, test/probe, finding closure or architecture receipt.

- **2026-10-21 — Fix: mail send result enum (CR-A4-1, INV §10/§11/§17).** `send_message` now
  returns TS-04.R53's `delivery: queued_waking` / `queued_deferred`; the three messaging tests
  that asserted the stale `waking`/`deferred` spelling now assert the specified enum alongside the
  existing stored-wake and activation checks. No spec change (restores specified behavior). CR-A4
  closes; other core-audit and ASF findings remain open. Ready-change edits preserved.

- **2026-10-21 — Core audit before-fix synthesis:** two focused GPT-6 Luna readers and parent
  integration reconciled the completed CR-A1–CR-A7 findings against `3c8b0cec`.
  [Assessment](CORE-RELIABILITY-POST-REVIEW-ASSESSMENT.md) records interruption-safe reindex
  constraints, task claim/turn/capacity coupling, and publication/overflow/replay convergence;
  bounded source checks identified missing desktop run-detail invalidation on reconnect, retained
  under CR-A7-1. Finding owners/counts and executable units are unchanged. Existing dirty CR-A4-1
  implementation and ready-change edits were preserved. Independent checks confirmed the scoped
  source maps; spec, local-reference and diff checks passed. No product/spec edit, audit rerun,
  probe, product test execution, fix closure or release architecture receipt was performed.

Settled work through v0.18.0 is in
[`HANDOFF-through-2026-10-20`](../archive/state/HANDOFF-through-2026-10-20.md).
v0.18.0 was published 2026-10-21 with the explicit one-release architectural-review waiver.

- **2026-10-21 — Review: sustained capacity and cost (CR-A7).** Audited source `8c837942`;
  recorded three Must-fix findings for missing SSE overflow recovery, uncancelled/unadmitted
  reference scans and post-hydration traffic retention, plus two Worth-fixing findings for idle
  full-history reconciliation and shed-generation marker lifetime. Disposable corpora/probes
  establish scan/allocation growth and retained-state mechanisms; focused existing contracts pass.
  No native quality/usage or cross-agent starvation claim is made. Other audit/fix ownership and
  live receipts remain unchanged; only the handoff is committed and ready-change edits preserved.

- **2026-10-21 — Review: operator action and observation truth (CR-A6).** Audited product source at
  `3334471d`; recorded three Must-fix findings (refused attachment Steer reported as sent, phone run
  page reusing state across runs, hidden checkout-deletion failure in CLI project archive) and three
  Worth-fixing findings (dead peer-Stop approvals in CLI status, duplicate live transcript events
  after refetch, Steer refusal error mapping). The refused-Steer probe fails as described; focused
  CLI follow/status tests pass. Only the handoff is committed; the four ready-change edits are preserved.

- **2026-10-21 — Review: supervision and result authority (CR-A5).** Audited product source at
  `d39cdc9d` through state-only `785e3f4e`; recorded one Must-fix missing task-result publication
  and one Worth-fixing conflict between pipeline fallback requirements and the shared native-load
  refusal contract. The independent source-task event probe fails for ordinary and pipeline
  reports; focused lifecycle, room/capture, native refusal and both SQLite state/pipeline checks
  pass. Existing recovery/cleanup/result-bound owners and live provider gates remain unchanged.
  Only the handoff is committed; the four ready-change edits are preserved.

- **2026-10-21 — Review: admission, waiting and recovery (CR-A2, second pass).** A parallel
  audit of the same `f1c59961` source independently reproduced CR-A2-1–6 and added CR-A2-7
  (failed turn silently blocks automatic turns), CR-A2-8 (room Resume after quota never takes a
  turn) and CR-A2-9 (startup reap contention strands a start). Probes rerun by the parent; fix
  model stays medium. Only the handoff is committed; ready-change edits remain untouched.

- **2026-10-21 — Review: information and operating knowledge (CR-A4).** Audited current HEAD
  `7e0e1a0a`; recorded one Must-fix mail-result enum mismatch. An independent specification-based
  MCP probe reproduces both waking/deferred spellings; focused content/access, annotation,
  mail-receipt, package/overlay, tool-result and handoff/native-request checks pass. No provider
  adoption is claimed and existing ASF/core-audit ownership and live gates remain unchanged.
  Only the handoff is committed; the operator's four ready-change edits remain untouched.

- **2026-10-21 — Review: executable workspace and configuration (CR-A3).** Audited current HEAD
  `7ec20a01`; recorded three Must-fix findings for accepting a foreign repository at an owned
  path, resolving latest-setup Resume against a different cwd, and Claude environment redirection
  bypassing personal-source validation. Real Git/server, native-wire and actual child observers
  reproduce the mechanisms; root probes pass in both SQLite variants. The connected
  fork/setup/loss/frozen-Resume child control and focused suites/spec checks pass. Existing
  ASRA-5/6 remain with ASF-5; no other finding owner or live receipt is closed. Only the handoff
  is committed; the operator's ready-change edits remain untouched.

- **2026-10-21 — Review: admission, waiting and recovery (CR-A2).** Audited current HEAD
  `f1c59961`; recorded six Must-fix findings for premature cleanup, retained-slot undercount,
  lost retry assignee, stranded watches, waking mail outside continuation admission and blocked
  task/mail prefixes. Isolated store/fake-wire probes reproduce all six in both SQLite builds;
  waiting-exclusivity and real-process failed-reap/recovery controls pass. Focused existing
  admission, mail, quota and cleanup checks pass. Other audit/fix owners and live gates remain
  unchanged. Only the handoff is committed; entry ready-change edits are preserved.

- **2026-10-21 — Review: durable truth and recovery (CR-A1).** Audited current HEAD `da44be64`;
  recorded three Must-fix findings for interrupted reindex retention, destructive missing-input
  handling and inconsistent torn-record readback. Real child-process/populated-store probes
  reproduce the maintenance failures; a real-file framing probe fails as expected. Focused
  journal, persistence, native continuation, archive and reconciliation checks pass, including
  metadata fallback and FTS5. Live provider/terminal receipts remain owed. This unit stays open
  for fixes; the other six core audits and existing finding owners remain unchanged. Only this
  handoff is committed; the operator's ready-change edits are preserved.

- **2026-10-21 — Review: workflow reliability skills and architectural release gate.**
  Closed the substantive unit `e96c47fe` / `97c67f39` at current HEAD
  `ff759c401020b63ff322c3796a89f745c8756525`, including current skill twins, callers and
  the specified v0.18.0 waiver. No findings or local-choice questions; no fix model is needed.
  Checked TS-06.R35–R36, FS-00.R3/R13–R15 and workflow §§8/16–18 in both directions.
  Parent reviewed workflow/skill contracts; an independent focused reviewer checked gate/tests
  and CI wiring. Real-Git fixtures prove original review → verified finding repair → recorded
  closure → release without another architectural review, while missing/incomplete receipts,
  invalid provenance, open findings and uncovered substantive work remain blocked.
  INV sweep: §4 fixture cleanup, §10 twin/routing/CI wiring, §12 Git invocation/output handling,
  §15 committed evidence before publication and §17 independent Git/mutation evidence apply
  and pass; §§1–3, §§5–9, §11, §§13–14 and §16 have no applicable surface in this unit.
  Eight gate tests, launcher/twin checks, launcher mutation fixtures and `make check-specs`
  passed. No product code/spec changes, live release run, real architectural receipt or core
  audit is claimed; other findings and gates retain their owners. Entry dirty-tree edits under
  `docs/ready-changes/` remain untouched. This state-only closure creates no new review unit.

## Blocked on human

- **v0.18.1 verification-only fixes:** approval requested to correct the two pre-existing
  fixture assumptions in REL-VERIFY-1 (~20 lines, no product behavior change). The ARCH-1 repair
  is committed as `69082b39`; full FTS5/distribution/995 UI tests/vet pass. The no-tag Go checkpoint
  remains blocked by bootstrap readiness. Proposal: `/private/tmp/chuck-release-test-proposal.patch`.
  Version and publishing approval will be requested only after the candidate is verified.

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

See [RELEASE-v0.18.0-notes.md](../archive/state/RELEASE-v0.18.0-notes.md).

See [`RELEASE-v0.17.0-notes.md`](../archive/state/RELEASE-v0.17.0-notes.md),
[`RELEASE-v0.16.0-notes.md`](../archive/state/RELEASE-v0.16.0-notes.md),
[`RELEASE-v0.15.1-notes.md`](../archive/state/RELEASE-v0.15.1-notes.md) (supersedes the
never-published v0.14.0 and v0.15.0 notes) and
[`RELEASE-v0.13.0-notes.md`](../archive/state/RELEASE-v0.13.0-notes.md).
