# Chuck core reliability: audit plans and priorities

Prepared 2026-10-20; expanded after the human requested a deeper documentation-grounded view.
Three additional GPT-6 Luna readers mapped operator/workspace capabilities, coordination/control
planes and workflow/testing policy. The parent read the product object/control-plane model,
worktree/resources, federation/knowledge, task admission/waiting, room completion and CLI contracts,
and checked source entry paths. The committed baseline for this expansion is `f7809690`; concurrent
product fixes are outside this discovery. This supersedes the earlier two-investigations-plus-one-
journey recommendation, which compressed distinct core architectures too much.

**Status: all seven subjects mapped, one at a time as requested.** A1–A7 have researched review charters
and waiting review units, [CR-A1](CORE-RELIABILITY-REVIEW-A1.md),
[CR-A2](CORE-RELIABILITY-REVIEW-A2.md), [CR-A3](CORE-RELIABILITY-REVIEW-A3.md),
[CR-A4](CORE-RELIABILITY-REVIEW-A4.md), [CR-A5](CORE-RELIABILITY-REVIEW-A5.md),
[CR-A6](CORE-RELIABILITY-REVIEW-A6.md) and [CR-A7](CORE-RELIABILITY-REVIEW-A7.md), in the handoff.
No audit or product repair from these charters has started. Existing
remediation retains its own units. The purposeful testing and architectural-review skills are implemented.
Refresh current handoff/fix status before each review; concurrent fixes are landing. Hypotheses
below are questions to investigate, not new findings.

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

## Audit scopes and planning progress

A1–A7 are mapped into review-ready charters, each one independently selectable unit in the handoff.
The sections below retain their research baselines; refresh current code and fix ownership before
executing a review. This plan does not start one.

### A1. Preserve saved work through repair and interruption

**Purpose:** challenge the storage architecture beyond the already repaired META-1 case. This is
the first fresh source audit because stored work is Chuck's recovery foundation.

**Mapped 2026-10-20 — [CR-A1: Durable truth and recovery](CORE-RELIABILITY-REVIEW-A1.md).** Two
focused GPT-6 Luna researchers and parent integration mapped current authorities, shipped entry
points, governing requirements, reusable tests and failure-boundary questions against `97c67f39`.
One frontier-sized review follows conversation/session persistence through Stop/restart,
archive/readback/Restore, offline repair and native continuation. It distinguishes visibility
restoration from data recovery and status reconciliation from index rebuild. The charter includes
interrupted repair, damaged/missing input, legacy shapes and existing ASW/META ownership; these
are questions for the review, not confirmed new defects. No audit/probe or product fix ran.

### A2. Admit, wait, wake and recover work correctly

**Purpose:** audit the control plane as a deterministic work system, including progress and safe
refusal. A task can be perfectly stored yet never run; a stale activation can run already-closed work.

**Mapped 2026-10-20 — [CR-A2: Admission, waiting and recovery](CORE-RELIABILITY-REVIEW-A2.md).**
Two focused GPT-6 Luna researchers and parent integration mapped durable task/activation authority,
capacity/assignment/lifecycle distinctions, wait/yield, mail and quota policies, actual startup
paths and existing proof limits against `bdc13d02`. One frontier-sized review checks safe admission
and eventual progress together, including lost hints, interrupted starts and retained cleanup.
The charter distinguishes mail's attempted-at-most-once rule, task-owned confirmed-start retry,
quota authorization and cleanup-only reconciliation. Existing ASW/ASF owners remain; these are
questions for review, not confirmed new defects. No audit/probe or product fix ran.

### A3. Start and continue in the correct usable workspace

**Purpose:** treat workspace readiness and execution configuration as one launch precondition.
This includes the code branch, dependencies and shared artifacts needed to do real development.

**Mapped 2026-10-20 — [CR-A3: Executable workspace and configuration](CORE-RELIABILITY-REVIEW-A3.md).**
Two focused GPT-6 Luna researchers and parent integration mapped project/Git/ownership authority,
setup readiness, stable resources, fresh versus frozen configuration and actual provider setup
against `a312b597`. One frontier-sized review follows current callers through real process
cwd/branch/environment/access, including recreation after config edits, stale cache refusal and
consented deletion. It distinguishes completed setup failure warnings from unfinished setup,
recomposed environment from frozen inputs, and current pipeline/task dispatch from the obsolete
executor. Existing ASW/ASF owners remain; these are questions for review, not new defect claims.
No audit/probe or product fix ran.

### A4. Preserve information identity and usable operating knowledge

**Purpose:** audit information from selection/sending through storage, authorization, rendering
and provider use. Include knowledge needed to operate Chuck, not only the user's prompt text.

**Mapped 2026-10-20 — [CR-A4: Information and operating knowledge](CORE-RELIABILITY-REVIEW-A4.md).**
Two focused GPT-6 Luna researchers and parent integration mapped selected information through
accepted bytes/source identity, authorization, bounded reads/native input and delivery receipts
against `735ea6e5`. One frontier-sized review joins attachments/annotations/mail/context with
managed knowledge and truthful tool guidance. It distinguishes grants from injection, wake from
content receipt, awareness from availability, and requested manifests from native adoption;
annotation New task and durable task assignment retain their actual different paths. Current
ASW/ASF owners remain; these are review questions, not new defect claims. No audit/probe or fix ran.

### A5. Make supervision advance only from authoritative results

**Purpose:** inspect the pipeline/task/room join as an architecture, including cleanup and
human intervention. It is separate from A2's admission mechanics and A4's content delivery.

**Mapped 2026-10-20 — [CR-A5: Supervision and result authority](CORE-RELIABILITY-REVIEW-A5.md).**
Two focused GPT-6 Luna researchers and parent integration mapped accepted task reports, execution
settlement, descendant closure, retained progression and room/judge publication against `b58708c6`.
One frontier-sized review joins ordinary stages and room-backed stages through actual successor
work. It distinguishes task cleanup authority from canonical room-source requirements, concurrent
opening attempts from the discussion floor, published synthesis/read projection from stage
acceptance, and Continue/Retry/Replace identities. Existing ASW/ASF owners and live evidence limits
remain; no audit/probe or product fix ran. Refresh concurrent ASW-1 closure before review.

### A6. Make operator commands and observations trustworthy

**Purpose:** audit reliable control of a development hub from scripts, CLI and dashboard using
the same domain mutations. This is not a layout or visual-polish audit.

**Mapped 2026-10-20 — [CR-A6: Operator action and observation truth](CORE-RELIABILITY-REVIEW-A6.md).**
Two focused GPT-6 Luna researchers and parent integration mapped real operator input/guards,
acceptance/completion/ambiguity, CLI output, live state and snapshot/stream/phone recovery against
`4b8c1265`. One frontier-sized review joins actions to subsequent trustworthy observation, including
lost acknowledgements, partial bulk effects, pending controls and missed final updates. It preserves
non-atomic status limits, live versus durable state, receipt-specific retries and caller scope;
it does not reopen broad CLI parity or cosmetic review. Existing ASW/ASF owners remain; no
audit/probe or product fix ran. Refresh concurrent fix closure before review.

### A7. Measure overhead as real development accumulates

**Purpose:** find silent performance deterioration and repeated host-generated work. Start with
measurement before proposing a new cache, index, queue or architecture.

**Mapped 2026-10-20 — [CR-A7: Sustained capacity and cost](CORE-RELIABILITY-REVIEW-A7.md).**
Two focused GPT-6 Luna researchers and parent integration mapped actual reference/peer/window
reads, retained-data queries, snapshot/view fan-out, conversation queues, domain admission caps
and generation/resource teardown against `44d285fc`. One frontier-sized review starts with retrieval
growth and follows supported cost/lifetime joins into shared pressure, intervention and extra native
work. It separates bounded output from scan work, durable retention from temporary resource ownership,
and Chuck overhead from provider cost. The charter defines proportional measurement/evidence and
reuses existing proofs and ASR-I's matched comparison rather than creating a benchmark program.
Existing ASW/ASF owners remain; no audit, measurement/probe or product fix ran.

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

## How to continue

All seven researched charters are ready as separately selectable handoff review units. A7's
measurements can proceed independently of other review queues. The integration proof above spans
the selected architectures after the relevant repairs; it remains a cross-audit proposal, not an
eighth queued review or work started by this plan. The order is a proposed investment priority,
not chronological gating of independent role queues or automatic audit execution.

For each selected slice, record what was actually exercised, the current source/provider versions,
confirmed findings, and evidence still owed. Link new findings to their original owners where
appropriate. Judge success by preserved work, faithful delivery, truthful outcomes and measured
overhead; do not replace those observations with a count of files read or tests passed.
