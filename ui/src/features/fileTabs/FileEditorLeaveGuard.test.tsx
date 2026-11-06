import React from "react";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { afterEach, expect, it } from "vitest";
import { FileEditorLeaveGuard } from "./FileEditorLeaveGuard";
import { markDraftUnsafe } from "./drafts";

afterEach(() => { cleanup(); markDraftUnsafe("test", false); });
it("guards imperative navigation and Back when local recovery is unavailable", async () => {
  const router = createMemoryRouter([
    { path: "/before", element: <p>Before</p> },
    { path: "/edit", element: <><FileEditorLeaveGuard /><p>Editor</p></> },
    { path: "/next", element: <p>Next</p> },
  ], { initialEntries: ["/before", "/edit"], initialIndex: 1 });
  render(<RouterProvider router={router} />);
  act(() => markDraftUnsafe("test", true));
  await act(async () => { void router.navigate("/next"); });
  await screen.findByRole("dialog", { name: "Leave unsaved edits?" });
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(router.state.location.pathname).toBe("/edit");
  await act(async () => { void router.navigate(-1); });
  await screen.findByRole("dialog", { name: "Leave unsaved edits?" });
  fireEvent.click(screen.getByRole("button", { name: "Leave anyway" }));
  await screen.findByText("Before");
  expect(router.state.location.pathname).toBe("/before");
});
