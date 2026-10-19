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
  ASR-D (`c12c4251`) have been audited
  and have open findings; the remaining integrated Codex/Claude reviews are listed under
  [Review queue](#review-queue), sized for frontier reviewers. The
  [charters](AGENT-SURFACE-REVIEW.md) combine interacting mechanisms and require cross-system
  joins plus a final synthesis/performance review. All original 25 research areas remain in the
  [search catalogue](AGENT-SURFACE-SEARCH-CATALOGUE.md), not as separate tasks. Their
  evidence and findings are below; the campaign as a whole is not reviewed. Other findings and live gates
  remain independent.
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

- None. File reader formats and path copying closed 2026-10-19 (see Changelog); its
  real-device/room/native-anchoring gates were not claimed by the stubbed browser review.

## Acceptance gates still owed

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

## Review queue

Nine integrated reviews are sized for frontier reviewers to follow interacting mechanisms and
whole workflows. These are explicitly requested subsystem audits, not reviews of queue commits.
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
| [ASR-E](AGENT-SURFACE-REVIEW.md#asr-e--agent-tools-communication-and-durable-delegation) | Agent tools, communication and durable delegation | waiting to start |
| [ASR-F](AGENT-SURFACE-REVIEW.md#asr-f--pipelines-think-tanks-and-orchestration) | Pipelines, Think Tanks and orchestration | waiting to start |
| [ASR-G](AGENT-SURFACE-REVIEW.md#asr-g--failure-quota-and-continuation-recovery) | Failure, quota and continuation recovery | waiting to start |
| [ASR-H](AGENT-SURFACE-REVIEW.md#asr-h--human-intent-across-desktop-phone-cli-and-http) | Human intent across desktop, phone, CLI and HTTP | waiting to start |
| [ASR-I](AGENT-SURFACE-REVIEW.md#asr-i--whole-system-synthesis-and-agent-performance) | Whole-system synthesis and agent performance | waiting to start |

## Review findings

The `NewAgentModal`/phone gap-test race fix is confirmed remotely (main CI run 39007164597).

Closed reviews through the `v0.17.0` cut are in the archive above.

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
- **Must fix** — ASRA-4 — **Setup cancellation/timeout does not contain descendants**
  (INV §4/§9/§12/§16). `internal/server/worktree.go:254–280`: `exec.CommandContext` kills only
  `/bin/sh`, with no descendant cleanup or bounded pipe-drain wait. A setup command whose child
  retains stdout/stderr keeps `cmd.Run` waiting after caller cancellation or the ten-minute
  deadline, holding recreation and its start request until that child exits. Probe
  `sleep 0.6 & wait` with cancellation at 50 ms returns after about 615 ms, when the descendant
  exits; a long-running descendant defeats the execution bound (TS-12.R4/R5). Use the existing
  process-containment patterns to terminate/reap the setup tree and bound drain on cancellation;
  regress inherited output descriptors plus cancellation/deadline and concurrent start cleanup.
  Reusable probe: `/tmp/chuck-asr-probe/setup_timeout_probe_test.go`.
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

### ASR-B — Codex and Claude native integration fidelity — reviewed `6b5de19b` — **Fix model:** difficult — Codex Sol.

Audit complete; eight findings keep this unit open. Product code and specifications unchanged.
Fix sizing is independent of severity: ASRB-3/7 are easy; ASRB-1/2/4/6/8 are medium;
ASRB-5 is difficult because external-driver process identity, containment and restart ownership
must agree. Source mode: repository source plus locally installed lockfile packages, Claude ACP
0.85.1 / Agent SDK 0.3.286 and Codex ACP 2.1.1 / CLI 0.159.3. Assembly pins Node 22.22.0 and
Codex's patched component `2.1.1+chuck.1`; local test Node is 22.23.1. No native model turn ran.
The pre-existing dirty ASR-C row and its concurrent administrative closure were preserved;
`07f201a3` changes review state, not the inspected implementation. Three bounded read-only audits
covered Codex selection/profile, Claude negotiation and terminal drivers; findings were re-read
and integrated here.

- **Must fix** — ASRB-1 — **Terminal Stop loses the native conversation needed by Resume**
  (INV §1/§2/§11/§15/§17). `internal/runtime/terminal/terminal.go:170–189,209,347`,
  `internal/state/manager.go:139–147,348–364`, `internal/runtime/registry.go:232`:
  a fresh terminal launch indexes an empty native id. The CLI's SessionStart hook later records
  the real id only in `running`, while Stop deletes that row without updating
  `sessions.last_session_id`. Ordinary Resume reads the empty snapshot and omits `--resume`,
  starting a fresh Claude conversation despite FS-07.R16–R17's continuity contract. A terminal
  resumed from an earlier conversation can similarly retain its old snapshot id. Persist the
  authenticated native id at the shared hook/session boundary before it can be lost; regress
  launch → SessionStart → Stop/crash/restart → Resume and terminal→chat switch. The isolated
  real-state/PTY probe sees running id `native-sentinel`, saved id empty, and resume argv
  `["claude"]`. Existing persistence tests supply an invented resume id instead of recovering it.
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
- **Must fix** — ASRB-4 — **Non-PTY driver exits never release runtime ownership**
  (INV §4/§6/§9/§10). `internal/runtime/terminal/terminal.go:480–510`,
  `tmux.go:37–49`, `iterm2.go:66–76`, `internal/server/reconcile.go:211–237`:
  tmux/iTerm2 return no exit channel, so their watcher returns immediately. After the CLI or
  external viewer exits, the periodic sweep can remove the running row, but never removes
  runtime/registry ownership, closes persistence or invokes generation-scoped registration
  teardown. Resume can remain blocked as already started until explicit Stop. FS-01.R18 and
  INV §4 require exit cleanup, not only a done card. Feed confirmed driver exits into the existing
  generation-scoped teardown and regress native exit/viewer close plus later Resume. Source trace
  and a no-process driver fixture confirm the retained registry claim after pruning; this is
  distinct from ASRC-4's fresh-launch registration race.
- **Must fix** — ASRB-5 — **iTerm2 launches have no process identity for control or restart**
  (INV §4/§6/§9/§12). `internal/runtime/terminal/iterm2.go:66–76`,
  `terminal.go:221–224,294–306,355–363`, `internal/server/reconcile.go:211–237`:
  the accepted driver returns tty/window/session identifiers but leaves PGID zero. Launch records
  PID 0, so the next sweep marks a live CLI stopped and invalidates its hook token; Cancel cannot
  signal it. After a server crash the orphan reaper has neither a PID nor a retained driver handle
  to stop, so a lifecycle action can report success while the CLI survives (FS-07.R1/R15,
  FS-01.R20–R21). Establish a corroborated process/group identity and supported orphan recovery,
  or refuse this driver until those contracts exist. Regress live launch across a sweep,
  cancellation and crash→Stop/Switch. The PID-0 fixture proves false pruning plus retained runtime
  ownership; actual iTerm automation/child containment remains unverified.
- **Must fix** — ASRB-6 — **tmux panes can inherit another launch's environment**
  (INV §2/§6/§11/§12). `internal/runtime/terminal/tmux.go:24–34` sets the tmux client's
  `cmd.Env`, but supplies neither a per-session environment nor a child bootstrap. With an
  existing tmux server, new panes inherit its global/session environments; only configured
  `update-environment` names are copied from the client. Fresh `CHUCK_AGENT_ID`, hook URL/token,
  resources and backend/model variables can therefore be absent or stale, breaking FS-07.R3/R12
  and potentially addressing hooks to another agent. This follows the driver's argv plus the
  [primary tmux environment/new-session contract](https://man.openbsd.org/tmux.1#GLOBAL_AND_SESSION_ENVIRONMENT),
  not an actual local tmux run (tmux is unavailable here). Supply each launch's composed environment
  through a supported bounded mechanism without exposing secrets; regress two distinct launches
  against an already-running isolated tmux server and verify child bytes and hook identity.
- **Must fix** — ASRB-7 — **Distinct terminal agents collide on tmux session names**
  (INV §6/§9/§10/§12). `internal/runtime/terminal/terminal.go:573–583,642–644`,
  `tmux.go:24–34,71–84`: session names derive solely from the sanitized display title, which
  omits agent id/generation. Two agents with the same name/role/project, including ordinary
  unnamed agents, map to the same `new-session -s` and the second launch fails; punctuation-only
  title differences also collide. FS-07.R9/R17's independent runtime contract needs a unique
  machine identity separate from the title. Use agent/generation identity for the session and
  retain the title for display; regress duplicate and sanitization-colliding names. A probe of
  the actual helpers maps `a_one` and `a_two` to `chuck-Same---dev-p`; no live tmux launch ran.
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

### ASR-C — Runtime execution, concurrency and control — reviewed `6b5de19b` — **Fix model:** medium — Codex Terra or Claude Opus.

The status-ownership and registration-window fixes coordinate runtime, registry and server state;
the others extend existing seams. Audit complete; findings keep this same unit open. No product code
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
  `internal/server/launch.go:863–883` (`reapOrphanRuntime`, used by Stop/Switch/Archive/pipeline)
  and `chat.go:1309–1332` (`reconcileOrphanStop`) signal a recorded PID whenever `kill(pid,0)`
  succeeds; startup `ReconcileStale` (`runtime/reconcile.go:21`) and the prune sweep likewise keep
  a reused PID as live. After a server crash or reboot, a recorded adapter PID reused by an
  unrelated process keeps a ghost card, and Stop/Switch/Archive SIGKILLs that process. INV §9
  requires corroboration where liveness gates destruction; `ProcessIsRecorded` (`procstart.go:17`)
  already does this for peer Stop. `reapOrphanRuntime` also kills only the leader, leaving the
  setpgid group's tools/MCP children alive (FS-01.R21 "actually gone"). Corroborate start time at
  every reaper/reconciler, kill the group, and test reused-PID and descendant cases.
- **Worth fixing** — ASRC-4 — **The fresh-launch registration window is unfenced** (INV §4/§5/§9).
  `registry.go:173–200`, `chat.go:458–505`, `launch.go:70`: an adapter exit after the handshake but
  before `Registry.Launch` commits runs `handleAgentExit` (dropping the sentinel and tearing down
  registration); `Start` then writes running/idle and Launch re-inserts the dead runtime. Launch
  returns success, `Owns` stays true with no generation, and resume fails `ErrAlreadyStarted` until a
  manual Stop. Delegated probe `/tmp/asrc/probe_test.go` (injected kill in the opener) confirms.
  Manual `handleLaunch` also takes no lifecycle claim, so shutdown quiescence and a Stop during the
  handshake can miss or tear down an in-flight launch (TS-01.R16). Fence commit against exit and put
  manual launch behind the same claim; regress exit-in-window and shutdown-during-launch.
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
- **Worth fixing** — ASRC-7 — **Restart reconciliation disagrees with FS-01.R20** (INV §2/§10).
  `runtime/reconcile.go:32–35` writes `error`/"stale session reconciled on startup"; FS-01.R20
  says `done`/"process exited", which `pruneStaleRunning` writes. Pick one outcome and share it.

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

Five Must-fix findings remain on this unit. Each fix extends existing transcript, lifecycle,
receipt or reconciliation seams; the hardest fixes coordinate bounded concurrency and durable
publication across those seams. Audit complete; no product code or specifications changed.
Tree clean at entry. Source mode: repository source at
`c12c425109a7bdd5682451f02afe958477319e7a`; release lock pins Claude ACP 0.85.1 / Agent SDK
0.3.286, Codex ACP 2.1.1 / CLI 0.159.3. No credentialed provider run was performed.

- **Must fix** — ASRD-1 — **Independent writers reuse transcript sequences across history operations**
  (INV §1/§2/§5/§9/§11/§15). `internal/server/primer.go:65–81`, `switch.go:277–281`,
  `sessions.go:347–398`, `internal/runtime/chat.go:2305–2318`: a cross-backend switch starts
  the target's long-lived writer, then opens another writer for its switch marker. The target's
  sequence cursor never observes that append. A successful Claude → Codex switch followed by one
  prompt produces both `backend_switch` and `user_text` at sequence 2. Independently, two inactive
  annotation requests, or annotation versus Resume, can both open at the same next sequence;
  writer mutexes are instance-local. This breaks TS-02.R8's monotonic identity, FS-01.R14's
  continuous history and FS-13.R5/R10's annotation replay/search. Route active markers through the
  runtime's serialized allocator and fence inactive appends against one another and lifecycle
  transitions using existing ownership seams. Also preserve truthful switch failure handling:
  marker append failure currently returns 500 after the target is already active with no rollback,
  and deferred `Close` ignores its sync error. Regress unique increasing sequences through both
  handoff directions → first prompt, concurrent inactive annotations and annotation → Resume,
  plus marker append/sync failure after activation. Evidence: the independent fake-ACP overlay
  `/tmp/asr07_overlay.json` fails `TestASR07SwitchMarkerSequenceCollision` on the duplicate;
  `/tmp/transcript_race_overlay.json` confirms two independent writers append the same sequence.
- **Must fix** — ASRD-2 — **Concurrent live events are persisted and published out of sequence**
  (INV §2/§5/§11/§15). `internal/runtime/chat.go:2080–2100,2123–2128,2158–2175`:
  sequence allocation/in-memory insertion release `as.mu` before persistence and publication.
  An annotation of a busy source sent to another agent can overtake ACP output, even with one
  active writer: event N+1 may be durable/published before event N. Reader/context turn boundaries
  and live/replayed ordering then disagree with TS-02.R8 and FS-03.R8/R13. The temporary scheduled
  writer probe records `2 (annotation), 1 (assistant_text)` in the durable file. Serialize sequence
  allocation, append and ordered publication through one existing event seam, including special
  annotation/notice append paths, without holding a mutex across reentrant provider callbacks.
  Regress simultaneous ACP output, annotation and permission/notice events with barriers and assert
  increasing durable and observed order, not merely race-detector success.
- **Must fix** — ASRD-3 — **Failed transcript writes still become live conversation history**
  (INV §7/§9/§15). `internal/runtime/chat.go:2097–2129`: only a required user prompt is
  withdrawn when `persistEvent` fails; assistant text, tools and other ordinary events still reach
  the hub and external sink. A disk/write failure therefore shows output that disappears on reload,
  archive or later history handoff, contrary to FS-03.R13/TS-02.R8. An injected failed file append
  returns `persisted=false`, leaves zero durable bytes, and still delivers `assistant_text` to the
  subscriber. Make failure explicit and prevent unretained events from masquerading as durable
  history; keep recovery/status publication bounded and avoid recursively using the failing writer.
  Regress append and sync failures against both live subscribers and durable replay. Evidence for
  ASRD-2/3: `/tmp/asrd_persist_overlay.json`, `TestASRDConcurrentEmitReordersDurableEvents` and
  `TestASRDAppendFailureStillPublishes`; both reproduce the defects.
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
- **Must fix** — ASRD-5 — **Reconciliation reads arbitrary session artifacts as transcript truth**
  (INV §1/§7/§8/§16). `internal/server/reconcile.go:87–100,118–165`: the sweep recursively
  admits every file under an `a_*` session directory, including attachments, then `os.ReadFile`
  and `strings.Split` retain its whole contents; assistant deltas also accumulate without a bound
  before the final 120-rune clip. Large attachments/long conversations incur repeated unbounded
  reads every sweep. More directly, an uploaded NDJSON log containing an `assistant_text` record
  can replace a stale card preview with unrelated file content. The isolated probe confirms exactly
  that false preview. TS-01.R8 restricts fallback to Chuck's normalized log; TS-02.R1/R8 preserve
  that authority. Restrict candidates to the authoritative transcript path and use bounded streaming
  retention for the preview, preserving stale-generation guards and reporting real read failures.
  Regress unrelated artifacts, long streamed replies and bounded reads. Evidence:
  `/tmp/asrd_annotation_overlay.json`, `TestASRDReconcileAcceptsAttachmentAsTranscript`.

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
