import { describe, expect, it } from "vitest";
import { decodeQuotaState, formatQuotaTime } from "./quota";

const validQuota = {
  episode_id: "qe_1",
  revision: 2,
  observed_at: "2026-10-11T12:00:00Z",
  reset_at: null,
  due_at: null,
  continuation_state: "unknown_reset",
};

describe("quota projection", () => {
  it("accepts a complete quota episode with an unknown reset", () => {
    // Records without an action predate checks and mean continue.
    expect(decodeQuotaState(validQuota)).toEqual({ ...validQuota, next_action: "continue" });
  });

  it("decodes a pending check and bounds its allowance", () => {
    const check = { ...validQuota, continuation_state: "scheduled", due_at: "2026-10-11T13:00:00Z", next_action: "check", checks_remaining: 6 };
    expect(decodeQuotaState(check)).toEqual(check);
    expect(decodeQuotaState({ ...check, checks_remaining: 7 })).toBeUndefined();
    expect(decodeQuotaState({ ...check, next_action: "probe" })).toBeUndefined();
  });

  it("rejects malformed projection data", () => {
    expect(decodeQuotaState({ ...validQuota, revision: "2" })).toBeUndefined();
    expect(decodeQuotaState({ ...validQuota, reset_at: "not a timestamp" })).toBeUndefined();
    expect(decodeQuotaState({ ...validQuota, continuation_state: "retrying" })).toBeUndefined();
  });

  it("formats valid instants and treats invalid or absent times as unknown", () => {
    expect(formatQuotaTime("2026-10-11T12:00:00Z")).toBeTruthy();
    expect(formatQuotaTime("invalid")).toBeUndefined();
    expect(formatQuotaTime(null)).toBeUndefined();
  });
});
