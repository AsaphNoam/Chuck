# Agent-surface search catalogue

Research prepared 2026-10-19 against `047c5c466bbaf807465360e9f543402c79c3374a`.
The original 25 review items are retained here as search topics so consolidation loses no coverage.
They are **not separate review tasks**. The nine integrated charters in
[AGENT-SURFACE-REVIEW.md](AGENT-SURFACE-REVIEW.md) govern selection, responsibility and completion;
[HANDOFF.md](HANDOFF.md#review-queue) owns live queue status.

Use the relevant entries as prompts for a connected review, not as sequential miniature audits.
A topic's Focus and Evidence to inspect identify questions and sources, not a file boundary or
an independent pass/fail result. References to other numbered topics are navigation within this
catalogue: they do not permit deferring an interacting mechanism to another reviewer. The integrated
charter and shared review method take precedence over any narrow ownership wording retained below.
Selecting an old ASR-01–ASR-25 alias selects its whole containing charter.

This is research/navigation, not specifications or substantive review findings. Existing findings,
closed CI-1 evidence, dirty work and live-provider gates must be rechecked in the current handoff.
Paths are starting points; inspect all transitive code needed to establish the group's contracts.

## Checklist ASR-01 — Instruction composition, personas and operating knowledge

**Focus:** what Chuck tells the agent, why it is included, its priority/lifetime, and secure publication
of the embedded operating skill. Follow role/project prompts, awareness opt-out, shared pointers,
resource/task/orchestrator additions and exact legacy migrations. Provider wire translation belongs
to 04/05; configuration selection to 02; comparative quality claims to 24.

**Start:** `internal/server/launch.go` (`joinSystemPrompt`, `applyKnowledgeOverlay`), `resume.go`,
`switch.go`, `internal/runtime/runtime.go`, `internal/config/seed.go`,
`internal/agentknowledge/package.go`, and `internal/agentknowledge/operating-chuck/**`.
Read FS-18.R1–R6/R8/R12/R13/R15–R20, TS-11, TS-01.R6/R9; inspect
`knowledge_overlay_test.go`, `package_test.go`, and `pipeline_orchestrator_instructions_test.go`.

**Search for:** native guidance replacement; contradictory or excessively procedural mandates;
over-broad skill triggers; repeated/accumulating overlays; stale or unavailable skill advertisement;
wrong-role instructions; hidden prompt turns; one-shot context persisted as standing instructions;
tool/schema documentation drift; migration changing customized roles or frozen conversations.
Separate repository AGENTS/skills from Chuck-authored policy.

**Evidence to inspect:** a layer/lifetime inventory with inclusion conditions, bounded size and
ordering checks, and a call-site trace across fresh/resumed paths. Explain which suspected process
burdens have source support and which need ASR-24 measurement. FS-18.A12/A13 and FS-14.A54 require
provider adoption evidence beyond assembled prompt bytes.

## Checklist ASR-02 — Effective configuration, federation and frozen launch values

**Focus:** selection and precedence through the final base `LaunchSpec`: role/no persona, project,
backend/model, effort/fast, permission policy, env, directories, native source bindings, refresh and
snapshots. Native encoding belongs to 04/05 and runtime mode changes to 11.

**Start:** `internal/server/launch.go`, `resume.go`, `switch.go`, `config_sources.go`,
`internal/configsource/{manager,claude,codex,security,watch}.go`, `internal/config`, and
`internal/runtime/runtime.go`. Read FS-04, FS-08, FS-09.R35–R42/R50–R56,
TS-01.R6/R9/R12/R13, TS-07, TS-02.R43/R44. Inspect `config_sources_test.go`,
`internal/configsource/*_test.go`, and `internal/config/backends_validate_test.go`.

**Search for:** unset versus explicit false/empty losing intent; ambient/profile/project precedence;
wrong cwd/profile/trust root; stale or failed resolution treated as fresh; ordinary resume resolving
new defaults; refresh silently ignored; frozen permissions/model changing; env secrets/reserved
variables overwritten; native-home writes; mirrored cache becoming authority; incomplete rollback.

**Evidence to inspect:** a field-by-path precedence table, frozen-versus-runtime field inventory,
and isolated tests of explicit/override/native/default/invalid/refresh cases for both providers.
Trace caller-provided options from task/pipeline/room paths as well as HTTP launch.

## Checklist ASR-03 — Provider selection, packaging, authentication and model discovery

**Focus:** which executable/adapter actually runs and whether login/readiness/model discovery agree
with it. Cover source and release builds, Installed/Bundled mode, overrides, refresh/update, catalog
import, pinned dependencies and patches. Native session semantics belong to 04/05; general Chuck
updater behavior is included only where it changes the agent runtime.

**Start:** `internal/backend/providerexec`, `credcheck`, `providerauth`, `adapter.go`,
`internal/server/provider_runtime.go`, `provider_refresh.go`, `internal/cli/auth.go`,
`internal/config/codexmodels.go`, `scripts/release/{package.json,package-lock.json,assemble.sh}`
and `scripts/release/patches/`. Read FS-09, FS-10.A10–A12, TS-04.R15/R16/R71–R79,
TS-06.R12–R22/R30–R32. Inspect resolver, credential, provider-runtime and release assembly tests.

**Search for:** consumers bypassing the resolver; silent mode fallback; PATH/symlink/managed-root
shadowing; different login/probe/launch binaries or homes; nested-Claude env contamination; native
login falsely requiring an API key; probe hanging/leaking output; stale catalogs changing model or
effort; package dedup/patch/hash drift; update losing an explicit provider choice.

**Evidence to inspect:** consumer/executable/home matrix and finite source/Installed/Bundled failure
matrix. Record exact package pins. Keep TS-06.R21/R31/R32 and the finite credentialed four-combination
smoke distinct from resolver and assembly proof.

## Checklist ASR-04 — Codex native adapter and instruction/configuration delivery

**Focus:** Codex-specific argv/env/wire meaning, capability claims and external oracles, consuming the
base spec from 02 and selected executable from 03. Shared lifecycle, transport and permission
settlement remain with their owners.

**Start:** Codex adapter in `internal/backend/adapter.go`, `internal/config/codexprofile.go`,
Codex environment/session builders in `internal/runtime/chat.go`, `capabilities.go`, `acpmap.go`,
`internal/runtime/codex_native_instructions_test.go`, `params_test.go`, and the pinned steering
patch. Read TS-04.R14/R18–R22/R46–R51/R54/R61/R62/R69/R77/R79/R82, TS-06.R22/R32,
FS-18.R17 and FS-09.R35–R42/R50–R56.

**Search for:** `CODEX_CONFIG` replacement of native `developer_instructions`; dotted-key/profile/
project precedence drift; isolated `CODEX_HOME` dropping auth/config/skills/MCP; model/effort options
accepted but ineffective; fast-mode false success; sandbox/approval policy mismatch; unsupported
hooks/terminal falsely claimed; changed advertisements after load; steering patch assumptions;
load replay treated as new work; policy refusal followed by an unintended model/default retry.

**Evidence to inspect:** a pinned native-source/official-contract oracle for every claimed Codex
delivery mechanism and optional extension, exact outbound request captures, and explicit remaining
fresh/resume/handoff/compaction/permission/Steer live receipts. Reuse PROMPT-1/PROMPT-2 history without
assuming a requested payload proves adoption.

## Checklist ASR-05 — Claude native adapter and instruction/configuration delivery

**Focus:** Claude-specific argv/env/metadata/config meaning, native coding preset, option negotiation
and external oracles. ASR-21 owns PTY/drivers; inspect terminal flags here for native meaning.

**Start:** Claude adapter in `internal/backend/adapter.go`, `claudeSystemPrompt`, session builders and
ordered configuration in `internal/runtime/chat.go`, `capabilities.go`, `acpmap.go`,
`params_test.go`, `acp_session_schema_test.go`, `session_mode_test.go`, and `acceptance_test.go`.
Read TS-04.R13/R14/R18/R19/R25/R45–R47/R54/R69/R79/R80/R82/R92,
FS-18.R17/A13, FS-09.R39/R40/R48/R50–R55 and FS-07.R3.

**Search for:** replacing/omitting the native `claude_code` preset instead of appending; new/load
metadata drift; wrong model/add-dir/trust metadata; effort applied before model and reset; stale
option advertisements; native model-policy refusal mutating Chuck identity; fast-mode value-shape
errors; permission mode accepted but ineffective; terminal `--settings`/`--effort` differences;
Claude version/notice/usage/Steer assumptions embedded only in fake fixtures.

**Evidence to inspect:** pinned adapter/CLI contract table and exact request/readback cases for
fresh/load/model-change/refusal/unsupported options. Keep FS-18.A13, FS-03.A67 and TS-06.R32 live
adoption receipts open unless actually exercised; object-shape tests alone do not close them.

## Checklist ASR-06 — Fresh launch, process registration, crash and restart recovery

**Focus:** allocating/claiming a new runtime generation, startup sequencing, publication, crash exits,
server shutdown/startup reconciliation and stale ownership. Cancellation/termination mechanics are
09; resumed native history is 07.

**Start:** `internal/server/launch.go`, `sessions.go`, `reconcile.go`, `internal/runtime/chat.go`
(`Start`), `registry.go`, `reconcile.go`, `procstart*.go`, `internal/state/manager.go`, and
`status.go`. Read FS-01, TS-01.R4/R7–R9, TS-02's running/session/status contracts, TS-04.R22,
INV §§1/4/5/9/15. Inspect `launch_test.go`, `registry_crash_test.go`, `session_failure_test.go`,
`internal/server/reconcile_test.go` and `internal/state/manager_test.go`.

**Search for:** publish-before-commit; concurrent launches or duplicate claims; registration visible
before initialize/load/config finishes; failures leaving a live invisible child; late exit of N
clearing N+1; PID reuse mistaken for ownership; restart falsely adopting a process; status repair
overwriting newer state; startup/shutdown races with dispatchers and background loops.

**Evidence to inspect:** state/resource transition table and injected failure at each startup and
publication step, including process-exits-before/after-registration and server restart. Assert both
durable truth and absence of leaked processes/registrations; a successful start alone is insufficient.

## Checklist ASR-07 — Resume, clone, fork, switch and native history continuity

**Focus:** preserving stable identity, native conversation, model/config and user context across
history-changing lifecycle operations; primer lifetime and rollback. Prompt content belongs to 01,
base precedence to 02, artifact durability to 13/14.

**Start:** `internal/server/{resume,switch,clone,primer,archive,archive_actions}.go`,
`internal/runtime/chat.go` (`Resume`, session load/replay gates), `fork.go`,
`internal/transcript/clone.go` and frozen/history-bridge state. Read FS-01.A7/A8/R14/R15/R30,
FS-05, TS-02.R35/R47/R48, TS-04.R50/R65/R69 and TS-11.R4/R5.
Inspect `fork_test.go`, `switch_test.go`, `clone_test.go`, `primer_test.go` and continuity tests.

**Search for:** native session ID confused with agent/generation ID; resume becoming a fresh session
without notice; duplicate native replay; stale or repeated handoff primer; bridge retired before
effective compaction; explicit target model lost on immediate Stop/wake; switch failure destroying
the original; clone/fork crossing identities, authority or artifact ownership; refresh losing history.

**Evidence to inspect:** fresh→turn→Stop→resume, restart, clone/fork, both handoff directions,
immediate Stop/wake, target refusal/rollback and compaction matrix. Distinguish native history from
Chuck's displayed transcript. Carry forward FS-01.A7/A8 and TS-02.R48 live-context-adoption gaps.

## Checklist ASR-08 — Message submission, held input, Steer and turn ownership

**Focus:** one executing Chuck turn, queue/promotion/withdrawal, acknowledgement, native idle/active
Steer, same-turn background continuation and activation admission. Mail content is 17; task/room
admission claims are 18/20; cancellation is 09.

**Start:** `internal/runtime/chat.go` (`SendMessageOrHold`, `SteerMessage`, turn claims,
`runMessageTurn`, `finishTurn`), `activation_kinds.go`, `registry_message.go`,
`internal/server/queue_steer_test.go`, `internal/runtime/{queue_steer,turn_ownership,steer_background}_test.go`.
Read FS-03.R40–R50, FS-01.R52, TS-04.R48/R49/R51/R81 and TS-01 activation contracts.

**Search for:** lost/doubled prompts after ambiguous acknowledgement; two turns claimed at once;
late old completion releasing a new turn; held input overwritten, promoted twice or sent after
withdrawal; Steer fallback incorrectly starting a task; private Send becoming public room input;
background tool continuation making an idle card busy or reopening a completed task; activation
inserting a user turn or racing a manual send.

**Evidence to inspect:** deterministic interleavings around claim, send, provider acceptance,
finish, promotion and withdrawal, including transport failure and Stop. Show that displayed user
input, persisted input and actual provider submissions have the intended count and order.

## Checklist ASR-09 — Stop, cancellation, process containment and teardown

**Focus:** cancel versus Stop, escalation, generation/turn fencing, orphan termination and shared
resource cleanup on success/failure/shutdown. Terminal-driver mechanics are 21; registration
creation is 06/16, but verify teardown symmetry against both.

**Start:** `internal/runtime/{permission,registry,chat,procstart}.go`, platform `procstart` files,
`internal/server/launch.go` (`teardownAgentRegistration`), stop callers in sessions/archive/switch,
`internal/runtime/{orphan_stop,room_stop,registry_crash,permission}_test.go` and
`internal/server/permission_teardown_test.go`. Read FS-01's Stop/cancel/crash contracts,
FS-06.R41/R42, TS-01.R9, TS-04.R95, TS-05 and INV §§4/5/9/15.

**Search for:** stale cancellation killing a successor; a cancel signal sticky across turns;
grace timer escalating after completion; parent exits leaving descendants; PID/group identity
mistakes; cleanup blocking behind a provider call; hook/MCP/settings/temp-file leaks; double teardown
removing a newer token; stopped agents still accepting automatic work; shutdown silently dropping
uncommitted results. Check bounds on waits, hook calls and process exit.

**Evidence to inspect:** idle/active/permission-paused/background/crashed/orphaned generations,
late frames, rapid Stop/resume, refused/ignored cancel and server shutdown. Check the current CI-1
closure evidence and cancellation regression rather than reopening the historical finding without
new evidence. Verify test-peer cancellation semantics independently.

## Checklist ASR-10 — ACP transport, callbacks, locking and backpressure

**Focus:** NDJSON/JSON-RPC framing and meaning, pending-call routing, read/write/shutdown scheduling,
protocol bounds and inbound callbacks. Event interpretation is 12; provider shapes are 04/05.

**Start:** `internal/runtime/{transport,jsonrpc,chat,ringbuffer,hub}.go`, notification/request
handlers, `transport_test.go`, `errors_test.go`, and `acp_session_schema_test.go`.
Read TS-04.R9/R11/R22/R54/R62, TS-05, INV §§5/9/11/12/16/17.

**Search for:** a synchronous notification/callback needing a lock held across `Call`; reader
blocked by persistence, approval or slow consumers; partial/oversized/malformed frames; stderr
contamination of stdout; unknown method/version/extension handling; mismatched/string/late IDs;
unbounded pending maps/goroutines; write contention; context cancellation not releasing a waiter;
EOF leaving pending calls or live state hung; unsupported FS/terminal/auth callbacks left waiting.

**Evidence to inspect:** wire fault injection before/between/after initialize/new/load/config/prompt
responses, full-duplex callback cases, bounded memory/work and timeout/shutdown assertions.
Explicitly reproduce or disprove the handoff's SetSessionConfig/`as.mu` notification hypothesis;
include launch-time and live configuration calls, since both consume the same read loop.

## Checklist ASR-11 — Permissions, session modes and human decisions

**Focus:** human/provider permission lifecycle, normalized policy/mode state, denial/approval/timeout,
host-owned peer Stop consent and decision receipts. ASR-02 selects default policy; 04/05 validate
native option meaning; 16 owns token authorization.

**Start:** `internal/runtime/{permission,host_permission,session_mode,chat}.go`, permission decoding in
`acpmap.go`, `internal/server/session_config.go`, `internal/server/session_mode_test.go`,
`internal/server/peer_stop_test.go`,
`internal/runtime/{permission_scope,host_permission,session_mode}_test.go`, and
`ui/src/components/chat/renderers/PermissionPrompt.tsx`.
Read FS-03.A67, FS-06.R41/R42/A29/A30, FS-04.R60, TS-04.R92/R94/R95 and TS-02.R49.

**Search for:** skip-permissions bypassing a required human-only action; unsafe normalization of
provider choices; mismatched tool-call/turn/generation identity; duplicate decisions; losing a pending
decision after failed write; stale UI approving a new request; timeout/Stop/turn-end resolving twice;
model changes leaving stale modes; mode refusal persisting false success; restart retaining invalid
approval authority; peer Stop using a broader caller/target than the approved snapshot.

**Evidence to inspect:** single-winner race tests and durable-before-provider-response ordering;
reload/reconnect and stale decision cases. Most-permissive real Claude/Codex peer-Stop gates and
native mode adoption remain distinct from fake permission tests.

## Checklist ASR-12 — Native events, tools, child agents, background activity and context usage

**Focus:** normalization of provider output into meaningful tool/thought/notice/child/background,
usage/compaction, status and turn-boundary events. Byte persistence/fan-out is 13; native feature
truth is 04/05; queued-turn ownership is 08.

**Start:** `internal/runtime/{acpmap,event,activity,subagent,tasks,filereport,chat}.go`,
`internal/state/status.go`, `internal/server/hook.go`, `internal/runtime/{acpmap,activity,subagent,tasks,tool_input,notices,session_failure}_test.go`.
Read FS-03, FS-01.R52, TS-04.R24/R25/R63/R64/R66/R68/R80–R82/R98/R99 and TS-08's
turn-activity contracts.

**Search for:** event identity collisions; fragmented/late tool input lost or assigned to a new turn;
tool failure reason discarded; thought/final text mixed; unknown tool status stuck running;
child output becoming parent/public output; background native jobs mistaken for Chuck tasks;
out-of-order context totals or compaction resetting the wrong session; stale capability/command
advertisements; late native work turning an idle completed card active; provider error hidden as success.

**Evidence to inspect:** provider-shaped independent fixtures plus fragmented, interleaved,
late-after-cancel, model-change and load-replay sequences. Verify raw context pair and derived
percentage semantics, status truth and correlation IDs. Reuse CI-1 and late-tool/live-Claude gates.

## Checklist ASR-13 — Transcript durability, replay, indexing and stream continuity

**Focus:** accepted normalized events from append through durable transcript/state/index and live or
reconnected subscribers, including fallback reconciliation. ASR-07 owns native history adoption;
14 owns attachment bytes; 12 owns event meaning.

**Start:** persistence paths in `internal/runtime/chat.go`, `internal/transcript`, `internal/index`,
`internal/state`, `internal/bus/bus.go`, `internal/server/{sse,reconcile,transcript_window}.go`;
inspect writer/reader, event, bus, reconciliation and transcript-window tests.
Read FS-03.R3/R35, FS-05, TS-01.R5/R8, TS-02 transcript/session contracts and INV §§7/9/11/15/16.

**Search for:** emit-before-commit; partial write/disk-full acknowledged as accepted work; event IDs
reused or reordered; snapshot+subscribe gaps; slow clients blocking the agent; reconnect duplicates
or missing turn boundaries; index/reconcile treating native artifacts as Chuck events; reads
silently swallowing corruption; reload turning displayed history into resent provider input;
unbounded transcript scans or retained buffers impairing provider throughput.

**Evidence to inspect:** failure injection for append/sync/index/status publication, partial tails,
restart, subscriber gaps and dropped clients. Compare actual provider submissions with durable and
displayed event counts; verify repair never overwrites a newer generation's state.

## Checklist ASR-14 — Submitted text, mentions, files, images and attachment bytes

**Focus:** constructing and retaining the exact user `Message` sent to the provider: file/skill
mentions, negotiated image input, ordinary-file references, immutable accepted bytes, retries and
assistant image output. ASR-15 owns source authorization/context links; 23 owns client action parity.

**Start:** `internal/runtime/{message,images,chat,filereport}.go`,
`internal/server/{attachments,attachment_recovery,filesearch,files_commands}.go`,
`internal/attachment/store.go`, `ui/src/components/chat/Composer.tsx`,
`ui/src/components/chat/attachments.ts`, and `internal/runtime/{attachments,image_frame}_test.go`.
Read FS-03.R81–R86, FS-13.R25/R26, TS-04.R90/R91, TS-02.R46/R47 and TS-05 attachment contracts.

**Search for:** mention expansion dropping text or changing instruction priority; ambiguous paths;
skill names not resolvable by the provider; binary file silently sent as text; image capability
ignored; changed bytes under one request ID; lost response causing duplicate submit; image/file
limits applied after allocation; provider filesystem references outside granted roots; blob data in
transcripts/SSE/logs; Stop/wake/clone/archive losing ownership; failed preflight consuming a draft.

**Evidence to inspect:** exact ACP message parts for text/file/image/skill combinations, advertised
and unsupported capabilities, changed/missing files, failed writes and retry/wake paths. Use
attachment-security/recovery/failure tests; distinguish browser acceptance from actual native input.

## Checklist ASR-15 — Authorized context, peer reads and annotations

**Focus:** context beyond the submitted message: pull references/grants, peer transcripts, annotation
anchors and task-provided sources. Native config precedence is 02; message/media encoding is 14;
task scheduling is 18; project/worktree/resource execution context is 25.

**Start:** `internal/contextref`, `internal/messaging/context_tools.go`,
`internal/server/think_tank_annotations.go`,
`ui/src/lib/annotations.ts`, `ui/src/components/chat/annotationDelivery.ts`, and corresponding
context/annotation tests. Read FS-13/FS-15, TS-02.R24/R47 and TS-05 context/path contracts.

**Search for:** a link granting more than its source/target permits; peer reads accidentally waking
or copying private history; stale or mislabeled annotation ranges; clipping without disclosure;
context duplicated or unavailable at execution; symlink/path escape; source paths resolved against
the wrong checkout; granted content silently broadening after restart or source mutation.

**Evidence to inspect:** source→grant→render→delivery→read traces and isolated permission/path/size/
staleness tests. Exercise prompt/mail/new-task delivery and later pull reads, including stopped,
archived, room and unauthorized targets, without claiming file-reader rendering closure.

## Checklist ASR-16 — Agent-facing MCP, hooks, caller authority and result contracts

**Focus:** gateway setup/teardown and token-derived identity, hook/settings injection, tool discovery,
schema/result/error/retry envelopes and privilege separation. Individual tool effects belong to
11/15/17–20; terminal transport is 21. Planned direct-action transport remains out of scope as an
implementation task; verify it is not accidentally advertised as shipped.

**Start:** `internal/messaging/{messaging,tools}.go` and tool registration/result helpers,
`internal/server/{messaging_registration,hook,launch}.go`, `internal/hooks/hooks.go`,
`internal/hooks/scripts/`, `tool_result_contract_test.go`, `messaging_registration_test.go`,
`hook_test.go`. Read FS-17, FS-06, TS-04.R6/R7/R17/R28/R78/R83/R93–R95,
TS-05 and TS-11.R7/R8; inspect the installed operating references for contract drift.

**Search for:** stale token authorizing a replaced generation; caller-supplied IDs overriding
token identity; leaked token/config in argv/logs; over-broad permission exemptions; incomplete
registration on resume/rollback; unavailable tools still advertised; hooks hanging native work;
native MCP servers unintentionally removed/duplicated; `isError`, text and structured results
disagreeing; retryable errors causing repeated side effects; successful effects reported as failure.

**Evidence to inspect:** tool inventory with authority/effect/result/retry owner and generation-scoped
registration/teardown tests. Check unknown/denied/ambiguous-loss results and hook network failure.
Distinguish native hook receipt from a generated settings-file assertion.

## Checklist ASR-17 — Peer mail, deferred delivery, waking and message budgets

**Focus:** address resolution, durable mailbox semantics, inline mail receipts, waking/deferred
activation, quotas/budgets and provider-facing mail prompts. ASR-08 owns generic turn admission;
18 owns task dependency waits; 16 owns authentication.

**Start:** `internal/messaging/messaging.go`, `peer_tools.go`, `internal/state/messages.go`,
`activations.go`, `internal/server/{messaging_loops,messaging_registration,resume}.go`,
mail preparation/settlement in `internal/runtime/chat.go` and activation/wake tests.
Read FS-06, TS-04.R26/R27/R53/R67, TS-02.R23/R28/R34 and TS-01 activation contracts.

**Search for:** accepted-but-undelivered mail; delivery acknowledged before provider receipt;
duplicate inline mail after retry/restart; stopped/wrong-project recipients waking incorrectly;
budget bypass through alternate tools/surfaces; excessive prompt wrappers or polling pressure;
activation consuming a queued user message; mail lost during resume failure; deferred delivery never
rearmed or becoming a wake loop; private mail leaking into room/public context.

**Evidence to inspect:** sender→durable inbox→claim→prompt→receipt→settlement trace with wake,
busy/idle/stopped, ambiguous prompt failure, restart and budget exhaustion interleavings. Verify
same native conversation, bounded prompt content and non-destructive failure recovery for both providers.

## Checklist ASR-18 — Durable tasks, waits, reports and Chucky delegation

**Focus:** task instruction/context freezing, assignee identity/authority, dispatch claims, dependency
continuations, reports and Chucky/group placement. Generic prompt composition is 01; pipelines
and rooms own their higher-level progression in 19/20.

**Start:** `internal/state/{tasks,task_waits}.go`, `internal/server/task_dispatcher.go`, task handlers,
`internal/messaging/task_tools.go`, `internal/runtime/tasks.go`, task dispatch/HTTP/state/wait tests
and task guidance in `internal/agentknowledge/operating-chuck/references/coordinate-work.md`.
Read FS-16, TS-10, TS-02.R51, FS-18 and relevant FS-14 task contracts.

**Search for:** duplicate launch after claim/restart; task marked running/completed without actual
execution; report accepted from a stale generation/attempt; blocked/wait interpreted as success;
task/report instructions omitted on resume; dependencies reopening finished work; repair affecting
unrelated descendants; user follow-up treated as a new assigned task; Chucky inheriting excessive
authority or wrong role/group/cwd; extra report prompts compensating for missing initial context.

**Evidence to inspect:** durable claim→launch→read→work→report/wait→continuation trace with failed
launch, Stop, late report, crash and restart. Preserve FS-16.A33/A34 live gates: actual agents must
adopt/report their task and late native background work must leave a completed card idle.

## Checklist ASR-19 — Pipelines and standing orchestration

**Focus:** pipeline attempt progression, coordinator context and authority, stage launch/resume,
acceptance, pause/recovery and standing owner behavior. ASR-18 owns the underlying task service;
20 owns room internals; inspect both interfaces without duplicating their implementations.

**Start:** `internal/pipeline`, `internal/server/pipeline_lifecycle.go`, pipeline handlers,
`internal/server/task_dispatcher.go`, `internal/messaging/pipeline_tools.go`,
`internal/state/{pipelines,pipeline_tasks,pipeline_projection}.go`, pipeline room/orchestrator tests.
Read FS-14, TS-09, TS-10 and FS-18/TS-11 standing-context requirements.

**Search for:** competing progression authorities; dispatch before durable attempt claim;
stale stage result advancing a replacement; Stop/quota/restart launching a successor; rejected
launch continuing with old config; coordinator instructions absent or duplicated; background tail
treated as a result; accepted result changed by late output; human Continue losing state or starting
twice; unintended chat access bypassing a paused stage; cleanup stopping unrelated agents.

**Evidence to inspect:** attempt/state/side-effect matrix for ordinary and room-backed stages,
failed launch/resume/report, concurrent Stop/accept, quota pause and restart. Reconcile the known
409 pause race as evidence, and preserve FS-14.A49/A50/A54 adoption/restart gates until proved.

## Checklist ASR-20 — Think Tanks, room turns, private input and judge results

**Focus:** participant/judge instructions, public/private context, room publication and turn scheduling,
ceilings, addressed input, synthesis capture and retained result. Pipeline ownership/acceptance is
19, generic task service 18 and native-turn service 08.

**Start:** `internal/server/think_tanks.go`, `think_tank_capture.go`, `think_tank_wire.go`, room
handlers, `internal/state/think_tank*.go`, `internal/messaging/think_tank_tools.go`, and room
capture/exit/admission/opening tests. Read FS-21, TS-14, TS-04.R84 and TS-09 room-stage contracts.

**Search for:** one participant seeing another's private input; openings serialized/duplicated
contrary to policy; wake or continuation bypassing ceilings/End; judge accepting the wrong execution
turn or late background text; tool publish/report racing end; denied proposal executing anyway;
restarted room losing unread input or spawning duplicate agents; empty/stale synthesis counted as
accepted; oversized room history degrading context; room Stop cancelling the wrong native work.

**Evidence to inspect:** room/member/attempt/entry/turn correlation trace through parallel openings,
read/submit, addressed/private Send/Steer, permission denial, native resume, End-only judge and
Stop/End races. Keep FS-21.A31/A32 and FS-14.A50 live-provider gates separate from fake-room tests.

## Checklist ASR-21 — Claude interactive terminal, PTY and external drivers

**Focus:** Claude terminal execution, PTY I/O/draining, xterm/tmux/iTerm2 drivers, keystroke injection,
hook-derived runtime behavior and interface parity. Native flag meaning is 05; hook authorization
is 16; shared cancellation policy is 09. Codex terminal must remain an explicit refusal.

**Start:** `internal/runtime/terminal/{terminal,driver,ptyhub,bridge,xterm,tmux,iterm2,applescript,capabilities}.go`,
`internal/server/terminal.go`, `ui/src/components/chat/TerminalTab.tsx`, and terminal/hub/bridge/
AppleScript/persistence tests. Read FS-07, FS-01.R15/R17–R24, TS-01.R4/R6/R9 and TS-05.

**Search for:** unopened/disconnected viewer blocking the child; snapshot/fan-out loss; a viewer
closing shared PTY; resize text becoming user input; terminal escape/output abuse; shell/AppleScript
quoting errors; raw keystroke delivery dropping or duplicating prompts; unsupported task/context
contracts silently accepted; hooks falsely marking idle; driver child surviving Stop; sensitive
settings in terminal output; interface switch resuming the wrong native session.

**Evidence to inspect:** no-viewer/multi-viewer/reconnect/slow-viewer/resize matrix plus driver
unavailability, quoting, process exit and orphan cleanup. State which native terminal flag/hook
receipts remain unverified rather than treating fake PTY or generated settings as provider proof.

## Checklist ASR-22 — Provider failures, quota admission and recovery

**Focus:** classification and visibility of startup/live failures and quota evidence, automatic
successor admission, reset provenance, isolated checks and continuation. ASR-03 owns login/readiness;
04/05 own native evidence shape; 08/18–20 consume the common admission result.

**Start:** `internal/runtime/{errors,session_failure,quota,quota_reset,quota_check}.go`,
`internal/server/quota*.go`, `internal/state/quota*.go`, terminal completion decoding in
`acpmap.go`, and quota/failure/dispatch/concurrent tests. Read FS-01.R40–R51/A28–A34,
FS-04 quota settings, TS-04.R76/R85–R89/R96/R97, TS-02.R45/R50 and TS-10 quota contracts.

**Search for:** arbitrary 429/text treated as terminal account quota; partial metadata proving
more than it says; seconds/milliseconds/timezone errors; unknown reset causing auto-run; quota
blocked agent admitting queued/task/room work; wrong account/model check; isolated probe persisting
or polluting history; repeated `/rename`/archive/control turns; native MCP-disable overlay malformed;
late success reviving cancelled recovery; quiet checks consuming unbounded tokens/processes.

**Evidence to inspect:** provenance→block→admission→check→continue/hold trace across all callers,
with reset unknown/stale/invalid, setting disabled, restart and stop races. Preserve the exact live
Claude `_claude/rateLimit` and Codex quota/isolation receipts. Passive paired snapshots are TBD;
do not invent a prompt-based quota feature to fill that evidence gap.

## Checklist ASR-23 — Operator entry-point parity and recovery

**Focus:** converting human intent to the correct server request from desktop, paired phone, CLI and
direct HTTP, then exposing accepted/refused/ambiguous outcomes without changing the action. This is
a behavioral boundary review, not a broad visual redesign or another full server-domain audit.

**Start:** `ui/src/features/launch/NewAgentModal.tsx`, `ui/src/components/chat/{ChatPanel,Composer,PermissionModeToggle}.tsx`,
`ui/src/remote/{NewAgentForm,AgentScreen,RunScreen,PhoneApp}.tsx`, `ui/src/api`,
`ui/src/remote/api.ts`, `ui/src/store/agentStore.ts`, `internal/cli/operator*.go`,
`internal/cli/{launch,resume}.go`, and HTTP handlers for the selected actions.
Read FS-01/FS-03/FS-09, FS-20, FS-22, TS-03 and TS-13; inspect corresponding client/CLI contract tests.

**Search for:** omitted versus explicit option drift; displayed model/effort/mode unlike the sent
value; double submit after timeout; stale selection targeting another agent; Stop/Resume/Retry losing
draft/context; optimistic state hiding provider refusal; reconnect resending an accepted turn;
phone suspension leaving stale controls; CLI interrupt cancelling work unexpectedly; advertised
surface support exceeding backend/interface capability. Cover launch, send/hold/withdraw/Steer,
Stop/resume/switch, config/mode and permission decisions.

**Evidence to inspect:** a supported/excluded action-by-entry-point matrix with exact request
equivalence and failure/retry journeys. Keep paired-device/background-return gates and independent
rendered verification visible where required. Existing lack of phone terminal/mail/task UI is a
documented boundary, not automatically a parity defect.

## Checklist ASR-24 — Diagnostic sufficiency and measured agent performance

**Focus:** whether Chuck-induced degradation can be detected and attributed, plus a bounded,
reproducible comparative review of correctness, context, unnecessary work and latency. This item
must produce an experiment design even if live evidence is unavailable; it must not infer model
quality from prompt length or fake-provider success. It can start independently of other slices.

**Start:** prompt fingerprint/turn diagnostics in `internal/runtime/chat.go`, `acpmap.go`,
`internal/transcript`, `internal/state/status.go`, provider version/source observations, TS-11.R5/R15/R17,
FS-18.R8/R17/A12/A13, TS-04.R54, and the handoff's prompt/harness investigation (PROMPT-2).

**Search for:** inability to correlate exact requested instruction/skill/tool revisions, frozen
versus current persona, executable/model/effort/permission and generation/turn; hidden context
growth, retries, work duplication, extra native turns, blocking/backpressure or excessive process
mandates. Requested delivery and effective provider adoption need separate evidence. Use mechanisms
reported by other slices as hypotheses, not automatic causal conclusions. Completed sibling reviews
are useful inputs, not prerequisites; do not re-audit their implementations to start this item.

**Evidence to inspect:** assess existing diagnostics without adding production instrumentation;
record missing observability as evidenced findings. Specify repeated paired native-versus-Chuck
tasks (question, small fix, review, coordination) with held-constant executable/model/effort,
repository guidance, tools/context, permissions and fresh/resumed state. Compare correctness,
scope creep, calls/delegation/turns, available usage, elapsed time and answer/brief length using
blinded assessment where practical. Test awareness/persona and repository-harness effects separately;
No persona/awareness off is not a clean native baseline. Run only a bounded authorized live matrix,
retain redacted receipts, and state sample limitations and unresolved causality. This review
does not authorize prompt rewrites, a benchmark platform or a performance refactor. If live work
cannot run, record the source/diagnostic assessment, concrete experiment plan and remaining gate;
do not claim that quality has been measured or causality established.

## Checklist ASR-25 — Project execution context, worktrees and shared resources

**Focus:** the actual cwd, checkout readiness, setup execution, shared resource location and directory
capabilities delivered to an agent. ASR-02 owns configuration selection/freeze; 15 owns context-read
authorization; 06 owns starting the final process. Limit worktree review to operations that can
misdirect, block, contaminate or destroy an agent's working context.

**Start:** `internal/server/{worktree,worktree_handlers,launch,resume}.go`,
`internal/config/{resources,worktrees}.go`, `internal/state/worktrees.go`, `internal/worktree/git.go`,
`internal/server/{project_resources,worktree_projects}_test.go`, `internal/worktree/git_test.go` and
`internal/config/{resources,worktrees}_test.go`. Read FS-11/FS-19, FS-01's cwd/resume contracts,
TS-01.R6/R9, TS-12 and TS-05 directory/path constraints.

**Search for:** wrong repository/worktree cwd; source root accepted before setup/recreation finishes;
setup env differing from agent env; shell interpretation of argv-only commands; unbounded setup or
Git calls; setup warning treated as successful readiness; resume recreating a checkout while an old
generation uses it; cleanup deleting a live/dirty/unowned checkout; resources missing, shared across
the wrong project or granted too broadly; shared resource instruction/env/add-dir drift among manual,
task, pipeline, room and resume paths.

**Evidence to inspect:** selected-project→canonical cwd→setup→resource grants→spawn trace,
with isolated wrong/missing/deleted/dirty checkout, setup failure/timeout and recreate/Stop/restart
interleavings. Verify the actual child cwd/env and native access to the intended directories;
do not extend this slice into worktree UI redesign or unrelated Git feature review.
