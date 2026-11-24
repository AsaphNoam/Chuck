import React from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { AnnotationJumpContext } from "../annotationNavigation";
import { DiffBlock } from "./DiffBlock";

afterEach(cleanup);

// FS-13.R38, TS-08.R175: a sent excerpt may point at unchanged context the real
// viewer folds away; the jump must reveal that saved line, not only lines the
// viewer already draws. Uses the real viewer, whose folding is the contract.
const lines = Array.from({ length: 40 }, (_, index) => `line ${index + 1}`);
const oldText = lines.join("\n");
const newText = lines.map((line, index) => (index === 34 ? "changed 35" : line)).join("\n");

function landedRow(line: { side: "old" | "new"; number: number }) {
  const { container } = render(
    <AnnotationJumpContext.Provider value={{ seq: 7, line, land: () => {} }}>
      <DiffBlock event={{ seq: 7, path: "main.go", old_text: oldText, new_text: newText }} onAnnotate={() => {}} />
    </AnnotationJumpContext.Provider>,
  );
  return container.querySelector("[data-annotation-line]")?.closest("tr")?.textContent ?? null;
}

it("reveals a saved line inside collapsed unchanged context", () => {
  expect(landedRow({ side: "new", number: 5 })).toContain("line 5");
  cleanup();
  expect(landedRow({ side: "old", number: 8 })).toContain("line 8");
});

it("keeps landing on a changed line on either side", () => {
  expect(landedRow({ side: "new", number: 35 })).toContain("changed 35");
  cleanup();
  expect(landedRow({ side: "old", number: 35 })).toContain("line 35");
});
