# Chuck core reliability: proposed audit priorities

Prepared 2026-10-20 from the human's reliability priorities, the
[defect-origin research](AGENT-SURFACE-META-ANALYSIS.md), current source through `fc4a1b33`, and
the existing ASR/ASW/ASF evidence. One focused GPT-6 Luna reader challenged overlap with previous
audits; the parent checked the architecture map, current fixes and source entry paths.

**Status: proposed and deferred.** Finish defining the workflow/skill changes first. This list
does not start an audit, create executable review units, change requirements, or authorize product
repairs. At the next audit selection, refresh the current handoff and fix status; concurrent fixes
are landing. Hypotheses below are questions to investigate, not new findings.

## What matters

Chuck must preserve development work, deliver the intended information to the intended agent,
carry messages and results faithfully, run work in the correct environment, and remain usable as
the working day and history grow. Give priority to repeatable normal-use failures, silent loss,
false success and growing overhead. Cosmetic problems, occasional display glitches and exotic
combinations without a credible development impact are lower priority.

A broken display becomes core reliability work when it hides lost input, a false completion,
missing recovery or a Stop that did not actually stop work. A disclosed, recoverable refusal can
be better than apparent success with the wrong conversation or missing context.

## Architecture map

These are six connected responsibilities, not six new broad source reviews. Existing evidence
and findings must be reused. The three proposed work slices below group the remaining gaps.

| Priority / responsibility | User-visible promise | Important failure to search for | Evidence still worth obtaining / existing owner |
|---|---|---|---|
| **1. Conversation storage and recovery** | My conversation and the ability to continue it survive normal Stop, restart, archive/restore and maintenance. | Text exists in one store but disappears from another; repair destroys the only copy of an identity or setting; a partial read makes intact work inaccessible; an interrupted rebuild cannot recover. | A fresh audit of storage/repair boundaries, using current fixes as baseline. ASW-1 owns event commit/history failures; ASW-3 owns native identity/lifecycle. **META-1 is fixed**, including ID/effort/fast retention; do not reopen its old reproduction. |
| **2. Context that actually reaches the agent** | The agent receives the intended instructions, files, references and conversation context for this task. | The UI/history looks complete while the provider gets missing, stale, duplicated or wrong-project material; a handoff or resumed assignment drops facts; clipping is silent. | Post-fix native input captures plus a small real-provider task. Reuse ASR-A/B/D and ASW-2; do not repeat their configuration inventory. ASF-3 owns missing watched-change context; ASF-2 owns input/attachment uncertainty. Provider compaction need not preserve every historical byte; test the promised context and disclose limits. |
| **3. Agent-to-agent messages** | A message reaches the right recipient with its intended body, sender and references, and delivery status tells the truth. | Mail is marked read/delivered too early, stays stranded after wake/restart, goes to the wrong generation, or retries cause repeated work. | A post-fix delivery journey, not another full source audit: ASR-E already traced transactional mail, budgets, activation and native delivery without another confirmed mail finding. Test receipt/consumption uncertainty explicitly; do not invent an exactly-once transport guarantee. |
| **4. Tasks, pipelines and Think Tank outcomes** | Accepted assignments and results remain actionable; successors run only from the correct accepted outcome. | A controller says done without a valid result, forgets a result after restart, launches a stage twice, or loses the pending action needed to continue. | Verify interactions after ASF-1/3/4 and ASW-1/3 fixes; preserve their finding ownership. Trace task/attempt/result identity through one ordinary pipeline and a bounded room-backed variant. Real task-reporting adoption remains an existing owed gate. |
| **5. Correct execution and workspace** | The right agent continues the right conversation in the right repository/worktree with the intended settings, and Stop controls that execution. | A resume silently starts fresh, work lands in the wrong checkout, defaults replace frozen choices, or delayed cleanup stops another execution. | Use ASW-2/3 and ASF-5 regressions as baseline; join project setup, delegation, resume and archive/restore in the integration journey. Existing real tmux/iTerm2 gates remain separate. A new generic process/worktree review is not justified solely by this list. |
| **6. Performance over a real working day** | Growing history, several active agents and repeated context reads do not progressively stall development or multiply unnecessary work. | Every late context page rescans old history; a slow reader blocks agent progress; queues/caches retain old executions; extra prompts or retries inflate latency and cost. | Fresh measurement, starting with the concrete unmeasured retrieval-scan hypothesis. ASW-1 queue-bound defects already have an owner. ASR-I's native-versus-Chuck comparison remains unproved; reuse that gate, with matched provider/model/context, rather than infer model degradation from host latency. |

## Proposed work slices, after workflow decisions

### A. Preserve saved work through repair and interruption

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

### B. Measure overhead as real development accumulates

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

### C. Verify one complete development job across agents

**Purpose:** establish the combined user-value chain after the existing fixes. This is one
integration audit, not separate source re-reviews of context, mail, tasks, rooms and lifecycle.

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

- **Updater/rollback:** still worth a later operational audit of recoverability and truthful failure
  after partial pointer changes. Inspect `Layout.Rollback`, not just `Activate`: activation already
  changes `previous` before `current`; rollback changes `current` first. No new defect is asserted.
- **Phone lifecycle and remote revocation:** retain existing gates and security candidates; they do
  not displace saved work, context or coordination in this round. A demonstrated active exposure
  or lost-work issue would change that priority.
- **Visual polish and exhaustive rare timing combinations:** spend capacity only when they expose
  missing work, misleading success or a credible repeated development failure.

## How to use this list later

Finish the workflow decisions, then select a small first tranche rather than commissioning every
possible audit. A and B can be independent; C consumes relevant fix results and existing evidence.
The sequencing is a proposed investigation plan, not a change to independent role queues.

For each selected slice, record what was actually exercised, the current source/provider versions,
confirmed findings, and evidence still owed. Link new findings to their original owners where
appropriate. Judge success by preserved work, faithful delivery, truthful outcomes and measured
overhead; do not replace those observations with a count of files read or tests passed.
