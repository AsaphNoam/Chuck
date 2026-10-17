import React, { useState } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { act, cleanup, fireEvent, render, renderHook, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { AgentState } from "../api/types";
import { useAnnotationStore } from "../store/annotationStore";
import { useAnnotationDelivery } from "../components/chat/annotationDelivery";
import { PhoneAnnotations, usePhoneAnnotationDelivery } from "./AnnotationForm";
import { useConnection } from "./connection";
import { AgentScreen } from "./AgentScreen";

// Deferred launch/delivery races for the shared annotation owner
// (FS-13.R29–R30, TS-08.R134–R135, INV §1).

const mocks = vi.hoisted(() => ({ send: vi.fn(), validate: vi.fn(), launch: vi.fn() }));
vi.mock("../api/client", async (original) => ({
  ...await original<object>(), sendAnnotations: mocks.send, validateAnnotations: mocks.validate, launchAgent: mocks.launch,
  getHeldPrompt: vi.fn(async () => ({ text: "" })),
  getTranscriptWindow: vi.fn(async () => ({ agent_id: "a1", has_more: false, events: [
    { agent_id: "a1", seq: 1, type: "user_text", ts: "", data: { text: "first" } },
    { agent_id: "a1", seq: 2, type: "assistant_text", ts: "", data: { text: "second response" } },
  ] })),
}));
vi.mock("../api/config", () => ({ useRoles: () => ({ data: { implementer: { title: "Implementer" } } }) }));
vi.mock("./api", async (original) => ({ ...await original<object>(), phoneFetch: vi.fn(async (path: string) => path === "/api/roles" ? { implementer: { title: "Implementer" } } : path === "/api/projects" ? { project: { title: "Project" } } : {}), getRuntimeOptions: vi.fn(async () => ({ backends: [] })) }));

const agent = { agent_id: "a1", name: "Source", role: "implementer", project: "project", interface: "chat", running: true, state: "idle" } as AgentState;
const draft = { seq: 1, excerpt: "first", instruction: "review first" };
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: unknown) => void;
  const promise = new Promise<T>((done, fail) => { resolve = done; reject = fail; });
  return { promise, resolve, reject };
}
beforeEach(() => {
  vi.clearAllMocks();
  mocks.validate.mockResolvedValue({ valid: true });
  mocks.send.mockResolvedValue({ accepted: true });
  useAnnotationStore.setState({ bySource: { a1: [draft] }, overallBySource: {}, editedAt: {}, collapsedBySource: {} });
  useConnection.setState({ link: "connected", agents: { a1: agent } });
  window.history.replaceState(null, "", "/agent/a1");
  window.scrollTo = vi.fn();
});
afterEach(cleanup);
function TrayHarness() {
  const delivery = usePhoneAnnotationDelivery("a1", agent);
  const [open, setOpen] = useState(true);
  return <PhoneAnnotations delivery={delivery} open={open} focusNewest={0} onClose={() => setOpen(false)} />;
}
function renderTray() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={client}><TrayHarness /></QueryClientProvider>);
}
async function startPendingLaunch() {
  fireEvent.click(screen.getByRole("radio", { name: "New task" }));
  fireEvent.click(screen.getByRole("button", { name: "Continue to launch" }));
  const form = await screen.findByRole("form", { name: "New agent" });
  await waitFor(() => expect(within(form).getByRole("button", { name: "Launch and send" })).toBeEnabled());
  fireEvent.click(within(form).getByRole("button", { name: "Launch and send" }));
  expect(within(form).getByRole("button", { name: "Cancel" })).toBeDisabled();
  // Neither Close nor Escape returns to a tray that could launch again.
  fireEvent.click(screen.getByRole("button", { name: "Close dialog" }));
  fireEvent.keyDown(form, { key: "Escape" });
  expect(screen.getByRole("form", { name: "New agent" })).toBe(form);
  expect(screen.queryByRole("button", { name: "Continue to launch" })).toBeNull();
  return form;
}

it("keeps a pending launch open and delivers its batch once", async () => {
  const pending = deferred<{ agent: { agent_id: string } }>();
  mocks.launch.mockReturnValue(pending.promise);
  renderTray();
  await startPendingLaunch();
  await act(async () => { pending.resolve({ agent: { agent_id: "a3" } }); await pending.promise; });
  await waitFor(() => expect(window.location.pathname).toBe("/agent/a3"));
  expect(mocks.launch).toHaveBeenCalledTimes(1);
  expect(mocks.send).toHaveBeenCalledTimes(1);
  expect(mocks.send.mock.calls[0][1].target).toEqual({ kind: "agent", agent_id: "a3" });
  expect(useAnnotationStore.getState().bySource.a1).toBeUndefined();
});

it("lets a refused pending launch close back to the intact tray", async () => {
  const pending = deferred<{ agent: { agent_id: string } }>();
  mocks.launch.mockReturnValue(pending.promise);
  renderTray();
  const form = await startPendingLaunch();
  await act(async () => { pending.reject(new Error("no capacity")); await pending.promise.catch(() => {}); });
  expect(await within(form).findByRole("alert")).toHaveTextContent("no capacity");
  fireEvent.click(within(form).getByRole("button", { name: "Cancel" }));
  expect(await screen.findByRole("button", { name: "Continue to launch" })).toBeEnabled();
  expect(mocks.launch).toHaveBeenCalledTimes(1);
  expect(mocks.send).not.toHaveBeenCalled();
  expect(useAnnotationStore.getState().bySource.a1).toEqual([draft]);
});

it("clears an acknowledged batch but does not navigate after its owner unmounted", async () => {
  const pending = deferred<{ accepted: boolean }>();
  mocks.send.mockReturnValue(pending.promise);
  const { result, unmount } = renderHook(() => usePhoneAnnotationDelivery("a1", agent));
  act(() => result.current.onLaunched("a3"));
  expect(mocks.send).toHaveBeenCalledTimes(1);
  unmount();
  window.history.replaceState(null, "", "/agent/a2");
  await act(async () => { pending.resolve({ accepted: true }); await pending.promise; });
  expect(window.location.pathname).toBe("/agent/a2");
  expect(useAnnotationStore.getState().bySource.a1).toBeUndefined();
});

it("ignores a late launch callback from a previous source", async () => {
  const callback = vi.fn();
  const { result, rerender } = renderHook(({ sourceId }) => useAnnotationDelivery({ sourceId, sourceActive: true, source: agent, recipients: [], onLaunchDelivered: callback }), { initialProps: { sourceId: "a1" } });
  const oldLaunchCallback = result.current.onLaunched;
  rerender({ sourceId: "a2" });
  await act(async () => { oldLaunchCallback("a3"); });
  expect(mocks.send).not.toHaveBeenCalled();
  expect(callback).not.toHaveBeenCalled();
  expect(useAnnotationStore.getState().bySource.a1).toEqual([draft]);
});

it("keeps a late delivery for a previous source out of the new source", async () => {
  const pending = deferred<{ accepted: boolean }>();
  mocks.send.mockReturnValue(pending.promise);
  const callback = vi.fn();
  const { result, rerender } = renderHook(({ sourceId }) => useAnnotationDelivery({ sourceId, sourceActive: true, source: agent, recipients: [], onLaunchDelivered: callback }), { initialProps: { sourceId: "a1" } });
  act(() => result.current.onLaunched("a3"));
  rerender({ sourceId: "a2" });
  await act(async () => { pending.resolve({ accepted: true }); await pending.promise; });
  expect(callback).not.toHaveBeenCalled();
  expect(result.current.sentTo).toBeNull();
  expect(result.current.launchedAgentId).toBeNull();
  expect(useAnnotationStore.getState().bySource.a1).toBeUndefined();
});

it("keeps a new event capture made while an earlier send is pending", async () => {
  const pending = deferred<{ accepted: boolean }>();
  mocks.send.mockReturnValue(pending.promise);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(<QueryClientProvider client={client}><AgentScreen agentId="a1" /></QueryClientProvider>);
  await screen.findByText("second response");
  fireEvent.click(screen.getByRole("button", { name: /Pending annotations/ }));
  fireEvent.click(screen.getByRole("button", { name: "Send annotations" }));
  expect(mocks.send).toHaveBeenCalledTimes(1);
  fireEvent.click(screen.getByRole("button", { name: "Close dialog" }));
  const row = screen.getByText("second response").closest<HTMLElement>(".phone-annotatable")!;
  fireEvent.click(within(row).getByRole("button", { name: "Annotate" }));
  expect(useAnnotationStore.getState().bySource.a1).toHaveLength(2);
  await act(async () => { pending.resolve({ accepted: true }); await pending.promise; });
  expect(mocks.send.mock.calls[0][1].annotations).toHaveLength(1);
  expect(useAnnotationStore.getState().bySource.a1).toEqual([expect.objectContaining({ seq: 2, excerpt: "second response" })]);
});
