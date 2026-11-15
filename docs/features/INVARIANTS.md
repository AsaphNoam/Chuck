# Chuck invariants — normative technical-spec appendix

Every class below cost at least one review cycle; most recurred in two or more subsystems before
being named. The evidence base is multiple full top-to-bottom reviews, the repo's `review fix:`
commit history, and the field bug investigations of §12 of the workflow. Treat these as **load-bearing rules, not suggestions**: a diff that violates one is
wrong until proven otherwise, and a review finding that matches one is almost certainly real.
This file is governed by TS-01 and indexed with the technical specs as `INV`; it is not a third
product-spec set. Add or change a binding rule here through the same spec-delta lifecycle as a TS.

**The principle behind most classes: every fact has one owner.** One builder (§2), one teardown
(§4), one claim (§5), one receipt (§19), one outcome (§20). Any other copy of a fact — an in-memory
duplicate, a browser cache, a parallel list — states which authority it derives from, how long it
lives, and how it is rebuilt. When a case fits no class below, apply that test.

The hot-spot areas: launch/resume/switch composition, `internal/runtime` concurrency,
`internal/state`/`internal/index` persistence, terminal/PTY, UI forms over seeded config.

How the loop uses this file:
- **/work** — before building in a hot-spot area, read the matching class; new interfaces
  must complete the §6 contract checklist.
- **/review** — check the diff against the classes **by the trigger index below**; tag each finding
  with its class number.
- **/fix** — note the class in the commit message; if a fix reveals a new canonical pattern, add it
  to its class. Keep this file curated — merge near-duplicates, don't let it become a graveyard.

**A new rule arrives with a guard.** Written classes were violated after they were written; the
one rule with a hook (§7's `rows.Err`) stopped recurring. So a new class, or a new rule inside one,
names the test, lint or hook that fails when it is broken, as a `**Guard:**` line. A rule nobody can
check mechanically is added only after two independent incidents, and never in place of a guard
that could have been written. Incident history is not kept here: the evidence behind each class
through 2026-11-13 is in the
[incident archive](../archive/reviews/invariant-incidents-through-2026-11-13.md), and later
evidence belongs in the commit that fixes it.

Read the trigger index first. Its rule column is binding on its own; a class body adds the
canonical patterns and guards. Read in full only the classes your diff actually touches — reading
every class for a diff that touches one is waste, not diligence. Class numbers are stable citations
(`INV §n`): narrow or extend a class, never renumber. Read one class with:

```bash
awk '/^## 7\./{f=1} f&&/^## 8\./{exit} f' docs/features/INVARIANTS.md
```

## Trigger index

| # | Rule | Read the body when the diff touches |
|---|---|---|
| 1 | Crossing a lifecycle boundary explicitly resets or republishes every piece of state derived from the old side. | reconnect, relaunch, resume, switch, or any lifecycle transition that leaves derived state behind |
| 2 | Two paths that build the same artifact or projection share one helper from the moment the second path appears. | a second code path constructing an artifact or projection that already has a builder (check the canonical-helpers registry below) |
| 3 | A persisted field never receives one-shot data, and a form over seeded config merges instead of replacing. | a field that is both sent to the runtime and persisted; any form that writes seeded config |
| 4 | Everything created at registration is torn down by one shared, generation-scoped function on every exit path. | anything created at registration — hook tokens, MCP sessions/files, hook-settings files, temp artifacts |
| 5 | Check-then-act across goroutines claims atomically or rechecks a generation token; only the winner emits side effects. | "if pending then resolve" / "if active then fire" across goroutines; `internal/runtime` concurrency |
| 6 | A new interface, runtime or driver walks the full contract checklist; silence on any line is a bug. | a new interface, runtime, driver, or adapter — walk the checklist line by line |
| 7 | Iteration and repair code treats each record as independently failable: check the real error, skip, continue, report. | iteration, repair, migration, or recovery code over records |
| 8 | What a person sees is parsed, bounded and in vocabulary; every mutation surfaces its failure and names what blocks a disabled action. | card previews, `status.detail`, toasts, rendered output, disabled controls, or a UI mutation's error/recovery path |
| 9 | OS and SQLite liveness and durability primitives are weaker than they look; corroborate before any destructive action. | SQLite, file locks, signals, process liveness, timeouts, fsync |
| 10 | A feature is done when every surface that promises it can reach it and every document describing it matches. | a feature reachable from some surfaces but not all; a gating stub; docs left behind |
| 11 | Data crossing a boundary keeps its meaning: collections are never `null`, and identity, status and error semantics survive transport. | Go structs crossing to the UI; nil slices/maps; ACP or streaming payload identity/status/error semantics |
| 12 | A user-installed tool runs at whatever version the user has: optional flags retry bare, parsing is defensive, and uninterrogable means unknown, never failed. | any `exec.Command` or adapter call to a user-installed tool — agent CLIs, probers, git |
| 13 | The test suite cannot see CSS or layout: every className has a selector, and geometry contracts get a real-browser assertion. | `ui/src` markup or `ui/src/styles/**`; anything about where an element is, how large it is, or what it covers |
| 14 | A loopback bind is not a security boundary: every route sits behind the local or remote guard, and everything under `~/.chuck` is owner-only. | HTTP handlers, CORS, file paths from requests, anything reachable from a browser |
| 15 | Persist and publish local state before an effect becomes externally observable; a retryable multi-store mutation is atomic, rolled back, or idempotent. | an operation that makes work externally observable or lets a peer proceed |
| 16 | Streams, queues, retained collections, sweeps and fan-out are bounded at admission, with a stated limit and lifetime. | external output, streams, queues, sweeps, retained maps/lists, goroutine fan-out, or per-client state |
| 17 | A test's oracle is independent of the code it checks, and its doubles mirror the real producer's shapes and failures. | regression or acceptance tests, contract enumerations, mocks/fixtures, build tags, packaging or release gates |
| 18 | A durable wait records who can end it, gives that party a legal move from every honest state, and raises attention when they cannot act. | a durable state that waits on an agent, person, child task or peer; a status derived from an event stream; turn-end/exit handlers; protocol dispositions |
| 19 | Submitted input is accepted durably under one owning record or refused with no side effect; handoffs between owners are acknowledged. | prompts, steers, answers, reports, proposals or annotations being queued, held, withdrawn, recovered, rerouted, or carried into an assignment or handoff |
| 20 | Offered comes from the executor, effective is what it reported back, missing evidence is `unknown`, and one validated outcome feeds every consumer. | a model, effort, capability, session identity, saved setting or turn outcome decided by a peer or another component and then persisted, displayed, offered for selection, or used to settle work |
| 21 | Text an agent acts on is machine contract: it shares the code's vocabulary, and each refusal has one meaning and names the next valid move. | assignment text, tool descriptions, standing guidance, refusal codes/messages, or forwarding a provider-native command under an assignment |
| 22 | Every refusal, delivery transition, process start/stop and boundary error logs who, which operation and the typed reason — never payloads. | a refusal, a delivery or ownership transition, process start/stop, an error crossing a process or wire boundary, client error handling |
| 23 | Anything with a lifecycle is keyed by its name plus a generation or birth proof, and a late action rechecks that pair. | a map, cooldown, cache, token, session or signal keyed by agent id, PID, session id or another reusable name; work that can arrive after its owner was replaced |
| 24 | A fallback fails closed or becomes a recorded, visible, time-limited mode that is re-tested for recovery. | a default for an absent field, a retry on a different path, a degraded mode, a catch-all route, a cache of a failed response, "warn and continue" |

Open a class body only when its trigger matches.

---

## 1. Crossing a boundary must reset or republish derived state

**Rule:** whenever execution crosses a lifecycle boundary — reconnect, relaunch, resume, switch,
agent-tab change — every piece of state *derived from the old side* must be explicitly reset or
republished. Nothing "just stays valid."

**Canonical patterns:** reset connection-scoped state in `onopen`, not the constructor; republish
the affected agent after any read/delete mutation; per-agent request tokens for async UI fetches.

## 2. Parallel paths that build "the same thing" must share one helper

**Rule:** two code paths constructing the same logical artifact or projection (a LaunchSpec, a
session params struct, a pidfile guard, a validation, a live/replayed event stream) WILL drift.
Extract the shared helper the moment the second path appears, and route both through it.

**Canonical helpers:** the registry at the end of this file is the list. Check it before writing a
builder, and add a row when a second path appears.

**Guard:** `TestStatusTableWriterInventory` (`internal/state/status_writer_inventory_test.go`) pins
how many statements write `quota_intents`, `tasks` and `pipeline_runs` in each file. A new writer
fails it until the change either routes through an existing writer or raises the count on purpose.

## 3. Persisted fields never receive one-shot data; forms merge, never replace

**Rule:** a field that is both sent to the runtime AND persisted must never have transient data
concatenated into it — one-shot additions need a separate field the persistence path is
structurally blind to. Symmetrically: a UI form pre-populated from seeded server config must
merge-preserve the seeded collection on submit, never write back its partial on-screen view.

## 4. Create/teardown symmetry: one teardown function, generation-scoped, on every exit path

**Rule:** every artifact created at registration (hook token, MCP session/file, hook-settings file,
DB row) is torn down by exactly one shared function, invoked from **every** exit path — stop,
switch, failed launch, failed resume, and unsolicited crash. Teardown is scoped to a launch
generation, not just an agent id. Teardown of the OLD strictly precedes registration of the NEW
under the same key, and every failure branch after that point routes through the same rollback.

## 5. Check-then-act needs an atomic claim or a generation token

**Rule:** any "if pending then resolve" or "if active then fire" across goroutines must either
claim atomically under one critical section (take-and-delete) or capture a generation counter and
recheck it at fire time. Only the winner of the race may emit side-effect events. Snapshot+subscribe
is one lock acquisition, never two.

## 6. A new interface/runtime must join every existing contract (checklist)

**Rule:** the terminal runtime shipped as a second-class citizen and produced the single largest
concentration of BLOCKING findings (6 findings, one review). Any new interface, runtime, or driver
must explicitly walk this checklist — silence on any line is a bug, not a default:

- [ ] **Persistence:** gets a `sessions` row → visible in archive, resumable, survives Stop.
- [ ] **LaunchSpec:** honors the full composed spec (model, system prompt, add_dirs, MCP
      registration) — via the shared resolvers of §2.
- [ ] **Fan-out/drain:** output readable by N viewers and drained when *zero* viewers are attached
      (a full kernel tty queue stalled an unobserved CLI indefinitely). Pattern:
      `internal/runtime/terminal/ptyhub.go` — one always-on reader per PTY, bounded scrollback
      ring, non-blocking fan-out that drops slow subscribers (§16). Never `dup()` a shared fd per
      viewer (splits the stream), and never let a transient view's teardown close a long-lived fd
      it doesn't solely own (a WS unmount once SIGHUP'd the live CLI). Keystrokes use binary
      WebSocket frames; text frames are reserved for the `{cols,rows}` resize channel — a
      text-frame keystroke is silently eaten (`ui/src/components/chat/TerminalTab.tsx`).
- [ ] **Messaging:** either it can drain its mailbox (`check_messages`) or `ResolveRecipient`
      excludes it — an undrainable nudge loop once burned thousands of paid turns.
- [ ] **Turn boundaries:** its turn signal is identified and wired into every per-turn reset
      (`ResetTurnBudget` via `terminalTurnID`) — not lumped into a generic status hook.
- [ ] **Reconcile:** the sweep knows its shape (no chat-shaped preview stomping, §1).
- [ ] **Hooks/status:** feature flags scoped to the actual risk surface, not blanket-applied
      (`CHUCK_HOOK_REGISTRATION` once muted terminal agents entirely).
- [ ] **Capabilities honesty:** never advertise what no API/UI surface can select (`tmux:true`
      with no selector shipped).
- [ ] **Teardown:** joins §4's single teardown on every exit path.

## 7. Read paths must not swallow errors or amplify damage

**Rule:** iteration and repair code treats each record/entity as independently failable: check the
real error signal, skip the bad record, continue, and report — never abort the whole stream or
wipe-then-fail.

## 8. User-facing surfaces get parsed, bounded, in-vocabulary data — and errors always surface

**Rule:** anything rendered to the human (card previews, `status.detail`, toasts) is parsed,
human-meaningful, length-clamped at the write boundary, and drawn from the declared vocabulary.
Every mutating UI action surfaces failure; every notification funnels through the one payload
builder. A disabled action names the condition that blocks it, and alternatives offered together
state their different consequences beside the controls. A mutation's warning or success-adjacent
feedback stays owned by a surface that remains mounted, or is handed off before that surface closes.

## 9. Liveness & durability primitives are weaker than they look

**Rule:** the OS and SQLite primitives this codebase leans on all have sharp edges that already cut:

- **Bare `kill(pid, 0)`** proves only that *some* process has the PID (reuse!). Used in
  `internal/cli/pidfile.go` — corroborate (start-time/comm/nonce) where it gates destructive
  action, and never remove a pidfile without verifying it names your own PID. Orphan reapers and
  both reconcilers once trusted it and could SIGKILL a reused PID; they now compare the running
  row's recorded kernel birth (`runtime.ClassifyProcess`) and refuse unknown proof.
- **Atomic-write-via-rename** must fsync the file AND the parent directory
  (`internal/config/atomic.go`, `pidfile.go`).
- **`CREATE VIRTUAL TABLE IF NOT EXISTS`** freezes a degraded fallback forever — a
  capability-upgrading migration must detect and replace the fallback object
  (`ensureSessionsFTS`, `internal/state/migrate.go`).
- **Append-only writers** must truncate a crash-torn trailing record on `Open()` before appending,
  or the torn tail fuses with the next record (`internal/transcript/writer.go`).
- **In-memory accumulators feeding replace-style (DELETE+INSERT) writes** must lazily reseed from
  the durable table on first use per process — an empty-seeded buffer once wiped all FTS content
  after a restart (`seedLocked`, `internal/index/indexer.go`).
- **Destructive CLI ops sharing the live server's DB hard-refuse on liveness** (`reindex` once only
  warned).

## 10. Ship the wiring; kill the drift

**Rule:** a feature isn't done until every surface that promises it can reach it, and every doc
that describes it matches. Gating stubs are un-gated in the same effort that ships the gate.

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

## 12. External-CLI invocations must tolerate version and environment variance

**Rule:** any `exec.Command` of a user-installed tool (agent CLIs, probers) runs against whatever
version the user has, not the one the author tested. Optional flags need a detect-and-retry
fallback; output parsing is defensive (substring vocabulary, not exact format); a tool that can't
be interrogated reports "unknown"/"skipped", **never** "failed" — a wrongly failed gate blocks the
user harder than no gate at all. Exit zero is not success: validate the output's shape before
relying on it. What Chuck may then claim about the tool's effective state is §20.

**Canonical pattern:** `runClaudeAuthStatus` (`internal/backend/credcheck/claude.go`) — try with
optional flags, sniff the error output, retry bare. The `/usability-review` S3 sweep audits every
external `exec.Command` for this class.

---

## 13. The test suite cannot see CSS or layout

**Rule:** the UI styles itself with hand-written global CSS (`ui/src/styles/global.css`, tokens in
`tokens.css`) — no Tailwind, no CSS modules. A `className="x"` with **no** matching selector renders
as unstyled default-browser markup. Testing Library never evaluates CSS and MSW/Vitest render into
jsdom, so a whole surface can reference dozens of undefined classes and every unit test stays green;
the breakage is only visible in a real browser. Any TSX className string literal must have a
selector in a stylesheet (utility/state classes applied via template literals are the exception — they
carry their own defined selectors).

**Canonical guard:** the `/usability-review` S2 sweep three-ways the referenced-className set against
defined selectors (both directions) and every journey renders the surface in a real browser and
checks computed styles / stylesheet rule count, not just DOM presence.

---

## 14. A loopback bind is not a browser or filesystem security boundary

**Rule:** binding 127.0.0.1 keeps remote *sockets* out, but not remote *attackers*. A malicious
web page can still reach the server through the victim's own browser (DNS rebinding makes
attacker.com resolve to 127.0.0.1, so the page becomes "same-origin" with the dashboard;
cross-origin WebSocket handshakes and "simple" no-preflight POSTs are sent regardless of CORS
response headers). And any other account on the machine can read world-readable files. Therefore:

- Every HTTP route — API, raw-mounted transports (`/mcp`), WebSockets, static UI — must sit behind
  the `localOnly` guard (`internal/server/security.go`), which rejects non-local `Host` headers
  (DNS rebinding) and non-local `Origin` headers (cross-site WS/CSRF) with 403. The guard wraps
  the **entire mux** in `routes.go`; never mount a handler outside it, and never rely on CORS
  headers as access control — they only gate what a compliant browser lets a page *read*.
  (TS-05.R23) The tailnet remote listener is the one other chain: its routes sit behind
  the remote guard, device authentication, and allowlist of TS-13.R4/R5 instead — never behind
  neither guard, and never mixed with `localOnly`.
- Everything under `~/.chuck` (config with backend env/API keys, `state.db`, transcripts,
  hook/MCP token files, logs) is owner-only: `0o700` dirs, `0o600` files (hook scripts `0o700`).
  `MkdirAll` never re-modes an existing dir and SQLite creates files umask-relative, so creation
  paths must pass tight modes AND `EnsureLayout`/`state.Open` explicitly `Chmod` what may already
  exist from older builds.

**Canonical guard:** `localOnly` + `isLocalHost`/`isLocalOrigin` (`internal/server/security.go`).
Test requests must carry a loopback Host — use `newLocalRequest` (`server_test.go`), not bare
`httptest.NewRequest` (whose default Host, example.com, is rejected by design).

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

---

## 17. Tests must prove their contract independently

**Rule:** a regression, acceptance, agreement, or release test must have an oracle independent of
the implementation it is checking and must be capable of failing when the claimed contract breaks.
Derive the expected surface from a separate authority such as registered producers, the wire
boundary, artifact metadata, or an explicit requirement enumeration — never from a copy of the
table or helper under test. A regression test is confirmed against the pre-fix behavior when that
check is practical. Test doubles mirror the real producer's failure modes and serialized shapes,
not the idealized contract.

**Canonical patterns:** build request fixtures with the real client serializer and response
fixtures from a captured frame of the pinned peer; enumerate producers independently from their policy table; drive the real
wire/serialization boundary; make fixtures trigger the named OS or storage failure; inspect built
artifacts rather than trusting build commands; for a bug regression, demonstrate the test fails on
the old behavior before relying on it.

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

---

## 24. A fallback is a state change: fail closed, or make it visible, bounded and reversible

**Rule:** when the intended path is unavailable, the code either refuses or enters a second mode.
A second mode is real state: it is recorded as the mode now in effect, reported to whoever relies
on the result, limited in duration, and re-tested so the system returns to the intended path when
it can. A fallback never reports the intended path's success, never becomes permanent by default,
and never supplies a permissive or terminal value for missing evidence. Where the caller needs the
intended guarantee — permissions, history, identity — fail closed.

**Canonical patterns:** refuse where the guarantee matters; otherwise name the mode in the result
(`history_handoff` reports what actually happened) and in status; give every degraded mode an exit
condition and a test that it is taken; cache only validated responses; treat an absent field as
`unknown` (§20).

---

## Canonical helpers registry (reuse, don't re-derive)

| Helper | Where | Use for |
|---|---|---|
| `useInputDelivery` / `RecoveryOffer` / `withdrawHeldMessage` | `ui/src/components/chat`, `ui/src/lib/heldMessage.ts` | desktop and phone prompt submission, retry, recovery and withdrawal (§2, §19) |
| `teardownAgentRegistration` | `internal/server/launch.go` | every agent exit/failure path (§4) |
| `startResumedRuntime` | `internal/server/resume.go` | compose, start and persist explicit resume, wakes and both Switch starts under the lifecycle claim (§2, §4) |
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
| `seedLocked` | `internal/index/indexer.go` | in-memory buffers feeding replace-style writes (§9) |
| `config.ValidSlug` | `internal/config/validate.go` | every path-param on every verb (§2) |
| `foldTranscript` / `appendRenderedEvent` | `ui/src/store/transcriptStore.ts` | identical live and replay event projection (§2) |
| `transcript.ProjectEvent` / `runtime.AllEventTypes` | `internal/transcript`, `internal/runtime` | every Go transcript text consumer and exhaustive normalized-event coverage (§2) |
| `resolveTurnOutcome` / `runtime.DecodeTurnEnd` | `internal/runtime/turn_outcome.go` | resolve terminal turn evidence once and interpret the same outcome from live or stored events (§20) |
| `groupTranscriptRows` / `ToolRun` | `ui/src/components/chat/toolRun.tsx` | folding tool calls and results into one collapsed row on desktop and phone (§2) |
| `clipAnnotationExcerpt` | `ui/src/lib/annotations.ts` (server copy authoritative: `internal/server/sessions.go`) | every UI surface that captures an annotation excerpt (§2) |
| `copyText` | `ui/src/lib/copyText.ts` | UI copy actions that report clipboard failures as a toast (§2, §8) |
| `offerWebLink` / `claimWebLink` / `webLinkActions` | `ui/src/lib/linkActions.ts` | rendered web-link menus; every annotation menu claims the link so one right-click opens one menu (§2) |
| `capVisible` / `OverflowMenu` | `ui/src/lib/navCap.ts`, `ui/src/components/shell/OverflowMenu.tsx` | every compact nav's five-cap with current-visible and its `+n` disclosure (§2) |
| `displayLabel` / `displayLabels` | `ui/src/lib/labels.ts` | every model/backend/project/role/template label; the id appears only for duplicate names (§2) |
| `AutoGrowTextarea` | `ui/src/components/ui/AutoGrowTextarea.tsx` | every multi-line text field in `ui/src`; a source-scan test rejects raw `<textarea>` (§2) |
| `FileTabsReader` / `fileTabs` / `useFileTabRoute` / `readFileTarget` / `writeFileTarget` | `ui/src/features/fileTabs/` | every file reader surface, saved file-tab mutation and `?file=` address read/write (§2, §16) |
| `localOnly` | `internal/server/security.go` | wraps the whole mux; every new route inherits it (§14) |
| `notificationPayload` | `internal/bus/` | all notification payloads (§8) |
| `fakeacp` test double | `internal/runtime/testdata/fakeacp` | env-driven protocol-level repros (`FAKEACP_LOAD_DUMP`, `FAKEACP_PROTO_VERSION`, `ignore_cancel`); it drops out-of-schema session-request members like the pinned peer (§17) |
| `backend.Types` | `internal/backend/adapter.go` | enumerating every backend type instead of restating the union (§2, §17) |
| `backend.Support` / `launchSupportFor` | `internal/backend/adapter.go`, `ui/src/schemas/backends.ts` | every interface/effort/fast launch-support decision; never a browser or server backend-type allowlist (§2, §10) |
