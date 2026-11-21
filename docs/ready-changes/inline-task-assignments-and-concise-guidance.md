# Inline task assignments and concise Chuck guidance

**State:** Waiting to start
**Why:** Operator's 2026-11-21 `/design-feature` request following TASK-PROMPT, with inline delivery
approved after discussing its recovery and ownership implications.
**Relevant requirements:** FS-16.R60–R61/A42–A44, FS-18.R22–R23/A18–A19,
TS-01.R52, TS-10.R63–R65, TS-11.R25–R26; INV §1, §2, §5, §15, §16, §17, §19, §21, §22, §23.

## Outcome

An assigned agent receives its work and current execution context in the turn that starts it,
without an initial assignment-fetch round trip or repeated communication instructions. General
completion guidance lives once in awareness-gated shared context. One operating skill routes
workers to their basics and agents undertaking delegation to coordination guidance.

## Included work

- Remove the generic reporting suffix and fixed fetch-only task prompt. Fresh launches keep authored
  instructions first and verbatim. Existing-agent assignments and continuations begin with a short
  attribution/continuation heading and carry the verbatim instruction as assignment content;
  resuming must not invoke a leading skill/native command anew. Metadata is necessary context,
  not another instruction manual.
- Reuse one assignment projection for inline delivery and explicit `get_assigned_task` refresh:
  identity/current handle, attachment references, wait changes and pipeline facts/contract.
- Bind the admitted live execution before a provider can call tools. Preserve durable activation
  signals, task states, lifecycle admission, authoritative results and existing recovery policies.
- Replace shared context's vague reporting sentence with: “For assigned Chuck tasks, finish through
  `report_task_result` with success, failure or blocked and a short summary.” No chat-reply warning,
  mandatory initial lookup or fallback reminder for awareness-off agents.
- Add `worker-basics.md`, narrow `coordinate-work.md`, and shorten the skill entry's duplicate
  guidance. Keep agent operations, pipelines and rooms separately readable. Any authorized child
  can discover delegation and load its reference; these are activity routes, not role restrictions.
- Preserve task-specific pipeline decisions, room contribution contracts, tool authority and native
  command semantics. No new skill, setting, schema, delivery acknowledgement protocol, role-file
  migration, history purge, live prompt rewrite or product UI. Package updates use ordinary release
  publication; updating the installed runtime is not part of design or implementation.

## Evidence and design constraints

`internal/runtime/activation_kinds.go` currently appends `taskReportContract` after the instruction
and gives existing agents a fixed lookup trigger. `internal/runtime/registry_message.go` and
`internal/server/task_dispatcher.go` join the runtime through its existing turn seam.
`internal/messaging/task_tools.go` currently owns the assignment projection; factor it rather than
copying its wait/pipeline rules. `internal/server/launch.go` already owns awareness-gated context.

`ChatRuntime.beginTurn` schedules a provider goroutine before dispatcher's current
`recordTaskExecutionTurn`/`ConfirmTaskStart` calls. It must not expose the inline handle while that
binding is incomplete. The reserved attempt id already exists; confirmation copies it to the
execution handle. Extend admitted-turn preparation with one state-owned binding/confirmation
before provider execution; do not declare queued work running or invent another token.
`Transport.Call` has no separate provider receipt before its prompt result, so local confirmation
means admitted/staged, never remotely acknowledged. Stop can still win before the provider write.

Native dispatch is route-sensitive: the pinned Codex adapter's `Session.prompt` parses the first
text block (`scripts/release/node_modules/@agentclientprotocol/codex-acp/dist/index.js`, command
branches for `/review`, `/compact` and `/logout`). Moving a formerly pulled instruction there on
an existing/resumed task changes its meaning and can restart a command on continuation. Preserve
fresh-launch behavior only; a minimal assignment/continuation heading keeps those other routes
as supplied task content. Provider tests must check this boundary, not just string presence.

The skill is already progressively loaded: about 400 words in its entry, 6,000 across the package;
only the entry/selected references are read on demand. The coordination reference mixes worker and
delegator jobs. Reorganization improves the reading boundary; do not claim 6,000 words are currently
injected into every prompt or that the split alone proves better model behavior.

## How we will know it works

- FS-16.A42–A43 extend existing task dispatch/wait/pipeline/runtime fixtures: exact instruction and
  current metadata, immediate report/wait, no incomplete binding, stale-handle rejection and
  Stop/Cancel/restart/quota interleavings. Independent fixtures check the actual serialized payload,
  rather than calling the same renderer to construct expectations.
- TS-10.R65 verifies maximum legal Unicode/JSON escaping, pipeline assignments and coexisting mail;
  the derived envelope bound cannot reduce admitted task sizes or clip required execution data.
- FS-18.A18–A19 verify awareness on/off and unavailable-package composition, all five direct links,
  identical provider views and worker-to-delegator discovery. FS-16.A44 adds bounded pinned Claude
  and Codex smokes; record synthetic versus real-provider evidence and remaining acceptance gates.
- Run the applicable TS-06 closure once after final implementation edits. This design adds no tests
  or product changes; specification lint, twin-skill comparison and diff checks close design only.

## State and ownership

No new state is added. Each cell states next durable state / owner / observation. “Task service”
means the existing server/domain logic writing through `internal/state`; the runtime owns only its
claimed turn and provider effect. Source-specific pipeline pauses remain with the pipeline owner.

| State | Restart | Stop or cancel | Owner replaced | Input refused or failed | Quota or timeout hold | Late callback from previous owner |
|---|---|---|---|---|---|---|
| Unadmitted: armed, ready, missed, dependency_failed or interrupted | Retain state, re-evaluate eligible ready work / task service / no stored prompt replay | Cancel becomes cancelled; Stop has no unclaimed provider turn to submit / task service / cancellation or unchanged pending work | Retain eligible task or refuse unavailable target / task service / fresh attempt only after admission | Retain state on validation refusal; failed starts use bounded attempt/attention rule / task service / reason visible, no assignment sent | Retain eligibility hold / quota owner and task service / no provider call until eligible | Ignore obsolete attempt/generation / task service / no state change or delivery |
| Starting: reserved, not confirmed | Borrowed start becomes interrupted; created/woken runtime reaped before bounded retry, or retained starting if reap fails / task service / uncertainty or retry visible | Cancel records cancellation and claim cleanup; Stop prevents submission and triggers interruption/pause as applicable / task service and runtime / no unowned provider work | Reject obsolete generation and settle its reservation / task service / replacement needs its own admission | No provider call; release deferral without attempt charge, or count real failure and park at limit / task service / actionable reason | Hold/release through quota admission; never confirm a refused turn / quota owner and task service / pending work retained | Reject attempt/generation mismatch / task service / no confirmation from obsolete start |
| Running: locally confirmed, including staged-before-write | Generic running task becomes interrupted and claims release; quota-owned task preserves its episode / task or quota owner / no unverified replay | Stop interrupts or pauses its owner; Cancel becomes cancelled, cleans only the bound turn/claim / task service and runtime / queued human input remains recoverable if unsent | Old execution interrupted or resumed only through owning recovery policy / task service / fresh binding needed before replacement tools | Provider failure interrupts or establishes recognized quota hold; rejected report stays actionable / task or quota owner / typed failure, no invented result | Preserve unfinished assignment and episode; timeout is not quota evidence / quota or task owner / verified recovery only | Reject stale handle/generation/turn; terminal result immutable / task service / no cross-execution settlement |
| Waiting, including pending_yield | Preserve wait set; rediscover committed child changes for normal continuation admission / task service / inline current changes on next execution | Cancel becomes cancelled and removes active wait; Stop of a live yielding runtime follows owned exit/pause policy / task service / no fabricated completion | Retained assignment waits or receives admitted replacement execution on same eligible assignee / task service / fresh handle when re-admitted | Invalid wait changes nothing; release failure retains cleanup ownership for retry / task service / attention rather than duplicate worker | Preserve wait/hold; resume only unfinished eligible assignment / task or quota owner / current changes, no replay of old envelope | Ignore previous execution's report/yield/turn end / task service / wait and new execution protected |
| Finished or cancelled, including pending release | Retain immutable result/cancellation; finish outstanding claim cleanup / task service / no assignment rerun | Retain terminal state, idempotent cleanup / task service / no new result or turn | No replacement execution for this terminal task / task service / new work needs a new task | Reject report/wait/start mutations; retained cleanup may retry / task service / typed terminal refusal | No continuation of completed work; cancel supersedes hold / task service / no quota replay | Reject stale settlement or accept only matching idempotent cleanup / task service / result unchanged |

## Design review

Fresh-context review on 2026-11-21 checked all 30 transition cells and the current turn, task,
pipeline, quota and pinned-adapter boundaries. The native-command replay finding was resolved by
the attributed existing-agent/continuation route. The stale fetch-on-activation wording in
FS-18.R12 is explicitly included in planned supersession. No open material findings.

## Waiting on

None. Specifications are planned; implementation and its provider acceptance checks have not run.
