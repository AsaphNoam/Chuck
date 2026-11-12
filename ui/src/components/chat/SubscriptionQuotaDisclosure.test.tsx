import React from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { SubscriptionQuota } from "../../api/types";
import { SubscriptionQuotaDisclosure } from "./SubscriptionQuotaDisclosure";

const complete: SubscriptionQuota = {
  provider: "claude",
  availability: "complete",
  checked_at: "2026-11-12T10:00:00Z",
  windows: {
    five_hour: { used_percent: 42.5, reset_at: "2026-11-12T16:00:00Z", observed_at: "2026-11-12T10:00:00Z" },
    weekly: { used_percent: 7, reset_at: "2026-11-15T09:30:00Z", observed_at: "2026-11-12T10:00:00Z" },
  },
};

describe("SubscriptionQuotaDisclosure (FS-09.A49/A52)", () => {
  afterEach(cleanup);
  it("renders nothing for a backend without a subscription view", () => {
    const { container } = render(<SubscriptionQuotaDisclosure />);
    expect(container).toBeEmptyDOMElement();
  });

  it("opens from its native summary without any request and shows both windows with labelled meters", () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch");
    render(<SubscriptionQuotaDisclosure value={complete} />);
    const summary = screen.getByText("Subscription quota");
    fireEvent.click(summary);
    expect(summary.closest("details")).toHaveAttribute("open");
    expect(screen.getByText("Claude subscription")).toBeVisible();
    expect(screen.getByRole("meter", { name: "5-hour 43% used" })).toHaveAttribute("value", "42.5");
    expect(screen.getByRole("meter", { name: "Weekly 7% used" })).toBeInTheDocument();
    expect(screen.getAllByText(/^Resets /)).toHaveLength(2);
    expect(screen.getByText(/^Last checked /)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /refresh/i })).toBeNull();
    expect(fetchSpy).not.toHaveBeenCalled();
    fetchSpy.mockRestore();
  });

  it("keeps an unknown window as text rather than an empty meter", () => {
    render(<SubscriptionQuotaDisclosure value={{
      ...complete, availability: "partial",
      windows: { five_hour: complete.windows.five_hour, weekly: { observed_at: "2026-11-12T10:00:00Z" } },
    }} />);
    expect(screen.getAllByRole("meter", { hidden: true })).toHaveLength(1);
    expect(screen.getByText("Unknown")).toBeInTheDocument();
    expect(screen.getByText("Reset time unknown")).toBeInTheDocument();
  });

  it("marks retained values stale after a failed read", () => {
    render(<SubscriptionQuotaDisclosure value={{ ...complete, reason_code: "read_failed" }} />);
    expect(screen.getByText("The last check failed; showing earlier values.")).toBeInTheDocument();
    expect(screen.getAllByRole("meter", { hidden: true })).toHaveLength(2);
  });

  it.each([
    [{ reason_code: "no_subscription" as const, checked_at: "2026-11-12T10:00:00Z" }, "This sign-in has no subscription quota."],
    [{ reason_code: "unsupported_adapter" as const, checked_at: undefined }, "This adapter version cannot read subscription quota."],
    [{ reason_code: undefined, checked_at: undefined }, "Not checked yet. It updates after this backend's next turn."],
    // A failed first read has no earlier values to show.
    [{ reason_code: "read_failed" as const, checked_at: undefined }, "The check failed. It updates after this backend's next turn."],
    [{ reason_code: "read_timeout" as const, checked_at: undefined }, "The check timed out. It updates after this backend's next turn."],
  ])("explains an unavailable snapshot (%o)", (extra, text) => {
    render(<SubscriptionQuotaDisclosure value={{
      provider: "codex", availability: "unavailable", windows: { five_hour: {}, weekly: {} }, ...extra,
    }} />);
    expect(screen.getByText(text)).toBeInTheDocument();
    expect(screen.queryAllByRole("meter", { hidden: true })).toHaveLength(0);
  });
});
