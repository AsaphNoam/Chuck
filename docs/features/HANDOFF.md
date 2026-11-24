# Chuck — Implementation handoff

Settled history through v0.23.0 preparation is archived in
[HANDOFF-through-2026-11-08](../archive/state/HANDOFF-through-2026-11-08.md).
Settled review units in the v0.24.0 range are archived in
[HANDOFF-through-2026-11-16](../archive/state/HANDOFF-through-2026-11-16.md).
What v0.25.0 settled is archived in
[HANDOFF-through-2026-11-22](../archive/state/HANDOFF-through-2026-11-22.md).
Use [AGENT-WORKFLOW.md](AGENT-WORKFLOW.md) for role rules; archives are not live queues.

## Current position

- **Release:** [v0.25.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.25.0) published
  on 2026-11-22 at `74699ab6` ([notes](../archive/state/RELEASE-v0.25.0-notes.md)).
  [Release CI](https://github.com/AsaphNoam/Chuck/actions/runs/44996541049) passed; the
  downloaded archive's size and SHA-256 match the manifest, and the installer is attached. The
  operator authorized shipping with QUOTA-DISMISS-1 open; it is fixed on main after v0.25.0.
- **Architecture:** the v0.25.0 review is complete in
  [v0.24.0-next.json](../archive/reviews/architecture/v0.24.0-next.json); the gate passed. The
  next release needs its own review (receipt `v0.25.0-next.json`).
- **Verification:** Native, provider and device gates in
  [ACCEPTANCE-GATES.md](ACCEPTANCE-GATES.md) remain owed.
- **Installed runtime:** managed current was v0.20.0 at audit; v0.25.0 is not installed locally.
  Global PATH adapters remain unpatched.
- **Checks:** unset CHUCK_RUNTIME_ROOT; use GOFLAGS=-p=1 and a writable isolated GOCACHE.
  Full Go suites need loopback-capable execution.
- **Public docs:** the README was rewritten for newcomers with screenshots and a tour video
  captured from the scripted demo fixture. Reference material moved to `docs/configuration.md`,
  `docs/development.md` and `docs/operator-cli.md`. The AgentDeck cutover guide and compatibility
  shims were retired 2026-11-22 (FS-10.R25, FS-13.R24, TS-08.R58).

## Active change

None.

## Decisions needing input

None.

## Resolved decisions

- Keep the current HTTP error envelopes and provider model-ID ownership. Revisit each
  compatibility contract when there is a concrete caller need and a migration plan.
- Keep **Open agent** hidden after a failed launch or resume. Explain the pause and offer
  **Retry stage**, consistent with restart recovery.
- Treat an unechoed model or effort request as **Unconfirmed** where confirmation evidence is
  available. The current launch path discards that distinction before the UI sees it, so the
  display change is deferred. The operator approved using existing evidence only.
- AUTH-TRUST-1: keep the existing authority boundary. First reproduce the original failure
  with explicit Continue input on v0.24.0 or later and a live provider reviewer. Revisit
  assignment delivery if current evidence calls for it (INV §21).

## Design consistency notes

- Desktop and phone share compact, menu, reading/form, and input text sizes (TS-08.R11).
  Annotation labels and prose use the reading family, source references retain mono, and the
  shared pointer menu styles load in both entries.

- Next presentation review: decide whether crisp asymmetric technical-surface radii are deliberate
  under FS-12.R52.
- When direct-action work resumes, align TS-01.R25/TS-03.R32 with TS-04.R40 and scope FS-17
  section 6's opening sentence to the planned boundary.
- Injected-steer lifetime edge case needs `/investigate-bug` before `/fix`.
- The shared dialog entrance (`ad-dialog-enter`, `components/primitives.css`) ignores
  `prefers-reduced-motion` in every dialog, including Project Files.
- The phone agent header still repeats the model as text under the name now that the model
  switch shows it above the composer.

## Review queue

- Sent annotation source navigation — `2480ca75`, FS-13.R38/A28 and TS-08.R175.

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

### TAB-BLANK — Chrome shows only the theme background on tab return (2026-11-17)

Cause undetermined; native Chrome painting is unverified. Its findings were fixed: desktop browser
failures and a lost shell on foreground return now log `browser diagnostic` records
(TS-03.R92), and settled Markdown is no longer re-parsed on each streamed update (jsdom, 900
messages: ~300 ms to ~8 ms per update). The synthetic Chrome long-task fixture was not rerun;
the event projection and row render still run over the whole history per delta. Next evidence
from the next incident: those `dashboard.log` records for the time, plus a screenshot taken before
clicking. No records with the shell present points to a paint problem; the CSS comparison of
`ui/src/styles/base.css:16–26` is then the next step.

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

### ANNOTATION-QUEUE — delivered annotation remains queued; mid-turn message delivery (investigated 2026-11-24 at `e0e82d1f`) — **Fix model:** trivial/easy — Claude Sonnet or Codex Luna.

**Report (verbatim):** “sending an annotation message while the agent is running put it in queue,
when the turn ended the message was sent (shown in transcript) but the queue display in the dashed
border still showed until the message was sent. Also a regular message I sent mid turn popped up
as a notification and then the message appeared regularly in the transcript rather than in a dashed
border queue appearance, with no withdraw option - is this some new change? Why/when does this happen?”
No affected agent identity, incident timestamp, notification text, clicked action, installed version,
device or incident logs were supplied. Source investigation is on macOS; inspected local dashboard
logs do not identify the reported incident. An optional clarification was requested.

**Expected behavior / diagnosis:** FS-03.R48–R50/R118 and TS-08.R56 distinguish Send (one held,
withdrawable follow-up behind an active turn) from explicit Steer (immediate delivery into the turn,
ordinary user row, no hold to withdraw). A Send received after actual turn completion starts its own
turn even if the browser still looks busy. FS-13.R37 and TS-03.R78 apply these seams to self annotations;
an annotation source card is recorded on acceptance, before its generated prompt is delivered, so
the card's presence alone does not prove delivery. Assignment-owned input has its own labelled tail
and continuation (FS-03.R48), separate from the runtime's withdrawable hold.

**Confirmed history:** distinct Send/Steer dates to `ae3740eb` (2026-09-10); no automatic-Steer
default was found. `e7699576` (2026-11-20) changed desktop delivery feedback to three-second toasts;
`3590f5c5` later that day records Steer before calling the provider and settles its initially
unconfirmed row. Fresh ordinary Send/Enter still selects the prompt path on desktop and phone.
Whether the reported regular message used Steer, arrived after turn completion, or took another
path is **undetermined**. No regular-Send defect or missing diagnostic is established.

- **Must fix** — ANNOTATION-QUEUE-1 (confirmed; code defect; fix trivial/easy):
  `ui/src/store/transcriptStore.ts:55–62,160–168`,
  `ui/src/components/chat/TranscriptView.tsx:89` and
  `ui/src/components/chat/useHeldProjection.ts:19`. An assignment-queued self annotation's
  delivered `user_text` folds into its annotation card, preserving `prompt_seq` and
  `prompt_request_id`. `undeliveredQueued` only searches remaining `user_text` rows, so it
  leaves that exact request in the dashed assignment tail after delivery (FS-03.R118,
  FS-13.R37, INV §1/§2/§19). This is a real desktop schedule: runtime publishes busy before
  its goroutine claims queued input (`internal/runtime/chat.go:1342–1388`), so a busy-triggered
  snapshot can still contain queued input; arrival of the delivered prompt does not refresh
  that snapshot, and the desktop read key stays unchanged during the turn. Phone uses the same
  defective filter (`ui/src/remote/AgentScreen.tsx:742`) but also refetches on transcript updates,
  normally limiting the stale display to the next successful read. Runtime-held annotations
  already settle from paired metadata in `useHeldProjection.ts:36–43`; this reproduction does
  not establish which queue or device the person observed. Extend the existing exact-request
  settlement to paired annotation deliveries, preserving unrelated queued requests and pending
  delivery semantics. The skipped `TranscriptView.test.tsx` case “drops a delivered assignment
  annotation after its prompt folds into the source card” reproduces the visible stale tail;
  unskip for the fix and retain an unrelated-request guard.
- **Worth fixing** — ANNOTATION-QUEUE-2 (confirmed; repeated-fix inventory; fix trivial/easy):
  the same browser queue projection was repaired by `de246a42` (2026-11-15) and `cac7edfb`
  (2026-11-20). Before another patch, refresh the existing INPUT-8 / PROMPT-4 / QUEUED-TAIL-4
  inventories in `docs/ideas.md`: enumerate held/queued snapshot writers, Send/Steer/self-annotation
  admission, raw-to-rendered prompt folding, and each desktop/phone settlement consumer; identify
  where exact-ID acceptance, delivery and removal rules are stated. Keep this bounded to the
  projection failure; the broader lifecycle redesign ideas remain separate. Required by workflow
  §7's second-fix-round rule; extend the existing inventory rather than duplicate it.

**Verification / limits:** the new component reproduction failed on the stale queue assertion
with the exact request already paired into the annotation card, then was marked skipped as §12
requires. Focused TranscriptView/ChatPanel and delegated Composer/phone checks are recorded in
the investigation commit. These are frontend fixtures and source/history evidence, not a live
provider reproduction of the person's incident. No product code or specifications changed.
Next: `/fix` this unit; correlate the original ordinary-message incident if its identity/action
or notification text becomes available.

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

- **Worth fixing** (undetermined) — AUTH-TRUST-3: the preamble that now accompanies the person's
  text in the prompt says it “does not approve a permission request or a pipeline checkpoint”
  (`internal/runtime/message.go:77`). Whether a provider reviewer reads that sentence as negating
  an explicit “push the branch and open the PR” is unknown. Evidence needed: on v0.24.0, with the
  provider's automatic review on, approve a push through Continue and record whether
  `create_task` is allowed. Reword only if that fails. The refusal happens inside the provider
  CLI, so Chuck has no log line for it; the agent's transcript is the only trace. INV §21.


## Blocked on human

None. The file reader's withdrawal test already matches the exact recovery error.
The full UI suite passed all 1,294 tests on 2026-11-19.

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.25.0](../archive/state/RELEASE-v0.25.0-notes.md).
Latest published version: [v0.25.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.25.0).
