# Inventory: representations of agent state

Read-only inventory at `0a8bda69`, 2026-11-14. Evidence for [ARCHITECTURE.md](ARCHITECTURE.md) and
the checklist for [the ready change](../../../ready-changes/give-agent-execution-one-owner.md).
Paths are under `internal/`. Counts come from grep over non-test Go and were not re-verified
line by line; re-read before relying on one.

**Counts.** 31 durable facts across 17 tables; about 27 in-memory structures keyed by agent
(about 40 as raw maps and fields); 30 availability-predicate sites for about 14 distinct
questions; 17 call sites that write the `status` row; 16 reconcilers. There is no table of turns.

## 1. Where a turn is recorded today

- **Open turn:** memory only, `agentState.turn.turnActive` (`runtime/chat.go:362`).
- **Identity:** `execTurnID`, `settledTurnID`, `turnSeq` in memory; copied to
  `tasks.execution_turn`, `quota_intents` (`executing_turn`, `continued_turn`, `manual_turn` with
  their generations), `think_tank_attempts.turn_id`, `pipeline_runs.owner_stall_turn_key`,
  `messages.inline_delivery_turn_key`, `turn_budget (agent_id, turn_id)`.
- **Outcome:** the transcript `turn_end` event (`runtime/turn_outcome.go:93`) and, for room turns
  only, `think_tank_attempts.state` and `disposition`. Task and quota rows keep the turn id, not
  the outcome.
- **After a restart** nothing rewrites `status.state` from a turn; readers of `turn_end` are
  `server/reconcile.go:184` (preview), `server/think_tanks.go:512`, `server/quota_notice.go:60`,
  `server/fork_point.go:24` and the reindex.

## 2. Status writers (17 sites, one delete)

| Site | States | Guard |
|---|---|---|
| `runtime/chat.go:587` Start | `idle` | status lock, not finalized |
| `runtime/chat.go:1735` Resume | `idle` | same |
| `runtime/chat.go:1154` prompt turn start | `busy`; error ignored | same |
| `runtime/chat.go:1857` activation start | `busy`; failure releases the gate | same |
| `runtime/chat.go:1479` Stop | `done` | closes the writer |
| `runtime/chat.go:2709` `updateOwnedBusy` | `busy` | active turn, not stopped, not settled, no pending permission |
| `runtime/chat.go:2722` `applyTurnEndStatus` | `idle` or `error` | sets `settledTurnID` |
| `runtime/chat.go:2747` via `permission.go:83`, `host_permission.go:106` | `waiting_input` | status lock only |
| `runtime/chat.go:2256` crash | `error` | status lock only |
| `runtime/chat.go:2775` `republishLive` | none (context, background count) | row exists |
| `runtime/reconcile.go:52` `MarkProcessExited` | `done` | process exited, not a leaderless group |
| `runtime/terminal/terminal.go:596` | `idle` | no row yet |
| `runtime/terminal/terminal.go:639` | `done` | none |
| `state/manager.go:266` `ApplyStaleCorrection` | rewrites current state, clips detail | running row, nothing newer |
| `state/manager.go:459` `applyStatusHook` | whatever the hook reports | hook token matches |
| `server/switch.go:321` `failSwitch` | `error` | row exists |
| `server/launch.go:174` rollback, and cascade on agent delete | row removed | launch failed |

`busy` and `waiting_input` follow facts that exist only in memory; `error` is the only durable
record of "the last turn failed" besides the transcript.

## 3. Availability predicates (30 sites)

`runtime/registry.go`: `claimStart` (:263), `Owns` (:652), `ContinueSettledChat` (:347),
`StopGuardedIdle` (:557). `runtime/chat.go`: `claimTurnLocked` (:1035), `claimTurn` and
`claimTurnOrHold` (:979), `claimIdleStop` (:1050), `StartActivation` (:1765).
`runtime/terminal/terminal.go:462` (unsupported stub). `server/resume.go`: `claimLifecycle` (:47),
`wakeCandidate` (:462), `addressableAgents` (:489). `server/archive_gate.go`:
`projectArchiveGate` (:18), `acquireProjectStart` and `acquireAgentStart` (:38, :49).
`state/messages.go`: `stoppedWakeGates` (:45), `AddressableAgents` (:69).
`messaging/messaging.go:141`. `server/quota.go:103` `quotaBlocksAgent`.
`server/quota_dispatch.go:267` `quotaWorkEligible`. `state/quota_admission.go`:
`quotaWorkEligibleTx` (:19), `quotaUnassignedTx` (:74), `quotaTaskClaimTx` (:88), `AdmitQuotaTurn`
(:99), `AdmitQuotaTaskResume`. `state/quota_checks.go:48`. `server/messaging_loops.go:309`
`assignmentNeedsAdmission`. `server/task_dispatcher.go`: `planExistingAgentStart` (:200),
`startExistingAgentTask` (:351). `state/tasks.go:1450` `AdmitReadyTask`.
`server/think_tanks.go:220` `thinkTankIneligible`. `state/think_tank_turns.go:40`
`BeginThinkTankAttempt`. `state/think_tanks.go:1648`. `server/think_tank_wire.go:291–322`.
`server/pipeline_stall.go:20`. `server/queued_input.go:41–76`. `state/types.go:160, 180`.
`server/switch.go:502`. `server/keepawake.go:20`.

**The same question answered from different inputs.**

1. *Idle?* `status == idle` alone (`queued_input.go:53`, `pipeline_stall.go:29`,
   `think_tank_wire.go:322`); `idle or error` (`chat.go:1791`, `quota_dispatch.go:273`);
   `StartActivation` adds three in-memory flags.
2. *Quota-blocked?* `quotaBlocksAgent` reads the row; `StartActivation` inlines the same check and
   adds an in-memory flag; `ContinueSettledChat` checks only the flag.
3. *Running?* The registry map (`queued_input.go`) versus the `running` row (task planner,
   `quotaWorkEligible`, `think_tank_wire.go`). A row from a previous process passes the second and
   fails the first.
4. *Assigned elsewhere?* Four copies: `quotaUnassignedTx`; the unique index plus a room-attempt
   probe in `AdmitReadyTask`; a task count in `BeginThinkTankAttempt` (:95); `AssignedTask` in the
   room wire.
5. *May input be delivered now?* `claimTurnLocked` with the hold queue; `StartActivation` with
   status plus the gate; `ContinueSettledChat` adds open background tasks and the settled turn id.
6. *Busy?* `state == busy` (`switch.go:502`, `keepawake.go:20`); `CloneBusyState` also counts
   `waiting_input`.

## 4. Reconcilers (16)

| Function | Makes these agree | When |
|---|---|---|
| `ReconcileStale` via `pruneStaleRunning` | running row and OS process, then `status = done` | start, file events, every 30 s |
| `reconcileSessionsOnce`, `ApplyStaleCorrection` | status detail and transcript | start, every 30 s, file events |
| `Manager.recompute`, `Touch` | read-only join for the UI view | every touch |
| `RecoverQuotaClaims` | claimed quota intents and the dead process | startup |
| `repairQuotaNotices`, `ReconcileQuotaNotice` | quota notice and transcript | startup, every 2 s |
| `RecoverMailActivations` | activations and mail | startup |
| `recoverTasks` | task state and the live runtime | startup |
| `recoverStartingTasks` | `starting` tasks and their runtime claim | every 2 s |
| `reconcileTaskCleanup` | task cleanup retry state | every 2 s |
| `ReconcileTaskWaits` | task waits and work results | startup, every 2 s |
| `recoverMissedStageStalls` | stage task and owner agent | every 2 s |
| Pipeline `Startup`, `ReconcilePending` | run pending action | startup, every 2 s |
| `RecoverThinkTanks` | room attempts, members and the process | startup |
| `recoverAttachmentStorage` | attachment files and receipts | startup |
| `reconcileRemovingWorkspaces` | workspace state and the filesystem | startup |
| `resolveDueTimeArms`, `processDueSchedules` | arms, schedules and the clock | every 2 s |

## 5. In-memory claims held by the server

`hookTokens`, `mcpCleanups`, `switching`, `lifecycleBusy`, `startCheckouts`, `agentArchiving`,
`agentStartLeases`, `permissionTools`, `heldPermissions`, `peerStopRequesters`, `attachmentBusy`,
`quotaActions`, `quotaChecks`, `quotaCleanupTried`, `quotaChecked`, `quotaCleanupSpecs`,
`thinkTankCaptures` (`server/server.go:60–175`). All are empty after a restart.
