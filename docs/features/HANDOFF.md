# Chuck — Implementation handoff

Settled history through v0.23.0 preparation is archived in
[HANDOFF-through-2026-11-08](../archive/state/HANDOFF-through-2026-11-08.md).
Settled review units in the v0.24.0 range are archived in
[HANDOFF-through-2026-11-16](../archive/state/HANDOFF-through-2026-11-16.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archives are not live queues.

## Current position

- **Release:** [v0.24.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.24.0) published
  on 2026-11-16 at `4d511e1d`. The local build and 1,293 UI tests passed.
  [Release CI](https://github.com/AsaphNoam/Chuck/actions/runs/43819855461) passed assembly,
  transaction/bootstrap and fresh installation. The published archive's size and SHA-256 match
  the manifest, and the installer is attached.
- **Architecture:** v0.24.0 shipped under the
  [v0.24.0-only waiver](../archive/state/RELEASE-v0.24.0-notes.md). The owed combined review of
  `v0.23.0..d26e5fe2` is now recorded in the
  [receipt](../archive/reviews/architecture/v0.23.0-next.json); it did not gate v0.24.0. EXEC-6c
  and ARCH-STALL-1 are open Must-fix items. The next release needs its own review and
  `v0.24.0-next.json`.
- **Verification:** [Main CI](https://github.com/AsaphNoam/Chuck/actions/runs/43819859225)
  was still running when release CI completed. The prior waiting-input UI fixture failure had a
  local repair. Native, provider and device gates in [ACCEPTANCE-GATES.md](ACCEPTANCE-GATES.md)
  remain owed.
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
  TS-02.R67–R68, TS-08.R168); the 2026-11-15 fix rounds left only the
  EXEC-6c remainder, blocked on a human decision under **Review findings**. Its ready change and plan stay until they close. Both implementer notes were
  confirmed: the room rule stays inside task admission's claiming compare-and-swap, and the
  gate-versus-row span is sound except on a failed end write (EXEC-2). Credentialed provider and
  device gates are not closed by it.

**Core reliability audit units (added 2026-11-13):** explicitly requested subsystem reviews of the
complete current implementation and callers, not of a commit. Read the charter, refresh fix
ownership and record the reviewed HEAD. Intended for a frontier reviewer; the charters' leads are
questions, not findings.

| Unit | Responsibility | Status |
|---|---|---|
| [CR-A10](CORE-RELIABILITY-REVIEW-A10.md) | Durable state transitions have one rule: status-writer inventory, unguarded and divergent writers, recommendation for a transition-rule design | reviewed at `8faec36f`; findings owned by [`give-agent-execution-one-owner.md`](../ready-changes/give-agent-execution-one-owner.md) |

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

### Give agent execution one owner (`193f301e..1d6077fa`) — reviewed 2026-11-15 at `1d6077fa` — **Fix model:** medium — Codex Terra or Claude Opus.

Fix rounds closed every finding except the one below. The inventory (EXEC-12) stays in
`docs/ideas.md` ("One exit rule for a claimed chat turn"), with the independent check's
same-cause concerns; on 2026-11-15 the operator declined a design pass for it while
TURN-STOP-1–3 were open. They are closed and the Stop inventory is the next `docs/ideas.md` entry.
The independent check is owed again only for the EXEC-6c repair.

- **Must fix** — EXEC-6c (remainder, blocked on human): 11 exported `*Store` writers are called
  only from other packages' tests — `AddThinkTankRecord`, `AttachTaskContext`,
  `BindAssignmentInput`, `BindPipelineStageTaskStandingAgent`, `ConsumeTurnBudget`,
  `FinalizeThinkTankAttempt`, `InsertPipelineAttempt`, `InterruptTask`,
  `MarkQuotaProviderStarted`, `ObserveQuota`, `StartQuotaCheck` (about 85 call sites, 30 for
  `ObserveQuota`). Each wraps an unexported state helper with no exported production writer of the
  same effect, so retargeting means driving each test through a longer production path, and a raw
  `DB().Exec` fixture would copy write logic that can drift (INV §17). They stay in the
  shrink-only `fixtureOnlyWriters` list (`internal/state/store_writer_callers_test.go`); the
  decision is under **Blocked on human** (TS-01.R50, INV §10, §17).

### Architectural review of `v0.23.0..d26e5fe2` (shipped as v0.24.0) — reviewed 2026-11-16 — **Fix model:** medium — Codex Terra or Claude Opus.

[Receipt](../archive/reviews/architecture/v0.23.0-next.json). Fixes append to that receipt and need
no second architectural review (workflow §8/§18). Pipeline pause/recovery (CR-A8) and chat turn
exits (EXEC, TURN-STOP, OUTCOME) both had fix rounds in the last 30 days, hence ARCH-STALL-2; the
turn-exit inventory already exists in `docs/ideas.md` ("One exit rule for a claimed chat turn")
and ARCH-TURN-1 is one more row for it.

- **Must fix** — ARCH-STALL-1: a pipeline stage owner whose assigned turn ends in a provider
  failure while its process stays alive is never continued or paused. The turn end writes status
  `error`; `nudgeStalledOwner` (`internal/server/pipeline_stall.go:28–31`) and the sweep's
  `check_stall` selection (`internal/state/pipeline_tasks.go:1092–1094`) both require `idle`, and
  no other recovery action matches a live owner in `error`. The task stays `running` and the run
  stays `running` with no attention reason until a person notices the agent. Trigger: any provider
  service, connection or request error on an owner turn; since `7e2539a3` also a `refusal` or
  unknown stop reason, which v0.23.0 left `idle` and nudged. FS-14.R101 ("a run cannot remain
  nominally active indefinitely"), TS-09.R70, INV §8. Reproduced:
  [probe](../archive/reviews/architecture/v0.23.0-next-owner-failure-repro.go.txt) ends with
  `owner status=error run=running attention="" task=running stall_turns=0 prompts=1`. Fix: make a
  settled failed turn of the assigned generation (not quota-blocked, not stopped) eligible for
  the same bounded continuation-then-pause in both the nudge and the sweep query, or pause the
  run with a cause; convert the probe to the regression and check the ordinary task assignee.
- **Worth fixing** — ARCH-STALL-2 (inventory): list every settled owner state (`idle`, `error`
  by outcome, `waiting_input`, `done`, quota-held, background work) against who continues, pauses
  or deliberately leaves the stage, for the turn-end nudge and
  `PipelineOwnersNeedingRecovery` (`internal/state/pipeline_tasks.go:1081`), and state the rule in
  TS-09.R70 (INV §8).
- **Worth fixing** — ARCH-TURN-1: a refused turn start stores outcome `refused`
  (`internal/runtime/chat.go:1192–1198`), the same value as a provider refusal, which
  `ComposeChatStatus` maps to `error` (`internal/state/chat_status.go:27`) while the stored status
  stays `idle`. Trigger: a mail wake whose waking message was already read, then a permission
  request resolved outside a turn recomposes the healthy agent to `error`
  (`internal/runtime/chat.go:2876–2885`). TS-01.R46, INV §5. Fix: a distinct not-started value that
  composes to `idle`; test a stale wake against the `statetest` compose helper.
- **Worth fixing** — ARCH-INPUT-1: after the adapter definitively refuses a desktop text-only
  Steer, pressing Steer again with the unchanged draft reuses the draft's request ID
  (`ui/src/components/chat/useInputDelivery.ts:70–74`), so the server replays the stored `failed`
  receipt (`internal/server/attachments.go:771–777`) without asking the adapter, and Send with
  the same text gets a content conflict; it works only after an edit. FS-03.R50, TS-08.R163,
  INV §19. Fix: rotate the draft request ID on a definite refusal; test refuse-then-retry.


## Blocked on human

- **File reader full UI closure:** permission requested through the asynchronous question to
  scope the pre-existing withdrawal test assertion to its exact error message (one test line).
  The four requested product changes are committed and independently reviewable. No unrelated
  test correction has been made without approval (repository instruction on separate fixes).
- **EXEC-6c test-only store writers (TS-01.R50):** choose (a) rewrite the ~85 test call sites of
  the 11 remaining writers to reach the same rows through production paths, then delete the
  writers (large, test-by-test); or (b) amend R50 so a writer kept for other packages' tests is
  allowed when named in the shrink-only `fixtureOnlyWriters` list, closing EXEC-6c as is.

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.24.0](../archive/state/RELEASE-v0.24.0-notes.md).
Latest published version: [v0.24.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.24.0).
