# Chuck agent-surface review plan

Prepared 2026-10-19; consolidated from 25 boundary-level items into nine integrated reviews.
This is a review backlog, not specifications, review findings, or evidence of product correctness.
[HANDOFF.md](HANDOFF.md#review-queue) owns live status. No substantive review has been performed.

## Review shape

These reviews are sized for a frontier reviewer (Opus, Sol or Astra) to reason about several
interacting mechanisms together. The unit of review is an end-to-end contract or state machine,
not a source file, helper, protocol layer or individual checklist. A reviewer may delegate bounded
discovery, but must integrate the connected behavior and conflicting evidence personally.

The original research remains in the [search catalogue](AGENT-SURFACE-SEARCH-CATALOGUE.md).
Its 25 topics preserve specific source/spec entry points and failure hypotheses; they are neither
25 queued jobs nor 25 miniature reviews to perform in sequence. Each integrated charter below
supersedes those topics' former ownership boundaries. Source files may be examined by more than one
reviewer for different system questions. Finding a bug in an adjacent component is still in scope
when it breaks the selected charter's contract.

Codex chat and Claude chat/terminal are in scope, including instructions, native configuration and
capabilities, process behavior, context, tools, supervision, recovery and performance. Other backends
matter only where shared code affects these providers or eligibility must refuse an unsupported
operation. This is not a review of provider internals or unrelated visual/product features.

## Queue and retained coverage

| Integrated review | Responsibility | Research topics retained inside it |
|---|---|---|
| [ASR-A](#asr-a--instructions-configuration-and-execution-environment) | Instructions, configuration and execution environment | [ASR-01](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-01--instruction-composition-personas-and-operating-knowledge), [ASR-02](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-02--effective-configuration-federation-and-frozen-launch-values), [ASR-25](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-25--project-execution-context-worktrees-and-shared-resources) |
| [ASR-B](#asr-b--codex-and-claude-native-integration-fidelity) | Codex and Claude native integration fidelity | [ASR-03](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-03--provider-selection-packaging-authentication-and-model-discovery), [ASR-04](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-04--codex-native-adapter-and-instructionconfiguration-delivery), [ASR-05](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-05--claude-native-adapter-and-instructionconfiguration-delivery), [ASR-21](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-21--claude-interactive-terminal-pty-and-external-drivers) |
| [ASR-C](#asr-c--runtime-execution-concurrency-and-control) | Runtime execution, concurrency and control | [ASR-06](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-06--fresh-launch-process-registration-crash-and-restart-recovery), [ASR-08](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-08--message-submission-held-input-steer-and-turn-ownership), [ASR-09](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-09--stop-cancellation-process-containment-and-teardown), [ASR-10](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-10--acp-transport-callbacks-locking-and-backpressure), [ASR-11](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-11--permissions-session-modes-and-human-decisions), [ASR-12](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-12--native-events-tools-child-agents-background-activity-and-context-usage) |
| [ASR-D](#asr-d--conversation-input-and-history-fidelity) | Conversation, input and history fidelity | [ASR-07](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-07--resume-clone-fork-switch-and-native-history-continuity), [ASR-13](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-13--transcript-durability-replay-indexing-and-stream-continuity), [ASR-14](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-14--submitted-text-mentions-files-images-and-attachment-bytes), [ASR-15](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-15--authorized-context-peer-reads-and-annotations) |
| [ASR-E](#asr-e--agent-tools-communication-and-durable-delegation) | Agent tools, communication and durable delegation | [ASR-16](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-16--agent-facing-mcp-hooks-caller-authority-and-result-contracts), [ASR-17](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-17--peer-mail-deferred-delivery-waking-and-message-budgets), [ASR-18](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-18--durable-tasks-waits-reports-and-chucky-delegation) |
| [ASR-F](#asr-f--pipelines-think-tanks-and-orchestration) | Pipelines, Think Tanks and orchestration | [ASR-19](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-19--pipelines-and-standing-orchestration), [ASR-20](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-20--think-tanks-room-turns-private-input-and-judge-results) |
| [ASR-G](#asr-g--failure-quota-and-continuation-recovery) | Failure, quota and continuation recovery | [ASR-22](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-22--provider-failures-quota-admission-and-recovery) |
| [ASR-H](#asr-h--human-intent-across-desktop-phone-cli-and-http) | Human intent across desktop, phone, CLI and HTTP | [ASR-23](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-23--operator-entry-point-parity-and-recovery) |
| [ASR-I](#asr-i--whole-system-synthesis-and-agent-performance) | Whole-system synthesis and agent performance | [ASR-24](AGENT-SURFACE-SEARCH-CATALOGUE.md#checklist-asr-24--diagnostic-sufficiency-and-measured-agent-performance) |

Every original topic has exactly one primary group; all 25 remain covered. The legacy numeric IDs
are aliases: requesting `/review ASR-01`, for example, selects the whole ASR-A review. New invocations
should use the lettered ID. None of the original topics has been completed or dropped.

## Shared review method

Read the handoff's Current position and Active change, the selected charter, its catalogue topics,
relevant FS/TS requirements, the invariant trigger index and workflow §§3–7. Inspect the complete
owned implementation and callers at the HEAD recorded when starting, not merely the latest diff.
Record provider/adapter versions, source mode and relevant dirty work. Preserve existing work.

For each integrated review:

- Trace normal work through the complete chain: human or agent intent → effective instructions,
  configuration and context → authority/claim → native call → acknowledgement/output → durable
  state/publication → failure/retry/recovery. Start from the charter's question and inspect every
  component needed to answer it, including neighboring groups' code.
- Review interactions, not just component contracts. Deliberately pair the mechanisms most likely
  to conflict: resume with config/model changes, Stop with late tool input, permissions with
  cancellation, held attachments with retries, quota with dispatch, native compaction with handoff,
  or private room input with judge capture. Use judgment about realistic failure cases; do not
  substitute a huge Cartesian test matrix or a checklist of isolated happy paths for reasoning.
- Cover each applicable entry path at least once: fresh launch, ordinary/wake resume, clone/fork,
  switch/rollback, manual/agent-created task, pipeline, room, terminal and isolated quota check.
  Record unsupported paths explicitly. Trace each critical boundary's identity, lifetime,
  acknowledgement, persistence point, failure behavior and retry/recovery owner.
- Account for every retained catalogue topic in one concise evidence ledger for the group. State
  what code/scenarios were inspected, the evidence and its limits, and any unresolved question.
  These are coverage statements, not separate findings, statuses or subtask closures.
- Check code against shipped requirements and requirements against shipped behavior. Read matching
  invariant bodies and record applicable/non-applicable classes. Planned behavior and absent
  live receipts are not automatically defects. Recheck PROMPT-2, CI-1's closure record, the
  SetSessionConfig lock hypothesis and existing acceptance gates rather than duplicating history.
- Distinguish Chuck/fake-ACP proof, independent pinned-native source/official contract evidence,
  and actual provider adoption. A mock authored by Chuck cannot prove the external behavior it
  imitates. Use focused deterministic failure/race evidence as warranted; do not modify product
  code or specifications during review. Follow TS-06.R5/R9 and INV §17.
- Keep tests isolated and the server loopback-only. Establish existing authorization before bounded
  credentialed runs; obtain missing authorization/credentials only when necessary. Do not claim
  native/provider/device gates from stubs. Record exact unavailable evidence instead.
- Record all real findings in the handoff's §7 format under the integrated ID, with normal-use
  trigger, impact, location, spec/invariant and evidence. A finding needs a supported mechanism;
  do not infer reduced agent quality from prompt size alone. Assign one Fix model based on actual
  open fixes. Existing findings retain their identity; add corroborating evidence without duplication.

Any integrated unit is selectable independently; no mandatory order gates its start. A completed
domain review may leave explicitly named live-evidence gates, but it cannot claim they passed.
Update only that unit's queue state, retain findings/gates, and make the required administrative
commit. Do not fix product code, create a review of this planning commit or silently close siblings.

## Mandatory joins between reviews

The following are shared system questions, not extra queued tasks. The named primary owner traces
the whole scenario, including adjacent code. Other involved reviewers challenge that conclusion
from their own evidence. “Owned elsewhere” is not a reason to stop at a function boundary.

| Join | Primary review | Whole-chain question and other involved reviews |
|---|---|---|
| J1 — Effective agent contract | ASR-A | Do selected/frozen persona, native guidance, model/effort/permissions, cwd, env and directory grants become the intended native behavior after launch, resume and switch? B/C/D challenge requested-versus-applied claims. |
| J2 — One conversation under concurrent control | ASR-C | Does held or steered multimodal input survive permission wait, Stop, late old tool output, reconnect and restart with the intended submission count, turn ownership and durable history? D/H verify context and observed results. |
| J3 — Authority through native execution | ASR-E | Do token/task/room identities, MCP/hook registration and human approval authorize exactly the intended effect under permissive provider modes, resumption and generation replacement? B/C/F verify native and host boundaries. |
| J4 — Interrupted supervised work | ASR-G | Do quota, provider refusal, transport loss, Stop and server restart pause/continue the same task/stage/room once, without stale reports or background tails admitting successors? Include Claude terminal/driver/viewer interruption and its supported recovery or explicit refusal. B/C/E/F/H challenge cleanup, admission and recovery. |
| J5 — Context in delegated work | ASR-D | Do task/room assignment, annotations, resources, attachments and native history arrive in the right checkout/conversation with correct grants and private/public boundaries, including a handoff and compaction? A/B/E/F verify actual consumption. |
| J6 — Ambiguous human action | ASR-H | After an accepted-but-lost response, stale UI, phone suspension or CLI interruption, can retry/reload/Stop/resume preserve intent without resending work, losing context or bypassing an approval? C/D/G verify server/native effects. |
| J7 — Harness cost and quality | ASR-I | Can instructions, tool exposure, native defaults, extra turns, retries, context growth and blocking together explain measured correctness, unnecessary process or latency? Reconcile evidence from A–H and separate provider/repository effects. |

All domain groups must consider amplification and combinations: a harmless prompt wrapper repeated
by resume, a truthful error retried by several dispatchers, a bounded stream blocking the read loop,
or a stale capability used by a different entry point can become a system failure. Look explicitly
for parallel authorities, conflicting defaults, duplicate side effects and a missing owner.

## ASR-A — Instructions, configuration and execution environment

**Question:** Does the agent receive a coherent, minimal and correctly prioritized working contract
that preserves native guidance and the user's choices for the actual project/conversation?

**Integrated scope:** catalogue 01/02/25 together: persona/project/awareness/operating knowledge,
native federation and overrides, frozen configuration and runtime-only additions, resource/cwd/
worktree readiness, environment, directories, skill discovery and exact migrations. Follow shared
composition into both native providers rather than ending at a correct-looking LaunchSpec.

**Start:** `internal/server/{launch,resume,switch,config_sources,worktree}.go`,
`internal/runtime/runtime.go`, `internal/configsource`, `internal/config`,
`internal/agentknowledge`, `internal/worktree` and the catalogue's focused tests/specs.
Read FS-04/FS-08/FS-11/FS-18/FS-19, TS-01.R6/R9/R12/R13, TS-07/TS-11/TS-12.

**Connected searches:** explicit false/empty versus unset through federation and native defaults;
runtime additions leaking into frozen snapshots and accumulating after wake/switch; wrong cwd or
recreated checkout changing project guidance/profile; skill installation failure plus stale
advertisement; native instruction replacement; contradictory/excessive procedural mandates;
resource grants not matching actual paths; setup failure followed by launch; environment precedence
silently changing model, permissions or credentials. Separate Chuck policy from repository guidance.

**Required result:** an effective-contract/lifetime map for both providers with fresh/resumed/
switched examples, source-backed instruction-content assessment and J1 trace. Inspect J5's execution
context. Native payload/adoption evidence from B must agree with the selected/frozen values; a
correct builder alone does not establish that the agent used the contract.

## ASR-B — Codex and Claude native integration fidelity

**Question:** Does the actual selected native runtime faithfully implement Chuck's claims, across
chat, supported terminal use, provider versions, configuration and native resumption?

**Integrated scope:** catalogue 03/04/05/21 together: Installed/Bundled selection, release pins/
patches, auth/readiness/catalog consumers, Codex isolated home and instruction overlay, Claude
native preset/metadata, model/effort/fast/mode negotiation, optional capabilities, PTY/drivers/hooks
and native CLI variance. Evaluate the whole realization of an effective contract for each provider;
compare the providers so shared abstractions do not conceal different meanings.

**Start:** `internal/backend`, `internal/config/codexprofile.go`,
`internal/runtime/{chat,capabilities,acpmap}.go`, `internal/runtime/terminal`,
`internal/server/{provider_runtime,provider_refresh,terminal}.go`, `internal/cli/auth.go`,
`scripts/release` and provider/schema/terminal/assembly tests. Read FS-07/FS-09/FS-10,
TS-04's native delivery/capability/selection contracts and TS-06's packaged-provider requirements.

**Connected searches:** login/probe/launch resolving different executables/homes; bundle patch
or package drift altering Steer/resume; preserved native prompt bytes but lost effective preset;
isolated Codex profile losing native config/skills/MCP/auth; options accepted with a different
model/default/policy; stale advertisements after model change; optional unsupported features
blocking ordinary work; shell/AppleScript/terminal flags and settings changing semantics; unopened
or slow terminal viewer blocking the process; driver children/hooks surviving stop.

**Required result:** provider-by-path realization matrix grounded in independent pinned contracts,
exact requested/applied options and bounded live receipts or explicit gaps. Challenge J1/J3/J5
with native evidence, and verify terminal lifecycle behavior through C. Source or fake capture
cannot close live native-preset, permission, handoff, compaction or credentialed packaging gates.
For Claude terminal, explicitly trace viewer/driver disconnect, Stop, native failure/quota and
server restart through B/C/G: child cleanup, retained context and supported continuation or truthful
refusal, without duplicate work. Do not assume chat's automatic recovery contract applies to PTY.

## ASR-C — Runtime execution, concurrency and control

**Question:** Is Chuck's execution engine one coherent state machine under concurrent calls,
callbacks, late output, cancellation, permission decisions and process failure?

**Integrated scope:** catalogue 06/08/09/10/11/12 together. Keep process/generation registration,
turn admission, Send/hold/withdraw/Steer, RPC read/write/callback scheduling, approvals/modes,
native event/tool/child/background mapping, status, cancel/escalation/Stop and teardown in one
review. Follow acknowledgements through durable publication and inspect resume/switch callers;
history fidelity gets a second perspective in D.

**Start:** `internal/runtime` (especially chat/registry/transport/permission/acpmap/event/activity),
`internal/server/{launch,sessions,hook,session_config}.go`, `internal/state/{manager,status}.go`
and concurrency/crash/permission/tool-input tests. Read FS-01/FS-03/FS-06,
TS-01 runtime/state contracts, TS-04 and matching INV §§1/4/5/9/11/15/16/17.

**Connected searches:** callback lock cycles during configuration; notification/persistence/
subscriber work stalling RPC replies; startup published before ready; late generation N exit
clearing N+1; cancellation ending a successor or permission resolution winning twice; held input
lost/doubled by Steer fallback; malformed/oversized frames leaving waiters hung; native background
tail reopening completed work; late tool input/failure reason assigned to the wrong turn;
process-group/orphan cleanup removing a newer registration; leaked buffers/goroutines/tokens.

**Required result:** one state/identity/resource model, with adversarial interleavings from incoming
intent through native acceptance, event settlement, persistence and cleanup. Lead J2 and challenge
J3/J4/J6. Explicitly investigate the SetSessionConfig read-loop hypothesis and recheck CI-1 closure
evidence without assuming either a product defect or its absence from a mock-only test.

## ASR-D — Conversation, input and history fidelity

**Question:** Does the intended information reach the right agent exactly as intended, remain
available across lifecycle changes, and remain truthful in durable and observed history?

**Integrated scope:** catalogue 07/13/14/15 together: user text/files/skills/images and accepted
artifact bytes; authorized context, peer reads and annotations; native/Chuck identity and history
during resume/fork/clone/switch; handoff primer and compaction; transcript append/sync/index/
reconciliation, snapshot/subscription and reconnect. Review intake, native use, persistence and
readback together rather than treating input delivery and displayed history as separate successes.

**Start:** `internal/server/{resume,switch,clone,primer,attachments,attachment_recovery,sse,reconcile,transcript_window}.go`,
`internal/runtime/{message,images,chat,fork}.go`, `internal/attachment`,
`internal/contextref`, `internal/transcript`, `internal/index`, `internal/bus` and corresponding
continuity/input/security/stream tests. Read FS-01/FS-03/FS-05/FS-13/FS-15,
TS-02's conversation/artifact contracts and TS-04/TS-05 input/history boundaries.

**Connected searches:** actual native context unlike Chuck's displayed history; replay resent as new
work; lost/duplicated primer or premature bridge retirement; snapshot/subscribe gaps; disk failure
after provider acceptance; attachment retry accepting changed bytes or resending a turn; broken
skill/file references after cwd change; context clipping without disclosure; stale/widened grants;
private room or peer history copied into the wrong context; reconnect/reconcile overwriting newer
truth; unbounded scans/fan-out causing execution latency.

**Required result:** information/identity/ownership traces through fresh input, native reply,
Stop/resume/restart, both handoff directions, fork/clone, compaction and recovery, comparing actual
submissions, durable events and observed results. Lead J5; jointly challenge J1/J2/J6. A legible
transcript or successful write does not prove provider context adoption.

## ASR-E — Agent tools, communication and durable delegation

**Question:** Can an agent discover, authorize and complete an intended Chuck action or delegation,
with truthful results and bounded context, without losing work or causing duplicate side effects?

**Integrated scope:** catalogue 16/17/18 together: MCP/hook registration and lifetime, caller identity,
tool/schema/result/error/retry contracts, operating guidance, peer mail/wake/deferred delivery/
budgets, tasks/dependencies/waits/reports, Chucky authority and group placement. Follow each effect
through native tool use, durable claim, target launch/context, report and continuation.

**Start:** `internal/messaging`, `internal/hooks`,
`internal/server/{messaging_registration,messaging_loops,task_dispatcher,task_handlers}.go`,
`internal/state/{messages,activations,tasks,task_waits}.go`, `internal/runtime/tasks.go` and
tool-result/registration/wake/task tests. Read FS-06/FS-16/FS-17/FS-18,
TS-02 activation/task contracts, TS-04's scoped MCP/action protocol, TS-05/TS-10/TS-11.

**Connected searches:** stale generation token plus resumed task authority; broad exemption bypassing
host approval; requested identity overriding token identity; side effect committed but tool reported
retryable failure; mail acknowledged before native receipt; wake/budget loops; dispatcher duplicate
launch after ambiguous failure; task/report guidance missing on resume; dependency repair affecting
unrelated work; late background completion reopening a finished task; Chucky over-delegating from
combined role/tool/reference instructions; descriptions advertising unavailable or planned actions.

**Required result:** complete action/delegation flows, authority and retry semantics, with durable
claim→native execution→effect/report traces under failure/restart and real-provider adoption gaps.
Lead J3 and challenge J4/J5. Explicitly reconcile human-only Stop, ordinary provider permissions and
agent-controlled calls rather than evaluating their checks in isolation.

## ASR-F — Pipelines, Think Tanks and orchestration

**Question:** Do the supervisory systems compose with tasks and native agents to advance exactly
the intended work, preserve private/public context and retain the correct result under interruption?

**Integrated scope:** catalogue 19/20 together: standing coordinators, stages/tasks/attempts,
ordinary and room-backed stages, Think Tank participants/openings/turn ceilings/addressed/private
input, judge/end/result capture, human Continue and cleanup. Inspect the common task/runtime services
where needed to prove progression rather than assuming their local tests establish composition.

**Start:** `internal/pipeline`, `internal/server/pipeline_lifecycle.go`, pipeline handlers,
`internal/server/think_tanks.go`, `think_tank_capture.go`, `think_tank_wire.go`,
`internal/state` pipeline/room/attempt records and `internal/messaging` pipeline/room tools.
Read FS-14/FS-16/FS-21, TS-09/TS-10/TS-14 and shared orchestrator knowledge requirements.

**Connected searches:** multiple services advancing the same attempt; launch before durable claim;
quota/Stop/restart admitting successors; room End and stage acceptance disagreeing; stale or wrong-turn
judge output becoming canonical result; private Send/Steer escaping room privacy; openings or wake
bypassing ceilings; late report/background tail changing an accepted result; recreated coordinators
losing frozen instructions or gaining authority; cleanup stopping unrelated agents; accumulated room
history and process-heavy guidance reducing useful work.

**Required result:** connected ordinary-stage and room-stage state/effect traces through launch,
read/work/report, judge capture, accept/pause/Continue, quota, Stop and restart. Challenge J3/J4/J5,
including the known pause/resume 409 and existing provider-adoption/Stop-race gates. State-machine
tests do not by themselves prove native participants follow the assigned workflow.

## ASR-G — Failure, quota and continuation recovery

**Question:** When execution is refused, uncertain or interrupted, do all layers agree whether work
happened and which continuation is safe, without changing the agent's contract or duplicating work?

**Integrated scope:** catalogue 22 as a cross-system review, not just a quota parser audit. Trace
provider/startup/model-policy/transport/storage failures, quota evidence/reset provenance,
isolated checks, admission and automatic/manual recovery through C/E/F/H and back into A/B/D.

**Start:** `internal/runtime/{errors,session_failure,quota,quota_reset,quota_check}.go`,
`internal/server/quota*.go`, `internal/state/quota*.go`, launch/resume/dispatch/continuation
callers and their failure tests. Read FS-01.R40–R51, relevant FS-04/FS-09 recovery contracts,
TS-02.R45/R50, TS-04.R76/R85–R89/R96/R97 and TS-10.

**Connected searches:** classification falsely proving terminal account quota; unknown/stale reset
authorizing work; a parser's conservative result overridden by a dispatcher or UI Retry; multiple
recovery authorities; check using another account/model/home; probe polluting native history or
creating extra charged turns; recovery re-resolving frozen config; accepted work retried after lost
ack; late success reviving cancelled work; stopped/permission-paused tasks or rooms auto-continuing;
refusal falling back to a different model, mode or prompt.

**Required result:** failure/admission/retry ownership map across every applicable caller, bounded
probe and recovery traces and J4. Challenge J6 and native J1 assumptions. Keep unknown quota windows
and passive-view TBDs explicit; do not invent prompt-based quota features or classify missing live
receipts as proven bugs.

## ASR-H — Human intent across desktop, phone, CLI and HTTP

**Question:** Do human actions preserve the same intent and truthful result all the way to the
native agent, particularly after stale state, transport ambiguity and recovery?

**Integrated scope:** catalogue 23 plus the actual runtime/context/recovery chains it invokes.
This is not merely endpoint or payload parity: follow launch, config/model/mode, send/hold/withdraw/
Steer, attachments, Stop/resume/switch and permission decisions to actual effects and durable state.
Cover documented exclusions instead of assuming every interface offers every action.

**Start:** desktop launch/chat/composer controls, `ui/src/api`, `ui/src/remote`,
`ui/src/store/agentStore.ts`, `internal/cli/operator*.go`,
`internal/cli/{launch,resume}.go` and their server handlers/tests.
Read FS-01/FS-03/FS-09/FS-20/FS-22, TS-03/TS-13 and relevant context/permission contracts.

**Connected searches:** display/request/native option drift; explicit false omitted; wrong selected
agent after navigation; duplicate send after accepted-but-lost response; Stop/resume/retry losing
draft or attachment ownership; reconnect resending accepted input; phone suspension leaving stale
mode/approval controls; CLI follow/interrupt unexpectedly cancelling native work; optimistic state
hiding provider refusal; a UI-only recovery bypassing safe task/room admission.

**Required result:** supported action/entry-point map and complete normal/ambiguous-failure recovery
journeys with native submission/durable-state observations, leading J6 and challenging J2/J4.
Explicitly include annotation/file-source capture, desktop/phone navigation that changes the
selected source or recipient, and paired-device background return with stale state. Check the
agent-facing consequences; unrelated file-reader visual polish remains outside this campaign.
Record independent rendered/paired-device/live-provider evidence and limits. This does not authorize
a redesign or substituting mocked browser success for native or device receipts.

## ASR-I — Whole-system synthesis and agent performance

**Question:** What failures, conflicting assumptions or performance effects appear only when
the instruction, provider, runtime, context, action and supervision systems are considered together?

**Integrated scope:** catalogue 24 expanded into an explicit integrative review of A–H. This is
a substantive second perspective on whole workflows, not a summary or approval of eight reports.
Start from user/agent goals and challenge the combined design, duplicate mechanisms, hidden
defaults, feedback loops and cumulative harness cost. Inspect code across group boundaries where
the assembled evidence is weak or contradictory.

**Start:** the evidence and unresolved joins from A–H, current code/versions, prompt/turn diagnostics,
native-provider captures and representative workflows. Read FS-18.R8/R17/A12/A13, TS-11.R5/R15/R17,
TS-04.R54 and the handoff's prompt/harness investigation; follow other governing contracts as needed.

**Required system journeys:** a manual conversation with queued/steered multimodal input, permission
wait and Stop/resume; a cross-provider handoff followed by immediate Stop/wake and compaction; a
Chucky delegation with mail/task/reporting and a late native continuation; an ordinary pipeline and
a room-backed stage interrupted by quota/Stop/restart; and accepted-but-lost actions recovered through
desktop/phone/CLI. Reconcile J1–J7, not just one component's expected result. Use bounded,
risk-selected combinations and state which scenarios remain source-only or unexecuted.

**Performance assessment:** determine whether existing evidence can identify complete requested
instruction/skill/tool revisions, native adoption, model/effort/permissions, generation/turn and
extra work/latency. PROMPT-2 is an existing diagnostic issue, not automatically a quality cause.
Design and, when authorized/available, run repeated paired native-versus-Chuck tasks (question,
small fix, review, coordination). Hold executable/model/effort, repository guidance, tools/context,
permissions and fresh/resumed state constant where possible. Compare correctness, scope creep,
unnecessary calls/delegation/turns, available usage, elapsed time and output/brief length using
blinded assessment where practical. Separate persona/awareness and repository-harness effects;
No persona/awareness off is not a clean native baseline.

**Required result:** a reconciled system assurance ledger: every topic and join accounted for,
contradictory findings resolved or explicitly open, evidence tied to actual code/provider revisions,
remaining gates distinguished from proven defects, and bounded causal conclusions. Revalidate
relevant conclusions when intervening fixes change a reviewed contract. This unit may start at
any time to design experiments or identify gaps, but an early source pass cannot close the final
synthesis before A–H evidence has been reconciled. Findings in those groups need not all be fixed
for synthesis to finish; they must be accounted for.

Do not claim “nothing was missed” or “no degradation” from independent source reviews or a small
sample. State the achieved coverage, observed mechanisms and residual uncertainty. Missing live
evidence yields a concrete remaining gate/experiment, not a pass. No prompt rewrite, benchmark
platform, instrumentation project or product refactor is authorized by this planning/review task.
