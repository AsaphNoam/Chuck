import { describe, expect, it } from "vitest";
import { parseCsv, sortedRowOrder } from "./csv";

const rowsOf = (text: string) => {
  const parsed = parseCsv(text);
  if (!parsed.ok) throw new Error(parsed.reason);
  return parsed.rows;
};

describe("parseCsv", () => {
  it("keeps quoted commas, newlines, escaped quotes and empty fields", () => {
    expect(rowsOf('a,"b,c","line1\nline2","say ""hi"""\n,,x,\n')).toEqual([
      ["a", "b,c", "line1\nline2", 'say "hi"'],
      ["", "", "x", ""],
    ]);
  });

  it("accepts a BOM, CRLF and a missing or present trailing newline", () => {
    expect(rowsOf("﻿a,b\r\n1,2\r\n")).toEqual([["a", "b"], ["1", "2"]]);
    expect(rowsOf("a,b\n1,2")).toEqual([["a", "b"], ["1", "2"]]);
  });

  it("keeps formula-like text literal", () => {
    expect(rowsOf("=SUM(A1),<b>x</b>\n")).toEqual([["=SUM(A1)", "<b>x</b>"]]);
  });

  it("rejects differing widths and bad quoting", () => {
    expect(parseCsv("a,b\n1\n")).toEqual({ ok: false, reason: "malformed" });
    expect(parseCsv('a,"b\n')).toEqual({ ok: false, reason: "malformed" });
    expect(parseCsv('a,b"c"\n')).toEqual({ ok: false, reason: "malformed" });
    expect(parseCsv('"a"x,b\n')).toEqual({ ok: false, reason: "malformed" });
  });

  it("rejects over-limit tables", () => {
    expect(parseCsv("a\n".repeat(1001))).toEqual({ ok: false, reason: "too-large" });
    expect(parseCsv(Array(101).fill("a").join(",") + "\n")).toEqual({ ok: false, reason: "too-large" });
    expect(parseCsv((Array(100).fill("a").join(",") + "\n").repeat(201))).toEqual({ ok: false, reason: "too-large" });
  });
});

describe("sortedRowOrder", () => {
  const rows = [["10", "b"], ["2", "a"], ["", ""], ["2", "a"]];
  it("sorts numeric columns numerically with blanks last in both directions", () => {
    expect(sortedRowOrder(rows, { column: 0, direction: "ascending" })).toEqual([1, 3, 0, 2]);
    expect(sortedRowOrder(rows, { column: 0, direction: "descending" })).toEqual([0, 1, 3, 2]);
  });
  it("sorts other columns as text and keeps ties in file order", () => {
    const text = [["b"], ["a"], ["a"], [""]];
    expect(sortedRowOrder(text, { column: 0, direction: "ascending" })).toEqual([1, 2, 0, 3]);
    expect(sortedRowOrder(text, { column: 0, direction: "descending" })).toEqual([0, 1, 2, 3]);
  });
  it("is file order without a sort", () => {
    expect(sortedRowOrder(rows, null)).toEqual([0, 1, 2, 3]);
  });
});
