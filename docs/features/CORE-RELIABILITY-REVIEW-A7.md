# CR-A7 — Sustained capacity and cost

Prepared 2026-10-20 from two focused GPT-6 Luna research passes and parent integration against
`44d285fc`. This maps **A7 only** from the
[core reliability priorities](CORE-RELIABILITY-AUDIT-PLAN.md). It is one integrated review for a
frontier reviewer. Research inspected requirements, current producers/consumers and existing test
assertions; it performed no audit, workload measurements, failure probes, test execution or product
repairs. Concurrent ASW/ASF remediation can change these paths; refresh the live ledger and dirty
work before review. [HANDOFF.md](HANDOFF.md#review-queue) owns status.

## Question and responsibility

Does Chuck remain a usable development hub as conversations, retained work and concurrent activity
accumulate, without multiplying native work? Can an agent retrieve needed context and make progress
while the operator can still intervene, and do temporary resources leave with their actual owners?

Trace retained history/work → requested page or current view → physical reads/rendering/queries →
shared journal, bus and scheduler pressure → another agent's progress/control → settled resource
ownership. Where this path creates repeated submissions, follow the effect into actual native input
and usage. Prioritize repeatable slowdown, memory/resource growth, control starvation and unnecessary
model work. Cosmetic frame timing and isolated noisy latency are outside this scope.

Start with transcript/context retrieval at realistic accumulated sizes. Follow shared I/O,
reconciliation, fan-out, registrations and retries when source/lifetime evidence or measured pressure
connects them to a consequential cost. This is one cost-and-lifetime review, not a second exhaustive
review of every runtime, query, command, provider feature or UI surface.

## Read first

Read the current handoff's Current position and Active change, this charter and related findings.
Use workflow §§3–7 for review/reporting, with this explicit subsystem scope replacing a commit-diff
scope. Read requirements before source:

- FS-15.R9–R11/A5–A6, TS-01.R22 and TS-04.R28: explicit bounded context retrieval, shared
  projection, fixed source/cursor semantics and no activation or content fan-out from reference
  management. FS-06.R39, TS-04.R93 and TS-02.R49 define the separate direct peer-read path,
  complete-prefix fingerprint, cancellation and eight-scan admission limit.
- TS-02.R7–R10/R16/R26: durable identity versus runtime state, ordered canonical transcript,
  rebuildable search projections and the specific delegated-task read index. R26 is not a
  universal index requirement for other queries or file reads. A1 owns recovery/repair correctness.
- TS-03.R7–R9/R31, FS-02.R48 and FS-03.R39: shared stream, snapshot/live handoff, bounded
  subscribers and open surfaces, request fencing and temporary reconciliation tails.
  FS-20.R13 governs the phone's windowed transcript. FS-22/TS-03.R66–R67 govern CLI follow's
  different bounds if it contributes to the selected workload.
- TS-01.R6/R8/R9/R20–R21/R31–R33/R39/R41: composition/teardown, activation ownership,
  confirmed inline mail, per-conversation control dispatch and execution lifetimes. Follow the
  selected task/quota/room domain's capacity and retry rules; their caps are not interchangeable.
  FS-06.R7–R14/R20/R28–R36 defines mail bounds/budgets and waking versus deferred delivery.
- FS-18.R8/R17/R20–R21, TS-11.R24 and TS-04.R54: process-bound guidance, preserved native
  instructions, identifiable requested delivery and independent provider evidence. A manifest is
  bounded diagnostic metadata, not a complete native usage trace or proof of adoption/quality.
- Read the invariant trigger index and matching bodies, especially §§1/4/5/7/9/16/17; §6 applies
  if following a driver. INV §16 requires admission/capture bounds and explicit lifetimes, not
  trimming after arbitrary accumulation. TS-06.R35 requires the smallest adequate evidence.

Returned-page limits do not establish a scan-cost or latency bound. There is no universal performance
budget supplied by this charter. Separate a demonstrated invariant violation from a measured normal-
use cost and from an optimization idea that has no consequential workload evidence yet.

## Preserve the cost and ownership distinctions

- Canonical history, durable identities and unfinished recovery obligations may legitimately remain
  after Stop. Ephemeral registrations, subscriptions, in-flight buffers and dispatchers have their
  own release boundaries. Do not call every retained byte a leak or erase work to meet a graph.
- Bounded output, peak memory, scan work, in-flight concurrency and lifetime are different facts.
  A small response can require a full scan; a per-conversation cap can multiply across conversations;
  independent scheduler caps can overlap. None alone proves a user-visible defect or a missing
  universal cap. Establish the actual producer, owner, contention and effect.
- Context reference reads, direct peer reads, windowed HTTP transcripts and full local transcript
  reads have different contracts. An FTS index does not accelerate an NDJSON scan merely because
  both consume transcript text. Reuse does not permit skipping authorization, fingerprint or
  omission semantics to make a benchmark faster.
- Separate Chuck CPU/I/O/allocations/network and retained native processes from provider queueing,
  inference, tool execution and repository tests. Count Chuck-originated submissions and retries;
  distinguish one authorized turn from provider-native child/tool activity or a selected room's
  intentionally parallel opening. A read/refetch is not inherently another model turn.
- Requested prompt bytes, heuristic tokens, provider-reported usage and actual adopted context are
  separate measurements. Longer guidance or a larger tool inventory alone does not prove worse
  reasoning. A missing live receipt is a coverage limit, not a performance pass or a new defect.

## Starting cost and lifetime map

Verify this map at the reviewed HEAD. These are source-backed search leads, not measured findings.

| Work/resource | Current owner and entry points | Cost/lifetime boundary to establish |
|---|---|---|
| Reference selection and pages | `internal/contextref/{service,page,render,limits}.go`, `internal/transcript/{reader,project}.go` | `resolveTranscriptSpan` scans turn boundaries; `Read`/`renderSpan` stream from the file's beginning, while the renderer filters the fixed sequence span and `pageWriter` retains at most 32 KiB after a rendered-byte offset. Measure records/bytes decoded and rendered per small or repeated page, not just page size. |
| Direct peer pages | `internal/contextref/peer.go`, `internal/state/agent_history.go` | `ReadPeerTranscript` scans/hashes the captured complete prefix on each page, revalidates replacement and checks cancellation. Eight concurrent peer scans are admitted; do not assume this cap also governs reference reads, HTTP windows or full reads. |
| HTTP transcript/readback and primer | `internal/server/{sessions,transcript_window,primer}.go` | Windowed reads visit the disk transcript, track current summaries and retain a tail; `since_seq`/`before_seq` do not seek. Default/max 750 events and approximate 1 MiB tail accounting retain one newest event even when larger; the physical-record reader cap is 8 MiB. Full local reads materialize all events. Primer output is bounded separately; measure its input work too. |
| Retained identities, tasks and search | `internal/state/{agents,agent_history,pipeline_projection}.go`, `internal/index/indexer.go`, `internal/server/reconcile.go` | Distinguish all-row lists from bounded keyset reads, file replay from incremental indexing, and queries using an intended index from mere limited results. Reconciliation skips unchanged files; determine repeated work when one file changes amid a large retained set. |
| Snapshot, views and reconnects | `internal/bus/bus.go`, `internal/server/{sse,remote_home}.go`, `ui/src/api/{sse,sse-shared-worker}.ts`, `ui/src/store/{agentStore,transcriptStore}.ts` | Snapshot cost follows current agent rows; worker retains latest state per agent and ref-counted ports, not every delta. Open views refetch authoritative transcripts and keep temporary reconciliation tails. Measure total work across views/tabs/reconnects, including phone attention reads, rather than one reducer's output. |
| Journal/control dispatch | `internal/runtime/{journal,control,chat}.go` | One conversation-owned FIFO/drainer spans generations: 256 pending observations plus one in flight, reservation before journal entry, cancellable admission, generation shedding and idle journal reclamation. Check actual event references/metadata and sink latency; no provider/controller callback under journal/agent locks. Pending domain recovery is not owned by this FIFO. |
| Sweeps and host starts | `internal/server/{messaging_loops,task_dispatcher,quota_dispatch,think_tanks,server}.go` | Bounded queries and process-wide admission slots belong to each producer. Mail, task/quota and room opening/progression limits are distinct; count overlapping admitted work, held slots, repeated sweeps and exits. A local cap or successful cleanup helper does not prove aggregate pressure or shutdown joining. |
| Generations, streams and processes | `internal/runtime/{registry,transport}.go`, `internal/runtime/terminal/ptyhub.go`, `internal/server/{launch,resume,switch,messaging_registration}.go` | Trace hook/MCP artifacts, native processes, subscriptions, buffers and cleanup closures to last owner across failed start, replacement, Stop, view detach and shutdown. PTYs drain with no viewer; slow viewers cannot own the producer's lifetime. Reuse ASW-3 process proof. |
| Native work/context growth | `internal/runtime/{chat,message,manifest}.go`, `internal/server/{primer,messaging_loops}.go`, actual ACP prompt/config/load callers | Count effect-producing native calls, supplied bytes and causal retries/activations across a chosen journey. Process manifests identify requested layers with explicit unknowns; they do not include every future tool result or prove model consumption. Reuse ASR-I's comparison design. |

Treat comments claiming a bounded path as hypotheses about the stated dimension. For example, the
HTTP window's bounded retained tail does not mean its reader seeks or avoids decoding old records;
the peer fingerprint scan is part of its correctness contract, not accidental work to remove blindly.
Follow the real caller to determine active-file, retained-file, snapshot and provider paths.

## Connected searches and useful workloads

These are facets of one review, not separately queued tasks or a required test matrix.

| Workload / search | Evidence that distinguishes normal growth from harmful amplification |
|---|---|
| One long conversation, fixed selected span, early/late/repeated page; same read after appends | Hold returned facts/page size constant while varying old history and page position. Count physical bytes/records, decode/render work, peak retained memory and service latency. Compare reference and direct peer semantics without weakening fixed-source/fingerprint rules. |
| Same total corpus distributed across more agents/tasks/archives | Separate individual-file length from retained object count and index/database size. Trace actual list/query/hydration/reconcile entry points and plans; determine which cost scales with live set versus all history. Do not infer a universal scan defect from an intentionally all-row administrative read. |
| Long read or history burst while another agent progresses and operator Cancels/Stops | Observe per-agent journal/control queues, shared I/O/bus/HTTP/SQLite pressure, control latency and actual guarded effect. Distinguish intended per-conversation backpressure from cross-agent starvation; preserve A2/A6's authority and receipt rules. |
| Open/collapse/navigate several views or tabs, reconnect, and return from phone background | Count streams/ports/subscriptions, hydration rows, transcript fetches/bytes and reconciliation tails before/during/after. Legitimate current-view catch-up must not become repeated whole-hub work or permanent duplicate retention. Actual device suspension claims need device evidence, not simulated visibility alone. |
| Repeated generations and failures, then quiescence or server shutdown | Compare ephemeral owner/resource counts and release paths after launch/resume/switch/crash/Stop. Allow retained history and exact unfinished cleanup. Include a slow sink or detached viewer only where it tests a distinct lifetime; check actual shutdown completion rather than assuming a shared cancellation context joins every goroutine. |
| Work waits at capacity while mail/task/quota/room signals recur | Count admitted workers, attempts, native starts and durable obligations over several sweeps. Separate each producer's retry policy from overlapping capacity; no lost hint/retry loop may silently multiply paid work. Reuse A2's admission evidence and current ASW/ASF owners. |
| A small delegated job repeatedly reads context, resumes, reports and continues | Relate actual submitted bytes/calls, tool reads, activation attempts and usage to the expected authorized work. Inspect Chuck's causal mechanism for extra turns/context, then reuse ASR-I's matched experiment if a quality/provider comparison is needed. Do not commission a general benchmark from prompt length alone. |

## Proportional measurement and evidence

1. Define a representative bounded development workload and expected outcome before running it.
   Record exact HEAD/dirty work, OS/host, build/FTS capability, installed versus source stack,
   provider/adapter versions when used, corpus shape and sizes, active/open-agent count, concurrency,
   cache state and whether a measurement includes provider time. Use disposable homes and known data.
2. Establish a baseline, then vary one useful axis at a time: retained bytes/records, object count,
   page position, active producers or views. Use a few increasing sizes and repetitions sufficient
   to distinguish a trend from noise; add a combined-pressure journey only to test a supported join.
   Cold/warm filesystem or database cache differences must remain explicit.
3. Pair timings with work/ownership observations: bytes read and decoded/rendered events, rows/query
   plan where relevant, allocations or retained memory, queue depth/in-flight owners, registrations/
   processes, hydration/fetch volume, and native submissions/available usage. Measure another
   agent's progress and actual intervention effect. Record unavailable counters honestly; do not
   invent precision or equate one sampled RSS/goroutine count with a leak.
4. Trace an observed trend to its current producer/consumer/lifetime. A deterministic source-backed
   violation of an explicit bound or forbidden effect can be a finding without a timing threshold;
   a scaling concern needs a credible workload and impact. Separate measured costs, causal source
   evidence, suspected mechanisms and optimization ideas. Absence of telemetry alone is not a defect
   unless a requirement demands it; explain what evidence remains unavailable.
5. Reuse existing tests/profiles/harness seams and temporary external measurement first. Follow
   workflow §7's permitted probes/reporting; introduce no product instrumentation, benchmark platform
   or exhaustive stress suite. A new check must discriminate a distinct consequential failure.
   Measurement informs a separately designed remedy, not an automatic cache/index/queue rewrite.

## Reuse these proofs before adding any

These tests were researched, not run or certified. Inspect actual assertions and applicable build
tags before reuse; count/response-bound checks are not throughput or long-run proofs.

| Seam | Existing starting proof and its limit |
|---|---|
| Fixed context/peer pages | `internal/contextref/service_test.go`: `TestReadPagesTraverseTheFixedSource`, `TestReadRejectsForgedCursorOffsets`; `internal/contextref/peer_test.go`: `TestPeerCursorRejectsSameBoundaryReplacement`. Correct traversal/rejection does not assert scan cost, cancellation latency or cross-path admission. |
| Transcript window | `internal/server/transcript_window_test.go`: `TestTranscriptWindowBoundsAndContinues`, `TestTranscriptWindowBoundsBytes`, `TestRemoteTranscriptIsAlwaysWindowed`. These bound/route output; they do not measure bytes decoded or impose a hard 1 MiB envelope on a single event. |
| Query/index/reconcile | `internal/state/pipeline_projection_test.go`: `TestPipelineDelegatedTasksUsesCreatorWindowsAndIndex`; `internal/index/indexer_fts_test.go`: `TestIndexerFTSLongTranscript`; `internal/server/reconcile_test.go`: `TestReconcileSessionPathTouchesOnlyChangedTranscript`. A targeted SQL plan, correct long-history search and unchanged-file skip do not prove other queries or file-page scaling. |
| Conversation FIFO | `internal/runtime/journal_test.go`: `TestControlObservationsSettleInCommitOrder`, `TestControlOrderSpansGenerationReplacement`, `TestCancelledControlRemainsBounded`. Synthetic sinks prove selected order/cancel/cap behavior, not live whole-server shutdown or aggregate multi-conversation cost. |
| Bus and PTY fan-out | `internal/bus/bus_test.go`: `TestPublishDropsOldestForSlowSubscriber`; `internal/runtime/terminal/ptyhub_test.go`: `TestPTYHubDrainsWithNoSubscriberThenReplaysScrollback`, `TestPTYHubDropsSlowSubscriber`, `TestPTYHubCloseUnblocksReaderAndSubscribers`. These isolate consumers/drain/close locally; they do not prove browser resnapshot or every runtime lifetime. |
| Start capacity and teardown | `internal/server/activation_test.go`: `TestMailActivationDefersWhenNoAdmissionSlotIsFree`, `TestShutdownWaitsForActivationResumeAndLeavesNoOrphan`; `internal/server/task_dispatch_test.go`: `TestTheBudgetLimitsRuntimesWithoutNarrowingReadiness`; `internal/server/messaging_registration_test.go`: `TestRegisterMessagingMCPWritesHTTPConfigAndCleanup`. Selected admission and cleanup paths do not establish aggregate caps, repeated-generation flatness or all goroutines joined. |
| Room opening concurrency | `internal/server/think_tanks_test.go`: `TestThinkTankOpeningsRunConcurrently`; `internal/server/think_tank_admission_test.go`: `TestThinkTankSlowSetupEndDeletion`. Expected parallel opening and compensation do not establish cumulative host pressure or cancellation across every room. |
| Actual browser consumers | `ui/src/api/sse.test.ts`, `ui/src/store/transcriptStore.test.ts`, `ui/src/store/agentStore.test.ts`, `ui/src/remote/connection.test.ts`: inspect shared-worker retained snapshot/reopen/port teardown and temporary transcript-tail assertions. Mocked events prove selected semantics, not browser connection-pool, device-suspension or representative memory/volume behavior. |

The highest-value new evidence joins a cost trend to a real resource/effect and another client's
ability to work. Existing functional suites narrow the search. Neither their test count nor a
single successful large response proves sustained capacity.

## Joins and finding ownership

Refresh the [fix ownership map](AGENT-SURFACE-FIX-UNITS.md), live handoff and intervening commits.
Reuse [ASR-I/J7](AGENT-SURFACE-REVIEW.md#asr-i--whole-system-synthesis-and-agent-performance):
its source synthesis is complete, while comparative performance and native adoption remain unproved.
Use its repeated matched native-versus-Chuck experiment and controls when needed; do not create
a competing benchmark program or claim awareness-off is a clean native baseline.

- Existing ASRC-6/ASRD-5/ASRE-2 retention/result-bound findings keep their assigned ASW/ASF owners.
  ASW-1 owns history/control recovery, including its current conversation FIFO fixes; ASW-2/3 are
  closed for frozen native execution/process ownership. Refresh exact closures rather than trusting
  old summary counts or interpreting an owed native/device receipt as a new bug.
- A1 owns durable truth/repair; A2 safe admission/retry/liveness; A3 usable environment/config;
  A4 faithful information/knowledge delivery; A5 result/progression authority; A6 action/observation
  truth. A7 owns sustained resource/work amplification and its development impact. Corroboration
  keeps the original finding owner; report a new mechanism only with distinct current evidence.
- Native/phone/terminal acceptance gates remain independent. A source/fixture timing cannot close
  provider adoption, model quality, physical suspension or real tmux/iTerm2 behavior. Record the
  concrete remaining experiment and owner when the environment cannot supply it.

Do not add global retention/deletion, a new context cache/index, universal scheduler/admission layer,
automatic replay, prompt rewrite, telemetry service or benchmark framework during review. A valuable
architectural remedy is a separately designed work item with measured rationale and preservation
requirements, not implementation in this unit.

## Required review result

Produce one concise cost/ownership ledger tied to exact reviewed HEAD: representative corpus and
workloads, returned versus processed data, active versus retained resources, baseline/scaling and
interference observations, native effect counts where relevant, causal source paths and proof limits.
Distinguish correct intentional growth, measured degradation, confirmed bound/lifetime violations,
unavailable evidence and proposals. State residual uncertainty; do not claim general absence of
degradation or worse model quality from source inspection or a small experiment.

Record real findings under **CR-A7** in the handoff using workflow §7: current location, credible
trigger/workload, development impact, requirement/invariant and reproducible evidence or precise
source-backed mechanism. Give the unit one Fix model recommendation based on its hardest open fix.
Do not invent an SLO or classify every linear scan as a Must-fix; explain the actual consequence.

Use isolated temporary Chuck/native homes and loopback-only servers, preserve user data/dirty work
and follow the review skill's allowed reporting rules. Change no product code or normative specs.
Close only this unit, retaining findings and honest native/device/measurement limits. Do not run
other audit units, fixes, the cross-audit integration journey or a release architectural-review
receipt as part of A7.
