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
  ownership and ASF-1 turn/control settlement are closed. Select remaining work through
  [AGENT-SURFACE-FIX-UNITS.md](AGENT-SURFACE-FIX-UNITS.md); ASF-2–ASF-6 wait.
  Their source reviews and open findings remain below; do not reopen completed refactors.
- **Core audits:** CR-A1–CR-A7 are researched and waiting in the review queue; none has run.
  [CORE-RELIABILITY-AUDIT-PLAN.md](CORE-RELIABILITY-AUDIT-PLAN.md) owns the charters.
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

- None. v0.18.0 is published; both Go variants, 980 UI tests, static/spec/link checks and
  the distribution build passed.
- Other available review/fix units and owed live receipts remain below.

## Acceptance gates still owed

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

**ASW-1, ASW-2, ASW-3 and ASF-1 closed; all other units waiting.** The complete [unit designs and primary finding map](AGENT-SURFACE-FIX-UNITS.md)
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
| [ASF-2](AGENT-SURFACE-FIX-UNITS.md#asf-2--input-acceptance-and-recovery) | `/fix ASF-2` — annotation receipts, drafts, exact withdraw and uncertain Steer | ASRD-4, ASRH-1, ASRH-2, ASRH-4 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-3](AGENT-SURFACE-FIX-UNITS.md#asf-3--durable-continuation-and-controller-recovery) | `/fix ASF-3` — task/pipeline/quota recovery and obsolete executor removal | ASRE-1, ASRF-1, ASRF-2, ASRG-1, ASRG-2, ASRG-5 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-4](AGENT-SURFACE-FIX-UNITS.md#asf-4--retention-and-result-bounds) | `/fix ASF-4` — native-child retention and task-result bounds | ASRC-6, ASRE-2 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-5](AGENT-SURFACE-FIX-UNITS.md#asf-5--project-readiness-and-resource-permissions) | `/fix ASF-5` — checkout readiness and restrictive resource modes | ASRA-5, ASRA-6 | Waiting — medium, Codex Terra or Claude Opus |
| [ASF-6](AGENT-SURFACE-FIX-UNITS.md#asf-6--cli-archive-selection) | `/fix ASF-6` — certain CLI resume/new selection | ASRH-3 | Waiting — medium, Codex Terra or Claude Opus |

## Review queue

**Core reliability audit units:** explicitly requested subsystem reviews of the complete current
implementation and callers, not reviews of planning commits. Read the selected charter, refresh
current fix ownership and record the reviewed HEAD. No core audit has started; all seven subjects are mapped.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A1](CORE-RELIABILITY-REVIEW-A1.md) | Durable truth and recovery: conversation/session authority through Stop/restart, archive/Restore, interrupted reindex and native continuation | waiting to start |
| [CR-A2](CORE-RELIABILITY-REVIEW-A2.md) | Admission, waiting and recovery: task/dependency eligibility, assignment/capacity, mail/quota retry policies, retained cleanup and startup progress | waiting to start |
| [CR-A3](CORE-RELIABILITY-REVIEW-A3.md) | Executable workspace/configuration: checkout/setup readiness, shared resources, fresh/frozen selection and actual native cwd/environment/setup | waiting to start |
| [CR-A4](CORE-RELIABILITY-REVIEW-A4.md) | Information/operating knowledge: selected content and source identity, authorized reads, mail receipts, managed guidance and actual native delivery | waiting to start |
| [CR-A5](CORE-RELIABILITY-REVIEW-A5.md) | Supervision/result authority: accepted reports, matching execution settlement, descendant closure, judge publication, room-stage acceptance and retained progression | waiting to start |
| [CR-A6](CORE-RELIABILITY-REVIEW-A6.md) | Operator action/observation truth: exact targets/guards, acceptance versus completion, partial/unknown effects, CLI output and current-state recovery | waiting to start |
| [CR-A7](CORE-RELIABILITY-REVIEW-A7.md) | Sustained capacity/cost: retrieval scan work, accumulated data/views, shared pressure, temporary resource lifetime and unnecessary native work | waiting to start |

The [audit plan](CORE-RELIABILITY-AUDIT-PLAN.md) records all seven charters and the separate proposed
cross-audit integration proof. These subsystem audits do not substitute for the candidate-bound
architectural review before release or start any fix workflow.

**Original agent-surface campaign:**

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

## Review findings

The `NewAgentModal`/phone gap-test race fix is confirmed remotely (main CI run 39007164597).

Closed reviews through the `v0.17.0` cut are in the archive above.

### ASR-A — Instructions, configuration and execution environment — reviewed `4fbeb316` — **Fix model:** difficult — Codex Sol.

Source/profile delivery must agree with native authentication,
session isolation and frozen continuation choices; the other fixes mostly extend existing seams.
Audit complete; findings keep this same unit open. No product code or specifications changed.
Source mode: repository source plus installed, lockfile-pinned packages: Claude ACP 0.85.1,
Claude Agent SDK 0.3.286, Codex ACP 2.1.1 and Codex CLI 0.159.3. Tree was clean at entry.

ASRA-1/2 are verified closed by the ASW-2 review: new contracts preserve source model overrides
and native inheritance through continuation, and empty effort resets use the declared default.

PROMPT-1 and ASRA-3 are closed by the ASW-2 fixes (ASW2-7 effective root markers, ASW2-2 frozen
profile refusal; Changelog 2026-10-20).
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
review closed ASRC-4/7 and its fixes closed ASRC-3; ASF-1 closed ASRC-1/2/5 (2026-10-20, Changelog); ASRC-6 (ASF-4) remains. Audit complete; findings keep this same unit open. No product code
or specifications changed. Source mode: repository source; real-adapter shapes read from the
release lockfile packages (Claude ACP 0.85.1; Codex ACP 2.1.1 is a native binary, not inspected).
Bounded discovery for catalogue 06/11/12 was delegated and every reported finding re-read in code.

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
and independently reviewed in ASW-1; ASRD-3's history suppression and its remaining
recovery/control obligations closed with the ASW-1 fixes (Changelog, 2026-10-20). The retained audit evidence below
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

ASF-1 closed ASRG-3 (2026-10-20, Changelog). Audit complete; findings keep this same unit open. The fixes extend existing admission,
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

Settled work through v0.18.0 is in
[`HANDOFF-through-2026-10-20`](../archive/state/HANDOFF-through-2026-10-20.md).
v0.18.0 was published 2026-10-21 with the explicit one-release architectural-review waiver.

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
