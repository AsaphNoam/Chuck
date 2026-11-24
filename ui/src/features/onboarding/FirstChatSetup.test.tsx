import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { FirstChatSetup } from "./FirstChatSetup";

const catalog = { version: 2, backends: {
  claude: { name: "Claude", type: "claude-acp", default: true, default_model: "sonnet", models: { sonnet: { name: "Sonnet", model: "sonnet" } } },
  codex: { name: "Codex", type: "codex-acp", default: false, default_model: "gpt", models: { gpt: { name: "GPT", model: "gpt" } } },
} };
const ready = { runtime: { source: "detected", state: "available" }, credentials: { status: "ok" }, catalog: { status: "disabled", added_count: 0 } };
const server = setupServer(
  http.get("/api/backends", () => HttpResponse.json(catalog, { headers: { ETag: '"catalog"' } })),
  http.get("/api/config", () => HttpResponse.json({ default_role: "default", default_project: "", onboarding: { first_chat: true, launchable_projects: [] } })),
  http.get("/api/projects", () => HttpResponse.json({})),
);
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => { cleanup(); server.resetHandlers(); });
afterAll(() => server.close());

function show() {
  const onComplete = vi.fn();
  render(<MemoryRouter><QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><FirstChatSetup onComplete={onComplete} /></QueryClientProvider></MemoryRouter>);
  return onComplete;
}

describe("first chat setup", () => {
  it("disables a selected provider removed during Settings repair while preserving input", async () => {
    let removed = false;
    server.use(
      http.get("/api/backends", () => HttpResponse.json(removed ? { ...catalog, backends: { claude: catalog.backends.claude } } : catalog)),
      http.post("/api/backends/:id/refresh-provider", ({ params }) => HttpResponse.json(params.id === "claude" ? { ...ready, credentials: { status: "failed", detail: "not_logged_in" } } : ready)),
    );
    show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("codex"));
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/kept" } });
    fireEvent.click(screen.getByText("Advanced"));
    fireEvent.click(screen.getByRole("link", { name: "Open backend settings in a new tab" }));
    removed = true;
    fireEvent.focus(window);
    await screen.findByText("The selected backend is unavailable. Choose another ready provider.");
    expect(screen.getByLabelText("Provider")).toHaveValue("codex");
    expect(screen.getByLabelText("Project folder")).toHaveValue("/tmp/kept");
    expect(screen.getByRole("button", { name: "Start chat" })).toBeDisabled();
  });

  it("saves an explicit bundle choice for only the affected backend and retains form input", async () => {
    const requests: Array<{ id: string; body: unknown }> = [];
    server.use(http.post("/api/backends/:id/refresh-provider", async ({ params, request }) => {
      const body = await request.json() as { backend?: unknown };
      requests.push({ id: String(params.id), body });
      return HttpResponse.json(params.id === "claude" && !body.backend ? { ...ready, runtime: { source: "detected", state: "missing", bundle_available: true }, credentials: { status: "skipped", detail: "cli_not_installed" } } : ready);
    }));
    show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("codex"));
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/work" } });
    fireEvent.click(await screen.findByRole("button", { name: "Use Chuck bundle" }));
    await waitFor(() => expect(screen.getByRole("option", { name: "Claude" })).toBeEnabled());
    expect(requests[2]).toEqual({ id: "claude", body: { backend: { ...catalog.backends.claude, provider_mode: "bundled" } } });
    expect(screen.getByLabelText("Provider")).toHaveValue("codex");
    expect(screen.getByLabelText("Project folder")).toHaveValue("/tmp/work");
  });

  it("refreshes only the repaired provider on return without losing the folder or launching", async () => {
    const checks: string[] = [];
    server.use(http.post("/api/backends/:id/refresh-provider", ({ params }) => {
      checks.push(String(params.id));
      return HttpResponse.json(params.id === "claude" ? { ...ready, credentials: { status: "failed", detail: "not_logged_in" } } : ready);
    }));
    show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("codex"));
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/kept" } });
    fireEvent.click(screen.getByRole("link", { name: "Open backend settings for Claude Code in a new tab" }));
    fireEvent.focus(window);
    await waitFor(() => expect(checks).toHaveLength(3));
    expect(checks[2]).toBe("claude");
    expect(screen.getByLabelText("Provider")).toHaveValue("codex");
    expect(screen.getByLabelText("Project folder")).toHaveValue("/tmp/kept");
  });

  it("reports an unavailable Codex credential observation without claiming sign-in is needed", async () => {
    server.use(http.post("/api/backends/:id/refresh-provider", ({ params }) => HttpResponse.json(params.id === "codex" ? { ...ready, credentials: { status: "skipped", detail: "no_api_key" } } : ready)));
    show();
    await screen.findByText("The check could not complete. Native sign-in could not be confirmed.");
    expect(screen.queryByText("Sign in to continue.")).not.toBeInTheDocument();
    expect(screen.queryByText("chuck auth codex")).not.toBeInTheDocument();
  });
  it("checks independently and keeps a ready selection when the default finishes later", async () => {
    let finishClaude!: () => void;
    const claudeBarrier = new Promise<void>((resolve) => { finishClaude = resolve; });
    const checked: string[] = [];
    server.use(http.post("/api/backends/:id/refresh-provider", async ({ params, request }) => {
      expect(await request.json()).toEqual({ check_only: true });
      checked.push(String(params.id));
      if (params.id === "claude") await claudeBarrier;
      return HttpResponse.json(ready);
    }));
    show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("codex"));
    expect(checked.sort()).toEqual(["claude", "codex"]);
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/work" } });
    expect(screen.getByRole("button", { name: "Start chat" })).toBeEnabled();
    finishClaude();
    await waitFor(() => expect(screen.getByRole("option", { name: "Claude" })).toBeEnabled());
    expect(screen.getByLabelText("Provider")).toHaveValue("codex");
    expect(screen.getByLabelText("Project folder")).toHaveValue("/tmp/work");
  });

  it("retries only completion after a confirmed launch and keeps the exact chat link", async () => {
    let creates = 0;
    let launches = 0;
    let completions = 0;
    server.use(
      http.post("/api/backends/:id/refresh-provider", () => HttpResponse.json(ready)),
      http.post("/api/projects", async ({ request }) => {
        creates++;
        expect(await request.json()).toMatchObject({ cwd: "/tmp/work", title: "work" });
        return HttpResponse.json({ project: "work" });
      }),
      http.post("/api/sessions", async ({ request }) => {
        launches++;
        expect(await request.json()).toMatchObject({ project: "work", backend: "claude", interface: "chat" });
        return HttpResponse.json({ agent: { agent_id: "first", name: "First" } });
      }),
      http.put("/api/config", async ({ request }) => {
        completions++;
        expect(await request.json()).toMatchObject({ onboarding_complete: true, onboarding_backend: "claude", default_project: "work" });
        return completions === 1 ? HttpResponse.json({ error: { message: "Disk unavailable" } }, { status: 500 }) : HttpResponse.json({});
      }),
    );
    const onComplete = show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("claude"));
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/work" } });
    fireEvent.click(screen.getByRole("button", { name: "Start chat" }));
    expect(await screen.findByRole("link", { name: "Open chat" })).toHaveAttribute("href", "/agent/first");
    fireEvent.click(screen.getByRole("button", { name: "Finish setup" }));
    await waitFor(() => expect(onComplete).toHaveBeenCalledWith("first"));
    expect([creates, launches, completions]).toEqual([1, 1, 2]);
  });

  it("retains a confirmed project after a definite failed launch", async () => {
    let creates = 0;
    let launches = 0;
    server.use(
      http.post("/api/backends/:id/refresh-provider", () => HttpResponse.json(ready)),
      http.post("/api/projects", () => { creates++; return HttpResponse.json({ project: "work" }); }),
      http.post("/api/sessions", () => { launches++; return HttpResponse.json({ error: { message: "Sign in required" } }, { status: 400 }); }),
    );
    show();
    await waitFor(() => expect(screen.getByLabelText("Provider")).toHaveValue("claude"));
    fireEvent.change(screen.getByLabelText("Project folder"), { target: { value: "/tmp/work" } });
    fireEvent.click(screen.getByRole("button", { name: "Start chat" }));
    await screen.findByText("Sign in required");
    fireEvent.click(screen.getByRole("button", { name: "Start chat" }));
    await waitFor(() => expect(launches).toBe(2));
    expect(creates).toBe(1);
  });
});
