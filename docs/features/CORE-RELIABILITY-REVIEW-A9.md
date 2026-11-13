# CR-A9 — Think Tank lifecycle and its quota join

Prepared 2026-11-13 from one focused Sonnet mapping pass and parent integration against
`e8a8a128`. This extends the [core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md) with one
integrated review for a frontier reviewer. The mapping read source and test names; it ran no
probes, tests or repairs, and the parent checked only the leads marked *(checked)*.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Is every Think Tank room phase, turn attempt and member launch owned by exactly one party and
fenced against concurrent lifecycle actions, and does a room reach a state a person can act on
after a quota stop, a crash or a server restart?

Trace setup claim → member or judge launch → turn admission → attempt settlement → requested
Pause/End boundary → judge publication → deletion, and repeat the trace with Stop, Archive, a
quota refusal and a restart injected at each step. A room that only a database edit can move is a
defect; so is a launch that another lifecycle action can half undo.

This review owns `internal/server/think_tank*.go`, `internal/state/{think_tanks,think_tank_turns}.go`,
`internal/messaging/think_tank_tools.go`, and the room branches of
`internal/state/{quota_owners,quota_success}.go` and `internal/server/quota_dispatch.go`. ASR-F
reviewed rooms together with pipelines on 2026-10-19, thirteen days after rooms were written; its
two findings were pipeline ones.

## Read first

The handoff's Current position and Active change and this charter. Use workflow §§3–7 with this
subsystem scope in place of a commit-diff scope.

- FS-21.R17–R18, R20, R31–R32, R34, R36–R37, R40, R48, R50–R51, R53 and their acceptance items.
- TS-14.R2, R4–R6, R13–R14, R19–R21, R23; TS-10.R38–R47 and R52 for quota ownership and room
  holds; TS-01.R16 and R41 for the lifecycle claim and launch fencing.
- FS-01.R40–R45 for quota continuation as a person sees it.
- INV trigger index, then §4, §5, §15, §18, §20, §23, §24.

## Starting authority map

Verify against the selected HEAD. All room transitions are SQLite transactions on one connection,
so a read-modify-write inside one transaction is atomic; separate store calls are not.

| Fact | Authority | Fence |
|---|---|---|
| Room progression | `think_tanks.phase` / `control` / `hold`, settled by `thinkTankTransition` | room claim for the sweep only; controls and exit handlers do not take it |
| Member setup | `think_tank_members.setup_state` with `launch_generation` | `ClaimThinkTankMemberSetup`; no per-agent lifecycle claim |
| Turn attempt | `think_tank_attempts`, keyed by agent, generation and turn id | revision compare-and-swap, or an opportunity recheck for openings |
| Requested Pause/End | `control`, applied at the attempt boundary by `applyThinkTankBoundaryTx` | only on the finish path |
| Quota stop of a room turn | `quota_intents` plus `hold='Quota reached'` | `ObserveOwnedQuota` fails the attempt directly |
| Restart | `RecoverThinkTanks`, before the listener starts | converts `running` and `pause_requested` only |

## Leads to settle

Each is a question for the review, not a finding. None has a test.

1. *(checked: no claim on the path; `Registry.Stop` deletes the start sentinel and generation)*
   `launchThinkTankSetup` and `launchThinkTankJudge` reach `launchAgent` without `claimLifecycle`,
   unlike manual launch, Clone and the task dispatcher. Does Stop during the handshake remove the
   in-flight registration, report stopped, and leave a live agent with no tools and an empty
   generation that the sweep retries every five seconds with no hold or attention?
2. In the same window, is an Archive reversed and a rename lost when the launch's second
   `WriteAgent` upserts the whole row?
3. Does shutdown quiescence wait for an in-flight room launch, given that only `claimLifecycle`
   joins the wait group?
4. *(checked: the room branch of `ObserveOwnedQuota` neither settles nor applies the boundary)*
   If End or Pause is requested during a turn that then hits quota, does `control` stay
   `end_requested` or `pause_requested` with no active attempt, with End a no-op, Resume and
   Delete refusing, and restart recovery leaving `end_requested` in place?
5. `ResolveQuota` writes `continued` guarded by revision only and does not release the room hold;
   `ReleaseQuotaRoomHold` has no production caller. Can `hold='Quota reached'` outlive its intent?
6. A crash between the first `WriteAgent` and the end of launch leaves an agent row with no
   session. Does setup then mark the member ready and never relaunch it?
7. After restart, a `launching` slot becomes `failed`. Does Resume clear the hold but leave the
   room in setup with no progress until Retry setup, and does anything tell the person?
8. `RecoverThinkTanks` does not publish and `SyncRoomPhase` has one caller. Does a restart-held
   pipeline room pause or flag its run?
9. Failures of `MarkThinkTankMemberSetup` and `MarkThinkTankJudgeLaunched` after a successful
   launch are logged at Debug. What moves a slot out of `launching` in that process?
10. `ClosePipelineThinkTank` re-runs the end transition on a room that already ended another way.
    Does it rewrite `ended_at` and clear a hold?
11. Standalone rooms raise no attention beyond their own update event. Which holds can only a
    person clear, and how would that person learn of them away from the room page?

12. `handleRename` takes no lifecycle claim at all. Beyond the launch window, what else can it
    race?
13. `MarkThinkTankMemberSetup` sets a hold on failure with no phase check. Can a setup failure
    after End put a hold on an ended room, and does that block the judge?
14. An unreadable judge config is logged at Debug and leaves the judge `ready`. Is it retried
    every five seconds indefinitely?

Weaker suspicions from the map, needing a probe: Stop between the running-row write and
`commitStart` leaving the registry holding a dead runtime; a Resume or mail wake in the
milliseconds between the session row and the running row tearing down the launch's registration;
hold writes whose own failure is swallowed.

The writer inventory for the room and quota tables is in
[HOTSPOT-REVIEW-LEADS-2026-11-13.md](HOTSPOT-REVIEW-LEADS-2026-11-13.md).

## Reuse these proofs before adding any

`internal/server/{think_tank_exit_test,think_tank_admission_test,think_tank_capture_test,
think_tanks_test,pipeline_rooms_test}.go`, `internal/state/{think_tanks_test,
think_tank_admission_test,think_tank_openings_test,quota_room_test}.go`,
`internal/pipeline/rooms_test.go`. The only pause/end quota test acts after the attempt has
already failed, and recovery is tested at the store level only.

## Joins and finding ownership

- ASRC-4 fenced the registration window for manual launch and Clone with `launchClaimed`; lead 1
  is the same shape on an entry point that fix did not cover. ASW1-4 and ASRG-1 own failed room
  capture and room-start compensation; their siblings looked handled in mapping.
- CR-A2-8 fixed room Resume after a quota stop. Leads 4 and 5 are a different interleaving.
- CR-A8 owns pipeline pause and answer; this review owns the room side of a room-backed stage up
  to accepted output.
- The ready change `share-resume-start-and-teardown-in-switch.md` excludes the room launch path
  and defers it to a check like this one.

No new room engine, scheduler or attention framework is proposed. A structural remedy is a
separately designed work item; this review fixes nothing.

## Required review result

Record the reviewed HEAD and relevant dirty work. Produce one ledger of room states, the fence
each transition holds, and the outcome of Stop, Archive, quota and restart at each step. Separate
confirmed findings from hypotheses and from live-provider evidence still owed.

Record findings under **CR-A9** in the handoff using workflow §7, with one Fix model
recommendation for the unit. Use isolated temporary homes and loopback-only servers; change no
product code or normative specs.
