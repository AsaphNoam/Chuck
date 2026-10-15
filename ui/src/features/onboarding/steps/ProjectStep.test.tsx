import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { ProjectStep } from "./ProjectStep";

const server = setupServer();

beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => { cleanup(); server.resetHandlers(); });
afterAll(() => server.close());

describe("ProjectStep", () => {
  // FS-04.A19: onboarding uses the same six-preset picker and sends its RGB
  // triple through the unchanged project API.
  it("defaults to Slate and submits a selected preset", async () => {
    let created: Record<string, unknown> | null = null;
    server.use(http.post("/api/projects", async ({ request }) => {
      created = await request.json() as Record<string, unknown>;
      return HttpResponse.json({ project: "app", ...created }, { status: 201 });
    }));
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><ProjectStep onDone={() => {}} /></QueryClientProvider>);

    expect(screen.getAllByRole("button", { name: /project color$/ })).toHaveLength(6);
    expect(screen.getByRole("button", { name: "Slate project color" })).toHaveAttribute("aria-pressed", "true");
    fireEvent.change(screen.getByPlaceholderText("e.g. My App"), { target: { value: "App" } });
    fireEvent.change(screen.getByPlaceholderText("~/Projects/my-app"), { target: { value: "/tmp/app" } });
    fireEvent.click(screen.getByRole("button", { name: "Green project color" }));
    fireEvent.click(screen.getByRole("button", { name: "Create project" }));

    await waitFor(() => expect(created).toMatchObject({ color: [34, 197, 94] }));
  });

  // FS-04.R30 / INVARIANTS §8: a cwd_not_found warning stays visible beside
  // the field; the step advances only after explicit acknowledgement and never
  // re-creates the project.
  it("holds on a cwd_not_found warning until the user continues", async () => {
    let posts = 0;
    server.use(http.post("/api/projects", () => {
      posts += 1;
      return HttpResponse.json({
        project: "smoke",
        warnings: [{ field: "cwd", code: "cwd_not_found", message: "Directory relative/path does not exist." }],
      }, { status: 201 });
    }));
    const done: string[] = [];
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><ProjectStep onDone={(id) => done.push(id)} /></QueryClientProvider>);

    fireEvent.change(screen.getByPlaceholderText("e.g. My App"), { target: { value: "Smoke" } });
    fireEvent.change(screen.getByPlaceholderText("~/Projects/my-app"), { target: { value: "relative/path" } });
    fireEvent.click(screen.getByRole("button", { name: "Create project" }));

    expect(await screen.findByText(/Directory relative\/path does not exist\./)).toBeInTheDocument();
    expect(done).toEqual([]);
    expect(screen.queryByRole("button", { name: "Create project" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Continue anyway" }));
    expect(done).toEqual(["smoke"]);
    expect(posts).toBe(1);
  });

  // FS-04.A22: onboarding browses for its cwd and stays manually editable.
  it("fills the working directory from the folder panel and still accepts typing", async () => {
    let posts = 0;
    server.use(
      http.post("/api/projects", () => { posts += 1; return HttpResponse.json({ project: "app" }, { status: 201 }); }),
      http.post("/api/directory-picker", () => HttpResponse.json({ path: "/Users/me/code" })),
    );
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(<QueryClientProvider client={queryClient}><ProjectStep onDone={() => {}} /></QueryClientProvider>);

    const cwdInput = screen.getByPlaceholderText("~/Projects/my-app") as HTMLInputElement;
    fireEvent.click(screen.getByRole("button", { name: "Browse for working directory" }));
    await waitFor(() => expect(cwdInput.value).toBe("/Users/me/code"));
    expect(posts).toBe(0);

    fireEvent.change(cwdInput, { target: { value: "/Users/me/code/sub" } });
    expect(cwdInput.value).toBe("/Users/me/code/sub");
  });
});
