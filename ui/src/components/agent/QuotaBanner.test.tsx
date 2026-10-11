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

  it("ignores malformed quota objects", () => {
    const { container } = render(<QuotaBanner value={{ ...quota, episode_id: "" }} />);
    expect(container.firstChild).toBeNull();
  });
});
