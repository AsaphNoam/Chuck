# Handoff history settled through v0.18.0 — 2026-10-20

Release-era implementation and planning history. Open findings, queues and owed gates remain
in the [live handoff](../../features/HANDOFF.md).

## Changelog

- **2026-10-20 — Mapped the seventh core reliability audit; planning complete:** two focused
  GPT-6 Luna research passes and parent integration produced
  [CR-A7](CORE-RELIABILITY-REVIEW-A7.md), one frontier-sized review of sustained capacity and cost.
  It maps actual retrieval scans, retained queries, view fan-out, bounded control/dispatch work and
  ephemeral resource ownership into representative measurement and cross-agent intervention;
  ASR-I keeps its matched native-comparison design and existing ASW/ASF finding owners remain.
  All seven charters now have waiting review units. No audit, workload measurement, failure probe,
  product repair or release architecture receipt was performed. Independent checks confirmed the
  retrieval and runtime lifetime maps. Spec, link/source/test-reference (22 named tests), table
  shape and diff checks passed; product tests were not run for this planning-only change.

- **2026-10-20 — Mapped the sixth core reliability audit:** two focused GPT-6 Luna research passes
  and parent integration produced [CR-A6](CORE-RELIABILITY-REVIEW-A6.md), one frontier-sized review
  of operator actions through trustworthy observation. It maps exact input/targets/guards,
  acceptance versus completion/unknown effects, bulk outcomes, CLI output and live/reconnected
  desktop/phone state, preserving existing proof limits and ASW/ASF owners. Added one waiting
  review unit; A7 remains unmapped. No audit, failure probe, product repair or release architecture
  receipt was performed. An independent charter check clarified current phone scope and bounded
  permission-history coverage. Specification, local-link, source-path, 20 named-test reference and
  whitespace checks passed; product tests were not run for this planning-only change.

- **2026-10-20 — Mapped the fifth core reliability audit:** two focused GPT-6 Luna research passes
  and parent integration produced [CR-A5](CORE-RELIABILITY-REVIEW-A5.md), one frontier-sized review
  of result authority through actual next work. It maps task acceptance/cleanup/progression,
  current pipeline controls, concurrent openings, judge publication/retained projection and later
  bound stage acceptance, preserving existing ASW/ASF owners. Added one waiting review unit;
  A6–A7 remain unmapped. Independent charter checking sharpened task-cleanup versus room-source
  authority. Spec, link/source/test-reference (23 named tests) and diff checks pass; no product
  tests were run. No audit, failure probe, product repair or release architecture receipt was performed.

- **2026-10-20 — Mapped the fourth core reliability audit:** two focused GPT-6 Luna research passes
  and parent integration produced [CR-A4](CORE-RELIABILITY-REVIEW-A4.md), one frontier-sized review
  of faithful information delivery and operating guidance. It maps selected sources/bytes, access,
  actual annotation/task/mail consumers, bounded receipts, managed knowledge/awareness and requested
  versus native adoption evidence, preserving existing ASW/ASF owners. Added one waiting review
  unit; A5–A7 remain unmapped. Independent charter checking added persona-selection/preservation
  requirements. Spec, link/source/test-reference (24 named tests) and diff checks pass; no product
  tests were run. No audit, failure probe, product repair or release architecture receipt was performed.

- **2026-10-20 — Closed ASW-1 conversation event commits (all nine findings fixed):** ASW1-5 (INV §7/§9) — journal open advances
  allocation past a persisted `ReservedSeq` and a degraded clean close keeps it;
  `TestJournalReservedSequenceSurvivesCloseAndRestart` fails before the fix. ASW1-8 (INV §10) —
  TS-01.R39 now states the shipped inactive-writer fence (journal lock plus live-writer refusal).
  ASW1-3 (INV §4) — `openPersistence` publishes the writer attachment right after `Journal.Open`
  and closes it itself when exit teardown already ran; `TestExitDuringPersistenceOpenClosesJournal`
  fails before the fix. ASW1-1 (INV §5) — a held successor refused by history admission is put
  back as the hold (never over a newer one) instead of failing unseen;
  `TestHeldSurvivesHistoryRefusal` fails before the fix. ASW1-6/7 (INV §5/§16) — the control
  queue now belongs to the conversation's journal (one FIFO across generations; a busy queue
  blocks reclaim), and a generation cancelled while the queue is full is shed so the bound holds
  and its exit teardown settles cleanup; `TestControlOrderSpansGenerationReplacement` and
  `TestCancelledControlRemainsBounded` (focused `-race` ×3 passes; runtime/server suites pass).
  ASW1-9 (INV §1/§5) — turn-end cleanup carries the observation's own generation and turn; the
  task-owned runtime stop is generation-scoped, never stops a newer generation, and at a turn
  boundary stops only while idle or settling that exact turn (`StopGuardedIdle` gained the ending
  turn); person/cancel/retry paths keep R19's immediate stop. `TestStaleTurnEndCannotReleaseCurrentTask`
  (old generation and old same-generation turn) fails before the fix; task/pipeline tests ×4 pass.
  Not added: a reentrant-Stop regression (cleanup stays off the drainer, unchanged); the pipeline
  stage lookup still picks the latest stage, but only drives a state-derived `Reconcile`.
  ASW1-2 (INV §8/§10, delegated, reviewed) — the server holds an uncommitted, still-pending
  permission request as a live generation-scoped projection (`held_permissions.go`): windowed
  `pending_permission`, full-read `unretained_permissions`, desktop re-read on a notification with
  no loaded request; resolution/exit clears it; TS-01 §5 and TS-03.R73 updated.
  `TestUnsavedPermissionStaysApprovable` and an `sse.test.ts` case fail before the fix. ASW1-4
  (INV §5/§15, delegated, reviewed) — an attempt's failure is retained in memory until its
  idempotent transition commits, fences finalization, keeps its first reason and is retried by the
  room sweep; `TestThinkTankFailedSettlementRetainsCaptureFailure` (terminal and sweep retries)
  failed before the fix on its branch. Residual: that retained failure is memory-only (restart
  recovery still fences the attempt with the generic reason). Not checked: rendered desktop/phone
  approval prompt for an unsaved request. Closure: `GOFLAGS=-p=1 make test`, `make build`,
  `ui` `npm test` (980) and `npm run build` pass; one earlier full server run failed on an
  unnamed test and two reruns plus task/pipeline ×4 passed.

- **2026-10-20 — Mapped the third core reliability audit:** two focused GPT-6 Luna research passes
  and parent integration produced [CR-A3](CORE-RELIABILITY-REVIEW-A3.md), one frontier-sized review
  of workspace and executable configuration. It maps checkout ownership/setup, resources,
  fresh/frozen selection, current pipeline-to-task callers and real native execution evidence;
  setup warning semantics, deliberate deletion and ASW/ASF ownership remain explicit. Added one
  waiting review unit; A4–A7 remain unmapped. Independent charter checking found no material
  correction. Spec, link/source/test-reference (23 named tests) and diff checks pass; no product
  tests were run. No audit, failure probe, product repair or release architecture receipt was performed.

- **2026-10-20 — Mapped the second core reliability audit:** two focused GPT-6 Luna research
  passes and parent integration produced [CR-A2](CORE-RELIABILITY-REVIEW-A2.md), one frontier-sized
  control-plane review. The charter maps work readiness, assignment/runtime claims, wait/yield,
  distinct mail/dependency/quota policies, actual startup and retained cleanup; it records existing
  proof limits and ASW/ASF ownership. Added one waiting review unit; A3–A7 remain unmapped.
  Independent charter checking clarified source-selection authorization and actual startup order.
  Spec, link/source/test-reference (24 named tests) and diff checks pass; no product tests were run.
  No audit, failure probe, product repair or release architecture receipt was performed.

- **2026-10-20 — Mapped the first core reliability audit:** two focused GPT-6 Luna research passes
  and parent integration produced [CR-A1](CORE-RELIABILITY-REVIEW-A1.md), one frontier-sized review
  of durable conversation/session truth and maintenance recovery. The charter maps authority,
  actual archive/Restore/reconcile/reindex boundaries, requirements, existing tests, interruption
  questions and cross-unit owners. Added one waiting review unit; A2–A7 remain unmapped. No audit,
  failure probe, product repair or release architecture receipt was performed. Independent charter
  checking added the separate journal-health recovery authority. Spec, link/source/test-reference
  (30 named tests) and diff checks pass; existing product tests were researched, not rerun.

- **2026-10-20 — Created architectural-review and enforced the no-loop release gate:** after
  committing the testing skill, two Luna researchers supplied scenario/risk-based review practice
  and gate design. Twinned skills and workflow §18 center Chuck's purpose as a reliable development
  hub; TS-06.R36 and release routing/CI require one completed review. Exact verified finding-fix
  commits close that original receipt without a new architectural review. Eight real-Git cases,
  independent Luna implementation review and a synthetic startup-liveness forward test verify
  behavior; skill/twin/mutation, syntax/spec/reference and diff checks pass. No product fix or real
  audit/release review ran. The seven-slice audit plan is recorded/deferred for human expansion.
  [Research/evidence](../archive/reviews/workflow-skills-2026-10-20.md). Four unrelated dirty ready-change
  files remain outside the commits.

- **2026-10-20 — Created purposeful test-authoring companion:** two Luna research slices informed
  twinned `write-tests` skills, TS-06.R35, workflow §17 and conditional work/fix routing. Each test
  must add meaningful confidence worth its cost; no quotas, exhaustive matrices or duplicate
  layers. Review checks evidence value. Independent Luna forward-testing added one useful failing
  repair/Resume regression and no prose-only test. Skill validation, launcher mutation/twin checks,
  spec/reference and diff checks pass. [Research/evidence](../archive/reviews/workflow-skills-2026-10-20.md).
  This is the first sequential checkpoint; architecture skill is next, audits remain deferred.

- **2026-10-20 — Fixed and closed ASW-3 process/terminal ownership (ASW3-1–6; ASRB-1 and ASRC-3
  remainders):** ASW3-6 (INV §9/§12/§17): `ClassifyProcess` requires the exact microsecond birth
  (slack removed). ASW3-4 (INV §4/§9/§10): `LeaderlessGroup` makes `StopRecordedProcess` refuse
  and `ReconcileStale` retain a row whose leader is gone while its group has members. ASW3-1
  (INV §4/§5/§9/§15): terminal Stop propagates the refusal and keeps view, row, runtime agent and
  `Registry` claim (restored on `ErrProcessUnverified`); server Stop/Switch/Archive map it to 409.
  ASW3-2 (INV §1/§4/§7/§9): the external-view poll publishes only a confirmed exit; unknown proof
  stays owned. ASW3-5 (INV §4/§5/§9): xterm Cancel/Stop re-prove the exact birth and never signal a
  reaped child; xterm Stop now uses the shared group sweep. ASW3-3 (INV §1/§11/§15): `_post.sh`
  retries a 401 SessionStart within ~3 s, so a fast CLI's native ID reaches the row written after
  launch. Specs: FS-01.R54, TS-02.R54 implementation. Regressions: `TestClassifyProcessRejectsNearBirthMismatch`,
  `TestStopRecordedProcessRefusesLeaderlessGroup`, terminal `ownership_test.go` (refused Stop and
  retry on tmux/iTerm2 fakes, unknown → recovered → exit, reaped/stale xterm Cancel), hooks
  `TestSessionStartRetriesUntilAdmitted`; the terminal Stop and xterm Cancel tests fail on the
  previous code. Closure: serial `make test`, `make build`, focused `-race` (terminal ×3, proof
  helpers). Still open: FS-07.A12 real tmux/iTerm2/CLI receipts, including early SessionStart →
  Stop → Resume.

- **2026-10-20 — Closed ASW-2 by human decision on ASW2-1/5 (INV §1/§2/§11/§17):** the human
  accepted both recommendations. ASW2-1: legacy unbound/explicit-switch rows keep taking the
  catalogue's current provider id at first continuation (then frozen); TS-02.R53 and TS-07.R19
  amended, `TestContractLegacySnapshots` (unbound and switched cases) pins it. ASW2-5: a Claude fork load keeps sending
  the source's resume layers because the pinned adapter keeps no system prompt in the conversation;
  FS-01.R53 and TS-04.R65/R101 amended; `TestClaudeForkLoadCarriesSourceResumeInstructions` pins
  it. No product code changed. The originating PROMPT-1, PROMPT-2, ASRA-3 and ASRG-4 remainders
  close with the ASW-2 fixes. ASW-2 is closed; native clone/check/adoption and unpatched
  idle-Steer race receipts remain unclaimed live evidence.

- **2026-10-20 — Simplified workflow proposals and scoped the next reliability audits:** added
  concrete plain-language examples to the [meta-analysis](AGENT-SURFACE-META-ANALYSIS.md) and a
  [deferred core audit plan](CORE-RELIABILITY-AUDIT-PLAN.md). One focused GPT-6 Luna scope reader
  challenged duplicate reviews; parent checked current architecture/fixes and prioritized preserved
  work, context/message fidelity, correct execution and measured long-history overhead. Six core
  responsibilities map to two fresh investigations and one combined post-fix journey, not six
  repeated source audits. META-1 stays closed; updater/phone work is secondary under the human's
  priorities. No product/spec/workflow/skill edits or new executable review units. Documentation,
  reference and diff checks pass; unrelated dirty ready-change files are preserved.

- **2026-10-20 — Fixed ASW2-7 (INV §2/§12/§17):** Codex ancestor-instruction discovery now
  takes `project_root_markers` from the effective pre-project layers with native precedence (system
  config < user < the child's `CODEX_CONFIG` overlay < home/system managed config); a malformed
  layer fails the launch closed. TS-04.R101 note updated. Regressions:
  `TestCodexDeveloperInstructionsProjectChain/effective_project_root_markers`,
  `TestCodexOverlayRootMarkersPreserveAncestorInstructions`. ASW2-1 and ASW2-5 are left open
  under Blocked on human (compatibility decision; requirement conflicts with Claude adapter
  behavior). Closure for ASW2-2–4/6–11: serial `make test` (untagged and FTS5) and `make build`
  pass; focused runtime `-race` passes. Four pre-existing dirty ready-change files remain outside.

- **2026-10-20 — Fixed ASW2-4 (INV §2/§10/§12/§17):** Codex quota-check isolation now refuses
  (`unsupported`) when the check home holds a nonblank `AGENTS.md`/`AGENTS.override.md` or any
  native layer sets the base `instructions` key; the overlay's own `instructions` is dropped.
  Routing/auth and the shared profile are untouched. TS-04.R101 note updated;
  `TestCodexCheckIsolationSuppressesWorkInstructions` covers both sources. A user with a global
  `AGENTS.md` now gets manual quota continuation for Codex rather than an unisolated check.

- **2026-10-20 — Fixed ASW2-8/9/10 (INV §2/§10/§11/§16/§17):** the execution manifest now
  records a `native_instructions` layer for the Codex text actually merged at process start
  (`codexLaunchOverlay`, captured by `startCmd`) and `artifact_sha256`, the managed adapter
  entrypoint digest Steer's proof already compares (`backend.ManagedArtifactDigest`; empty for a
  PATH adapter). Terminal starts get a manifest through the shared metadata builder with adapter
  identity unknown. Every text field clips to 128 bytes and an over-16 KiB record drops its layers
  with `truncated`. TS-11.R24 note updated. Regressions in `internal/runtime/manifest_test.go`.

- **2026-10-20 — Fixed ASW2-2/3/6/11 (INV §1/§2/§3/§11/§17):** ordinary continuation refuses a
  frozen or legacy setup naming a profile (`422 source_invalid`) before registration or switch
  teardown; the stopped-agent quota check probes `continuedContract`'s provider model, so a
  catalogue edit no longer splits check from Resume; a clone keeps its source's frozen permission
  choice; the runtime records negotiated fast in the contract before metadata persistence.
  TS-02.R53/TS-07.R19 notes updated. `TestQuotaCheckFollowsInterruptedSelection` encoded the old
  catalogue-following check; it now proves a catalogue edit keeps the check valid and an effort
  change still forces manual handling. New regressions: `TestLegacyProfileRefusesBeforeRegistration`,
  `TestStoppedQuotaCheckUsesFrozenModel`, `TestForkKeepsSourcePermissions`,
  `TestContractRecordsAppliedFast`.

- **2026-10-20 — Fixed META-1, closing the META unit (INV §1/§9):** offline `chuck reindex` now
  reads the SQLite-only session fields (hook-learned native ID, live effort/fast) before the wipe and
  restores them after each agent's replay, including a failed replay; a read failure leaves the
  archive untouched. The inventory found effort/fast had the same loss as the native ID; every other
  `sessions` column is replayed from transcript metadata. TS-02.R9 records the rule;
  `TestReindexRetainsSQLiteOnlySessionFields` covers the review's hook → teardown → reindex path,
  a legacy chat row and a failed replay, and fails without the fix. Archive repair, updater, remote
  revocation and retrieval performance remain unconfirmed audit candidates. Serial `make test`:
  untagged variant passed in full; the FTS5 variant passed except
  `internal/cli` `TestBootstrapContenderExitsDuringDownload` (its 1 s wait for the installer's fake
  download to start expired; unrelated installer code, now a known flake — 30 isolated reruns pass).
  `make build` passed. Four pre-existing dirty ready-change files remain outside this commit.

- **2026-10-20 — Researched defect causes, review escapes and wider audit value:** six GPT-6 Luna
  discovery slices covered history, refactors, workflow, test oracles and backend/client surfaces;
  parent verified representative diffs and a new real-state reindex probe. The
  [report](AGENT-SURFACE-META-ANALYSIS.md) distinguishes design, implementation and integration drift,
  proposes four bounded workflow changes and ranks wider audits with explicit confidence limits.
  Added confirmed META-1 and retained its reproducer; both SQLite variants intentionally fail the
  preservation assertion. No product/spec/skill policy changed and no future audit selected.
  Documentation/spec-reference and diff checks pass; four pre-existing dirty ready-change files
  remain outside this commit.

- **2026-10-20 — Assessed completed refactors before fix queueing:** compared all three ASW
  reviews with remaining ASF batches and current source. Added ASW1-9 (terminal cleanup loses
  execution identity) and ASW3-6 (near birth mismatch authorizes control), both confirmed by
  intentionally failing isolated overlay probes (INV §1/§2/§5/§9/§11/§12/§15/§17).
  [The assessment](AGENT-SURFACE-POST-REFACTOR-ASSESSMENT.md) records bounded shared-seam repairs,
  integration constraints and joint regressions; no fourth broad refactor or product fix selected.
  Updated stale ASW review status in the unit index. No provider/browser/real-driver run.
  Documentation/spec-reference checks and diff checks pass; the four pre-existing dirty
  ready-change files are preserved outside this commit.

- **2026-10-20 — Reviewed ASW-2 native execution contract:** reviewed `43da6bec..9a15afa6`
  from isolated snapshots. Nine Must-fix and two Worth-fixing findings cover legacy model/setup
  continuation, stopped quota selection/isolation, Claude fork layers, clone permissions, effective
  root markers and manifest coverage/bounds/applied fast (INV §1/§2/§3/§7/§10/§11/§12/§15/§16/§17).
  Original ASRA-1/2 and ASRB-2/3/8 reproductions are corrected; other owned remainders stay with
  ASW-2. Existing focused checks pass both SQLite variants and `-race`; independent probes expose
  the gaps those tests miss. Product/spec files unchanged; native adoption/live gates remain owed.
- **2026-10-20 — Implemented ASW-2 native execution contract:** sessions freeze a versioned
  effective model/setup choice that every continuation and clone reuses; empty overrides reset to
  inheritance; unrealizable roots/profiles and undecidable legacy rows refuse before effects
  (INV §1/§2/§3/§11). Forks are prepared by purpose (Claude opens its fork by load, Codex keeps
  inherited instructions); quota checks blank Codex work instructions; Codex project layers follow
  the native root chain; Codex Steer requires the verified patched artifact; each process records a
  bounded requested-delivery manifest. Slices 4–6 were delegated and integrated.
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
