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
- **v0.26.0 candidate** (2026-11-26): prepared at `11b0a4b2`
  ([notes](../archive/state/RELEASE-v0.26.0-notes.md)); nothing agent-facing changed. Not tagged:
  the three review-queue units and ARCH-POST025-2/3/4 (Worth fixing) await the operator's
  decision to ship with them open.
- **Architecture:** the [post-v0.25.0 receipt](../archive/reviews/architecture/v0.25.0-next.json)
  is complete: source review at `1edbe326`, ARCH-POST025-1 closed by `992725bd`, and coverage for
  the later README, check-specs and release-notes commits. No release-blocking finding is open.
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

- **Phone skill picker** — FS-20.R57–R58/A29–A30 scope confirmed; TS-08.R176–R177 drafted.
  TS-13.R21 currently denies `available-commands` to paired devices. Decide whether to allow
  the existing read with provider-advertised names, descriptions and hints, or expose names
  only. Record the approved response in TS-13 and promote the idea to the ready queue.

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

- **Phone bottom pull to refresh** — available for independent review. One commit on `56f23882`,
  "phone: pull up at the newest message to refresh". Scope: FS-20.R59/A31, `pullUpRefresh.ts`,
  the phone chat hint. The paired-device gate is in ACCEPTANCE-GATES.

- **Invariant inventory maintenance** — available for independent review. Baseline `993d8dd0`;
  finding citation checker `22e5edd4` plus the inventory and startup changes closing this unit.
  Scope: INV §§1–24, TS-01 §4, TS-06 spec lint. Decision replays covered a replacement agent
  inheriting a cooldown and a private formatter interface outside the execution lifecycle.

- **Streamlined developer workflows** — available for independent review. Baseline `0cf9a04f`,
  model routing in `1d1e9614`, launcher/procedure migration in `b7b17477`, followed by the
  verification and handoff closure. Scope covers TS-06.R5/R35/R36, scoped role reads,
  review/fix authority and browser evidence. [Matched replay results](../archive/reviews/workflow-skills-2026-11-25.md).

**Start a first chat with Claude or Codex** is closed; its remaining same-cause completion/default
questions are an entry in `docs/ideas.md`.

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

### MIDTURN-SEND — regular message appeared delivered without a pending tail (2026-11-24)

The ANNOTATION-QUEUE report also described a regular mid-turn message appearing as a notification
and then an ordinary transcript row without Withdraw. No affected agent, time, clicked action,
notification wording or installed version was supplied. Fresh Send still queues behind an active
turn; explicit Steer and Send received after turn completion deliver immediately. Cause remains
undetermined, with no established code defect. Correlate the incident before selecting a fix;
the separately reproduced annotation queue-display defect is closed.

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

### ARCH-POST025 — combined source review at `1edbe326` — **Fix model:** medium — Codex Sol 6.1/6.2 or Claude Opus 5.5.

[Review evidence and probes](../archive/reviews/architecture/v0.25.0-next-evidence.md).

- **Worth fixing** — ARCH-POST025-2: quota resume can take the last task runtime slot without
  invalidating the open Tasks view. `internal/server/quota_dispatch.go:282–299` commits the
  borrowed-to-woke claim before attempting Resume. If Resume refuses before a process starts,
  such as a missing session snapshot, only a non-running `state_update` arrives. The live stream
  at `ui/src/api/sse.ts:196–208` skips that update for Tasks. The real REST and event-bus probe
  changes `waiting_for_runtime_slot` from false to true with no matching invalidation. Publish
  the capacity change at the committed claim boundary, including when launch fails, and verify
  the connected cache against a fresh read. FS-16.R7/R42, TS-10.R11/R17, INV §1/§8.
- **Worth fixing** — ARCH-POST025-3: a timed-out dynamic import in the SSE watchdog test can
  outlive module reset and contaminate the next revision-discard test (`ui/src/api/sse.test.ts:81–132`).
  An initially loaded run produced both failures. A controlled delay on the SSE module transform
  reproduced the same pair, and the ordinary isolated suite passed. Fence or finish import setup
  before resetting the module/store identity, and verify the timeout branch cannot yield a second
  misleading recovery failure. TS-06.R35, INV §17.
- **Worth fixing** — ARCH-POST025-4: the retired cutover still appears as an owed rehearsal in
  `docs/features/ACCEPTANCE-GATES.md:198` and as planned in TS-02.R41. Both refer to FS-10.R25/A14,
  which now record that the operator completed the cutover and retired it. Reconcile the remaining
  planned text and checklist entry to the recorded retirement so release work has one answer.
  FS-10.R25/A14, TS-02.R41, INV §10.

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

- **AUTH-TRUST-3** (2026-11-25 `/fix`): the installed managed runtime is still v0.20.0, so the
  finding cannot be checked at its real entry point. Needed from the operator: install v0.24.0 or
  later, enable the provider's automatic review, approve a push/PR through Continue, and record
  whether `create_task` is allowed. The finding stays open in **Review findings** until then.

The file reader's withdrawal test already matches the exact recovery error.
The full UI suite passed all 1,294 tests on 2026-11-19.

The published `v0.11.0` release body remains empty; its notes are in
[`RELEASE-v0.11.0-notes.md`](../archive/state/RELEASE-v0.11.0-notes.md).
Editing that older release is separate publication work.

## Release record

Prepared notes: [v0.26.0](../archive/state/RELEASE-v0.26.0-notes.md) (candidate, untagged).
Latest published version: [v0.25.0](https://github.com/AsaphNoam/Chuck/releases/tag/v0.25.0).
