# Agent-surface refactors — pre-fix systemic assessment

2026-10-20, source `eaaf1c1f` (product state through `9a15afa6`). Inputs: all three ASW
implementation reviews in [HANDOFF.md](HANDOFF.md#review-findings), the original
[fix units](AGENT-SURFACE-FIX-UNITS.md), and targeted current-source traces. Four pre-existing
dirty ready-change files were outside this assessment. No product code or requirement was changed.

## Conclusion

The three refactors remain the right boundaries. The new findings mainly expose incomplete
ownership and enforcement inside those boundaries; another wholesale runtime/configuration rewrite
is not justified. Merely introducing a journal, contract or process-proof type did not make all
callers obey it. The highest-value fixes make the final effect conditional on the original owner
and retain that owner when an operation refuses or partially fails.

Two additional defects were reproduced and recorded as **ASW1-9** and **ASW3-6**. They extend
the existing ASW fix units rather than creating more queue items. The current primary closure work
is 26 ASW review findings plus the original 19 findings in ASF-1–6; originating remainders such as
ASRB-1 and PROMPT-1 are aliases/provenance, not additional independent fixes.

## 1. Ordered observation is not ordered, correctly owned cleanup

ASW1-6/7 require one bounded conversation dispatcher across generations. That correction is
necessary, but insufficient: `server.go` reads the current registry generation for a queued
terminal event, then starts task cleanup in another goroutine. That work is no longer serialized
by the FIFO. `PendingReleaseTask`/`PendingYieldTask` resolve only agent/generation, and pipeline
lookup intentionally selects the latest stage for that pair. Neither describes which execution
the old terminal event ended.

**ASW1-9 adds a demonstrated consequence:** injecting an old observation into the actual installed
server callback stops the current task-created generation while its fake-ACP turn remains held.
An old-turn observation carrying the same generation has the same result. These are controlled
callback-injection reproductions, not a captured live-provider replacement incident.

Carry the immutable event identity through settlement, resolve the exact task assignment/reporting
execution, and recheck ownership under the lifecycle claim before destructive cleanup. Retain
existing durable release/yield obligations. Reuse the generation-guarded stop pattern already used
by room cleanup; do not invent another scheduler.

**Do not fix this by making all task cleanup synchronous in the FIFO.** A cleanup can Stop a
runtime, which emits/waits for another terminal observation on that same queue. That creates a
self-wait. The terminal handoff must bind the exact eligible cleanup obligation before returning;
asynchronous destructive work must retain that identity and never resolve "whatever is current."
The precise existing task execution-handle/turn association must also cover legacy missing identity
without guessing. Same-generation successive borrowed assignments deserve an explicit regression.

## 2. Process identity, group containment and teardown are separate facts

ASW3-1/2/4/5 should be corrected as one control contract, not independent error-return patches:

- Unknown proof retains the registry claim, runtime handle, running row, registration and view.
  Returning an error after deleting those owners is too late.
- An unreadable identity is not an exit. A dead/reaped leader is not proof that its descendants
  exited. Neither fact authorizes blindly signaling a historical numeric process group.
- An exit-channel pointer is not continuing ownership after Wait reaps the child.
- A successful Stop is the commit point for releasing ownership, not the intent to stop.

**ASW3-6 is an additional common-primitive flaw:** the classifier accepts birth mismatches up to
and including one second. The Darwin reader supplies microsecond birth identity, while even the
Linux reader's normalization does not justify treating different reconstructed births as the same
process. A test-owned child with a deliberately incorrect birth offset by 250 ms was terminated.
No actual OS PID reuse or unowned-process signaling was attempted. Fix exact/platform-precision
comparison before using the classifier as the oracle for the other containment regressions.

Leaderless orphan recovery remains the difficult design point. Preserve continuity evidence or
refuse uncertain cleanup; a numeric PGID alone cannot replace it. If that requires a new persistent
supervisor or a changed containment policy, propose that decision separately rather than quietly
enlarging ASW-3. Real tmux/iTerm2 receipts remain required.

## 3. A frozen record is useful only if every purpose realizes it

ASW2-1/2/3/5/6/11 are related contract consumers, not six unrelated configuration bugs. Continuation
must validate realizability without refreshing it; clone must carry the whole source policy,
including permissions; stopped checks must use the continuation's selection. Distinguish frozen
intent, last confirmed applied settings and newly negotiated values.

Use the existing shared contract/composition seam to enforce these decisions for all purposes.
Do not add a second configuration manager or fix each caller with another list of copied fields.
ASW2-6 is an early priority inside that unit: changing a persona default must not make Clone skip
approvals that its source required.

ASW2-4/7 show why instruction fidelity cannot be established by sanitizing one convenient field:
native global files, base instructions and effective root-marker precedence are separate inputs.
Use independent pinned-native fixtures, or refuse isolation/merges that cannot be established.
Do not route around this with a new auth home or unapproved setup copying.

ASW2-8–10 should derive bounded diagnostic hashes from the final assembled request/configuration,
including terminal delivery and available artifact evidence. A manifest built from earlier inputs
can repeat the same omission as the request builder. Manifest equality is not evidence that the
provider adopted identical instructions; output-quality comparisons are still premature.

## 4. Degraded history needs operational control, not synthetic history

ASW1-1/2 and ASF-2 share an intent boundary: an accepted hold must remain recoverable, and a live
permission must remain actionable when its history append fails. A notification that opens a
conversation with no controls does not satisfy that boundary.

Project pending controls from their current generation-owned runtime authority for client reload
and desktop/phone parity. Keep failed records out of canonical history. Distinguish this live
projection from obligations that must survive a server restart; a universal durable permission
or control-event database is not required.

Restore held input only on definite pre-consumption refusal. Possibly consumed Steer stays explicit
uncertainty. Otherwise an ASW1-1 "restore on any error" patch can undo ASF-2's duplicate-prevention
work. Preserve exact held IDs, text, attachments and request identity without overwriting newer input.

## 5. Clear a recovery obligation only after its exact transition commits

ASW1-4, ASRF-1 and ASRG-1 express the same rule at different domain boundaries: a failed room
capture, accepted stage result or started quota attempt still needs an owner for the next action.
An in-memory flag/queue cannot be the only restart authority, and deleting the flag before a
failure write commits can later turn failure into apparent success.

Retain the exact room failure until its idempotent fail/hold commits. Separately committed task
results may still release their runtimes. Pipeline recovery retries post-result bookkeeping, never
the accepted provider work; quota startup compensates its exact room attempt. Use the existing
domain rows and dispatcher repair, not a generic transaction/workflow engine. Repairing the
journal's reserved watermark alone does not repair any of these controller obligations.

## Queueing and joint closure checks

Keep the three existing ASW fix units and six ASF batches. Do not create one fix job per symptom.
Suggested priorities and integration constraints, without chronological queue gating:

1. **ASW-3 control safety** (including ASW3-6), **ASF-1 read-loop/control liveness**, and
   **ASF-2 uncertain-input replay** remain immediate priorities. Do not delay the small unlocked
   settings call behind a larger ownership repair. Give overlapping registry/runtime changes a
   single integration owner rather than running conflicting edits without coordination.
2. **ASW-1** must join attachment lifetime, shared bounded dispatch, exact terminal identity and
   monotonic room failure. ASW1-5's small watermark correction can land early, but it does not
   close those other obligations. Coordinate its hold/control projection with ASF-1/2.
3. **ASW-2** can proceed independently with current-source coordination. Its permission, legacy
   selection and check-isolation fixes take precedence over diagnostic completeness.
4. **ASF-3** can start at existing task/controller seams; integrate its retry behavior only with
   ASW1-4's monotonic failure fence and ASW1-9's exact execution ownership. ASF-4/5/6 remain
   independently selectable bounded fixes.

Add joint regressions to closure, beyond each original reproducer:

- Delayed old terminal callback → replacement generation with pending result; repeat within one
  generation across distinct assignments. Only the originating execution may be settled/stopped.
  Stop-triggered terminal emission must not deadlock its own control dispatcher.
- Unknown process proof → refused Stop/watch recheck → recovered proof → safe retry. Verify row,
  registry, registration and view retention, exact small birth mismatches, and leaderless children.
- Journal failure while a hold and permission exist → reload on desktop/phone → exact recovery
  or one approval; no invented history, lost draft or automatic uncertain replay.
- Failed capture plus failed attempt/hold writes → storage recovery → repeated terminal settlement;
  never accept the failed attempt, replay activity or block independent task-result cleanup.
- Block old callback at queue capacity → exit/replace/cancel → late frames. Verify one dispatcher,
  bounded pending work, writer reclamation and exact durable recovery ownership.
- Change catalogue/persona/native setup after launch → Resume/Clone/stopped check. Verify full
  source policy and actual outgoing selection, then compare final manifest hashes with that request.

These joins strengthen existing acceptance; they are not evidence that the full combined native
journeys have run. Broad passing suites did not exercise the failure timing that the reviews found.

## Evidence

Targeted source review and two temporary Go-overlay probes against current product source:

```sh
env -u CHUCK_RUNTIME_ROOT GOCACHE=/private/tmp/chuck-asw2-gocache GOFLAGS=-p=1 \
  go test -overlay=/private/tmp/chuck-systemic-20261020-overlay.json \
  ./internal/runtime ./internal/server -run '^TestSystemic' -count=1 -timeout=60s
```

`TestSystemicNearBirthMismatchCannotAuthorizeControl` fails on all four ±250 ms/±1 s comparisons
and confirms termination of its own mismatched-proof child. The server probe's first run hit a
sandbox loopback-bind refusal; the fixture was changed to avoid a listener. The subsequent actual
callback probe fails both `old_generation` and `old_turn_same_generation` by stopping the held
current reporter. It reads the installed callback via reflection in the temporary fixture; no
production callback or state is replaced. Probes remain in `/private/tmp/chuck-systemic-20261020-*.go`
with their overlay for the fix agents to port. No credentialed provider, browser/device or real
external terminal was exercised; no full-suite success is claimed for these intentional failures.
