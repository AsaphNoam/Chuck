import React from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import * as client from "../../api/client";
import type { AgentState, TranscriptEvent } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { foldTranscript, useTranscriptStore } from "../../store/transcriptStore";
import { RunningTasks } from "./RunningTasks";
import { collectTasks, markBackgrounded } from "./runtimeActivity";
import { toolLine } from "./toolLine";
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
const wire = (seq: number, type: string, data: Record<string, unknown>): TranscriptEvent => ({ agent_id: "a1", seq, type, ts: "2026-10-30T09:00:00Z", data });
const running = [wire(1, "tool_call", { tool_call_id: "tc", name: "Bash", args: { command: "npm run dev" } }), wire(2, "background_task_state", { task_id: "one", tool_call_id: "tc", state: "running", can_stop: true })];
function setup(events = running, live = true, control = true) {
  useAgentStore.setState({ agents: { a1: { agent_id: "a1", running: live, runtime_capabilities: { background_task_stop: control } } as AgentState } });
  useTranscriptStore.setState({ byAgent: { a1: foldTranscript(events) } });
  return render(<RunningTasks agentId="a1" />);
}
describe("running background work (FS-03.A83)", () => {
  it("names tools with the shared label and locates their call", () => {
    const reveal = vi.fn(); window.addEventListener("chuck-reveal-task", reveal);
    setup(); fireEvent.click(screen.getByRole("button", { name: "1 running" }));
    fireEvent.click(screen.getByRole("button", { name: "Ran npm" }));
    expect(reveal).toHaveBeenCalled(); window.removeEventListener("chuck-reveal-task", reveal);
  });
  it("waits for terminal publication after Stop", async () => {
    const stop = vi.spyOn(client, "stopBackgroundTask").mockResolvedValue({ accepted: true });
    setup(); fireEvent.click(screen.getByRole("button", { name: "1 running" }));
    fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(stop).toHaveBeenCalledWith("a1", "one", undefined);
    expect(screen.getByText("Stopping…")).toBeInTheDocument();
    useTranscriptStore.setState({ byAgent: { a1: foldTranscript([...running, wire(3, "background_task_state", { task_id: "one", state: "stopped" })]) } });
    await waitFor(() => expect(screen.queryByText("1 running")).toBeNull());
  });
  it("keeps a retryable Stop error and respects runtime authority", async () => {
    vi.spyOn(client, "stopBackgroundTask").mockRejectedValue(new Error("refused"));
    setup(); fireEvent.click(screen.getByRole("button", { name: "1 running" })); fireEvent.click(screen.getByRole("button", { name: "Stop" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Try again");
    expect(screen.getByRole("button", { name: "Stop" })).toBeEnabled();
    cleanup(); setup(running, true, false); fireEvent.click(screen.getByRole("button", { name: "1 running" })); expect(screen.queryByRole("button", { name: "Stop" })).toBeNull();
  });
  it.each(["session_meta", "fork_boundary"])("fences old work at %s", (kind) => {
    const events = [...running, wire(3, kind, { resumed_at: "later" })];
    expect(collectTasks(foldTranscript(events))[0].state).toBe("stopped");
    expect(toolLine(markBackgrounded(foldTranscript(events))[0]).state).toBe("background");
    setup(events); expect(screen.queryByText(/running/)).toBeNull();
  });
  it("hides stale work when the agent stops", () => { setup(running, false); expect(screen.queryByText(/running/)).toBeNull(); });
  it("dismisses with Escape after reopening", () => {
    setup();
    fireEvent.click(screen.getByRole("button", { name: "1 running" }));
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    fireEvent.click(screen.getByRole("button", { name: "1 running" }));
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(screen.queryByRole("dialog")).toBeNull();
  });
  it("orders two commands by announcement and includes the native child owner", () => {
    setup([...running,
      { ...wire(3, "activity_started", { name: "researcher" }), activity_id: "child" },
      { ...wire(4, "background_task_state", { task_id: "two", name: "go test", state: "running", can_stop: true }), activity_id: "child" },
    ]);
    fireEvent.click(screen.getByRole("button", { name: "2 running" }));
    const rows = document.querySelectorAll("[data-slot='task']");
    expect(rows[0]).toHaveTextContent("Ran npm");
    expect(rows[1]).toHaveTextContent("go test");
    expect(rows[1]).toHaveTextContent("researcher");
  });
});
