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
  are recorded there; EXEC-6c is its only open Must-fix item. The next release needs its own review and
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

- Message actions use light, borderless icons at the top right on hover, with Copy on every
  message, Edit only on the last message, and Fork on every message. Confirm the draft's
  interpretation of "last message" as the latest sent message, then finish Edit's reply,
  retention/recovery and technical design. Draft FS-01.R60–R62/A42–A44; see Ideas being defined.
- Whether to standardize TS-03.R3–R4 mixed legacy error envelopes or TS-04.R3 provider model-ID
  ownership (compatibility change).
- Whether a pipeline stage paused after failed launch/resume continues withholding **Open agent**,
  matching restart recovery, or permits chat under a wider continuation contract.
- Whether a model or effort the adapter accepted without echoing it back should show as
  unconfirmed (CR-A10 lead 13; INV §20 reads missing evidence as unknown).
- WITHDRAW-SEND-3 (FS-03.R48, TS-09.R71): pending chat input renders in two places — a runtime
  hold at the transcript's end (`TranscriptView.tsx`) and input queued for the agent's assignment
  under the composer (`Composer.tsx`, `QueuedInput.tsx`); a person reported the second as a re-sent
  message “at the bottom of the text box” after Withdraw. Decide whether assignment-queued input
  renders as the same pending tail at the transcript's end, labelled as delivered with the
  assignment's next turn; then specify and test it. Field evidence still wanted: the agent and a
  screenshot from the next occurrence; the server now logs `conversation input queued for
  assignment` with the task and its state.

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

### TASK-STOP — Stop in the running-commands popover answers “Could not stop this task” (investigated 2026-11-18) — **Fix model:** difficult — Codex Sol.

**Report (verbatim):** “I tried stopping a running process via the running popup in the agent chat
menu and received - could not stop this task - try again”. No agent, build or log was supplied.
The local dashboard (`bin/chuck` running since 2026-11-06, logging every request) has no
`background-task-stop` request in its log and no transcript written today, so it is not the source.
Source inspected at `2d6c7af9` against the pinned `claude-agent-acp` 0.85.1 package. The UI shows
that one message for every failure (404, 409, 422, network), so the server's answer is unknown.

**Mechanism, reproduced for subagent tasks.** The adapter keeps one async-task registry per root
session and its `stopAsyncTask` resolves `this.sessions[params.sessionId]`, where only root sessions
are registered (`dist/acp-agent.js:2215–2229`). A task started by a native subagent is announced
routed to the child session id (`native-subagents.js:233–256`). Chuck stores that routed id and sends
it back on Stop, so the adapter answers `{stopped:false}` → 409 → the reported message. Whether the
person's task belonged to a subagent is **undetermined**; a root task fails only on an adapter
refusal or a task that already ended. Reproduction: `TestStopRunningChildBackgroundTaskAddressesTheRootSession`
(`internal/runtime/tasks_child_stop_repro_test.go`, skipped).

- **Must fix** — TASK-STOP-1 (confirmed; FS-03.R59/R107, TS-04.R64, INV §11). `onAsyncTaskUpdate`
  records the notification's `sessionId` as the task owner (`internal/runtime/tasks.go:97`) and
  `StopBackgroundTask` sends it (`tasks.go:151`). Stopping any running subagent task therefore
  always fails. Send the root session id (`as.sessionID`) with the raw task id. Make fakeacp's
  `_session/async_task/stop` refuse a non-root `sessionId`, as the real adapter does; it currently
  accepts any id (`internal/runtime/testdata/fakeacp/main.go:460–475`). Then un-skip the
  reproduction and add a running-child stop to the fake-ACP tests.
- **Must fix** — TASK-STOP-2 (probable; spec gap; FS-03.R59/R107, TS-04.R64, INV §1/§18). Once a
  subagent is terminal, the adapter drops every update routed to it, including later
  `async_task_state_update`s for tasks it started (`native-subagents.js:233–237, 242–256`). A
  background command started by a subagent outlives it, so Chuck's row and running count stay
  `running` until a resume fence. The Stop on that row fails too after the command ends on its own.
  R59 says the state is “exactly as reported by the runtime”, but this runtime goes silent.
  Specify what Chuck shows for a task whose child ended (for example, keep it stoppable via the
  root and say its final state is unknown). Implement and test it with the fake reproducing the
  dropped updates. This is external-protocol behaviour and sets the unit's fix model.
- **Worth fixing** — TASK-STOP-3 (confirmed; FS-03.R59, TS-03.R44, INV §8).
  `RunningTasks.tsx:22` maps every rejection to “Could not stop this task. Try again.” A 404
  (task no longer running) and a 422 (no stop control) invite retries that cannot succeed. Show
  the stale or unavailable outcome for those, and keep the retry text for 409 and network failures.
  Add component tests for each status.
- **Worth fixing** — TASK-STOP-4 (observability; INV §22). `StopBackgroundTask` returns the RPC
  error or `stopped:false` without logging it (`tasks.go:151–160`), and the route logs only its
  status. Log agent, task id, whether the target was root or child, and the bounded RPC
  code or `stopped:false` at Warn. A report like this one could then separate an id mismatch,
  an adapter refusal and a stale row.
- **Worth fixing** — TASK-STOP-5 (inventory; workflow §7 second fix round). The background-task Stop
  flow was fixed on 2026-10-31 (`0277c5da`, BGRUN-UI-1/2). Before patching, list every writer of a
  task's running state (adapter routing, `as.tasks`, generation teardown, `collectTasks` fences),
  every Stop entry point (desktop popover, phone, API), and where their legal moves are specified.

### STEER-HOLD — a refused Steer of a queued message cycles into “held message changed” (investigated 2026-11-17) — **Fix model:** difficult — Codex Sol.

**Report (verbatim):** “trying to steer doesn't do anything (message stays queued) and I get "Could
not steer — held message changed; reload before steering." Reloading the page didn't change
anything. When the agent finished its turn the message was sent as usual queued message behavior.
Also the 'Queued — it will be delivered when the agent's assignment continues.' appears a lot of
times even if there is no queued message, I think it just doesn't leave, when steer does work the
message 'Delivered into the running turn.' just stays”. No field build, agent, adapter or log was
supplied. The local dashboard (a v0.22.0-era build running since 2026-11-06) recorded no Steer
request, so it is not the source; source inspected at `81e67283`.

**Mechanism, reproduced** with the fake adapter set to refuse: the first empty-composer Steer
returns “The agent refused this steered message”, the server re-holds the message under a new
`held_id`, and the browser keeps the old one, so the next Steer is `held_changed`; that refreshes the
hold, and the next Steer is refused again. Turn end then sends the restored hold as an ordinary
queued message. **Why the person's adapter refused is undetermined.** Neither the refusal reason nor
desktop requests reach the log. Reproduction: `TestRefusedPromotedSteerKeepsTheHoldUnchanged`
(`internal/server/queue_steer_test.go`, skipped).

- **Must fix** — STEER-HOLD-1 (confirmed; FS-03.R50, TS-03.R39, INV §5/§19). A refused promoted
  Steer restores the hold through `hold()`, which mints a new `held_id`
  (`internal/runtime/chat.go:1032–1035`). The desktop composer refreshes its hold only on
  `held_changed`, not on a refusal (`ui/src/components/chat/useInputDelivery.ts:171–178`). Every
  retry therefore fails. Restore the original hold identity (or refresh it on any promoted-Steer
  refusal), and un-skip the reproduction.
- **Must fix** — STEER-HOLD-2 (confirmed; FS-03.R48/R50, INV §15/§19). A request-bearing Steer emits
  its durable `user_text` before the native call (`internal/runtime/chat.go:839–848`). A refusal
  leaves that entry behind, even though the response says nothing was delivered and the code
  comment says a refusal writes nothing. Each refused attempt adds another copy, and the restored
  hold is later sent again. The desktop held projection can match that `request_id` and clear the
  pending tail early (`ui/src/components/chat/useHeldProjection.ts:35–45`). The skipped test's
  transcript assertion covers it. Typed Steer has the same ordering.
- **Worth fixing** — STEER-HOLD-3 (confirmed code path; spec gap; FS-03.R48/R50, INV §1). Composer notices
  are cleared only by the next Send or Steer, Withdraw, Cancel or an agent switch
  (`ui/src/components/chat/Composer.tsx:44,110,188,285,298`;
  `useInputDelivery.ts:141–147`). “Delivered into the running turn.”, “That turn had already
  ended…” and “Queued — …assignment continues.” therefore stay after the turn or queued input that
  they describe has settled. FS-03 names the text but not the lifetime. **Operator decision
  (2026-11-17):** each of these Send/Steer outcome notices becomes a toast that lasts 3 seconds and
  then fades out, rather than a persistent composer line. Specify that in FS-03, and reuse the
  existing toast surface (`ui/src/components/shell/NotificationCenter.tsx`, which currently
  auto-dismisses after 6s) with a per-toast 3s duration and a fade. Add a Composer test for the
  3s dismissal. Errors stay where they are.
- **Worth fixing** — STEER-HOLD-4 (probable contributor; spec gap; FS-03.R48/R50, FS-14.R100, INV §8). When
  the visible pending message is assignment-queued input rather than a runtime hold, Steer stays
  shown but returns silently with an empty composer (`Composer.tsx:178–183`). This matches
  “steer doesn't do anything”. FS-03.R50 does not say whether assignment-queued input can be steered.
  Specify the decision (promote it, or withhold Steer and explain it), then test it.
- **Worth fixing** — STEER-HOLD-5 (observability; INV §22). The refusal transition logs only
  `provider_refused` (`internal/runtime/chat.go:891–909`). The adapter's RPC code and method are not
  logged, and desktop route outcomes are not logged either. Log the bounded RPC code with the
  `request_id`/`held_id` for a refused Steer and `held_changed`. A report like this one could then
  tell an unsupported adapter, invalid parameters and a stale client apart.
- **Worth fixing** — STEER-HOLD-6 (inventory; workflow §7 second fix round). Chat input
  holding and Steer received fixes within 30 days (`7c407643`, `1bf0ffb9`, `de246a42`,
  `0e98d3c4`). Before patching, list every writer of `turn.held` and `turn.steerFallback`, every
  Send/Steer/Withdraw/turn-end/stop entry point, the transcript and receipt writes each makes,
  and where the legal moves are specified.

### CODEX-FAST — a slow turn with Fast enabled has no tier receipt (investigated 2026-11-17) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “codex seems to be running really slow even in fast mode, any chance that
chuck isn't really using fast mode even if it's toggled on? Is there a way for me to verify?”
No affected chat, turn time, field build, or log was supplied. Source inspected at `289bda92`.

**Cause of the reported slowness: undetermined.** FS-03.R45–R46 and TS-04.R45–R47 distinguish the
setting applied for the next turn from provider execution. Chuck sends `fast-mode=on` only when the
live adapter advertises it and checks a contradictory `currentValue`. The installed adapter
(codex-acp 1.1.2) passes Fast to the app-server at turn start. In an isolated GPT-6.1 Sol test with
that adapter, Codex 0.161.0, and a dummy API key, the Fast-on turn emitted
`service_tier: "priority"` to a synthetic loopback Responses endpoint; the Fast-off turn omitted
the field, with the same model and effort.
Changing effort after enabling Fast preserved the adapter's Fast readback. No live provider was
contacted. The focused runtime and HTTP configuration tests passed. There is no failing
reproduction to retain.

**Separate local mismatch, confirmed:** the only Codex chat listed as running on the inspected
dashboard had Fast off. The saved GPT-6.1 Sol model declares `fast: false` while the current Codex
catalog advertises `fast`. Autosync intentionally leaves saved model entries alone (FS-09.R28/R53);
Settings → Backends → Codex → GPT-6.1 Sol → Fast mode capability restores the option, and the chat's
own toggle still needs enabling. This explains a missing switch, not a slow turn with it on.

- **Worth fixing** — CODEX-FAST-1 (confirmed, observability; INV §20, §22; TS-04.R54).
  A live toggle returns Chuck's recorded setting
  (`internal/server/session_config.go:88–104`), and
  `internal/runtime/chat.go:3796–3823` accepts an unreported setting as well as an explicit match.
  The sampled Codex logs and turn metadata provided no tier receipt, and the reported turn was
  not identified. Make the attempted value and adapter readback distinguishable on the live path
  and launch/resume path, including an explicit unknown when there is no readback. Correlate
  those facts with a turn and expose the tier sent or served only when supported by adapter or
  provider evidence. Keep diagnostics to identifiers and validated tier values. Verify missing
  and contradictory readback separately; verify request delivery against an independent capture
  such as the loopback on/off comparison. Treat provider execution as unknown without a receipt.

### TAB-BLANK — Chrome displays only the theme background on tab return (investigated 2026-11-17) — **Fix model:** medium — Codex Terra or Claude Opus.

**Report (verbatim):** “sometimes the page just goes blank when I switch between tabs and go back,
click anywhere on the page makes the view render”. The operator confirmed Chrome browser tabs and
the newest Chuck build: “the Chuck theme color is still active but other than that the screen goes
completely blank”. No incident time, exact field build or browser error was supplied.

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

**Multi-tab follow-up:** Headless Chrome opened six same-origin pages (Dashboard, Tasks, Pipelines,
Archive, Settings and a project) with one shared stream worker. All retained the same mounted shell,
visible header and route text through a server stop/restart and reconnection. A simulated
hidden-to-visible transition on Dashboard triggered two deliberately failed config requests; the
skin reverted to Core as specified (TS-08.R30), and the shell stayed visible. No page errors. This
exercises data refresh and reconnect, but does not verify native Chrome painting. The outside-click
handlers dismiss menus or dialogs; they do not mount or reveal the shell. The fixed, full-viewport
theme decoration in `ui/src/styles/base.css:16–26` is painted behind the shell and ignores pointer
input. It is a candidate for a temporary CSS comparison if the failure reproduces with the shell
still present and no obscuring overlay. Chrome repaint remains a hypothesis.

**Loaded-chat follow-up:** The operator reported an individual agent chat as the affected view.
Their projects include 120 chats, 104 all stopped, and 70, with 20 running and over 300 stopped
overall. An isolated headless Chrome fixture matched that scale with 350 synthetic identities
(20 running, 330 stopped) across four projects. Six tabs hosted three project grids and three
individual chats (900-turn running, 200-turn running, 200-turn stopped). Over a simulated tab
return, live update burst and forced stream reconnect, every sampled header and chat remained
present, with no shell detachments or page errors. Reconnect refetched open chat histories and
made zero transcript requests from the grid tabs. Native screen painting remains unverified.

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
- **Worth fixing** (confirmed in synthetic Chrome workload) — TAB-BLANK-2, long desktop histories
  stall streaming and reconnect. A burst of 48 live assistant deltas per synthetic running chat
  produced main-thread tasks up to 669 ms in the 900-turn tab, which rendered about 54,000 transcript
  elements. The 200-turn running tab logged tasks up to 319 ms in the same burst. No project grid
  logged a task of 50 ms or longer. These are local stress-run measurements with synthetic Markdown,
  not a field profile or proof of the blank screen. Desktop reconnect fetches complete history
  (`ui/src/api/sse.ts:80–82,371–383`; `ui/src/api/client.ts:71–75`), and live deltas reproject and
  render completed turns along with the changing tail (`ui/src/components/chat/TranscriptView.tsx:148–155`,
  `ui/src/components/chat/TurnList.tsx:56–86`; FS-03.R12, INV §16). Reuse unchanged turn work on
  streaming updates, then compare the long-task trace with the same fixture and verify scroll,
  selection and transcript correctness across reconnect. This is a bounded render-path fix.

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
  one's is not. INV §21.
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
  CLI, so Chuck has no log line for it; the agent's transcript is the only trace. INV §21.


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
