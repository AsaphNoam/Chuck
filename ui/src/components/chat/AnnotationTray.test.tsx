import React from "react";
import { readFileSync } from "node:fs";
import { join } from "node:path";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { useAgentStore } from "../../store/agentStore";
import { useAnnotationStore } from "../../store/annotationStore";
import { AnnotationTray } from "./AnnotationTray";
import { RoomAnnotationTray } from "../../features/thinktank/RoomAnnotationTray";
import { roomAnnotationSource } from "../../features/thinktank/roomText";
import { thinkTankDetailSchema } from "../../schemas/thinkTank";
import roomFixture from "../../features/thinktank/fixtures/room.json";

let launches = 0;
vi.mock("../../features/launch/NewAgentModal", async () => {
  const React = await import("react");
  return { NewAgentModal: ({ open, onLaunched }: { open: boolean; onLaunched: (id: string) => void }) =>
    open ? React.createElement("button", { onClick: () => { launches++; onLaunched("new-agent"); } }, "Launch agent") : null };
});

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterAll(() => server.close());

afterEach(() => {
  cleanup();
  useAgentStore.setState({ agents: {} });
  useAnnotationStore.setState({ bySource: {}, overallBySource: {}, editedAt: {}, collapsedBySource: {}, targetBySource: {}, recipientBySource: {}, recordSourceBySource: {} });
  server.resetHandlers();
  launches = 0;
});

const sourceAgent = {
  agent_id: "source", name: "Atlas", role: "implementer", project: "app", backend: "claude", model: "sonnet",
  interface: "chat", created_at: "2026-07-26T00:00:00Z", running: true, state: "idle", detail: "", context_pct: 0, updated_at: 1,
};

function seedTray(count: number) {
  useAgentStore.setState({ agents: { source: sourceAgent } });
  useAnnotationStore.setState({
    bySource: { source: Array.from({ length: count }, (_, i) => ({ seq: i + 1, excerpt: `Excerpt ${i + 1}`, instruction: `Instruction ${i + 1}` })) },
    overallBySource: {}, editedAt: { source: Date.now() }, collapsedBySource: {},
  });
}

function renderTray(sourceActive = true) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
  return render(<QueryClientProvider client={client}><AnnotationTray sourceId="source" sourceActive={sourceActive} /></QueryClientProvider>);
}

function agentStylesheet() {
  return readFileSync(join(process.cwd(), "src/styles/features/agent.css"), "utf8");
}

describe("AnnotationTray", () => {
  it("offers the recording checkbox only for assignments and names a zero-sequence destination", async () => {
    seedTray(1);
    useAgentStore.setState({ agents: { source: sourceAgent, peer: { ...sourceAgent, agent_id: "peer", name: "Peer" } } });
    let submitted: unknown;
    server.use(http.post("/api/sessions/source/annotations", async ({ request }) => {
      submitted = await request.json();
      return HttpResponse.json({ accepted: true, seq: 0 }, { status: 202 });
    }));
    renderTray();
    expect(screen.queryByLabelText("Include in current agent's transcript")).toBeNull();
    fireEvent.click(screen.getByLabelText("Another agent"));
    const checkbox = screen.getByLabelText("Include in current agent's transcript");
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    fireEvent.click(screen.getByLabelText("Current agent"));
    expect(screen.queryByLabelText("Include in current agent's transcript")).toBeNull();
    fireEvent.click(screen.getByLabelText("Another agent"));
    expect(screen.getByLabelText("Include in current agent's transcript")).not.toBeChecked();
    fireEvent.change(screen.getByLabelText("Annotation recipient"), { target: { value: "peer" } });
    fireEvent.click(screen.getByRole("button", { name: "Send annotations" }));
    expect(await screen.findByRole("status")).toHaveTextContent("Annotations sent to Peer.");
    expect(submitted).toMatchObject({ omit_source_record: true, target: { kind: "agent", agent_id: "peer" } });
  });
  // FS-13.R34/A24: empty -> capture is a new opening, while adding to a
  // nonempty batch preserves the chosen scope, instructions and recipient.
  it.each(["agent", "room"])("resets outside recipients when a fresh %s batch opens", (kind) => {
    const room = thinkTankDetailSchema.parse(roomFixture.room);
    const sourceId = kind === "agent" ? "source" : roomAnnotationSource(room.room_id);
    const project = kind === "agent" ? "app" : room.origin_project;
    useAgentStore.setState({ agents: {
      source: { ...sourceAgent, project },
      near: { ...sourceAgent, agent_id: "near", name: "Near", project },
      far: { ...sourceAgent, agent_id: "far", name: "Far", project: "outside" },
    } });
    act(() => { useAnnotationStore.getState().add(sourceId, { seq: 1, excerpt: "First", instruction: "Keep" }); });
    const client = new QueryClient({ defaultOptions: { queries: { retry: 0 } } });
    render(<QueryClientProvider client={client}>{kind === "agent"
      ? <AnnotationTray sourceId={sourceId} sourceActive />
      : <RoomAnnotationTray room={room} />}</QueryClientProvider>);
    fireEvent.click(screen.getByLabelText(kind === "agent" ? "Another agent" : "An agent"));
    fireEvent.click(screen.getByLabelText("Include other projects"));
    fireEvent.change(screen.getByLabelText("Annotation recipient"), { target: { value: "far" } });
    act(() => { useAnnotationStore.getState().add(sourceId, { seq: 2, excerpt: "Second", instruction: "Also keep" }); });
    expect(screen.getByLabelText("Include other projects")).toBeChecked();
    expect(screen.getByLabelText("Annotation recipient")).toHaveValue("far");
    expect(screen.getAllByLabelText("Instruction")[0]).toHaveValue("Keep");
    fireEvent.click(screen.getByRole("button", { name: "Discard all" }));
    expect(screen.queryByLabelText("Annotation recipient")).toBeNull();
    act(() => { useAnnotationStore.getState().add(sourceId, { seq: 3, excerpt: "New capture", instruction: "New instruction" }); });
    expect(screen.getByLabelText("Include other projects")).not.toBeChecked();
    expect(screen.getByLabelText("Annotation recipient")).toHaveValue("");
    expect(screen.queryByRole("option", { name: /Far/ })).toBeNull();
    expect(screen.getByLabelText("Instruction")).toHaveValue("New instruction");
  });
  it("focuses a new instruction and preserves focus while editing", () => {
    seedTray(1);
    renderTray();
    expect(screen.getByLabelText("Instruction")).toHaveFocus();
    const overall = screen.getByLabelText("Overall instruction (optional)");
    overall.focus();
    fireEvent.change(overall, { target: { value: "Keep this focus" } });
    expect(overall).toHaveFocus();
    act(() => { useAnnotationStore.getState().add("source", { seq: 2, excerpt: "Next", instruction: "" }); });
    expect(screen.getAllByLabelText("Instruction")[1]).toHaveFocus();
  });

  it("reveals and focuses a captured instruction when its source tray was collapsed", () => {
    seedTray(1);
    act(() => { useAnnotationStore.getState().setCollapsed("source", true); });
    act(() => { useAnnotationStore.getState().setCollapsed("other-source", true); });
    const { container } = renderTray();
    expect(container.querySelector(".annotation-tray")).toHaveAttribute("data-state", "collapsed");

    act(() => { useAnnotationStore.getState().add("source", { seq: 2, excerpt: "Captured event", instruction: "" }); });

    expect(container.querySelector(".annotation-tray")).toHaveAttribute("data-state", "expanded");
    expect(screen.getAllByLabelText("Instruction")[1]).toHaveFocus();
    expect(useAnnotationStore.getState().collapsedBySource.source).toBeUndefined();
    expect(useAnnotationStore.getState().collapsedBySource["other-source"]).toBe(true);
  });

  it("validates the batch before opening launch, leaving an invalid tray intact", async () => {
    seedTray(1);
    useAnnotationStore.getState().updateInstruction("source", 0, "  ");
    let preflights = 0;
    server.use(http.post("/api/sessions/source/annotations/validate", () => {
      preflights++;
      return HttpResponse.json({ error: { code: "validation", message: "annotation 1 needs an anchor, excerpt, and instruction" } }, { status: 422 });
    }));
    renderTray();
    fireEvent.click(screen.getByLabelText("New agent"));
    fireEvent.click(screen.getByRole("button", { name: "Continue to launch" }));
    expect(await screen.findByText("annotation 1 needs an anchor, excerpt, and instruction")).toBeInTheDocument();
    expect(preflights).toBe(1);
    expect(launches).toBe(0);
    expect(screen.queryByRole("button", { name: "Launch agent" })).toBeNull();
    expect(useAnnotationStore.getState().bySource.source).toHaveLength(1);
  });

  it("retries failed post-launch delivery to the same agent without launching another", async () => {
    seedTray(1);
    const recipientIds: string[] = [];
    server.use(
      http.post("/api/sessions/source/annotations/validate", () => HttpResponse.json({ valid: true })),
      http.post("/api/sessions/source/annotations", async ({ request }) => {
        const body = await request.json() as { target: { agent_id: string } };
        recipientIds.push(body.target.agent_id);
        return recipientIds.length === 1
          ? HttpResponse.json({ error: { code: "internal", message: "mail unavailable" } }, { status: 500 })
          : HttpResponse.json({ accepted: true, seq: 1 }, { status: 202 });
      }),
    );
    renderTray();
    fireEvent.click(screen.getByLabelText("New agent"));
    fireEvent.click(screen.getByRole("button", { name: "Continue to launch" }));
    fireEvent.click(await screen.findByRole("button", { name: "Launch agent" }));
    expect(await screen.findByText(/Agent new-agent launched, but annotation delivery failed/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry delivery to new-agent" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry delivery to new-agent" }));
    await waitFor(() => expect(useAnnotationStore.getState().bySource.source).toBeUndefined());
    expect(launches).toBe(1);
    expect(recipientIds).toEqual(["new-agent", "new-agent"]);
    // Acknowledged delivery ends the retry: the next batch's New task launches again.
    act(() => { useAnnotationStore.getState().add("source", { seq: 9, excerpt: "Next", instruction: "Again" }); });
    expect(await screen.findByRole("button", { name: "Continue to launch" })).toBeInTheDocument();
  });

  // FS-13.R15: an archived source cannot be the target, so the tray defaults to
  // an available target instead of a checked-but-disabled Current agent.
  // FS-13.A24 — the recipient list starts at the source project; outside agents
  // appear only on request, and hiding them clears an outside choice.
  it("lists the source project's recipients until other projects are included", () => {
    seedTray(1);
    useAgentStore.setState({ agents: {
      source: sourceAgent,
      near: { ...sourceAgent, agent_id: "near", name: "Near" },
      far: { ...sourceAgent, agent_id: "far", name: "Far", project: "other" },
    } as never });
    renderTray();
    fireEvent.click(screen.getByLabelText(/Another agent/));
    const recipient = () => screen.getByLabelText("Annotation recipient") as HTMLSelectElement;
    const names = () => [...recipient().options].map((option) => option.textContent);
    expect(names()).toEqual(["Choose an agent", expect.stringContaining("Near")]);
    fireEvent.click(screen.getByLabelText("Include other projects"));
    expect(names()).toContainEqual(expect.stringContaining("Far ("));
    expect(names().find((name) => name?.startsWith("Far"))).toContain("@other");
    fireEvent.change(recipient(), { target: { value: "far" } });
    fireEvent.click(screen.getByLabelText("Include other projects"));
    expect(recipient()).toHaveValue("");
    expect(names()).not.toContainEqual(expect.stringContaining("Far"));
    expect(screen.getByLabelText("Instruction")).toHaveValue("Instruction 1");
  });

  it("defaults an archived source to another running agent", () => {
    seedTray(1);
    useAgentStore.setState({ agents: {
      source: { ...sourceAgent, running: false },
      peer: { ...sourceAgent, agent_id: "peer", name: "Peer" },
    } });
    renderTray(false);
    expect(screen.getByLabelText("Current agent")).toBeDisabled();
    expect(screen.getByLabelText("Current agent")).not.toBeChecked();
    expect(screen.getByLabelText("Another agent")).toBeChecked();
    expect(screen.getByRole("combobox", { name: "Annotation recipient" })).toBeInTheDocument();
  });

  it("defaults an archived source with no running peers to a new task", () => {
    seedTray(1);
    useAgentStore.setState({ agents: { source: { ...sourceAgent, running: false } } });
    renderTray(false);
    expect(screen.getByLabelText("Current agent")).not.toBeChecked();
    expect(screen.getByLabelText("New agent")).toBeChecked();
    expect(screen.getByRole("button", { name: "Continue to launch" })).toBeInTheDocument();
  });

  it("keeps delivery controls outside the scrolling multi-draft body", () => {
    seedTray(3);
    const { container } = renderTray();

    const body = container.querySelector(".annotation-tray-body");
    const footer = container.querySelector(".annotation-tray-footer");
    expect(body).not.toBeNull();
    expect(footer).not.toBeNull();
    expect(within(body as HTMLElement).getAllByText(/Excerpt/)).toHaveLength(3);
    expect(within(footer as HTMLElement).getByRole("button", { name: "Send annotations" })).toBeInTheDocument();
    expect(screen.getByText("3/20")).toBeInTheDocument();
  });

  // FS-13.A12. jsdom evaluates no CSS and knows nothing of container queries, so
  // the stylesheet is the only witness that a wide transcript region docks the
  // tray and a narrow one keeps the shipped overlay (INV §13). What matters is
  // that the threshold is asked of the transcript region — a viewport query would
  // dock the dashboard's narrow chat pane inside a wide window.
  it("docks through the transcript region's own container query and falls back to the overlay", () => {
    const css = agentStylesheet();
    expect(css).toMatch(/\.transcript-wrap \{[^}]*container: transcript \/ inline-size/);
    expect(css).toMatch(/\.transcript-wrap \{[^}]*grid-template-columns: auto minmax\(0, 1fr\) auto/);
    // The fallback is the shipped presentation itself, not a second one.
    expect(css).toMatch(/\.annotation-tray \{[^}]*position: absolute/);
    const docked = css.match(/@container transcript \(min-width: 860px\) \{([\s\S]*?)\n\}/);
    expect(docked).not.toBeNull();
    expect(docked![1]).toMatch(/\.annotation-tray \{[^}]*position: static/);
    expect(docked![1]).toMatch(/\.annotation-tray \{[^}]*width: min\(30cqi, 460px\)/);
    // The region has a leading file-viewer track (FS-03.R53), so the in-flow
    // children are placed explicitly: auto-placement would drop the transcript
    // into the viewer's content-sized column. Every other child still has to stay
    // out of flow — giving the jump button a grid area of its own displaced the
    // transcript into the tray's column, invisible to jsdom and obvious in Chrome.
    expect(css).toMatch(/\.transcript-view \{[^}]*grid-column: 2/);
    // The overlay is placed against the whole region: the trailing track is
    // zero wide while the tray floats, and a containing block of that track
    // squeezed the overlay to a sliver. Only the docked form takes it.
    expect(css).toMatch(/\.annotation-tray \{[^}]*grid-column: 1 \/ -1/);
    expect(docked![1]).toMatch(/\.annotation-tray \{[^}]*grid-column: 3/);
    expect(css).toMatch(/\.jump-to-latest \{[^}]*position: absolute/);
    expect(css).not.toMatch(/\.jump-to-latest \{[^}]*grid-area/);
    // The collapse control belongs to the docked form only (FS-13.R21).
    expect(css).toMatch(/\.annotation-tray-collapse \{[^}]*display: none/);
    expect(docked![1]).toMatch(/\.annotation-tray-collapse \{[^}]*display: block/);
    // The collapsed strip keeps the count and the control, and drops the rest.
    expect(docked![1]).toMatch(/\.annotation-tray\[data-state="collapsed"\] \.annotation-tray-body/);
    expect(docked![1]).toMatch(/\.annotation-tray\[data-state="collapsed"\] \.annotation-tray-title/);
  });

  // FS-13.A12: the tray is one component in two CSS states, so collapsing is a
  // state on the same markup rather than a second tray. The drafts stay mounted,
  // which is what lets a resize move between the forms without touching them.
  it("collapses and expands the docked tray without disturbing the drafts", () => {
    seedTray(3);
    const { container } = renderTray();

    const tray = container.querySelector(".annotation-tray") as HTMLElement;
    expect(tray).toHaveAttribute("data-state", "expanded");
    fireEvent.click(screen.getByRole("button", { name: "Collapse pending annotations" }));

    expect(container.querySelector(".annotation-tray")).toHaveAttribute("data-state", "collapsed");
    expect(useAnnotationStore.getState().collapsedBySource.source).toBe(true);
    // The strip still names how many drafts are waiting behind it.
    expect(screen.getByText("3/20")).toBeInTheDocument();
    expect(useAnnotationStore.getState().bySource.source).toHaveLength(3);

    fireEvent.click(screen.getByRole("button", { name: "Expand pending annotations" }));
    expect(container.querySelector(".annotation-tray")).toHaveAttribute("data-state", "expanded");
    expect(useAnnotationStore.getState().collapsedBySource.source).toBe(false);
  });

  // FS-13.A13 — the anchor is the row's heading, not bold text sharing a line
  // with the control that deletes the draft.
  it("gives each docked draft an anchor heading separate from its controls", () => {
    seedTray(1);
    renderTray();

    const heading = screen.getByRole("heading", { name: "Event 1" });
    expect(heading).toBeInTheDocument();
    expect(within(heading).queryByRole("button")).toBeNull();
    const row = heading.closest(".annotation-draft") as HTMLElement;
    expect(within(row).getByRole("button", { name: "Remove" })).toBeInTheDocument();
    expect(within(row).getByText("Excerpt 1")).toBeInTheDocument();
    expect(within(row).getByLabelText("Instruction")).toHaveValue("Instruction 1");
  });
});
