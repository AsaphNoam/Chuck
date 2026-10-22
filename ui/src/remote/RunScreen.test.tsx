import React from "react";
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { RunScreen } from "./RunScreen";
import { screenFor } from "./PhoneApp";
import { useConnection } from "./connection";

const template = { version: 2, title: "Delivery", orchestrator_role: "implementer", inputs: [{ name: "feature", description: "Feature", required: true }], stages: [
  { id: "work", title: "Work", objective: "Do the work.", coordination: "standing", inputs: [], outputs: [] },
  { id: "review", title: "Review", objective: "Review the work.", coordination: "standing", inputs: [], outputs: [] },
  { id: "fix", title: "Fix", objective: "Address findings.", coordination: "standing", inputs: [], outputs: [] },
] };
const eligible = { eligible: true, reason: "" };
const controls = { continue: eligible, retry: eligible, replace: { eligible: false, reason: "" }, stop: eligible, repair_cleanup: { eligible: false, reason: "" } };
const run = { run_id: "run_1", template_id: "delivery", template_snapshot: template, display_name: "Ship", project: "app", goal: "Ship the feature and review it.", inputs: { feature: "Search normalization" }, assignments: {}, orchestrator: { backend: "codex", model: "m" }, dedicated_assignments: {}, state: "paused", revision: 7, pending_action: "", current_stage_id: "review", current_task_id: "task_review", current_attempt_id: "", current_agent_id: "a_live", attention_reason: "stage_needs_input", final_outcome: "", created_at: "2026-10-05T00:00:00Z", updated_at: "2026-10-05T00:00:00Z" };
const stageTask = (taskId: string, stageId: string, stageIndex: number, state: string, attempt: number, summary = "") => ({
  task_id: taskId, run_id: run.run_id, stage_id: stageId, stage_index: stageIndex, attempt_number: attempt, state,
  assignment_text: "", standing_owner: { agent_id: "a_live", name: "Chucky", state, route: "live" as const },
  result: summary ? { outcome: "success", summary, details: "", checks: "", outputs: {} } : undefined,
  work: [], created_at: "2026-10-05T00:00:00Z", updated_at: "2026-10-05T00:00:00Z",
});
const detail = { run, template, inputs: run.inputs, orchestrator: run.orchestrator, dedicated_assignments: {}, stage_tasks: [stageTask("task_work", "work", 0, "completed", 1, "Implementation completed."), stageTask("task_review", "review", 1, "waiting", 1)], assignments: {}, values: [], diagnostics: [], controls };

let posts: string[] = [];
let refuse = false;
const control = (name: string) => async ({ request }: { request: Request }) => {
  posts.push(`${name} ${await request.text()}`);
  if (refuse) return HttpResponse.json({ error: { code: "conflict", message: "run revision changed" } }, { status: 409 });
  return HttpResponse.json(detail);
};

const server = setupServer(
  http.get("/api/pipeline-runs/run_1", () => HttpResponse.json(detail)),
  http.post("/api/pipeline-runs/run_1/continue", control("continue")),
  http.post("/api/pipeline-runs/run_1/retry", control("retry")),
  http.post("/api/pipeline-runs/run_1/stop", control("stop")),
);

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
beforeEach(() => {
  posts = [];
  refuse = false;
  window.history.replaceState(null, "", "/run/run_1");
  useConnection.setState({ link: "connected", revision: 0 });
});
afterEach(() => { cleanup(); server.resetHandlers(); });
afterAll(() => server.close());

function renderScreen() {
  render(<QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}><RunScreen runId="run_1" /></QueryClientProvider>);
}

// FS-20.R14/R27 — the phone run screen's controls send the desktop's requests at
// the current revision and keep typed input when refused.
describe("RunScreen", () => {
  it("shows the run's stage and reason and continues with the typed input", async () => {
    renderScreen();
    expect(await screen.findByRole("heading", { name: "Ship" })).toBeInTheDocument();
    expect(screen.getByText("Stage 2 of 3")).toBeInTheDocument();
    expect(screen.getByText("Stage needs input")).toBeInTheDocument();
    expect(screen.getByText("Implementation completed.")).toBeInTheDocument();
    expect(screen.getAllByText("Review", { selector: "strong" })).toHaveLength(2);
    expect(screen.queryByText("Not started")).not.toBeInTheDocument();
    expect(screen.getAllByText("Ship the feature and review it.")).toHaveLength(2);
    expect(screen.getByText("Search normalization")).toBeInTheDocument();
    const input = screen.getByRole("textbox", { name: "Input for the stage (optional)" });
    fireEvent.change(input, { target: { value: "use the staging host" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    await waitFor(() => expect(posts).toEqual(['continue {"revision":7,"input":"use the staging host"}']));
    await waitFor(() => expect(input).toHaveValue(""));
  });

  it("shows the Mac's reason and keeps the input when Continue is refused", async () => {
    refuse = true;
    renderScreen();
    const input = await screen.findByRole("textbox", { name: "Input for the stage (optional)" });
    fireEvent.change(input, { target: { value: "use the staging host" } });
    fireEvent.click(screen.getByRole("button", { name: "Continue" }));
    expect(await screen.findByText("run revision changed")).toBeInTheDocument();
    expect(input).toHaveValue("use the staging host");
  });

  it("retries the stage and stops the run", async () => {
    renderScreen();
    fireEvent.click(await screen.findByRole("button", { name: "Retry stage" }));
    await waitFor(() => expect(posts).toEqual(['retry {"revision":7}']));
    await waitFor(() => expect(screen.getByRole("button", { name: "Stop run" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Stop run" }));
    fireEvent.click(screen.getByRole("dialog", { name: "Stop this pipeline?" }).querySelector(".btn-danger")!);
    await waitFor(() => expect(posts).toEqual(['retry {"revision":7}', 'stop {"revision":7}']));
  });

  // FS-20 exact target — moving between runs (a notification open) never carries
  // one run's open Stop confirmation or typed input into another.
  it("resets dialogs and input when the phone navigates to another run", async () => {
    server.use(http.get("/api/pipeline-runs/run_2", () => HttpResponse.json({ ...detail, run: { ...run, run_id: "run_2", display_name: "Other", revision: 3 } })));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = render(<QueryClientProvider client={client}>{screenFor("/run/run_1")}</QueryClientProvider>);
    fireEvent.change(await screen.findByRole("textbox", { name: "Input for the stage (optional)" }), { target: { value: "for run one" } });
    fireEvent.click(screen.getByRole("button", { name: "Stop run" }));
    expect(screen.getByRole("dialog", { name: "Stop this pipeline?" })).toBeInTheDocument();
    view.rerender(<QueryClientProvider client={client}>{screenFor("/run/run_2")}</QueryClientProvider>);
    expect(screen.queryByRole("heading", { name: "Ship" })).not.toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Other" })).toBeInTheDocument();
    expect(screen.queryByRole("dialog", { name: "Stop this pipeline?" })).not.toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Input for the stage (optional)" })).toHaveValue("");
    expect(posts).toEqual([]);
  });

  // FS-20.R52, TS-13.R29 — a refresh after reconnecting keeps an entered form.
  it("keeps the entered replacement when a refresh brings a newer run revision", async () => {
    const replaceable = { ...detail, controls: { ...controls, replace: { eligible: true, reason: "" } } };
    let current = replaceable;
    let reads = 0;
    server.use(
      http.get("/api/pipeline-runs/run_1", () => { reads++; return HttpResponse.json(current); }),
      http.get("/api/remote/runtime-options", () => HttpResponse.json({ backends: [{ id: "codex", name: "Codex", models: [{ id: "m", name: "M", efforts: [], fast: false }, { id: "n", name: "N", efforts: [], fast: false }] }] })),
    );
    renderScreen();
    fireEvent.click(await screen.findByRole("button", { name: "Replace orchestrator" }));
    const form = await screen.findByRole("form", { name: "Replace orchestrator" });
    const modelSelect = form.querySelectorAll("select")[1];
    await waitFor(() => expect(modelSelect.querySelector('option[value="n"]')).not.toBeNull());
    fireEvent.change(modelSelect, { target: { value: "n" } });
    current = { ...replaceable, run: { ...run, revision: 8 } };
    act(() => useConnection.setState({ revision: 1 }));
    await waitFor(() => expect(reads).toBe(2));
    await screen.findByRole("form", { name: "Replace orchestrator" });
    await waitFor(() => expect(form.querySelectorAll("select")[1]).toHaveValue("n"));
    expect(screen.getByRole("form", { name: "Replace orchestrator" })).toBe(form);
  });

  it("opens the current stage's conversation", async () => {
    renderScreen();
    fireEvent.click(await screen.findByRole("button", { name: "Open conversation" }));
    expect(window.location.pathname).toBe("/agent/a_live");
  });

  it("disables every control while the Mac is unreachable", async () => {
    useConnection.setState({ link: "unreachable" });
    renderScreen();
    expect(await screen.findByRole("button", { name: "Continue" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Retry stage" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Stop run" })).toBeDisabled();
  });
});
