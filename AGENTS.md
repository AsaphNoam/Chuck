# AGENTS.md — Chuck

Chuck is a local Go server with an embedded React/Vite UI that supervises agent CLIs.

## Repository work

Use [shared startup](docs/features/AGENT-WORKFLOW.md#1-start-with-the-current-state) and the invoked role's launcher. Read only the linked sections needed at this stage. `/work`, `/review`, `/fix`, `/design-feature`, `/review-design`, `/usability-review`, `/investigate-bug` and `/release` require explicit invocation. Design, UX and test companions support authorized work; architectural review also runs before release. Outside an explicit role request, work only on what the human asked for.

Feature and technical specs define the product. The workflow governs process, the handoff records current state, and ideas or archives supply context. Read [spec authoring rules](docs/specs/README.md) when changing requirements. Use the [invariant index](docs/features/INVARIANTS.md#trigger-index) as directed by shared startup. Use the [completion checks](docs/features/AGENT-WORKFLOW.md#completion-checks) for the changed inputs.

- Preserve concurrent edits. Use the native patch/edit tool for source.
- Edit UI source in `ui/src`. Generate `internal/server/ui/dist` with `make embed`; keep the server loopback-only.
- Keep provider launchers in `.agents/skills` and `.claude/skills` identical.
- Before substantial exploration, assess whether focused delegation saves time, cost or context. Give independent agents bounded ownership, necessary evidence and verification criteria; integrate their results. Keep tightly coupled decisions together. Use the [model routing table](docs/features/MODEL-ROUTING.md) when choosing a model.

Answer directly. Prefer the simplest maintainable change. Propose valuable out-of-scope refactors separately with their cost and benefit before implementation.
