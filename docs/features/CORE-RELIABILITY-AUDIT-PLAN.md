# Chuck core reliability: proposed audit priorities

Prepared 2026-10-20; expanded after the human requested a deeper documentation-grounded view.
Three additional GPT-6 Luna readers mapped operator/workspace capabilities, coordination/control
planes and workflow/testing policy. The parent read the product object/control-plane model,
worktree/resources, federation/knowledge, task admission/waiting, room completion and CLI contracts,
and checked source entry paths. The committed baseline for this expansion is `f7809690`; concurrent
product fixes are outside this discovery. This supersedes the earlier two-investigations-plus-one-
journey recommendation, which compressed distinct core architectures too much.

**Status: recorded and deferred for human expansion/selection.** The purposeful testing and
architectural-review skills are implemented. This list starts no audit or product repair and
creates no executable audit queue. At the next selection, refresh current handoff/fix status;
concurrent fixes are landing. Hypotheses below are questions to investigate, not new findings.

The concrete workflow/skill proposal is in
[WORKFLOW-RELIABILITY-DESIGN.md](WORKFLOW-RELIABILITY-DESIGN.md). This document maps the audit
subjects that the architectural-review role should understand; the plan itself starts no review.

## What matters

Chuck must preserve development work, deliver the intended information to the intended agent,
carry messages and results faithfully, run work in the correct environment, and remain usable as
the working day and history grow. Give priority to repeatable normal-use failures, silent loss,
false success and growing overhead. Cosmetic problems, occasional display glitches and exotic
combinations without a credible development impact are lower priority.

A broken display becomes core reliability work when it hides lost input, a false completion,
missing recovery or a Stop that did not actually stop work. A disclosed, recoverable refusal can
be better than apparent success with the wrong conversation or missing context.

## What Chuck adds beyond a collection of agent terminals

FS-00.R3/R13–R15 define persistent sessions plus distinct conversation, context and control planes.
The detailed specs show why the audit must cover more than sending text:

| Shipped development capability | Architectural responsibility and authority | Governing docs |
|---|---|---|
| Launch the intended role/provider/model in a project and continue it later | JSON config/native source bindings compose into frozen session inputs; ordinary continuation and explicit refresh have different authority. | FS-01/FS-04/FS-08/FS-09; TS-01/TS-07 |
| Run parallel work on isolated branches with setup and shared project files | Worktree ownership lives in SQLite, readiness gates process start, and resources live outside the checkout. A project is the isolation unit; two agents in one project share its files. | FS-11/FS-19; TS-12 |
| Delegate durable work, wait for prerequisites, yield capacity and resume | Task assignment, dependency readiness, capacity reservation and runtime ownership are separate durable facts. Waiting retains the assignment; idle/exit never means task success. | FS-16; TS-10 |
| Coordinate without unnecessary model turns | Waking and deferred mail have different activation rules; mail, dependency and quota paths cannot borrow each other's retry policy. | FS-06/FS-09/FS-16; TS-01/TS-04/TS-10 |
| Turn selected evidence into actionable work | Annotations, attachments, immutable context references and accepted reports preserve source identity and access. References do not inherently inject content or wake a recipient. | FS-03/FS-13/FS-15; TS-02/TS-05 |
| Supervise ordered workflows and deliberate with several agents | A pipeline is a cursor over stage tasks; room participants remain ordinary agents; judge publication and accepted task results have distinct commit boundaries. | FS-14/FS-21; TS-09/TS-14 |
| Inspect, intervene and automate from a terminal as well as a dashboard | Shared domain mutations must preserve target identity, consent, revisions, receipts, partial failures, output bytes and truthful acceptance/completion status. | FS-01/FS-03/FS-22; TS-03/TS-06 |
| Run the same product after installing or updating it | The selected binary/private runtime, provider source, managed knowledge and durable home must agree; clean-source tests alone do not prove the installed stack. | FS-10/FS-18; TS-06/TS-11 |

Planned direct-action replacement, detached configuration import, general migration promises and
unproved provider compatibility are not assumed shipped. Feature `Partial` status requires reading
the individual requirement and its shipped/superseded notes. For example, pipeline named values
are opaque text, not verified/copied file artifacts; a completed room is not proof that its goal
was achieved. These limits are product contracts, not defects to invent.

## Prioritized audit subjects

Seven coherent slices follow different authorities; they are not seven tiny tasks or seven new
claims that all previous review work is useless. Begin each with existing proofs and fix ownership,
then challenge an unresolved architectural assumption at the combined current revision. Prior
source review narrows the search; it does not forbid a new source-level counterexample.

| Order | Slice / core question | Why this deserves capacity beyond the earlier plan |
|---|---|---|
| 1 | **A1 — Durable truth and recovery:** can accepted development work be reconstructed and continued? | Authority is split across normalized history, SQLite, provider sessions and maintenance paths; the fixed META case showed a real missed consumer. |
| 2 | **A2 — Admission, waiting and recovery:** will eligible work run, and will ineligible/stale work stay stopped? | Dependency readiness, runtime slots, assignment ownership, mail wake and quota recovery have different retry/liveness rules. A single conversation journey cannot establish them. |
| 3 | **A3 — Executable workspace and configuration:** does work start only in its intended, usable environment? | Worktree creation/recreation, setup, resources, native-source freshness and frozen continuations jointly define the real development environment. |
| 4 | **A4 — Information and operating knowledge:** do exact messages, selected sources and required Chuck capabilities reach the right consumer? | Content can be durable without being delivered; reference grants, annotation identity, mail confirmation and managed skill availability have different meanings. |
| 5 | **A5 — Supervision and result authority:** does the right accepted outcome drive the right next work? | Pipeline cursor, task result, descendant cleanup and room/judge publication cannot be treated as one generic completion event. |
| 6 | **A6 — Operator action and observation truth:** can a person or script reliably control and recover work? | CLI success/exit status, bulk failures, lost responses, stale targets and reconstructed live status affect whether development automation can trust Chuck. |
| 7 | **A7 — Sustained capacity and cost:** does the hub remain responsive without multiplying model work? | Retrieval scans are only one possible cost; control loops, stream consumers, registrations and unnecessary activations can amplify one another. |

Release/installed-runtime verification is cross-cutting in the proposed mandatory architectural
release review, rather than an eighth broad source audit by default. Phone polish remains lower
priority; permission/Stop/failed-delivery observations matter on every operator surface.

## Proposed audit scopes, after workflow decisions

### A1. Preserve saved work through repair and interruption

**Purpose:** challenge the storage architecture beyond the already repaired META-1 case. This is
the first fresh source audit because stored work is Chuck's recovery foundation.

**Scope:** normalized transcript, durable SQLite session inputs, native conversation identity,
archive visibility and the repair/reconciliation paths that connect them. Follow ordinary chat and
terminal Stop/restart, archive/restore and offline reindex. Check relevant migrations when they
change these fields; do not add a backup feature or audit every unrelated database table.

**Start:** `internal/transcript/{writer,reader}.go`, `internal/runtime/journal.go`,
`internal/index/{reindex,indexer}.go`, `internal/state/{session,running,manager}.go`,
`internal/archive/archive.go`, and `internal/server/{archive,reconcile,resume}.go`;
`internal/cli/reindex.go` is the repair entry point. Read FS-01/FS-03/FS-05/FS-07 and TS-01/TS-02
only for the relevant durable/derived and continuation contracts.

**Look for:** a path treating authoritative state as disposable; errors after a destructive reset;
partial/missing input causing more loss than the damaged record; loss of live-updated fields;
successful repair that leaves a conversation unresumable or silently rewound. Inspect the current
retention regression before proposing another field inventory: the META fix already did one.

**Useful proof:** capture known messages, native ID and live settings before the operation; compare
canonical content, archive readback and the native Resume target after it. Add one realistic
interruption at the destructive boundary and demonstrate recovery from durable inputs. Reuse
existing commit/index failure regressions. Never run destructive probes on the user's archive.

**Do not duplicate:** ASW-1 event failures, ASW-3 early hooks, or the closed META-1 finding. A new
finding needs a current counterexample, not an old failing overlay.

### A2. Admit, wait, wake and recover work correctly

**Purpose:** audit the control plane as a deterministic work system, including progress and safe
refusal. A task can be perfectly stored yet never run; a stale activation can run already-closed work.

**Start:** `internal/state/{tasks,task_waits,activations,quota_admission}.go`,
`internal/server/{task_dispatcher,messaging_loops,quota_dispatch}.go`,
`internal/runtime/{tasks,activation_kinds,registry_message}.go`. Read FS-06/FS-16 and
TS-10.R2–R6/R17–R19/R25–R35 plus the relevant FS-09 quota contracts.

**Challenge:** dependency completion while a task waits; capacity deferral versus consumed attempt
allowance; coalesced watched changes; created/woken versus borrowed runtimes; retained cleanup
claims; manual input/mail to a waiting assignee; restart reconstruction of capacity; cancellation
winning over pending wake. Durable rows are correctness and bus hints only acceleration: show what
ensures eventual progress when a hint is missed. Never apply mail's attempted-at-most-once policy
to a dependency activation that must remain actionable until confirmed start.

**Proof/ownership:** use actual committed task/activation/claim rows and native-submission counts,
then a bounded real-provider adoption case where needed. Check both no duplicate execution and no
permanently stranded work after the advertised repair. Reuse ASF-1/3 and ASW-1/3 regressions; any
new finding needs a different current trigger, not a second owner for their existing findings.

### A3. Start and continue in the correct usable workspace

**Purpose:** treat workspace readiness and execution configuration as one launch precondition.
This includes the code branch, dependencies and shared artifacts needed to do real development.

**Start:** `internal/server/{launch,resume,switch,worktree,pipeline_lifecycle}.go`,
`internal/worktree/git.go`, `internal/config/{resources,worktrees}.go`,
`internal/state/worktrees.go`, `internal/configsource`, and the backend/runtime composition seam.
Read FS-04/FS-08/FS-09/FS-11/FS-19 and TS-01.R6/R9, TS-07.R1–R8, TS-12.R1–R4.

**Challenge:** fork/create → setup → two concurrent starts; missing-checkout recreation from the
recorded branch; task/pipeline/wake paths versus manual launch; frozen cwd/resource paths after
project edits; fresh native resolution versus ordinary frozen continuation; failed source refresh
with a visible last-known-good cache. A cache shown in Settings must not authorize a launch that
requires fresh resolution. Project isolation is explicit: same-project agents share a checkout;
Chuck does not automatically merge branches or isolate each agent.

**Proof/ownership:** observe the real child's cwd, repository/branch, accessible resource location
and applied invocation, not only the composed struct. A recreated checkout does not restore lost
uncommitted files and must not be reported as such. Preserve ASW-2's closed selection fixes,
ASW-3's control owner and ASF-5's readiness/resource findings; target unproved combinations after
those fixes, not the same old setup reproduction.

### A4. Preserve information identity and usable operating knowledge

**Purpose:** audit information from selection/sending through storage, authorization, rendering
and provider use. Include knowledge needed to operate Chuck, not only the user's prompt text.

**Start:** `internal/contextref`, `internal/agentknowledge`,
`internal/server/{primer,attachments,attachment_recovery,launch}.go`,
`internal/messaging/{peer_tools,context_tools}.go`, `internal/state/messages.go`, and
`internal/runtime/{message,images,fork}.go`. `applyKnowledgeOverlay` in `launch.go` owns the shared
knowledge overlay. Read FS-03/FS-06/FS-13/FS-15/FS-18 and TS-11.R1–R5.

**Challenge:** wrong source/version/cwd in a reference; cursor gaps or silent clipping; altered
attachment bytes on retry; annotation-to-task losing its anchor; private data entering public room
activity; native history/primer adoption after handoff; old sessions receiving current managed
operating knowledge without contaminating frozen user configuration. Simulate unavailable knowledge
installation and verify the documented warning/no-advertisement degradation rather than a stale
path being advertised as usable.

Waking mail has an activation; deferred mail does not. Deferred content is supplied on the next
independently authorized turn, within bounds, or explicitly read; it is not generally pull-only.
Reference grants do not start turns or prove model memory. Direct peer inspection has its own
documented policy; do not invent confidentiality boundaries that the product does not promise.

**Proof/ownership:** compare exact source facts/revisions, consumer-decoded native inputs, message
receipts and a task that depends on the supplied facts. Message delivery is not comprehension.
Reuse ASR-A/B/D/E evidence and ASW-2 closure; ASF-2/3/4 retain input uncertainty, watched-context
and result-bound findings. Live skill/native-context adoption remains distinct from fake-wire proof.

### A5. Make supervision advance only from authoritative results

**Purpose:** inspect the pipeline/task/room join as an architecture, including cleanup and
human intervention. It is separate from A2's admission mechanics and A4's content delivery.

**Start:** `internal/pipeline/{manager,reconcile,actions,rooms}.go`,
`internal/state/{pipeline_tasks,tasks}.go`,
`internal/server/{pipeline_lifecycle,think_tank_capture}.go` and the room state/service/messaging
owners. Read FS-14/FS-16/FS-21 and TS-09.R35–R49/R51–R59, TS-10.R25–R35, TS-14.R4–R14.

**Challenge:** accepted report → matching turn end → runtime release → next cursor action;
descendant cancellation and attempt closure; preserved pending approval/finalization across restart;
room submission intent versus successful publication; partial openings without exposure to peers;
captured room activity versus unrelated private turns; judge result attached to the exact stage.
Named outputs are opaque text, not copied filesystem artifacts, and an idle agent is not a result.

**Proof/ownership:** inspect the immutable accepted result, current attempt/cursor, retained cleanup
intent and actual launched successor. A staged tool receipt or fluent judge answer does not prove
canonical publication or task acceptance. Reuse ASF-3 and ASW-1/3 ownership and existing room
fixtures; recheck combined changes and any remaining current assumptions. Owed real-provider
task/room behavior is not satisfied by repeating the source audit.

### A6. Make operator commands and observations trustworthy

**Purpose:** audit reliable control of a development hub from scripts, CLI and dashboard using
the same domain mutations. This is not a layout or visual-polish audit.

**Start:** `internal/cli/operator_*.go`, `internal/server/{sessions,archive_actions,task_handlers}.go`,
`internal/messaging/tool_result_contract_test.go`, the authoritative route/tool producers and
status/SSE reconciliation. Read FS-22.R1–R4 and TS-06.R34, plus each selected action's own contract.

**Challenge:** multiline/Unicode/stdin bytes; stable target identity; stale consent/revision/permission
targets; accepted-but-lost response; mixed bulk outcomes; interruption during follow/file output;
whether apparent idle/done hides a real permission wait, active execution or lost failure. CLI exit
zero may mean accepted action, not completed model work; control operations must expose that truth.

**Proof/ownership:** use the actual CLI/server or MCP wire, output streams/exit status and resulting
durable state, rather than a mirrored command table. Reuse existing independent-wire tests; a
route inventory is not itself a missing audit. ASF-2/6 and ASF-1 retain uncertain input, archive
selection and status/control findings. Extend only where current evidence leaves a consequential gap.

### A7. Measure overhead as real development accumulates

**Purpose:** find silent performance deterioration and repeated host-generated work. Start with
measurement before proposing a new cache, index, queue or architecture.

**Scope:** long transcript reads and context pagination first; then expand to shared I/O, stream
backpressure, accumulated registrations or retries only when measurements identify that path.

**Start:** `internal/contextref/{service,page,limits}.go`, `internal/transcript/reader.go`,
`internal/server/transcript_window.go`; follow `internal/runtime/journal.go`, `internal/bus/bus.go`
and messaging/dispatcher loops when they contribute to an observed stall. Read FS-15, TS-02/TS-03
and INV §16/§17; use ASR-I for the already-defined matched native comparison.

**Look for:** work proportional to all old history for every small page; repeated scans during a
single task; resource use that grows after agents stop; one slow consumer delaying unrelated agents;
unnecessary native submissions, tool loops or context growth introduced by Chuck.

**Useful proof:** pick a documented representative development workload and host, then vary history
size and concurrency one at a time. Count bytes/records scanned, native submissions and retained
resources; measure context-read latency and whether another agent can still progress and Stop.
Compare late and repeated pages with early pages. Separate provider time from Chuck's overhead.
Report the workload where a problem matters before choosing a fix or a performance budget; a
small returned page is not evidence of cheap retrieval, and one noisy timing sample is not a bug.

## Integration proof across selected audits

**Purpose:** establish the combined user-value chain after the existing fixes. This journey joins
the relevant audit results; it does not substitute for all seven different architectural questions.

**Scope:** a coordinator delegates a bounded repository task with explicit source facts and a file
or reference; a worker receives a peer message, reads the assignment, produces a result, and the
coordinator consumes that result to continue. Use Claude and Codex. Include correct worktree/cwd,
frozen execution settings and retained conversation identity in the observations.

**Start:** server `launch.go`, `resume.go`, `switch.go`, `primer.go`, `messaging_loops.go`,
`task_dispatcher.go`, `pipeline_lifecycle.go` and `think_tank_capture.go`; runtime `message.go`,
`registry_message.go`, `tasks.go` and `fork.go`; `internal/messaging/{peer_tools,task_tools}.go`,
`internal/state/{messages,activations,tasks,task_waits}.go`, `internal/pipeline`, and
`internal/contextref`. Read FS-06/FS-14–FS-18/FS-19/FS-21 and TS-09–TS-12/TS-14 only for the
selected journey. These paths locate the joins; they are not a requirement to reread all files.

**Useful proof:** write the expected facts and outcome before running. Record message/assignment
IDs, exact bodies and source revisions, actual native inputs, execution/cwd, native submission
counts, accepted results and the next durable action. A screenshot, a successful send, and text
visible only in Chuck's transcript are insufficient proof of delivery. Observe the native recipient;
use a task whose result depends on the supplied facts, while recognizing model output is not an
exact transport oracle.

Repeat the same small job at a few consequential boundaries: recipient busy/asleep; Stop/resume or
server restart after acceptance; and one handoff/context-compaction case. Use existing fixtures to
inject lost acknowledgements and count native submissions. Do not build a Cartesian matrix. Add an
ordinary pipeline and room-backed result variant only to check their distinct acceptance/recovery
boundary. Match existing ASR-I journeys and retain owed live gates when providers/devices are absent.

**Do not duplicate:** ASW/ASF finding-fix responsibilities. Validate the changed seams after fixes
or report a clearly new counterexample. Reuse ASR-E's source trace unless that new evidence gives
a reason to inspect further. No assumption that every message is automatically injected into every
active provider turn; verify the documented inbox/deferred-delivery behavior.

## Lower priority for this reliability-first round

- **Updater/rollback:** include installed-runtime and data-preservation evidence in the proposed
  architectural release review; a dedicated deep audit of pointer failures can follow the core
  control/data/workspace slices. It still needs recoverability and truthful failure
  after partial pointer changes. Inspect `Layout.Rollback`, not just `Activate`: activation already
  changes `previous` before `current`; rollback changes `current` first. No new defect is asserted.
- **Phone lifecycle and remote revocation:** retain existing gates and security candidates; they do
  not displace saved work, context or coordination in this round. A demonstrated active exposure
  or lost-work issue would change that priority.
- **Visual polish and exhaustive rare timing combinations:** spend capacity only when they expose
  missing work, misleading success or a credible repeated development failure.

## How to use this list later

After the human expands/selects this recorded plan, choose a small first tranche rather than commissioning every
possible audit. Start with A1's durable foundation and A2's deterministic work admission/recovery;
A3's executable workspace is the next distinct high-value architecture. A7 can collect measurements
independently. A4–A6 use prior findings and current changes to choose their highest-consequence
unproved joins. The integration proof spans them after the relevant repairs. This is a proposed
investment order, not chronological gating of independent role queues or automatic audit execution.

For each selected slice, record what was actually exercised, the current source/provider versions,
confirmed findings, and evidence still owed. Link new findings to their original owners where
appropriate. Judge success by preserved work, faithful delivery, truthful outcomes and measured
overhead; do not replace those observations with a count of files read or tests passed.
