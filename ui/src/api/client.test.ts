import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { cancelTurn, launchAgent, resumeAgent, switchRuntime } from "./client";
import { useUiStore } from "../store/uiStore";

let cancelBody: unknown;
const server = setupServer(http.post("/api/sessions/a1/cancel", async ({ request }) => {
  cancelBody = await request.json();
  return HttpResponse.json({ cancelled: true });
}));

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  cancelBody = undefined;
  useUiStore.setState({ toasts: [] });
});
afterAll(() => server.close());

describe("cancelTurn", () => {
  it("sends an explicit turn target", async () => {
    await cancelTurn("a1", { target: "turn" });
    expect(cancelBody).toEqual({ target: "turn" });
  });

  it("sends the quota episode revision and preserves a server conflict reason", async () => {
    server.use(http.post("/api/sessions/a1/cancel", async ({ request }) => {
      cancelBody = await request.json();
      return HttpResponse.json({ error: { message: "quota continuation changed" } }, { status: 409 });
    }));
    await expect(cancelTurn("a1", { target: "quota_continuation", episode_id: "qe_1", revision: 4 })).rejects.toThrow("quota continuation changed");
    expect(cancelBody).toEqual({ target: "quota_continuation", episode_id: "qe_1", revision: 4 });
  });
});

// FS-19.R7: every start path reports a recreated worktree checkout instead of
// silently re-materializing it.
describe("worktree recreation notice", () => {
  const notice = { recreated: true, branch: "feature/x", warning: "setup exited 1" };

  it.each([
    ["launch", "/api/sessions", () => launchAgent({ role: "r", project: "p" })],
    ["resume", "/api/sessions/a1/resume", () => resumeAgent("a1")],
    ["switch", "/api/sessions/a1/switch-runtime", () => switchRuntime("a1", { backend: "codex" })],
  ])("%s pushes a notice naming the branch and setup warning", async (_name, path, call) => {
    server.use(http.post(path, () => HttpResponse.json({ agent: { agent_id: "a1", name: "Atlas" }, worktree: notice })));
    await call();
    const toasts = useUiStore.getState().toasts;
    expect(toasts).toHaveLength(1);
    expect(toasts[0]).toMatchObject({ title: "Worktree checkout recreated", agentId: "a1" });
    expect(toasts[0].body).toMatch(/feature\/x.*setup exited 1/);
  });

  it("stays quiet on an ordinary start", async () => {
    server.use(http.post("/api/sessions", () => HttpResponse.json({ agent: { agent_id: "a1", name: "Atlas" } })));
    await launchAgent({ role: "r", project: "p" });
    expect(useUiStore.getState().toasts).toEqual([]);
  });
});
