import React from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import type { AgentState } from "../../api/types";
import { useAgentStore } from "../../store/agentStore";
import { ThinkTankSetupDialog } from "./ThinkTankSetupDialog";
import fixture from "./fixtures/room.json";

let created: FormData | null = null;
const server = setupServer(
  http.get("/api/projects", () => HttpResponse.json({ alpha: { title: "Alpha", cwd: "/tmp" } })),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => { cleanup(); vi.restoreAllMocks(); });
afterAll(() => server.close());

const agent = (id: string, name: string, extra: Partial<AgentState> = {}) =>
  ({ agent_id: id, name, project: "alpha", interface: "chat", model: "opus", effort: "high", state: "idle", archived: false, ...extra }) as AgentState;

beforeEach(() => {
  created = null;
  useAgentStore.setState({
    agents: {
      a_one: agent("a_one", "Ari"),
      a_two: agent("a_two", "Bea", { model: "gpt-6", effort: "medium" }),
      a_gone: agent("a_gone", "Olde", { archived: true }),
    },
  });
});

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter><ThinkTankSetupDialog open onClose={() => undefined} originProject="alpha" /></MemoryRouter>
    </QueryClientProvider>,
  );
}

// FS-21.A39 (R54, R56): the picker omits archived agents and names each
// choice's model and effort; Add new agent sits above it; chosen rows show
// model and effort; goal files go out as multipart parts after `room`.
describe("ThinkTankSetupDialog", () => {
  it("lists live agents with model and effort below a distinct Add new agent button", async () => {
    renderDialog();
    const picker = await screen.findByRole("combobox", { name: "Existing agent" });
    const options = Array.from(picker.querySelectorAll("option")).map((o) => o.textContent);
    expect(options.some((o) => o?.includes("Olde"))).toBe(false);
    expect(options).toContain("Ari (alpha) · opus · high");
    const add = screen.getByRole("button", { name: "+ Add new agent…" });
    expect(add.compareDocumentPosition(picker) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    fireEvent.change(picker, { target: { value: "a_two" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(screen.getByText("gpt-6 · medium")).toBeTruthy();
    expect(screen.queryByLabelText("Synthesis instructions")).toBeNull();
  });

  it("sends attached goal files with the room metadata", async () => {
    // The multipart body is inspected as sent; jsdom's FormData does not cross
    // into the node request interceptor.
    const realFetch = globalThis.fetch;
    vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
      if (String(input) !== "/api/think-tanks") return realFetch(input, init);
      created = init?.body as FormData;
      return new Response(JSON.stringify(fixture.room), { status: 201, headers: { "Content-Type": "application/json" } });
    });
    renderDialog();
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Cache" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Pick a cache" } });
    const picker = await screen.findByRole("combobox", { name: "Existing agent" });
    for (const id of ["a_one", "a_two"]) {
      fireEvent.change(picker, { target: { value: id } });
      fireEvent.click(screen.getByRole("button", { name: "Add" }));
    }
    fireEvent.change(screen.getByLabelText("Choose files to attach"), { target: { files: [new File(["# Brief"], "brief.md", { type: "text/markdown" })] } });
    expect(screen.getByRole("list", { name: "Attached files" }).textContent).toContain("brief.md");
    fireEvent.click(screen.getByRole("button", { name: "Start Think Tank" }));
    await waitFor(() => expect(created).not.toBeNull());
    const room = JSON.parse(String(created!.get("room")));
    expect(room.files).toEqual([{ part: "file_0", name: "brief.md" }]);
    expect(room.participants.map((p: { agent_id: string }) => p.agent_id)).toEqual(["a_one", "a_two"]);
    expect((created!.get("file_0") as File).name).toBe("brief.md");
  });
});
