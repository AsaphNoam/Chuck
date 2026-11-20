import { afterEach, expect, it, vi } from "vitest";
import { openProjectFileTab } from "./projectFileTab";

afterEach(() => vi.restoreAllMocks());

it("opens the preview page with no opener and the path in the fragment", () => {
  const open = vi.spyOn(window, "open").mockReturnValue(null);
  openProjectFileTab("p 1", "docs/a b.html");
  expect(open).toHaveBeenCalledWith("/project/p%201/preview#docs%2Fa%20b.html", "_blank", "noopener,noreferrer");
});
