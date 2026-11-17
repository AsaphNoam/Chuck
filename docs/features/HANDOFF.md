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

### TAB-BLANK — Chrome displays only the theme background on tab return (investigated 2026-11-17) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “sometimes the page just goes blank when I switch between tabs and go back,
click anywhere on the page makes the view render”. The operator confirmed Chrome browser tabs and
the newest Chuck build: “the Chuck theme color is still active but other than that the screen goes
completely blank”. No route, incident time, exact field build or browser error was supplied.

**Cause undetermined.** The shell renders a header separately from its route outlet
(`ui/src/App.tsx:8–15`); route loading and caught render errors show text or a retry, not a blank
shell. Desktop stream reconnect refreshes data rather than remounting the shell (TS-03.R7–R9).
The theme marker lives on `<html>` (`ui/src/features/appearance/appearance.ts:17–24`), so the
remaining color doesn't prove the React tree is still mounted. A click-triggered browser repaint is
plausible, but no evidence establishes where this failure originates. Source checked at `4d5bce32`;
isolated browser checks used packaged v0.24.0 (`4d511e1d`), Chrome `157.0.8103.79` and macOS `15.7.7`
on arm64. Dashboard and Settings kept their shell, content and stable sampled header pixels in
the automation, with no page error. Automation could not verify real hidden-to-visible transitions;
native screen verification was blocked by the locked Mac. No failing reproduction or skipped test.

- **Worth fixing** (confirmed from code) — TAB-BLANK-1, missing browser diagnostics: the desktop
  entry point (`ui/src/main.tsx:12–21`) has no browser error reporting, and the route boundary
  (`ui/src/components/ErrorBoundary.tsx:22–24`) only logs to the browser console. Ordinary server
  logs cannot tell whether a foreground return lost the application shell, hit a JavaScript error,
  or left a healthy DOM that Chrome didn't paint. At the desktop bootstrap and error boundary,
  report a bounded, deduplicated, visit-correlated event to Chuck's local diagnostic log for an
  unexpected browser error or a previously mounted shell missing on foreground return (INV §22;
  TS-01.R15). Allowlist the reason, route kind and mount state; omit console text, paths, and page
  contents. Verify with an injected render failure and shell loss, and that healthy returns stay
  quiet. The page cannot attest to pixels on screen; an in-incident screenshot taken before a
  click, with the approximate time and page, is the next useful field evidence.

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

### AUTH-TRUST — a person's authorization reaches the agent only as tool output (investigated 2026-11-16 at `3f1700d3`) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “Bug: Chuck’s user authorization is not recognized by automatic execution
review. Reproduction: A user approves scoped tests or instructs the agent to push branches/open
PRs. Chuck delivers that response inside get_assigned_task → Human continuation input. The agent
then calls create_task to perform the action. Actual: Automatic review rejects the call, saying
“the transcript contains no trusted user message explicitly authorizing” it. No child task is
created. Repeating approval through the same pipeline channel does not resolve it. Impact: Blocks
fixture tests, Git pushes/PR creation and keystore retrieval despite explicit user direction.”
No Chuck version, provider, run/agent identity or logs were supplied.

**Diagnosis.** The refusal text exists nowhere in Chuck; the reviewer is the agent CLI's own
automatic approval review, which runs before the `create_task` call reaches Chuck (which provider:
undetermined). **Confirmed from code and history:** through v0.23.0 a person's Continue text was
written only into the assignment's `Human continuation input:` section
(`v0.23.0:internal/pipeline/actions.go:156,166`; `v0.20.0:…:83`), which the agent reads as the
`get_assigned_task` tool result, while the turn's actual prompt was the fixed payload-free
instruction (`internal/runtime/activation_kinds.go:57–61`). A reviewer that trusts only user turns
could never see the approval, and repeating it used the same path. `0a8bda69` (2026-11-13, first
released in v0.24.0) moved the person's text into the continuation turn's own prompt
(`internal/pipeline/actions.go:16–24,138–152`, `internal/runtime/message.go:79–97`,
`internal/runtime/chat.go:1247–1250`); `TestConversationAnswerContinuesQuestionPause` and
`TestQueuedInputSurvivesStartRetry` cover that delivery. **Probable:** the report is the
pre-v0.24.0 behavior — the managed runtime was v0.20.0 at the last audit — and v0.24.0 removes it
for run-page and conversation Continue. Not reproduced against a live provider reviewer; no
skipped test was added because the reported path no longer fails at HEAD. First step for whoever
picks this up: install v0.24.0 and repeat the report's approval through Continue.

- **Worth fixing** (spec gap; confirmed from code) — AUTH-TRUST-1: person-authored work still
  reaches an already-running agent only as tool output. A task a person creates for an existing
  agent (`internal/server/task_handlers.go:69`) and every later stage of a run on its standing
  owner start through the payload-free activation (`internal/server/task_dispatcher.go:426`), so
  the instruction, run goal and stage inputs arrive only through `get_assigned_task`. FS-16.R26
  requires that; FS-00.R15 calls an assignment “a durable bounded prompt the recipient reads as
  ordinary conversation”. No requirement says which person-authored text must cross as a user
  turn, so the same reviewer refusal remains reachable when the authorization lives in a task
  instruction or run goal. Needs a §3 security decision before any change: whether, and for which
  authors, Chuck places person-authored instructions in the prompt; agent-authored text must never
  gain that standing. Then FS-16.R26/FS-00.R15 via §10, the activation seam, and a test that a
  person-created task's instruction is in the `session/prompt` content while an agent-created
  one's is not.
- **Worth fixing** (confirmed from code) — AUTH-TRUST-2: `Human continuation input:`
  (`internal/pipeline/assignment.go:112–114`) no longer carries anything a person wrote. Its
  callers now pass Chuck's resume notice, recovery evidence, the replacement notice, and the
  owner agent's own `decision.Reason` (`internal/pipeline/decisions.go:78`), so agent-authored
  text is presented to the next agent as human input. INV §21. Fix: rename the heading to what it
  holds (why this visit exists) and assert in `assignment_test.go` that no caller-supplied text
  appears under a heading naming a human.
- **Worth fixing** (undetermined) — AUTH-TRUST-3: the preamble that now accompanies the person's
  text in the prompt says it “does not approve a permission request or a pipeline checkpoint”
  (`internal/runtime/message.go:77`). Whether a provider reviewer reads that sentence as negating
  an explicit “push the branch and open the PR” is unknown. Evidence needed: on v0.24.0, with the
  provider's automatic review on, approve a push through Continue and record whether
  `create_task` is allowed. Reword only if that fails. The refusal happens inside the provider
  CLI, so Chuck has no log line for it; the agent's transcript is the only trace.


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
