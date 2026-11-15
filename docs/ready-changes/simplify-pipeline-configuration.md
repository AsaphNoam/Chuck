# Simplify pipeline configuration

**State:** In progress
**Why:** The operator wants to configure a goal-led development workflow with fewer fields. A
stage's name and instructions should be enough for ordinary work, and explicit handoffs should
remain available when Chuck needs to enforce them.
**Relevant requirements:** FS-14.R110–R111/R113, FS-14.A72/A74, TS-08.R169–R170,
TS-09.R68/R71/R73, INV §2/§3/§5/§8/§11/§13/§17/§20.

## Outcome

Create and launch a multi-stage development pipeline using stage names, instructions, references
to project files and any completion or approval requirements. Chuck manages internal keys.

## Included work

- Simplify the existing desktop template editor. Surface configured requirements and use existing
  stage defaults. Offer explicit handoffs by source when needed and preserve existing templates.
- Use names and understandable fallbacks in ordinary pipeline surfaces and diagnostics. Keep
  exact proposal details available with their current confirmation and consumption behavior.
- Make run naming optional to edit and group equivalent runtimes in desktop Setup. Retain desktop
  Review and the phone's existing launch flow. Use the existing template schema, endpoints,
  workspace boundaries and execution contracts.

## How we will know it works

- Exercise FS-14.A72 in a browser. Author a Design → Implement → Review → Fix → Verify template
  using instructions and a file reference. Separately select a required handoff by source and
  exercise the existing missing-value refusal. Verify stable keys on rename, draft preservation,
  create-conflict handling and round trips for named bindings and Think Tanks. Check new room
  member labels and runtime assignment together.
- Exercise FS-14.A74 on desktop and phone, with configured defaults, a custom name, an invalid
  setup and a shared-checkout conflict. Compare serialized template and launch requests to the
  existing contracts and confirm Chucky proposals still require the exact approved configuration.
