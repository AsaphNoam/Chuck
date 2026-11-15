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
  TS-02.R67–R68, TS-08.R168); the 2026-11-15 fix rounds left only the
  EXEC-6c remainder, blocked on a human decision under **Review findings**. Its ready change and plan stay until they close. Both implementer notes were
  confirmed: the room rule stays inside task admission's claiming compare-and-swap, and the
  gate-versus-row span is sound except on a failed end write (EXEC-2). Credentialed provider and
  device gates are not closed by it.

- **Shared desktop and phone input delivery** — reviewed 2026-11-15, `dfd644be^..91828210`
  (TS-08.R164), with reconnect behavior checked at `80414288` (TS-08.R168). INPUT-1–8 are
  under **Review findings** and wait for `/fix`. Desktop transcript refetch fences older reads;
  the phone's held query and sequenced-only transcript are sound projection choices. The desktop
  recovery read still misses reconnect and peer settlement (INPUT-2).

- **One typed turn outcome** — reviewed 2026-11-15, `4f90befa^..ed3fec37` (FS-03.R117/A92,
  TS-01.R42). OUTCOME-1–7 wait for `/fix` under **Review findings**.

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

### One typed turn outcome (`4f90befa^..ed3fec37`) — reviewed 2026-11-15 at `d1b28fc4` — **Fix model:** difficult — Codex Sol.

Second fix round: `bafade3e` and `efc65aaa` repaired Cancel precedence and terminal-success
quota settlement in the last 30 days. The fork test's wait for retained history matches the
journal boundary. Plain Clone keeps its existing provider behavior under TS-04.R110.

- **Must fix** — OUTCOME-3. Replay and the visible transcript still use different meanings
  for the same ending. The decoder collapses the request limit and refusal
  (`internal/runtime/turn_outcome.go:19–26,122–132`); for example, the room then calls a request
  limit an output limit (`internal/server/think_tanks.go:579–586`). Desktop and phone classify
  the raw reason independently (`ui/src/components/chat/turnActivity.ts:68–69,148–163`).
  Their projection rendered quota and process exit as generic failures, and displayed no outcome
  for retained `end_turn` carrying quota evidence. Preserve typed distinctions through the
  shared decode and supply that interpretation on both live and retained transcript paths.
  Exercise both paths while preserving the stored wire labels (FS-03.R74/R117, TS-01.R42,
  INV §2, §11, §20).
- **Must fix** — OUTCOME-4. When an accepted Cancel ends the provider process, the room
  attempt settles as a generic process exit before the cancelled event reaches the room reader.
  `internal/runtime/chat.go:2313–2345` runs the exit hook first, and
  `internal/runtime/registry.go:99–115` passes a fixed `process_exit` cause. The hook fails the
  attempt and publishes that cause (`internal/server/server.go:511–532`,
  `internal/server/think_tanks.go:419–429`), so the later event cannot settle it as cancelled.
  Carry the resolved ending into generation-scoped room cleanup and verify the room hold and
  transcript agree when a speaker ignores cooperative Cancel (FS-03.R9/R117, TS-01.R42,
  TS-14.R6, INV §15, §20, §23).
- **Worth fixing** — OUTCOME-5. Reconciliation joins adjacent replies when a retained
  turn-end payload has the wrong shape. A temporary test with a valid event envelope and
  numeric `stop_reason` reproduced `old turn new turn` in the card preview.
  `internal/server/reconcile.go:183–190` should keep the turn boundary while treating its outcome
  as unknown. Test the last reply after a malformed boundary (INV §2, §7, §24).
- **Worth fixing** — OUTCOME-6. The component tests leave the end-to-end agreement in A92
  unproved. The resolver round trip (`internal/runtime/turn_outcome_test.go:13–47`) starts with
  resolved inputs, and the source guard (`:70–102`) excludes the browser classifier.
  The prompt RPC rejection fixture (`internal/runtime/chat_test.go:1301–1339`) checks logs but
  does not validate the outcome, status, quota settlement or mail. Extend the existing fixtures
  to assert the missing effects from the same emitted event and guard the live and retained
  projections against reintroducing separate classifications (FS-03.A92, TS-01.R42, TS-06.R35,
  INV §17).
- **Worth fixing** — OUTCOME-7. Inventory the terminal decision across ordinary and activation
  prompts, transport delivery and EOF, Cancel and Stop. For each exit, name who settles the
  turn row, publishes the terminal event, and settles quota, mail and the room. Include the legal
  ordering when the response, Cancel and EOF overlap, and use the Stop writer inventory in
  `docs/ideas.md` ("One settlement rule for a stopped or exited process") for the overlapping
  paths (TS-01.R42/R44/R45, INV §5, §15, §18, §20).

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


### Shared desktop and phone input delivery (`dfd644be^..91828210`) — reviewed 2026-11-15 at `80414288` — **Fix model:** difficult — Codex Sol.

Second fix round: exact input identity, successful Steer recovery and withdrawn-input reads
had findings fixed in the last 30 days (`2268e279`, `0e98d3c4`, `8b3ba830`). INPUT-8 asks
for an inventory (§7, §8).

- **Must fix** — INPUT-1: a delayed rejection restores text into the wrong desktop chat.
  `useInputDelivery.ts:121,139` checks the original draft revision, then `Composer.tsx:93`
  writes both that draft and the now-reused textarea. Reproduced with a delayed 409 after
  navigating from chat A to chat B: both stored drafts were correct, but chat B displayed
  chat A's text and could send it to B. Restore the submitted draft through its agent-scoped
  owner and fence the view update. Make the existing switch-chat test wait for the request
  and restoration to complete; cover refusal and uncertainty (FS-03.R36, INV §1, §23).
- **Must fix** — INPUT-2: one recovery offer can submit after another client settles it.
  `RecoveryOffer.tsx:18–41` keeps a browser retry ID until local success. A peer's Dismiss
  changes the receipt without an event (`sessions.go:108–116`), and desktop recovery uses a
  direct read keyed to agent state (`ChatPanel.tsx:141–146`), outside reconnect invalidation.
  A stale card can send with a new ID and no source receipt (`api/client.ts:218–223`);
  `attachments.go:480–547,629–635` admits it before checking for a recovery and infers the
  source from the payload. Even after an authoritative refresh, the old browser key remains.
  Bind recovery Send to the exact source, arbitrate settlement durably before delivery or
  dismissal, preserve same-ID replay, refresh desktop held/recovery on reconnect, and bound
  retry-key lifetime using authoritative settlement. Test two clients, concurrent actions,
  a lost response, and a newer offer masking an older one (FS-03.R100, TS-03.R62/R74,
  TS-08.R168, INV §1, §15, §16, §19, §23).
- **Must fix** — INPUT-3: an attachment retry keeps blocking the composer after recovery
  settles. `useInputDelivery.ts:80–88,114–119` persists the original receipt as non-retryable;
  `RecoveryOffer.tsx:33–41` clears only the recovered Send's retry key and projection.
  After the person sends or dismisses the recovered offer, the matching original draft can
  remain disabled and still point to that offer, including after reload. Reconcile the exact
  pending receipt with confirmed settlement on either client, preserving any newer unrelated
  draft. Test uncertainty, recovery Send and Dismiss, and reload (FS-03.R84/R100, INV §8, §19).
- **Must fix** — INPUT-4: a released text receipt renders as sent and clears the draft.
  `attachments.go:726–738` returns `202` with `delivery: released` for a same-ID replay after
  release. `Composer.tsx:75–80` treats it as sent when the draft has no files, and
  `useInputDelivery.ts:79–94` clears the draft. Reproduced against that response shape. The
  phone also clears the draft, while `RecoveryOffer.tsx:33–35` treats any 2xx as settlement.
  Handle release by receipt status for both text and files, including recovered Sends; assert
  no echo, no draft loss and an actionable recovery on a replay after Stop (TS-03.R62/R74,
  FS-03.R100, INV §8, §11, §19).
- **Must fix** — INPUT-5: a stale Send now can steer a different held message.
  The phone supplies the displayed request ID (`AgentScreen.tsx:802`) and desktop supplies
  no expected hold ID (`Composer.tsx:197`). The server replaces the request identity with
  the hold it sees on arrival (`attachments.go:458–470`), so a second client replacing the
  hold before the click arrives can cause the replacement to be sent. Carry and verify the
  displayed hold identity before consuming it; reconcile only that hold on completion.
  Test two clients replacing the held message before promotion (FS-03.R50/R100,
  TS-03.R74, INV §19, §23).
- **Worth fixing** — INPUT-6: the Steer completion can misreport which turn got the message.
  `useInputDelivery.ts:101–105` reports every persisted desktop Steer as delivered into
  the running turn, even when the server returns `new_turn`. Phone gives neither successful
  Steer outcome. Use the authoritative outcome on both surfaces, and check the attachment
  and phone turn-end race with `new_turn` from the server (FS-03.R50, INV §8, §11).
- **Worth fixing** — INPUT-7: desktop Steer can do nothing while Send is pending.
  `Composer.tsx:286` disables Steer only while steering, but `useInputDelivery.ts:44`
  silently refuses every delivery while either request is pending. Send from a busy chat,
  type new text before the reply, then click the enabled Steer. Keep the control aligned
  with admission and give feedback if it must wait; verify with a deferred Send
  (TS-08.R164, INV §8).
- **Worth fixing** — INPUT-8: inventory (§7). List every writer and reader of browser draft
  and retry state, held and recovery projections, and durable receipts. Identify the entry
  points for Send, Steer, withdraw, recover and dismiss, and which evidence permits each
  owner to release a specific request across navigation, reconnect and peer settlement.
  Start from the echo and held-tail inventory in `docs/ideas.md` ("One owner for a sent
  prompt's echo and the held tail").

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
