import { describe, expect, it } from "vitest";
import { caret, ComposerHistory } from "./composerHistory";
import { editNumberedList } from "./composerLists";

const at = (text: string, pos = text.length) => ({ text, selection: caret(pos) });

describe("composer history", () => {
  it("restores adjacent held Backspaces across a line and separates a later deletion", () => {
    const history = new ComposerHistory();
    const hold = {};
    history.record(at("ab\nc"), at("ab\n"), { direction: "backward", at: 0, hold });
    history.record(at("ab\n"), at("ab"), { direction: "backward", at: 600, hold });
    history.record(at("ab"), at("a"), { direction: "backward", at: 1200 });
    expect(history.replay("a", false)).toEqual(at("ab"));
    expect(history.replay("ab", false)).toEqual(at("ab\nc"));
    expect(history.replay("ab\nc", true)).toEqual(at("ab"));
  });

  it("combines forward Delete at a fixed caret and restores a backward selection separately", () => {
    const history = new ComposerHistory();
    history.record(at("aabc", 0), at("abc", 0), { direction: "forward", at: 0 });
    history.record(at("abc", 0), at("bc", 0), { direction: "forward", at: 100 });
    const selected = { text: "bc", selection: { start: 0, end: 2, direction: "backward" } } as const;
    history.record(selected, at(""), { direction: "forward", at: 150 });
    expect(history.replay("", false)).toEqual(selected);
    expect(history.replay("bc", false)).toEqual(at("aabc", 0));
    history.record(at("aabc", 0), at("Xaabc", 1));
    expect(history.replay("Xaabc", true)).toBeUndefined();
  });

  it("evicts old steps and starts fresh after an edit above the text budget", () => {
    const history = new ComposerHistory();
    let text = "x".repeat(2 * 1024 * 1024 + 1);
    history.record(at(""), at(text));
    expect(history.replay(text, false)).toBeUndefined();
    text = "x";
    for (let i = 0; i < 101; i++) {
      history.record(at(text), at(text + "a"));
      text += "a";
    }
    for (let i = 0; i < 100; i++) text = history.replay(text, false)!.text;
    expect(text.endsWith("xa")).toBe(true);
    expect(history.replay(text, false)).toBeUndefined();
  });
});

describe("numbered composer lists", () => {
  it("continues, nests by the parent's content column, numbers children and promotes", () => {
    expect(editNumberedList(at("10. First"), "continue")).toEqual({
      intermediate: at("10. First\n"), next: at("10. First\n11. "),
    });
    expect(editNumberedList(at("10. First\n11. "), "indent")?.next).toEqual(at("10. First\n    1. "));
    expect(editNumberedList(at("10. First\n    1. Child\n11. "), "indent")?.next)
      .toEqual(at("10. First\n    1. Child\n    2. "));
    expect(editNumberedList(at("10. First\n    1. Child\n    2. "), "promote")?.next)
      .toEqual(at("10. First\n    1. Child\n11. "));
  });

  it("changes only the current prefix and preserves a body-relative caret", () => {
    const before = "1. First\n2. Existing\n3. Last";
    const after = "1. First\n   1. Existing\n3. Last";
    expect(editNumberedList(at(before, before.indexOf("isting")), "indent")?.next)
      .toEqual(at(after, after.indexOf("isting")));
    expect(editNumberedList(at("1. First\n2. "), "continue")?.next).toEqual(at("1. First\n"));
  });

  it.each(["prose", "````text\n```\n1. example", "1. ```\n   1. example", "    1. indented code", "prose\n2. continuation"])(
    "leaves ordinary newlines alone in %j", (text) => {
      expect(editNumberedList(at(text), "continue")).toBeUndefined();
    },
  );
});
