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

- [Simplify pipeline configuration](../ready-changes/simplify-pipeline-configuration.md) — in progress.
  Both Go variants, the complete UI suite and the embedded build passed. The isolated browser
  journeys covered authoring and launching five stages, required handoff serialization, existing
  template and exact proposal round trips, configured launch defaults, custom names, validation,
  and shared-checkout confirmation on desktop and phone. The short desktop viewport keeps the
  focused stage field clear of the action footer. Close the specifications and queue the review.

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

- **Match sent prompts by request ID** — reviewed 2026-11-15, `fb245d40` (FS-03.R118/A93,
  TS-08.R163). PROMPT-1–4 are under **Review findings** and wait for `/fix`.

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

### Provider Refresh version observation (intermittent test, found 2026-11-15) — **Fix model:** medium — Codex Terra or Claude Opus.

- **Worth fixing** — PROVIDER-REFRESH-1. `TestRefreshProviderObservesVersionWithoutExecOnRead`
  (`internal/server/provider_refresh_test.go:88`) returned HTTP 200 with empty Version and CheckedAt
  in the full SQLite FTS suite. The test took 2.06 s, close to the version probe's 2 s bound.
  Five focused SQLite FTS runs passed. Check whether the marker process missed its deadline or
  inherited competing probe work, and make the fixture distinguish correct timeout behavior from
  a failed fast probe. The input-delivery fix did not change the probe or this fixture
  (FS-09.A38/A43, TS-04.R72, INV §12, §17, §20).

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

Prepared notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).
Latest published notes: [v0.23.0](../archive/state/RELEASE-v0.23.0-notes.md).
