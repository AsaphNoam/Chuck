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

- Inline task delivery and guidance cleanup (FS-16.R60–R61, FS-18.R22–R23) keeps native-command
  handling on fresh launches and attributed content on continuations. Implemented 2026-11-21;
  review repairs finished at `10d01e1b`.
- Project Files starts with all shared folders and offers a folder selector. HTML opens in a
  sandboxed viewer in a new browser tab. The shared folder is desktop-only and symbolic links
  cannot be followed. Shipped 2026-11-20 (FS-11.R11–R16).
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
- Injected-steer lifetime edge case needs `/investigate-bug` before `/fix`.
- The shared dialog entrance (`ad-dialog-enter`, `components/primitives.css`) ignores
  `prefers-reduced-motion` in every dialog, including Project Files.
- The phone agent header still repeats the model as text under the name now that the model
  switch shows it above the composer.

## Review queue

- **Dismiss quota banner when automatic continuation is off** (FS-01.R40): available for review.
  See the `Add Dismiss to disabled quota banners` commit. Dismissal is local to the mounted
  view and scoped to one interruption.

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

- **Browse project files** (FS-11.R11–R16, FS-03.R119–R120, TS-03.R93–R94, TS-05.R30–R31,
  TS-08.R172–R174): reviewed `8cba163c^..f2239ad0` at `f2239ad0`. **Finished** on 2026-11-20
  by the PROJECT-FILES fix commit. PROJECT-FILES-7's inventory and remaining owner, focus and
  menu-claim gaps are in `docs/ideas.md` as “One owner rule for the file reader, its menus and
  focus”. The 1–4 repairs are proven in jsdom and API tests, not re-run in real Chromium. Native
  Finder selection remains owed in ACCEPTANCE-GATES.

- **Message hover actions and current-conversation Edit** (FS-01.R60–R62, FS-22.R13,
  TS-01.R51, TS-02.R69–R70, TS-03.R90–R91, TS-04.R113, TS-08.R171): reviewed at
  `b2986ebf` for `99e4672b`, `1bfa4d28`, `b16c722f` and `b2986ebf`. **Finished** on
  2026-11-20 by the EDIT-REVIEW fix commits (`e31222f9`..closure). EDIT-REVIEW-4, EDIT-EDGE-4
  and the inventory's remaining gaps (room ownership, dispatch failure, Stop before dispatch)
  are in `docs/ideas.md` as “One commit point for an Edit's history, session and replacement”.
  Packaged live provider receipts, including two successive edits per adapter, remain owed in
  ACCEPTANCE-GATES.

- **Inline task assignments and concise guidance** (FS-16.R60–R61, FS-18.R22–R23, TS-01.R52,
  TS-10.R63–R65, TS-11.R25–R26): reviewed `c872c240` and `7965a17c` at `7965a17c`.
  **Finished** on 2026-11-21 by `10d01e1b` and closure. INLINE-TASK-5–7's
  inventory and remaining guidance/payload-bound gaps are in `docs/ideas.md` as
  “One rule for task execution across continuations”. The packaged Claude/Codex gate remains owed.

## Investigations awaiting evidence

### TASK-PROMPT — reporting paragraph after a skill invocation (investigated 2026-11-21)

**Report (verbatim):** “I specifically told you to only trigger the instruction skill and yet
you still wrote an entire wall of text”. The annotation quotes transcript event 1 of
`a_06bd8f`: `/fix` followed by “This is your Chuck task” and the reporting/coordination paragraph.
The request was to launch one Claude Opus Medium agent at a time with only the fix skill.
No separate diagnostic log or exact installed version was supplied.

**Confirmed — works as specified, not a code defect.** The durable task
`tk_e67d15ecd4cbe7df` stores `instruction: "/fix"`, `backend: "claude"`, `model: "opus"`,
`effort: "medium"`. The orchestrator supplied no extra assignment prose. Chuck appends it:
`Registry.StartLaunchedTask` (`internal/runtime/registry_message.go:22`) calls
`LaunchedTaskPrompt` (`internal/runtime/activation_kinds.go:32`), which preserves the instruction
and adds the shared `taskReportContract`. FS-16.R49 explicitly requires this paragraph and A33
requires it in the first prompt. `TestALaunchSpecTaskStartsItselfAndBecomesRunning`
(`internal/server/task_dispatch_test.go`) covers the delivered instruction and reporting contract;
it was inspected, not rerun. The annotation itself is the live reproduction. History traces the
contract to `4ddedff1` and the expanded coordination guidance to `a674e6bc`.

No investigation fix finding: the shipped behavior matched FS-16.R49 and INV §21. The approved
replacement (FS-16.R60, FS-18.R22) was implemented on 2026-11-21 and removed the appended
paragraph; see the “Inline task assignments” review unit.

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

### Usability review J1–J17 + X1–X7 (2026-11-21 at `10d01e1b`) — **Fix model:** medium — Codex Terra or Claude Opus.

Repros and evidence: [run record](../archive/reviews/usability-review-run-2026-11-21.md). Homes
remain under `/tmp/chuck-ur-2026-11-21/work/` while that directory lasts. Its Must-fix items
closed on 2026-11-21 by the usability fix commit; only Worth-fixing items remain (trivial/easy).
The per-attempt run cards are proven in jsdom only, not re-checked in a real browser.

- **Worth fixing** (trivial/easy) — J2 onboarding: each Check again PUTs backends and makes the last
  checked backend (e.g. OpenHands) the default; reloading mid-wizard drops the remaining steps once a
  backend passes (onboarding stays incomplete); Launch lands on Projects instead of the agent; a
  placeholder "My App" exists while the step says "Create your first project" and stays default;
  sign-in copy says "below" where nothing is; OpenCode/OpenHands still give no install command;
  wizard labels are not associated with inputs.
- **Worth fixing** (trivial/easy) — J9 settings: role id error shows the raw regex; an empty model id
  error shows internal keys and a generated id, only beside the distant Save; text typed in
  Additional directories without Add is silently dropped on Update.
- **Worth fixing** (trivial/easy) — J3/J4 chat: file-viewer header scrambles absolute and `../`
  paths (`etc/hosts/`, `outside.txt/..`); permission requests raise two stacked toasts that outlive
  the prompt and a toast appears when skip-permissions auto-approves; at 1280×800 the transcript is
  about 244px tall (130px with the viewer); a long unbroken token overflows the bubble; an
  intermittent "Conversation history may be incomplete" notice appears after a clean `dashboard stop`
  (seen on three agents, not reproduced on a fresh one).
- **Worth fixing** (trivial/easy) — J6 terminal: shrinking the window does not re-fit the terminal
  (overflows its card); Resume of a terminal agent without a recorded conversation id returns 409
  with no visible reason.
- **Worth fixing** (trivial/easy) — J7/J8/J11: a send error ("Failed to fetch") stays under the
  composer after reconnect, and controls look live while reconnecting; disabled Stop has no reason
  and disabled Clone on a stopped agent says the runtime cannot fork; Archive's counter counts
  projects ("1 result" for 58 agents); untagged search misses the model name.
- **Worth fixing** (trivial/easy) — J13: tray validation says "needs an anchor, excerpt, and
  instruction" without naming the item or the missing field.
- **Worth fixing** (trivial/easy) — J14: restart-paused runs still offer Open agent with code-like
  "owner not reporting" copy (FS-14.A26/R48); paused-run actions have inverted weight and Stop run and
  Delete have no confirmation; the shared-workspace 409 dialog lists "agent: Spec" uncapped with no
  run names; Start-run dialog still fades under reduced motion; CLI `pipeline` errors hide the API
  diagnostics; 121 scripted `run stop` calls left runs paused (unconfirmed).
- **Worth fixing** (trivial/easy) — J15/J16 tasks: the create form keeps the previous Wait-for
  prerequisite, silently arming the next task; raw `display_name`/`signal_name`/`dependency_failed`
  in errors and options with no limit stated; Delete has no confirmation; parked reason stays
  generic; budget-queued rows do not say they wait for a slot.
- **Worth fixing** (trivial/easy) — X1 Think Tanks: Pause during a turn gives no "pausing after this
  turn" acknowledgement; ended rooms keep "<name>'s unfinished turn" rows for turns later retried; a
  loaded template silently turns existing agents into new duplicate-named agents.
- **Worth fixing** (trivial/easy) — X2: rendered Markdown in Project Files splits into misaligned
  columns (table and code squeezed left).
- **Worth fixing** (trivial/easy) — X4/X6: the phone says "That code did not work" for a declined
  code as for a wrong one (FS-20.R6); expanding Subscription quota on the desktop agent page leaves a
  one-line transcript at 900px height.
- **Worth fixing** (trivial/easy) — X7 CLI: with no pidfile the CLI falls back to the config port and
  can read another home's dashboard; positional arguments report `unknown command`; pipeline errors
  print raw JSON, `project show nope` leaks `/api/projects`, `task list --project nope` exits 0.
- **Worth fixing** (trivial/easy) — Matrix coverage: add §3 charters for Think Tanks, Project Files,
  phone remote control, operator CLI, quota and message actions (USABILITY-REVIEW §7).

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
