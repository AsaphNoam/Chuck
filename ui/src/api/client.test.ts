import { afterAll, afterEach, beforeAll, describe, expect, it } from "vitest";
import { http, HttpResponse } from "msw";
import { setupServer } from "msw/node";
import { cancelTurn } from "./client";

let cancelBody: unknown;
const server = setupServer(http.post("/api/sessions/a1/cancel", async ({ request }) => {
  cancelBody = await request.json();
  return HttpResponse.json({ cancelled: true });
}));

beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  cancelBody = undefined;
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
