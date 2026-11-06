import React from "react";
import { cleanup, render } from "@testing-library/react";
import { afterEach, expect, it } from "vitest";
import { foldTranscript, useTranscriptStore } from "../../store/transcriptStore";
import { AssistantText } from "./renderers/AssistantText";
import { SearchEventContext, SearchPassageContext, sourceTextRanges, utf8Offset } from "./searchPassage";

afterEach(cleanup);

// Independent source positions distinguish identical occurrences and UTF-8
// bytes from UTF-16 units after the shared live/replay merge boundary.
it("marks the constituent later delta at its Unicode source position", () => {
  const raw = [
    { seq: 10, kind: "assistant_text", delta: "same 🐈\n\n" },
    { seq: 11, kind: "assistant_text", delta: "same 🐈" },
  ];
  const event = foldTranscript(raw)[0];
  const reveal = {key:"later",ranges:[{seq:11,part_index:0,start_byte:5,end_byte:9}]};
  expect(sourceTextRanges(event,String(event.text),0,reveal)).toEqual([{start:14,end:16}]);
  expect(utf8Offset("🐈",1)).toBeNull();
  useTranscriptStore.getState().discardAgent("source-proof");
  for (const delta of raw) useTranscriptStore.getState().appendMessage("source-proof",delta);
  expect(useTranscriptStore.getState().byAgent["source-proof"][0].search_sources).toEqual(event.search_sources);
  const {container} = render(<SearchPassageContext.Provider value={reveal}><SearchEventContext.Provider value={event}><AssistantText event={event}/></SearchEventContext.Provider></SearchPassageContext.Provider>);
  const marks=container.querySelectorAll("mark");
  expect(marks).toHaveLength(1);
  expect(marks[0].textContent).toBe("🐈");
  expect(marks[0].closest("p")).toBe(container.querySelectorAll("p")[1]);
});

it("reveals transformed Markdown syntax as an honest source excerpt", () => {
  const event={seq:20,kind:"assistant_text",text:"[label](https://example.com)"};
  const reveal={key:"syntax",ranges:[{seq:20,part_index:0,start_byte:8,end_byte:27}]};
  const {container} = render(<SearchPassageContext.Provider value={reveal}><SearchEventContext.Provider value={event}><AssistantText event={event}/></SearchEventContext.Provider></SearchPassageContext.Provider>);
  expect(container.querySelector("mark")).toBeNull();
  expect(container.querySelector("[data-search-excerpt]")?.textContent).toBe("Matched source: https://example.com");
  expect(container.querySelector("a")?.getAttribute("rel")).toContain("noopener");
});
