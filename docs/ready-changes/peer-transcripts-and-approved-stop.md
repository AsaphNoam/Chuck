# Peer transcript reading and human-approved Stop

**State:** In progress
**Why:** Direct `/design-feature` request on 2026-10-16 to expose peer transcript reading familiar
from Codex/Claude within Chuck, and let agents stop peers. The operator confirmed cross-project
reads including stopped/archived history and all peer target types, required explicit user approval
for Stop, and retained existing local-process trust.
**Relevant requirements:** FS-06.R39–R44/A28–A32; FS-03.R91; FS-15.R18/A8;
FS-17.R22/A13; TS-04.R93–R95; TS-05.R27; TS-02.R49; TS-03.R71; TS-11.R23.
**Invariants:** INV §1/§2/§4/§5/§7/§8/§9/§10/§11/§14/§15/§16/§17.

## Outcome

Chuck chat agents discover durable peers and explicitly read their recorded conversations across
projects and archives. An agent may request Stop for any other agent, but the new tool cannot
terminate it until the operator approves that specific target execution. History remains intact,
and ordinary lifecycle and managed-work consequences apply.

## Included work

Extend the existing MCP authority with `read_agent_transcript` and `stop_agent`, optional bounded
history discovery, shared context-text projection/paging, host-owned permission waiters through the
existing operator decision endpoint, and attributed request/decision/effect history. Reuse current
desktop/phone permission presentation and operator CLI decisions. Refresh the embedded operating
guidance when shipping. Read outputs may enter ordinary caller tool history; no separate source
snapshot, approval database, grant, retention policy or background Stop retry is introduced.

Stop requires Approve once despite provider bypass/skip-permissions and Chuck tool exemptions.
Approval is bound to requester generation/turn and target execution. Local shell/API trust remains
unchanged: this feature does not provide process isolation or prevent a same-user process from
bypassing the tool. Exclude direct peer Resume/launch, self/group Stop, new Cancel behavior,
provider-native transcript import, raw terminal scrollback, attachment/file retrieval and the
paused direct-action transport migration.

## How we will know it works

FS-06.A28–A32 cover cross-project/archive reads, byte-bounded stable pages, unknown/empty/failed
reads, explicit approval despite permissive provider policies, denial/cancellation/reload/restart,
stale-generation and lifecycle races, storage failure/uncertain effect, and parity with ordinary
Stop for terminal/task/pipeline/Think Tank agents. FS-15.A8 checks independent grant revocation;
FS-17.A13 checks producer-derived outcomes and text/structured parity. Apply the relevant TS-06
closure matrix once after implementation, plus focused desktop/paired-phone journeys and a bounded
credentialed Claude/Codex host-gate receipt. Design-time code inspection and fake providers do not
count as those live receipts.

## Verified seams

- `internal/messaging/{messaging.go,tools.go}` and
  `internal/server/messaging_registration.go`: one generation-bound HTTP MCP authority; existing
  `list_agents` uses mail addressability, so history discovery must be opt-in.
- `internal/contextref/{service.go,render.go,page.go,limits.go}` and
  `internal/transcript/{reader.go,project.go}`: existing streaming transcript projection, 32 KiB
  UTF-8 text pages and 8 MiB physical-record bound. Expose a pure range-rendering seam without
  weakening reference authorization or creating a grant; keep assistant folding bounded too.
- `internal/runtime/{permission.go,chat.go}`: current pending permissions wrap ACP requests, and
  `onRequest` auto-approves registered Chuck tools or skip-permissions. The host waiter is a
  narrow extension of that single-decision mechanism, independent of adapter permission emission.
  No provider capability limitation or new adapter behavior is assumed.
- `internal/server/sessions.go::{handlePermission,stopAgent,stopAgentClaimed}`: existing operator
  decision route and shared lifecycle/quota claims. Recheck target identity inside those claims;
  the current `running` row already records generation/PID/start identity for orphan checks.
  Existing notice append/sync and replay paths provide the audit seam; no new approval journal.
- `ui/src/components/chat/renderers/PermissionPrompt.tsx` and
  `ui/src/remote/AgentScreen.tsx`: existing desktop/phone permission actions. TS-03.R71 extends
  their event handling and labels, without selecting a new visual layout.

## Waiting on

None. Product and trust-boundary decisions are confirmed. All listed new requirements remain
planned; this document authorizes no publication and records no implementation or live test result.
