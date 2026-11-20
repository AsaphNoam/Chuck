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

None.

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
  the assignment's next turn. The in-progress FS-03.R48 change puts it at the transcript tail.
  The empty composer hides Steer when only assignment-queued input is pending. Runtime-held
  input keeps its Withdraw action.
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
