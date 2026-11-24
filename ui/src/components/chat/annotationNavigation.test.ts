import { afterEach, expect, it } from "vitest";
import { annotationClippedMarker } from "../../lib/annotations";
import { findPassage } from "./annotationNavigation";

afterEach(() => { document.body.innerHTML = ""; });

function row(html: string) {
  const host = document.createElement("div");
  host.innerHTML = html;
  document.body.append(host);
  return host;
}

// FS-13.R38, TS-08.R175: the excerpt is the browser's selection text, which puts
// line breaks between rendered paragraphs that the DOM's text nodes do not hold.
it("locates a quote selected across two rendered paragraphs", () => {
  const source = row("<p>Intro words. The plan <em>starts</em> here.</p><ul><li>Then it ends.</li></ul>");
  const range = findPassage(source, "plan starts here.\n\nThen it ends");
  expect(range?.toString()).toBe("plan starts here.Then it ends");
  expect(range?.startContainer.textContent).toBe("Intro words. The plan ");
  expect(range?.endContainer.parentElement?.tagName).toBe("LI");
});

it("locates a single-paragraph quote and a clipped excerpt's retained text", () => {
  const source = row("<p>Alpha beta gamma.</p><p>Delta epsilon.</p>");
  expect(findPassage(source, "beta gamma")?.toString()).toBe("beta gamma");
  expect(findPassage(source, `gamma.\nDelta${annotationClippedMarker}`)?.toString()).toBe("gamma.Delta");
});

it("falls back to the row when the quote is ambiguous or absent", () => {
  const source = row("<p>repeat once</p><p>repeat twice</p>");
  expect(findPassage(source, "repeat")).toBeUndefined();
  expect(findPassage(source, "missing words")).toBeUndefined();
  expect(findPassage(source, "  \n ")).toBeUndefined();
});
