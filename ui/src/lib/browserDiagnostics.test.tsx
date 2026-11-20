import React from "react";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import { cleanup, render, waitFor } from "@testing-library/react";
import { ErrorBoundary } from "../components/ErrorBoundary";
import { installBrowserDiagnostics, resetBrowserDiagnosticsForTests, routeKind } from "./browserDiagnostics";

// TS-03.R92 / INV §22: desktop browser failures reach the server log as
// allowlisted words; a healthy foreground return stays quiet.
function Boom(): never {
  throw new Error("secret page text");
}

function setVisibility(state: "visible" | "hidden") {
  Object.defineProperty(document, "visibilityState", { configurable: true, value: state });
  document.dispatchEvent(new Event("visibilitychange"));
}

function frame() {
  return new Promise((resolve) => requestAnimationFrame(resolve));
}

function reports(fetchSpy: ReturnType<typeof vi.spyOn>) {
  return fetchSpy.mock.calls
    .filter(([url]) => url === "/api/diagnostics/browser")
    .map(([, init]) => JSON.parse(String((init as RequestInit).body)) as Record<string, string>);
}

beforeAll(() => installBrowserDiagnostics());

afterEach(() => {
  cleanup();
  resetBrowserDiagnosticsForTests();
  setVisibility("visible");
  vi.restoreAllMocks();
});

describe("browser diagnostics", () => {
  it("reports a caught render failure once, without its text", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    vi.spyOn(console, "error").mockImplementation(() => undefined);
    window.history.replaceState(null, "", "/agent/a_1");
    render(<div data-ui="app-shell"><ErrorBoundary><Boom /></ErrorBoundary><ErrorBoundary><Boom /></ErrorBoundary></div>);

    // React's development build also replays the throw as a window error.
    const rendered = () => reports(fetchSpy).filter((body) => body.reason === "render_error");
    await waitFor(() => expect(rendered()).toHaveLength(1));
    const [body] = rendered();
    expect(Object.keys(body).sort()).toEqual(["mount_state", "reason", "route_kind", "visit_id"]);
    expect(body).toMatchObject({ reason: "render_error", route_kind: "agent", mount_state: "mounted" });
    expect(body.visit_id).toMatch(/^[a-z0-9]{8,32}$/);
    expect(JSON.stringify(body)).not.toContain("secret");
    expect(JSON.stringify(body)).not.toContain("a_1");
  });

  it("reports a once-mounted shell missing on foreground return", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    const { unmount } = render(<div data-ui="app-shell" />);
    setVisibility("hidden");
    unmount();
    setVisibility("visible");

    await waitFor(() => expect(reports(fetchSpy)).toHaveLength(1));
    expect(reports(fetchSpy)[0]).toMatchObject({ reason: "shell_missing", mount_state: "missing" });
  });

  it("stays quiet when the shell is still there on return", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(null, { status: 204 }));
    render(<div data-ui="app-shell" />);
    setVisibility("hidden");
    setVisibility("visible");
    await frame();
    await frame();
    expect(reports(fetchSpy)).toHaveLength(0);
  });

  it("maps paths to allowlisted route kinds only", () => {
    expect(routeKind("/")).toBe("dashboard");
    expect(routeKind("/think-tank/t1")).toBe("think_tank");
    expect(routeKind("/pipelines/runs/r1")).toBe("pipelines");
    expect(routeKind("/somewhere/private")).toBe("other");
  });
});
