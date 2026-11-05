import { afterEach, expect, it, vi } from "vitest";
import { getHeldPrompt, dismissPromptRecovery } from "../api/client";
import { getChatDraft, clearChatDraftCacheForTests } from "../components/chat/drafts";
import { useHeldStore } from "../store/heldStore";
import { readHeldRecovery, reconcileSteerRecovery, restoreWithdrawnDraft } from "./heldMessage";

vi.mock("../api/client", () => ({ getHeldPrompt: vi.fn(), dismissPromptRecovery: vi.fn(), withdrawPrompt: vi.fn() }));
afterEach(() => {
  vi.restoreAllMocks(); vi.resetAllMocks(); localStorage.clear(); clearChatDraftCacheForTests();
  useHeldStore.setState({ recoveryByAgent: {}, recoveryUnreadByAgent: {} });
});

// R100: the read starts during native uncertainty and finishes after success.
// An unrelated older recovery must survive both the success and that late read.
it("rejects a late pre-success recovery read and preserves unrelated recovery", async () => {
  const older = { request_id: "older", text: "older input", delivery: "unknown" };
  const transient = { request_id: "steered", text: "current input", delivery: "unknown" };
  useHeldStore.getState().setRecovery("agent", transient);
  let finish!: (snapshot: Awaited<ReturnType<typeof getHeldPrompt>>) => void;
  vi.mocked(getHeldPrompt).mockImplementationOnce(() => new Promise((resolve) => { finish = resolve; }));
  const pending = readHeldRecovery("agent");
  vi.mocked(getHeldPrompt).mockResolvedValue({ agent_id: "agent", text: "", after_seq: 0, recovery: older });
  await reconcileSteerRecovery("agent", "steered");
  finish({ agent_id: "agent", text: "", after_seq: 0, recovery: transient });
  await pending;
  expect(useHeldStore.getState().recoveryByAgent.agent).toEqual(older);
});

const withdrawn = { accepted: true as const, agent_id: "agent", withdrawn: true,
  message: { text: "withdrawn input", request_id: "exact-withdrawal", attachments: [] } };

it("acknowledges only the returned receipt after saving its draft", async () => {
  vi.mocked(dismissPromptRecovery).mockImplementation(async (agent, request) => {
    expect(getChatDraft(agent)).toBe("withdrawn input");
    expect(request).toBe("exact-withdrawal");
    return {};
  });
  await restoreWithdrawnDraft("agent", withdrawn);
  expect(dismissPromptRecovery).toHaveBeenCalledTimes(1);
});

it("keeps durable recovery when the restored draft cannot be saved", async () => {
  vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("storage full"); });
  await restoreWithdrawnDraft("agent", withdrawn);
  expect(dismissPromptRecovery).not.toHaveBeenCalled();
});

it("keeps the restored draft when recovery acknowledgement fails", async () => {
  vi.mocked(dismissPromptRecovery).mockRejectedValue(new Error("offline"));
  await expect(restoreWithdrawnDraft("agent", withdrawn)).rejects.toThrow("offline");
  expect(getChatDraft("agent")).toBe("withdrawn input");
});
