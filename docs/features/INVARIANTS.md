# Chuck invariants — normative technical-spec appendix

Chuck's recurring failure modes have stable citations (`INV §n`). Each numbered class owns a binding constraint under TS-01. Its check shows how to distinguish a violation at the affected boundary. When reporting a defect, establish the reachable consequence and cite the class that explains it. A matching trigger is a reason to inspect the code.

Read cited and affected classes directly. The [shared startup](AGENT-WORKFLOW.md#1-start-with-the-current-state) defines when to scan the index. Use the helper registry at the end to locate an existing implementation. When ownership is unclear, identify who writes the fact, who observes it and what happens after replacement or restart.

**Maintaining a class.** Extend an existing rule when it explains the failure. A distinct failure mechanism can earn a new class through the technical-spec change process. For each new rule, use **Guard** to name the test, lint or hook and its coverage. If a mechanical guard is impractical, cite two independent incidents and give reviewers a distinguishing check. Keep IDs and meanings stable. Preserve incident evidence in the repair commit or [incident archive](../archive/reviews/invariant-incidents-through-2026-11-13.md).

## Trigger index

| INV § | Boundary or change to inspect |
|---|---|
| 1 | Reconnect, resume, switch or config change leaves old derived state visible. |
| 2 | A second path builds the same artifact, projection or transition. |
| 3 | One-shot launch data enters storage, or a form overwrites seeded config. |
| 4 | Registration allocates resources that exits, failures or replacement must release. |
| 5 | Competing callers can both pass a check before claiming an effect. |
| 6 | A new execution backend joins the agent lifecycle through a runtime, driver or external interface. |
| 7 | Iteration, repair or migration can skip a read error or abort later records. |
| 8 | A human-facing preview, status, mutation error or disabled action obscures the outcome. |
| 9 | Process liveness, SQLite, file writes or locks underpin a durability or destructive decision. |
| 10 | A promised entry point, integration or published instruction can be left behind. |
| 11 | Serialization changes a collection, identity, status or error seen by a consumer. |
| 12 | An external CLI's version, flags, environment or output can differ from the developer's. |
| 13 | CSS, layout or markup changes visual behavior that DOM-only tests cannot see. |
| 14 | An HTTP route or local file exposes private state, credentials or actions. |
| 15 | An external effect can precede durable state, or retry can repeat a partial commit. |
| 16 | Incoming streams, queues, retained collections, sweeps or fan-out can grow without bounds. |
| 17 | A test, double or release gate can agree with a broken implementation. |
| 18 | Durable work waits on a party that lacks a legal move or can disappear. |
| 19 | Submitted input crosses owners, gets refused, retried, withdrawn or recovered. |
| 20 | A consumer claims an offered, effective or terminal value from incomplete peer evidence. |
| 21 | An agent acts on assignment text, tool guidance, a command or a refusal. |
| 22 | A refusal, delivery transition, process boundary or error needs a trace safe to retain. |
| 23 | A late action targets an agent ID, PID, token or session name that can be reused. |
| 24 | A retry, default or degraded path can silently change guarantees or persist after recovery. |

---

## 1. Crossing a boundary must reset or republish derived state

**Rule:** whenever execution crosses a lifecycle boundary — reconnect, relaunch, resume, switch,
agent-tab change — every piece of state *derived from the old side* must be explicitly reset or
republished. Nothing "just stays valid."

**Canonical patterns:** reset connection-scoped state in `onopen`, not the constructor; republish
the affected agent after any read/delete mutation; per-agent request tokens for async UI fetches.

**Failure/check:** a reconnect leaves an old permission badge or an agent-tab fetch overwrites the
new tab. Cross the boundary with stale state and a delayed response; assert only the new owner's
state is visible and its current snapshot is published.

## 2. Parallel paths that build "the same thing" must share one helper

**Rule:** two code paths constructing the same logical artifact or projection (a LaunchSpec, a
session params struct, a pidfile guard, a validation, a live/replayed event stream) WILL drift.
Extract the shared helper the moment the second path appears, and route both through it.

**Canonical helpers:** the registry at the end of this file is the list. Check it before writing a
builder, and add a row when a second path appears.

**Guard:** `TestStatusTableWriterInventory` (`internal/state/status_writer_inventory_test.go`) pins
how many statements write `quota_intents`, `tasks` and `pipeline_runs` in each file. A new writer
fails it until the change either routes through an existing writer or raises the count on purpose.

**Failure/check:** launch and resume compose different settings, or replay differs from live chat.
Compare both entry paths for the same input and verify they call the shared builder. The inventory
guard above covers status writers, not every builder in this class.

## 3. Persisted fields never receive one-shot data; forms merge, never replace

**Rule:** a field that is both sent to the runtime AND persisted must never have transient data
concatenated into it — one-shot additions need a separate field the persistence path is
structurally blind to. Symmetrically: a UI form pre-populated from seeded server config must
merge-preserve the seeded collection on submit, never write back its partial on-screen view.

**Failure/check:** a resumed prompt repeats a one-shot instruction, or saving one form erases hidden
seeded entries. Submit, reload the durable record and resume; verify transient text is absent and
untouched seeded entries remain.

## 4. Create/teardown symmetry: one teardown function, generation-scoped, on every exit path

**Rule:** every artifact created at registration (hook token, MCP session/file, hook-settings file,
DB row) is torn down by exactly one shared function, invoked from **every** exit path — stop,
switch, failed launch, failed resume, and unsolicited crash. Teardown is scoped to a launch
generation, not just an agent id. Teardown of the OLD strictly precedes registration of the NEW
under the same key, and every failure branch after that point routes through the same rollback.

**Failure/check:** failed resume leaves a usable hook token, or a late crash removes a replacement's
registration. Inject failures after registration and process start; check artifacts on every exit
and preserve the replacement generation. §5 governs the claim; §23 governs identity at teardown.

## 5. Check-then-act needs an atomic claim or a generation token

**Rule:** claim pending work atomically or recheck the captured generation (§23) at fire time.
Only the winning caller emits the effect. Take a snapshot and subscribe under the same lock.

**Failure/check:** concurrent permission resolution emits two results, or snapshot/subscription loses
an intervening update. Release competing callers together and assert one claim and one side effect;
insert an update at the snapshot boundary and verify it is observed.

## 6. A new interface/runtime must join every existing contract (checklist)

**Rule:** a new runtime, driver, or external execution interface participating in agent lifecycle
must explicitly walk this checklist — silence on an applicable line is a bug. Routine internal Go
interfaces do not trigger it; mark genuinely unsupported contracts with their refusal behavior:

- [ ] **Persistence:** gets a `sessions` row → visible in archive, resumable, survives Stop.
- [ ] **LaunchSpec:** honors the full composed spec (model, system prompt, add_dirs, MCP
      registration) — via the shared resolvers of §2.
- [ ] **Fan-out/drain:** with zero or many viewers, one reader drains each PTY into bounded
      scrollback and drops slow subscribers (§16, `internal/runtime/terminal/ptyhub.go`).
      Duplicating a shared fd splits the stream. A departing viewer must leave the live PTY open.
      `TerminalTab.tsx` sends binary keystrokes and reserves text frames for `{cols,rows}` resize.
- [ ] **Messaging:** either it can drain its mailbox (`check_messages`) or `ResolveRecipient`
      excludes it from nudges.
- [ ] **Turn boundaries:** its turn signal is identified and wired into every per-turn reset
      (`resetTurnBudgetTx` via `terminalTurnID`) — not lumped into a generic status hook.
- [ ] **Reconcile:** the sweep knows its shape (no chat-shaped preview stomping, §1).
- [ ] **Hooks/status:** scope registration flags to the affected runtime so other agents still report status.
- [ ] **Capabilities:** advertise only options that a supported API or UI can select.
- [ ] **Teardown:** joins §4's single teardown on every exit path.

**Failure/check:** a terminal agent is absent from archive or stalls when no viewer drains its PTY.
Exercise the new execution path through launch, detached output, messaging, turn end, stop, resume
and crash; record evidence for each applicable checklist line rather than inferring chat parity.

## 7. Read paths must not swallow errors or amplify damage

**Rule:** iteration and repair code treats each record/entity as independently failable: check the
real error signal, skip the bad record, continue, and report — never abort the whole stream or
wipe-then-fail.

**Failure/check:** one corrupt row hides later valid records, or a repair deletes good data before
its replacement fails. Mix valid and corrupt records, inject iteration/write failure, and verify
continued progress, retained good data and a reported error. Check `rows.Err()` after SQL iteration.

**Guard:** the [Claude post-edit hook](../../.claude/hooks/post-edit.sh) flags a Go file with
`rows.Next()` and no `rows.Err()`. Review individual loops when a file contains several.

## 8. User-facing surfaces get parsed, bounded, in-vocabulary data — and errors always surface

**Rule:** anything rendered to the human (card previews, `status.detail`, toasts) is parsed,
human-meaningful, length-clamped at the write boundary, and drawn from the declared vocabulary.
Every mutating UI action surfaces failure; every notification funnels through the one payload
builder. A disabled action names the condition that blocks it, and alternatives offered together
state their different consequences beside the controls. A mutation's warning or success-adjacent
feedback stays owned by a surface that remains mounted, or is handed off before that surface closes.

**Failure/check:** a closing dialog loses a failed save's toast, or a card renders raw provider JSON.
Fail the mutation and close its initiating surface; verify persistent feedback and recovery. Feed
oversized/unknown output and inspect the bounded, declared vocabulary at the render boundary.

## 9. Liveness & durability primitives are weaker than they look

**Rule:** corroborate OS/SQLite guarantees before destructive maintenance or durable writes:

- **Bare `kill(pid, 0)`** proves only that *some* process has the PID (reuse!). Used in
  `internal/cli/pidfile.go` — corroborate (start-time/comm/nonce) where it gates destructive
  action, and never remove a pidfile without verifying it names your own PID. Process identity and
  late-action fencing belong to §23; use `runtime.ClassifyProcess` for recorded process proof.
- **Atomic-write-via-rename** must fsync the file AND the parent directory
  (`internal/config/atomic.go`, `pidfile.go`).
- **`CREATE VIRTUAL TABLE IF NOT EXISTS`** cannot upgrade an existing fallback object. Detect and
  replace it (`ensureSessionsFTS`, `internal/state/migrate.go`); §24 governs fallback recovery.
- **Append-only writers** must truncate a crash-torn trailing record on `Open()` before appending,
  or the torn tail fuses with the next record (`internal/transcript/writer.go`).
- **In-memory accumulators feeding replace-style (DELETE+INSERT) writes** must lazily reseed from
  the durable table on first use per process — an empty-seeded buffer once wiped all FTS content
  after a restart. Alternatively retain independent durable documents rather than replacing the
  whole history from a process-local buffer (`insertTranscriptDocument`, `internal/index/indexer.go`).
- **Destructive CLI ops sharing the live server's DB hard-refuse on liveness** (`reindex` once only
  warned).

**Failure/check:** restart with a torn transcript, durable indexed history and a live server; verify
tail repair before append, history preservation and refusal of destructive maintenance. Simulate
write/rename failure and inspect file and parent-directory sync ordering; syscall success alone is
not proof of crash durability. `TestOpenTruncatesTornTrailingLine` covers the transcript case.

## 10. Ship the wiring; kill the drift

**Rule:** a feature isn't done until every surface that promises it can reach it, and every doc
that describes it matches. Gating stubs are un-gated in the same effort that ships the gate.

**Failure/check:** desktop exposes a supported action while phone still refuses through a stub.
List the promised entry surfaces from the FS acceptance criteria, exercise each, and compare its
capabilities and guidance with the shipped behavior.

## 11. Cross-boundary serialization and protocol contracts preserve meaning

**Rule:** Go nil slices/maps marshal to JSON `null`, not `[]`/`{}`. Any collection field the UI
iterates must be non-nil at the marshal boundary (initialize with `make`/literal, or
`append([]T{}, …)` — **never** `append([]T(nil), …)`, which stays nil for empty input), and the UI
API layer defends with `?? []` regardless. Protocol success must also preserve the identity, status,
and error semantics the caller relies on: sentinel errors survive transport intact; identity
ownership follows the operation's contract even when a response omits an echo; and an absent status
never acquires a terminal meaning by default.

**Canonical patterns:** `append([]T{}, src...)` at marshal boundaries; `?? []` / `?? {}` where the UI
first touches a server collection; structurally validate required nested collections decoded from
hand-editable files; validate required protocol identity/status fields before mutating runtime state.

**Failure/check:** an empty API result crashes a UI iterator, or an omitted peer status settles a
turn as successful. Serialize empty collections through the real wire boundary and inject missing
identity/status fields; assert usable empty collections and no invented terminal state.

## 12. External-CLI invocations must tolerate version and environment variance

**Rule:** any `exec.Command` of a user-installed tool (agent CLIs, probers) runs against whatever
version the user has, not the one the author tested. Optional flags need a detect-and-retry
fallback; output parsing is defensive (substring vocabulary, not exact format); a tool that can't
be interrogated reports "unknown"/"skipped", **never** "failed" — a wrongly failed gate blocks the
user harder than no gate at all. Exit zero is not success: validate the output's shape before
relying on it. What Chuck may then claim about the tool's effective state is §20.

**Canonical pattern:** `claudeProber.Check` (`internal/backend/credcheck/claude.go`) — retry without
`--no-color` only when the diagnostic identifies that unsupported flag. Inspect affected tool calls
under [targeted static leads](USABILITY-REVIEW.md#4-targeted-static-leads).

**Failure/check:** an older CLI rejects a cosmetic flag and Chuck reports invalid credentials.
Use a fake executable that rejects that flag, then returns valid bare output; also return malformed
exit-zero output and an unsupported status command. Verify retry, shape validation and unknown or
skipped results without converting genuine authentication failure into success.

---

## 13. The test suite cannot see CSS or layout

**Rule:** TSX class names, including dynamically composed state classes, must resolve to the
intended stylesheet selectors. The cascade enters through `ui/src/styles/index.css`. jsdom
can establish DOM behavior; geometry and visual states require a rendered browser.

**Failure/check:** a dialog has valid DOM but undefined classes or covers its own controls at phone
width. Compare affected class names with stylesheet selectors, then render supported widths and
appearances in a real browser and assert computed styles, geometry and reachable controls. DOM
presence and stylesheet rule count alone cannot establish layout correctness. Follow
[execution and evidence](USABILITY-REVIEW.md#5-execution-and-evidence); no universal CSS guard is implied.

---

## 14. A loopback bind is not a browser or filesystem security boundary

**Rule:** browsers can reach loopback services, and other local users can read permissively
created files. Keep both boundaries enforced:

- On the local listener, `localOnly` wraps the entire mux, including static UI, API, MCP and
  WebSockets. Reject non-local Host or Origin with 403 (TS-05.R23). CORS cannot authorize requests.
  The tailnet listener has its own route chain with the remote guard, device authentication
  and the TS-13.R4/R5 allowlist.
- Under `~/.chuck`, create directories and executable hook scripts with mode `0700`, and other
  files with `0600`. `EnsureLayout` and `state.Open` must also tighten existing paths; `MkdirAll`
  leaves existing modes intact and SQLite creates files relative to umask.

**Canonical guard:** `localOnly`, `isLocalHost` and `isLocalOrigin` in `internal/server/security.go`.
Use `newLocalRequest` for ordinary local-route tests; the default `httptest.NewRequest` host is rejected.

**Failure/check:** a cross-origin WebSocket bypasses an API-only guard or old state files remain
world-readable. `TestDNSRebindingHostRejected` and `TestCrossOriginRequestRejected` cover local
request rejection; additionally exercise each listener's full route chain and reopen an existing
permissive layout to verify modes, including remote device-auth/allowlist refusal where applicable.

---

## 15. Commit local truth before releasing external side effects

**Rule:** before an operation makes work externally observable or lets an external peer proceed,
persist and publish the local state needed to explain that effect. A retryable mutation spanning
multiple stores or processes must be atomic, roll back every partial effect, or carry a stable
idempotency key that makes replay harmless. Never return a retryable failure after an irreversible
effect if the natural retry can duplicate that effect.

**Canonical patterns:** durable event/outbox before notification; one transaction when stores share
one database; otherwise compensating rollback or a caller-supplied idempotency key with a uniqueness
constraint. Tests inject the failure after the first would-be side effect and retry the operation.

**Failure/check:** a notification releases a peer before the task write commits, or retry creates
two externally visible jobs. Inject failure between stores and before publication, then retry with
the same operation id; assert one durable result and no uncommitted effect visible to the peer.

---

## 16. Bound work and memory where they enter the system

**Rule:** any stream, queue, retained collection, background sweep, or fan-out receiving input of
unknown size must be bounded at admission or capture. Trimming a value only after it accumulated
does not bound memory. State the byte/item limit, in-flight concurrency limit, and lifetime that
apply; admit only that much work, retain only what consumers need, and release it when its last
owner or lifecycle ends (§4).

**Canonical patterns:** bounded tail writers for external output; oldest-first limited queries plus
a process-wide in-flight cap for sweeps; ref-counted registration with per-owner teardown; temporary
reconciliation tails instead of permanent duplicate projections.

**Failure/check:** a slow subscriber or endless CLI output grows memory despite a final truncation.
Drive more than the stated limit while a consumer stalls; inspect retained bytes/items and active
workers at admission, then detach the last owner and verify release.

---

## 17. Tests must prove their contract independently

**Rule:** a regression, acceptance, agreement, or release test must have an oracle independent of
the implementation it is checking and must be capable of failing when the claimed contract breaks.
Derive the expected surface from a separate authority such as registered producers, the wire
boundary, artifact metadata, or an explicit requirement enumeration — never from a copy of the
table or helper under test. A regression test is confirmed against the pre-fix behavior when that
check is practical. Test doubles mirror the real producer's failure modes and serialized shapes,
not the idealized contract.

**Canonical patterns:** use the real client serializer and a pinned peer's captured response at
the wire boundary. Compare policy tables with registered producers, exercise real OS/storage
failure modes, and inspect the built artifact for packaging claims.

**Failure/check:** a capability test copies the implementation's incomplete table and stays green
when a backend is omitted. Remove a real producer from the policy under test or replay its actual
wire shape; verify the independent oracle detects the omission or malformed handling.

---

## 18. A waiting state names who can end it, and something notices when they cannot

**Rule:** every durable non-terminal state that waits on a party — an agent's report, a person's
Continue, a child task, a peer's reply — records which party and generation can end it, gives that
party a legal move from every state it can honestly be in (including "nothing to add" and "cannot
proceed"), and has a detector that turns "that party can no longer act" into an attention state. A
wait that looks identical to active work is a defect. So is a status derived from an event stream
with no owner responsible for settling it. A handler that loses a compare-and-swap on the waited
record re-reads under the lock and decides again; it never returns quietly (§5).

**Canonical patterns:** the wait record carries the waiter's identity and generation; the waiter's
turn end, exit, and a periodic sweep all re-evaluate the wait under its lock; "cannot advance"
is an enumerated attention reason, not an absence of one; every protocol state offers a decline,
pass or blocked move. In-process timers do not survive a restart (§9), so the detector is driven
from durable state. A soft request to an external peer (cancel) has a time-bounded escalation tier
(grace, then SIGINT) distinct from Stop's hard kill.

**Failure/check:** a task waits forever after its assigned agent exits, or restart loses the only
timer. Enter the wait, end/replace its owner and restart; verify a legal completion/decline or
attention transition from durable evidence, including a lost-CAS re-evaluation.

---

## 19. Accepted input has one durable owner; a refusal changes nothing

**Rule:** anything a person or agent submits — a prompt, steer, answer, report, proposal or
annotation — is either accepted or refused. Accepted means recorded durably with its provenance
before the response, with exactly one record that says where it is now (held, delivered,
withdrawn, unknown). Refused means no side effect at all: no wake, no state advance, no partial
write. Moving input between owners (queue to turn, receipt to composer draft, chat to pipeline
assignment, transcript to handoff primer) is one acknowledged handoff, and the old owner lets go
only on the new owner's confirmation of that exact request. No route-specific branch bypasses the
receipt, and accepted input is never reconstructed from a display artifact such as
adapter-rendered tool output, an optimistic bubble, or a browser-local pointer. §15 orders the
commit before the effect; this class says who owns the input while that happens.

**Canonical patterns:** a caller-supplied command id on every submission; the receipt state machine
is the sole authority and every surface derives held/recovered/delivered from it instead of keeping
its own copy; a handoff acknowledges the request id it took; validation and eligibility checks run
before the first write so a refusal has nothing to undo.

**Failure/check:** a timed-out prompt retry delivers twice, or a refused annotation still wakes an
agent. Retry the same request across disconnect/restart and interrupt an owner handoff; assert one
durable owner and acknowledged transfer. Snapshot state before a refusal and verify no mutation.

---

## 20. Requested, offered and effective are different facts

**Rule:** when a peer or another component decides something — the model, effort, session identity,
a capability, a saved setting, how a turn ended — Chuck keeps three facts apart. *Offered* is
derived from the component that will execute the choice, at the version and schema it actually
runs; it is never a parallel list. *Requested* is what Chuck sent. *Effective* is what the executor
reported back, and only that is persisted as current, displayed, and returned as the operation's
result. Missing evidence is `unknown`, a distinct value that never collapses into false, success,
or the requested value. Each operation has one validated terminal outcome, and every consumer —
status, settlement, receipts, handoff kind, notification — reads that outcome instead of
re-deriving success from the raw envelope. §11 and §12 govern reading a peer correctly; this class
governs what Chuck then claims.

**Canonical patterns:** read back the effective value after every configuration call and fail
closed on a mismatch for required settings; decode capabilities as present-true, present-false or
unknown, and let unknown reach the server-side recheck; derive selectable values and admitted
request fields from the executing runtime or the shared request type (`backend.Support`); one
typed turn-outcome value consumed by status, settlement and receipt code alike.

**Failure/check:** the UI reports the requested model after the peer selected another, or EOF is
treated as successful completion. Return a mismatch, omitted evidence and competing terminal
signals; verify unknown remains distinct and all consumers use the same outcome.
`TestResolveTurnOutcome` (`internal/runtime/turn_outcome_test.go`) covers outcome precedence.

---

## 21. Agent-facing instructions and refusals are protocol

**Rule:** an agent acts literally on the text Chuck gives it and cannot ask what was meant.
Assignment text, tool descriptions, standing guidance and refusal messages are therefore part of
the machine contract. They use the same vocabulary as the code (retry class, state names), state
each boundary in terms of the accepted result rather than the call made, and reach every route by
which an agent can receive the work. A refusal code has one meaning; its message names the true
condition and the next valid move. Forwarding text that a provider interprets natively (a leading
slash command) is a separate route with its own policy and is chosen deliberately. This class is
about truth and consistency, not volume: procedure belongs in the skill the text points to.

**Canonical patterns:** compose agent-facing text for new, existing and continued assignees in one
place; test that each refusal's message matches its condition and its retry class; when a protocol
gains a state, add the move that lets an agent express it.

**Failure/check:** a refusal tells an agent to retry an operation that requires human approval, or
a continued assignee misses an instruction sent only on fresh launch. Compare fresh/existing/
continued assignment text with the actual state machine, and exercise each refusal's stated next
move; verify it is legal for that condition and preserves the declared retry class.

---

## 22. Failure points leave a bounded, correlated trace

**Rule:** every refusal, every delivery or ownership transition of durable work, every process
start and stop, and every error that crosses a process or wire boundary writes one bounded
structured record to Chuck's own log: who (agent and generation, run/visit/request id), which
operation, and the typed reason. A peer's typed reason survives to the surface that shows the
failure, bounded and allowlisted; message bodies, prompt text, attachment contents and raw provider
data are never logged. A semantic failure carried in a successful envelope is recorded like a
failed call. Browser-side failures reach the server log. Reports arrive from machines where live
debugging is impossible, so a failure point that logs nothing is a defect in its own right.

**Canonical patterns:** log at the seam that makes the decision, at Warn for a refusal; carry
identities and typed codes, never payloads; preserve `code` plus an allowlisted reason through
error wrapping; give terminal-result failures the same diagnostic as failed RPCs.

**Failure/check:** a semantic RPC failure returns HTTP success and leaves no correlated diagnostic,
or a raw provider error leaks prompt text. Capture logs for refusals, transport and envelope-level
failures; assert bounded identity/operation/reason fields and absence of submitted payloads.

---

## 23. A reusable name is not an identity

**Rule:** agent ids are reused across launches, PIDs across processes, session ids across a load
that became a new session, counters across restarts. Anything with a lifecycle — a registration,
cooldown, cache entry, token, transport session, pending action, signal target — is keyed by its
name **plus** a generation or birth proof. Any action that can arrive late (a crash teardown, a
timer, an async reply, a reaper) rechecks that pair before it acts. A refusal says which half
failed to match: wrong party, wrong generation, or already done.

**Canonical patterns:** launch generation on every registration and its teardown
(`teardownAgentRegistration`); recorded leader PID plus kernel birth (`runtime.ClassifyProcess`);
capture-and-recheck of `turnSeq`; transport sessions bound to the launch that created them; re-read
identity while holding the claim.

**Failure/check:** a delayed crash callback tears down a new launch, or a reaper signals a reused
PID. Replace the owner before releasing a delayed action and supply mismatched/unknown process
birth proof; assert replacement state survives and no unverified process is signaled. §4 specifies
cleanup coverage, §5 the atomic claim, and §9 the underlying OS/durability limits.

---

## 24. A fallback is a state change: fail closed, or make it visible, bounded and reversible

**Rule:** when the intended path is unavailable, the code either refuses or enters a second mode.
A second mode is real state: it is recorded as the mode now in effect, reported to whoever relies
on the result, limited in duration, and re-tested so the system returns to the intended path when
it can. A fallback never reports the intended path's success, never becomes permanent by default,
and never supplies a permissive or terminal value for missing evidence. Where the caller needs the
intended guarantee — permissions, history, identity — fail closed.

**Canonical patterns:** `history_handoff` identifies the mode in the result and status.
Cache only validated responses and treat absent evidence as `unknown` (§20).

**Failure/check:** a temporary FTS fallback persists after FTS becomes available, or history recovery
claims native resume. Force the primary path unavailable, inspect the recorded/reported mode, then
restore it and trigger recovery; assert the fallback exits and required guarantees refuse while
unavailable. §9 covers the SQLite object upgrade mechanism.

---

## Canonical helpers registry (reuse, don't re-derive)

| Helper | Where | Use for |
|---|---|---|
| `useInputDelivery` / `RecoveryOffer` / `withdrawHeldMessage` | `ui/src/components/chat`, `ui/src/lib/heldMessage.ts` | desktop and phone prompt submission, retry, recovery and withdrawal (§2, §19) |
| `teardownAgentRegistration` / `handleAgentExit` | `internal/server/launch.go`, `internal/server/server.go` | shared registration cleanup; exit callback generation checks live in `handleAgentExit`, and callers must hold the lifecycle claim or establish ownership before cleanup (§4, §5, §23) |
| `startRuntimeClaimed` | `internal/server/lifecycle_start.go` | compose, start, rollback and persist applied identity for launch/resume origins while the caller holds its lifecycle claim (§2, §4) |
| `composeLaunch`, `composeResumeSpec`, `composeSwitchSpec`, `resolveSkip`, `expandAddDirs`, `composeEnv` | `internal/server/{launch,resume,switch}.go` | any path that builds/rebuilds a LaunchSpec (§2) |
| `resolveStartCheckout` / `ensureRunWorkspace` | `internal/server/pipeline_workspace.go` | every launch/resume/switch/clone cwd readiness, project-owned or run-owned (§2) |
| `placeRunTaskTx` / `checkRunTarget` | `internal/state/pipeline_workspaces.go` | run group placement and existing-target checkout eligibility for run work (§2, §5) |
| `sessionNewParams` / `sessionLoadParams` | `internal/runtime/chat.go` | kept in lockstep: any session member added to one is added to the other (§2) |
| `insertTaskRowTx` / `taskRunOpenClause` | `internal/state/tasks.go` | every task row a transaction writes; every mutation that makes an existing task executable again (§2) |
| `LatestPipelineStageTask` / `Manager.settleCleanup` | `internal/state/pipeline_tasks.go`, `internal/pipeline/reconcile.go` | the run cursor; stage and run cleanup convergence (§2) |
| `takePending` | `internal/runtime/permission.go` | atomic claim for racy resolutions (§5) |
| `SubscribeWithSnapshot` | `internal/bus/bus.go` | any snapshot+subscribe consumer (§5) |
| `runtime.ClassifyProcess` / `StopRecordedProcess` / `ReconcileStale` | `internal/runtime/procstart.go`, `reconcile.go` | every liveness decision, signal or prune for a process the registry doesn't hold; proof is recorded leader PID + kernel birth (§4, §9) |
| PTY hub pattern | `internal/runtime/terminal/ptyhub.go` | any shared-fd broadcast need (§6) |
| `insertTranscriptDocument` / `replaceMetadataDocument` | `internal/index/indexer.go` | durable transcript documents and separately replaced metadata; do not replace whole history from a process-local buffer (§9) |
| `config.ValidSlug` | `internal/config/validate.go` | every path-param on every verb (§2) |
| `foldTranscript` / `appendRenderedEvent` | `ui/src/store/transcriptStore.ts` | identical live and replay event projection (§2) |
| `transcript.ProjectEvent` / `runtime.AllEventTypes` | `internal/transcript`, `internal/runtime` | every Go transcript text consumer and exhaustive normalized-event coverage (§2) |
| `resolveTurnOutcome` / `runtime.DecodeTurnEnd` | `internal/runtime/turn_outcome.go` | resolve terminal turn evidence once and interpret the same outcome from live or stored events (§20) |
| `groupTranscriptRows` / `ToolRun` | `ui/src/components/chat/toolRun.tsx` | folding tool calls and results into one collapsed row on desktop and phone (§2) |
| `clipAnnotationExcerpt` | `ui/src/lib/annotations.ts` (server copy authoritative: `internal/server/sessions.go`) | every UI surface that captures an annotation excerpt (§2) |
| `copyText` | `ui/src/lib/copyText.ts` | UI copy actions that report clipboard failures as a toast (§2, §8) |
| `offerWebLink` / `claimWebLink` / `webLinkActions` / `fileActions` | `ui/src/lib/linkActions.ts` | rendered web-link menus; every enclosing menu (annotation or file viewer) claims the link so one right-click opens one menu; chat and project file menus share `fileActions` (§2) |
| `capVisible` / `OverflowMenu` | `ui/src/lib/navCap.ts`, `ui/src/components/shell/OverflowMenu.tsx` | every compact nav's five-cap with current-visible and its `+n` disclosure (§2) |
| `displayLabel` / `displayLabels` | `ui/src/lib/labels.ts` | every model/backend/project/role/template label; the id appears only for duplicate names (§2) |
| `AutoGrowTextarea` | `ui/src/components/ui/AutoGrowTextarea.tsx` | every multi-line text field in `ui/src`; a source-scan test rejects raw `<textarea>` (§2) |
| `FileTabsReader` / `fileTabs` / `useFileTabRoute` / `readFileTarget` / `writeFileTarget` | `ui/src/features/fileTabs/` | every file reader surface, saved file-tab mutation and `?file=` address read/write (§2, §16) |
| `localOnly` | `internal/server/security.go` | wraps the whole mux; every new route inherits it (§14) |
| `notificationPayload` | `internal/bus/` | all notification payloads (§8) |
| `fakeacp` test double | `internal/runtime/testdata/fakeacp` | env-driven protocol-level repros (`FAKEACP_LOAD_DUMP`, `FAKEACP_PROTO_VERSION`, `ignore_cancel`); it drops out-of-schema session-request members like the pinned peer (§17) |
| `backend.Types` | `internal/backend/adapter.go` | enumerating every backend type instead of restating the union (§2, §17) |
| `backend.Support` / `launchSupportFor` | `internal/backend/adapter.go`, `ui/src/schemas/backends.ts` | every interface/effort/fast launch-support decision; never a browser or server backend-type allowlist (§2, §10) |
