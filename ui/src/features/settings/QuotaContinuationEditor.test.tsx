import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { QuotaContinuationEditor } from "./QuotaContinuationEditor";

const configDoc = {
  version: 1,
  port: 4317,
  default_project: "my-app",
  default_role: "default",
  skip_permissions: false,
  onboarding_complete: true,
  task_concurrency: 4,
  notifications: { desktop_enabled: true, muted: {} },
  onboarding: { satisfied: true, steps: { backend: { done: true }, project: { done: true }, role: { done: true } } },
};
let lastPut: unknown;
let currentConfig: Record<string, unknown> = { ...configDoc };
const server = setupServer(
  http.get("/api/config", () => HttpResponse.json(currentConfig)),
  http.put("/api/config", async ({ request }) => {
    lastPut = await request.json();
    currentConfig = { ...currentConfig, ...(lastPut as object) };
    return HttpResponse.json(currentConfig);
  }),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  cleanup();
  server.resetHandlers();
  lastPut = undefined;
  currentConfig = { ...configDoc };
});
afterAll(() => server.close());

function renderEditor() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={queryClient}><QuotaContinuationEditor /></QueryClientProvider>);
}

describe("QuotaContinuationEditor", () => {
  it("defaults an older config without auto_continue to enabled and saves changes", async () => {
    renderEditor();
    const toggle = await screen.findByRole("checkbox", { name: "Auto continue after quota reset" });
    await waitFor(() => expect(toggle).toBeEnabled());
    expect(toggle).toBeChecked();
    fireEvent.click(toggle);
    await waitFor(() => expect(lastPut).toEqual({ auto_continue: false }));
    await waitFor(() => expect(toggle).not.toBeChecked());
  });

  it("restores the saved value and shows the server reason when saving fails", async () => {
    server.use(http.put("/api/config", () => HttpResponse.json({ error: { message: "disk is read-only" } }, { status: 500 })));
    renderEditor();
    const toggle = await screen.findByRole("checkbox", { name: "Auto continue after quota reset" });
    await waitFor(() => expect(toggle).toBeEnabled());
    fireEvent.click(toggle);
    expect(toggle).not.toBeChecked();
    await waitFor(() => expect(toggle).toBeChecked());
    expect(await screen.findByRole("alert")).toHaveTextContent("disk is read-only");
  });

  it("keeps the persisted off setting when cancelling pending continuations fails", async () => {
    server.use(http.put("/api/config", () => {
      currentConfig = { ...currentConfig, auto_continue: false };
      return HttpResponse.json({
        auto_continue: false,
        error: { message: "Auto continue is off, but pending continuations could not be cancelled." },
      }, { status: 500 });
    }));
    renderEditor();
    const toggle = await screen.findByRole("checkbox", { name: "Auto continue after quota reset" });
    await waitFor(() => expect(toggle).toBeEnabled());
    fireEvent.click(toggle);
    await waitFor(() => expect(toggle).not.toBeChecked());
    expect(await screen.findByRole("alert")).toHaveTextContent("pending continuations could not be cancelled");
    expect(screen.getByRole("alert")).not.toHaveTextContent("was not saved");
  });
});
