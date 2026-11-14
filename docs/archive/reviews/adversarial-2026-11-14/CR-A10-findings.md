# CR-A10 findings as recorded 2026-11-13

Moved out of the live handoff on 2026-11-14, when
[`give-agent-execution-one-owner.md`](../../../ready-changes/give-agent-execution-one-owner.md)
took ownership of CR-A10-1 to CR-A10-5 and replaced the transition-rule recommendation with
TS-01.R49 and R50. The text below is unchanged apart from its heading level.


Subsystem review of the complete implementation at `8faec36f` ([charter](CORE-RELIABILITY-REVIEW-A10.md)),
read and probed from a clean export of that commit. Dirty work at review time was another session's
uncommitted CR-A8 fix and `docs/ideas.md`; neither was read as evidence. "Probe" means a throwaway
test against the real store or test server, run and then deleted; "source" means read and traced,
not executed. No product code, test or specification changed. The level is set by CR-A10-1 and -2,
each a bounded change to durable concurrent state at an existing seam with a server-level test.
The transition-rule design below is separate work for `/design-feature`, not part of this fix run.

**Writer ledger.** "Lost race" is what a writer does when its guard matches no row.

| Status column | Where the legal moves are stated | Guard style | Lost race | Shared rule warranted |
|---|---|---|---|---|
| `quota_intents.state` | nowhere; 16 UPDATEs each carry their own from-set | mixed: revision, attempt token, turn tuple, or state only | 9 typed conflicts, 7 silent | yes, first (CR-A10-1, -4) |
| `tasks.state` | nowhere; 23 UPDATEs; the closed-run fence is a per-statement SQL fragment | SQL from-set, or a read in the same transaction | mostly "not mine" without error | yes, second (CR-A10-2) |
| `pipeline_runs.state` | nowhere; 15 callers of a revision-only compare-and-swap hold the from-set in Go | revision | typed conflict; 4 callers swallow it | yes, third (CR-A10-4, CR-A8-5) |
| `attachment_receipts.delivery`, `annotation_receipts.state` | nowhere; `SetAttachmentDelivery` and `SetAnnotationEffect` accept any value from any value | none in SQL; the runtime's per-agent lock and the upload claim serialize callers | silent | no: fold a forward-only guard into the PIPE-CHAT input-record change |
| `task_arms.state` | one from-state (`unsatisfied`), every writer inside its caller's transaction | SQL | harmless: the follow-on re-reads arm state | no |
| `activations.state` | claim → attempt → retire, token-fenced | SQL plus token | silent by design: a lost token means another claimant won | no |
| `think_tank_members.setup_state`, `think_tank_attempts.state` | room transactions beside `thinkTankTransition` | SQL or a read in the same transaction | conflict | no |

**Confirmed findings.**

- **Must fix** CR-A10-1 — probe, medium. A cancelled quota wait is re-armed as needs-attention by
  the person's next failed turn. `HoldQuotaManualTurn` (`internal/state/quota.go:487`) writes
  `needs_attention` from every state except `continued`, with no revision check and no rows
  check. Every ordinary prompt binds its turn to the stored intent
  (`internal/runtime/chat.go:1103`; `BindQuotaManualTurn` refuses only `continued`), Send then
  cancels the intent, and a turn that ends without success and without quota evidence reaches the
  hold through `settleQuotaTurn` (`internal/server/quota.go`). Probed on the test server: after
  Cancel continuation the intent was `cancelled` and waking mail was admitted; after the bind and
  a failed settlement it was `needs_attention` / `non_quota_failure`, `quotaBlocksAgent` was true
  and waking mail was no longer delivered. Task dispatch and the pipeline stall check use the same
  gate. The next successful turn or owner control clears it. Hold only from the states a
  continuation can still follow, compare the revision, and test the sequence at the server.
  TS-10.R43/R52, FS-01.R43, INV §5/§18.
- **Worth fixing** CR-A10-2 — probe for the state, source for the trigger, medium. A task that
  finishes while a continuation start is unconfirmed keeps its agent's exclusive slot.
  `RecordAgentTaskResult` (`internal/state/tasks.go:1829`) and `commitStageResultTx`
  (`internal/state/pipeline_tasks.go:413`) accept a report in `starting` and leave
  `continuation_pending`; only confirmation, interruption and `finishTask` clear it. The unique
  index `idx_tasks_active_assignee` counts `continuation_pending = 1` in any state. Probed: a
  waiting assignment continued, admitted, reported before `ConfirmTaskStart` and released ended
  `finished` with the flag set, and `AdmitReadyTask` then refused every later task for that agent
  with no error. The trigger is a report landing between admission and confirmation, normally a few
  store calls; not reproduced end to end. Finish through one helper that clears what `finishTask`
  clears, and bound the index to unfinished states. TS-10.R18/R19, INV §1/§2.
- **Worth fixing** CR-A10-3 — source, with CR-A9's probe of the launch window, easy. Rename can be
  silently undone. `handleRename` (`internal/server/sessions.go:979`) reads and rewrites the whole
  agent row without the lifecycle claim, while Launch, Resume and Switch write the whole row they
  composed before the provider handshake once it returns (`launch.go:180`, `resume.go:398`,
  `switch.go:245`/`:290`). A rename from the card menu or the phone during that handshake answers
  200 and is overwritten. `handleIdentity` already takes the claim. Share its claim, and make the
  post-handshake write update only the applied setting. INV §2/§3/§5.
- **Worth fixing** CR-A10-4 — source, easy. Three more writers report success after losing their
  compare-and-swap, the shape of CR-A8-5. `pauseStartupRun` (`internal/pipeline/manager.go:954`)
  runs without the run lock while the dispatcher is already started and returns nil, so a failed
  startup reconcile leaves no attention reason. `SyncRoomPhase` (`internal/pipeline/rooms.go:182`)
  returns nil, so a room hold is not projected onto its run until the next room event.
  `ResolveQuota` (`internal/state/quota_success.go`) is guarded by revision alone, commits on zero
  rows and returns nil. Re-read and decide again, or return the conflict. Occurrence needs a
  revision write in the same instant; not reproduced. INV §5/§18/§22.
- **Worth fixing** CR-A10-5 — source, trivial. Seven exported store writers have no production
  caller: `UpdateTaskCAS`, `ClosePipelineStageTask`, `ClearQuota`, `CreatePipelineAttemptCAS`,
  `UpdatePipelineAttemptAndRunCAS`, `MarkPipelineQuiescent` and `AcceptPipelineReport`. The first
  three are kept alive by tests only. `UpdateTaskCAS` accepts any caller-supplied state and is the
  one path that could leave `pending_yield` on a finished task, which `CompleteTaskYield` would
  then move to `ready`. The annotation value `rejected` is never written or read. Delete them with
  their tests. INV §2/§10.

**Leads refuted or already owned.** Lead 1: the missing hold release was CR-A9-5 and is fixed; the
rest is CR-A10-4. Lead 2: `pending_yield` is set only on a running task and every reachable
terminal writer refuses or clears it. Lead 3: a crash between confirmation and the standing-agent
bind is interrupted and requeued at startup; the swallowed bind stays with CR-A8-8. Lead 4: the two
closed-run rules do disagree about a deleted run, and deletion leaves `task_lineage` and
`pipeline_stage_tasks` rows, but stage release and Stop settle every member before a run can be
deleted. Leads 5 and 10: no reachable backward move; see the ledger. Lead 6: all three
`waiting → ready` writers carry the run fence and differ by wait kind. Lead 7: six `ready` writers
lack the fence, but admission has it and Stop cancels `ready` rows, so a stopped run's task cannot
start. Lead 8: CR-A10-1. Lead 9: CR-A10-2; the other column differences have no reader. Lead 11:
`RetryThinkTankJudge` inserts nothing; the next reservation adds the fresh judge and the failed one
stays by design. A restart leaves that row `pending` rather than `failed`, with no reader. Lead 12:
CR-A10-5.

**Not a defect, for a product decision.** Lead 13: the agent row stores the requested model and
effort. Claude and Codex launches fail on a rejected or contradicted setting, but an adapter that
answers success without echoing the value is taken as applied (`setConfigOption`,
`internal/runtime/chat.go:3585`, a recorded choice), and other backends are not confirmed at all.
INV §20 reads missing evidence as unknown; decide whether an unechoed model should show as
unconfirmed.

**Recommendation for `/design-feature`.** One rule per machine: a function from the current row
and a named event to the next state, the columns that state owns, and the loser's answer.

1. Quota intents first. The rule must name each event's from-set, which identity it compares
   (revision, attempt token or turn tuple), what `cancelled` may still become, and the side
   effects a transition owns: room hold release, runtime block and due time. It replaces three
   cancellation from-sets, the unguarded resolve and the hold in CR-A10-1.
2. Tasks second. The rule must make the closed-run fence part of every move into `ready`,
   `waiting` or `starting` rather than a fragment each statement remembers, and give each target
   state its column contract, starting with one `finished` helper for the four terminal writers.
3. Pipeline runs third. Keep the revision compare-and-swap, add a table of legal pairs checked
   under the run lock, and one conflict policy: re-read and decide again. Sequence it after the
   CR-A8 fix, which is changing these callers now.

**Invariant sweep.** Findings touch §1, §2, §3, §5, §10, §18 and §22; lead 13 is §20. Checked with
no finding: §7 (startup quota and task recovery handle each row in guarded statements), §15
(confirmation and bind heal at restart), §19 (receipt writers), §23 (start attempts, quota
episodes and activation tokens are compared on every late write) and §24. §4, §6, §8, §9, §11–§14,
§16, §17 and §21 have no surface in this scope; no test changed.
