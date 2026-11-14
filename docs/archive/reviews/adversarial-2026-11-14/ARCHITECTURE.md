# Structural assessment and target architecture — 2026-11-14

Second pass of the [adversarial assessment](ASSESSMENT.md) at `0a8bda69`. The first pass ranked
defects. This one asks which design decisions produce them, including decisions the specifications
state on purpose, and what to build instead. No product code, test or specification changed;
nothing is recorded in the handoff or committed.

**Operator decision, 2026-11-14.** The restructuring is accepted with no product or feature
change: nothing in section 3 is removed, and section 4.4, the origin check and the other
behavior-changing items are recorded in `docs/ideas.md` for their own decisions. A design review
the same day found that section 4.2's single request protocol and section 4.1's separate ledger
table also change behavior or duplicate existing rows; they are deferred the same way, and the
design keeps their seams (one turn begin, one admission function, the existing per-turn row).
ADV-1, cited below as a stranded-intent defect, was corrected the same day: the room routes
already cancel those intents, and only a latent weakness in the due pass remains.
The accepted part is designed in
[`give-agent-execution-one-owner.md`](../../../ready-changes/give-agent-execution-one-owner.md).

**Evidence.** Five read-only inventories: every representation of agent state; every path that
starts or continues agent work; every dependency on model behaviour; cost and coupling of twenty
mechanisms; and a root-cause classification of 399 recorded findings. The integrating reviewer
read `StartActivation`, the stall nudger, the quota continuation path, the status write at
`chat.go:1154`, the per-plane start bounds, the execution-handle round trip and TS-01.R18–R24
directly. Other inventory cells were not re-read. The classification marks about a quarter of its
rows uncertain; its percentages are orders of magnitude, not measurements. Usage figures are row
counts from the operator's live database, read-only.

## 1. Answers to the four questions

### 1.1 Does Chuck keep too many representations of agent state? Yes.

| What | Count |
|---|---|
| Durable facts about an agent's lifecycle, activity or next-turn owner | 31, across 17 tables |
| In-memory structures keyed by agent | about 27 (about 40 as raw maps and fields) |
| Call sites answering "may this agent take a turn now?" | 30, for about 14 distinct questions |
| Call sites that write the `status` row | 17 |
| Reconcilers that make two representations agree again | 16; eight run every 2 s |
| Durable records of a turn | none |

The last row is the root. A turn is the unit every plane cares about, and it has no row. Its
outcome lives in the transcript; its identity is copied into `tasks.execution_turn`, three pairs
of quota columns, `think_tank_attempts`, `owner_stall_turn_key`, `inline_delivery_turn_key` and
`turn_budget`; "a turn is open" exists only in memory (`turnActive`). `status.state` therefore has
to be a separately written fact, and after a restart each plane guesses what happened to its turn
by its own rule.

The same question is answered from different inputs. "Idle" is `status == idle` in three places,
`idle or error` in two, and those plus three in-memory flags in `StartActivation`. "Running" is
the registry map in one caller and the `running` row in three, and a row left by a previous
process passes the second and fails the first. "Assigned elsewhere" has four copies of one rule
(`quotaUnassignedTx`, the task unique index plus a room-attempt probe, a task count inside
`BeginThinkTankAttempt`, and `AssignedTask`).

### 1.2 Do the start paths share one lifecycle semantics? No.

Twelve paths were traced: Launch, Resume, Switch, human Send, mail wake, task start, task
continuation, quota continuation, pipeline stage and nudge, room turn, schedule, clone. They share
two things, the in-memory lifecycle claim and the runtime turn gate, and Send skips the first.
Above those sit **eight distinct claim/attempt protocols**, each with its own retry rule, failure
surface and restart rule, and **22 places where one plane reads another plane's tables** to decide
whether it may use an agent. No function sees every turn start: there are two turn-start code
paths, each with its own provider goroutine, and they disagree (one ignores a failed `busy` write,
`chat.go:1154`; the other treats it as fatal, `:1857`).

Fifteen differences have no stated reason. The ones with consequences:

- **Agent busy** is held, released, deferred free, deferred 2 s, left for a sweep, silently
  skipped, charged as a failed start, or refused, depending on the caller.
- **A crash mid-turn** interrupts a task, marks a quota intent uncertain, fails a room attempt and
  pauses the room, deletes a mail activation, and charges a retry to a task on a created runtime.
- **The archive lease** is taken by quota and room turns on a running agent and not by mail,
  borrowed tasks or the nudge, so only some turns block an archive.
- **Process-start bounds** are per plane (8, 8, 8, 4 and 128), and the task concurrency budget
  applies only to tasks; mail, room and quota wakes start processes outside it.
- **Only quota** holds two global locks across a full provider resume.

This is specified. TS-01.R19 says an activation "does not define universal coalescing, retry,
successful-start, or completion semantics. Those belong to the source domain." The intent was to
keep mail's at-most-once rule from leaking into tasks. The effect is that every new plane brings a
new protocol and must add reads against every earlier plane, so the join count grows with the
square of the planes. Quota shows the cost: it had to learn four owner kinds, and
`StartActivation`, the "one small primitive", now inlines quota-state checks and per-kind
branches.

### 1.3 Is the model/host boundary in the right place? No, in both directions.

**Given to the model, though the host has the facts.**

- **Waiting.** A task is waiting only if the agent calls `wait_for_tasks` and then ends its turn.
  The host already knows the turn ended, the assignment is open and its children are unfinished.
  About 600 lines (`pending_yield`, `resume_needed`, cleanup retries) exist for this handshake.
- **Finishing.** A result exists only if the agent calls `report_task_result`. When an ordinary
  task's agent simply ends its turn, nothing notices: the stall check covers pipeline owners only
  (`pipeline_stall.go:20`) and the task stays `running`. Pipeline owners get up to three nudge
  turns, then the run pauses for a person.
- **Identity fences.** The agent must fetch an `execution_handle` and echo it into every wait and
  report, and repair a `wait_conflict` if it gets it wrong. The server already knows the caller's
  agent, generation and open turn from the per-launch token.
- **Truth of completion.** `complete` checks that the named evidence tasks finished; whether the
  evidence still applies is the owner's attestation, and stage outputs are opaque text.
- **Room contributions.** A turn that does not call the submit tool fails and the room is held for
  a person; there is no retry.

**Fixed by the host, though judgment or patience would serve better.**

- Three failed starts park a task, and a parked task never restarts on its own, so a transient
  provider outage needs a person.
- Recovery rounds, stall turns and the attempt count are constants, not template settings.
- Work after a report must arrive as a new task; an agent cannot continue in place.

**The protocol is large.** Twenty-seven tools, about 14,000 characters always in context and
38,000 more in the operating skill, including host bookkeeping the model does not need (the
recovery-round count, handles, run and task ids). The one field report in this area, a worker
inventing a budget from its own configuration, fits that pattern.

### 1.4 What could be removed? See section 3.

## 2. How much each root cause explains

From the classification of 399 findings (uncertain rows included):

| Shape | Share of all | Share of the core server audits (about 183) |
|---|---|---|
| Client showed stale or wrong state | 24% | — |
| Two planes disagreed about an agent or item | 14% | about 11.5% |
| Chuck assumed something about the provider | 9.5% | — |
| Turn and process concurrency | 8% | about 14% |
| Input acceptance and receipts | 6% | — |
| Start paths built the same thing differently | 5% | about 10% |
| A post-commit notification was lost | 4.5% | about 10% |
| Column contract, sweep liveness | 4% together | growing in November |

No single cause dominates, so no single change fixes Chuck. Three readings matter. The ownership
family (cross-plane, concurrency, composition drift, lost propagation, column contract) is roughly
half of the core server findings, and section 4.1–4.3 addresses it. Five of nine field bugs,
the ones a person actually hit, are provider-boundary assumptions (section 4.5). The largest
single shape is the client (section 4.6), which this assessment inventoried only lightly.

Explicit recurrence is recorded eighteen times in the project's own documents: the same shape
fixed in one place and found in the next.

## 3. What can be removed

Usage in the live database: 105 agents, all chat; 68 tasks, 5 with dependency arms; 0 schedules;
7 pipeline runs; 2 rooms; 1 quota continuation ever.

**Remove now — no capability in observed use is lost.**

| Mechanism | Size (source / test lines) | Why |
|---|---|---|
| Terminal-interface agents and the hooks channel | 2,750 / 2,620 | 0 of 105 agents. Hooks are terminal-only; chat derives status in-process. Removal deletes a status writer, per-launch hook tokens, hook-settings files, the terminal WebSocket (the widest route in A6) and the INV §6 burden that every contract be met twice. Product decision. |
| Quota snapshot pre-check | 1,080 / 760, three adapter patches | Only improves postponement; continuation works without it. The largest dependence on patched adapters. |
| Legacy pipeline attempts and reset | about 190 plus dead writers | No production caller. |
| About 60 exported store functions with no production caller | — | Several are unguarded writers (`UpdateTaskCAS`). |
| Stubs that ship disabled | small | Primer summarizer, detached config import, messaging stdio fallback, `notImplementedRuntime`. |

**Fold into the turn ledger (section 4.1) — the capability stays, the mechanism goes.**

| Mechanism | What replaces it |
|---|---|
| `activations` table and its five kinds | A requested turn |
| Quota owner kinds, claim tokens and three turn-identity pairs | The blocked turn, re-requested with a not-before time |
| `continuation_pending`, the active-assignee index, four "assigned elsewhere" rules | One engagement field on the agent |
| Runtime-claim kinds created / woke / borrowed | Who started the process, recorded once on the execution |
| Room attempt states | Turn states |
| `pending_yield` handshake | Host settlement at turn end (section 4.4) |

**Decide as product questions.**

- **Quota isolated checks** (1,220 lines, undocumented adapter options). A plain backoff retry of
  the blocked turn gives the same recovery when the reset time is unknown.
- **Reset-time text parsing** (460 lines pinned to two CLI versions). With backoff as the fallback
  it becomes an optimization that can fail safe.
- **Think Tanks** (6,450 lines, 9 tables, 36 coupled files; 2 rooms ever) and **schedules**
  (1,700 lines; 0 used). Keep, freeze, or cut; either way they should stop carrying their own
  turn protocol.

## 4. Target architecture

Six changes. The first three are one design and remove the ownership family at its source.

### 4.1 A turn ledger and one execution record per agent

**Decision replaced:** a turn is not a durable entity; each plane tracks "its" turn.

- **`turns`**: turn id, agent, generation, origin (kind and reference: receipt, mail batch, task
  attempt, room turn), state `requested → admitted → running → ended`, typed outcome (TS-01.R42's
  outcome, stored), attempt number, not-before time. A partial unique index allows one row per
  agent in `admitted` or `running`: the database, not a memory flag, enforces one turn at a time.
- **`agent_execution`**, one row per agent: the live process (generation, pid, birth, who started
  it), the current engagement (which task or room the agent is committed to, or none), and
  blocked-until.
- **Status is a projection** of those two, computed by one function. The 17 writers go. A dead
  agent cannot show `busy` (ADV-4), because an open turn without a live execution is, by
  definition, interrupted.
- **One restart rule.** `admitted` returns to `requested`; `running` without a process becomes
  `ended (interrupted)`. The origin's policy decides what follows.

### 4.2 One turn-request protocol; policy stays with the domain

**Decision replaced:** TS-01.R19's per-domain semantics.

- A plane that wants a turn inserts a `requested` row in the transaction that commits its own
  fact. TS-01.R20 already requires this for activations. No plane starts a turn itself.
- One scheduler per agent admits the next request in a stated priority order, starts the process
  if needed, commits `running` before the provider frame, and at the end writes the outcome and
  calls the origin's `settle` in the same transaction.
- What R19 wanted to protect becomes a small policy table per origin: maximum attempts, what to do
  on an uncertain delivery (drop, retry, hold), backoff. Mail stays at-most-once as a row in that
  table, not as a protocol.
- **Quota becomes a property of the turn.** A turn that ends quota-blocked returns to `requested`
  with a not-before time, keeping its origin. Owner kinds, claim tokens and the room-hold release
  disappear; deleting a room deletes its requests by foreign key (ADV-1). Cancel is deleting the
  request.
- Cross-plane exclusion is the engagement field plus the priority order. The 22 pairwise reads and
  30 predicates reduce to the scheduler's admission function.

### 4.3 One way to make an agent run

**Decision replaced:** each plane composes its own start.

- `EnsureRunning(agent)` is the only caller of launch and resume composition, with one spec
  builder, one registration, one teardown and one process-start budget for every origin.
- Launch, Resume, Switch, Stop, Archive, Rename and Delete become commands on a per-agent serial
  queue. That queue replaces the in-memory claim maps (`lifecycleBusy`, `switching`, the archive
  leases, per-agent quota actions, task start locks), whose ordering is by convention today.
  Rename-during-handshake (CR-A10-3) and stop-during-wake become orderings, not races.
- The process fixes from ADV-5 and ADV-9 land here: group kill after reap, parallel stop, a file
  lock on the home.

### 4.4 The host settles every assignment turn

**Decision replaced:** durable progress requires a tool call.

- At the end of any turn whose origin is an assignment, the host records exactly one of:
  **reported**, **waiting** (the assignment has unfinished children), or **ended without result**.
  The third is a visible state for every task, with one policy (a bounded nudge, then attention).
  Nothing stays `running` silently.
- `wait_for_tasks` and `report_task_result` remain as the way an agent states intent and returns a
  semantic result. They stop being the only way the host learns that a turn is over.
- The host derives the execution from the caller's token and open turn. Handles leave the
  protocol, and with them `wait_conflict` as something a model must repair.
- Stages may declare checks the host runs (a command's exit status, a file's existence).
  `complete` then rests on verified facts where they are declarable and on the owner's judgment
  only where they are not.
- Transient start failures retry with backoff; parking for a person is for failures that a retry
  cannot fix. Budgets become template settings.
- Bookkeeping leaves agent-visible text.

### 4.5 A provider port with proof

**Decision replaced:** provider behaviour is assumed and then patched.

- One narrow interface for everything Chuck asks of an adapter. Each capability is either echoed
  back and recorded as effective (INV §20), or is behind a named fallback.
- A conformance suite replays captured real frames for each pinned adapter version in the release
  gate. The test doubles are derived from those captures, not written by hand.
- Vendor `_meta`, adapter patches and text parsing are listed in one place with their fallback.
  Removing the snapshot patches and the isolated check shrinks that list to fork and steering.

### 4.6 One client synchronization rule

**Decision replaced:** each feature wires its own events and invalidations.

- Every server resource family carries a revision. The stream carries the transcript, which is
  already sequenced, and otherwise only "family X is now at revision N".
- Reconnect invalidates everything. The omitted-family defect, found twice in separate audits,
  cannot recur.
- Desktop and phone share one data layer, as the input-delivery work has begun.

This is the least investigated proposal here; the client deserves its own inventory before design.

### 4.7 Two rules that outlive the migration

- **Derive, do not notify.** A control loop is a function from durable rows to actions. A
  post-commit call may only wake a loop early. This is how pipelines already work and how park
  propagation, room-to-run projection and turn-end release do not.
- **A bounded sweep changes every row it visits or rotates a cursor.**

## 5. Alternatives considered

- **Keep eight protocols and add guards** (the first pass's A1–A4). Much cheaper, and correct if
  no further plane is planned. It leaves the quadratic growth in place.
- **Event sourcing.** Not needed. Rows plus the ledger give the same recovery without a replay
  engine.
- **An actor per conversation inside the runtime.** The per-agent queue in 4.3 and the single
  turn-start in 4.2 remove most entry paths into `chat.go`; revisit only if defects persist after
  that.
- **A generic workflow engine.** TS-01.R19 was right to refuse this. The ledger is a record of
  turns, not a DSL.

## 6. Migration

Each stage ships on its own and is guarded by a count the repository can assert in a test, as the
untracked status-writer inventory already does.

| Stage | Change | Guard |
|---|---|---|
| 0 | Removals from section 3; fix ADV-1 to ADV-3 | Dead-code and usage inventory |
| 1 | Write `turns` in shadow: one `beginTurn` and one `endTurn` in the runtime, no behaviour change | Assert status equals the projection after every store test; log divergence at runtime |
| 2 | Status becomes the projection | Status writers 17 → 1 |
| 3 | One turn-start path; mail and dependency activations become requests | Turn-start paths 2 → 1; `activations` dropped |
| 4 | Engagement field replaces the assignee index and its flags | "Assigned elsewhere" rules 4 → 1 |
| 5 | Quota becomes a re-requested turn | Quota states and fences shrink; owner kinds gone |
| 6 | `EnsureRunning` and the per-agent command queue | Claim maps removed; one start budget |
| 7 | Host settlement at turn end; handles leave the protocol | No task `running` with a settled turn |
| 8 | Rooms and pipelines consume turn outcomes through `settle` | Protocols 8 → 1; pairwise reads 22 → 0 |

**Risk.** Stages 1 to 3 touch `chat.go`, the file with 80 commits in 60 days. Shadow mode and the
projection assertion are what make that tolerable, and the removals come first so there is less to
migrate. Stages 5 and 8 change specified behaviour (TS-01.R19, the quota sections of TS-10, FS-21's
attempt rules) and need `/design-feature` first. The end state has fewer states than today, which
is the test of whether a stage is going the right way.
