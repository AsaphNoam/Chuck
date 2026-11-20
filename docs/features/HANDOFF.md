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
  [receipt](../archive/reviews/architecture/v0.23.0-next.json); it did not gate v0.24.0. Its fixes
  are recorded there. The operator resolved EXEC-6c by retaining the existing shrink-only
  test-fixture exceptions in TS-01.R50. The next release needs its own review and
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

- **Project Files:** Confirm an all-folders default with a folder selector, a sandboxed HTML preview
  in a new browser tab, and desktop-only shared-folder access without symbolic-link traversal.
  [FS-11.R11–R16/A6–A10](../specs/features/FS-11-project-resources.md#project-file-browser) and
  FS-03.R119–R120/A94–A95 cover the file browser, Finder and CSV. Technical design follows confirmation.

## Resolved decisions

- Message action design is ready in
  [message-hover-actions-and-current-conversation-edit.md](../ready-changes/message-hover-actions-and-current-conversation-edit.md)
  (FS-01.R60–R62/A42–A44). The last sent message has Copy/Edit/Fork; other messages have
  Copy/Fork as top-right hover icons. Fresh-context design review found no material findings
  on 2026-11-20; the change is waiting to start.
- Keep the current HTTP error envelopes and provider model-ID ownership. Revisit each
  compatibility contract when there is a concrete caller need and a migration plan.
- Keep **Open agent** hidden after a failed launch or resume. Explain the pause and offer
  **Retry stage**, consistent with restart recovery.
- Treat an unechoed model or effort request as **Unconfirmed** where confirmation evidence is
  available. The current launch path discards that distinction before the UI sees it, so the
  display change is deferred. The operator approved using existing evidence only.
- WITHDRAW-SEND-3 and STEER-HOLD-4: show assignment-queued input once, labelled as queued for
  the assignment's next turn. Implemented under FS-03.R48 (2026-11-20): it sits at the
  transcript tail, the empty composer hides Steer when only assignment-queued input is pending,
  and runtime-held input keeps its Withdraw action.
- EXEC-6c: TS-01.R50 permits the 11 existing test-only store writers under the shrink-only
  `fixtureOnlyWriters` list. The archived architecture receipt records closure at `596821bf`.
- AUTH-TRUST-1: keep the existing authority boundary. First reproduce the original failure
  with explicit Continue input on v0.24.0 or later and a live provider reviewer. Revisit
  assignment delivery if current evidence calls for it (INV §21).
- AUTH-TRUST-2: the assignment heading now says “Why this stage visit started.”

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

- **Message hover actions and current-conversation Edit** (FS-01.R60–R62, FS-22.R13,
  TS-01.R51, TS-02.R69–R70, TS-03.R90–R91, TS-04.R113, TS-08.R171): reviewed at
  `b2986ebf` for `99e4672b`, `1bfa4d28`, `b16c722f` and `b2986ebf`; open
  EDIT-REVIEW findings below. EDIT-EDGE-1–3 are closed by the commit after `58708881`: a
  post-decision failure now stops the unpublished candidate and runs restart recovery in
  process, which bears on EDIT-REVIEW-4's reproduction; recheck it there.
  Packaged live provider receipts remain owed in ACCEPTANCE-GATES.
- **Assignment-queued input at the transcript tail** (WITHDRAW-SEND-3/STEER-HOLD-4; FS-03.R48):
  reviewed at `b16c722f` for implementation `c02083df`; open findings QUEUED-TAIL-1–4 below.

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

### QUEUED-TAIL — assignment-queued input at the transcript tail (2026-11-20) — **Fix model:** difficult — Codex Sol.

- **Must fix** — QUEUED-TAIL-1 (medium; FS-03.R48/R118, TS-09.R69/R71, INV §1/§19). The card
  follows the latest prompt snapshot independently of the matching transcript event. The runtime
  publishes busy before claiming the receipt (`internal/runtime/chat.go:1250–1271`). A read after
  claim returns `queued:[]` while the receipt is still `prepared`, before content preparation or
  any `user_text` (`internal/state/attachment_receipts.go:247,348–351`). A read before claim can
  instead leave a stale queued card after that same request enters history; desktop reconciles
  runtime holds only (`ui/src/components/chat/useHeldProjection.ts:28–42`,
  `TranscriptView.tsx:88,275`). A server probe paused at the real claim and confirmed the message
  was absent from the prompt snapshot, recovery and transcript. Chromium reproduced disappearance
  on desktop and phone and duplicate pending/sent copies on the agent screen and expanded dashboard.
  Exercise both snapshot/event orders with exact request identity and a reload during claim.
- **Must fix** — QUEUED-TAIL-2 (difficult; FS-03.R48, INV §15/§19). The continuation records a
  durable `user_text` before the provider write (`internal/runtime/message.go:79–90`,
  `internal/runtime/chat.go:1270–1294`). A controlled fake-provider run held the send at the
  existing Cancel lock, stopped the agent, and verified that no provider prompt was written. The
  receipt was released for recovery, but the journal still contained the ordinary user row for
  that exact request. The history therefore says the message was sent while the receipt offers it
  back as unsent. Keep the durable acceptance and recovery guarantees while making the transcript
  and search reflect the actual handoff. Use the controlled Stop schedule as a regression.
- **Worth fixing** — QUEUED-TAIL-4 (medium; workflow §7, INV §19). The same subsystem received
  `119880aa` (moved assignment receipt and attachments) and `de246a42` (prompt identity and
  held-tail reconciliation) within 30 days. Before patching, inventory the writers and entry
  points from assignment acceptance, claim and provider submission through Stop, refusal,
  replacement, restart and recovery. Include the prompt snapshot, transcript, search and each
  browser projection, and identify which existing rule governs each handoff.

### EDIT-REVIEW — Message hover actions and current-conversation Edit (2026-11-20) — **Fix model:** difficult — Codex Sol.

- **Must fix** — EDIT-REVIEW-1 (difficult, FS-01.A43, TS-04.R113, INV §1/§20). A successful
  later-message Edit changes the native session but keeps the previous turn's fork point bound
  to the original session. The next Edit of the replacement consults that point and refuses it
  as belonging to an earlier runtime (`internal/server/edit.go:307–315`,
  `internal/server/fork_point.go:26–35`). A three-turn fake-provider run reproduced the
  disabled Edit in Chromium after reload and the matching HTTP 422 through the built CLI.
  Preserve immutable provenance and prove that the retained boundary is usable for repeated
  same-conversation edits. Verify two successive edits of the last message with both adapters.
- **Must fix** — EDIT-REVIEW-2 (medium, TS-02.R69, INV §15/§19). Dispatch and restart/Stop
  release mark the edit dispatched before creating the ordinary receipt
  (`internal/server/edit.go:537–546,632–649`). If the process exits or the insert fails between
  those steps, the revised conversation has no replacement prompt and neither recovery path
  owns it. A controlled store test confirmed it disappeared from both open edits and
  receipt recovery. Transfer ownership and record the recovery disposition in one transaction
  and test failures on both sides of that transaction.
- **Must fix** — EDIT-REVIEW-3 (difficult, FS-01.R61, TS-01.R51, INV §5/§19). The shared
  Edit gate accepts an idle chat with `BackgroundRunning=1` in a store-backed probe, and
  preparation stops the runtime that owns that work (`internal/state/edits.go:73–90`,
  `internal/server/edit.go:239,412`). Separately, the reservation uses a map and mutex outside
  the actual turn claim (`internal/runtime/edit_reservation.go`, `chat.go:672–683,2032–2082`).
  An activation can pass the map check, pause, then claim a turn after Edit has validated
  idle and accepted. Use the runtime's existing turn admission owner for exclusion and include
  background work, pending input, and task and room ownership in the gate inventory.
- **Must fix** — EDIT-REVIEW-4 (difficult, TS-01.R51, TS-02.R69–R70, TS-03.R91, INV §1/§15/§18).
  The journal appends and publishes the revision before candidate selection commits, and the
  transcript handler samples native session, revision, and history separately
  (`internal/server/edit.go:477–508`, `internal/server/sessions.go:743–782`). A SQLite
  fault at `FinishEdit` made the ordinary API return revision 1 with the old turn hidden
  while the durable native selection still named the original. Status asked the operator to
  Resume, which returned “agent is already running.” Exercise read, stream and resume on both
  sides of the decision, including after repair. The change must keep context, history and
  search consistent and admit no ordinary work while they disagree.
- **Must fix** — EDIT-REVIEW-5 (trivial/easy, FS-01.R60, TS-02.R70, INV §1/§11).
  Clone copies source revision metadata into its already folded prefix
  (`internal/transcript/clone.go:34–39`, `internal/runtime/fork.go:207–219`). The clone gets
  new sequence IDs, so the source's superseded range can hide the clone's active messages.
  An isolated test lost the entire edited third turn after copying it. Exclude source revision
  records when writing the folded prefix and verify the next message in the new agent stays visible.
- **Must fix** — EDIT-REVIEW-6 (trivial/easy, FS-15.R2, TS-02.R70, INV §1/§11). An existing
  immutable transcript link resolves through the active fold (`internal/contextref/service.go:468–469`).
  In a real store/journal test, sharing a turn and then editing it made the saved link return
  `context_source_unavailable` although its bytes remained retained. Read retained bytes for
  the exact already-authorized span and keep new selection and unpinned reads on active history.
- **Must fix** — EDIT-REVIEW-7 (medium, FS-22.R13, TS-03.R91, INV §1/§11). Follow checks
  step-by-step continuity before looking for a revision later in the same folded response
  (`internal/cli/operator_follow.go:446–470,508–520`). A retained annotation in the superseded
  range stopped catch-up before the revision. After a live Edit, the built CLI also emitted
  earlier sequence IDs twice and delivered the revision record only after that replay
  (`internal/cli/operator_edit.go:100–127`). Give the consumer a reset boundary before the
  refreshed view and verify live, reconnect and initial reads with a retained control record.
- **Must fix** — EDIT-REVIEW-8 (medium, FS-01.R61, TS-02.R69, TS-03.R90, INV §10/§18/§19).
  The only UI handle for a failed precommit Edit is a component-local request ID
  (`ui/src/components/chat/MessageEditor.tsx:41–45`). On reload the snapshot lists only open
  edits and the prompt recovery endpoint reads only ordinary receipts
  (`internal/state/manager.go`, `internal/server/sessions.go:93–107`). An induced precommit
  failure left the replacement retrievable by its known ID but undiscoverable in either
  endpoint. Expose and acknowledge an Edit recovery successor that revalidates the intended
  message, with a reload test that starts without the ID in memory.
- **Worth fixing** — EDIT-REVIEW-9 (trivial/easy, FS-12.R24, TS-08.R171, INV §8/§13).
  Opening Edit on a long message scrolls the pane off the tail and leaves Jump to latest
  over the editor or explanation (`ui/src/components/chat/TranscriptView.tsx:285–288`,
  `ui/src/styles/features/agent.css:1692,2697`). Chromium showed the overlap at 1024px on
  the agent page and at 1440px in a three-column dashboard pane in Boring Night. Keep the
  editor and its controls clear while it has focus, including during scroll and focus return.
- **Worth fixing** — EDIT-REVIEW-10 (medium, workflow §7, INV §1/§5/§15/§19). Commit
  `b2986ebf` already repaired Edit interruption and staged lifecycle findings, and the
  underlying turn owner had repairs in `bb903498` and `4c896077`. Inventory the start, stop,
  and turn-admission entry points, the writers of candidate selection, revision visibility,
  and replacement receipt, and each legal transition. Include restart, failed publication,
  application-initiated turns, and readers that need active versus retained history.
- **Must fix** — EDIT-REVIEW-11 (medium, FS-01.R61, TS-01.R51, INV §8/§19). The route
  accepts the last person message whenever any turn end follows it, then supersedes from
  that message through the current tail (`internal/server/edit.go:250–306`).
  In a server test, an autonomous waking-mail turn completed after the last person turn.
  Edit accepted, committed and hid the later turn's end. It must refuse because the selected
  message did not start the latest turn, and the desktop gate must explain the same restriction.
  Assert the later reply and its turn remain visible after refusal.
- **Worth fixing** — EDIT-REVIEW-12 (trivial/easy, FS-01.R58/R60–R62, INV §10). The
  feature spec's deviations section still says clone-based Edit is shipped
  (`docs/specs/features/FS-01-agent-lifecycle.md:907–911`). Reconcile that historical
  behavior with the implemented current-conversation editor and retained legacy API.
- **Worth fixing** — EDIT-EDGE-4 (easy, TS-01.R51, INV §5). Group release stops an editing
  member's Edit only after claiming every other member, but the interrupt cannot be undone. A
  later conflict (the member's claim retaken after the interrupt, a start lease, the membership
  recheck, or a second editing member) still returns 409 after that Edit was aborted
  (`internal/server/groups.go` `groupAction`). Found by the EDIT-EDGE independent check.

### STEER-HOLD — refused Steer of a held message (fixed 2026-11-20; remaining work) — **Fix model:** difficult — Codex Sol.

STEER-HOLD-1/2/3/5 are fixed and the inventory (STEER-HOLD-6) is in `docs/ideas.md` as “One
transition table for a held message and a Steer”. One gap from the fix's independent check remains.

- **Must fix** — STEER-HOLD-7 (medium; FS-03.R13/R50/R100, INV §19). To keep a refused Steer out
  of history, the Steer `user_text` is now written after the native `_session/steering` answer,
  not before it (`internal/runtime/chat.go` `SteerMessage`). The receipt still records `unknown`
  before the call, so the input stays recoverable. A crash, Stop or transport close while that
  call is pending can now leave a delivered or possibly delivered Steer out of the transcript; a
  late append after Stop fails and only logs. An injected Steer's first reply chunks can also be
  written before its row (suspected, not reproduced). R13 says every accepted prompt is appended
  before publication. Choose with the transition-table idea: write ahead with a retraction record
  on refusal, or narrow R13 for Steer to “recoverable through its receipt”.

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

Prepared notes: [v0.24.0](../archive/state/RELEASE-v0.24.0-notes.md).
Latest published version: [v0.24.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.24.0).
