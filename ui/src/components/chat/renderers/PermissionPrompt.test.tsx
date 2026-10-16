import React from "react";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PermissionPrompt } from "./PermissionPrompt";
import { useAgentStore } from "../../../store/agentStore";

afterEach(cleanup);

describe("PermissionPrompt", () => {
  it("posts approve decisions", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response("{}", { status: 200 }));
    render(<PermissionPrompt agentId="a_1" event={{ kind: "permission_request", tool_call_id: "tc_1", name: "Bash" }} />);
    fireEvent.click(screen.getByText("Approve"));
    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    expect(fetchMock.mock.calls[0][0]).toBe("/api/sessions/a_1/permission");
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: "POST" });
    fetchMock.mockRestore();
  });

  const peerStop = {
    kind: "permission_request",
    tool_call_id: "chuck_peer_stop_1",
    name: "Stop Nova",
    reason: "it is looping",
    origin: "chuck_peer_stop",
    peer_stop: {
      requester: { agent_id: "a_req", name: "Atlas", project: "alpha" },
      target: { agent_id: "a_tgt", name: "Nova", project: "beta" },
    },
  };

  it("presents a peer Stop with both agents, the reason, consequences and Approve once", async () => {
    useAgentStore.setState({ agents: {} });
    let finish: (value: Response) => void = () => {};
    const fetchMock = vi.spyOn(globalThis, "fetch").mockReturnValue(new Promise((resolve) => { finish = resolve; }));
    render(<PermissionPrompt agentId="a_req" event={peerStop} />);
    expect(screen.getByText("Stop Nova")).toBeInTheDocument();
    expect(screen.getByText("Atlas · alpha · a_req")).toBeInTheDocument();
    expect(screen.getByText("Nova · beta · a_tgt")).toBeInTheDocument();
    expect(screen.getByText("it is looping")).toBeInTheDocument();
    expect(screen.getByText(/history is kept/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Approve once" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Approve once" })).toBeDisabled());
    expect(screen.getByRole("button", { name: "Deny" })).toBeDisabled();
    expect(JSON.parse(String(fetchMock.mock.calls[0][1]?.body))).toEqual({ tool_call_id: "chuck_peer_stop_1", decision: "approve" });
    finish(new Response("{}", { status: 200 }));
    fetchMock.mockRestore();
  });

  it("never offers a peer Stop whose requester is no longer waiting on it", () => {
    useAgentStore.setState({ agents: { a_req: { agent_id: "a_req", state: "idle" } as never } });
    render(<PermissionPrompt agentId="a_req" event={peerStop} />);
    expect(screen.queryByRole("button", { name: "Approve once" })).not.toBeInTheDocument();
    expect(screen.getByText("Cancelled")).toBeInTheDocument();
    useAgentStore.setState({ agents: {} });
  });

  it.each([
    ["approve", "Approved", "approved"],
    ["deny", "Denied", "denied"],
    ["cancelled", "Cancelled", "cancelled"],
    ["timeout", "Timed out", "timed-out"],
  ] as const)("renders %s as a distinct resolved chip", (resolved, label, state) => {
    const { container } = render(
      <PermissionPrompt agentId="a_1" event={{ kind: "permission_request", tool_call_id: "tc_1", name: "Bash", resolved }} />,
    );

    expect(screen.getByText(label)).toBeInTheDocument();
    expect(container.querySelector(".permission-prompt")).toHaveAttribute("data-state", state);
  });
});
