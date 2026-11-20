import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { ScopedProjectDashboard } from "./ProjectDashboard";
import { useAgentStore } from "../../store/agentStore";

// FS-11.R11–R16: the project Files dialog. The oracle is the server catalog
// shape of TS-03.R93, already sorted newest first.

const entry = (path: string, kind = "file", mod_time: string | null = "2026-11-01T12:00:00Z") =>
  ({ path, name: path.split("/").pop()!, kind, mod_time });

const catalog = {
  project_id: "app", folder: "", partial: false,
  entries: [
    entry("docs/Plan.md", "file", "2026-11-03T12:00:00Z"),
    entry("zz/plan.md", "file", "2026-11-03T12:00:00Z"),
    entry("data/Report.CSV", "file", "2026-11-02T12:00:00Z"),
    entry("tool.ts"),
    entry("Makefile"),
    entry("page.html"),
    entry(".hidden"),
    entry("link", "symlink"),
    entry("docs", "directory"),
    entry("data", "directory"),
    entry("empty", "directory"),
    entry("zz", "directory"),
  ],
};

let catalogBody: unknown = catalog;
let catalogStatus = 200;
const reads: string[] = [];
const reveals: string[] = [];

const server = setupServer(
  http.get("/api/layout", () => HttpResponse.json({ order: [], density: { perRow: 3, gap: 12 }, expanded: [] })),
  http.get("/api/roles", () => HttpResponse.json({})),
  http.get("/api/backends", () => HttpResponse.json({ version: 2, backends: {} })),
  http.get("/api/capabilities", () => HttpResponse.json({ terminal: { available: true, default_driver: "xterm", drivers: {} }, reveal_in_finder: true })),
  http.get("/api/tasks", () => HttpResponse.json({ tasks: [] })),
  http.get("/api/projects", () => HttpResponse.json({
    app: { title: "App", color: [100, 116, 139], cwd: "/tmp/app", add_dirs: [], context_prompt: "", archived: false, resource_dir: "/home/.chuck/project-resources/app" },
  })),
  http.get("/api/projects/app/resource-files", () => HttpResponse.json(catalogBody as object, { status: catalogStatus })),
  http.get("/api/projects/app/resource-file", ({ request }) => {
    const path = new URL(request.url).searchParams.get("path")!;
    reads.push(path);
    if (path === "gone.md") return HttpResponse.json({ error: { code: "not_found", message: "that file no longer exists" } }, { status: 404 });
    return HttpResponse.json({ agent_id: "", project_id: "app", path, size: 6, mod_time: "", line_count: 1, content: "# Plan", truncated: false, language: "markdown", editable: false, edit_refusal: "project_read_only" });
  }),
  http.post("/api/projects/app/resource-file/reveal-in-finder", async ({ request }) => {
    reveals.push(((await request.json()) as { path: string }).path);
    return new HttpResponse(null, { status: 204 });
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
  catalogBody = catalog;
  catalogStatus = 200;
  reads.length = 0;
  reveals.length = 0;
  useAgentStore.setState({ agents: {}, order: [] });
});
afterAll(() => server.close());

async function openFiles() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={["/project/app"]}>
        <Routes><Route path="project/:project" element={<ScopedProjectDashboard />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  fireEvent.click(await screen.findByRole("button", { name: "Files" }));
  const dialog = await screen.findByRole("dialog");
  await within(dialog).findByRole("table");
  return dialog;
}

const rowNames = (dialog: HTMLElement) => within(dialog).getAllByRole("row").slice(1).map((row) => row.getAttribute("data-path"));

describe("Project Files dialog", () => {
  it("lists every non-folder entry newest first and focuses search", async () => {
    const dialog = await openFiles();
    expect(rowNames(dialog)).toEqual(["docs/Plan.md", "zz/plan.md", "data/Report.CSV", "tool.ts", "Makefile", "page.html", ".hidden", "link"]);
    expect(within(dialog).getByRole("searchbox", { name: "Search files" })).toHaveFocus();
    expect(within(dialog).getByRole("heading", { name: "Files · App" })).toBeInTheDocument();
    expect(within(dialog).getByText("link · unavailable")).toBeInTheDocument();
  });

  it("combines case-insensitive search, type badges and folder scope", async () => {
    const dialog = await openFiles();
    const search = within(dialog).getByRole("searchbox", { name: "Search files" });
    fireEvent.change(search, { target: { value: "PLAN" } });
    expect(rowNames(dialog)).toEqual(["docs/Plan.md", "zz/plan.md"]);
    fireEvent.change(search, { target: { value: "" } });
    const typePicker = within(dialog).getByRole("combobox", { name: "Add a file type filter" });
    fireEvent.change(typePicker, { target: { value: "md" } });
    fireEvent.change(typePicker, { target: { value: "csv" } });
    expect(rowNames(dialog)).toEqual(["docs/Plan.md", "zz/plan.md", "data/Report.CSV"]);
    fireEvent.change(typePicker, { target: { value: "" } });
    expect(rowNames(dialog)).toContain("Makefile");
    // Backspace in the empty search removes the last badge by keyboard.
    fireEvent.keyDown(search, { key: "Backspace" });
    expect(within(dialog).queryByRole("button", { name: "Remove No extension filter" })).toBeNull();
    fireEvent.click(within(dialog).getByRole("button", { name: "Remove CSV filter" }));
    expect(rowNames(dialog)).toEqual(["docs/Plan.md", "zz/plan.md"]);
    fireEvent.change(within(dialog).getByRole("combobox", { name: "Folder" }), { target: { value: "zz" } });
    expect(rowNames(dialog)).toEqual(["zz/plan.md"]);
    fireEvent.change(search, { target: { value: "nothing" } });
    expect(within(dialog).getByText("No files match this search.")).toBeInTheDocument();
  });

  it("distinguishes an empty folder from no matches", async () => {
    const dialog = await openFiles();
    fireEvent.change(within(dialog).getByRole("combobox", { name: "Folder" }), { target: { value: "empty" } });
    expect(within(dialog).getByText("This folder is empty.")).toBeInTheDocument();
  });

  it("previews a file by keyboard and returns to the same list and row", async () => {
    const dialog = await openFiles();
    const search = within(dialog).getByRole("searchbox", { name: "Search files" });
    fireEvent.change(search, { target: { value: "plan" } });
    const row = dialog.querySelector<HTMLElement>('tr[data-path="zz/plan.md"]')!;
    fireEvent.keyDown(row, { key: "Enter" });
    await waitFor(() => expect(reads).toEqual(["zz/plan.md"]));
    expect(await within(dialog).findByRole("heading", { name: "Plan" })).toBeInTheDocument();
    expect(within(dialog).queryByRole("button", { name: "Edit" })).toBeNull();
    fireEvent.click(within(dialog).getByRole("button", { name: "Back to files" }));
    await waitFor(() => expect(dialog.querySelector('tr[data-path="zz/plan.md"]')).toHaveFocus());
    expect(search).toHaveValue("plan");
  });

  it("explains a vanished file in the viewer and keeps Back to files", async () => {
    catalogBody = { ...catalog, entries: [entry("gone.md")] };
    const dialog = await openFiles();
    fireEvent.click(dialog.querySelector('tr[data-path="gone.md"]')!);
    expect(await within(dialog).findByText(/no longer exists/)).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Back to files" }));
    await waitFor(() => expect(dialog.querySelector('tr[data-path="gone.md"]')).toHaveFocus());
  });

  it("offers Copy path, Open in Finder and an HTML new tab from the row menu", async () => {
    const open = vi.spyOn(window, "open").mockReturnValue(null);
    const dialog = await openFiles();
    fireEvent.contextMenu(dialog.querySelector('tr[data-path="page.html"]')!);
    expect(within(dialog).getAllByRole("button").map((b) => b.textContent)).toEqual(expect.arrayContaining(["Copy path", "Open in Finder", "Open in new tab"]));
    fireEvent.click(within(dialog).getByRole("button", { name: "Open in new tab" }));
    expect(open).toHaveBeenCalledWith(expect.stringContaining("#page.html"), "_blank", expect.stringContaining("noopener"));
    fireEvent.contextMenu(dialog.querySelector('tr[data-path="tool.ts"]')!);
    expect(within(dialog).queryByRole("button", { name: "Open in new tab" })).toBeNull();
    fireEvent.click(within(dialog).getByRole("button", { name: "Open in Finder" }));
    await waitFor(() => expect(reveals).toEqual(["tool.ts"]));
    expect(rowNames(dialog)).toHaveLength(8);
    open.mockRestore();
  });

  it("shows a failed read with Retry, and a partial catalog as incomplete", async () => {
    catalogStatus = 429;
    catalogBody = { error: { code: "resource_scan_busy", message: "busy" } };
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/project/app"]}>
          <Routes><Route path="project/:project" element={<ScopedProjectDashboard />} /></Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole("button", { name: "Files" }));
    const dialog = await screen.findByRole("dialog");
    expect(await within(dialog).findByText(/Try again in a moment/)).toBeInTheDocument();
    catalogStatus = 200;
    catalogBody = { ...catalog, partial: true, partial_reason: "unreadable" };
    fireEvent.click(within(dialog).getByRole("button", { name: "Retry" }));
    expect(await within(dialog).findByText(/some entries could not be read/)).toBeInTheDocument();
  });

  it("shows an empty shared folder for a project with no files", async () => {
    catalogBody = { project_id: "app", folder: "", partial: false, entries: null };
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/project/app"]}>
          <Routes><Route path="project/:project" element={<ScopedProjectDashboard />} /></Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    fireEvent.click(await screen.findByRole("button", { name: "Files" }));
    expect(await screen.findByText(/No shared files yet/)).toBeInTheDocument();
  });
});
