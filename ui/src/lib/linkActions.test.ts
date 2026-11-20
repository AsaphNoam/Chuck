import { afterEach, describe, expect, it, vi } from "vitest";
import { webLinkActions } from "./linkActions";

afterEach(() => { window.history.pushState({}, "", "/"); });

describe("webLinkActions Open in Finder", () => {
  it("offers Open in Finder before Copy path when a reveal callback is given", () => {
    const reveal = vi.fn().mockResolvedValue(undefined);
    const actions = webLinkActions("src/a.go", vi.fn(), undefined, reveal);
    expect(actions.map((a) => a.label)).toEqual(["Open in Finder", "Copy path"]);
    actions[0].select();
    expect(reveal).toHaveBeenCalledWith("src/a.go");
  });

  it("omits it without a callback and under /phone", () => {
    expect(webLinkActions("src/a.go", vi.fn()).map((a) => a.label)).toEqual(["Copy path"]);
    window.history.pushState({}, "", "/phone/agent/a_1");
    expect(webLinkActions("src/a.go", vi.fn(), undefined, vi.fn()).map((a) => a.label)).toEqual(["Copy path"]);
  });

  it("reports a failed reveal without throwing", async () => {
    const pushError = vi.fn();
    const actions = webLinkActions("src/a.go", pushError, undefined, () => Promise.reject(new Error("missing file")));
    actions[0].select();
    await vi.waitFor(() => expect(pushError).toHaveBeenCalledWith("Could not open in Finder", "missing file"));
  });
});
