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
    await waitFor(() => expect(Array.from(picker.querySelectorAll("option")).map((o) => o.textContent)).toContain("Ari (Alpha) · opus · high"));
    const add = screen.getByRole("button", { name: "+ Add new agent…" });
    expect(add.compareDocumentPosition(picker) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();

    fireEvent.change(picker, { target: { value: "a_two" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    expect(screen.getByText("gpt-6 · medium")).toBeTruthy();
    expect(screen.queryByLabelText("Synthesis instructions")).toBeNull();
  });

  // FS-21.A41 (R58): origin-project agents first; outside agents only on request,
  // an added outside row survives hiding them, and reopening resets the choice.
  it("lists the origin project's agents until other projects are included", async () => {
    server.use(http.get("/api/projects", () => HttpResponse.json({ alpha: { title: "Alpha", cwd: "/tmp" }, beta: { title: "Beta", cwd: "/tmp" } })));
    useAgentStore.setState({ agents: {
      a_one: agent("a_one", "Ari"),
      b_one: agent("b_one", "Cy", { project: "beta" }),
      b_two: agent("b_two", "Dee", { project: "beta" }),
    } });
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = (open: boolean) => (
      <QueryClientProvider client={client}>
        <MemoryRouter><ThinkTankSetupDialog open={open} onClose={() => undefined} originProject="alpha" /></MemoryRouter>
      </QueryClientProvider>
    );
    const { rerender } = render(view(true));
    const picker = () => screen.getByRole("combobox", { name: "Existing agent" }) as HTMLSelectElement;
    const names = () => [...picker().options].map((option) => option.textContent ?? "");
    await waitFor(() => expect(names().some((name) => name.startsWith("Ari"))).toBe(true));
    expect(names().some((name) => name.startsWith("Cy") || name.startsWith("Dee"))).toBe(false);

    fireEvent.click(screen.getByLabelText("Include other projects"));
    await waitFor(() => expect(names()).toContain("Cy (Beta) · opus · high"));
    fireEvent.change(picker(), { target: { value: "b_one" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    fireEvent.change(picker(), { target: { value: "b_two" } });
    fireEvent.click(screen.getByLabelText("Include other projects"));
    expect(picker()).toHaveValue("");
    expect(screen.getByText("Cy")).toBeTruthy();

    rerender(view(false));
    rerender(view(true));
    expect(screen.getByLabelText("Include other projects")).not.toBeChecked();
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

const savedTemplate = {
  id: "ttt_1", name: "Storage team", version: 1,
  setup: {
    openings: true,
    participants: [
      { launch: { role: "critic", backend: "claude", model: "opus", effort: "high", fast: false, name: "Cy" }, limit: 4, may_leave: false, instructions: "Argue for SQLite." },
      { launch: { role: "gone", backend: "claude", model: "opus", effort: "", fast: false, name: "Di" }, limit: 2, may_leave: true },
    ],
    judge: { role: "critic", backend: "claude", model: "opus", effort: "", fast: false, name: "" },
    judge_instructions: "Rank them.",
  },
};

function templateServer() {
  server.use(
    http.get("/api/roles", () => HttpResponse.json({ critic: { title: "Critic", system_prompt: "", skip_permissions: null } })),
    http.get("/api/think-tank-templates", () => HttpResponse.json({ version: 1, templates: [{ id: "ttt_1", name: "Storage team" }] })),
    http.get("/api/think-tank-templates/ttt_1", () => HttpResponse.json(savedTemplate)),
  );
}

// FS-21.A43–A45 (R60–R63): templates load into this project's editable draft.
describe("ThinkTankSetupDialog templates", () => {
  it("loads a template as fresh launches here, keeps the goal, flags an unavailable persona, and starts after repair", async () => {
    templateServer();
    let body: Record<string, unknown> | null = null;
    server.use(http.post("/api/think-tanks", async ({ request }) => {
      body = await request.json() as Record<string, unknown>;
      return HttpResponse.json(fixture.room, { status: 201 });
    }));
    renderDialog();
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Cache" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Pick a cache" } });
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    expect(await screen.findByText("Cy")).toBeTruthy();
    // X1: loading says rows are new agents, never silently replacing existing chats.
    expect(screen.getByText(/Each template participant starts as a new agent/)).toBeTruthy();
    expect(screen.getByLabelText("Title")).toHaveValue("Cache");
    expect(screen.getByLabelText("Goal")).toHaveValue("Pick a cache");
    expect(screen.getByLabelText("Instructions for Cy")).toHaveValue("Argue for SQLite.");
    expect(await screen.findByText(/Persona “gone” is no longer available/)).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "Start Think Tank" }));
    expect(screen.getByRole("alert").textContent).toContain("unavailable settings");
    fireEvent.click(screen.getAllByRole("button", { name: "Remove" })[1]);
    fireEvent.change(screen.getByRole("combobox", { name: "Existing agent" }), { target: { value: "a_one" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    fireEvent.click(screen.getByRole("button", { name: "Start Think Tank" }));
    await waitFor(() => expect(body).not.toBeNull());
    const sent = body as unknown as { openings: boolean; judge: { project: string }; judge_instructions: string; participants: { agent_id?: string; new?: { project: string; role: string; name: string }; limit: number; may_leave: boolean; instructions?: string }[] };
    expect(sent.openings).toBe(true);
    expect(sent.judge.project).toBe("alpha");
    expect(sent.judge_instructions).toBe("Rank them.");
    expect(sent.participants[0]).toMatchObject({ new: { project: "alpha", role: "critic", name: "Cy" }, limit: 4, may_leave: false, instructions: "Argue for SQLite." });
    expect(sent.participants[1]).toMatchObject({ agent_id: "a_one" });
  });

  it("asks before a template replaces setup edits and keeps them on cancel", async () => {
    templateServer();
    renderDialog();
    fireEvent.change(await screen.findByRole("combobox", { name: "Existing agent" }), { target: { value: "a_one" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    expect(await screen.findByText("Replace the current setup?")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByText("Replace the current setup?")).toBeNull());
    expect(screen.getByText("Ari")).toBeTruthy();
    expect(screen.getByLabelText("Template")).toHaveValue("");
  });

  it("saves an unstarted setup without a goal, resolving existing chats on the server", async () => {
    templateServer();
    let saved: Record<string, unknown> | null = null;
    server.use(http.post("/api/think-tank-templates", async ({ request }) => {
      saved = await request.json() as Record<string, unknown>;
      return HttpResponse.json({ id: "ttt_2", name: "Pair" }, { status: 201 });
    }));
    renderDialog();
    const picker = await screen.findByRole("combobox", { name: "Existing agent" });
    for (const id of ["a_one", "a_two"]) {
      fireEvent.change(picker, { target: { value: id } });
      fireEvent.click(screen.getByRole("button", { name: "Add" }));
    }
    fireEvent.change(screen.getByLabelText("Instructions for Bea"), { target: { value: "Be skeptical." } });
    fireEvent.click(screen.getByRole("button", { name: "Save setup…" }));
    fireEvent.change(screen.getByLabelText("Template name"), { target: { value: "Pair" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Saved “Pair”.")).toBeTruthy();
    const sent = saved as unknown as { name: string; setup: { participants: { agent_id?: string; instructions?: string }[] } };
    expect(sent.name).toBe("Pair");
    expect(sent.setup.participants).toMatchObject([{ agent_id: "a_one" }, { agent_id: "a_two", instructions: "Be skeptical." }]);
    expect(screen.getByRole("button", { name: "Start Think Tank" })).toBeTruthy();
  });

  it("keeps a turn limit edited while a template was still loading", async () => {
    templateServer();
    let release!: () => void;
    const gate = new Promise<void>((resolve) => { release = resolve; });
    server.use(http.get("/api/think-tank-templates/ttt_1", async () => { await gate; return HttpResponse.json(savedTemplate); }));
    renderDialog();
    fireEvent.change(await screen.findByRole("combobox", { name: "Existing agent" }), { target: { value: "a_one" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    fireEvent.click(await screen.findByRole("button", { name: "Replace setup" }));
    fireEvent.change(screen.getByLabelText("Turn limit"), { target: { value: "27" } });
    release();
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(screen.getByLabelText("Turn limit")).toHaveValue(27);
    expect(screen.queryByText("Cy")).toBeNull();
  });

  it("does not restore a template that answers after returning to blank setup", async () => {
    templateServer();
    let release!: () => void;
    const gate = new Promise<void>((resolve) => { release = resolve; });
    server.use(http.get("/api/think-tank-templates/ttt_1", async () => { await gate; return HttpResponse.json(savedTemplate); }));
    renderDialog();
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "" } });
    release();
    await new Promise((resolve) => setTimeout(resolve, 20));
    expect(screen.queryByText("Cy")).toBeNull();
    expect(screen.getByLabelText("Template")).toHaveValue("");
  });

  it("flags a saved effort or fast mode the model no longer supports and blocks Start", async () => {
    templateServer();
    server.use(http.get("/api/backends", () => HttpResponse.json({
      version: 2,
      backends: { claude: { name: "Claude", type: "claude-acp", default_model: "opus", models: { opus: { name: "Opus", model: "opus", efforts: ["low", "medium"] } } } },
      backend_support: { "claude-acp": { chat: { available: true, effort: true, fast: true } } },
    })));
    const fastJudge = { ...savedTemplate, setup: { ...savedTemplate.setup, judge: { ...savedTemplate.setup.judge, fast: true } } };
    server.use(http.get("/api/think-tank-templates/ttt_1", () => HttpResponse.json(fastJudge)));
    renderDialog();
    fireEvent.change(screen.getByLabelText("Title"), { target: { value: "Cache" } });
    fireEvent.change(screen.getByLabelText("Goal"), { target: { value: "Pick a cache" } });
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    expect(await screen.findByText(/high effort is no longer available for opus/)).toBeTruthy();
    expect(screen.getByText(/Fast mode is no longer available for opus/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Start Think Tank" }));
    expect(screen.getByRole("alert").textContent).toContain("unavailable settings");
  });

  it("loads further template pages on request", async () => {
    templateServer();
    server.use(http.get("/api/think-tank-templates", ({ request }) => {
      const cursor = new URL(request.url).searchParams.get("cursor");
      return HttpResponse.json(cursor === "p2"
        ? { version: 1, templates: [{ id: "ttt_old", name: "Older setup" }] }
        : { version: 1, templates: [{ id: "ttt_1", name: "Storage team" }], next_cursor: "p2" });
    }));
    renderDialog();
    await screen.findByRole("option", { name: "Storage team" });
    expect(screen.queryByRole("option", { name: "Older setup" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Show more templates" }));
    expect(await screen.findByRole("option", { name: "Older setup" })).toBeTruthy();
    await waitFor(() => expect(screen.queryByRole("button", { name: "Show more templates" })).toBeNull());
  });

  it("counts participant instructions in Unicode characters up to the server's limit", async () => {
    renderDialog();
    fireEvent.change(await screen.findByRole("combobox", { name: "Existing agent" }), { target: { value: "a_one" } });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));
    const field = screen.getByLabelText("Instructions for Ari") as HTMLTextAreaElement;
    fireEvent.change(field, { target: { value: "😀".repeat(8000) } });
    expect(Array.from(field.value)).toHaveLength(8000);
    fireEvent.change(field, { target: { value: "😀".repeat(8001) } });
    expect(Array.from(field.value)).toHaveLength(8000);
  });

  it("resends one save command after a lost response and rotates it once acknowledged", async () => {
    templateServer();
    const ids: string[] = [];
    server.use(http.post("/api/think-tank-templates", async ({ request }) => {
      ids.push((await request.json() as { command_id: string }).command_id);
      return ids.length === 1 ? HttpResponse.error() : HttpResponse.json({ id: "ttt_2", name: "Pair" }, { status: 201 });
    }));
    renderDialog();
    const picker = await screen.findByRole("combobox", { name: "Existing agent" });
    for (const id of ["a_one", "a_two"]) {
      fireEvent.change(picker, { target: { value: id } });
      fireEvent.click(screen.getByRole("button", { name: "Add" }));
    }
    fireEvent.click(screen.getByRole("button", { name: "Save setup…" }));
    fireEvent.change(screen.getByLabelText("Template name"), { target: { value: "Pair" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText(/could not be saved/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByText("Saved “Pair”.")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Save setup…" }));
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(ids).toHaveLength(3));
    expect(ids[1]).toBe(ids[0]);
    expect(ids[2]).not.toBe(ids[0]);
  });

  it("returns focus to the row when its launch editor is cancelled by keyboard", async () => {
    templateServer();
    renderDialog();
    await screen.findByRole("option", { name: "Storage team" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "ttt_1" } });
    fireEvent.click(await screen.findByRole("button", { name: "Edit Cy" }));
    const editor = await screen.findByRole("dialog", { name: "Edit Cy" });
    fireEvent.keyDown(editor, { key: "Escape" });
    await waitFor(() => expect(screen.getByRole("button", { name: "Edit Cy" })).toHaveFocus());
    expect(screen.getByLabelText("Instructions for Cy")).toHaveValue("Argue for SQLite.");
  });
});
