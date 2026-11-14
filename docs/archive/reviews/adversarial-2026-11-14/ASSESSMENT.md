# Adversarial correctness and architecture assessment — 2026-11-14

Explicitly requested whole-system assessment at `0a8bda69` (clean `git archive` exports; the
working tree's uncommitted workflow edits were not read as evidence). This is not the TS-06.R36
release review: it writes no receipt, records nothing in the handoff, changes no product code,
test or specification, and is not committed. Finding IDs use the `ADV-` prefix so an owner can
lift them into the handoff. The design decisions behind these defects, and the target architecture
that removes them at the source, are in [ARCHITECTURE.md](ARCHITECTURE.md). Since 2026-11-14 the
findings that need no product decision are owned by
[`give-agent-execution-one-owner.md`](../../../ready-changes/give-agent-execution-one-owner.md);
the rest are in `docs/ideas.md`.

**Method.** Seven independent read-only probes (runtime turn lifecycle; task control plane;
persistence and process recovery; messaging and wake; lifecycle and workspace; pipelines, rooms
and quota; UI sync, remote control and updater), each told which findings CR-A1–A10 and the live
handoff already own. Every defect ranked below was re-checked by the integrating reviewer: the
probe was re-run from its own cache, or the cited source was read. Probe sources are archived
beside this file as `*.go.txt`; each ran in a throwaway export under `/tmp` against the real store,
the real runtime and the repository's fake ACP adapter.

**Evidence levels.** *Reproduced* — an executed probe shows the wrong durable state or output.
*Source* — traced and read, not executed. *Measured* — a timing taken on a private copy of a real
49 MB `state.db` (105 conversations), deleted afterwards; only durations were recorded.

**Limits.** Fake provider only: no credentialed Claude/Codex run, no phone or tailnet, no browser.
No full suite was run; `go test -race -tags sqlite_fts5` over `internal/runtime`, `bus`, `pipeline`
and `state` passed with no race report. ADV-4's interleaving is forced with test hooks. The real
adapters' process-group and exit behaviour (ADV-5) was not observed.

## Verdict

The storage and delivery foundations are sound and well proved: the conversation journal, turn
settlement against process exit, task admission, schedules, inline-mail settlement, migrations,
file save and the remote guard chain all held up, and five weeks of the operator's own dashboard
log show 25 starts, no `ERROR` line and 29 warnings.

The control plane above them does not yet reliably keep its promise that work either progresses or
asks for attention. Ten subsystem audits each found five to eight defects; this pass found about
ten more in the same four shapes (one, ADV-1, was later corrected to a latent weakness). The rate is not falling, which says the cause is
structural: these four shapes are conventions that each new writer must remember, not mechanisms.
The invariant catalogue already records that only rules with a guard stop recurring. The highest-
value work is therefore four small guards (A1–A4 below), not another audit and not a rewrite.

## 1. Confirmed correctness defects

Ranked by consequence × likelihood.

| ID | Defect | Evidence | Confidence |
|---|---|---|---|
| ADV-1 | **Corrected 2026-11-14: the stated trigger does not occur.** What remains is latent: the quota due pass skips an intent whose owner cannot be read and has no cursor, so eight such rows would stop every later continuation | Pinned pass reproduced with injected rows; no route produces such a row | Low |
| ADV-2 | A parked continuation start keeps its agent's exclusive slot; that agent's other tasks stay `ready` forever | Reproduced (store) | High |
| ADV-3 | Dependents of a parked prerequisite stay `armed` with no attention: after a lost propagation, and whenever created or re-armed later | Reproduced (store + server) | High |
| ADV-4 | A queued follow-up that starts as the process exits leaves a dead agent `busy`, the message unsent and its receipt `unknown` | Reproduced (forced interleaving) | High mechanism, medium frequency |
| ADV-5 | Stop does not guarantee the process group is gone | Reproduced (group) + source (budgets) | High mechanism; adapter impact unverified |
| ADV-6 | Consented checkout deletion destroys gitignored files the dirty check never reports; project delete checks for running agents before taking its claim | Source + git behaviour (ignored files); source (race) | High / medium |
| ADV-7 | Cancel holds the turn lock across a stdin write; a wedged child makes Stop hang | Lock-holding reproduced; trigger reasoned | Medium |
| ADV-8 | A crash with an ordinary permission request pending leaves Approve/Deny live forever | Reproduced | High (low consequence) |
| ADV-9 | The single-instance guard is a pidfile plus `kill(pid, 0)` | Source | Medium |
| ADV-10 | A pipeline run whose owner's quota intent needs attention stays `running` with no attention | Source | Medium (matches spec text) |

### ADV-1 — The quota due pass can be pinned (corrected)

**Correction, 2026-11-14.** This finding first said that deleting a paused room strands its
members' quota intents. That is wrong. The room routes are wrapped by `wrapQuotaControl`
(`internal/server/quota_controls.go:294–338`), which snapshots the room's intents and cancels the
unstarted ones after the action succeeds. An end-to-end probe through the real routes
(`quota-room-delete-through-routes-probe.go.txt`) shows the intent `cancelled` after Pause, no
due intent after Delete, and the agent not blocked. The original reproduction called the store's
`DeleteThinkTank` directly and injected ownerless rows by SQL, so it bypassed the wrapper. The
design review of the follow-on change found this.

**What remains.** `claimDueQuota` (`internal/server/quota_dispatch.go:53`) logs and returns when
an intent's owner cannot be read, while `startQuotaContinuation` holds `owner_unavailable` for
the same condition one step later. `DueQuotaIntents` (`internal/state/quota.go:336`) is
`ORDER BY due_at, agent_id LIMIT 8` with no cursor. With eight injected ownerless rows, a valid
intent stayed `scheduled` through five passes (`quota-orphan-starvation-server-probe.go.txt`).
No route is known to produce such a row, so this is hardening, not a defect a person can reach.

**Direction.** Take the existing hold on an unreadable owner and rotate a cursor (A3).

### ADV-2 — A parked continuation holds the agent's slot

**Cause.** `ParkTaskStart` and `FailTaskStart` (`internal/state/tasks.go:1586–1608`) move a task to
`dependency_failed` without clearing `continuation_pending`. The unique index
`idx_tasks_active_assignee` counts that flag in any state, while `TaskAssignment` treats only
`ready AND continuation_pending` as an assignment. Admission swallows the unique violation.

**Evidence.** Probe: wait → yield → continuation prepared → admitted → parked gives
`dependency_failed, continuation_pending = 1`; a second task for the same agent then returns
`admitted=false err=nil` and stays `ready`.

**Impact.** Any continuation that exhausts its three starts (provider down, `session/load`
refused) or meets an archived target silently freezes every later task for that agent. Same leak
as open CR-A10-2 through a different writer; CR-A10-2's proposed bound ("unfinished states") would
not cover it, because `dependency_failed` is unfinished.

**Direction.** A1 below. The precise index predicate is
`state IN ('starting','running','waiting') OR (state = 'ready' AND continuation_pending = 1)`.

### ADV-3 — Parked prerequisites do not reliably park their dependents

**Cause.** Four writers park a task and register no work result. Dependents are parked afterwards
by an in-memory call (`propagateTaskFailure`, `internal/server/task_dispatcher.go:868`) whose error
is logged at Debug. Startup re-evaluation calls only `EvaluateSource`, which returns immediately
when no work result exists, and no tick sweeps armed tasks. Separately, `stampArmsFromResults`
(`tasks.go:587`) consults only `work_results`, so a task created or re-armed against an
already-parked prerequisite becomes `armed`.

**Evidence.** Server probe: `first=dependency_failed second=armed attention=""` after
`recoverTasks` and three ticks. Store probe: create and re-arm onto a parked prerequisite both
yield `armed / unsatisfied`.

**Impact.** FS-16.R8 promises "nothing is silently left waiting". The second case needs no crash:
a start-time task that parks as "missed start time" while the Mac sleeps, followed by any
follow-on work, leaves that work waiting forever with nothing shown.

**Direction.** Make the park a durable source fact that both `stampArmsFromResults` and startup
read; A2 covers the missing sweep.

### ADV-4 — A queued turn can start on a process that has just exited

**Cause.** `finishTurn` hands the turn gate to the held message (`internal/runtime/chat.go:1340`)
at the moment `onTransportClosed` is released. The successor's turn id is set only after the mail
reservation and quota bind (`chat.go:1103–1121`), so the exit handler sees no active turn, emits
no `turn_end` for it and writes `error`; the successor then commits `user_text`, marks its receipt
submitted and writes `busy`. Nothing on the start path re-checks `stopped`.

**Evidence.** Probe transcript ends `turn_end(process_ended)` at seq 5 then `user_text "second"`
at seq 6 with no settlement; final status `busy / "thinking"` with no running row. The same
claim-then-start window exists in `StartActivation`, plain Send, promoted Steer and
`ContinueSettledChat` (source).

**Impact.** A provider that exits right after answering while a follow-up is queued leaves a dead
agent shown as thinking, the follow-up never sent, and a quota bind and mail reservation owned by
a turn that never settles. Startup reconciliation only visits running rows.

**Direction.** A4 below; the minimal repair is a `stopped` check inside `setExecTurn` plus an
ownership-checked `busy` write.

### ADV-5 — Stop does not guarantee the process group is gone

**Cause.** `agentState.shutdown` (`chat.go:3151–3181`) signals the group with SIGTERM and escalates
to SIGKILL only if the leader outlives the 5 s grace; when the leader exits in time, surviving
members are never signalled and the running row is deleted. The orphan path does sweep the group.
Separately `StopAll` (`runtime/permission.go:352`) stops agents one at a time ignoring its context,
under a 5 s server budget, while `chuck dashboard stop` SIGKILLs the server after its own 5 s
(`internal/cli/dashboard.go:29, 303–321`).

**Evidence.** Probe: a TERM-ignoring descendant in the agent's group was alive after `Stop`
returned nil. The budget mismatch is arithmetic from source.

**Impact.** FS-01.R6 says Stop terminates the process group. A surviving build, watcher or shell
keeps writing to the worktree with no record. With several agents, `dashboard stop` can kill the
server mid-`StopAll`; survivors keep live running rows that can only be Stopped, not resumed.
There is no parent-death binding, so after a server SIGKILL the only lever is the adapter's own
reaction to stdin EOF, which was not verified.

**Direction.** One `kill(-pgid, SIGKILL)` after the leader is reaped; stop agents in parallel;
make the CLI wait longer than the server's budget.

### ADV-6 — Checkout deletion can lose more than was disclosed

**Cause.** `IsDirty` runs `git status --porcelain` (`internal/worktree/git.go:387`), which omits
ignored files; `RemoveWorktree` runs `worktree remove --force` (`git.go:327`), which deletes them.
The code comment and FS-19.R8 both state that uncommitted changes are the only thing deletion can
lose. `handleDeleteProject` checks for running agents (`config_handlers.go:501`) before
`beginProjectArchive` (`:526`) and never rechecks; the archive handler does this in the right
order.

**Impact.** `.env.local`, local databases and ignored notes in a worktree are removed after a
dialog that reported the checkout clean. A start already inside its lease when delete arrives can
have its checkout removed underneath it.

**Direction.** Report ignored files separately in the probe and the dialog; rerun the running
check inside the claim.

### ADV-7 — Cancel can make Stop hang

**Cause.** Cancel sends `session/cancel` while holding the per-agent turn lock
(`runtime/permission.go:226–230`); the transport write is a blocking pipe write with no deadline
(`transport.go:258`). Stop takes the same lock (`chat.go:1455`) before `shutdown()`.

**Evidence.** Probe: with an unread stdin, the turn lock was still held one second after Cancel.
Not reproduced end to end: it needs a child that has stopped reading and a pipe already filled by
a prompt larger than the pipe buffer (an image attachment).

**Impact.** Control fails exactly when it is needed: the agent is frozen until killed externally.

**Direction.** Send the notification after releasing the lock; have Stop signal before anything
that can block.

### ADV-8 — A crash leaves a permission card live

`onTransportClosed` (`chat.go:2222–2232`) records the `cancelled` resolution only for host-owned
requests; Stop and Cancel record it for all. Probe transcript: `permission_request`, `error`,
`turn_end(process_ended)`, no resolution. The card stays actionable in the live view and on
replay; Approve fails. Record the resolution for every pending request, as Stop does.

### ADV-9 — Single-instance is a pidfile

`dashboard start` checks the pidfile at `cli/dashboard.go:183` and writes it at `:205`, after
`state.Open`, migrations and `ReconcileStale`. Two starts inside that window both proceed; with
different `--port` values both serve one home, breaking TS-01.R10's sole writer. A doomed second
start on the same port still runs `ReconcileStale`, `ReleaseRestartedAttachments` and
`recoverAttachmentStorage` against the live server's database before its bind fails, then its
deferred cleanup deletes the winner's pidfile. After a reboot a reused PID makes `start` print
"already running" and exit 0, and `stop` signal an unrelated process. Agents get birth-time
identity proof; the server does not. An `flock` on `home/.lock` closes all of these.

### ADV-10 — Quota attention does not reach the run

`nudgeStalledOwner` returns whenever `quotaBlocksAgent` is true (`server/pipeline_stall.go:32`),
including `needs_attention`, and `internal/pipeline` never reads quota. A stage owner whose reset
cannot be parsed ends in `needs_attention / check_exhausted` after six hourly checks; the run
shows `running` indefinitely. TS-09 specifies the suppression, so this is a design gap: project
the intent's attention onto the run, as room phase is.

### Lower-priority items, each small and local

- `send_message` accepts the sender as recipient and creates a wake for it (reproduced by the
  messaging probe); a one-line refusal.
- `--rollback` reports success for a binary that will refuse the migrated schema (source).
- `Manager.Touch` recomputes and publishes outside `writeMu`, so a stale snapshot can overwrite a
  newer one until the next change (source; microsecond window).
- The reconnect invalidation list omits schedules and config sources (source).
- A task left in `starting` by a failed settle write is retried only at the next restart (source).
- A crash during Launch's handshake leaves an agent row nothing removes; hook-settings and MCP
  files leak per crash (source).
- A crash mid cross-backend Switch leaves an agent Resume refuses, safely (source).
- The manifest's archive name is not restricted to a base name before download (source; requires
  a compromised release).

## 2. Architectural weaknesses

### A1 — Status columns have many writers and no column contract

**Evidence.** 35 `UPDATE tasks` statements, 16 on `quota_intents.state`, 15 callers of the run
compare-and-swap. `continuation_pending` has three different contracts depending on the writer:
of four writers to `finished` one clears it (CR-A10-2), of three to `dependency_failed` none does
(ADV-2), and interrupt does.

**Challenge to CR-A10's recommendation.** CR-A10 proposes one `transition(row, event)` rule per
machine. For tasks that is more than the evidence supports: neither CR-A10 nor this pass found an
illegal `(from, to)` move. Every task leak is a writer that reached a legal state without the
columns that state owns, which a legal-pairs table would not have caught. A SQLite trigger is not
the cheaper guard either: callers log store errors at Debug, so an abort would turn a slot leak
into a stuck `starting` row.

**Direction.** For tasks: one helper per target state (`finishTx`, `parkTx`, `requeueTx`) owning
its column list; one predicate shared by the index, `TaskAssignment` and `AssignedTask`; and a
row-shape assertion run after every store test (`continuation_pending` implies `ready` or
`starting`; `pending_yield` implies `running`), extending the untracked writer-inventory test from
counting writers to checking the rows they leave. One to two days, no migration beyond the index.
For quota intents CR-A10's from-set rule is justified, because CR-A10-1 is a from-set defect. For
runs, keep the revision compare-and-swap and adopt one conflict policy.

### A2 — Derived effects ride on one-shot calls after commit

**Evidence.** The codebase has a good pattern and applies it unevenly. Pipelines commit a durable
`pending_action` and re-drive it every 2 s; task waits are rediscovered each tick. But park
propagation (ADV-3), room phase onto its run (`SyncRoomPhase`, a goroutine on publish with no
periodic sweep; CR-A10-4), quota attention onto its run (ADV-10) and turn-end release
(`go s.dispatchTurnEnd`) each depend on a call that a crash or a lost compare-and-swap drops.

**Direction.** No outbox or event log is warranted. State one rule with a guard: every post-commit
propagation names the sweep that re-derives it from durable rows, and a test enumerates the pairs.
Three sweeps are missing today: parked-prerequisite dependents, `SyncRoomPhase` (add it to the
existing 5 s room sweep), and quota-to-run attention.

### A3 — Bounded sweeps can be pinned by rows they cannot advance

**Evidence.** CR-A2-6 fixed this for tasks and mail with rotating cursors; the quota due sweep
never got one (ADV-1), and admission leaves a refused task `ready` with no state change (ADV-2).
Same class, third and fourth occurrence.

**Direction.** One helper and one rule: a bounded sweep either changes each row it visits (hold
with a reason, or defer with a later due time) or rotates a cursor. A row that cannot be acted on
must become visible, never merely skipped.

### A4 — Turn state is a guarded accretion, not one owner

**Evidence.** Seven locks on the session path, eleven boolean flags, about nine turn or generation
fences, 22 status-writer call sites and 20 writers of `turnActive`; `chat.go` took 80 commits in
60 days. ADV-4, ADV-7 and the low-confidence lead that the crash path deletes the running row
without comparing generation are the same two shapes that TS-01.R42–R44 were meant to end: a
goroutine assuming another owns settlement, and a lock held across I/O.

**Options.** One actor goroutine per conversation would make these impossible by construction,
but the migration is high-risk: 3,700 lines, roughly 50 test files that read the lock and turn
fields directly, and ordering contracts with the journal's commit hook. Not recommended now.

**Direction.** Two bounded consolidations inside the existing structure. First, one turn-start
gate: the five entry paths each claim the gate, reserve, bind and set the turn id in their own
order; a single `beginTurn` that checks `stopped` and sets the id atomically with the claim
removes ADV-4's class. Second, a guarded rule that no transport write happens under the turn
lock. Reconsider the actor model only if a further defect appears that these do not prevent.

### A5 — The process boundary is safe only on graceful paths

ADV-5 and ADV-9 together: no parent-death binding, serial stop under a shared budget, group not
swept on ordinary Stop, pidfile-only instance guard, no startup sweep of leaked registration files
or ghost rows. Each fix is local (`flock`, parallel stop, a post-reap group kill, a startup sweep).
A supervisor process is not warranted.

### A6 — The loopback origin check trusts every local port

`isLocalOrigin` (`internal/server/security.go:54–61`) accepts any `http://localhost:<port>`; the
terminal WebSocket skips the library's own check and relies on it (`terminal.go:92`); handlers
decode JSON regardless of Content-Type. A page served from any other local port can therefore open
a PTY socket and send launch or prompt requests without a preflight. TS-05.R2–R3 accept loopback
as the trust boundary, but a page is not a process: in a product whose agents routinely start dev
servers that render untrusted content, this is the realistic route in. Pin the Origin to the
server's own origin plus the configured dev origin, and require JSON on POST. Small change, no
migration.

### A7 — One SQLite connection serves everything

`SetMaxOpenConns(1)` (`internal/state/state.go:37`) is a sound choice for writes: it removes
in-process `SQLITE_BUSY` and lock-upgrade failures, and a syntactic scan of `state`, `index`,
`archive` and `server` found no nested pool use that could self-deadlock. The cost is that reads
queue too. **Measured:** a point read that takes 8 µs took 37 ms on average and 127 ms at worst
while searches ran; one project search holds the connection 260–400 ms at today's history size and
grows with it. The runtime's event observer performs a store read for every streamed text delta
and tool event (`confirmQuotaProgress`, `server/quota_notice.go:25`), so a search stalls every
agent's stream. Separately, `startQuotaContinuation` holds the global quota and config locks
across a full provider resume (`quota_dispatch.go:121–126, 249`), which serializes continuations
that all come due together after a shared reset and blocks every quota settlement and config save
meanwhile. Direction: a second read-only connection for search, archive and transcript reads
(WAL permits it); an in-memory "this agent has a claimed intent" check before the per-event read;
and narrow the quota locks to the claim decision. Not urgent at current scale.

## 3. Improvement opportunities

1. **Share spec construction and stop-and-reap across Launch, Resume and Switch.** TS-01.R9's
   shared seam is real for the claim and teardown, aspirational for composition: three hand-built
   25-field `LaunchSpec` literals with drift (Resume sets no driver; only Launch sets fast), and
   five copies of stop → orphan reap → teardown, one of which alone cancels the quota snapshot.
   One `buildSpec(agent, snapshot, mode)` and one `stopAndReap` remove roughly 250–350 lines and
   close CR-A10-3's rename overwrite as a side effect. Worth doing before a further start path is
   added.
2. **Reconsider the three quota snapshot patches.** The quota subsystem is 4,591 lines with eight
   states and eight identity fences. It fails closed and its owner settlement is transactional;
   that complexity is mostly earned by four owner kinds. The snapshot pre-check (about 600 runtime
   lines and three adapter patches pinned to exact CLI versions) only improves postponement; text
   reset plus bounded checks works without it. Dropping it removes the largest dependence on
   undocumented provider behaviour.
3. **Tell someone when a wake fails.** Waking mail is attempted at most once by design, and unread
   waking mail is deleted after seven days by specification. The sender only ever sees
   `queued_waking`, and a failed wake is logged at Debug. A dashboard fact for stranded waking mail
   costs little and changes no delivery policy. A single shared outbox for all delivery kinds is
   feasible for four of them but not for held prompts, room turns or stage input; the migration is
   not justified.
4. **Decide the database's durability level.** `synchronous=NORMAL` means a power loss can roll
   back committed task, quota and message rows while the transcript, which is fsynced at turn
   boundaries, keeps them. TS-02 makes no statement either way. State the choice.
5. **Check the schema before swapping pointers on rollback**, and add a startup `quick_check` and a
   pre-migration copy. The store has no backup or corruption detection of any kind.

## 4. Verified sound

- **Conversation journal:** one lock covers sequence, append, publication and control order;
  prompts, turn ends and resolutions are fsynced before publication; torn tails are repaired with
  a gap notice. No acknowledged-but-lost prompt on server crash.
- **Turn settlement against process exit:** exactly one `turn_end`; Stop's terminal status is
  fenced against late writers; Cancel cannot end the next turn.
- **Task admission:** capacity, exclusivity and the closed-run fence are one statement; no
  duplicate start across restart. **Schedules:** no double fire, bounded catch-up.
- **Inline mail:** only a completed or truncated turn settles; every other outcome re-offers.
  **Mail activation:** single-statement claim; identity is the stable agent id; tokens are per
  launch.
- **Pipelines over tasks:** result, cursor and release intent commit together and are re-driven
  from durable state.
- **Migrations:** one transaction, ordered, newer-schema refusal. **Config writes:** temp, fsync,
  rename, directory fsync.
- **File save:** pinned parent, no-follow, base-revision compare twice, explicit uncertain outcome.
- **Remote control:** separate listener, node-bound hashed device credential, revocation cancels
  open streams. **Updater:** size and digest before extraction, traversal and symlink refusal.
- **Client sync:** overflow ends the stream and every reconnect rehydrates; transcript replay
  cannot double-apply; prompt request ids are durable receipts that survive restart.

## 5. Recommended order

1. ADV-2 and ADV-3: silent stuck work, each reproduced, each a local store or dispatcher change
   with the probe ready to become its regression. (ADV-1 was listed here before its correction.)
2. A1–A3 as guards, so the next writer cannot reintroduce them. Fold open CR-A10-2 into A1.
3. ADV-4 with A4's turn-start gate; ADV-5, ADV-7, ADV-8 beside it in `internal/runtime`.
4. A6 and ADV-9: small, independent, and they close the two widest doors.
5. ADV-6 and ADV-10, then the improvement list as capacity allows.
