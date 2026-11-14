# Give agent execution one owner

**State:** In progress
**Why:** Operator request, 2026-11-14, after the
[adversarial assessment](../archive/reviews/adversarial-2026-11-14/ASSESSMENT.md) and its
[structural pass](../archive/reviews/adversarial-2026-11-14/ARCHITECTURE.md). Operator decision:
no product or feature change; no capability is removed.
**Relevant requirements:** TS-01.R45–R50, TS-02.R67–R68, TS-08.R168; extends
TS-01.R9, R16, R21, R42–R44; INV §1, §2, §4, §5, §9, §10, §16, §18, §20, §23

## Outcome

Chuck behaves as it does today, with fewer ways to get stuck and fewer copies of the same rule to
keep in agreement. A conversation turn begins and ends in one place and has a durable row. Status
is chosen by one function. A plane that wants an agent to work asks one function whether it may,
and one function brings an agent up. The defects listed in the
[plan](../plans/agent-execution-ownership.md#findings-this-plan-owns) stop occurring; nothing
else a person or an agent sees changes.

## Included work

- The seven slices of the [plan](../plans/agent-execution-ownership.md).
- Every finding from the 2026-11-14 assessment that needs no product decision, and CR-A10-1 to
  CR-A10-5. This change owns them; they are not queued for `/fix`.

Each plane keeps its own durable intent, claim tokens, attempt boundary, retry policy, periodic
pass and restart recovery, and the `activations` table is unchanged (TS-01.R19–R21). Terminal
agents, hooks, Think Tanks, schedules, quota checks and the quota snapshot stay exactly as they
are.

Not included, each recorded in `docs/ideas.md` for its own decision: one request protocol for
every plane; one server per home proved by a lock; what a run shows for its owner's quota
attention; gitignored files in checkout
deletion; the loopback Origin check; host settlement of assignment turns and the tool-protocol
changes around it; self-addressed mail; rollback messaging; provider conformance captures;
revisioned client resources; database durability and backup.

## How we will know it works

- Every existing feature and recovery test passes unchanged in both Go variants and the UI
  suite. A slice that needs to edit one stops and asks.
- Each consolidating function has a table test, written before any call site moves, with one row
  per site it replaces and today's result in each row. The rows come from
  [`INVENTORY-agent-state.md`](../archive/reviews/adversarial-2026-11-14/INVENTORY-agent-state.md)
  and
  [`INVENTORY-lifecycle.md`](../archive/reviews/adversarial-2026-11-14/INVENTORY-lifecycle.md),
  re-read against the code at the time.
- Each defect is reproduced through its real route before it is fixed, then kept as a
  regression. The archived probes are the starting point:
  [`docs/archive/reviews/adversarial-2026-11-14/*.go.txt`](../archive/reviews/adversarial-2026-11-14/ASSESSMENT.md).
- Each requirement's *Verify* clause and the plan's progress counts are asserted by tests.
- Fake-provider evidence only. The credentialed provider and device gates in
  `docs/features/ACCEPTANCE-GATES.md` stay owed and are not closed by this change.

## State and ownership

The one durable state this change adds is the state of the per-turn row (TS-02.R67): one row per
agent, owned by `internal/state`, replaced at the next begin. Nothing recovers from it and no
plane's restart rule reads it; its readers are status composition (TS-01.R46) and the admission
function (TS-01.R47). "The origin" is the task, mail, quota intent, room attempt or receipt that
asked for the turn; its own rows and rules are unchanged.

| State | Restart | Stop or cancel | Owner replaced | Input refused or failed | Quota or timeout hold | Late callback from previous owner |
|---|---|---|---|---|---|---|
| `running` | Startup reconciliation, which already marks an exited process, ends the row `process_ended` / `internal/state` / the person sees `done`, "process exited", as today. A live orphan process keeps its row and its recorded status, as today. Each origin's restart recovery runs as it does now and does not read the row. | Cancel: the runtime ends the turn → `ended (cancelled)`; status `idle`, as today. Stop: → `ended (process_ended)`; the transcript records no `turn_end` for a Stop, as today; status `done`. | Resume and Switch cancel the turn and stop the old process under the lifecycle claim, so the row is `ended` before the new generation exists. The new generation's first begin replaces it. | The provider call fails or refuses → `ended` with the resolved outcome; status `error`; the origin handles the failure as it does today. A begin that fails after the row was written ends the row in the same function, so no `running` row outlives a turn that never reached the provider. | The turn ends quota-blocked → `ended (quota_blocked)`; the quota intent is created as today and is the quota record. No turn watchdog exists and none is added. | The end function compares the generation and turn: an outcome or exit for another generation is ignored, and an old generation's teardown cannot end a newer generation's row. |
| `ended` | Stays `ended`; status composition reads its outcome for the live generation and reads it as no row for any other. | Stop sets status `done` through the lifecycle input; the row is unchanged. Cancel with no open turn does nothing, as today. | A new generation reads the old row as no row, so a resumed agent shows `idle` whatever its previous turn did, as today. Its first begin replaces the row. | The next begin replaces the row; the terminal hook's reset and the implicit budget row write it `ended` with no origin. A begin whose write fails refuses the turn on every path, as today: the person's Send is refused and its receipt rolled back, and a host-initiated turn does not start. | Not applicable: an ended row with a quota-blocked outcome stays until the next begin and gates nothing; the quota intent gates. | A second end for an ended row is ignored: the first end wins. |

## Design review

Three fresh-context rounds on 2026-11-14 (workflow §13). Round 1 found that a separate ledger
table and one request protocol for every plane changed behavior in six places; both were removed
and recorded in `docs/ideas.md`. Round 2 found four more one-sentence behavior changes (rename
answering with a conflict, a stale-status correction clearing `waiting_input`, a failed begin
write letting a Send through, an undefined run state for quota attention); each was removed. It
also showed that ADV-1 does not occur through the HTTP routes. Round 3 confirmed those and found
two visible changes in the single-instance lock (a detached start reporting success when its
port is taken, and an upgrade beside a server from an older version); the lock was removed and
recorded as a compatibility decision. No finding is open against what remains.

## Waiting on

Nothing.
