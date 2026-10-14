import React from "react";
import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { RoomList } from "./RoomList";
import fixture from "./fixtures/room.json";

// The summary comes from the Go-encoded room fixture (TS-14.R27).
const summary = () => structuredClone(fixture.room);
let rooms: unknown[] = [];
const server = setupServer(
  http.get("/api/projects", () => HttpResponse.json({ alpha: { title: "Alpha", cwd: "/tmp" } })),
  http.get("/api/think-tanks", () => HttpResponse.json({ version: 1, rooms, clipped: false })),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => cleanup());
afterAll(() => server.close());

function renderList(project: string | null = "alpha") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter><RoomList project={project ?? undefined} /></MemoryRouter>
    </QueryClientProvider>,
  );
}

// FS-02.A53, FS-21.R44: a titled card with the whole roster, each member's
// remaining ceiling, separate room and participant links, the current
// speaker and attention; a deleted source keeps its name without a link.
describe("RoomList cards", () => {
  it("shows title, roster allowances, speaker and attention", async () => {
    rooms = [{ ...summary(), active_attempts: [{ attempt_id: "tta_1", agent_id: "a_one", turn: "discussion", state: "running", started_at: "2026-10-07T00:00:00Z" }] }];
    renderList();
    const card = (await screen.findByRole("link", { name: "Cache choice" })).closest("li")!;
    expect(screen.getByRole("link", { name: "Cache choice" }).getAttribute("href")).toBe("/think-tank/tt_fixture");
    const roster = within(card as HTMLElement).getByRole("list", { name: "Participants" });
    expect(within(roster).getByRole("link", { name: "Ari" }).getAttribute("href")).toBe("/agent/a_one");
    expect(await within(roster).findByText(/Alpha · 2 turns left/)).toBeTruthy();
    expect(within(roster).getByText("Speaking")).toBeTruthy();
    expect(within(card as HTMLElement).getByText("Ari is speaking.")).toBeTruthy();
    expect(within(card as HTMLElement).getByText("Active")).toBeTruthy();
    const budget = card.querySelector(".room-card-budget") as HTMLElement;
    expect(budget.textContent).toContain("2turns remaining");

    cleanup();
    rooms = [{ ...summary(), hold: "Ari's turn failed.", roster: summary().roster.map((m) => ({ ...m, exists: m.agent_id === "a_one" })) }];
    renderList();
    expect(await screen.findByText("Ari's turn failed.")).toBeTruthy();
    expect(screen.getByText("Needs attention")).toBeTruthy();
    expect(screen.getByText("Gone")).toBeTruthy();
    expect(screen.getByText(/agent deleted/)).toBeTruthy();
    expect(screen.queryByRole("link", { name: "Gone" })).toBeNull();
  });

  it("keeps the unused collective allowance on an ended card", async () => {
    rooms = [{ ...summary(), phase: "ended" }];
    renderList();
    const card = (await screen.findByRole("link", { name: "Cache choice" })).closest("li")!;
    const budget = card.querySelector(".room-card-budget") as HTMLElement;
    expect(budget.textContent).toContain("2unused turns");
    expect(within(card as HTMLElement).getByText("Ended")).toBeTruthy();
    expect(within(card as HTMLElement).getByText(/2 turns left/)).toBeTruthy();
    expect(within(card as HTMLElement).queryByText("turns remaining")).toBeNull();
  });

  // FS-21.R39: Archive discovery names the origin, including a removed project.
  it("names a removed origin project outside the project page", async () => {
    rooms = [{ ...summary(), origin_project: "ghost" }];
    renderList(null);
    const card = (await screen.findByRole("link", { name: "Cache choice" })).closest("li")!;
    expect((await within(card as HTMLElement).findByText(/project removed/)).textContent).toBe("Origin: ghost · project removed");
  });

  // FS-02.R71: the heading collapses the cards, keeps the count and
  // remembers the choice for the next room list.
  it("collapses the room cards and remembers the choice", async () => {
    window.localStorage.clear();
    rooms = [summary()];
    renderList();
    const toggle = await screen.findByRole("button", { name: "Thinking together" });
    expect(toggle.getAttribute("aria-expanded")).toBe("true");
    fireEvent.click(toggle);
    expect(toggle.getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByRole("link", { name: "Cache choice" })).toBeNull();
    expect(screen.getByText(/1 room · separate from agent sessions/)).toBeTruthy();

    cleanup();
    renderList();
    expect((await screen.findByRole("button", { name: "Thinking together" })).getAttribute("aria-expanded")).toBe("false");
    fireEvent.click(screen.getByRole("button", { name: "Thinking together" }));
    expect(await screen.findByRole("link", { name: "Cache choice" })).toBeTruthy();
    expect(window.localStorage.getItem("chuck.think-tank-rooms-collapsed")).toBeNull();
  });

  // FS-02.R71, FS-05.R39: one browser preference collapses every project's
  // room section and Archive's.
  it("shares the collapsed choice across projects and Archive", async () => {
    window.localStorage.clear();
    rooms = [summary()];
    renderList();
    fireEvent.click(await screen.findByRole("button", { name: "Thinking together" }));

    cleanup();
    renderList("beta");
    expect((await screen.findByRole("button", { name: "Thinking together" })).getAttribute("aria-expanded")).toBe("false");

    cleanup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter><RoomList title="Retained room history" emptyText="No Think Tanks yet." /></MemoryRouter>
      </QueryClientProvider>,
    );
    expect((await screen.findByRole("button", { name: "Retained room history" })).getAttribute("aria-expanded")).toBe("false");
    expect(screen.queryByRole("link", { name: "Cache choice" })).toBeNull();
    window.localStorage.clear();
  });

  // FS-21.R44: a failed read stays visible while the cards are collapsed,
  // both on first load and when a refetch fails over cached rooms.
  it("shows a failed read while collapsed", async () => {
    window.localStorage.setItem("chuck.think-tank-rooms-collapsed", "1");
    server.use(http.get("/api/think-tanks", () => HttpResponse.json({ error: { code: "internal", message: "down" } }, { status: 500 })));
    renderList();
    expect((await screen.findByText("Could not load Think Tanks.")).closest("[hidden]")).toBeNull();
    expect(screen.getByRole("button", { name: "Thinking together" }).getAttribute("aria-expanded")).toBe("false");
    server.resetHandlers();

    cleanup();
    rooms = [summary()];
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <QueryClientProvider client={client}>
        <MemoryRouter><RoomList project="alpha" /></MemoryRouter>
      </QueryClientProvider>,
    );
    expect(await screen.findByText(/1 room · separate from agent sessions/)).toBeTruthy();
    expect(screen.queryByText("Could not load Think Tanks.")).toBeNull();
    server.use(http.get("/api/think-tanks", () => HttpResponse.json({ error: { code: "internal", message: "down" } }, { status: 500 })));
    await client.refetchQueries();
    expect((await screen.findByText("Could not load Think Tanks.")).closest("[hidden]")).toBeNull();
    expect(screen.getByText(/1 room · separate from agent sessions/)).toBeTruthy();
    server.resetHandlers();
    window.localStorage.clear();
  });
});
