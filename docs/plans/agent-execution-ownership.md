# Give agent execution one owner

This is sequencing guidance for the
[ready change](../ready-changes/give-agent-execution-one-owner.md). The linked specifications
remain authoritative. Evidence, probes and inventories are in
[`docs/archive/reviews/adversarial-2026-11-14/`](../archive/reviews/adversarial-2026-11-14/ASSESSMENT.md).

**Rule for every slice.** Product behavior does not change. What a person or an agent sees is the
same before and after, except that a defect listed below stops occurring. A slice that cannot
keep an existing feature test passing unchanged stops and records the conflict as a question; it
does not edit the test or the feature specification.

**Rule for consolidating.** Each consolidation replaces several call sites with one function.
First write the function with a table test that pins today's result for every site it will
replace, re-read from the code, with the inventories as the checklist. Then switch the sites one
at a time. A site whose present result the table cannot reproduce is a question, not a judgment
call.

**Rule for a defect.** Reproduce it through the real route before fixing it. One finding in the
assessment (ADV-1) was reproduced at the store and did not occur through the HTTP route, where a
wrapper already handled it.

## Slices

Slices 1, 2, 3, 6 and 7 can start in any order. Slice 4 needs 3. Slice 5 needs 1 and 3.

| # | Slice | Requirements | Closes |
|---|---|---|---|
| 1 | Task states own their columns | TS-01.R50 | ADV-2, CR-A10-1, CR-A10-2, CR-A10-5 |
| 2 | Derived effects are recomputed; passes cannot be pinned | TS-01.R49 | ADV-3, CR-A10-4, what is left of ADV-1 |
| 3 | One turn start and end; the per-turn row records the turn | TS-01.R45, TS-02.R67 | ADV-4, ADV-7 |
| 4 | Status is composed by one function | TS-01.R46 | ADV-8 |
| 5 | One function decides admission | TS-01.R47 | — |
| 6 | One way to make an agent run | TS-01.R48 | ADV-5, CR-A10-3, the second half of ADV-6 |
| 7 | Reads and reconnects | TS-02.R68, TS-08.R168 | The two reconnect omissions; the stale `Touch` publish |

### 1. Task states own their columns

- One helper per target state for `finished`, `dependency_failed` and the requeue move. Each sets
  the columns that state owns; route the four `finished` writers and three `dependency_failed`
  writers through them.
- One commitment predicate: a single SQL fragment and Go function used by the active-assignee
  index, `AdmitReadyTask`, `TaskAssignment`, `AssignedTask`, `quotaUnassignedTx` and the room
  turn check. Where two of those differ today, keep each caller's present result by parameter
  and record the difference; do not pick a winner. The index becomes
  `state IN ('starting','running','waiting') OR (state = 'ready' AND continuation_pending = 1)`.
  A migration first normalizes rows the old writers left (a continuation marker on a finished or
  parked task). `queuedInputTaskClause` stays separate.
- `HoldQuotaManualTurn` writes only from a state a continuation can follow and compares the
  revision.
- Delete exported store writers with no production caller, starting with CR-A10-5's seven and
  the legacy pipeline-attempt writers. Keep the legacy reset itself: other installations may
  still hold version-1 records.
- Add the row-shape assertion to the store test harness and extend the untracked
  `internal/state/status_writer_inventory_test.go` from counting writers to checking rows.
- **Evidence:** `task-park-and-slot-state-probes.go.txt` probe 3, driven through the dispatcher,
  becomes a regression; CR-A10-1's cancel → bind → failed-settle sequence at the server; both Go
  variants.

### 2. Derived effects are recomputed; passes cannot be pinned

- A parked task is a source fact its dependents can read. `stampArmsFromResults` and `RearmTask`
  park a task armed on an already-parked prerequisite, and startup re-evaluation parks
  dependents of a parked task that has no work result.
- `SyncRoomPhase` runs in the existing 5-second room pass; it writes and publishes nothing when
  nothing changed. `pauseStartupRun`, `SyncRoomPhase` and `ResolveQuota` re-read on a lost
  compare-and-swap or return the conflict.
- What is left of ADV-1: `claimDueQuota` returns without acting when the owner cannot be read,
  while `startQuotaContinuation` holds `owner_unavailable` for the same condition; and the quota
  due pass has no cursor, so eight rows it cannot claim stop every later one. Take the hold only
  when the owner is not found; on any other read error skip as today. Rotate a cursor either
  way. No known route produces an ownerless intent today.
- One test enumerates each derived effect with its pass and each bounded pass with its progress
  rule.
- **Evidence:** the task-park and restart probes become regressions;
  `quota-orphan-starvation-server-probe.go.txt` shows the pinned pass with injected rows.

### 3. One turn start and end; the per-turn row records the turn

- Merge the two conversation-turn start paths (`runMessageTurnWithAssignment` and
  `StartActivation`'s provider goroutine) into one begin function with one provider call site.
  It refuses a stopped or exited generation and sets the turn identity in the critical section
  that claims the gate. Callers pass their own preparation. The quota check
  (`runtime/quota_check.go`) and Codex session naming (`nameCodexSession`) are exempt and listed
  in the scan.
- One end function; the transport-closed path uses it. It writes the row's outcome in the
  transaction that writes the turn-end status. A begin that fails after the row was written ends
  the row in the same function.
- Extend `turn_budget` and its reset helper with generation, origin, state and outcome; put the
  generation in the upsert's update list, because turn ids repeat across generations. The reset
  already runs on every chat begin: in its own transaction on the prompt path
  (`runtime/chat.go:1097`), in `AttemptActivation` for a running agent and `StartAttemptedTurn`
  for a wake (`state/activations.go:278, 304`), in `AdmitQuotaTurn`
  (`state/quota_admission.go:134`) and in `BeginThinkTankAttempt`
  (`state/think_tank_turns.go:151`). Keep those sites and pass the origin. The terminal hook's
  reset (`state/manager.go:169`) and the implicit row from `currentBudgetTx` write `ended` with
  no origin.
- Assert in tests that the row is `running` exactly while `turnActive` is set, that a turn which
  emits `turn_end` has the same outcome on its row, and that a Stop ends the row
  `process_ended` without a `turn_end`.
- Send `session/cancel` after releasing the turn lock; Stop signals the process before any
  operation that can block.
- **Evidence:** `runtime-queued-turn-after-exit-probe.go.txt` and
  `runtime-cancel-lock-across-write-probe.go.txt` become regressions; focused `-race` on begin
  against exit, Stop and Cancel.

### 4. Status is composed by one function

- Write the pure function in `internal/state` with a table test that has one row per entry of
  the status-writer table in
  [`INVENTORY-agent-state.md`](../archive/reviews/adversarial-2026-11-14/INVENTORY-agent-state.md),
  each asserting the state that site writes today. A crash shows `error` with its cause; Stop and
  a reconciled exit show `done`; a fresh generation shows `idle` whatever the previous
  generation's last turn did.
- `ApplyStaleCorrection` and `republishLive` keep the stored state: a pending permission lives
  only in runtime memory, so a writer without the runtime cannot recompute `waiting_input`.
- Add the assertion helper to the runtime and server harnesses, then switch the composing sites.
- The crash path records a `cancelled` resolution for every pending permission request.
- Check one case before switching: a permission resolved after its turn settled appears to
  leave `waiting_input` today (`runtime/permission.go:76–83`, `runtime/chat.go:2711`; from the
  design review, not reproduced). If it does, it is a defect that stops with this slice and the
  commit names it.
- **Evidence:** the writer inventory expects one composing site;
  `runtime-crash-permission-unresolved-probe.go.txt` becomes a regression.

### 5. One function decides admission

- The function replaces these thirteen entries of section 3 of
  [`INVENTORY-lifecycle.md`](../archive/reviews/adversarial-2026-11-14/INVENTORY-lifecycle.md):
  1–4, 6, 7, 9, 10, 15, 16, 18, 21 and the restart reads of 22. Write it with a table test of one
  row per entry, each asserting today's answer. It reads tasks through slice 1's predicate and
  the open turn through slice 3's row.
- These stay where they are: 5 (the in-memory lifecycle claim), 8 (the workspace rule), 11, 12
  and 19 (quota's own compare-and-swap writes), 13 (a read that feeds the room-attempt write),
  14 (lock keys), 17 (a count that decides a hold release), 20 (input routing) and the release
  write in 22. If one of the thirteen turns out not to be an admission read, it joins this list.
- Replace the reads one plane at a time: mail, task, quota, room, runtime, owner continuation.
  Most callers have no transaction; the function takes either a transaction or the store.
- Each plane keeps its claim tokens, attempt boundary, retry policy, periodic pass and restart
  recovery. This slice moves no state and changes no protocol.
- **Evidence:** the inventory test finds none of the thirteen sites reading another plane's
  table; the mail, task, quota, room and pipeline suites pass unchanged.

### 6. One way to make an agent run

- One spec builder for launch, resume and Switch; one stop-and-reap helper for the five stop
  copies, which ends the remaining process-group members after the leader is reaped. Shutdown
  stops agents concurrently, and `dashboard stop` waits longer than the server's shutdown budget
  plus one agent stop grace before it escalates.
- One function every origin calls to obtain a live process. Each origin passes the checks it
  makes today. The fifteen differences in section 4 of the lifecycle inventory stay exactly as
  they are and become visible arguments.
- A write after a provider handshake (`server/launch.go:180`, `server/resume.go:398`,
  `server/switch.go:245, 290`) updates only the fields the runtime reported. Rename takes no new
  claim and keeps returning what it returns today.
- Project delete with checkout deletion repeats its running-agent check after it holds the
  project claim, as archive already does.
- Quota continuation stops holding `configMu` and `quotaControlMu` across the provider start;
  `AdmitQuotaTurn` already compares the revision.
- A startup pass removes registration files and agent rows that a crash during a launch
  handshake left with no session.
- **Evidence:** `runtime-stop-group-descendant-probe.go.txt` becomes a regression; a rename
  during a handshake is not lost; one site builds a `LaunchSpec`.

### 7. Reads and reconnects

- A read-only connection with its own busy timeout for archive and project search and reference
  scans.
- `confirmQuotaProgress` consults an in-memory fact before reading the store.
- `Manager.Touch` recomputes and publishes under `writeMu`.
- The desktop stream client invalidates every server-backed query on open.
- **Evidence:** `store-single-connection-search-timing-probe.go.txt` as a bounded timing test.

## Progress counts

Each is asserted by a test when its slice lands.

| Measure | Today | After |
|---|---|---|
| Sites choosing a status state | 17, two of them detail-only | 1 composing function (slice 4) |
| Conversation-turn prompt call sites | 2 | 1, with two named exempt sites (slice 3) |
| Cross-plane reads that decide admission | 13 | 0 outside the admission function (slice 5) |
| Copies of the commitment rule | 4 | 1, plus the named queued-input predicate (slice 1) |
| `LaunchSpec` builders | 3 | 1 (slice 6) |
| Stop-then-reap copies | 5 | 1 (slice 6) |

## Findings this plan owns

Nothing here is also queued for `/fix`.

| Finding | Slice |
|---|---|
| ADV-1, reduced: the quota due pass can be pinned and skips an unreadable owner | 2 |
| ADV-2 parked continuation holds the agent's slot | 1 |
| ADV-3 parked prerequisite does not park dependents | 2 |
| ADV-4 queued turn starts on an exited process | 3 |
| ADV-5 Stop leaves group members; stop budgets | 6 |
| ADV-6, second half: project delete checks for running agents before its claim | 6 |
| ADV-7 Cancel holds the turn lock across a write | 3 |
| ADV-8 crash leaves a permission card live | 4 |
| CR-A10-1 cancelled quota wait re-armed | 1 |
| CR-A10-2 finished task keeps the slot | 1 |
| CR-A10-3 rename overwritten by a handshake | 6 |
| CR-A10-4 three writers report success after a lost compare-and-swap | 2 |
| CR-A10-5 uncalled store writers | 1 |
| CR-A10 transition-rule recommendation | Replaced by TS-01.R49 and R50 |
| Reconnect omits schedules and configuration sources | 7 |
| `Touch` publishes outside `writeMu` | 7 |
| Crash during Launch leaves an agent row and files | 6 |
| Crash mid cross-backend Switch leaves an agent Resume refuses | 6 (one builder, applied-fields write) |

## Not in this change

Each needs a product, security or compatibility decision and is recorded in `docs/ideas.md`.

- One request protocol for every plane, replacing the eight claim and attempt protocols. The
  2026-11-14 design review showed that it changes mail's attempt count, a person's Send during a
  host-initiated start and when a stopped pipeline owner is woken. Slices 3 and 5 are the seams
  it would be built on.
- ADV-9: one server per home proved by a held lock. Three review rounds each found a visible
  change in it, and a server from an older version never holds the lock, so it is a
  compatibility decision.
- ADV-10: what a run shows when its owner's quota intent needs attention. TS-09 specifies the
  present suppression, so showing anything else is a product decision.
- ADV-6's first half: checkout deletion and gitignored files.
- The loopback Origin check accepting any local port.
- Host settlement of assignment turns, implicit waiting, handles leaving the tool protocol and
  retry with backoff. Host-run stage checks are being designed separately under FS-14 §4.12.
- A `starting` task whose settle write failed is retried only at restart.
- `send_message` to oneself; the rollback message for a binary that will refuse the schema; the
  manifest archive name.
- Provider conformance against captured frames; revisioned client resources.
- Database durability level, backup and integrity check.
