import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { QuotaBanner } from "./QuotaBanner";

afterEach(cleanup);

const quota = {
  episode_id: "qe_1",
  revision: 3,
  observed_at: "2026-10-11T12:00:00Z",
  reset_at: null,
  due_at: null,
  continuation_state: "unknown_reset" as const,
};

describe("QuotaBanner", () => {
  it("shows a clear unknown reset and no invented cancel action", () => {
    render(<QuotaBanner value={quota} />);
    expect(screen.getByText("Quota reached")).toBeInTheDocument();
    expect(screen.getByText("Reset time unknown")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel continuation" })).not.toBeInTheDocument();
  });

  it("offers a revision-scoped cancellation for a pending schedule and shows errors", () => {
    const onCancel = vi.fn();
    render(<QuotaBanner value={{ ...quota, reset_at: "2026-10-11T13:00:00Z", due_at: "2026-10-11T13:00:00Z", continuation_state: "scheduled" }} onCancel={onCancel} cancelError="Schedule changed. Refresh and try again." />);
    expect(screen.getByText(/Will continue at/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel continuation" }));
    expect(onCancel).toHaveBeenCalledOnce();
    expect(screen.getByRole("alert")).toHaveTextContent("Schedule changed");
  });

  it("shows a pending check apart from any reset, with its allowance and Cancel", () => {
    const onCancel = vi.fn();
    render(<QuotaBanner value={{ ...quota, due_at: "2026-10-11T13:00:00Z", continuation_state: "scheduled", next_action: "check", checks_remaining: 5 }} onCancel={onCancel} />);
    expect(screen.getByText("Reset time unknown")).toBeInTheDocument();
    expect(screen.getByText(/Will check again at/)).toBeInTheDocument();
    expect(screen.queryByText(/Will continue at/)).not.toBeInTheDocument();
    expect(screen.getByText("5 checks left")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel continuation" }));
    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("keeps an in-flight check cancellable and names exhausted checks", () => {
    const { rerender } = render(<QuotaBanner value={{ ...quota, continuation_state: "starting", next_action: "check", checks_remaining: 4 }} onCancel={() => undefined} />);
    expect(screen.getByText("Checking quota…")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Cancel continuation" })).toBeInTheDocument();
    rerender(<QuotaBanner value={{ ...quota, continuation_state: "needs_attention", reason_code: "check_exhausted", next_action: "check", checks_remaining: 0 }} onCancel={() => undefined} />);
    expect(screen.getByText("No quota checks left. Continue manually.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel continuation" })).not.toBeInTheDocument();
  });

  it("treats an old record without an action as a continuation", () => {
    render(<QuotaBanner value={{ ...quota, continuation_state: "starting" }} onCancel={() => undefined} />);
    expect(screen.getByText("Continuing…")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Cancel continuation" })).not.toBeInTheDocument();
  });

  it("ignores malformed quota objects", () => {
    const { container } = render(<QuotaBanner value={{ ...quota, episode_id: "" }} />);
    expect(container.firstChild).toBeNull();
  });
});
