import React from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createMemoryRouter, RouterProvider } from "react-router-dom";
import { expect, it } from "vitest";
import { CONVERSATION_FILE_KEYS, useFileTabRoute } from "./useFileTabRoute";
import { FileEditorLeaveGuard } from "./FileEditorLeaveGuard";
import { FileEditor } from "./FileEditor";
import { draftKey, draftPersistenceUnsafe, putDraft } from "./drafts";

// FS-03.R112: a capacity-refused draft remains mounted until leave is acknowledged.
it("preserves a refused draft and its selected file when opening another file is cancelled", async () => {
  const owner = "conversation:leave-regression";
  for (let i = 0; i < 50; i++) {
    const identity = { owner, path: `keep-${i}.txt` };
    putDraft({ ...identity, version: 1, key: draftKey(identity), lineage: String(i), generation: 1, text: "keep", baseRevision: "r1", displayMode: "source" });
  }
  function View() {
    const tabs = useFileTabRoute(owner, CONVERSATION_FILE_KEYS);
    return <><FileEditorLeaveGuard /><button onClick={() => tabs.open({ path: "other.txt" })}>Open other</button><p data-testid="active">{tabs.active?.path}</p>{tabs.active?.path === "refused.txt" && <FileEditor identity={{ owner, path: "refused.txt" }} initial={{ agent_id: "review", path: "refused.txt", content: "base", revision: "r1", language: "text", size: 4, line_count: 1, mod_time: "", truncated: false }} load={async () => { throw Error(); }} save={async () => { throw Error(); }} onSaved={() => {}} onCancel={() => {}} />}</>;
  }
  const router = createMemoryRouter([{ path: "/", element: <View /> }], { initialEntries: ["/?file=refused.txt"] });
  render(<RouterProvider router={router} />);
  const field = await screen.findByLabelText("File source draft") as HTMLTextAreaElement;
  await waitFor(() => expect(field.disabled).toBe(false));
  fireEvent.change(field, { target: { value: "only copy of refused text" } });
  expect(draftPersistenceUnsafe()).toBe(true);
  field.setSelectionRange(3, 7);
  fireEvent.click(screen.getByRole("button", { name: "Open other" }));
  await screen.findByRole("dialog", { name: "Leave unsaved edits?" });
  expect(screen.getByTestId("active").textContent).toBe("refused.txt");
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  await waitFor(() => expect(screen.queryByRole("dialog")).toBeNull());
  expect(router.state.location.search).toBe("?file=refused.txt");
  expect(screen.getByTestId("active").textContent).toBe("refused.txt");
  expect(screen.getByLabelText("File source draft")).toBe(field);
  expect(field.value).toBe("only copy of refused text");
  expect([field.selectionStart, field.selectionEnd]).toEqual([3, 7]);
  fireEvent.click(screen.getByRole("button", { name: "Open other" }));
  fireEvent.click(await screen.findByRole("button", { name: "Leave anyway" }));
  await waitFor(() => expect(screen.getByTestId("active").textContent).toBe("other.txt"));
  expect(router.state.location.search).toBe("?file=other.txt");
  expect(screen.queryByRole("dialog")).toBeNull();
});
