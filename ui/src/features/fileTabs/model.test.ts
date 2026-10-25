import { describe, expect, it } from "vitest";
import {
  closeTab, decodeTabs, encodeTabs, moveTab, openTab, pathKey, resolveTab, SavedTabsError,
  type Owners, type TabView,
} from "./model";

const KEY = "conversation:a1";
let next = 0;
const id = () => `t${++next}`;

function openAll(paths: string[], owners: Owners = {}, key = KEY) {
  for (const path of paths) {
    const result = openTab(owners, key, { path }, id());
    if ("refused" in result) throw new Error(result.refused);
    owners = result.owners;
  }
  return owners;
}
const paths = (owners: Owners, key = KEY) => owners[key]?.tabs.map((tab) => tab.path) ?? [];
const activePath = (owners: Owners, key = KEY) => owners[key]?.tabs.find((tab) => tab.id === owners[key].active)?.path;

describe("file tab model (FS-03.R101–R105)", () => {
  it("keeps the trail and selects an existing tab by normalized spelling", () => {
    let owners = openAll(["a.md", "b.md", "c.md"]);
    expect(paths(owners)).toEqual(["a.md", "b.md", "c.md"]);
    const again = openTab(owners, KEY, { path: "./a.md", line: 7 }, id());
    if ("refused" in again) throw new Error("refused");
    owners = again.owners;
    expect(again.added).toBe(false);
    expect(paths(owners)).toEqual(["a.md", "b.md", "c.md"]);
    expect(activePath(owners)).toBe("a.md");
    expect(owners[KEY].tabs[0].line).toBe(7);
    expect(pathKey("x//./y/")).toBe("x/y");
    expect(pathKey("/abs/../y")).toBe("/abs/../y");
  });

  it("keeps equal relative paths from different room sources distinct", () => {
    let owners: Owners = {};
    for (const source of ["s1", "s2", "s1"]) {
      const result = openTab(owners, "room:r", { path: "README.md", source }, id());
      if ("refused" in result) throw new Error("refused");
      owners = result.owners;
    }
    expect(owners["room:r"].tabs.map((tab) => tab.source)).toEqual(["s1", "s2"]);
  });

  it("closes to the right neighbour, then the left, and ends the owner on the last", () => {
    let owners = openAll(["a", "b", "c"]);
    const [a, b, c] = owners[KEY].tabs;
    owners = { ...owners, [KEY]: { ...owners[KEY], active: b.id } };
    owners = closeTab(owners, KEY, b.id);
    expect(owners[KEY].active).toBe(c.id);
    owners = closeTab(owners, KEY, a.id);
    expect(owners[KEY].active).toBe(c.id);
    owners = openAll(["d"], owners);
    owners = closeTab(owners, KEY, owners[KEY].active!);
    expect(owners[KEY].active).toBe(c.id);
    owners = closeTab(owners, KEY, c.id);
    expect(owners[KEY]).toBeUndefined();
  });

  it("reorders without changing the selection", () => {
    let owners = openAll(["a", "b", "c"]);
    const active = owners[KEY].active;
    owners = moveTab(owners, KEY, owners[KEY].tabs[2].id, 0);
    expect(paths(owners)).toEqual(["c", "a", "b"]);
    expect(owners[KEY].active).toBe(active);
    expect(moveTab(owners, KEY, owners[KEY].tabs[0].id, -3)).toBe(owners);
  });

  it("refuses the 51st tab and the 501st saved tab without evicting", () => {
    const fifty = Array.from({ length: 50 }, (_, i) => `f${i}`);
    let owners = openAll(fifty);
    expect(openTab(owners, KEY, { path: "one-more" }, id())).toEqual({ refused: "owner" });
    expect("refused" in openTab(owners, KEY, { path: "f3" }, id())).toBe(false);
    for (let owner = 1; owner < 10; owner += 1) owners = openAll(fifty, owners, `conversation:o${owner}`);
    expect(openTab(owners, "conversation:new", { path: "x" }, id())).toEqual({ refused: "total" });
    expect(Object.values(owners).reduce((n, o) => n + o.tabs.length, 0)).toBe(500);
  });

  it("merges a normalized duplicate into the existing tab", () => {
    let owners = openAll(["docs/a.md", "./docs/../docs/a.md"]);
    const [first, second] = owners[KEY].tabs;
    const merged = resolveTab(owners, KEY, second.id, "docs/a.md");
    expect(merged.kept).toBe(first.id);
    owners = merged.owners;
    expect(owners[KEY].tabs.map((tab) => tab.id)).toEqual([first.id]);
    expect(owners[KEY].active).toBe(first.id);
    const renamed = resolveTab(owners, KEY, first.id, "docs/A.md");
    expect(renamed.owners[KEY].tabs[0]).toMatchObject({ id: first.id, path: "docs/A.md" });
    expect(resolveTab(owners, KEY, "gone", "x").kept).toBeNull();
  });
});

describe("saved tab codec (TS-08.R141/R144)", () => {
  const view: TabView = { rendered: false, anchors: { rendered: null, formatted: null, source: { key: 40, offset: -3 } } };

  it("round-trips descriptors and views without file contents", () => {
    const owners = openAll(["a.md", "b.md"]);
    const views = new Map([[owners[KEY].tabs[0].id, view]]);
    const text = encodeTabs(owners, views);
    expect(Object.keys(JSON.parse(text).owners[KEY].tabs[0]).sort()).toEqual(["anchors", "id", "path", "rendered"]);
    const decoded = decodeTabs(text);
    expect(decoded.owners).toEqual(owners);
    expect(decoded.views.get(owners[KEY].tabs[0].id)).toEqual(view);
  });

  it.each([
    ["not JSON", "{"],
    ["unknown version", JSON.stringify({ v: 2, owners: {} })],
    ["bad owner key", JSON.stringify({ v: 1, owners: { "agent:x": { tabs: [{ id: "a", path: "p", rendered: true, anchors: {} }], active: null } } })],
    ["duplicate id", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: [{ id: "a", path: "p", rendered: true, anchors: {} }, { id: "a", path: "q", rendered: true, anchors: {} }], active: "a" } } })],
    ["room tab without source", JSON.stringify({ v: 1, owners: { "room:r": { tabs: [{ id: "a", path: "p", rendered: true, anchors: {} }], active: "a" } } })],
    ["conversation tab with source", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: [{ id: "a", path: "p", source: "s", rendered: true, anchors: {} }], active: "a" } } })],
    ["zero line", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: [{ id: "a", path: "p", line: 0, rendered: true, anchors: {} }], active: "a" } } })],
    ["unbounded anchor", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: [{ id: "a", path: "p", rendered: true, anchors: { source: { key: 1, offset: 1e12 } } }], active: "a" } } })],
    ["unknown active", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: [{ id: "a", path: "p", rendered: true, anchors: {} }], active: "b" } } })],
    ["too many tabs", JSON.stringify({ v: 1, owners: { [KEY]: { tabs: Array.from({ length: 51 }, (_, i) => ({ id: `a${i}`, path: "p", rendered: true, anchors: {} })), active: null } } })],
    ["oversized", "x".repeat(1_048_577)],
  ])("rejects %s with readable feedback", (_name, raw) => {
    expect(() => decodeTabs(raw)).toThrow(SavedTabsError);
  });
});
