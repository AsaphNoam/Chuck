import React from "react";
import { afterAll, afterEach, beforeAll, expect, it } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { ThinkTankTab } from "./RoomTurnNotice";
import fixture from "./fixtures/room.json";

// FS-21.A40: the agent-chat Think Tank tab is compact navigation. Rooms come
// from the Go-encoded fixture (TS-14.R27) with only the roster varied.
type Room = typeof fixture.room;
const me = { agent_id: "a_me", name: "Nova", project: "alpha", role: "participant", state: "active", limit: 3, completed: 1, remaining: 2, exists: true };
const judge = { agent_id: "a_judge", name: "Lee", project: "alpha", role: "judge", state: "active", limit: 0, completed: 0, remaining: 0, exists: true };
const attempt = (agentId: string) => ({ attempt_id: "at_" + agentId, agent_id: agentId, turn: "discussion", state: "running", started_at: "2026-10-06T09:00:00Z" });

function room(overrides: Partial<Room> & { active_attempts?: unknown[] } = {}): Room {
  const base = structuredClone(fixture.room);
  return { ...base, roster: [me, ...base.roster, judge], ...overrides } as Room;
}

let response: () => Response = () => HttpResponse.json({ version: 1, rooms: [] });
const server = setupServer(
  http.get("/api/projects", () => HttpResponse.json({ alpha: { title: "Alpha", cwd: "/tmp" } })),
  http.get("/api/think-tanks", () => response()),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => cleanup());
afterAll(() => server.close());

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter><ThinkTankTab agentId="a_me" /></MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
}

const rooms = (list: Room[]) => () => HttpResponse.json({ version: 1, rooms: list });

it("links the room and every other surviving member, omitting this agent", async () => {
  response = rooms([room()]);
  renderTab();
  expect(await screen.findByRole("link", { name: "Cache choice" })).toHaveAttribute("href", "/think-tank/tt_fixture");
  expect(screen.getByRole("link", { name: "Ari" })).toHaveAttribute("href", "/agent/a_one");
  expect(screen.getByRole("link", { name: "Lee" })).toHaveAttribute("href", "/agent/a_judge");
  expect(screen.queryByRole("link", { name: /Nova/ })).toBeNull();
  expect(screen.queryByRole("link", { name: /Gone/ })).toBeNull();
  expect(screen.getByText("Gone (unavailable)")).toBeInTheDocument();
  expect(screen.queryByRole("button")).toBeNull();
  expect(screen.queryByText(/Judge|turns left|Discussion/)).toBeNull();
});

it("moves active-turn emphasis with running attempts and clears it when settled", async () => {
  response = rooms([room({ active_attempts: [attempt("a_one")] })]);
  const client = renderTab();
  expect(await screen.findByRole("link", { name: "Ari, taking a room turn" })).toHaveAttribute("data-state", "active");
  expect(screen.getByRole("link", { name: "Lee" })).toHaveAttribute("data-state", "settled");

  response = rooms([room({ active_attempts: [attempt("a_judge")] })]);
  await client.invalidateQueries();
  expect(await screen.findByRole("link", { name: "Lee, taking a room turn" })).toHaveAttribute("data-state", "active");
  expect(screen.getByRole("link", { name: "Ari" })).toHaveAttribute("data-state", "settled");

  // The open agent's own running attempt never marks a peer active.
  response = rooms([room({ active_attempts: [attempt("a_me")] })]);
  await client.invalidateQueries();
  await waitFor(() => expect(screen.getByRole("link", { name: "Lee" })).toHaveAttribute("data-state", "settled"));
  expect(document.querySelector('[data-state="active"]')).toBeNull();
});

it("keeps one group per room, including ended rooms, with long names intact", async () => {
  const long = "A very long participant name that keeps going well past any compact width";
  response = rooms([
    room({ active_attempts: [attempt("a_one")] }),
    room({ room_id: "tt_old", title: "Old decision", phase: "ended", roster: [me, { ...fixture.room.roster[0], agent_id: "a_long", name: long }] }),
  ]);
  renderTab();
  expect(await screen.findByRole("link", { name: "Old decision" })).toHaveAttribute("href", "/think-tank/tt_old");
  expect(screen.getByRole("link", { name: long })).toHaveAttribute("data-state", "settled");
  expect(screen.getAllByRole("navigation")).toHaveLength(2);
});

it("surfaces failed and empty membership reads", async () => {
  response = () => HttpResponse.json({ error: { code: "internal", message: "boom" } }, { status: 500 });
  renderTab();
  expect(await screen.findByText("Could not load this agent's Think Tanks.")).toBeInTheDocument();
  cleanup();

  response = rooms([]);
  renderTab();
  expect(await screen.findByText("This agent is not in a Think Tank.")).toBeInTheDocument();
});
