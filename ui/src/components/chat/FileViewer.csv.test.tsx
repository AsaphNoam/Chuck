import React from "react";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { setupServer } from "msw/node";
import { http, HttpResponse } from "msw";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { FileViewer } from "./FileViewer";

let body: Record<string, unknown> = {};
const server = setupServer(http.get("/api/sessions/:id/file", () => HttpResponse.json(body)));
beforeAll(() => server.listen({ onUnhandledRequest: "bypass" }));
afterEach(() => cleanup());
afterAll(() => server.close());

function csv(content: string, overrides: Record<string, unknown> = {}) {
  body = {
    agent_id: "a_1", path: "data/t.csv", size: content.length, mod_time: "2026-09-10T10:00:00Z",
    line_count: content.split("\n").length, content, truncated: false, language: "csv", ...overrides,
  };
}
Element.prototype.scrollIntoView = vi.fn();
const open = (line?: number) => render(<FileViewer agentId="a_1" link={{ path: "data/t.csv", line }} onClose={vi.fn()} />);
const firstColumn = () => within(screen.getByRole("table")).getAllByRole("row").slice(1).map((row) => within(row).getAllByRole("cell")[0].textContent);

describe("FileViewer CSV", () => {
  it("defaults to Table with literal cells and the first row as data", async () => {
    csv('10,"a,b"\n2,=SUM(A1)\n');
    open();
    expect(await screen.findByRole("table")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "1" })).toBeInTheDocument();
    expect(screen.getByText("a,b")).toBeInTheDocument();
    expect(screen.getByText("=SUM(A1)")).toBeInTheDocument();
    expect(firstColumn()).toEqual(["10", "2"]);
  });

  it("sorts by header, reverses on repeat, and shows aria-sort", async () => {
    csv("10,x\n2,y\n");
    open();
    const header = await screen.findByRole("columnheader", { name: "1" });
    fireEvent.click(within(header).getByRole("button"));
    expect(header).toHaveAttribute("aria-sort", "ascending");
    expect(firstColumn()).toEqual(["2", "10"]);
    fireEvent.click(within(header).getByRole("button"));
    expect(header).toHaveAttribute("aria-sort", "descending");
    expect(firstColumn()).toEqual(["10", "2"]);
    fireEvent.click(within(screen.getByRole("columnheader", { name: "2" })).getByRole("button"));
    expect(header).not.toHaveAttribute("aria-sort");
    expect(screen.getByRole("columnheader", { name: /2/ })).toHaveAttribute("aria-sort", "ascending");
  });

  it("switches to exact Source text", async () => {
    csv("a,b\n1,2\n");
    open();
    await screen.findByRole("table");
    fireEvent.click(screen.getByRole("button", { name: "Source" }));
    await waitFor(() => expect(screen.queryByRole("table")).not.toBeInTheDocument());
    expect(document.querySelector('[data-file-line="2"]')?.textContent).toContain("1,2");
  });

  it("opens a truncated file in Source with an explanation", async () => {
    csv("a,b\n1,2\n", { truncated: true });
    open();
    expect(await screen.findByText(/larger than the viewer reads, so it cannot be shown as a table/)).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("opens a malformed file in Source with an explanation", async () => {
    csv("a,b\n1\n");
    open();
    expect(await screen.findByText(/invalid quoting or rows of different widths/)).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });

  it("keeps a cited line in Source", async () => {
    csv("a,b\n1,2\n");
    open(2);
    expect(await screen.findByText(/Showing the cited source line/)).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
});
