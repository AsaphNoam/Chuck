# Pipelines that complete the goal without routine human intervention

**State:** In progress
**Why:** Operator request, 2026-10-30: single-click reusable processes such as Work → Review → Fix
must own ordinary routing, repair and recovery rather than repeatedly requiring human continuation.
Expands the former “Continue a blocked pipeline stage from its agent conversation” idea and the
WAIT-PROMPT investigation; input in the coordinator conversation must reach and continue its run.
**Relevant requirements:** FS-14.R97–R103/A61–A68; TS-09.R66–R72; TS-10.R29/R35;
TS-02.R6/R17; INV §1/§2/§5/§7/§8/§9/§10/§11/§15/§16/§17.

## Outcome

Start a saved process and leave it to work, review, repair and recover. The existing orchestrator
judges the next useful step; Chuck enforces explicit requirements and executes ordinary transitions.
Only a real question, permission/checkpoint or exhausted recovery needs attention. Exhaustion pauses
with preserved position and work so the user can fix the problem and resume the same run.

## Included work

- Decision-in-existing-result contract, required-check evidence and chronological visits supporting
  repeats, revisits and omission of unnecessary steps; no general graph editor or polling LLM.
- Durable input before assignment wake, conversation/run-page continuation, bounded recovery and
  visibly fresh same-runtime owner replacement with preserved history, useful work and pending input.
- Three automatic repair/revisit rounds per allowance, followed by a recoverable pause; explicit
  Resume/instructions can renew allowance without invented continuation text.
- New behavior for new runs of unchanged valid version-2 templates, optional required-check markers,
  retained explicit gates, and an additive visit backfill. One controller; no template conversion,
  adoption toggle, compatibility engine or destructive reset. Legacy wording can be regenerated if
  desired without building an instruction-conversion subsystem.
- Consistent template/builder/agent guidance, desktop/phone and API/CLI supervision. WAIT-PROMPT-1/2
  are included; WAIT-PROMPT-3's untraced refresh problem, permission-policy redesign and automatic
  workspace deletion are excluded. Existing lifecycle, project/workspace and Stop fences remain.

## How we will know it works

FS-14.A61–A68 and TS-09.R72 cover unattended Work → Review → Fix, required-check enforcement,
question/repair/resume, input receipts across restart, bounded exhaustion, visit/Stop races, old
template serialization and same-runtime replacement. Use meaningful task/controller/server boundary
tests and the scoped desktop/phone journeys, followed by TS-06's applicable closure matrix. A bounded
packaged Claude/Codex replacement probe distinguishes real handoff delivery from fake-provider
composition evidence; model judgment quality and production cost are not proved by fixtures.

## Waiting on

None. Behavior and the agent-facing contract were confirmed; implementation started 2026-10-30
(see the handoff's Active change).
