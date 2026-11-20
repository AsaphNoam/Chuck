import { act, renderHook } from "@testing-library/react";
import { HttpResponse, http } from "msw";
import { setupServer } from "msw/node";
import { afterAll, afterEach, beforeAll, expect, it, vi } from "vitest";
import { getChatDraft, setChatDraft } from "./drafts";
import { chatDraftRequestId, clearAttachmentDraftCacheForTests, restorePendingAttachmentSend } from "./attachments";
import { useInputDelivery } from "./useInputDelivery";
import { reconcileRecoverySettlement } from "../../lib/heldMessage";
import { OUTCOME_TOAST_MS, useUiStore } from "../../store/uiStore";

const server = setupServer();
beforeAll(() => server.listen({ onUnhandledRequest: "error" }));
afterEach(() => {
  server.resetHandlers();
  localStorage.clear();
  clearAttachmentDraftCacheForTests();
});
afterAll(() => server.close());

const outcomes = ["accepted", "held", "queued", "released", "uncertain", "refused"] as const;

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
    const setAttachmentError = vi.fn();
    const rejected = outcome === "uncertain" || outcome === "refused";
    const retained = rejected || outcome === "released";
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
      phone, clearDraft, setError, setNotice, setAttachmentError,
    }));
    await act(async () => result.current.deliver(text, []));
    expect(request).toHaveBeenCalledTimes(1);
    expect(request).toHaveBeenCalledWith({ text, request_id: expect.any(String) });
    expect(getChatDraft(agentId)).toBe(retained ? text : "");
    expect(projectPrompt).toHaveBeenCalledTimes(phone || retained ? 0 : 1);
    if (!phone && !retained) expect(projectPrompt.mock.calls[0][2]).toBe(5);
    expect(refreshRecovery).toHaveBeenCalledTimes(outcome === "queued" || outcome === "uncertain" || outcome === "released" ? 1 : 0);
    if (outcome === "queued" && phone) expect(setNotice).toHaveBeenLastCalledWith(expect.stringContaining("Queued"));
    // Desktop outcomes are 3s toasts (FS-03.R50).
    if (outcome === "queued" && !phone) expect(useUiStore.getState().toasts.at(-1)).toMatchObject({ title: expect.stringContaining("Queued"), durationMs: OUTCOME_TOAST_MS });
    if (outcome === "held" && phone) expect(setNotice).toHaveBeenLastCalledWith(expect.stringContaining("Held"));
    if (retained) expect(setError).toHaveBeenLastCalledWith(expect.any(String));
  },
);

it.each([false, true])("settles an attachment Send after a lost response, with newer draft=%s", async (newer) => {
  const agentId = "attachment-settlement";
  const text = "Original attachment message";
  setChatDraft(agentId, text);
  const requestId = chatDraftRequestId(agentId);
  const files = [{ id: "file-1", name: "notes.txt", size: 5, type: "text/plain", blob: new Blob(["notes"]) }];
  server.use(
    http.post("/api/sessions/:id/prompt", () => HttpResponse.error()),
    http.get("/api/sessions/:id/prompt", ({ request }) => HttpResponse.json({
      recovery_status: new URL(request.url).searchParams.get("recovery_request_id") === requestId ? "settled" : "unavailable",
    })),
  );
  const setAttachmentError = vi.fn();
  const mount = () => renderHook(() => useInputDelivery(agentId, {
    acceptSteer: vi.fn(), refreshRecovery: vi.fn(),
  }, { clearDraft: vi.fn(), setError: vi.fn(), setNotice: vi.fn(), setAttachmentError }));
  const first = mount();
  await act(async () => first.result.current.deliver(text, files));
  expect(first.result.current.retry?.requestId).toBe(requestId);
  expect(first.result.current.retry?.retryable).not.toBe(false);
  if (newer) setChatDraft(agentId, "New unrelated text");
  first.unmount();

  const reopened = mount();
  await act(async () => reconcileRecoverySettlement(agentId));
  expect(reopened.result.current.retry).toBeNull();
  expect(await restorePendingAttachmentSend(agentId)).toBeNull();
  expect(getChatDraft(agentId)).toBe(newer ? "New unrelated text" : "");
});
