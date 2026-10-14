import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { ScopedProjectDashboard } from "./ProjectDashboard";
import { useAgentStore } from "../../store/agentStore";

// FS-02.R60's second entry point: the scoped project dashboard's own header
// offers the same fork action, and a worktree project's header names its branch
// (FS-19.R6).

const server = setupServer(
  http.get("/api/layout", () => HttpResponse.json({ order: [], density: { perRow: 3, gap: 12 }, expanded: [] })),
  http.get("/api/roles", () => HttpResponse.json({})),
  http.get("/api/backends", () => HttpResponse.json({ version: 2, backends: {} })),
  http.get("/api/capabilities", () => HttpResponse.json({})),
  http.get("/api/tasks", () => HttpResponse.json({ tasks: [] })),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
  useAgentStore.setState({ agents: {}, order: [] });
});
afterAll(() => server.close());

function renderScoped(id: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[`/project/${id}`]}>
        <Routes>
          <Route path="project/:project" element={<ScopedProjectDashboard />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ScopedProjectDashboard worktree surface", () => {
  it("offers the fork action in the header of a repo-backed project", async () => {
    server.use(http.get("/api/projects", () => HttpResponse.json({
      app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false, repo_backed: true },
    })));
    renderScoped("app");
    expect(await screen.findByRole("button", { name: "New worktree project" })).toBeInTheDocument();
  });

  it("offers no fork action when the project is not repo-backed", async () => {
    server.use(http.get("/api/projects", () => HttpResponse.json({
      app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false },
    })));
    renderScoped("app");
    await screen.findByText("Back to projects");
    expect(screen.queryByRole("button", { name: "New worktree project" })).toBeNull();
  });

  it("names the branch in the header of a worktree project", async () => {
    server.use(http.get("/api/projects", () => HttpResponse.json({
      fork: {
        title: "Fork", color: [100, 116, 139], cwd: "/home/wt/fork", add_dirs: [], context_prompt: "",
        archived: false, repo_backed: true, worktree: { owned: true, branch: "chuck/fork" },
      },
    })));
    renderScoped("fork");
    expect(await screen.findByText("⑂ chuck/fork")).toBeInTheDocument();
  });

  it("opens the creation form from the header", async () => {
    server.use(
      http.get("/api/projects", () => HttpResponse.json({
        app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false, repo_backed: true },
      })),
      http.get("/api/projects/app/worktree", () => HttpResponse.json({
        owned: false, repo_backed: true, branch: "", base: "trunk", dirty: false, dirty_known: true,
      })),
    );
    renderScoped("app");
    fireEvent.click(await screen.findByRole("button", { name: "New worktree project" }));
    expect(await screen.findByLabelText("Title")).toHaveValue("App");
    await waitFor(() => expect(screen.getByLabelText("Base")).toHaveValue("trunk"));
  });
});

describe("ScopedProjectDashboard background menu (FS-02.R77)", () => {
  const catalog = () => server.use(http.get("/api/projects", () => HttpResponse.json({
    app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false },
  })));

  it("offers New agent and New think tank on a background right-click", async () => {
    catalog();
    const { container } = renderScoped("app");
    await screen.findByText("Back to projects");
    fireEvent.contextMenu(container.querySelector(".scoped-project-header")!, { clientX: 30, clientY: 40 });
    expect(screen.getByRole("menuitem", { name: "New agent" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "New think tank" })).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("menu")).toBeNull();
  });

  // FS-02.A58: header form controls retain their own context-menu behavior.
  it("leaves header buttons and their nested content alone", async () => {
    server.use(http.get("/api/projects", () => HttpResponse.json({
      app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false, repo_backed: true },
    })));
    renderScoped("app");
    await screen.findByRole("button", { name: "New worktree project" });
    for (const name of ["New worktree project", "Think Tank", "New Agent"]) {
      const button = screen.getByRole("button", { name });
      const nested = document.createElement("span");
      button.append(nested);
      for (const target of [button, nested]) {
        expect(fireEvent.contextMenu(target)).toBe(true);
        expect(screen.queryByRole("menu")).toBeNull();
      }
    }
  });

  // FS-02.A58: excluded surfaces do not become project background targets.
  it("preserves room, form, dialog and open-menu right-clicks", async () => {
    catalog();
    const { container } = renderScoped("app");
    await screen.findByText("Back to projects");
    const header = container.querySelector(".scoped-project-header")!;
    for (const markup of [
      '<div class="room-card"><span></span></div>',
      '<input>', '<textarea></textarea>', '<select></select>',
      '<div role="dialog"><span></span></div>',
      '<div data-ui="dialog"><span></span></div>',
      '<div class="context-menu"><span></span></div>',
    ]) {
      const fixture = document.createElement("div");
      fixture.innerHTML = markup;
      header.append(fixture);
      expect(fireEvent.contextMenu(fixture.querySelector("span") ?? fixture.firstElementChild!)).toBe(true);
      expect(screen.queryByRole("menu")).toBeNull();
      fixture.remove();
    }
  });

  it("opens the dialog each header button opens", async () => {
    catalog();
    const { container } = renderScoped("app");
    await screen.findByText("Back to projects");
    fireEvent.contextMenu(container.querySelector(".scoped-project-header")!);
    fireEvent.click(screen.getByRole("menuitem", { name: "New think tank" }));
    expect(await screen.findByText("New Think Tank")).toBeInTheDocument();
    expect(screen.queryByRole("menu")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    await waitFor(() => expect(screen.queryByText("New Think Tank")).toBeNull());

    fireEvent.contextMenu(container.querySelector(".scoped-project-header")!);
    fireEvent.click(screen.getByRole("menuitem", { name: "New agent" }));
    expect(await screen.findByRole("dialog")).toBeInTheDocument();
  });

  it("leaves a right-click on an agent card to the card", async () => {
    catalog();
    useAgentStore.setState({
      agents: { a1: {
        agent_id: "a1", name: "alpha", role: "implementer", project: "app", backend: "claude", model: "sonnet",
        interface: "chat", created_at: "2026-07-29T00:00:00Z", running: false, state: "done", detail: "",
        context_pct: 0, updated_at: 1, archived: false,
      } },
      order: ["a1"],
    });
    const { container } = renderScoped("app");
    const card = await waitFor(() => {
      const el = container.querySelector(".agent-card");
      expect(el).not.toBeNull();
      return el!;
    });
    fireEvent.contextMenu(card);
    expect(screen.queryByRole("menuitem", { name: "New think tank" })).toBeNull();
  });
});
