# Serialize conversation event commits and separate control authority

**State:** Waiting to start
**Why:** Direct request on 2026-10-19 to formalize the systemic refactor proposals.
**Execution unit:** ASW-1; use `/work serialize-conversation-event-commits`.
**Owned findings:** ASRD-1, ASRD-2, ASRD-3, ASRD-5.
**Implementation model:** difficult — Codex Sol.
**Relevant requirements:** FS-03.R99/A78, TS-01.R39, TS-02.R52, TS-03.R73, TS-04.R100
(all planned); existing FS-01.R14, FS-13.R5/R10, TS-02.R8/R9/R14 and INV §1/§2/§4/§5/§7/§9/§11/§15/§16/§17.

## Outcome

One conversation owner gives live events, inactive annotations, notices and switch markers unique,
ordered history. Failed writes are explicit; successful task cleanup is not accidentally withheld
because its authority is a separate committed SQLite result.

## Included work

Implement TS-01.R39's registry-owned journal over the existing writer/indexer/hubs. Capture scope
before commit; serialize allocation, append/required sync and nonblocking retained-history
publication. Keep controllers/provider calls outside journal and agent mutexes; split the current
combined sink into history publication and bounded ordered control dispatch. History-only notices
can reenter without waiting on that dispatcher; terminal acknowledgement precedes held succession.
Preserve current streaming sync fences rather than adding per-delta fsync. Room capture requires a
committed canonical source; task release/quota cleanup use their independently durable records.

Route every current direct writer caller through this owner, including stopped annotations, peer/
quota notices, fork prefixes and switch markers; fence inactive append against resume/switch.
Retain file/event/SSE formats, bounded canonical-reader reconciliation and derived-index repair.
Expose bounded history health, suppress automatic successors on unknown history, and recover with
a durable gap notice without replaying native work. Preserve an already active switch target on
late marker failure, refresh actual state and report the existing truthful error envelope
(TS-03.R73); do not blindly roll back/restart it.

No generic event database, new workflow engine, blanket sink suppression or broad lock around
provider/controller callbacks. Process ownership changes remain ASW-3; input deduplication remains
ASF-2. Shared files are integration boundaries, not permission to close those findings.

## How we will know it works

FS-03.A78's barriers/failure matrix covers both switch directions followed by first prompt, two
inactive annotations, annotation/resume, provider/permission/notice interleavings, source generation
replacement, failed append/sync/index and canonical-only reconciliation. Assert canonical history
order equals history publication order, and control consumers settle in their required order.

Inject transcript failure after a task report is committed: exact pending-release cleanup converges.
Fail room source capture: it cannot publish/accept synthesis, but the exact attempt settles/holds
rather than admitting a successor, retaining failure/hold recovery if room-state persistence also
fails. Never replay an ambiguously committed room activity append; only its idempotent failure
transition is retried. An already captured partial contribution cannot finalize as accepted.
Exercise bounded dispatch and reentrant quota notice/control callbacks without
deadlock. Reproduce each original D finding in a durable regression before its correction.
Run TS-06's applicable closure matrix after the final implementation edit; do not claim native/
device gates from fixtures. Planned requirements become shipped only with their complete behavior.

## Waiting on

None for implementation. Coordinate overlapping runtime/sink edits with ASF-1/ASF-2/ASF-3.
Native/provider receipts remain explicit acceptance evidence, not a reason to invent another transport.
Architectural implementation receives the normal substantive review; closing its owned findings
does not close unrelated ASR-D findings or their live gates.
