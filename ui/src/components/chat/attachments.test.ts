// FS-03.A64, FS-03.A65: delayed acceptance preserves newer shared attachment drafts.
import { afterEach, describe, expect, it } from "vitest";
import { clearAttachmentDraftCacheForTests, isDefiniteSendRefusal, rememberAttachments, removeAcceptedAttachments, type DraftAttachment } from "./attachments";

const file = (id: string): DraftAttachment => ({ id, name: `${id}.txt`, type: "text/plain", size: 1, blob: new Blob([id]) });

afterEach(() => {
  localStorage.clear();
  clearAttachmentDraftCacheForTests();
});

describe("shared attachment drafts", () => {
  it("does not classify typed delivery uncertainty as a definite refusal", () => {
    expect(isDefiniteSendRefusal({ status: 409, code: "delivery_uncertain" })).toBe(false);
    expect(isDefiniteSendRefusal({ status: 409, code: "held_changed" })).toBe(true);
  });

  it("removes only accepted IDs from the current shared draft after a delayed response", async () => {
    const agentId = "attachment-race-agent";
    const sent = file("sent");
    const newer = file("newer");
    rememberAttachments(agentId, [sent]);

    const accepted = removeAcceptedAttachments(agentId, [sent.id], true);
    rememberAttachments(agentId, [sent, newer]);

    await expect(accepted).rejects.toThrow("IndexedDB is unavailable");
    await expect(import("./attachments").then(({ restoreAttachments }) => restoreAttachments(agentId))).resolves.toEqual([newer]);
  });
});
