import React, { useState } from "react";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { AgentState } from "../../api/types";
import { GroupPicker, groupNameError, projectGroupLabels } from "./GroupPicker";

afterEach(cleanup);

/** A parent form like the real hosts: it owns the value and records submissions. */
function Host({ groups, initial = "", onSubmit }: { groups: string[]; initial?: string; onSubmit: (value: string) => void }) {
  const [value, setValue] = useState(initial);
  return (
    <form onSubmit={(event) => { event.preventDefault(); if (!groupNameError(value)) onSubmit(value.trim()); }}>
      <GroupPicker value={value} groups={groups} onChange={setValue} />
      <button type="submit">Save</button>
    </form>
  );
}

const combobox = () => screen.getByRole("combobox", { name: "Group" });

describe("GroupPicker", () => {
  it("shows choices on focus, filters case-insensitively, and reuses exact labels", () => {
    const onSubmit = vi.fn();
    render(<Host groups={["Core", "Frontend"]} onSubmit={onSubmit} />);
    fireEvent.focus(combobox());
    expect(screen.getByRole("button", { name: "Core" })).toBeInTheDocument();
    fireEvent.change(combobox(), { target: { value: "front" } });
    expect(screen.queryByRole("button", { name: "Core" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Frontend" }));
    expect(combobox()).toHaveValue("Frontend");
    expect(screen.queryByRole("listbox")).toBeNull();
  });

  it("Enter accepts typed new and partial-match text without choosing Ungrouped or submitting", () => {
    const onSubmit = vi.fn();
    render(<Host groups={["Frontend"]} onSubmit={onSubmit} />);
    fireEvent.focus(combobox());
    fireEvent.change(combobox(), { target: { value: " Front " } });
    fireEvent.keyDown(combobox(), { key: "Enter" });
    expect(combobox()).toHaveValue("Front");
    expect(screen.queryByRole("listbox")).toBeNull();
    expect(screen.queryByRole("button", { name: /Create group/ })).toBeNull();
    expect(onSubmit).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onSubmit).toHaveBeenCalledWith("Front");
  });

  it("Enter takes an arrow-highlighted suggestion instead of the typed text", () => {
    const onSubmit = vi.fn();
    render(<Host groups={["Core", "Frontend"]} initial="Core" onSubmit={onSubmit} />);
    fireEvent.focus(combobox());
    fireEvent.change(combobox(), { target: { value: "front" } });
    fireEvent.keyDown(combobox(), { key: "ArrowDown" });
    fireEvent.keyDown(combobox(), { key: "ArrowDown" });
    fireEvent.keyDown(combobox(), { key: "Enter" });
    expect(combobox()).toHaveValue("Frontend");
    fireEvent.focus(combobox());
    fireEvent.keyDown(combobox(), { key: "ArrowDown" });
    fireEvent.keyDown(combobox(), { key: "Enter" });
    expect(combobox()).toHaveValue("");
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("blur and Escape keep the pending text; Escape only dismisses suggestions", () => {
    const onSubmit = vi.fn();
    render(<Host groups={["Core"]} onSubmit={onSubmit} />);
    fireEvent.focus(combobox());
    fireEvent.change(combobox(), { target: { value: "Pending" } });
    fireEvent.keyDown(combobox(), { key: "Escape" });
    expect(screen.queryByRole("listbox")).toBeNull();
    expect(combobox()).toHaveValue("Pending");
    fireEvent.blur(combobox());
    expect(combobox()).toHaveValue("Pending");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onSubmit).toHaveBeenCalledWith("Pending");
  });

  it("flags the reserved key and an overlong name and keeps them from submitting", () => {
    const onSubmit = vi.fn();
    render(<Host groups={[]} onSubmit={onSubmit} />);
    fireEvent.change(combobox(), { target: { value: "_ungrouped" } });
    expect(screen.getByRole("alert")).toHaveTextContent("reserved");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    fireEvent.change(combobox(), { target: { value: "x".repeat(257) } });
    expect(screen.getByRole("alert")).toHaveTextContent("256");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(onSubmit).not.toHaveBeenCalled();
    fireEvent.change(combobox(), { target: { value: "" } });
    expect(screen.queryByRole("alert")).toBeNull();
  });

  it("suggests only the target project's non-archived labels, including stopped agents", () => {
    const agent = (id: string, project: string, group: string, extra: Partial<AgentState> = {}) =>
      ({ agent_id: id, project, group, archived: false, running: true, ...extra }) as AgentState;
    const agents = {
      a: agent("a", "alpha", "Shared"),
      b: agent("b", "alpha", "Solo", { running: false }),
      c: agent("c", "alpha", "Gone", { archived: true }),
      d: agent("d", "beta", "Shared"),
      e: agent("e", "beta", "Other"),
    };
    expect(projectGroupLabels(agents, "alpha").sort()).toEqual(["Shared", "Solo"]);
    expect(projectGroupLabels(agents, "beta").sort()).toEqual(["Other", "Shared"]);
    expect(projectGroupLabels(agents, "")).toEqual([]);
  });
});
