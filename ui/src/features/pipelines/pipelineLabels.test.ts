import { expect, it } from "vitest";
import { workspaceConflictLines } from "./pipelineLabels";

// FS-14.R21/R85: a busy project lists the conflicting runs by name first and
// groups same-named stage agents, capped so the confirmation stays in view.
it("lists runs first, groups same-named agents and caps the warning", () => {
  const agents = Array.from({ length: 21 }, (_, index) => ({ kind: "agent" as const, name: "Spec", id: `a${index}` }));
  const runs = ["Bulk 1", "Bulk 2", "", "Bulk 4", "Bulk 5"].map((name, index) => ({ kind: "run" as const, name, id: `r${index}` }));
  expect(workspaceConflictLines([...agents, { kind: "agent", name: "Solo", id: "s" }, ...runs])).toEqual({
    lines: ["Run: Bulk 1", "Run: Bulk 2", "Run: Untitled run", "Run: Bulk 4", "Run: Bulk 5"],
    hidden: 2,
  });
  expect(workspaceConflictLines(agents)).toEqual({ lines: ["21 agents named Spec"], hidden: 0 });
});
