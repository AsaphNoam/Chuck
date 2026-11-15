# Chuck — Implementation handoff

Settled history through v0.23.0 preparation is archived in
[HANDOFF-through-2026-11-08](../archive/state/HANDOFF-through-2026-11-08.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archives are not live queues.

## Current position

- **Release:** operator confirmed v0.23.0 and authorized push on 2026-11-08. Main and the
  annotated v0.23.0 tag at `f3f34986` are pushed. Missing v0.22.0 baseline tag was pushed with
  explicit approval; its own workflow `42450809571` is cancelled. v0.23.0 retry `42450625734`
  passed architecture, assembly, transaction/bootstrap and fresh-install checks, then published
  [v0.23.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.23.0).
  [Notes](../archive/state/RELEASE-v0.23.0-notes.md) are published too. Archive/manifest/installer
  assets are present; manifest size and SHA-256 match GitHub's uploaded archive metadata.
- **Architecture:** combined source audit and all quota repairs are reused, as explicitly
  directed by the operator on 2026-11-08. No second architectural review.
  [Receipt](../archive/reviews/architecture/v0.22.0-next.json) includes release preparation.
  Original audit findings are closed.
- **Verification:** hosted release assembly and fresh installation pass. Main CI retry
  `42450626202` passes both Go variants, specs, gate tests and vet, but failed one of 1181 UI
  tests (waiting-input pane opening); that fixture race is fixed locally, unconfirmed by a new
  hosted run. Do not claim full clean-clone CI success. Native/provider/device gates in
  [ACCEPTANCE-GATES.md](ACCEPTANCE-GATES.md) remain owed.
- **Installed runtime:** managed current was v0.20.0 at audit; no install or running-dashboard
  update is part of this release preparation. Global PATH adapters remain unpatched.
- **Checks:** unset CHUCK_RUNTIME_ROOT; use GOFLAGS=-p=1 and a writable isolated GOCACHE.
  Full Go suites need loopback-capable execution.

## Active change

None.

## Decisions needing input

- Whether to standardize TS-03.R3–R4 mixed legacy error envelopes or TS-04.R3 provider model-ID
  ownership (compatibility change).
- Whether a pipeline stage paused after failed launch/resume continues withholding **Open agent**,
  matching restart recovery, or permits chat under a wider continuation contract.
- Whether a model or effort the adapter accepted without echoing it back should show as
  unconfirmed (CR-A10 lead 13; INV §20 reads missing evidence as unknown).

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

- **Give agent execution one owner** — reviewed 2026-11-15, `193f301e..1d6077fa` (TS-01.R45–R50,
  TS-02.R67–R68, TS-08.R168); findings EXEC-1 to EXEC-12 are open under **Review findings** and
  wait for `/fix`. Its ready change and plan stay until they close. Both implementer notes were
  confirmed: the room rule stays inside task admission's claiming compare-and-swap, and the
  gate-versus-row span is sound except on a failed end write (EXEC-2). Credentialed provider and
  device gates are not closed by it.

- **Single-owner turn state** — ready for review at `35f33a07` (TS-01.R44). Current-turn
  fields share one structure and the existing turn lock. Chat status writes share one
  transaction, and explicit Stop prevents late callbacks from overwriting `done`.
  The regression fails with the Stop guard removed and passed five consecutive runs
  with it restored. Both Go variants, the build, and focused race tests for cancellation,
  permission, Steer, held messages and Stop passed.

- **Shared desktop and phone input delivery** — ready for review (TS-08.R164). Both clients
  share send, Steer, retry, attachment completion, withdrawal and the recovery offer. Desktop
  projects the prompt into its transcript store; phone waits for sequenced events. The full
  UI suite (1261 tests), UI build and 390px Chromium harness passed. The harness uses stubbed
  responses. On reconnect, desktop refetch advances the transcript token and clears the
  reconciliation marker for open agents. Phone still uses its held query. Review the lifetime
  of browser retry IDs when another client settles the recovery offer.

- **Shared resume start and Switch rollback** — ready for review at `95d4f621` (TS-01.R43).
  Explicit resume, wakes and both Switch starts share registration cleanup and applied identity
  persistence. Switch stops a target whose final identity write failed before starting rollback.
  The build, specification checks, both Go variants and focused race checks passed. The first
  untagged run hit a pipeline Stop 409. Ten focused repetitions and the full rerun passed.
  Both new Switch persistence regressions fail against the original source.

- **One typed turn outcome** — ready for review. Runtime status, quota settlement, inline mail
  and event readers use the resolved outcome. See FS-03.R117/A92 and TS-01.R42.
  Both Go variants, the build, and focused race checks passed. The fork fixture now waits for the
  retained turn boundary and passed eight tagged repetitions. The known deferred-assignment
  lifecycle test failed once in the untagged matrix, then passed five focused runs and the full
  untagged rerun.

- **Match sent prompts by request ID** — ready for review. Desktop transcript reconciliation and
  held follow-up delivery use the request ID. See FS-03.R118/A93 and TS-08.R163.
  Verification on the implementation diff: `make check-specs`, UI suite (1,251 tests),
  `npm run build` and `git diff --check` passed.

**Core reliability audit units (added 2026-11-13):** explicitly requested subsystem reviews of the
complete current implementation and callers, not of a commit. Read the charter, refresh fix
ownership and record the reviewed HEAD. Intended for a frontier reviewer; the charters' leads are
questions, not findings.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A8](CORE-RELIABILITY-REVIEW-A8.md) | Pipeline pause, answer and recovery ownership: who can end each pause, refused input with no side effect, answers through replacement, stall detection | findings closed, including PIPE-CHAT-1 and PIPE-CHAT-3 |
| [CR-A9](CORE-RELIABILITY-REVIEW-A9.md) | Think Tank lifecycle and its quota join: fenced member launch, requested Pause/End under quota, restart recovery | findings closed at `0da7c581` |
| [CR-A10](CORE-RELIABILITY-REVIEW-A10.md) | Durable state transitions have one rule: status-writer inventory, unguarded and divergent writers, recommendation for a transition-rule design | reviewed at `8faec36f`; findings owned by [`give-agent-execution-one-owner.md`](../ready-changes/give-agent-execution-one-owner.md) |

CR-A9's result: eight findings, each fixable on its own. CR-A9-1 and -5 share the quota hold
release; CR-A9-2 and -3 share the rule for which holds a room may carry in `setup` and `ended`.

CR-A10's five findings and its transition-rule recommendation moved on 2026-11-14 into the
waiting change [`give-agent-execution-one-owner.md`](../ready-changes/give-agent-execution-one-owner.md)
(TS-01.R48, R49 and R50), together with the findings of the 2026-11-14 adversarial assessment.
They are not queued for `/fix`. The recorded text is in
[`CR-A10-findings.md`](../archive/reviews/adversarial-2026-11-14/CR-A10-findings.md).

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

### Give agent execution one owner (`193f301e..1d6077fa`) — reviewed 2026-11-15 at `1d6077fa`

**Fix model:** medium — Codex Terra or Claude Opus.

Second fix round: turn, quota and pipeline recovery state had findings fixed in the last 30 days
(`35f33a07`, CR-A8 `0a8bda69`, CR-A9 `0da7c581`), so EXEC-12 asks for an inventory (§7, §8).
The unit was meant to change no behavior; EXEC-1 to EXEC-5 are each a changed outcome, confirmed
against `193f301e`.

- **Must fix — EXEC-1: a prompt Stop overtakes stays `submitted`.** `runtime/chat.go:1229`
  (`beginTurn`'s provider goroutine returns when `canStartTurn` fails). Trigger: Stop or a crash
  lands after `beginTurn` succeeded and before the send, including while the goroutine waits on
  `cancelMu` behind a Cancel. The prompt's request and the carried queued assignment input are
  already `submitted` and nothing marks them; before, the failed `transport.Call` marked both
  `unknown`. The receipt misreports delivery (TS-01.R45, INV §19). Fix: on that return mark both
  request IDs `released` (never written) or `unknown`. Test: hold `cancelMu`, let a held
  successor begin, Stop; neither request is `submitted`.
- **Must fix — EXEC-2: a failed turn-end write strands the turn.** `runtime/chat.go:2779`
  (`endTurn` returns `ended && err == nil`) and `finishTurn` at `:1379`. Trigger: the end
  transaction fails (busy timeout, disk full). No `turn_end` is emitted, so task and room
  completion never run; the held successor stays held with the gate free, so a newer Send
  overtakes it; the row stays `running` and status `busy` on an idle live process, and
  `AdmitRuntimeTurn` refuses activations for the generation. Before, the write error was ignored
  and `turn_end` and the successor still ran (TS-01.R45, TS-02.R67, INV §5, §15). Fix: keep the
  gate and still emit `turn_end` and run the successor on a failed write, record the failure,
  and leave Stop and finalized fences as the only reasons for `!ended`. Test: fail the end
  write; `turn_end` is observed and the held message runs before a later Send.
- **Must fix — EXEC-3: a turn that completes as Stop arrives loses its `turn_end`.**
  `runtime/quota.go:204` (`completePromptTurn` checks `stopping`, then settles quota and inline
  mail) against `endTurn`'s `stopping` refusal at `chat.go:2785`. Trigger: Stop lands between the
  settlement claim and `finishTurn`. Mail is confirmed and quota settled as completed while the
  row records `process_ended` and no `turn_end` exists; before, `finishTurn` always emitted it
  (TS-01.R45). Fix: once `promptSettlement` is claimed, `endTurn` accepts `stopping` for that
  turn; Stop's later end is ignored by the `running` predicate. Test: block between the claim
  and `finishTurn`, Stop, assert outcome `completed` and one `turn_end`.
- **Must fix — EXEC-4: a lost compare-and-swap in pipeline startup recovery aborts server
  start.** `pipeline/manager.go:844` dropped `!errors.Is(recordErr, state.ErrPipelineConflict)`.
  Trigger: startup with a run whose retryable controller action fails to reconcile while the
  task dispatcher or room-phase sync, both already running, bumps the run's revision.
  `Server.Start` treats the error as fatal; before it logged "startup reconcile deferred"
  (TS-01.R49 allows re-deciding; `ReconcilePending` already does). Fix: restore the tolerance.
  Test: bump the revision between the failed reconcile and its record; `Startup` returns nil and
  the run stays due.
- **Must fix — EXEC-5: a transient owner-read error after a quota claim leaves the intent
  `claimed` until restart.** `server/quota_dispatch.go:112–121` now holds `owner_unavailable`
  only for `ErrNotFound` and otherwise returns. `claimed` is not in the due scan and only
  startup recovery reschedules it; the quota gate refuses the agent's mail and wakes meanwhile
  (TS-01.R49, INV §18). Fix: `DeferQuota(..., "store_failure")` for the other errors, as
  `deferQuotaPolicyFailure` does. Test: inject a non-NotFound error on the post-claim read; the
  intent becomes `deferred` and is admitted on the next pass.
- **Must fix — EXEC-6: four *Verify* clauses are not implemented.** (a) TS-01.R46's assertion
  helper — stored chat state equals the composition after runtime and server tests — does not
  exist; composing sites pass literal inputs (`chat.go:2769`, `:2794`, `:2799`, `:2802`, `:2826`,
  `reconcile.go:52`, `switch.go:318`), so nothing checks them against the turn row, and
  `TestChatStatusWritersUseComposition` scans three named files for string literals only.
  (b) TS-01.R45's table test of the begin function over every entry path is missing; origins are
  recorded only in a store-level test. (c) TS-01.R50's "no uncalled writer" inventory is missing
  and exported store writers without a production caller remain — `ResetTurnBudget`
  (`state/messages.go:464`, orphaned by this change), `CreateTask`, `InterruptTask`,
  `ObserveQuota`, `ConsumeTurnBudget` and others; `TestStatusTableWriterInventory` counts
  `UPDATE` statements per file. (d) TS-01.R47's table (`state/turn_admission_test.go:12`) runs
  every row on one fixture with duplicate rows, so it cannot detect a mis-transcribed outcome.
  Fix: add the harness helper in `newChatTest` and `testServer` cleanup; a runtime table over
  Send, held successor, Steer fallback, task first prompt, settled continuation and each
  activation kind reading the turn row; an inventory of exported `*Store` writers with no
  non-test caller, deleting or moving the leftovers to test helpers; one fixture per differing
  admission pair taken from the pre-change reads (TS-06.R35, INV §17).
- **Worth fixing — EXEC-7: a quota conflict reaches the person as a storage failure.**
  `state/quota_success.go:28` and `HoldQuotaManualTurn` now return `ErrQuotaConflict`;
  `server/quota.go:32`, `:52` pass it to the runtime, which sets `quotaBlocked` and shows "Quota
  recovery state could not be saved". Trigger: a manual turn ends while the due dispatcher or a
  quota Cancel bumps the revision (TS-01.R49: re-read and decide again). Fix: on conflict
  re-read and decide a bounded number of times in `settleQuotaTurn`.
- **Worth fixing — EXEC-8: `beginTurn`'s failure path writes to the store under `as.mu`.**
  `runtime/chat.go:1172–1189`. With one store connection and a five-second busy timeout every
  emit and notification for the agent waits behind it. Fix: choose the outcome under the lock,
  end the row unlocked, re-lock to release the gate.
- **Worth fixing — EXEC-9: a failed steer can leave a live idle agent `busy`.** Confirm first.
  `claimTurnLocked` (`chat.go:1040`) and `reserveHeldSuccessorLocked` now set the executing turn
  at claim, so `updateOwnedBusy` writes `busy` for a tool frame during an idle steer;
  `resolveSteerFallback` then releases the gate with no status write (TS-01.R46). Fix: require
  that the turn reached the provider, or release through the end path.
- **Worth fixing — EXEC-10: dead code.** `MarkProcessExited` (`runtime/reconcile.go:43`) has no
  caller and passes an empty generation, so it would not end the turn row; `applyTurnEndStatus`
  and `setExecTurn` serve tests only.
- **Worth fixing — EXEC-11: weak new tests.** `crash_permission_test.go:30` names an order it
  does not assert; `state_test.go:95` skips the row-shape check when the store is closed;
  `TestComposeChatStatusWriterInventory` has two identical rows ("launch", "resume ignores old
  generation").
- **Worth fixing — EXEC-12: inventory (§7).** EXEC-1, -2, -3 and -9 share one shape: a path
  leaves a claimed turn through an early return that assumes another owner settles it. List
  every exit from a claimed turn gate and every site that releases the gate, with who ends the
  row, emits `turn_end`, settles the request and runs the held successor on each, and where that
  rule is stated.

## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).
Latest published notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).
