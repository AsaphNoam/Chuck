import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { PermissionModeToggle } from "./PermissionModeToggle";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterAll(() => server.close());
afterEach(() => {
  cleanup();
  server.resetHandlers();
});

const advertised = (current: string) => ({
  current,
  options: [
    { id: "default", name: "Manual", description: "Prompts for dangerous operations" },
    { id: "acceptEdits", name: "Accept Edits", description: "Auto-accept file edits" },
    { id: "plan", name: "Plan Mode", description: "No tool execution" },
  ],
});

function renderToggle(agentState = "idle") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><PermissionModeToggle agentId="a1" agentState={agentState} /></QueryClientProvider>);
}

// FS-03.A67
describe("PermissionModeToggle", () => {
  it("offers exactly the advertised modes and follows the reported mode", async () => {
    const posts: unknown[] = [];
    server.use(
      http.get("/api/sessions/a1/modes", () => HttpResponse.json(advertised("default"))),
      http.post("/api/sessions/a1/mode", async ({ request }) => {
        posts.push(await request.json());
        return HttpResponse.json(advertised("plan"));
      }),
    );
    renderToggle();
    const manual = await screen.findByRole("radio", { name: "Manual" });
    expect(manual).toBeChecked();
    expect(screen.getAllByRole("radio")).toHaveLength(3);
    expect(screen.getByText("Plan Mode").closest("label")).toHaveAttribute("title", "No tool execution");
    fireEvent.click(screen.getByRole("radio", { name: "Plan Mode" }));
    await waitFor(() => expect(screen.getByRole("radio", { name: "Plan Mode" })).toBeChecked());
    expect(posts).toEqual([{ mode: "plan" }]);
  });

  it("restores the reported mode and shows the reason when the change is refused", async () => {
    server.use(
      http.get("/api/sessions/a1/modes", () => HttpResponse.json(advertised("default"))),
      http.post("/api/sessions/a1/mode", () => HttpResponse.json({ error: { code: "invalid_field", message: "provider rejected the setting: mode \"acceptEdits\"" } }, { status: 400 })),
    );
    renderToggle();
    fireEvent.click(await screen.findByRole("radio", { name: "Accept Edits" }));
    expect(await screen.findByRole("alert")).toHaveTextContent('provider rejected the setting: mode "acceptEdits"');
    await waitFor(() => expect(screen.getByRole("radio", { name: "Manual" })).toBeChecked());
  });

  it("rereads the mode when the agent's state changes", async () => {
    let current = "plan";
    server.use(http.get("/api/sessions/a1/modes", () => HttpResponse.json(advertised(current))));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = (state: string) => <QueryClientProvider client={client}><PermissionModeToggle agentId="a1" agentState={state} /></QueryClientProvider>;
    const { rerender } = render(view("busy"));
    expect(await screen.findByRole("radio", { name: "Plan Mode" })).toBeChecked();
    current = "default";
    rerender(view("idle"));
    await waitFor(() => expect(screen.getByRole("radio", { name: "Manual" })).toBeChecked());
  });

  it.each([
    ["no advertisement", () => HttpResponse.json({ current: "", options: [] })],
    ["no live session", () => HttpResponse.json({ error: { code: "agent_not_running", message: "agent is not running" } }, { status: 409 })],
  ])("renders nothing with %s", async (_label, reply) => {
    let read = false;
    server.use(http.get("/api/sessions/a1/modes", () => { read = true; return reply(); }));
    const { container } = renderToggle();
    await waitFor(() => expect(read).toBe(true));
    expect(container).toBeEmptyDOMElement();
  });
});
