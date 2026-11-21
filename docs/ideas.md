# AgentDeck ideas and improvements

This is a place to keep future thoughts without accidentally treating them as promises or approved
work. The specifications describe the product today; this file does not.

## New ideas

Put a half-formed thought here. It needs only a short title and, if useful, a sentence about what
prompted it.

Example:

```md
- **Pinned agents.** Let people keep frequently used agents at the top of the dashboard.
```

- **One rule for a background task's running state and Stop.** TASK-STOP-5 inventory (2026-11-19).
  TASK-STOP-1 was patched: Stop now addresses the root session. Writers: `onAsyncTaskUpdate`
  (`internal/runtime/tasks.go`) is the only server writer. It feeds `openTasks`, which becomes the
  persisted `background_running` (`applyLive` in `chat.go`), read by the agent card, the
  pipeline-stall and pipeline-task code, and the auto-settle gate (`registry.go`). The maps die
  with the generation's `agentState`, and no terminal event is written at teardown. The popover
  instead folds the transcript (`collectTasks` in `runtimeActivity.ts`) and fences running rows
  only at `resumed_at` or `fork_boundary`. Stop entry points: the desktop popover
  (`RunningTasks.tsx`), the operator CLI `background-task-stop`, and the
  `POST …/background-task-stop` route that both use (404 unknown, 422 unavailable, 409 otherwise).
  The phone has no Stop. Divergent or missing rules:
  - a subagent's end never closes its tasks, and the adapter drops later updates routed to an
    ended child. The row, the count and the auto-settle block then stay until teardown. Chuck also
    forgets ended children after `maxEndedChildren`, so their late updates are dropped too. Stop on
    such a row now reaches the adapter, but its terminal update is dropped, so the row stays
    “Stopping…” (TASK-STOP-2, Must fix, spec gap: R59 assumes the runtime keeps reporting);
  - the card count (server `openTasks`) and the popover (transcript fold) compute "running"
    differently. They disagree after a no-resume restart, a dropped late update or the 128-task cap.
    The server ignores `canStop`, while the UI hides Stop for it;
  - `as.stopped` leaves `openTasks` set, so pipeline readers may see a stale persisted count
    (unconfirmed);
  - the popover maps every failure to “Could not stop this task. Try again.” (TASK-STOP-3), and
    the server does not log a refused or failed stop (TASK-STOP-4);
  - the resume/fork fence, the 128/32 caps and FS-03.A41 (still planned) have no explicit rule.

  Governing items: FS-03.R59/R107, TS-04.R64/R104, TS-03.R44, FS-02.A64.

- **One owner for browser chat-input settlement.** INPUT-8 inventory from the shared desktop and
  phone delivery review. Browser text is written by composer editing, draft restoration and
  navigation, accepted delivery and agent deletion (`drafts.ts`, `useComposerEditing.ts`,
  `AgentScreen.tsx`, `heldMessage.ts`, stream removal). `draftIndex.ts` provides the draft revision's
  request identity and eviction. Attachment selection, rehydration and acceptance use
  `attachments.ts`; `useInputDelivery` writes and restores Send/Steer retry records, while
  `RecoveryOffer` holds the deliberate recovery command ID. The desktop held mirror is written by
  `Composer`, `useHeldProjection` and `heldMessage.ts`; the phone reads its own held query. The
  transcript may clear the matching held tail once its sequenced delivery arrives.
  Server Send, Steer and held withdrawal go through the session routes and runtime's single hold
  slot. Assignment admission and rerouting, runtime delivery/exit and restart update durable
  attachment receipts; exact recovery Send/Dismiss claims and settles the source there.
  Browser cleanup now has exact receipt evidence, but the draft index, IndexedDB retry rows,
  held mirror and echo still make separate decisions about when to release. Define the per-request
  transitions and ownership across acknowledgment, navigation, reconnect, partial storage failure
  and peer settlement alongside the existing sent-echo and held-tail idea.

- **Fix card drag-and-drop usability.** Dashboard card reorder (FS-02.R12) technically works but
  reads as broken: the drag listener is bound only to the tiny 28×28px `::` handle
  (`AgentCard.tsx`), not the card itself, so dragging the card body does nothing — and because the
  card's `onClick` navigates to the agent, a failed drag attempt looks like an accidental page
  change. Needs whole-card drag with an activation distance so a plain click still opens the agent.
  Cross-group membership drops are specified separately in
  `docs/ready-changes/agent-groups-on-desktop-and-mobile.md`.
- **No Content-Security-Policy.** Neither the Go server nor `ui/index.html` sets a CSP header or
  meta tag. CSP is the third layer every diagram/Markdown-rendering hardening guide recommends
  alongside sanitization, and it would also bound the existing Markdown, diff, and xterm surfaces.
  Kept separate from `docs/ready-changes/mermaid-diagrams-in-chat.md` on 2026-08-27 because it is a
  server-wide change that could affect the dev proxy, xterm, and inline styles. Noted while
  researching safe diagram rendering.
- **Choose an external base for AgentDeck-owned worktrees.** Support a deterministic per-project or
  per-repository checkout layout outside `$CHUCK_HOME` for operators whose repositories live under a
  separate workspace base. Ownership records, base-directory changes, recovery, migration and safe
  deletion need definition before changing the canonical worktree path.
- **One commit point for a started agent's identity and session.** From the shared resume start
  review's inventory (SW-START-4, 2026-11-15). SW-START-1–3 were patched: a Switch rollback
  restores the agent's selection and session snapshot in one transaction
  (`RestoreSwitchSource`) and records it in the transcript, and a fresh launch refused at its applied write removes its session,
  index rows, transcript and identity (`discardFreshLaunch`). Four records describe a started
  agent: the `agents` runtime columns, the `sessions` snapshot (no foreign key to `agents`, so
  `DeleteAgent` never removes it), the `running` row, and the registration (hook token, MCP
  token and file, hook settings, registry slot). Their writers per origin:
  - fresh launch and Clone: registration, `WriteAgent`, then the runtime writes the snapshot
    (`UpsertSessionMeta` from `openPersistence`) and the running row, then the applied identity;
  - Resume and wake: registration, snapshot, running row, applied identity; the identity is not
    written before the start;
  - Switch target: the target identity is written before the start (`switch.go` prepare), then
    snapshot, running row and applied identity; rollback restores identity and snapshot together
    and starts again;
  - Stop and exit: delete the running row (terminal and orphan reaping also copy its session ID
    into the snapshot; chat Stop and crash do not) and tear down registration;
  - startup: `ReconcileStale` removes exited running rows, `RestoreInterruptedSwitches` copies
    the snapshot's selection onto a stopped agent whose backend differs, and
    `recoverIncompleteLaunches` deletes an agent that has no snapshot.
  No rule says when a new process's snapshot becomes authoritative or who restores it, so each
  failure branch chooses. Divergences read in code, none patched:
  - a fresh start that fails inside the runtime after `openPersistence` (running or status
    write, exit during start; non-fork chat and every terminal start) deletes the agent but
    leaves the session row and transcript; the archive lists it and Resume answers 404. Only the
    applied-write branch calls `discardFreshLaunch`, and startup never removes a session without
    an agent;
  - a Clone refused after its start succeeded leaves the provider-side forked session;
  - a failed Resume restores nothing: the snapshot keeps the new native session ID, capabilities
    and, with overrides, a launch object and contract the `agents` row never received;
  - when `RestoreSwitchSource` itself fails and the target had not yet written a snapshot, the
    stopped agent has the target identity with the source's native session, and Archive Resume
    pairs them until the next restart; a model-, effort- or interface-only mismatch is never
    repaired at startup;
  - the transcript's newest session record rebuilds the row on reindex and feeds search, so every
    direct `sessions` write needs a matching record; the Switch rollback now appends one, while
    startup's `RestoreInterruptedSwitches` and the live effort/fast change do not;
  - Switch reads the source's native session ID before stopping it, so a terminal hook's newer
    ID arriving in between is overwritten by the restore (suspected, not reproduced);
  - `discardFreshLaunch` leaves the agent's `conversation_health` row, and an identity whose
    delete is refused (pending quota cleanup) stays without history until the next startup;
  - launch and Resume discard the stop error of a failed start that Switch reports, leaving a
    live process behind a generic error;
  - a project archive stops each agent without that agent's lifecycle claim;
  - the startup and periodic sweeps delete exited running rows without tearing down that
    generation's registration files;
  - Switch's stop of the old runtime reports `requested_stop`, which interrupts the agent's
    assigned task as a person's Stop would (unverified whether intended).
  Define the point at which a start's snapshot and selection commit together, the one rollback
  every origin uses before that point, and what startup removes. Coordinate the Stop and exit
  rows with "One settlement rule for a stopped or exited process" below.
- **One exit rule for a claimed chat turn.** From the agent-execution review's inventory (EXEC-12,
  2026-11-15). Each exit from a claimed turn gate decides on its own who ends the durable turn row,
  emits `turn_end`, settles the request receipt and runs or releases the held successor, and
  TS-01.R45 and INV §5/§19 state begin and end separately rather than as one exit matrix. EXEC-1–3
  were patched one at a time. EXEC-9 (idle Steer leaves `busy`) was fixed in `9493b929`.
  TURN-STOP-1–3 closed in `adc47b71`; the Stop and process-exit inventory is recorded under
  “One settlement rule for a stopped or exited process” below. This idea stays in the backlog
  pending a current inventory of chat-turn exits. Historical leads to recheck:
  - Stop ends the row but emits no `turn_end`, while crash and normal completion do;
  - finishTurn's `!ended` branch and onTransportClosed's `stopped` branch assume Stop settles,
    which holds only when `stopComplete` is set; a bare `shutdownProcess` skips every settlement
    (checked: every bare `shutdown()` caller is a start-failure or quota-probe path, so latent);
  - a quota block releases the gate but leaves the held message and Steer fallback in place with
    receipts `held`, and `SteerMessage` refuses until a later turn end or Stop;
  - a refused begin's receipt differs by caller (checked: `beginTurn`'s refusal now writes
    `released` for every caller, so nothing stays `submitted`; `runReservedHeld` then also writes
    `failed`);
  - an ordinary prompt's receipt is never moved past `submitted` in the runtime;
  - Stop's row end names the `execTurnID` captured after `finishTurn` may have cleared it;
  - after a failed turn-end write the row and status stay `running`/`busy` on an idle agent until
    the next begin or exit, with no sweep to correct them (INV §18);
  - when Stop's final `done` write wins the race, a completion that already settled quota and
    mail as `completed` loses its `turn_end` and the row records `process_ended`;
  - a prompt released before send already has its transcript bubble, so recovering it shows it
    twice.
  Define, per exit, the owner of row, `turn_end`, receipt and successor, and whether a gate claim
  without a turn row may exist at all.
  ARCH-TURN-1 (architectural review of v0.23.0..d26e5fe2) was one more row here: a turn refused
  before it reached the provider stored `refused` and composed to `error`; it now stores
  `not_started` and composes to `idle`.
- **One recovery rule for a settled pipeline owner.** From the v0.23.0..d26e5fe2 architectural
  review's inventory (ARCH-STALL-2, 2026-11-17). ARCH-STALL-1 was patched: a live owner whose turn
  failed (`error`) is now continued, then paused, like an idle one. The turn-end nudge
  (`nudgeStalledOwner`) and the sweep (`PipelineOwnersNeedingRecovery`) now agree on `idle` and
  `error`, but each other settled state is decided in a different place:
  - quota-held: both skip it; quota recovery owns it, and an owner that exits while held is not
    interrupted (`interruptTaskOnExit`). Unverified that quota continuation always reaches a
    pipeline stage owner;
  - `waiting_input`: both skip it and only the attention label changes; nothing pauses an
    abandoned permission prompt (unverified that a resolve re-runs the nudge);
  - background work, pending yield/release and child waits: skipped and re-evaluated at the next
    turn end; the sweep still selects child-waiting and quota-held rows every pass only for the
    runtime gate to refuse them;
  - `error` from a failed Switch: selected by both, but the runtime has no handle, so the nudge
    does nothing silently; unverified whether anything else interrupts the task;
  - process exit and Stop: the exit hook interrupts the task, then the sweep recovers or pauses.
  Ordinary (non-pipeline) task assignees have no stall detector at all: a turn ending without a
  report leaves the task running until the process exits. State one table in TS-09.R70 (and
  decide FS-16 for ordinary tasks): per settled owner state, who continues, pauses or
  deliberately leaves the visit, and what later event re-evaluates it (INV §8, §18).
- **One settlement rule for a stopped or exited process.** From the single-owner turn state
  review's inventory (TURN-STOP-3, 2026-11-15). TURN-STOP-1/2 now settle Stop by writing `done`
  and ending the generation's turn before deleting the running row. Live chat Stop goes through
  `endTurn` under the status fence. Reaped, no-handle and reconciled rows share
  `runtime.SettleStoppedRow`. A failed write keeps the row, and a retry or the liveness sweep
  settles it. The other exits still decide ordering on their own:
  - chat `onTransportClosed` deletes the running row whatever `endTurn("error")` returned, so a
    failed write leaves `busy` and a `running` turn that no sweep can find (INV §15, §18);
  - the terminal runtime's owned Stop and exit watcher delete the row first, then `setDone`
    discards its write error, and neither ends the turn row;
  - `WriteTurnEndStatus` returns success without writing when the running row names another
    generation, and `SettleStoppedRow` then deletes that row anyway (INV §23);
  - `StopGuardedIdle` callers (task dispatcher, pipeline stage stop) return an unsettled Stop's
    error without the orphan reap, so they depend on a later call reaching `ErrNoHandle`;
  - `reapOrphanRuntime` treats any running-row read error as "already gone";
  - `Registry.Stop` forgets the claim on errors other than unverified or unsettled;
  - `failSwitch` and terminal `scheduleInitialIdle` write status outside any fence.
  Define, per exit (Stop, guarded idle stop, reap, crash, reconcile, terminal), which store
  transaction settles the running row, turn row and final status, its order, and how a failed write
  is retried. Coordinate with the claimed-turn exit rule above.
- **One terminal decision for a turn across result, Cancel, Stop and exit.** From the typed turn
  outcome review's inventory (OUTCOME-7, 2026-11-15). OUTCOME-1–6 were patched: EOF waits for a
  delivered prompt result to be claimed (`promptDecided`), an exit publishes `turn_end` only for
  a turn it interrupts, the exit hook carries the interrupted turn's ending, and clients render
  the server-decoded `outcome`. Who settles what today, per exit:
  - prompt result or RPC error (`completePromptTurn` → `finishTurn`, ordinary and activation
    prompts alike): claims `promptSettlement` under the turn lock; settles quota, then inline
    mail, then the row and status in one write (`endTurn`), then `turn_end`; the room settles
    from that event (`finishThinkTankTurn`); the held successor starts in the same goroutine;
  - accepted Cancel (`cancel`): claims nothing itself; it marks `cancelledTurnSeq`, and the
    prompt's own completion resolves `cancelled` above. If the peer ignores it, escalation
    kills the process and the exit below settles as cancelled;
  - unsolicited exit (`onTransportClosed`): waits for the prompt's claim, then settles the
    interrupted turn's row and `error` status, quota, the exit hook (room, task, registration)
    and only then `turn_end`; it never settles inline mail, whose reservation waits for the
    next turn's reset;
  - Stop (`Stop`): fences with `stopComplete`, ends the row as `process_ended` with status
    `done`, emits no `turn_end`, settles neither quota nor inline mail, and the room and task
    settle from the `requested_stop` hook.
  Legal order today when they overlap: result before EOF → the result wins, then the exit adds
  only its fatal error; Cancel before result → cancelled whatever the result says; Cancel
  escalation then EOF → cancelled from the exit; Stop at any time before the result's claim →
  Stop wins and the result is dropped. Divergent or missing rules, unverified unless noted:
  - Stop still discards a result the transport already delivered, the same race OUTCOME-1
    closed for an exit (confirmed by reading: `completePromptTurn` returns on `stopping`); the
    turn records `process_ended` and completed-turn quota and mail are not confirmed;
  - the room and the task dispatcher settle from two signals (exit hook, then `turn_end`) and
    agree only where each was patched: a cancelled exit now holds the room as cancelled, but
    `interruptTaskOnExit` still treats it as `unexpected_exit` and may recover the work
    (OUTCOME-8, attached here);
  - a held successor begun inside the completing goroutine while the process exits has its
    receipt marked `unknown`/`provider_error` although nothing was written to the provider;
  - an exit that interrupts a turn leaves its inline-mail reservation to the next reset, and an
    agent that never runs again keeps it;
  - a turn gate that is reserved but not begun (idle Steer, a claim under preparation) counts
    as an interrupted turn on exit and gets a `turn_end` with no `user_text` before it;
  - `transcript.Project` rejects an undecodable `turn_end` while reconciliation now keeps the
    boundary, so index projection and the card preview treat the same record differently.
  - a process that dies before EOF is read can fail the prompt's write first, and the turn then
    settles `failed` rather than `process_ended` (independent check, not reproduced);
  - the browser marks a running child activity disconnected only at a root `turn_end`, so one
    started outside a turn would stay active after an idle exit (independent check, unverified
    that activities can start while idle);
  Define one function that takes the terminal evidence (result, RPC error, Cancel claim, Stop,
  EOF) in transport order and returns the single owner for row, status, `turn_end`, quota, mail,
  room and task; state Stop's rule for a delivered result; and make the exit hook and `turn_end`
  one signal for server-side settlement. Coordinate with the two entries above.
- **One owner for a sent prompt's echo and the held tail.** From the request-ID matching review's
  inventory (PROMPT-4, 2026-11-15). PROMPT-1–3 were patched: one echo per request ID, both desktop
  surfaces share `useHeldProjection`, and a hold read begun before a Send's accepted hold
  (`acceptHold`) leaves it alone. Writers today:
  - desktop echo (`transcriptStore`): inserted only by `Composer` on an accepted, non-held,
    non-queued Send, skipped when a sequenced event with its request ID (or, legacy, its text after
    `beforeSeq`) is present; replaced or suppressed by `reconcileUserMessage` on SSE append and on
    `setTranscript`; kept across a refetch only inside `beginReconciliation`, and dropped by a
    refetch without one and by `discardAgent` (SSE reconnect);
  - held tail (`heldStore`): set by `Composer`'s held acceptance and by `readHeldRecovery` with
    `includeHold` (surface mount and agent state changes, self-annotation delivery); cleared by
    the delivery match, held Steer, `useInputDelivery`'s release refresh, withdrawal, an empty
    snapshot and the stopped-composer restore; phone keeps its own React Query hold snapshot and
    releases the desktop store only on refresh.
  Missing or divergent rules: no evidence other than its own delivery or a refetch clears an echo
  whose request was later released, withdrawn or failed, so its lifetime depends on whether a
  reconciliation was open; `acceptHold`'s revision and the `invalidateHeldReads` stale flag are two
  fences for one projection; desktop and phone derive held state from different snapshots. Define, per
  request ID, which receipt state inserts, replaces or clears the echo and the tail on both
  surfaces across acknowledgment, delivery, refetch, reconnect and navigation. Coordinate the
  hold portion with INPUT-8's inventory.
- **One transition table for a held message and a Steer.** From the STEER-HOLD review's
  inventory (STEER-HOLD-6, 2026-11-20). STEER-HOLD-1/2/3/5 were patched: a refused promoted Steer
  restores the original hold (same `held_id`), a refused Steer leaves no visible user message, outcome
  notices are 3s toasts, and adapter refusals and `held_changed` are logged. Writers of
  `turn.held`: `SendMessageOrHold` (replace), `resolveSteerFallback` (refused-promotion restore),
  `restoreRefusedHeld` (history refusal), and clears in Withdraw, Steer promotion,
  `reserveHeldSuccessorLocked`, Stop and `onTransportClosed`; `claimTurnOrHold` also writes it
  with no `held_id` or receipt and appears to have no caller. Writers of `turn.steerFallback`:
  `SteerMessage`, `resolveSteerFallback`, `reserveHeldSuccessorLocked`, Stop and
  `onTransportClosed`. FS-03.R48–R50/R100, TS-03.R39 and INV §19 state outcomes but no transition
  table. Missing or divergent rules (code-confirmed unless marked):
  - Stop clears the hold and fallback before writing their receipts; Withdraw and transport close
    write the receipt first;
  - a refused Steer leaves `failed` when typed but `held` when promoted, and a promoted refusal
    whose hold was replaced meanwhile sets the receipt back to `held` with no hold, so the
    message is neither held nor offered for recovery;
  - a held successor reserved at turn end keeps receipt `held` until `submitted`; a Stop in that
    window has no owner for its receipt (suspected);
  - a quota block leaves the hold with no receipt change and no stated release (suspected);
  - a refused begin moves a held successor `submitted` → `released` → `failed`;
  - an `unknown` receipt left pending across the native Steer call has no stated resolution after
    a crash, and `held_id` minting and preservation are unspecified.
  Define the legal moves for Send, Steer, Withdraw, turn end, Stop, crash, quota block and refused
  begin, with the receipt and transcript write each makes and their order. STEER-HOLD-7 records
  the Steer before delivery and settles that row after the provider responds. Coordinate with
  the entry above and the QUEUED-TAIL review's inventory.
- **One transition table for assignment-queued input.** From the QUEUED-TAIL review's inventory
  (QUEUED-TAIL-4, 2026-11-20). QUEUED-TAIL-1–3 were patched: a claimed receipt stays listed as
  queued until `submitted`, every surface hides a card whose own `user_text` is in history, the
  continuation claims and records it only after the Cancel wait and stop check, and the card wraps
  long tokens. Writers of `attachment_receipts` for this input: acceptance
  (`QueueAssignmentInput`, `bindAssignmentInput` for pipeline continue-from-owner), owner
  replacement (`pipeline_tasks.go` moves `assignment_task_id`, not `agent_id`), claim
  (`TakeQueuedAssignmentInput` at continuation, `ClaimQueuedInputForTurn` in the sweep),
  content-preparation failure (`failed`), runtime `submitted`/`released`/`unknown`, the sweep's
  refusal (`released`), ended-assignment release (`releaseEndedAssignmentInput`) and startup
  release. Readers: the prompt snapshot's `queued`, history and search (only after `user_text`),
  the desktop held projection and the phone screen. FS-03.R48/R118, TS-09.R69, TS-10.R29 and INV
  §19 state outcomes but no transition table. Missing or divergent rules (code-confirmed unless
  marked):
  - no event is published on claim, submit, release or failure, so a card leaves only on a
    snapshot refetch or its own `user_text`; a release or failure stays visible until the next
    state change or recovery event;
  - Stop, Cancel, task cancel and run stop do not touch queued receipts; release waits for the next
    `QueuedInputWork` pass and only when the assignment ended;
  - failures before the provider write end `failed` (refused, send again deliberately) on content
    preparation but `released` on refusal, emit failure or a Stop after claim;
  - `SetAttachmentDelivery` has no from-state guard, and the runtime and the sweep both write it;
  - a claim rewrites `agent_id` to the assignment's current agent, so recovery for a moved receipt
    is offered in the new owner's conversation, and owner replacement leaves it unlisted for the
    new owner until claimed (suspected end to end);
  - only a person's prompt queues; Steer, annotation Steer and Edit dispatch never check a waiting
    assignment, and queued input has no withdraw or replace;
  - one receipt is claimed per continuation; the order between the two claim paths is unspecified;
  - the ordinary Send path, which also carries queued input the sweep claims for an idle running
    assignment, still writes its `user_text` before the Cancel wait, so a Stop that lands there
    releases the receipt for recovery while history keeps the row (the QUEUED-TAIL-2 sibling,
    found by the fix's independent check; the same write-ahead choice as STEER-HOLD-7);
  - a Stop after the final start check but before the provider write leaves the row with an
    `unknown` receipt, for every prompt kind (suspected window, not reproduced).
  Define acceptance, claim, record, provider write, Stop, Cancel, refusal, owner replacement,
  restart and recovery with the receipt and transcript write each makes and their order.
  Coordinate with the two entries above.
- **One commit point for an Edit's history, session and replacement.** From the EDIT-REVIEW
  inventory (EDIT-REVIEW-10, 2026-11-20). EDIT-REVIEW-1–3, 5–9, 11 and 12 were patched: fork-point
  lineage across committed edits, one-transaction dispatch, reservation decided under the turn
  lock with background/held/Steer/task refusals, clone and saved-link reads, follow's reset
  boundary, a durable offer for an aborted edit, and the later-turn refusal. Phases are enforced
  only by `WHERE phase=` guards in `state/edits.go`; TS-01.R51 and TS-02.R69 state outcomes, not
  a transition table. Entry points: Send, Steer, held promotion, activations (mail, quota, think
  tank), owner continuation, task dispatch, Resume/Start, Stop, turn Cancel, peer Stop, group
  release, switch and clone. Admission is excluded by the lifecycle claim (held for the worker's
  life) and the runtime reservation. Writers of the native selection: `FinishEdit`,
  `ApplyHook` SessionStart (ignored only while preparing), `RestoreSwitchSource`,
  `DeleteRunningRetainingSession`, `appendRestoredSessionMeta`, `PublishStaged` and reindex. The
  revision becomes visible in three steps: `DecideEdit` (HistoryRevision), the journal append
  (fold), then `FinishEdit` (selection). Missing or divergent rules (code-confirmed unless marked):
  - EDIT-REVIEW-4 (Must fix, carried here): between the journal append and `FinishEdit`, history
    shows revision N while `last_session_id` names the old session; the transcript handler samples
    native session, revision and history separately and does not consult the open edit. A
    `FinishEdit` fault left revision 1 visible with the old selection, and Resume returned "agent is
    already running." Readers must reconcile or report editing, and no ordinary work may be
    admitted while they disagree; exercise read, stream and Resume on both sides of the decision;
  - if `DispatchEdit` fails the committed edit stays open, but the reservation and lifecycle claim
    are released, so activations, mail and task paths can run on the revised conversation before
    its replacement until restart;
  - a Stop landing after `completeEditRevision` but before `dispatchEdit` still dispatches
    (`ctl.stop` is not re-read);
  - TS-02.R69 says recovery may submit an undispatched replacement once; the code only releases it;
  - `editReasonNotStarted`/`SearchUnavailable` are set on an edit that dispatch then closes, so
    `EditOperation.Reason` stops projecting them (suspected; UI not checked);
  - a room waking a member during Edit fails on the lifecycle claim and is dropped to the next
    sweep with no hold or reason; no explicit room-ownership refusal exists despite TS-01.R51;
  - owner continuation's `decide(settled)` may record a disposition before a claim the
    reservation then refuses (suspected);
  - `quotaTranscriptCounters` reads active history although TS-02.R70 lists audit counters as Raw
    (suspected);
  - EDIT-EDGE-4 (carried here): group release interrupts an editing member before its other
    claims can fail, so a later 409 leaves that Edit aborted.
  Define acceptance, decision, journal append, selection, publication, dispatch, Stop, Cancel,
  restart and repair with the reader view and admission rule at each step.
- **One owner rule for the file reader, its menus and focus.** From the Browse project files
  review's inventory (PROJECT-FILES-7, 2026-11-20). PROJECT-FILES-1–6 were patched: the dialog
  keeps its requested scope, a project switch remounts the scoped dashboard, Close/Escape return
  to **Files**, a keyboard row menu focuses its first action and returns to the row, the file
  viewer claims a rendered link so one menu opens, and chat and project menus share
  `fileActions`. Writers of the displayed file: `useFileReader` (`FileViewer.tsx`) resets on
  `path` only and has a per-read token; tab hosts (`FileTabsReader key=owner`), the project
  preview (`key=projectId`), the standalone preview page (`key=project/path`) and
  `ScopedProjectView key=id` remount by key; `useFileTabRoute` (`?file=`, Back/Forward) and the
  route-less phone `useFileTabSet` drive tab selection. Focus movers: the project dialog, Focus
  file dialog and Think Tank sources dialog restore focus; tab close refocuses the strip;
  `onEmptied` returns to the transcript (desktop), the Files heading (phone) or nowhere (Think
  Tank). Menu owners: FileViewer header/body/HTML bridge, SanitizedMarkdown links,
  AnnotationContextMenu (Transcript, ReaderWorkspace, Think Tank), FileTabs tab strip, FilesTab
  rows, the project row menu and the Think Tank files list. Missing or divergent rules
  (code-read, not reproduced):
  - an `agentId` or loader change without a path change re-reads but keeps the old loaded view,
    and the loader is not a dependency; correctness rests on every caller keying the reader;
  - stale-read fencing differs: reader token, the dialog's request counter, `FilesTab`'s
    unmount-only guard (a quick agent change can land late data), none in the tab hooks;
  - only the project row menu returns focus from a keyboard menu; the viewer, tab strip,
    FilesTab, Markdown and annotation menus never focus an item or restore focus, and focus drops
    to the body when the focused reader node unmounts;
  - one-menu-per-right-click uses three mechanisms (`claimWebLink` microtask claim,
    `stopPropagation` in the tab strip, plain `preventDefault`); TranscriptView's `openMenu`
    returns early without claiming;
  - Escape layering differs by dialog: DOM `.context-menu` check (Focus file, project dialog),
    per-state check (Think Tank), plus every menu's own window listener.
  Define the reader's identity (owner, loader, path) and reset rule, one focus-return contract
  for keyboard-opened menus and emptied readers, and one claim rule for nested menus.

## Ideas being defined

These are worth shaping into a possible change, but are not ready to build. Defining an idea updates
the relevant feature and technical specifications; it does not change product code.

- **Concise task instructions and guidance loaded by job.** Selected by the operator on
  2026-11-21 with `/design-feature`, following TASK-PROMPT's investigation. Remove repeated
  communication procedure from task prompts; give ordinary workers the essentials of assigned
  tasks and inter-agent communication through shared Chuck context and the existing
  `operating-chuck` skill. Keep personas focused on their purpose. Assess progressive reference
  boundaries before introducing separate skills: the current entry is about 400 words and routes
  four references; the complete package is about 6,000 words, not automatically loaded in full.
  `coordinate-work.md` mixes worker essentials with task creation, scheduling, repair, waiting
  on children and peer-history reads. Proposed direction: one short skill entry, worker-relevant
  guidance separately readable from coordinator/operator/pipeline/room details, and no repeated
  manual in assignment prompts. Preserve actual reporting, messaging, authority and recovery
  behavior. The operator confirmed that opting out of Chuck awareness is the user's choice:
  do not inject a fallback reporting reminder for unaware assignees. Proposed concrete delta:
  new-task first prompts contain only the authored instruction for both awareness choices;
  existing-agent task activations retain their short assignment-discovery trigger but lose the
  reporting/communication paragraph. Replace the shared context's vague outcome-reporting
  sentence with the essential assigned-task lookup and explicit result-reporting rule, delivered
  only to aware agents. Shorten the skill entry to basic choices and a reading map. Separate
  assignment and messaging basics from coordinator-only delegation, scheduling, child waiting,
  repair and peer-history guidance within the same skill package. No persona copies or new
  awareness exceptions. Product-specific instructions for pipeline decisions and Think Tank
  contributions remain outside this cleanup. Pending confirmation of this concrete feature
  delta before technical design and planned FS/TS authoring; no ready change yet. Governing
  existing scope: FS-18.R3–R6/R15/R20, FS-16.R26/R49.

## ACP Wait-list

These are capabilities AgentDeck implements above ACP, or has deliberately deferred, because the
pinned adapter contract is missing or unverified. An adapter release is a reason to recheck the
capability; it is not by itself authority to remove the fallback or ship the deferred feature.

- **Typed quota read and fork point in standard ACP.** No longer waited on: the operator approved
  packaged-adapter routes on 2026-11-11
  ([ready change](ready-changes/native-quota-snapshot-and-message-fork.md)). When ACP standardizes
  a subscription-quota read or a fork point, recheck whether TS-04.R109's two quota patches and
  R110's `_meta.jetbrains.air.fork` use can be dropped. Other vendor extensions still need the
  operator's explicit approval.
- **Steering.** The pinned Claude 0.85.1 and Codex 2.1.1 adapters advertise
  `_session/steering`, and AgentDeck uses it. Codex 2.1.1 still starts a detached turn when a steer
  arrives idle and ignores AgentDeck's `promptRequired` metadata, so the packaged steering patch
  remains necessary until upstream advertises the same no-consumption contract (still absent in
  Codex ACP 2.2.2, rechecked 2026-10-14).
- **Host-held queued Send.** AgentDeck holds a busy agent's next prompt because the pinned Claude
  adapter queues while the pinned Codex adapter supersedes and interrupts the active turn. Keep the
  host-side hold even after steering exists: Send must remain portable, withdrawable, and distinct
  from Steer.
- **Internal actions without MCP.** AgentDeck's fifteen coordination actions remain on its scoped,
  authenticated HTTP MCP server. `migrate-internal-actions-from-mcp.md` is paused until packaged
  Codex/ACP exposes a narrowly scoped direct transport reachable under the default sandbox. The
  gate stays closed. Rechecked 2026-10-05 against Codex ACP 2.1.1 (`acp:false`) and Claude ACP
  0.85.1 (`http`/`sse` only): still closed. Rechecked 2026-10-14 against Codex ACP 2.2.2 (`acp:false`,
  `type:"acp"` rejected) and Claude ACP 0.90.0 (`acp` servers silently dropped): still closed; ACP
  SDK 1.8.0 keeps MCP-over-ACP unstable. Watch ACP's unstable MCP-over-ACP server type (now
  stateless `mcp/message` in ACP SDK 1.7.0); adopting it would change the migration's goal and
  needs design review (TS-04 §5).
- **Semantic agent wake.** Mail and task activation use a short, host-generated `session/prompt`
  because ACP exposes no portable notification that wakes an idle model. Replace this bridge only
  if an adapter advertises a semantic wake capability; steering is not that capability.
- **MCP resources and templates for context.** Context links stay exposed as bounded MCP tools. MCP
  `resources/list` and `resources/read`, and ACP `Resource`/`ResourceLink` delivery, remain deferred
  until their behavior passes real Claude and Codex provider checks; prompt-level resource-link
  support alone does not satisfy that gate.
- **Declared MCP tool output schemas.** Structured tool results ship without declared
  `outputSchema`. Add schemas only after the pinned Claude and Codex adapters' handling of them is
  verified against real providers.
- **Codex session model and effort delivery.** AgentDeck applies model and effort through ordered
  post-session configuration because Codex ACP session creation has no model field. Keep this path
  until a replacement is both advertised and provider-verified.
- **Codex system-prompt delivery.** AgentDeck injects its prompt overlay through
  `CODEX_CONFIG.developer_instructions` because Codex ACP does not consume generic ACP
  `systemPrompt`. Keep and reverify the overlay on every adapter bump until the adapter exposes a
  proven portable replacement.
- **Codex executable authority.** [FS-09](specs/features/FS-09-backends.md)'s shipped
  installed-provider default and explicit bundle selection govern the native executable; the
  packaged adapter patches and deterministic bundled inventory remain required. Recheck those
  adapter/provider boundaries on updates rather than reviving the completed provider-selection
  ready change.

## Known things to improve

These describe incomplete or deliberately limited shipped behavior. Their owning specification is
the authority; move an item to ready changes only after its exact requirements and acceptance checks
are clear.

- **Same-machine trust boundary.** The loopback API relies on same-machine trust, and agent
  processes inherit the full environment except for backend strip keys (TS-05 §5). Revisit loopback
  authentication only with an explicit threat model and UI/CLI handshake design whose benefit
  outweighs its setup and compatibility cost; remote-control device authentication (FS-20) is a
  separate concern. Revisit an environment allowlist only if it preserves provider compatibility.
  Concrete case found 2026-11-14: the Origin check accepts any local port, the terminal WebSocket
  relies on it, and handlers decode JSON whatever the Content-Type, so a page served from another
  local port (an agent's dev server) can open a PTY socket and send launch or prompt requests.
  Pinning the Origin to the server's own origin plus the configured dev origin is a
  compatibility decision.
- **Chat and tracking refresh fidelity.** Make replayed streaming deltas match live deltas; show
  chat initial-load errors instead of a silent panel (FS-03 §6); refresh visible files/commands
  without stale-request overwrite; and let hook-only activity update recency.
- **Cross-turn transcript search.** Turn-document indexing intentionally chooses a small design over
  the more complete segmented model: all query terms and quoted phrases must occur within one turn,
  annotation flush, metadata document, or migrated legacy document. It does not combine a term from
  one turn with a term from another, does not match a phrase across turn boundaries, and one
  pathologically large turn can still use substantial temporary memory. Revisit with size-bounded
  chunks, boundary overlap, and per-term session aggregation only if real sessions show oversized
  individual turns or users repeatedly fail to find conversations because their query spans turns;
  those additions otherwise impose more schema, ranking, pagination, and phrase-boundary machinery
  than the observed long-session rewrite problem warrants.
- **Coordination notice hygiene.** Republish unread counts after janitor expiry, notify only on the
  first budget breach, and remove the duplicate waiting-input/permission notices (FS-06 §6).
- **Terminal and backend loose ends.** Either add an optional terminal driver picker or stop
  advertising unreachable drivers; implement or retire the planned terminal tab cap (FS-07); make
  OpenCode/OpenHands executable overrides actually launch and give them missing/old-CLI guidance
  like Claude's (FS-09 §6). Codex terminal rejection is intentional (FS-07 §6), not a gap.
- **Federation UI and watches.** Expose custom roots/profiles, refresh the effective view after
  source events, register prompt watches after binding, and clear preview consent on project change.
- **HTTP API contract.** Decide how mixed legacy error envelopes converge, and define shared JSON
  request-body limits with a structured over-limit error before enforcing them (TS-03 §5).
- **Frontend state ownership.** Define Zustand/React Query ownership and mutation-error behavior
  before broad frontend refactors. The 2026-11-14 classification put about a quarter of all
  recorded findings in "the client showed stale or wrong state". A candidate rule: every server
  resource family carries a revision, the stream announces only "family at revision N", and
  desktop and phone share one data layer.
- **Process and filesystem hardening.** Corroborate process identity and serialize concurrent
  events. Decide whether startup repairs existing descendant modes and whether valid-name
  role/project files may be symlinks (TS-05 §5); add adversarial tests for the chosen rules.
  Crash cleanup and the shutdown budget moved to
  [`give-agent-execution-one-owner.md`](ready-changes/give-agent-execution-one-owner.md) on
  2026-11-14.
- **One server per home.** The single-instance guard is a pidfile checked with `kill(pid, 0)`
  (ADV-9). `dashboard start` checks it before it writes it, after opening and reconciling the
  store, so two starts in that window both proceed; a doomed second start still reconciles and
  releases attachment holds in the live server's database; and after a reboot a reused PID makes
  `start` report a running server and `stop` signal an unrelated process. An advisory lock held
  for the server's lifetime fixes these, but a design review on 2026-11-14 found two cases to
  decide first: a detached start must still report a failed bind, and a server from an older
  version never holds the lock, so mixed versions need a rule.
- **Checkout deletion and gitignored files.** The dirty check (`git status --porcelain`) does not
  report ignored files, and consented deletion removes them, so a checkout reported clean can
  still lose `.env.local`, a local database or ignored notes. FS-19.R8 says uncommitted changes
  are the only thing deletion can lose. Decide what the dialog discloses before changing the
  probe. Found 2026-11-14 (ADV-6).
- **Host settlement of assignment turns.** A task is waiting or finished only when its agent
  calls `wait_for_tasks` or `report_task_result`. An ordinary task whose agent just ends its turn
  stays `running` with nothing shown; only pipeline owners are nudged. Agents also fetch and echo
  an `execution_handle` the server can derive from the caller's token. Candidate changes, each a
  product or agent-protocol decision: record "ended without a result" for every task, treat a
  turn that ends with unfinished children as waiting, drop the handle from the tools, and retry a
  transient start failure with backoff before parking. Host-run stage checks would require a
  distinct execution design. Pipeline setup simplification covers configuration of existing checks.
  The [structural pass](archive/reviews/adversarial-2026-11-14/ARCHITECTURE.md) has the evidence.
- **Quota attention on a pipeline run.** A run whose stage owner's quota intent ends in
  `needs_attention` stays `running` with nothing shown on the run; only the agent's own banner
  says so. TS-09 specifies the suppression of the stall nudge while quota-held, so what the run
  should show, and when it clears, is a product decision. Found 2026-11-14 (ADV-10).
- **One request protocol for every plane.** Mail, tasks, quota continuation, rooms and the
  pipeline owner each have their own claim, attempt, retry and restart protocol (eight in all;
  TS-01.R19 assigns those semantics to each domain). A single durable turn-request protocol with
  a per-origin policy table would replace them. A design review on 2026-11-14 found it cannot be
  introduced without behavior decisions: it changed mail's attempt count after a failed wake, a
  person's Send during a host-initiated start, and when a stopped pipeline owner is woken.
  [`give-agent-execution-one-owner.md`](ready-changes/give-agent-execution-one-owner.md) builds
  the two seams it would need (one turn begin and one admission function) without changing any
  protocol. Decide per origin before designing it.
- **Small truthfulness gaps found 2026-11-14.** A `starting` task whose settle write failed is
  retried only at the next restart. `send_message` accepts the sender as its own
  recipient and creates a wake for it. `--rollback` reports success for a binary that will refuse
  the migrated database. The update manifest's archive name is not restricted to a base name
  before download. A failed mail wake is logged at Debug and the sender only ever sees
  `queued_waking`.
- **Provider conformance from captured frames.** Five of nine field bugs were assumptions about a
  provider or adapter. Derive the fake adapter's frames from captures of each pinned adapter
  version and replay them in the release check; keep one list of every vendor extension, patch
  and text parse with its fallback.
- **Database durability, backup and integrity.** `synchronous=NORMAL` lets a power loss roll back
  committed task, quota and message rows while the transcript keeps them, and TS-02 states no
  durability level. There is no pre-migration copy and no startup integrity check.
