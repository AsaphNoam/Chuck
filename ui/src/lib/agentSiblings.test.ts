import { describe, expect, it } from "vitest";
import type { AgentState } from "../api/types";
import { deriveAgentSiblings } from "./agentSiblings";
import { capVisible } from "./navCap";

function agent(id: string, patch: Partial<AgentState> = {}): AgentState {
  return { agent_id: id, name: id.toUpperCase(), project: "app", running: true, state: "idle", ...patch } as AgentState;
}

function record(...agents: AgentState[]) {
  return Object.fromEntries(agents.map((entry) => [entry.agent_id, entry]));
}

// FS-03.A76 / TS-08.R137: membership, order and the current-stopped rule.
describe("deriveAgentSiblings", () => {
  const agents = record(
    agent("a", { state: "busy" }),
    agent("s", { running: false, state: "done" }),
    agent("w", { state: "waiting_input" }),
    agent("x", { archived: true }),
    agent("i"),
    agent("o", { project: "other" }),
  );

  it("lists the project's running agents in dashboard order", () => {
    expect(deriveAgentSiblings(agents, "a")).toEqual([
      { id: "a", name: "A", state: "busy" },
      { id: "w", name: "W", state: "waiting_input" },
      { id: "i", name: "I", state: "idle" },
    ]);
  });

  it("keeps a stopped current agent, marked stopped, after the running ones", () => {
    expect(deriveAgentSiblings(agents, "s").map(({ id, state }) => [id, state])).toEqual([
      ["a", "busy"], ["w", "waiting_input"], ["i", "idle"], ["s", "stopped"],
    ]);
  });

  it("keeps an error stop's tone", () => {
    const failed = record(agent("a"), agent("f", { running: false, state: "error" }));
    expect(deriveAgentSiblings(failed, "f")[1]).toEqual({ id: "f", name: "F", state: "error" });
  });

  it("does not move entries on a state change", () => {
    const changed = { ...agents, a: { ...agents.a, state: "idle" } };
    expect(deriveAgentSiblings(changed, "a").map(({ id }) => id)).toEqual(["a", "w", "i"]);
  });

  it("yields nothing for a lone agent, an archived page or an unhydrated store", () => {
    expect(deriveAgentSiblings(record(agent("a"), agent("s", { running: false })), "a")).toEqual([]);
    expect(deriveAgentSiblings(agents, "x")).toEqual([]);
    expect(deriveAgentSiblings({}, "a")).toEqual([]);
  });
});

describe("capVisible", () => {
  const ids = ["1", "2", "3", "4", "5", "6", "7"];

  it("shows the first five and overflows the rest", () => {
    expect(capVisible(ids, (id) => id === "2")).toEqual({ visible: ["1", "2", "3", "4", "5"], overflow: ["6", "7"] });
  });

  it("swaps a hidden current entry into the fifth slot, keeping order", () => {
    expect(capVisible(ids, (id) => id === "7")).toEqual({ visible: ["1", "2", "3", "4", "7"], overflow: ["5", "6"] });
  });
});
