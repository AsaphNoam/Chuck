# Inventory: how agent work is started and continued

Read-only trace at `0a8bda69`, 2026-11-14. Evidence for
[ARCHITECTURE.md](ARCHITECTURE.md) and the checklist for
[the ready change](../../../ready-changes/give-agent-execution-one-owner.md). Paths are under
`internal/` unless they start with `server/`, `state/`, `runtime/` or `pipeline/`, which are its
subdirectories. Line numbers were read once and not re-verified cell by cell; re-read before
relying on one. "Not traced" means exactly that.

## 1. Claim and attempt protocols today (eight)

1. **Lifecycle claim and archive lease** — in-memory exclusive claim plus counting lease.
   `server/resume.go:47–127`, `server/archive_gate.go`.
2. **Runtime turn gate** — `runtime/chat.go:965–1185` and `1765–1893`. Two claim sites:
   `claimTurnLocked` and `StartActivation`'s inline copy.
3. **Activation, mail** — pending → claimed → attempted → retired, released or discarded; at most
   once. `state/activations.go`, `server/messaging_loops.go`.
4. **Activation, dependency** — same states, re-created per start attempt, deleted at startup
   (`state/activations.go:378–447`).
5. **Task start** — admit → confirm, abandon, fail (3), park, interrupt. `state/tasks.go:1435–1630`,
   `server/task_dispatcher.go`.
6. **Quota** — claim → provider_started → notice → notice_confirmed → success_confirmed →
   continued, with defer, hold, cancel and a separate check allowance. `state/quota*.go`,
   `server/quota_dispatch.go`, `quota_check.go`, `quota_notice.go`, `quota_controls.go`.
7. **Pipeline pending-action controller** — failure cap 10 with backoff, stall count 3, recovery
   rounds 3. `pipeline/reconcile.go`, `continue_control.go`, `actions.go`, `manager.go:824–926`,
   `server/pipeline_stall.go`.
8. **Room attempt** — room claim, setup and judge reservation, attempt, finalize or fail, explicit
   retry. `state/think_tank_turns.go`, `server/think_tanks.go`, `think_tank_capture.go`.

Two input protocols sit over the turn gate: attachment-receipt delivery states
(`state/attachment_receipts.go`) and queued assignment input (`server/queued_input.go`).

## 2. Behaviour per origin today

This is the behaviour the restructuring must reproduce.

| Origin | Durable intent | Agent busy | Failure before the provider frame | Interrupted mid-turn (crash) | Final-failure surface | Process start |
|---|---|---|---|---|---|---|
| Person Send, held, Steer | Attachment receipt; the held message is memory only | Held (one) | Receipt `failed` or `released` | Hold lost; receipt `unknown` | Receipt | `wakeAgent` → `resumeSessionWithHooks` |
| Mail wake | Activation row committed with the mail | Released to pending | Before the attempt boundary (claim lost, validation): released to pending. The boundary is crossed in the resume pre-hook, before the process starts (`server/messaging_loops.go:241–266`), so a resume that fails afterwards is spent and not replayed. An opportunity whose mail was already delivered inline is retired without a turn (`runtime/chat.go:1833–1845`). | Attempted row deleted, never replayed | Debug log; mail stays unread | `resumeSessionWithHooks` |
| Task start, created runtime | Task row `ready → starting` | First prompt refused: runtime stopped, attempt charged | Abandon: free. Fail: 1 of 3, then park | `StopStage`, then `FailTaskStart` (charged) | `dependency_failed` attention | `launchAgent` |
| Task start, woke runtime | Same | Deferred, free | Same | Same | Same | `wakeForActivation` |
| Task start or continuation, borrowed runtime | Same; continuation sets `continuation_pending` | Deferred, free | Same | Interrupted | Same | none (`runActivationTurn`) |
| Quota continuation | `quota_intents` row | Deferred 2 s, reason `busy`, unbounded | Unstarted claim → `scheduled`; start failure → hold `provider_start_failed` | `needs_attention`, `uncertain_delivery` | Quota `needs_attention` | `resumeSessionWithHooks` under two global locks |
| Quota isolated check | Same row, check allowance (6) | — | — | `needs_attention`, `check_interrupted` | Same | `registry.RunQuotaCheck` (throwaway session) |
| Pipeline owner nudge | Stall count on the run | Silently skipped | — | — | Run pauses `owner_not_reporting` after 3 | none (`ContinueSettledChat`) |
| Room turn, setup, judge | Room state; attempt committed in the before-hook | Left for the next sweep | Start abandoned | Attempt failed "not replayed"; room paused | Room hold text | Turns: `resumeSessionWithHooks`; setup and judge: `launchClaimed` |
| Schedule occurrence | Inserts a `ready` task | as task | as task | as task | as task | as task |
| Launch, Resume, Switch, Clone | None | Clone refuses a busy source | HTTP error; Switch rolls back once | Left to the reconciliation sweep (not traced) | HTTP error; Switch sets `error` | own composition |

Start bounds per plane: activation 8, task 8 plus the concurrency budget, quota 8, room 4 with
opening slots 32 per room and 128 total. The task concurrency budget is enforced for task starts
and the quota task-resume only.

## 3. Cross-plane reads (22)

Each is one plane reading another plane's rows to decide whether it may use an agent.

1. mail → task: `server/messaging_loops.go:90`
2. mail → task: `server/messaging_loops.go:147`
3. mail → task: `assignmentNeedsAdmission`, `server/messaging_loops.go:309`, `server/resume.go:438`
4. mail → quota: `server/messaging_loops.go:117`
5. mail → lifecycle: `server/messaging_loops.go:132`
6. task → quota: `server/task_dispatcher.go:198`
7. task → room: `state/tasks.go:1454`
8. task → pipeline workspace: `server/task_dispatcher.go:373`
9. quota → task and room: `state/quota_admission.go:29–68`
10. quota → task and room: `state/quota_admission.go:74–86`
11. quota → task (write): `state/quota_admission.go:116–127`
12. quota → task (write): `state/quota_admission.go:172`
13. quota → room: `server/quota_dispatch.go:282`
14. quota → pipeline: `server/quota_controls.go:119–136`
15. room → task: `server/think_tanks.go:208`
16. room → task: `state/think_tank_turns.go:95`
17. room → quota: `state/think_tank_turns.go:140`
18. runtime → quota: `runtime/chat.go:1780`
19. runtime → quota and task (write): `state/quota.go:458–479`
20. Send → task and pipeline: `server/queued_input.go:17, 105`
21. nudge → pipeline, task, quota: `server/pipeline_stall.go:20, 32`; `runtime/registry.go:359`
22. restart task → quota; restart room → quota; quota → room hold release:
    `server/task_dispatcher.go:919, 1042`; `state/think_tank_turns.go:840, 867`; `state/quota.go:520`

Mail reads no room state and rooms read no mail state; those pairs rely on the turn gate alone.

## 4. Differences with no stated reason (15)

1. **Busy status write.** The prompt path ignores a failed write (`runtime/chat.go:1154`); the
   activation path treats it as fatal (`:1857–1863`). TS-01.R21 requires the commit first.
2. **Turn-gate claim.** `claimTurnLocked` resets `turn.resolved` (`runtime/chat.go:1041`);
   `StartActivation`'s inline claim does not (`:1816–1818`). Consequence not traced.
3. **Archive lease on a running agent.** Quota and room turns take `acquireAgentStart`
   (`server/quota_dispatch.go:223`, `server/think_tanks.go:289`); running-mail, borrowed task and
   the nudge do not (`server/messaging_loops.go:213–223`, `server/pipeline_stall.go:35`).
4. **Missing project definition.** Passes the resume and quota gate (`server/archive_gate.go:26`),
   holds a room (`server/think_tanks.go:234–236`), pauses a schedule (`server/task_time.go:64`),
   is rejected at launch (`server/launch.go:263–265`).
5. **Project archived while the agent runs.** Mail skips the check on purpose
   (`server/messaging_loops.go:154–165`, FS-01.R33); a borrowed task skips it with no stated
   reason (`server/task_dispatcher.go:395–431`); quota and rooms enforce it.
6. **"Assigned task" predicate.** Mail uses `TaskAssignment` (includes `ready` with
   `continuation_pending`, `state/tasks.go:1654`); rooms use `AssignedTask` (excludes it, `:1635`);
   quota uses `quotaUnassignedTx` (`state/quota_admission.go:76`).
7. **Agent busy.** See the table in section 2: eight different answers.
8. **A woken agent whose turn does not start.** Mail and dependency stop it
   (`server/resume.go:367`); a room leaves it idle (`server/think_tanks.go:337–342`); quota leaves
   it up and defers (`server/quota_dispatch.go:211–215, 260–262`).
9. **Failure after the attempt boundary.** Mail discards; a task retries to 3 then parks; quota
   holds for a person; a room needs explicit retry; a pipeline owner gets 3 nudges then 3 rounds.
10. **Crash with a turn in flight.** See the table in section 2.
11. **Quota gate.** `StartActivation` refuses any non-continued intent
    (`runtime/chat.go:1780–1789`); the prompt path only binds a manual turn (`:1110–1120`);
    `StartLaunchedTask` checks neither; `ContinueSettledChat` checks the in-memory flag
    (`runtime/registry.go:359`).
12. **Global locks across provider I/O.** Only quota holds `configMu` and `quotaControlMu`
    through `resumeSessionWithHooks` (`server/quota_dispatch.go:121–126, 248`).
13. **Start bounds are per plane**, and the concurrency budget covers tasks only.
14. **Pipeline room stage tasks** are inserted already `running`, bypassing task start
    (`state/pipeline_tasks.go:119`), and are skipped by restart recovery
    (`server/task_dispatcher.go:915`).
15. **TS-01.R16's "same wakeability gates".** Wakes use `StoppedWakeCandidates`
    (`state/messages.go:45–47`); explicit resume checks only archived and project
    (`server/resume.go:182–186`); quota and room resumes re-check their own subsets.

Items 5 (mail), 9 and 10 are deliberate where TS-01.R19 and the owning feature specifications say
so. The others have no cited reason.

## 5. Choke points

No function sees every turn start.

- `agentState.turn.turnActive` is the only state every runtime turn passes, set at two sites
  (`runtime/chat.go:1039, 1816`).
- `StartActivation` carries mail, existing-agent task, quota and room turns. Bypassed by a person's
  Send, the launched-task first prompt (`StartLaunchedTask`), the nudge (`ContinueSettledChat`) and
  Steer (`runtime/chat.go:817`).
- `runMessageTurnWithAssignment` carries Send, the launched-task prompt, the nudge and the held
  successor. Bypassed by every activation, which has its own provider goroutine
  (`runtime/chat.go:1874–1891`).
- The lifecycle claim is skipped by a person's Send and by queued-input delivery.
