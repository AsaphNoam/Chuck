import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { RoleForm } from "./RoleForm";

// FS-04.A36: role editors preserve explicit false and legacy checked defaults.
describe("RoleForm Chuck awareness", () => {
  afterEach(cleanup);
  it("defaults legacy roles on and preserves an explicit opt out", async () => {
    const submit = vi.fn();
    const view = render(<RoleForm onSubmit={submit} onCancel={() => {}} />);
    fireEvent.change(screen.getByPlaceholderText("e.g. security-reviewer"), { target: { value: "new-role" } });
    fireEvent.change(screen.getByPlaceholderText("e.g. Security Reviewer"), { target: { value: "New role" } });
    const checkbox = screen.getByRole("checkbox", { name: "Chuck aware agent" });
    expect(checkbox).toBeChecked();
    fireEvent.click(checkbox);
    fireEvent.submit(view.container.querySelector("form")!);
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ chuck_aware: false })));
    expect(checkbox).toHaveAttribute("aria-describedby");
    expect(screen.getByText("Adds Chuck operating guidance to the agent's system instructions.")).toBeInTheDocument();
  });

  it("keeps legacy edits enabled when chuck_aware is omitted", async () => {
    const submit = vi.fn();
    const view = render(<RoleForm initial={{ role: "reviewer", title: "Reviewer", system_prompt: "review", skip_permissions: null }} onSubmit={submit} onCancel={() => {}} />);
    expect(screen.getByRole("checkbox", { name: "Chuck aware agent" })).toBeChecked();
    fireEvent.submit(view.container.querySelector("form")!);
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ role: "reviewer", chuck_aware: true })));
  });
});

// FS-04.A39: the inheriting permission choice names the Settings control it follows.
describe("RoleForm permissions", () => {
  afterEach(cleanup);
  it("labels the inherited choice as default permissions and submits null", async () => {
    const submit = vi.fn();
    const view = render(<RoleForm initial={{ role: "reviewer", title: "Reviewer", system_prompt: "", skip_permissions: null }} onSubmit={submit} onCancel={() => {}} />);
    expect(screen.getByRole("option", { name: "Use default permissions" })).toBeInTheDocument();
    fireEvent.submit(view.container.querySelector("form")!);
    await waitFor(() => expect(submit).toHaveBeenCalledWith(expect.objectContaining({ skip_permissions: null })));
  });
});
