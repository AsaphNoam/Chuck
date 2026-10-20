# CR-A6 — Operator action and observation truth

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`4b8c1265`. This maps **A6 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, current callers and existing test assertions;
it performed no audit, failure probes, test execution or product repairs. Concurrent ASW-1 fixes
overlap these seams; refresh their live ledger and dirty work before starting the review.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status.

## Question and responsibility

Can a person or script trust what Chuck says after an action: which target accepted which effect,
what actually completed, and what remains unknown or recoverable? After a lost response, stale
view, partial bulk result or reconnect, can they recover current truth without repeating work or
acting on a different target?

Trace explicit intent/bytes/target → required consent and revision → existing domain mutation →
response/receipt → authoritative state → CLI/dashboard/phone observation → next intervention.
Prioritize false success, lost recoverability, wrong-target decisions, hidden permission waits and
stale observations that cause repeated or unsafe development work. Cosmetic glitches are outside
scope unless they conceal an action's real state or make required recovery unreachable.

A6 owns the human/script control-and-observation join. Inspect the actual CLI and allowed browser/
phone paths into shared server/runtime/state services. These callers retain different authority:
local operator CLI parity does not imply agent-MCP or paired-phone parity. Follow agent producers
only where needed to establish a specific shared action or host approval boundary.

## Read first

Read the current handoff's Current position and Active change, this charter and related findings.
Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a commit-diff
scope. Read requirements before following source:

- FS-22.R1–R12/A1–A9 and TS-03.R64–R69/TS-06.R34: exact CLI scope, input/output, exits, layered
  consent, observation, ambiguity and independent wire/process evidence. Existing broad parity
  coverage is a starting proof inventory, not an instruction to repeat every command.
- FS-01.R6–R7/R10/R20–R21/R33–R34/R43/R52/R54 and the selected operation's requirements:
  Stop versus Cancel, wake/Resume, current execution, late background tails and quota targets.
  FS-03's Send/Steer/hold, permission, attachment and history contracts (including R81–R86/R99)
  govern what an accepted conversation action promises.
- TS-03.R3/R7–R9/R60/R62–R63/R70–R73: error/commit/publication boundaries, explicit cancellation,
  owned inputs, permission modes/host approval and truthful history/applied-switch state. R74's
  stronger input/withdrawal receipt extension remains planned and belongs to existing ASF-2 work.
- FS-02.R3/R10/R15–R21 and related notification/current-state rules; FS-20.R11–R17/R23–R29 and
  R33–R41/A22–A24; TS-13.R27–R29: current phone agent/run controls, action eligibility, stale-state
  recovery and actual-device evidence. Later rules remove phone task screens/controls; do not
  restore older parity promises. Preserve planned/unproved notes rather than claiming browser
  simulation proves OS suspension behavior or demanding unshipped remote operations.
- TS-01.R8/R16/R39/R41 and matching domain revision/receipt requirements define live identity,
  settlement and projection. Read the invariant trigger index and matching bodies, likely
  §§1/2/5/7/8/9/10/11/14/15/16/17. TS-06.R35 governs proportional evidence; adequate existing
  proof can justify no new test.

The feature owning each operation defines its effect; an HTTP status or CLI exit code does not
create a universal completion contract. Specs, not this charter or a mirrored command table,
govern expected behavior.

## Preserve the action and caller distinctions

- CLI exit `0` means a successful read or accepted action under that command's contract, not model
  completion. Reading a stored failed task/run can succeed. A bulk response with any failed member
  prints all member outcomes and exits nonzero even when HTTP succeeded.
- Send distinguishes `sent` from `held`; Steer distinguishes current-turn steering from `new_turn`.
  A resolved permission identifies the exact pending call, while peer Stop may still await its
  guarded effect/result. A background Stop `202` acknowledges a request, not terminal completion.
  Idle Cancel can report no cancelled turn. Stop and quota-cancellation have different targets.
- `--yes` acknowledges a CLI consequence. It cannot supply checkout/dirty/workspace consent,
  federation preview authority, a newer revision/ETag/limit or a separate host approval. Preserve
  the identifiers required by each action; stale caller guards fail instead of being refreshed and
  retried behind the operator's back.
- A lost/invalid/truncated mutation response can leave an applied effect. Do not infer rollback or
  auto-replay. Only operations with existing receipts support an explicit identical retry under
  their declared IDs/payload rules. An applied Switch with failed history recording must report
  the active target and refresh current state, not claim the old runtime necessarily survived.
- Local operator commands use named existing HTTP/domain operations. FS-22.R12 excludes raw MCP/
  hook producers, agent-token and paired-phone authority, credential APIs, context-grant minting,
  browser layout and planned `chuck action` transport. A6 does not reopen the full CLI inventory
  or general remote-security review. A shared effect must still honor its caller's actual scope.
- CLI status combines several bounded reads; it is not an atomic cross-domain snapshot. Unknown
  values, an unavailable hold read or incomplete permission history cannot become a confident
  zero/empty/healthy claim. CLI interruption does not create a browser draft or cancel agent work.

## Starting authority and lifetime map

Verify this map against the selected HEAD; a visible label or successful transport is not the oracle.

| Fact | Authority / producer | Observation and action boundary |
|---|---|---|
| Intended target and guards | Explicit stable agent/task/run/room/project IDs; project-scoped group label; action-owned pending call/episode/revision/ETag/attempt/request ID | Preserve the exact target through parsing, request, response and late callback. Display names and a newly fetched guard cannot silently replace caller intent. |
| Accepted/applied action | Existing domain transaction or runtime/lifecycle/turn claim and receipt | The response describes that operation's actual boundary. Later failure can coexist with an earlier applied step; multi-step commands are not implicitly transactional. |
| Executing work and pending permission | Live runtime/permission broker and captured execution identity, with separately owned domain state | Persisted status/history helps observation but cannot substitute for actual pending decision/effect authority. Old or resolved controls must not authorize a different call/turn. |
| Held/recoverable input | Live runtime hold; durable attachment/receipt recovery only where specified | Ordinary held text is not universally durable. A missing/stopped runtime read differs from an empty available hold; exact withdrawn/recoverable input belongs to its own operation. |
| Retained conversation | Canonical normalized transcript and per-agent sequence | Follow/refill/permission summaries distinguish durable events from live-only reasoning. Global SSE IDs cannot be used as transcript cursors or cross-agent completion proof. |
| Hydrated/derived current state | Server snapshot, domain revisions, status and bounded transcript/permission summaries | Browser stores, CLI status and phone Home are projections with explicit limits. Reconciliation repairs derived state; it is not provider history recovery or a new action acknowledgement. |
| Bulk/partial outcome | Exact member/action result and any durable stopped/archived state or compensation error | An outer 2xx cannot hide member failure or imply prior Stop was rolled back. Preserve all outcomes so an operator need not repeat successful members blindly. |
| Output/file publication | CLI stdout/stderr/exit or complete explicit destination file | Machine JSON preserves server fields/shapes; human output escapes terminal controls. A partial file, error text on stdout or swallowed write failure cannot masquerade as complete usable output. |
| Connection/freshness | Per-connection hydration, foreground health and current target/request identity | Reconnect/target change resets connection-derived state; cached phone state gates mutations while stale. A notification opens current truth rather than replaying its old action. |

## Shipped paths to trace

| Path | Current source entry points and question |
|---|---|
| CLI input/target/consent → HTTP → result | `internal/cli/{root,operator_client,operator_output,operator_agent,operator_attachments,operator_management,operator_workflows}.go`; legacy launch/resume/pipeline entry points | Follow real Cobra parsing, escaped target paths, exact bytes/headers, domain calls, JSON/human output and exit. Use registered routes and selected capability requirements as an independent oracle. |
| Conversation intervention → live/durable effect | `internal/server/{routes,sessions,attachments,archive_actions,groups}.go`, `internal/runtime/{chat,registry,permission,host_permission}.go` | Trace Send/hold/withdraw/Steer, permission, Cancel/Stop and partial group/archive actions through their actual effect and observed state. |
| Task/run/room/config control | `internal/server/{task_handlers,pipeline_handlers,think_tank_handlers,config_handlers,config_sources}.go` and owning state services | Preserve caller revisions/receipts/consents and existing owner gates. A2/A5 own eligibility/results; here establish action/response/current-state truth. |
| CLI status and transcript follow | `internal/cli/{operator_agent_status,operator_follow}.go`, `internal/server/{sse,sessions,transcript_window}.go` | Check exact-agent hydration and permission coverage; subscribe before authoritative read, catch up on gaps/pings/reconnect and close the observer on Ctrl-C/EPIPE without mutating work. |
| Desktop snapshot/live/recovery | `internal/server/{sse,reconcile,transcript_window}.go`, `ui/src/api/{client,sse}.ts`, `ui/src/store/{agentStore,transcriptStore}.ts` | Follow snapshot-to-live handoff, dropped final event, unknown/gapped revisions, reload, target change and post-error refetch through the actual consumer. |
| Phone foreground/control path | `internal/server/{remote_routes,remote_home}.go`, `ui/src/remote/{connection,api}.ts`, `ui/src/remote/AgentScreen.tsx`, `ui/src/remote/RunScreen.tsx`, `ui/src/remote/HomeScreen.tsx` | Use actual allowlisted routes/classifiers, connection/target fencing and stale mutation gates. OS/device evidence remains separate from mocked visibility/network events. |
| Peer Stop decision → later effect/result | `internal/server/peer_stop.go`, `internal/messaging/peer_tools.go`, existing operator permission route | Agent request, human decision and exact-target Stop result are distinct. Native auto-approval or an operator acknowledgement cannot stand in for this host decision. |

CLI follow reconciles by per-agent transcript sequence after hydration and on gaps/pings; it does
not assume `Last-Event-ID` replay. A dropped final event needs recovery even without a later event
to reveal the gap. CLI status reads session detail, snapshot, held input and a 500-event permission
window; its explicit coverage limits must survive formatting. Phone window/health rules are
different; do not infer one consumer's consistency guarantees from another's helper tests.

The bounded CLI permission read supplements in-window events with one newest unresolved recorded
`pending_permission` summary. That can preserve a known out-of-window request while
`permissions.complete=false`; it does not enumerate every older pending request. The live runtime
broker remains the action authority, not a historical row or the completeness of this projection.

## Connected searches and useful scenarios

These are facets of one review, not separately queued checks or a required Cartesian test matrix.

| Scenario / search | Evidence that distinguishes truthful control from apparent success |
|---|---|
| Explicit multiline/Unicode/file input → mutation → machine/human result | Compare caller bytes, actual request/target and server effect; reject conflicting/invalid input before mutation. Preserve trailing newline and unknown JSON fields where promised without mixing diagnostics into success output or interpreting terminal escapes as control. |
| Accepted mutation loses its response while another client changes state | Compare committed receipt/revision/native effect count with the caller's `result_unknown`/retry instructions and subsequent exact-target read. No hidden replay, guessed rollback or fetch-new-guard-and-resubmit; use receipt-specific identical retry only where supported. |
| Busy Send → held/replaced/withdrawn input or ambiguous Steer, then reload/interruption | Compare exact input/target/receipt, current runtime hold, durable recoverable copy and draft ownership. Preserve unknown consumption and do not auto-send recovered content. ASF-2/ASW-1 retain their known mechanisms. |
| Permission or peer Stop waits while history is degraded, window clipped or another actor resolves it | Compare live broker identity/options with canonical summaries, reachable current controls and actual native/host effect. Stale/child/prefix-colliding call IDs cannot resolve another request; a permission relay response cannot claim peer Stop completed. ASW1-2 remains its existing owner while open. |
| Cancel, Stop or quota Cancel races turn settlement and a later assignment/episode | Compare exact turn/generation or required episode/revision, guarded effect, actual process/turn state and successor. Refusal/unknown ownership cannot look stopped; ordinary Cancel cannot consume a different quota continuation. Reuse current ASF-1/ASW1-9 evidence. |
| Group/project action has successes, contention and a persistence/compensation failure | Compare all member outcomes, durable state, HTTP envelope, CLI stdout/stderr/exit and dashboard refetch. Expose stopped-but-not-archived members; an outer success cannot erase failures or make a partially applied action look rolled back. |
| Snapshot/live race, missed final event, reconnect/overflow or agent navigation | Compare canonical event sequence and current-domain state with each consumer after catch-up. No missing durable tail, duplicate event, stale pending control or callback applied to the newly selected agent. Live-only reasoning cannot become retained CLI history. |
| Download/follow is interrupted or piped consumer exits | Observe complete destination publication/permissions/overwrite rules, stream/exit and cleanup. No partial destination presented as complete, no hidden successful file output after failure, and no agent Cancel/Stop caused by closing an observer. |
| Phone suspends/offlines, returns with old data, then an action/notification targets changed state | Compare foreground health, hydration/revision, current pending target and mutation gating through the allowed route. No queued mutation replay or privileged fallback on revoke/disable/refusal. Pursue this for control/recovery impact, not device polish. |
| Switch or multi-step management applies before a later error | Compare the actual active target or earlier committed step with error wording and refreshed state. A failed history marker, annotation-after-launch or run-after-save step cannot imply the earlier effect never occurred or silently recreate it. |

Choose a few connected disposable-home journeys, such as a real command → accepted-but-lost
response → cross-client inspection/intervention, a mixed bulk result, and a durable follow/phone
recovery boundary. Reuse established wire/process coverage. Add evidence only for a distinct
unproved failure mechanism; do not repeat the entire operator inventory or every UI/control pair.

## Reuse these proofs before adding any

These tests were researched, not run or certified. Check actual assertions before reuse.

| Seam | Existing starting proof and its limit |
|---|---|
| Explicit bytes/input | `internal/cli/operator_client_test.go`: `TestOperatorTextPreservesBytesAndRejectsConflictingSources`; `internal/cli/operator_contract_test.go`: `TestOperatorInputLimitsAndObjectShape`. Helper input proof is not OS-stdin-to-server effect proof. |
| Ambiguous mutation and mixed output | `internal/cli/operator_contract_test.go`: `TestOperatorMutationUncertaintyDoesNotReplay`, `TestOperatorErrorAndOutputContracts`, `TestOperatorHumanReceiptsAndStoredFailures`. Transport/synthetic result fixtures do not join a real committed lost response or real mixed group handler to CLI exit. |
| Layered consent/guards | `internal/cli/operator_management_test.go`: `TestOperatorManagementInvalidInputMakesNoRequest`, `TestOperatorBackendReplaceForwardsETagAndUnknownFields`; `internal/cli/operator_workflows_test.go`: `TestOperatorWorkflowInputAndConsentFailBeforeRequest`. Wire/no-request proof does not establish every server-side race. |
| Status target/permission limits | `internal/cli/operator_agent_status_test.go`: `TestOperatorAgentStatusRejectsWrongSessionAndInvalidInputWithoutRequests`, `TestOperatorStatusIncludesCanonicalPendingPermissionOutsidePartialWindow`, `TestOperatorStatusPermissionResolutionUsesExactToolCallID`. These do not make the component reads atomic or prove unsaved live controls reachable. |
| Durable follow and observer exit | `internal/cli/operator_follow_test.go`: `TestOperatorFollowE2ESubscribeBeforeReadDeduplicatesCatchup`, `TestOperatorFollowPingRecoversDroppedFinalTail`, `TestOperatorFollowE2EReconnectsAndResumesCursor`, `TestOperatorFollowE2EEpipeStopsCleanly`. Controlled HTTP streams do not prove simultaneous real-server persistence/network/device behavior. |
| Owned download and built process | `internal/cli/operator_agent_test.go`: `TestOperatorMultipartAndOwnedDownload`; `internal/cli/operator_process_test.go`: `TestOperatorBuiltBinaryRealServerJourney`. Strong built CLI/real server/fake ACP evidence includes explicit multipart retry, but download success/mode/overwrite assertions alone do not prove failure cleanup or live-provider adoption. |
| Live control/remote boundaries | `internal/server/wake_test.go`: `TestStopDuringWakeConflictsAndKeepsRegistration`; `internal/server/peer_stop_test.go`: `TestPeerStopRefusesStaleOrSettledTargets`; `internal/server/remote_routes_test.go`: `TestRemoteStreamFilteringRevokeAndDisable`. These prove selected guards, not every cross-client stale action or native OS suspension cause. |
| Actual browser consumers | `ui/src/api/sse.test.ts`, `ui/src/store/agentStore.test.ts`, `ui/src/store/transcriptStore.test.ts`, `ui/src/remote/connection.test.ts`, `ui/src/remote/AgentScreen.test.tsx` | Inspect reducer/connection/target-fence assertions. Mocked time/visibility and rendering do not close paired-device or genuine suspension gates. |

The main questions are the actual mutation-to-observation join under concurrency/ambiguity,
real partial outcomes through CLI formatting, and current recovery after missing final updates.
Existing extensive CLI evidence narrows this work; a cleared prior parity review is not reopened
by this charter. Expected answers come from domain contracts and real producers, not command
tables copied into tests. Preserve FS-20.A22–A24 and existing provider/browser/phone gates; fake
peers and synthetic reconnects cannot certify them.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md) and live handoff, including concurrent
ASW-1 closure rather than historical header counts. Reuse ASR-H/C/G and
[original joins J2/J4/J6](AGENT-SURFACE-REVIEW.md#mandatory-joins-between-reviews).

- ASF-1 owns status/turn/permission/cancellation settlement; ASF-2 owns uncertain input, annotation
  receipts and draft/withdrawal recovery; ASF-6 owns CLI archive/automatic-resume selection.
  Corroborate these or report a distinct current mechanism, not another owner for an old trigger.
- ASW-1 owns history/control recovery, including unreachable unsaved approvals and stale terminal
  cleanup while those findings remain open. ASW-2/3 are closed for native contract/process proof.
  A missing live receipt or historical failing overlay is not a new demonstrated action defect.
- A1 owns durable truth/repair; A2 admission/wake/retry; A3 execution environment; A4 content/knowledge;
  A5 accepted result/cleanup/progression authority. A6 owns what operator paths actually request,
  acknowledge and reconstruct at those seams, without replacing the underlying review owners.
- A7 will own sustained capacity/cost. Follow adjacent code when it breaks A6's control promise,
  but do not expand into cosmetic polish, full remote security or general performance work.

Do not add universal command idempotency, another state store/event log, a new generic action API,
automatic retry/guard refresh or a redesigned UI. A valuable architectural remedy found by review
is a separately designed work item, not implementation during this audit.

## Required review result

Record exact reviewed HEAD, relevant dirty work and provider/adapter versions where needed. Produce
one concise ledger of caller/target authority, acceptance/completion/uncertainty boundaries, real
wire/effect/output evidence and reconstructed state. Separate coverage limits/unknowns and owed
native/device evidence from confirmed findings.

Record real findings under **CR-A6** in the handoff using workflow §7: current location, credible
trigger, development impact, requirement/invariant and reproducible evidence or precise source-backed
mechanism. Give the unit one Fix model recommendation based on its hardest open fix. Intentional
scope limits, explicit unknowns or accepted-action receipts are not defects merely because a more
atomic or synchronous product could report stronger completion.

Use isolated temporary Chuck/native homes and loopback-only servers; preserve user data and dirty
work. Follow the review skill's allowed reporting rules; change no product code or normative specs.
Close only this unit, retaining findings and honest evidence limits. Do not run other audit units,
fixes or a release architectural-review receipt as part of A6.
