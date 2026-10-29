import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { RunDetail } from "./RunBrowser";
import { RunStartForm } from "./RunStartForm";
import { summarizeProposal } from "./proposalSummary";
import { WorkspacesPage } from "./WorkspacesPage";

// Mirrors internal/server/pipeline_workspace_handlers.go and state.PipelineWorkspace: optional
// fields are omitted when empty, never null.
const template = { version: 2, title: "Delivery", orchestrator_role: "implementer", inputs: [], stages: [{ id: "work", title: "Work", objective: "Do the work.", coordination: "standing", inputs: [], outputs: [] }] };
const workspace = {
  workspace_id: "ws_1", run_id: "run_1", project: "app", project_name: "App", run_name: "Ship", mode: "new_worktree",
  repo_path: "/tmp/app", base: "main", base_commit: "abc123", branch: "chuck/ship", cwd: "/tmp/app-ship", state: "ready", owned: true,
  revision: 3, removal_eligible: true, created_at: "2026-10-29T00:00:00Z", updated_at: "2026-10-29T00:00:00Z",
};
const wsDetail = { ...workspace, missing: false, dirty: false, dirty_known: true, removable: true };
const controls = { continue: { eligible: false, reason: "" }, retry: { eligible: false, reason: "" }, replace: { eligible: false, reason: "" }, stop: { eligible: false, reason: "" }, repair_cleanup: { eligible: false, reason: "" } };
const run = { run_id: "run_1", template_id: "delivery", template_snapshot: template, display_name: "Ship", project: "app", goal: "Ship it", inputs: {}, assignments: {}, orchestrator: { backend: "codex", model: "m" }, dedicated_assignments: {}, state: "completed", revision: 9, pending_action: "", current_stage_id: "work", current_task_id: "", current_attempt_id: "", current_agent_id: "", attention_reason: "", final_outcome: "success", created_at: "2026-10-29T00:00:00Z", updated_at: "2026-10-29T00:00:00Z" };
const runDetail = (extra: object = {}) => ({ run, template, inputs: {}, assignments: {}, orchestrator: run.orchestrator, dedicated_assignments: {}, stage_tasks: [], values: [], diagnostics: [], controls, ...extra });

let starts: Array<Record<string, unknown>> = [];
let removals: unknown[] = [];
let listQueries: string[] = [];
let currentDetail: Record<string, unknown> = wsDetail;
let removeStatus = 200;

const server = setupServer(
  http.get("/api/pipelines", () => HttpResponse.json([{ id: "delivery", template, valid: true, diagnostics: [] }])),
  http.get("/api/projects", () => HttpResponse.json({ app: { title: "App", cwd: "/tmp/app", color: [1, 2, 3], add_dirs: [], context_prompt: "" } })),
  http.get("/api/backends", () => HttpResponse.json({ version: 2, backends: { codex: { name: "Codex", type: "codex-acp", default: true, default_model: "m", models: { m: { name: "M", model: "m", fast: false } } } } })),
  http.get("/api/config", () => HttpResponse.json({ default_project: "app" })),
  http.post("/api/pipeline-runs", async ({ request }) => {
    starts.push(await request.json() as Record<string, unknown>);
    return HttpResponse.json({ error: { code: "validation_failed", message: "run cannot start", details: { diagnostics: [{ field: "workspace.branch", code: "exists", message: "branch \"chuck/ship\" already exists; choose a new branch name" }] } } }, { status: 422 });
  }),
  http.get("/api/pipeline-workspaces/ws_1", () => HttpResponse.json(currentDetail)),
  http.post("/api/pipeline-workspaces/ws_1/remove", async ({ request }) => {
    removals.push(await request.json());
    if (removeStatus !== 200) return HttpResponse.json({ error: { code: "conflict", message: "checkout changed since you reviewed it" } }, { status: removeStatus });
    return HttpResponse.json({ ...wsDetail, state: "removed" });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => { cleanup(); server.resetHandlers(); starts = []; removals = []; listQueries = []; currentDetail = wsDetail; removeStatus = 200; });
afterAll(() => server.close());

const client = () => new QueryClient({ defaultOptions: { queries: { retry: 0 } } });

function renderRun(detail: object) {
  server.use(http.get("/api/pipeline-runs/run_1", () => HttpResponse.json(detail)));
  return render(<QueryClientProvider client={client()}><MemoryRouter><RunDetail runID="run_1" onDeleted={() => {}} /></MemoryRouter></QueryClientProvider>);
}

describe("run start workspace", () => {
  // FS-14.R91: omission is the project checkout; a rejected branch keeps what was typed.
  it("sends workspace only for New worktree and keeps every value after a workspace diagnostic", async () => {
    render(<QueryClientProvider client={client()}><RunStartForm onStarted={() => {}} /></QueryClientProvider>);
    await screen.findByRole("option", { name: "Delivery" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "delivery" } });
    fireEvent.change(screen.getByLabelText("Run goal"), { target: { value: "Ship it" } });
    const start = screen.getByRole("button", { name: "Start run" });
    await waitFor(() => expect(start).toBeEnabled());

    fireEvent.click(start);
    await waitFor(() => expect(starts).toHaveLength(1));
    expect(starts[0]).not.toHaveProperty("workspace");

    fireEvent.click(screen.getByRole("radio", { name: /New worktree/ }));
    expect(screen.getByText(/shares one new checkout of this project on a new branch/)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: "Start run" })).toBeDisabled());
    expect(screen.getByText("Enter a branch name for the new worktree to continue.")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Branch"), { target: { value: "chuck/ship" } });
    fireEvent.change(screen.getByLabelText("Base (optional)"), { target: { value: "develop" } });
    expect(screen.getByLabelText("Base (optional)")).toHaveAttribute("placeholder", "Project's base branch");
    fireEvent.click(screen.getByRole("button", { name: "Start run" }));

    await waitFor(() => expect(starts).toHaveLength(2));
    expect(starts[1].workspace).toEqual({ mode: "new_worktree", branch: "chuck/ship", base: "develop" });
    expect(await screen.findByText(/already exists; choose a new branch name/)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Branch/)).toHaveValue("chuck/ship");
    expect(screen.getByLabelText(/^Base \(optional\)/)).toHaveValue("develop");
    expect(screen.getByLabelText("Run goal")).toHaveValue("Ship it");
    expect(screen.getByLabelText(/^Branch/)).toHaveFocus();
  });

  it("names the choice in the review step and in a proposal summary", async () => {
    render(<QueryClientProvider client={client()}><RunStartForm stepMode onCancel={() => {}} onStarted={() => {}} /></QueryClientProvider>);
    await screen.findByRole("option", { name: "Delivery" });
    fireEvent.change(screen.getByLabelText("Template"), { target: { value: "delivery" } });
    fireEvent.change(screen.getByLabelText("Run goal"), { target: { value: "Ship it" } });
    fireEvent.click(screen.getByRole("radio", { name: /New worktree/ }));
    fireEvent.change(screen.getByLabelText("Branch"), { target: { value: "chuck/ship" } });
    await waitFor(() => expect(screen.getByRole("button", { name: "Review" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Review" }));
    expect(screen.getByText("New worktree · branch chuck/ship")).toBeInTheDocument();

    const fields = summarizeProposal({
      proposal_id: "p1", digest: "d", created_at: "2026-10-29T00:00:00Z", kind: "start_run",
      payload: { request_id: "r", template_id: "delivery", display_name: "Ship", project: "app", goal: "Ship it", inputs: {}, orchestrator: { backend: "codex", model: "m", effort: "", fast: false }, dedicated_assignments: {}, workspace: { mode: "new_worktree", branch: "chuck/ship", base: "develop" } },
    }, undefined);
    expect(fields).toContainEqual({ label: "Workspace", value: "New worktree · branch chuck/ship from develop" });
  });
});

describe("run detail workspace", () => {
  it("shows the checkout kind, branch, path, state and failures", async () => {
    renderRun(runDetail({ workspace: { ...workspace, state: "error", error: "git worktree add failed", setup_ok: false } }));
    const section = await screen.findByRole("region", { name: "Workspace" });
    expect(within(section).getByText("New worktree")).toBeInTheDocument();
    expect(within(section).getByText("chuck/ship")).toBeInTheDocument();
    expect(within(section).getByText("/tmp/app-ship")).toBeInTheDocument();
    expect(within(section).getByText("Needs attention")).toBeInTheDocument();
    expect(within(section).getByText("git worktree add failed")).toBeInTheDocument();
    expect(within(section).getByText("Project setup did not finish")).toBeInTheDocument();
  });

  it("parses a run that predates workspaces and offers no workspace or removal", async () => {
    renderRun(runDetail());
    expect(await screen.findByRole("heading", { name: "Ship" })).toBeInTheDocument();
    expect(screen.queryByRole("region", { name: "Workspace" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Remove checkout/ })).not.toBeInTheDocument();
  });

  it("withholds removal for a running run, a project checkout and a removed checkout", async () => {
    const { unmount } = renderRun(runDetail({ run: { ...run, state: "running", final_outcome: "" }, workspace }));
    await screen.findByRole("region", { name: "Workspace" });
    expect(screen.queryByRole("button", { name: /Remove checkout/ })).not.toBeInTheDocument();
    unmount();
    const second = renderRun(runDetail({ workspace: { ...workspace, mode: "project", owned: false, branch: undefined } }));
    await screen.findByText("Project checkout");
    expect(screen.queryByRole("button", { name: /Remove checkout/ })).not.toBeInTheDocument();
    second.unmount();
    renderRun(runDetail({ workspace: { ...workspace, state: "removed" } }));
    await screen.findByText("Removed");
    expect(screen.queryByRole("button", { name: /Remove checkout/ })).not.toBeInTheDocument();
  });
});

describe("checkout removal dialog", () => {
  const open = async () => {
    renderRun(runDetail({ workspace }));
    fireEvent.click(await screen.findByRole("button", { name: "Remove checkout…" }));
    return screen.findByRole("dialog");
  };

  it("defaults to Keep, discloses uncommitted changes and sends what it showed", async () => {
    currentDetail = { ...wsDetail, dirty: true };
    const dialog = await open();
    expect(within(dialog).getByText(/branch, its commits and the agents' conversations are kept/)).toBeInTheDocument();
    expect(await within(dialog).findByText("Uncommitted changes")).toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Keep checkout" })).toHaveFocus();
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove and discard changes" }));
    await waitFor(() => expect(removals).toEqual([{ revision: 3, dirty_known: true, dirty: true, acknowledge_unknown: false }]));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
  });

  it("requires acknowledgement when the dirty state is unknown", async () => {
    currentDetail = { ...wsDetail, dirty_known: false };
    const dialog = await open();
    const remove = await within(dialog).findByRole("button", { name: "Remove checkout" });
    expect(remove).toBeDisabled();
    expect(within(dialog).getByText("Tick the box above to enable Remove.")).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("checkbox", { name: /uncommitted changes may be lost/ }));
    expect(remove).toBeEnabled();
    fireEvent.click(remove);
    await waitFor(() => expect(removals).toEqual([{ revision: 3, dirty_known: false, dirty: false, acknowledge_unknown: true }]));
  });

  it("deletes nothing when dismissed with Escape", async () => {
    const dialog = await open();
    await within(dialog).findByText("No uncommitted changes were found.");
    fireEvent.keyDown(document.activeElement ?? document.body, { key: "Escape" });
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(removals).toEqual([]);
  });

  it("shows the blocking reason instead of a Remove button", async () => {
    currentDetail = { ...wsDetail, removable: false, blocked: "agents in this checkout are still running" };
    const dialog = await open();
    expect(await within(dialog).findByText(/agents in this checkout are still running/)).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: /^Remove/ })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("button", { name: "Keep checkout" })).toBeInTheDocument();
  });

  it("surfaces a refused removal and keeps the dialog open", async () => {
    removeStatus = 409;
    const dialog = await open();
    fireEvent.click(await within(dialog).findByRole("button", { name: "Remove checkout" }));
    expect(await within(dialog).findByRole("alert")).toHaveTextContent("checkout changed since you reviewed it");
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
});

describe("retained workspace list", () => {
  const page = (items: object[], next?: string) => HttpResponse.json({ workspaces: items, ...(next ? { next_cursor: next } : {}) });

  it("lists retained rows with their original names, filters by project and pages", async () => {
    server.use(http.get("/api/pipeline-workspaces", ({ request }) => {
      const url = new URL(request.url);
      listQueries.push(url.search);
      if (url.searchParams.get("cursor") === "ws_1") return page([{ ...workspace, workspace_id: "ws_0", branch: "chuck/old", run_name: "Old run", state: "removed" }]);
      return page([{ ...workspace, run_name: "Deleted run" }], "ws_1");
    }));
    render(<QueryClientProvider client={client()}><MemoryRouter><WorkspacesPage /></MemoryRouter></QueryClientProvider>);

    // The run record is gone (no /api/pipeline-runs handler); the row keeps its names.
    expect(await screen.findByText("Deleted run")).toBeInTheDocument();
    expect(screen.getByText("App", { selector: "small" })).toBeInTheDocument();
    expect(screen.getByText("chuck/ship")).toBeInTheDocument();
    expect(screen.getByText("/tmp/app-ship")).toBeInTheDocument();
    expect(screen.getByText("Ready")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Remove checkout…" })).toHaveLength(1);

    fireEvent.click(screen.getByRole("button", { name: "More workspaces" }));
    expect(await screen.findByText("Old run")).toBeInTheDocument();
    expect(screen.getByText("Removed")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Remove checkout…" })).toHaveLength(1);
    expect(screen.queryByRole("button", { name: "More workspaces" })).not.toBeInTheDocument();

    await screen.findByRole("option", { name: "App" });
    fireEvent.change(screen.getByLabelText("Project"), { target: { value: "app" } });
    await waitFor(() => expect(listQueries.some((query) => query.includes("project=app"))).toBe(true));
  });

  it("treats a null collection as empty and offers removal from the row", async () => {
    server.use(http.get("/api/pipeline-workspaces", () => HttpResponse.json({ workspaces: null })));
    render(<QueryClientProvider client={client()}><MemoryRouter><WorkspacesPage /></MemoryRouter></QueryClientProvider>);
    expect(await screen.findByText("No retained workspaces")).toBeInTheDocument();
  });

  it("opens the same removal dialog from a row", async () => {
    server.use(http.get("/api/pipeline-workspaces", () => page([workspace])));
    render(<QueryClientProvider client={client()}><MemoryRouter><WorkspacesPage /></MemoryRouter></QueryClientProvider>);
    fireEvent.click(await screen.findByRole("button", { name: "Remove checkout…" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(await within(dialog).findByRole("button", { name: "Remove checkout" }));
    await waitFor(() => expect(removals).toHaveLength(1));
  });
});
