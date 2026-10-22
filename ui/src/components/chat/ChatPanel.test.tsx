import React from "react";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterEach, beforeEach, describe, it, expect, vi } from "vitest";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { useHeldStore } from "../../store/heldStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { useUiStore } from "../../store/uiStore";
import { ChatPanel, initialTab } from "./ChatPanel";

const mocks = vi.hoisted(() => ({
  getTranscript: vi.fn(async (id: string) => ({ agent_id: id, events: [] })),
  getHeldPrompt: vi.fn(async (id: string) => ({ agent_id: id, text: "", after_seq: 0 })),
  setSessionConfig: vi.fn(),
  switchRuntime: vi.fn(),
  getFileContent: vi.fn(async (agent_id: string, path: string) => ({
    agent_id, path, size: 14, mod_time: "2026-09-10T10:00:00Z",
    line_count: 1, content: "package state\n", truncated: false, language: "go",
  })),
  useBackends: vi.fn(),
  useProjects: vi.fn(),
  resumeAgent: vi.fn(),
}));

vi.mock("../../api/client", () => ({
  getTranscript: mocks.getTranscript,
  getHeldPrompt: mocks.getHeldPrompt,
  setSessionConfig: mocks.setSessionConfig,
  switchRuntime: mocks.switchRuntime,
  getFileContent: mocks.getFileContent,
  getTrackedFiles: vi.fn(async () => ({ files: [] })),
  resumeAgent: mocks.resumeAgent,
}));

// xterm cannot run in jsdom; the stub records whether the live bridge mounted.
vi.mock("./TerminalTab", () => ({ TerminalTab: () => <div data-testid="terminal-bridge" /> }));

vi.mock("../../api/config", async (importOriginal) => ({
  ...await importOriginal<typeof import("../../api/config")>(),
  useBackends: mocks.useBackends,
  useProjects: mocks.useProjects,
}));

vi.mock("../../api/sse", () => ({
  sseClient: { registerOpenAgent: vi.fn(() => vi.fn()) },
}));

beforeEach(() => {
  mocks.getHeldPrompt.mockImplementation(async (id: string) => ({ agent_id: id, text: "", after_seq: 0 }));
  mocks.useBackends.mockReturnValue({ data: undefined });
  mocks.useProjects.mockReturnValue({ data: undefined });
});

afterEach(() => {
  cleanup();
  mocks.switchRuntime.mockReset();
  mocks.getHeldPrompt.mockReset();
  mocks.setSessionConfig.mockReset();
  mocks.useBackends.mockReset();
  mocks.useProjects.mockReset();
  useAgentStore.setState({ agents: {}, order: [], hydrated: false, hydrating: false });
  useAnnotationStore.setState({ bySource: {}, overallBySource: {}, editedAt: {} });
  useHeldStore.setState({ byAgent: {}, afterSeqByAgent: {} });
  useTranscriptStore.setState({ byAgent: {}, rawByAgent: {}, pending: {}, appliedSeq: {} });
  useUiStore.setState({ toasts: [] });
});

// initialTab drives which tab a chat panel opens on. The load-bearing case for
// the Finding 9 secondary fix: a terminal-interface agent must default to the
// Terminal tab so a WS attaches after launch (chat agents stay on transcript).
describe("initialTab", () => {
  it("defaults a terminal-interface agent to the Terminal tab", () => {
    expect(initialTab(null, "terminal")).toBe("terminal");
  });

  it("defaults a chat-interface agent to the transcript tab", () => {
    expect(initialTab(null, "acp")).toBe("transcript");
    expect(initialTab(null, undefined)).toBe("transcript");
  });

  it("honors an explicit ?tab= over the interface default", () => {
    expect(initialTab("terminal", "acp")).toBe("terminal");
    expect(initialTab("files", "terminal")).toBe("files");
  });

  // FS-03.R80: an old ?tab=commands link opens Transcript, never an empty panel.
  it("maps the retired Commands tab to Transcript", () => {
    expect(initialTab("commands", "acp")).toBe("transcript");
    expect(initialTab("commands", "terminal")).toBe("transcript");
  });
});

function renderPanel(id: string, search = "") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/agent/${id}${search}`]}>
        <Routes><Route path="/agent/:id" element={<ChatPanel />} /></Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function liveAgent(id: string) {
  return {
    agent_id: id, name: "Nova", role: "implementer", project: "app", backend: "claude",
    model: "sonnet", interface: "chat", running: true, state: "idle", context_pct: 12,
    created_at: "2026-07-26T00:00:00Z",
  } as never;
}

const backends = {
  version: 2 as const,
  backends: {
    claude: {
      name: "Claude", type: "claude-acp" as const, default_model: "sonnet",
      models: { sonnet: { name: "Sonnet", model: "sonnet" } },
    },
    codex: {
      name: "Codex", type: "codex-acp" as const, default_model: "gpt-5",
      models: { "gpt-5": { name: "GPT-5", model: "gpt-5", efforts: ["low", "high"], default_effort: "high", fast: true } },
    },
  },
};

// FS-03.A61: the agent conversation offers no Commands tab, and an old
// ?tab=commands link lands on a working Transcript.
it("offers no Commands tab and opens Transcript for ?tab=commands", () => {
  useAgentStore.setState({ agents: { a_tabs: liveAgent("a_tabs") }, order: ["a_tabs"], hydrated: true, hydrating: false });
  renderPanel("a_tabs", "?tab=commands");

  const tabs = screen.getAllByRole("tab").map((tab) => tab.textContent);
  expect(tabs).toEqual(["Transcript", "Files"]);
  expect(screen.getByRole("tab", { name: "Transcript" })).toHaveAttribute("aria-selected", "true");
  expect(screen.getByRole("tabpanel")).toBeInTheDocument();
});

// FS-12.R61: an open file sits beside the card, outside it, while the composer
// stays inside the card.
it("opens a file beside the card and keeps the composer inside it", async () => {
  useAgentStore.setState({ agents: { a_side: liveAgent("a_side") }, order: ["a_side"], hydrated: true, hydrating: false });
  renderPanel("a_side", "?file=internal%2Fstate%2Fstate.go");

  const viewer = await waitFor(() => {
    const element = document.querySelector(".file-viewer");
    expect(element).not.toBeNull();
    return element!;
  });
  expect(viewer.closest(".chat-side-start")).not.toBeNull();
  expect(viewer.closest(".chat-card")).toBeNull();
  expect(screen.getByRole("textbox").closest(".chat-card > .chat-composer")).not.toBeNull();
});

// FS-03.R66/A47: the full agent screen's context meter renders the shared
// exact used/total figure beside the percentage when both raw counts are known.
it("shows the exact used/total token figure when both raw counts are known", () => {
  const agent = { ...liveAgent("a_exact"), context_pct: 0.06, context_used: 12345, context_size: 200000 };
  useAgentStore.setState({ agents: { a_exact: agent }, order: ["a_exact"], hydrated: true, hydrating: false });
  mocks.useBackends.mockReturnValue({ data: backends });
  renderPanel("a_exact");

  // FS-12.R61 — the header splits the same facts into a labelled meter.
  const meter = screen.getByLabelText("12K / 200K tokens · 6% context used");
  expect(meter).toHaveTextContent("Context usage6%");
  expect(meter).toHaveTextContent("12K / 200K tokens");
});

// A state with no raw pair keeps the existing percentage-only label.
it("keeps the existing percentage-only label when raw counts are unavailable", () => {
  const agent = { ...liveAgent("a_pctonly"), context_pct: 0.12 };
  useAgentStore.setState({ agents: { a_pctonly: agent }, order: ["a_pctonly"], hydrated: true, hydrating: false });
  mocks.useBackends.mockReturnValue({ data: backends });
  renderPanel("a_pctonly");

  expect(screen.getByLabelText("12% context used")).toHaveTextContent("12%");
  expect(screen.queryByText(/tokens/)).toBeNull();
});

// FS-12.A28: unsupported live controls leave no empty settings band, while a
// staged target with effort controls still exposes its existing settings.
it("omits empty session settings and shows them for a capable staged model", async () => {
  const agent = liveAgent("a_settings");
  useAgentStore.setState({ agents: { a_settings: agent }, order: ["a_settings"], hydrated: true, hydrating: false });
  mocks.useBackends.mockReturnValue({ data: backends });
  renderPanel("a_settings");

  expect(screen.getByRole("group", { name: "Runtime" })).toBeInTheDocument();
  expect(screen.queryByRole("group", { name: "Session settings" })).not.toBeInTheDocument();
  fireEvent.change(screen.getByLabelText("Backend"), { target: { value: "codex" } });
  expect(await screen.findByRole("group", { name: "Session settings" })).toBeInTheDocument();
  expect(screen.getByLabelText("Effort")).toBeInTheDocument();
});

// FS-13.R16/A8: the missing-source recovery is destructive, so it may only claim
// a source is gone once agent hydration has actually looked for it.
describe("ChatPanel missing-agent recovery", () => {
  it("lets a retained annotation tray be discarded when its source is gone", () => {
    useAgentStore.setState({ agents: {}, order: [], hydrated: true, hydrating: false });
    useAnnotationStore.getState().add("a_gone", { seq: 3, excerpt: "line", instruction: "check" });

    renderPanel("a_gone");

    expect(screen.getByText("1 pending annotation cannot be sent because the source agent no longer exists.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Discard pending annotations" }));
    expect(useAnnotationStore.getState().bySource.a_gone).toBeUndefined();
  });

  it("does not offer to discard drafts before the first hydration completes", () => {
    useAgentStore.setState({ agents: {}, order: [], hydrated: false, hydrating: true });
    useAnnotationStore.getState().add("a_pending", { seq: 3, excerpt: "line", instruction: "check" });

    renderPanel("a_pending");

    expect(screen.queryByText("Agent not found")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Discard pending annotations" })).not.toBeInTheDocument();
    expect(screen.getByText("Loading agent…")).toBeInTheDocument();
    expect(useAnnotationStore.getState().bySource.a_pending).toHaveLength(1);
  });

  it("shows the live workspace when hydration delivers the source", () => {
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });
    useAnnotationStore.getState().add("a_live", { seq: 3, excerpt: "line", instruction: "check" });

    renderPanel("a_live");

    expect(screen.getByRole("heading", { name: "Nova" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Discard pending annotations" })).not.toBeInTheDocument();
    expect(useAnnotationStore.getState().bySource.a_live).toHaveLength(1);
  });

  // FS-12.A14 (R38) — the pane work added a focus-cycling binding to the card
  // grid only. On /agent/:id there is nothing to cycle to, and the agent screen's
  // composer must keep every key it had: Ctrl+Alt+Arrow reaches no handler here
  // and moves focus nowhere.
  it("leaves the agent screen composer untouched by the pane focus-cycling keys", () => {
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    const composer = document.querySelector(".composer textarea") as HTMLTextAreaElement;
    expect(composer).not.toBeNull();
    composer.focus();
    for (const key of ["ArrowDown", "ArrowUp"]) {
      fireEvent.keyDown(composer, { key, ctrlKey: true, altKey: true });
      expect(composer).toHaveFocus();
    }
    expect(document.querySelectorAll("[data-agent-pane]")).toHaveLength(0);
  });
});

// FS-03.A12 (R27) — the chat header Back link targets the agent's project
// dashboard only when that project is a current, non-archived catalog member;
// otherwise it falls back to the projects home so a removed/archived project
// never strands the user on a dead-end route.
describe("ChatPanel back target", () => {
  it("targets the project dashboard for an active catalog project", () => {
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });
    mocks.useProjects.mockReturnValue({ data: { app: { title: "App", archived: false } } });

    renderPanel("a_live");

    expect(screen.getByRole("link", { name: "Back to project" })).toHaveAttribute("href", "/project/app");
    // FS-12.R61 — the breadcrumb names where the agent lives and its stable id.
    const crumb = screen.getByRole("navigation", { name: "Agent location" });
    expect(crumb).toHaveTextContent("App/Nova");
    expect(crumb).toHaveTextContent("a_live");
  });

  it("falls back to the projects home when the project is absent from the catalog", () => {
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });
    mocks.useProjects.mockReturnValue({ data: {} });

    renderPanel("a_live");

    expect(screen.getByRole("link", { name: "Back" })).toHaveAttribute("href", "/");
  });

  it("falls back to the projects home when the project is archived", () => {
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });
    mocks.useProjects.mockReturnValue({ data: { app: { title: "App", archived: true } } });

    renderPanel("a_live");

    expect(screen.getByRole("link", { name: "Back" })).toHaveAttribute("href", "/");
  });
});

// FS-03.A76 — the breadcrumb switches directly to the project's running agents.
describe("ChatPanel sibling agents", () => {
  function sibling(id: string, name: string, patch: Record<string, unknown> = {}) {
    return { ...(liveAgent(id) as object), name, ...patch } as never;
  }

  function seed(agents: Record<string, never>) {
    useAgentStore.setState({ agents, order: Object.keys(agents), hydrated: true, hydrating: false });
    mocks.useProjects.mockReturnValue({ data: { app: { title: "App", archived: false } } });
  }

  function crumb() {
    return screen.getByRole("navigation", { name: "Agent location" });
  }

  function entryNames() {
    return [...crumb().querySelectorAll(".chat-breadcrumb-siblings > .agent-mini-card")].map((entry) => entry.textContent);
  }

  it("lists running siblings in dashboard order and opens one in a click", async () => {
    seed({
      busy: sibling("busy", "Busy", { state: "busy" }),
      stopped: sibling("stopped", "Stopped", { running: false }),
      waiting: sibling("waiting", "Waiting", { state: "waiting_input" }),
      archived: sibling("archived", "Archived", { archived: true }),
      idle: sibling("idle", "Idle"),
      other: sibling("other", "Other", { project: "elsewhere" }),
    });
    renderPanel("busy");

    expect(entryNames()).toEqual(["BBusy", "WWaiting", "IIdle"]);
    const current = crumb().querySelector("[aria-current='page']");
    expect(current).toHaveTextContent("Busy");
    expect(current?.tagName).toBe("SPAN");
    const waiting = screen.getByRole("link", { name: "Waiting" });
    expect(waiting).toHaveAttribute("href", "/agent/waiting");
    expect(waiting).toHaveClass("waiting_input");

    fireEvent.click(waiting);
    await waitFor(() => expect(crumb()).toHaveTextContent("waiting"));
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Waiting");
    expect(crumb().querySelector("[aria-current='page']")).toHaveTextContent("Waiting");
  });

  it("joins and leaves live without reordering on a state change", () => {
    seed({
      a: sibling("a", "Ada"),
      b: sibling("b", "Bo", { running: false }),
      c: sibling("c", "Cy"),
    });
    renderPanel("a");
    expect(entryNames()).toEqual(["AAda", "CCy"]);

    act(() => useAgentStore.setState((state) => ({ agents: { ...state.agents, b: { ...state.agents.b, running: true } } })));
    expect(entryNames()).toEqual(["AAda", "BBo", "CCy"]);

    act(() => useAgentStore.setState((state) => ({ agents: { ...state.agents, a: { ...state.agents.a, state: "busy" } } })));
    expect(entryNames()).toEqual(["AAda", "BBo", "CCy"]);

    act(() => useAgentStore.setState((state) => ({ agents: { ...state.agents, c: { ...state.agents.c, running: false } } })));
    expect(entryNames()).toEqual(["AAda", "BBo"]);

    act(() => useAgentStore.setState((state) => ({ agents: { ...state.agents, b: { ...state.agents.b, running: false } } })));
    expect(crumb()).toHaveTextContent("App/Ada");
    expect(crumb().querySelector(".agent-mini-card")).toBeNull();
  });

  it("shows a stopped current agent among the running siblings", () => {
    seed({ a: sibling("a", "Ada"), s: sibling("s", "Sol", { running: false }) });
    renderPanel("s");

    expect(entryNames()).toEqual(["AAda", "SSol"]);
    expect(crumb().querySelector("[aria-current='page']")).toHaveClass("stopped");
  });

  it("caps at five with the current agent visible and a keyboard-operable overflow", async () => {
    const agents = Object.fromEntries(["one", "two", "three", "four", "five", "six", "seven"]
      .map((id) => [id, sibling(id, id[0].toUpperCase() + id.slice(1))]));
    seed(agents as Record<string, never>);
    renderPanel("seven");

    expect(entryNames()).toEqual(["OOne", "TTwo", "TThree", "FFour", "SSeven"]);
    const more = screen.getByRole("button", { name: "2 more running agents" });
    expect(more).toHaveTextContent("+2");
    more.focus();
    fireEvent.click(more);
    const menu = screen.getByRole("menu");
    expect([...menu.querySelectorAll("[role='menuitem']")].map((item) => item.getAttribute("href"))).toEqual(["/agent/five", "/agent/six"]);
    fireEvent.keyDown(menu, { key: "Escape" });
    expect(screen.queryByRole("menu")).toBeNull();

    fireEvent.click(more);
    fireEvent.click(screen.getByRole("menuitem", { name: "Six" }));
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Six"));
    expect(screen.queryByRole("menu")).toBeNull();
  });

  // FS-03.R98: a sibling opens as its dashboard card would, not with the
  // previous agent's tab or staged runtime choice.
  it("opens a chat sibling of a terminal agent on its Transcript", async () => {
    seed({ term: sibling("term", "Term", { interface: "terminal" }), chat: sibling("chat", "Chat") });
    renderPanel("term");
    expect(screen.getByTestId("terminal-bridge")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("link", { name: "Chat" }));
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Chat"));
    expect(screen.getByRole("tab", { name: "Transcript" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("tabpanel")).toBeInTheDocument();
  });

  it("opens a sibling from Files on its Transcript", async () => {
    seed({ a: sibling("a", "Ada"), b: sibling("b", "Bo") });
    renderPanel("a");
    fireEvent.mouseDown(screen.getByRole("tab", { name: "Files" }));
    expect(screen.getByRole("tab", { name: "Files" })).toHaveAttribute("aria-selected", "true");

    fireEvent.click(screen.getByRole("link", { name: "Bo" }));
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bo"));
    expect(screen.getByRole("tab", { name: "Transcript" })).toHaveAttribute("aria-selected", "true");
  });

  it("does not carry a staged runtime choice to a sibling with the same runtime", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    seed({ a: sibling("a", "Ada"), b: sibling("b", "Bo") });
    renderPanel("a");
    fireEvent.change(await screen.findByLabelText("Backend"), { target: { value: "codex" } });
    expect(screen.getByRole("button", { name: "Switch" })).toBeInTheDocument();

    fireEvent.click(screen.getByRole("link", { name: "Bo" }));
    await waitFor(() => expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Bo"));
    expect((screen.getByLabelText("Backend") as HTMLSelectElement).value).toBe("claude");
    expect(screen.queryByRole("button", { name: "Switch" })).not.toBeInTheDocument();
  });
});

// FS-03.A9 — the chat header is a second, explicit entry point to the existing
// runtime switch. It shares backend/model/effort reset behavior with launch and
// the dashboard switch dialog.
describe("ChatPanel runtime picker", () => {
  it("copies the stable thread identity from the header context menu", () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    const { container } = renderPanel("a_live");
    fireEvent.contextMenu(container.querySelector(".chat-header") as HTMLElement, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Copy thread identity" }));

    expect(writeText).toHaveBeenCalledWith("a_live");
  });

  it("surfaces a clipboard refusal from the header action", async () => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText: vi.fn().mockRejectedValue(new Error("clipboard denied")) } });
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    const { container } = renderPanel("a_live");
    fireEvent.contextMenu(container.querySelector(".chat-header") as HTMLElement, { clientX: 20, clientY: 30 });
    fireEvent.click(screen.getByRole("button", { name: "Copy thread identity" }));

    await waitFor(() => expect(useUiStore.getState().toasts.some((toast) => toast.title === "Copy failed" && toast.body === "clipboard denied")).toBe(true));
  });

  it("resets the model and effort, then switches a running chat agent", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    mocks.switchRuntime.mockResolvedValue({ history_handoff: "native_resume" });
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    const backend = await screen.findByLabelText("Backend");
    expect((screen.getByLabelText("Model") as HTMLSelectElement).value).toBe("sonnet");
    expect(screen.queryByLabelText("Effort")).not.toBeInTheDocument();
    fireEvent.change(backend, { target: { value: "codex" } });

    expect((screen.getByLabelText("Model") as HTMLSelectElement).value).toBe("gpt-5");
    expect((screen.getByLabelText("Effort") as HTMLSelectElement).value).toBe("high");
    fireEvent.click(screen.getByRole("button", { name: "Switch" }));

    await waitFor(() => expect(mocks.switchRuntime).toHaveBeenCalledWith("a_live", { backend: "codex", model: "gpt-5", effort: "high" }));
  });

  it("restores the current runtime and explains a rejected switch", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    mocks.switchRuntime.mockRejectedValue(new Error("no runtime change requested"));
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    fireEvent.change(await screen.findByLabelText("Backend"), { target: { value: "codex" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("no runtime change requested");
    expect((screen.getByLabelText("Backend") as HTMLSelectElement).value).toBe("claude");
    expect(screen.queryByRole("button", { name: "Switch" })).not.toBeInTheDocument();
  });

  it("keeps the active model and shows the provider's policy reason when a model is refused (FS-09.A48)", async () => {
    const reason = 'switch failed, rolled back to previous runtime: runtime: provider rejected the setting: model "opus": a model-switch policy hook blocked it: Opus is reserved for release work';
    const claude = backends.backends.claude;
    mocks.useBackends.mockReturnValue({ data: { ...backends, backends: { ...backends.backends, claude: {
      ...claude, models: { ...claude.models, opus: { name: "Opus", model: "opus" } },
    } } } });
    mocks.switchRuntime.mockRejectedValue(new Error(reason));
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    fireEvent.change(await screen.findByLabelText("Model"), { target: { value: "opus" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Opus is reserved for release work");
    expect((screen.getByLabelText("Model") as HTMLSelectElement).value).toBe("sonnet");
    expect(useAgentStore.getState().agents.a_live.model).toBe("sonnet");
    expect(mocks.switchRuntime).toHaveBeenCalledTimes(1);
  });

  it("keeps stopped agents' runtime identity static", () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    useAgentStore.setState({ agents: { a_stopped: { ...liveAgent("a_stopped"), running: false } }, order: ["a_stopped"], hydrated: true, hydrating: false });

    renderPanel("a_stopped");

    expect(document.querySelector(".agent-heading-details")).toHaveTextContent("claude · sonnet");
    expect(screen.getByText("Stopped")).toBeInTheDocument();
    expect(screen.queryByLabelText("Backend")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Switch" })).not.toBeInTheDocument();
  });

  // FS-12.R61 — Discard returns a staged backend/model to the current runtime
  // locally, without a switch request.
  it("discards a staged runtime change without a request", () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");
    fireEvent.change(screen.getByLabelText("Backend"), { target: { value: "codex" } });
    expect(screen.getByText("Unapplied changes")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));

    expect((screen.getByLabelText("Backend") as HTMLSelectElement).value).toBe("claude");
    expect((screen.getByLabelText("Model") as HTMLSelectElement).value).toBe("sonnet");
    expect(screen.queryByRole("button", { name: "Switch" })).toBeNull();
    expect(mocks.switchRuntime).not.toHaveBeenCalled();
  });

  it("keeps an unavailable current runtime visible until a listed target is chosen", async () => {
    mocks.useBackends.mockReturnValue({ data: { version: 2, backends: { codex: backends.backends.codex } } });
    useAgentStore.setState({ agents: { a_live: liveAgent("a_live") }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    expect((await screen.findByLabelText("Backend") as HTMLSelectElement).value).toBe("claude");
    expect((screen.getByLabelText("Model") as HTMLSelectElement).value).toBe("sonnet");
    expect(screen.queryByRole("button", { name: "Switch" })).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Backend"), { target: { value: "codex" } });
    expect(screen.getByRole("button", { name: "Switch" })).toBeEnabled();
  });

  it("applies effort immediately without staging a runtime switch", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    mocks.setSessionConfig.mockResolvedValue({ ...liveAgent("a_live"), backend: "codex", model: "gpt-5", effort: "low" });
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), backend: "codex", model: "gpt-5", effort: "high" } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");
    fireEvent.change(await screen.findByLabelText("Effort"), { target: { value: "low" } });

    await waitFor(() => expect(mocks.setSessionConfig).toHaveBeenCalledWith("a_live", { effort: "low" }));
    expect(mocks.switchRuntime).not.toHaveBeenCalled();
    expect((screen.getByLabelText("Effort") as HTMLSelectElement).value).toBe("low");
  });

  it("shows and applies fast mode only for a capable current model", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    mocks.setSessionConfig.mockResolvedValue({ ...liveAgent("a_live"), backend: "codex", model: "gpt-5", effort: "high", fast: true });
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), backend: "codex", model: "gpt-5", effort: "high", fast: false, fast_available: true } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");
    const fast = await screen.findByRole("checkbox", { name: /Fast mode/ });
    fireEvent.click(fast);

    await waitFor(() => expect(mocks.setSessionConfig).toHaveBeenCalledWith("a_live", { fast: true }));
    expect(fast).toBeChecked();
  });

  // FS-03.A29 / R46: a launch may ask for fast mode on a catalog-capable model
  // and not get it, because only the live session knows whether its current model
  // really offers the speed tier (FS-09.R55). The header must say so rather than
  // render an ordinary enabled off toggle that silently springs back, which left
  // the person unable to tell an unavailable tier from their own choice.
  it("explains fast mode the live session does not offer instead of offering a dead toggle", async () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), backend: "codex", model: "gpt-5", effort: "high", fast: false, fast_available: false } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    const fast = await screen.findByRole("checkbox", { name: /Fast mode/ });
    expect(fast).toBeDisabled();
    expect(fast).not.toBeChecked();
    expect(screen.getByText(/this model does not offer it/)).toBeInTheDocument();

    fireEvent.click(fast);
    expect(mocks.setSessionConfig).not.toHaveBeenCalled();
  });

  // FS-03.A29: a stopped chat agent renders its settings as static text, so the
  // unavailable-fast explanation never leaks into a surface with no live session.
  it("renders a stopped agent's fast mode as static text", () => {
    mocks.useBackends.mockReturnValue({ data: backends });
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), backend: "codex", model: "gpt-5", running: false, fast: true, fast_available: false } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    expect(screen.getByText("Fast mode")).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: /Fast mode/ })).toBeNull();
  });
});

// FS-03.A31/A33 — the panel decides when a queued message stops being pending
// and whether the Steer control exists at all, both from the live session's own
// report rather than from anything the client inferred.
describe("ChatPanel queued follow-up and steering", () => {
  it("clears the pending message only when the server's own event delivers it", async () => {
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), state: "busy" } }, order: ["a_live"], hydrated: true, hydrating: false });
    mocks.getHeldPrompt.mockResolvedValue({ agent_id: "a_live", text: "then run the tests", after_seq: 10 });
    useTranscriptStore.setState({ byAgent: { a_live: [
      { kind: "user_text", seq: 4, text: "then run the tests" },
      { kind: "user_text", text: "then run the tests" },
    ] } });

    renderPanel("a_live");

    // The local echo carries no seq, so it is not evidence of delivery.
    expect(await screen.findByText("Queued — sends when this turn ends")).toBeInTheDocument();
    expect(useHeldStore.getState().byAgent.a_live).toBe("then run the tests");

    useTranscriptStore.setState({ byAgent: { a_live: [{ kind: "user_text", seq: 12, text: "then run the tests" }] } });
    await waitFor(() => expect(useHeldStore.getState().byAgent.a_live).toBeUndefined());
  });

  it("rehydrates a live held message after the panel remounts", async () => {
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), state: "busy" } }, order: ["a_live"], hydrated: true, hydrating: false });
    mocks.getHeldPrompt.mockResolvedValue({ agent_id: "a_live", text: "still queued", after_seq: 21 });

    renderPanel("a_live");

    expect(await screen.findByText("still queued")).toBeInTheDocument();
    expect(useHeldStore.getState().afterSeqByAgent.a_live).toBe(21);
  });

  it("renders no Steer control for a session that does not advertise it", async () => {
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), state: "busy", steering_available: false } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    expect(await screen.findByRole("button", { name: "Send" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Steer" })).toBeNull();
  });

  it("renders Steer beside Send for a session that advertises it", async () => {
    useAgentStore.setState({ agents: { a_live: { ...liveAgent("a_live"), state: "busy", steering_available: true } }, order: ["a_live"], hydrated: true, hydrating: false });

    renderPanel("a_live");

    expect(await screen.findByRole("button", { name: "Steer" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
  });
});

// FS-03.A36 (R53, R54) — the address is the open file's single source of truth,
// so a reload reopens it and Close clears it, and the panel carries the state
// attribute the relaxed content width keys off.
describe("the open file on the agent route", () => {
  it("reopens the file named by ?file= and clears it on Close", async () => {
    useAgentStore.setState({ agents: { a_1: liveAgent("a_1") }, order: ["a_1"], hydrated: true });
    renderPanel("a_1", "?file=internal/state/messages.go&fileLine=12");

    // Reopening from the address is the reload case: the viewer reads the file
    // itself, so its header is the witness that the parameter was honored.
    expect(await screen.findByText("internal/state/messages.go")).toBeInTheDocument();
    expect(document.querySelector('[data-ui="agent-workspace"]')).toHaveAttribute("data-file-open", "true");

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    await waitFor(() => expect(screen.queryByText("internal/state/messages.go")).not.toBeInTheDocument());
    expect(document.querySelector('[data-ui="agent-workspace"]')).not.toHaveAttribute("data-file-open");
  });

  it("opens no viewer without ?file=", () => {
    useAgentStore.setState({ agents: { a_1: liveAgent("a_1") }, order: ["a_1"], hydrated: true });
    renderPanel("a_1");

    expect(screen.queryByRole("button", { name: "Reload" })).not.toBeInTheDocument();
    expect(document.querySelector('[data-ui="agent-workspace"]')).not.toHaveAttribute("data-file-open");
  });
});

// FS-03.A52, FS-21.R57: a member's chat keeps its ordinary header and shows a
// compact room cue plus a Think Tank tab linking the room and other members —
// also between turns, when the agent is idle.
it("shows a compact room cue and a Think Tank tab for room membership", async () => {
  const room = {
    version: 1, room_id: "tt_1", title: "Cache choice", goal: "A long goal about caches", origin_project: "app",
    phase: "discussion", control: "running", participants: ["Nova", "Ari"], revision: 2, created_at: "2026-10-07T00:00:00Z",
    updated_at: "2026-10-07T00:00:00Z", active_attempts: [], total_remaining: 3, judge_enabled: false,
    roster: [
      { agent_id: "a_room", name: "Nova", project: "app", role: "participant", state: "active", limit: 2, completed: 1, remaining: 1, exists: true },
      { agent_id: "a_ari", name: "Ari", project: "app", role: "participant", state: "active", limit: 2, completed: 0, remaining: 2, exists: true },
    ],
  };
  const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (String(input).startsWith("/api/think-tanks?agent_id=a_room")) return new Response(JSON.stringify({ version: 1, rooms: [room] }));
    return new Response("{}", { status: 404 });
  });
  useAgentStore.setState({ agents: { a_room: liveAgent("a_room") }, order: ["a_room"], hydrated: true, hydrating: false });
  mocks.useBackends.mockReturnValue({ data: backends });
  renderPanel("a_room");

  const cue = await screen.findByRole("link", { name: "Cache choice" });
  expect(cue.getAttribute("href")).toBe("/think-tank/tt_1");
  expect(screen.getByRole("heading", { name: "Nova" })).toBeInTheDocument();
  expect(screen.queryByText("A long goal about caches")).toBeNull();
  const tab = screen.getByRole("tab", { name: "Think Tank" });
  fireEvent.mouseDown(tab);
  fireEvent.click(tab);
  expect(screen.getByRole("link", { name: "Ari" }).getAttribute("href")).toBe("/agent/a_ari");
  expect(screen.queryByRole("link", { name: "Nova" })).toBeNull();
  expect(screen.queryByText("A long goal about caches")).toBeNull();
  expect(screen.queryByText(/most recent Think Tanks/)).toBeNull();
  fetchSpy.mockRestore();
});

it("keeps room navigation compact while exposing concurrent active and retained members", async () => {
  const room = {
    version: 1, room_id: "tt_compact", title: "Shared decision", goal: "long explanation", origin_project: "app",
    phase: "openings", control: "running", participants: ["Nova", "Ari", "Ari"], revision: 3,
    created_at: "2026-10-07T00:00:00Z", updated_at: "2026-10-07T00:00:00Z",
    active_attempts: [
      { attempt_id: "opening_a", agent_id: "a_ari", turn: "opening", state: "running", started_at: "2026-10-07T00:00:00Z" },
      { attempt_id: "opening_b", agent_id: "a_ari_2", turn: "opening", state: "running", started_at: "2026-10-07T00:00:00Z" },
    ],
    total_remaining: 4, judge_enabled: true,
    roster: [
      { agent_id: "a_room", name: "Nova", project: "app", role: "participant", state: "active", limit: 2, completed: 0, remaining: 2, exists: true },
      { agent_id: "a_ari", name: "Ari", project: "app", role: "participant", state: "active", limit: 2, completed: 0, remaining: 2, exists: true },
      { agent_id: "a_ari_2", name: "Ari", project: "ops", role: "participant", state: "active", limit: 2, completed: 0, remaining: 2, exists: true },
      { agent_id: "a_gone", name: "Lee", project: "old", role: "judge", state: "departed", limit: 0, completed: 0, remaining: 0, exists: false },
    ],
  };
  const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (String(input).startsWith("/api/think-tanks?agent_id=a_room")) return new Response(JSON.stringify({ version: 1, rooms: [room] }));
    return new Response("{}", { status: 404 });
  });
  useAgentStore.setState({ agents: { a_room: liveAgent("a_room") }, order: ["a_room"], hydrated: true, hydrating: false });
  renderPanel("a_room");

  const tab = await screen.findByRole("tab", { name: "Think Tank" });
  fireEvent.mouseDown(tab);
  fireEvent.click(tab);
  const nav = await screen.findByRole("navigation", { name: "Shared decision navigation" });
  expect(within(nav).getByRole("link", { name: "Shared decision" })).toHaveAttribute("href", "/think-tank/tt_compact");
  expect(screen.getByRole("link", { name: "Ari · app, taking a room turn" })).toHaveAttribute("href", "/agent/a_ari");
  expect(screen.getByRole("link", { name: "Ari · ops, taking a room turn" })).toHaveAttribute("href", "/agent/a_ari_2");
  expect(screen.getByText("Lee (unavailable)")).toBeInTheDocument();
  expect(screen.queryByText("long explanation")).toBeNull();
  expect(screen.queryByText(/turns left/)).toBeNull();
  expect(screen.queryByText("Judge")).toBeNull();
  fetchSpy.mockRestore();
});

// TS-14.R27: a clipped membership list says so in the Think Tank tab.
it("reports a clipped membership list in the Think Tank tab", async () => {
  const room = {
    version: 1, room_id: "tt_new", title: "Newest room", goal: "g", origin_project: "app",
    phase: "ended", control: "running", participants: ["Nova"], revision: 2, created_at: "2026-10-07T00:00:00Z",
    updated_at: "2026-10-07T00:00:00Z", active_attempts: [], total_remaining: 0, judge_enabled: false,
    roster: [{ agent_id: "a_room", name: "Nova", project: "app", role: "participant", state: "active", limit: 2, completed: 2, remaining: 0, exists: true }],
  };
  const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
    if (String(input).startsWith("/api/think-tanks?agent_id=a_room")) return new Response(JSON.stringify({ version: 1, rooms: [room], clipped: true }));
    return new Response("{}", { status: 404 });
  });
  useAgentStore.setState({ agents: { a_room: liveAgent("a_room") }, order: ["a_room"], hydrated: true, hydrating: false });
  mocks.useBackends.mockReturnValue({ data: backends });
  renderPanel("a_room");

  const tab = await screen.findByRole("tab", { name: "Think Tank" });
  fireEvent.mouseDown(tab);
  fireEvent.click(tab);
  expect(await screen.findByText("Showing the 1 most recent Think Tanks; older ones are not listed here.")).toBeInTheDocument();
  fetchSpy.mockRestore();
});

describe("ChatPanel stopped states", () => {
  afterEach(() => mocks.resumeAgent.mockReset());

  // FS-07.R24: a stopped terminal agent opens no PTY bridge, says it is stopped,
  // and offers Resume; the bridge mounts again once the agent is running.
  it("shows a stopped terminal agent with Resume instead of a dead terminal bridge", async () => {
    mocks.resumeAgent.mockResolvedValue({ resumed: true });
    const stopped = { ...liveAgent("a_term"), interface: "terminal", running: false, state: "done" };
    useAgentStore.setState({ agents: { a_term: stopped }, order: ["a_term"], hydrated: true, hydrating: false });

    renderPanel("a_term");

    expect(screen.queryByTestId("terminal-bridge")).not.toBeInTheDocument();
    expect(screen.queryByText("Terminal agents receive input in the terminal tab.")).not.toBeInTheDocument();
    expect(screen.getByText("This terminal agent is stopped. Resume to restart it.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Resume" }));
    await waitFor(() => expect(mocks.resumeAgent).toHaveBeenCalledWith("a_term"));

    useAgentStore.setState({ agents: { a_term: { ...stopped, running: true, state: "idle" } } });
    expect(await screen.findByTestId("terminal-bridge")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Resume" })).not.toBeInTheDocument();
  });

  it("surfaces a rejected Resume", async () => {
    mocks.resumeAgent.mockRejectedValue(new Error("resume refused"));
    useAgentStore.setState({ agents: { a_term: { ...liveAgent("a_term"), interface: "terminal", running: false, state: "done" } }, order: ["a_term"], hydrated: true, hydrating: false });

    renderPanel("a_term");
    fireEvent.click(screen.getByRole("button", { name: "Resume" }));

    await waitFor(() => expect(useUiStore.getState().toasts.at(-1)).toMatchObject({ title: "Resume failed", body: "resume refused" }));
  });

  // FS-03.R89: a chat agent whose process died on error reads "Error" with its
  // error text and says that a message or Resume restarts it.
  it("distinguishes a chat agent stopped on error and says how to restart it", () => {
    useAgentStore.setState({ agents: { a_err: { ...liveAgent("a_err"), running: false, state: "error", detail: "process exited" } }, order: ["a_err"], hydrated: true, hydrating: false });

    renderPanel("a_err");

    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.queryByText("Stopped")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Stopped on error: process exited. Send a message or Resume to restart it.");
    expect(screen.getByRole("button", { name: "Resume" })).toBeInTheDocument();
  });
});
