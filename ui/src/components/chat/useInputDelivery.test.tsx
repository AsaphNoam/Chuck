import { act, renderHook } from "@testing-library/react";
import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, expect, it, vi } from "vitest";
import { getChatDraft, setChatDraft } from "./drafts";
import { useInputDelivery } from "./useInputDelivery";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  localStorage.clear();
});
afterAll(() => server.close());

const outcomes = ["accepted", "held", "queued", "uncertain", "refused"] as const;

it.each(outcomes.flatMap((outcome) => [false, true].map((phone) => ({ outcome, phone }))))(
  "applies $outcome with phone=$phone",
  async ({ outcome, phone }) => {
    const agentId = "delivery-test";
    const text = "Retain this draft until accepted.";
    const request = vi.fn();
    const projectPrompt = vi.fn();
    const refreshRecovery = vi.fn();
    const setError = vi.fn();
    const setNotice = vi.fn();
    const clearDraft = vi.fn();
    const rejected = outcome === "uncertain" || outcome === "refused";
    server.use(http.post("/api/sessions/:id/prompt", async ({ request: incoming }) => {
      const body = await incoming.json() as { text: string; request_id: string };
      request(body);
      if (rejected) return HttpResponse.json({
        error: {
          code: outcome === "uncertain" ? "delivery_uncertain" : "turn_not_running",
          message: outcome === "uncertain" ? "Delivery uncertain" : "Turn stopped",
        },
      }, { status: 409 });
      return HttpResponse.json({
        accepted: true, agent_id: agentId, after_seq: 7, request_id: body.request_id,
        delivery: outcome === "accepted" ? "sent" : outcome,
      });
    }));
    setChatDraft(agentId, text);
    const { result } = renderHook(() => useInputDelivery(agentId, {
      beforeSend: phone ? undefined : () => 5,
      acceptPrompt: phone ? undefined : projectPrompt,
      acceptSteer: vi.fn(),
      refreshRecovery,
    }, {
      phone, clearDraft, setError, setNotice, setAttachmentError: vi.fn(),
    }));
    await act(async () => result.current.deliver(text, []));
    expect(request).toHaveBeenCalledTimes(1);
    expect(request).toHaveBeenCalledWith({ text, request_id: expect.any(String) });
    expect(getChatDraft(agentId)).toBe(rejected ? text : "");
    expect(projectPrompt).toHaveBeenCalledTimes(phone || rejected ? 0 : 1);
    if (!phone && !rejected) expect(projectPrompt.mock.calls[0][2]).toBe(5);
    expect(refreshRecovery).toHaveBeenCalledTimes(outcome === "queued" || outcome === "uncertain" ? 1 : 0);
    if (outcome === "queued") expect(setNotice).toHaveBeenLastCalledWith(expect.stringContaining("Queued"));
    if (outcome === "held" && phone) expect(setNotice).toHaveBeenLastCalledWith(expect.stringContaining("Held"));
    if (rejected) expect(setError).toHaveBeenLastCalledWith(expect.any(String));
  },
);
