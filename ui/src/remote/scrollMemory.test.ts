import { afterEach, describe, expect, it } from "vitest";
import { forgetPlaces, recallPlace, rememberPlace } from "./scrollMemory";

describe("scrollMemory", () => {
  afterEach(forgetPlaces);

  it("keeps the latest place per path and evicts the least recently saved beyond 50", () => {
    for (let index = 0; index < 50; index++) rememberPlace(`/p/${index}`, { y: index });
    rememberPlace("/p/0", { y: 99 });
    rememberPlace("/p/new", { newest: true });
    expect(recallPlace("/p/0")).toEqual({ y: 99 });
    expect(recallPlace("/p/1")).toBeUndefined();
    expect(recallPlace("/p/2")).toEqual({ y: 2 });
    expect(recallPlace("/p/new")).toEqual({ newest: true });
  });
});
