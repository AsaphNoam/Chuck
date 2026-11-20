import React from "react";
import { cleanup, render, screen } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { afterAll, afterEach, beforeAll, expect, it } from "vitest";
import { ProjectFilePreviewPage } from "./ProjectFilePreviewPage";

let requested: string[] = [];
const server = setupServer(
  http.get("/api/projects/:project/resource-file", ({ request, params }) => {
    const path = new URL(request.url).searchParams.get("path") ?? "";
    requested.push(`${params.project}:${path}`);
    return HttpResponse.json({ agent_id: "", path, size: 20, mod_time: "2026-09-10T10:00:00Z", line_count: 1, content: "<h1>Hi</h1>", truncated: false, language: "html" });
  }),
);
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => { cleanup(); server.resetHandlers(); requested = []; });
afterAll(() => server.close());

function page(hash: string) {
  return render(
    <MemoryRouter initialEntries={[`/project/p_1/preview${hash}`]}>
      <Routes><Route path="project/:project/preview" element={<ProjectFilePreviewPage />} /></Routes>
    </MemoryRouter>,
  );
}

it("reads the fragment path through the project loader into an opaque sandbox", async () => {
  page("#docs%2Freport.html");
  const frame = await screen.findByTitle("HTML preview: docs/report.html");
  expect(frame).toHaveAttribute("sandbox", "allow-scripts");
  expect(frame.getAttribute("sandbox")).not.toContain("allow-same-origin");
  expect(requested).toEqual(["p_1:docs/report.html"]);
});

it("refuses a path outside the project root without reading", () => {
  page("#..%2Fsecret.html");
  expect(screen.getByRole("alert")).toHaveTextContent("outside");
  expect(requested).toEqual([]);
});
