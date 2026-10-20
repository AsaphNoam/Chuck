# Agent-surface implementation and fix units

Designed 2026-10-19 from the completed ASR-A–I reviews and
[systemic assessment](AGENT-SURFACE-FIX-STRATEGY.md). **Updated 2026-10-20:** all three architectural
units are implemented and independently reviewed, with findings still open. Their original coverage
IDs below are historical provenance, not a claim that those original reproductions remain open.
Current ASW closure uses ASW1-1–9, ASW2-1–11 and ASW3-1–6 in the handoff. See the
[post-refactor assessment](AGENT-SURFACE-POST-REFACTOR-ASSESSMENT.md) for the additional joins and
two new reproduced defects. No fix unit is currently in progress.
The six fix units below repair existing seams; their original evidence and severity remain in
[HANDOFF.md](HANDOFF.md#review-findings). This is an execution index, not a replacement specification.

## Selection and closure

The human explicitly requested cross-review consolidation on 2026-10-19. For this campaign only,
an ASF unit is an authorized §8 fix batch across its listed originating reviews. `/fix ASF-2`,
for example, selects exactly four finding IDs, not all of ASR-D/H. Read workflow §§8/10 and the
fix launcher normally; follow each original finding's validation, invariant and closure obligations.
No global launcher/workflow rule is changed. Original ASR/PROMPT IDs remain the evidence and
closure identities. An explicit old-review selection must first reconcile this ownership map;
do not duplicate an ASW refactor inside an ordinary fix batch. If the request is ambiguous, select
a listed ready unit or clarify its scope before implementation.

Mark only the selected unit in progress. Validate every finding against current source before
editing, reproduce it with the smallest meaningful regression and update requirements before any
behavior change or coverage completion. A passing test for one member never closes its siblings.
When a finding closes, record its fix commit/evidence at its original handoff entry and here;
remove its original review row only after every finding belonging to that review is closed.
Unrun native/device acceptance gates remain open with their evidence limits.

ASW work uses workflow §§1–6/10 and receives independent substantive review after implementation.
Do not label architectural implementation as administrative fix closure to evade that review.
Ordinary fix-closure bookkeeping follows §8 and is not a new implementation review unit.

## Ready units and priority

| Unit / invocation | Scope | Original coverage IDs | State / fix model |
|---|---|---|---|
| ASW-1 — `/fix ASW-1` (reviewed `43da6bec`) | One conversation commit and ordered control boundary (FS-03.R99, TS-01.R39, TS-02.R52, TS-03.R73, TS-04.R100) | ASRD-1, ASRD-2, ASRD-3, ASRD-5 | Reviewed; ASW1-1–9 open — difficult, Codex Sol |
| ASW-2 — `/fix ASW-2` (reviewed `9a15afa6`) | Frozen native contract, purpose-aware provider preparation and diagnostic provenance (FS-01.R53, FS-08.R37, FS-18.R21, TS-01.R40, TS-02.R53, TS-04.R101, TS-07.R19, TS-11.R24) | ASRA-1, ASRA-2, ASRA-3, PROMPT-1, ASRB-2, ASRB-3, ASRB-8, ASRG-4, PROMPT-2 | Reviewed; ASW2-1–11 open — medium, Codex Terra or Claude Opus |
| ASW-3 — `/fix ASW-3` (reviewed `3618df3e`) | Owned process/driver proof, launch claim and native terminal identity (FS-01.R54, FS-07.R25, TS-01.R41, TS-02.R54, TS-04.R102, TS-05.R28, TS-12.R11) | ASRA-4, ASRB-1, ASRB-4, ASRB-5, ASRB-6, ASRB-7, ASRC-3, ASRC-4, ASRC-7 | Reviewed; ASW3-1–6 open — difficult, Codex Sol; FS-07.A12 receipts owed |
| ASF-1 — `/fix ASF-1` | [Turn, status, permission and settings settlement](#asf-1--turn-and-control-settlement) | ASRC-1, ASRC-2, ASRC-5, ASRG-3 | Closed 2026-10-20 — `66c6a984` (ASRC-1), `60777c14` (ASRG-3), `ad08b386` (ASRC-2/5); live mid-turn setting receipt owed |
| ASF-2 — `/fix ASF-2` | [Input acceptance, uncertain delivery and exact recovery](#asf-2--input-acceptance-and-recovery) | ASRD-4, ASRH-1, ASRH-2, ASRH-4 | Waiting — medium, Codex Terra or Claude Opus |
| ASF-3 — `/fix ASF-3` | [Durable continuation and controller recovery](#asf-3--durable-continuation-and-controller-recovery) | ASRE-1, ASRF-1, ASRF-2, ASRG-1, ASRG-2, ASRG-5 | Waiting — medium, Codex Terra or Claude Opus |
| ASF-4 — `/fix ASF-4` | [Bounded completed work and retained task results](#asf-4--retention-and-result-bounds) | ASRC-6, ASRE-2 | Waiting — medium, Codex Terra or Claude Opus |
| ASF-5 — `/fix ASF-5` | [Project setup readiness and resource permissions](#asf-5--project-readiness-and-resource-permissions) | ASRA-5, ASRA-6 | Waiting — medium, Codex Terra or Claude Opus |
| ASF-6 — `/fix ASF-6` | [Certain CLI archive selection](#asf-6--cli-archive-selection) | ASRH-3 | Waiting — medium, Codex Terra or Claude Opus |

Suggested first picks: ASF-1 (read-loop stall), ASF-2 (possibly consumed input replay) and ASW-3
(unsafe process control). These are independently startable; no chronological queue gating is added.
Within ASW-3, close unsafe orphan signaling/containment first, then driver/native-ID work. Do not
leave a known unsafe path enabled while waiting for the largest native integration test.

## Architecture and integration boundaries

ASW-1 replaces separate allocation/write/publication paths with one registry-owned conversation
journal. It retains the current NDJSON log, index, hubs and controller services. It separates
committed history from ordered typed control observations, so failed history cannot become visible
canonical output or block independently committed task cleanup. It includes a bounded in-memory
control dispatcher, truthful history health and recovery, not a general event database.

ASW-2 extends the existing launch/composition contract. It freezes the effective model/setup choice
instead of reconstructing it from today's catalogue, and explicitly distinguishes fresh, resume,
native fork and isolated quota check. It preserves native instruction inheritance, adds bounded
requested-layer provenance and independently guards the source-adapter Steer contract. Arbitrary
unproved roots/profiles refuse before effects; supporting additional setup sources without changing
authentication/session ownership is a separate feature, not an implicit expansion of this fix.

ASW-3 extends existing process helpers, registry generation claims and TerminalDriver results.
A successful execution must have verifiable process/group/birth identity and exit observation.
External terminals use a short-lived internal command of the current Chuck executable with a
private one-use launch record, exact environment and foreground-TTY proof. Native SessionStart ID
persists at its authenticated generation boundary. No new resident supervisor is introduced.
Unknown legacy ownership refuses destructive control; confirmed death uses existing
FS-01.R20's `done` / `process exited` outcome, resolving ASRC-7 without a new product state.

The six ASF units use current turn, receipt, task/room/quota, retention, worktree and CLI seams.
They do not depend on completing an architectural project first. The substantive shared boundaries
are as follows; coordinate edits or rebase rather than expanding a unit's finding ownership:

- ASW-1 ↔ ASF-1: permission/turn-end events commit in order; the turn owner determines terminal
  reason and status. ASW-1 must not absorb cancellation policy or hold `as.mu` across callbacks.
- ASW-1 ↔ ASF-2: annotation command identity precedes append. Receipts reconcile exact canonical
  references; index degradation must not trigger duplicate native/mail/task effects.
- ASW-1 ↔ ASF-3: transcript outcome is separate from already committed task results and pending
  post-result actions. Room output requires committed source references; task release does not.
- ASW-2 ↔ ASF-2: artifact proof controls whether Steer is offered; receipt state controls whether
  possibly consumed Steer may run again. Both tests are required; neither substitutes for the other.
- ASW-2/3 ↔ ASF-1/3: use captured generation/turn, not a later registry lookup; preserve existing
  lifecycle admission, rollback and task/room ownership when preparing or replacing a runtime.
- ASW-3 ↔ ASF-5: setup child containment and project readiness share setup code but are distinct
  obligations. Keep the existing per-project readiness barrier; do not build a generic job manager.

## ASF-1 — Turn and control settlement

**Outcome:** live configuration cannot stall the ACP reader; only the owning turn/permission gate
can determine status, cancellation or permission settlement.

**Requirements:** FS-01.R52, FS-02.R3/R22, FS-03.R9/R45/R47/R49,
TS-01.R16, TS-04 permission/turn contracts; INV §1/§5/§8/§11/§15/§16.
**Entry seams:** `internal/runtime/{chat,permission,host_permission,quota}.go`,
`internal/server/session_config.go` and their fake-ACP tests.

1. **ASRC-1:** capture runtime identity/options under `as.mu`, call ACP unlocked as the existing
   `SetSessionMode` path does, and merge confirmed options only into the same generation. Bound the
   request to 30 seconds (a shorter caller deadline wins), independently of browser lifetime;
   release its lifecycle claim on all terminal paths.
   A timeout/late reply is unconfirmed applied state, not proof the old value remains effective.
   Preserve last confirmed persisted values and report partial confirmed changes plus an actionable
   existing runtime error for unknown application. Do not silently retry an uncertain setting;
   explicit Stop/Resume reapplies the confirmed contract. Do not invent an unsupported ACP getter.
2. **ASRC-2:** derive card state atomically from the current turn and pending permission ownership.
   A parallel/child tool frame cannot erase `waiting_input`; background output cannot set `busy`
   after that turn's terminal transition. Avoid a second independently writable status machine.
3. **ASRC-5:** extend the existing published/resolved gate to ACP permissions. Cancel/Stop/timeout
   cannot publish resolution before request, and missing native decision options resolve the exact
   pending request once as cancelled. No actionable card may survive a settled gate.
4. **ASRG-3:** claim cancellation under the turn gate and preserve that fact through same-turn late
   completion; terminal reason is cancelled and cannot confirm successful quota recovery. Preserve
   allowed held follow-ups after cancellation and existing inline-mail semantics.

**Verification/closure:** chunk and permission before config reply; no reply and late success;
same-generation persistence versus replacement; concurrent tool/request/resolve; Cancel before
request publication; missing approve/deny option; late completion after Cancel; native background
frames at finish. Assert submission/terminal counts, status, quota block/receipt and accessible
Stop/permission controls. Run focused runtime/server race checks and applicable TS-06 closure.
ASW-1 will later change event plumbing; these turn/control regressions must survive it unchanged.

## ASF-2 — Input acceptance and recovery

**Outcome:** lost acknowledgements preserve intent and consumption uncertainty. Retrying the same
annotation cannot duplicate its canonical record or native/mail/task effect; withdraw returns the
actual removed message and successful replies cannot erase a newer draft.

**Planned requirements:** FS-03.R100/A79, FS-13.R32/A22, TS-02.R55, TS-03.R74;
existing FS-03.R36/R48–R50/R90 and FS-13 delivery rules; INV §1/§5/§7/§8/§11/§15/§16.
**Entry seams:** runtime hold/Steer, server prompt/attachment/annotation handlers, state receipts,
desktop Composer/AnnotationTray, phone input and CLI uncertainty handling.

1. **ASRD-4:** generate/reuse one command ID before submission, including annotation-to-new-agent.
   Store exact target, payload hash, source event and effect references through existing receipts
   and SQLite transactions. Identical retry returns the recorded outcome; changed payload conflicts.
   Reconcile partial canonical/SQLite/native boundaries without pretending they are one transaction.
   Unknown effect stays recoverable; never blindly append/send again. Preserve room command authority.
2. **ASRH-1:** correlate successful Steer completion with the submitted draft revision. Clear that
   revision only; text typed during the request remains. Apply the same rule to media descriptors.
3. **ASRH-2:** give held input an authoritative `held_id`. Atomic withdraw checks the optional
   expected ID and returns the exact removed message; mismatch leaves the new hold unchanged.
   Restore server-returned text/attachments/request identity, never a cached copy from another device.
4. **ASRH-4:** definite pre-consumption refusal may restore its original hold. Lost/ambiguous Steer
   consumption becomes explicit recoverable input, never an automatic hold, even for legacy callers
   without IDs. Generalize existing request receipts to text; deliberate new Send is the only retry
   when consumption cannot be established. Legacy id-less requests get no new retry guarantee.

**Verification/closure:** lost successful response, identical retry after reload, changed-payload
conflict, partial annotation/mail/task/native effect and index-only failure; delayed Steer while
typing; two-device hold replacement/withdraw; promoted text Steer consumed before timeout and later
turn-end. Assert one canonical command/effect/provider submission, exact recovery ownership and no
cross-agent draft loss. Cover desktop, phone, CLI and HTTP; perform the applicable independent
browser/device checks without representing jsdom/fake ACP as native receipts. No broad UI redesign.

## ASF-3 — Durable continuation and controller recovery

**Outcome:** every accepted result, deferred continuation and cleanup retains its next recovery
owner. A transient bookkeeping failure cannot require repeating successful agent work.

**Requirements:** FS-16.R30–R34, FS-14.R66/R71/R72/R74/R77, FS-01.R42/R44/R49/R50,
TS-09.R35/R41–R43/R46, TS-10.R29–R34/R42/R45–R47, TS-14.R6, TS-02.R50,
TS-12.R4 (correct obsolete executor references); INV §1/§2/§4/§5/§7/§8/§9/§10/§11/§15/§16/§17.
**Entry seams:** task state and assignment read, task dispatcher, pipeline manager/reconcile,
quota dispatch/check/cleanup, room attempt abandonment. Extend their existing durable state;
do not introduce another scheduler or replay engine.

1. **ASRE-1:** retain an authorized bounded continuation snapshot through wait → notify → yield
   release → re-admit/confirm → assignment read. Include watched source IDs/revisions/outcomes and
   retained result context, including deleted-source truth. Confirmation cannot erase the context
   before the resumed execution can read it. Use existing task scope and read projection.
2. **ASRF-1:** clear post-result pending action only when next-stage, approval or final progression
   commits. Keep the exact action/owner through startup and live callback failure, using current
   dispatcher retry/backoff and attention states. Stop's fence wins; retry bookkeeping, never the
   accepted stage. No action may disappear merely because runtime release already succeeded.
3. **ASRF-2:** remove unused `LaunchStage`/`ContinueStage`/dependent `IsRunning` obligations and
   obsolete direct-executor tests; retain used `StopStage` cleanup. Prove frozen/applied fast and
   grouping through real run → stage task → task dispatcher. Update TS-12.R4's stale path description.
4. **ASRG-1:** prompt-preparation/status/start failure after quota's room `before` callback must
   abandon that exact room attempt/capture, using ordinary room dispatch's compensation seam.
   Failed compensation remains retryable; no published contribution or phantom running opportunity.
5. **ASRG-2:** share side-effect-free lifecycle/task/room/capacity eligibility with continuation;
   revalidate at admission before charging a quota check. Ineligible work defers without spending
   allowance, reserving a runtime or mutating its owner. Accepted owner controls cancel exact episodes.
6. **ASRG-5:** advance the bounded cleanup sweep fairly over failed, in-flight and cooldown rows.
   Use existing per-reference retry/backoff with a cursor or equivalent fair selection; no scan of
   unrelated native history. One permanently failing batch cannot starve later cleanable references.

**Verification/closure:** early/late assignment reads, multiple/deleted watched sources, restart
and scope isolation; next-task/final/approval write faults at runtime and restart, Stop racing
retry, exhausted retry attention; quota startup failure with zero prompts and exact room cleanup;
full task budget/lifecycle contention with zero checks charged; more than one cleanup batch with
permanent failures ahead of a healthy reference. Count native launches/prompts and accepted stage
results to prove repair never reruns work. Exercise the real task-dispatch path after deleting
obsolete executor tests; run state/server/pipeline race checks and TS-06 closure.

## ASF-4 — Retention and result bounds

**Outcome:** a long-lived orchestrator can keep using native children, and task results cannot
store unbounded generated output. Extend current retention/validation, not a new cache service.

**Requirements:** TS-04.R63, TS-10.R33; INV §2/§11/§16.
**Entry seams:** runtime subagent/tool-name retention; shared ordinary/stage task result validation
in `internal/state/tasks.go` and `pipeline_tasks.go`.

- **ASRC-6:** evict completed children while keeping bounded tombstones for late frames; never
  evict a live permission owner to make room. Prune tool-name entries with their existing tool
  inputs. Preserve parent/child attribution and harmless handling of late output/permissions.
- **ASRE-2:** validate named output values (64,000 runes each) and aggregate encoded result
  (256 KiB) before committing either ordinary or pipeline-stage result. Use the same shared typed,
  non-mutating refusal; declaration/required-output validation remains intact.

**Verification/closure:** at least 300 sequential native children, active-child pressure and late
frames/permissions, bounded map sizes; exact limits and one above, Unicode/JSON encoding overhead,
aggregate individually legal values, ordinary/stage parity. Rejection cannot finish a task or
partially persist a result. Run focused runtime/state/messaging checks and TS-06 closure.

## ASF-5 — Project readiness and resource permissions

**Outcome:** every launch sees completed checkout setup, and validation cannot widen an owner's
existing restrictive resource permissions. Both repair existing launch preconditions.

**Requirements:** FS-11.R9, FS-19.R3/R7, TS-02.R13, TS-05.R13, TS-12.R4/R5;
INV §1/§5/§10/§14/§15. **Entry seams:** `internal/config/resources.go` and
`internal/server/worktree.go` recreation/readiness coordination.

- **ASRA-5:** create new paths with required owner-only mode; preserve existing owner bits and
  tighten group/world exposure only as required. An existing `0500` directory that cannot support
  the requested write fails actionably; validation must not change it to `0700`.
- **ASRA-6:** extend the existing per-project in-flight recreation/setup barrier. Directory
  existence does not mean readiness. Starts arriving both before and after directory creation
  wait for the same setup completion, warning, failure or cancellation before provider launch.
  A cancelled waiter does not cancel another owner's setup unless existing ownership allows it.

**Verification/closure:** restrictive modes, group/world bits and permissive umask; simultaneous
starts bracketing directory creation, setup success/warning/failure/cancellation and one cancelled
waiter. Assert no provider prompt/start crosses unfinished setup and one setup owns the project.
Keep A4 process-tree tests in ASW-3; run config/server/worktree checks and TS-06 closure.

## ASF-6 — CLI archive selection

**Outcome:** bare CLI launch never interprets an unavailable/incomplete archive lookup as proof
that a new conversation is wanted. This is deliberately isolated from runtime architecture.

**Requirements:** FS-01.R11, FS-22.R1; INV §7/§8/§11/§16.
**Entry seams:** `internal/cli/launch.go`, `resume.go` and CLI HTTP fixtures.

**ASRH-3:** propagate lookup errors. Page with existing bounded API semantics until uniqueness or
absence is proved; if the bound is exhausted, require explicit selection or existing `--new`.
Do not infer no match from the first 200 rows or select a duplicate found on an incomplete page set.
An explicit `--new` keeps its established deliberate-new behavior.

**Verification/closure:** HTTP failure, match after row 200, duplicate across pages, no match after
complete traversal, pagination bound and explicit `--new`. Assert zero launch on uncertainty and
correct resume only on a unique complete result. Run CLI tests and applicable TS-06 closure.

## Coverage ledger

The original-coverage column above preserves the historical assignment: **41 findings, 41 assignments,
no duplicates** (4 + 9 + 9 + 4 + 4 + 6 + 2 + 2 + 1). It is not the current open-finding count.
The handoff's ASW findings supersede corrected reproductions and own the explicitly linked original
remainders; do not queue those aliases twice. By origin:

| Original review | Refactor/fix owners |
|---|---|
| ASR-A | A1–3/PROMPT-1 → ASW-2; A4 → ASW-3; A5–6 → ASF-5 |
| ASR-B | B2/3/8 → ASW-2; B1/4–7 → ASW-3 |
| ASR-C | C1/2/5 → ASF-1; C3/4/7 → ASW-3; C6 → ASF-4 |
| ASR-D | D1/2/3/5 → ASW-1; D4 → ASF-2 |
| ASR-E | E1 → ASF-3; E2 → ASF-4 |
| ASR-F | F1–2 → ASF-3 |
| ASR-G | G1/2/5 → ASF-3; G3 → ASF-1; G4 → ASW-2 |
| ASR-H | H1/2/4 → ASF-2; H3 → ASF-6 |
| Prompt investigation | PROMPT-2 → ASW-2; reopened PROMPT-1 is counted with ASR-A |

ASR-I's paired performance experiment, repository workflow-policy simplification and unrelated
native/device acceptance gates are evidence/recommendations, not additional confirmed fixes.
No prompt/persona rewrite or generalized performance claim is bundled into these units.
