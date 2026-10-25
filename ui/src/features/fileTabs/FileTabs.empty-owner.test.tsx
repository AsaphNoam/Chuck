import React from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { afterEach, expect, it } from "vitest";
import { TranscriptView } from "../../components/chat/TranscriptView";
import { STORAGE_KEY } from "./model";
import { loadFileTabs, useFileTabsStore } from "./store";
import { CONVERSATION_FILE_KEYS, useFileTabRoute } from "./useFileTabRoute";

afterEach(cleanup);

function EmptyConversation() {
  const tabs = useFileTabRoute("conversation:new", CONVERSATION_FILE_KEYS);
  return <TranscriptView agentId="new" events={[{ kind: "assistant_text", seq: 1, text: "[Read file](new.md)" }]} fileTabs={tabs} onOpenFile={link => { if (link) tabs.open(link); }} />;
}

// FS-03.A82: refusing another tab preserves existing tabs and explains recovery.
it("explains total-cap refusal when this conversation has no existing tabs", () => {
  const owners = Object.fromEntries(Array.from({ length: 10 }, (_, owner) => [
    `conversation:a${owner}`,
    { active: `${owner}-0`, tabs: Array.from({ length: 50 }, (_, tab) => ({ id: `${owner}-${tab}`, path: `file-${tab}.md`, rendered: true, anchors: {} })) },
  ]));
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ v: 1, owners }));
  loadFileTabs();
  render(<QueryClientProvider client={new QueryClient()}><MemoryRouter><EmptyConversation /></MemoryRouter></QueryClientProvider>);
  fireEvent.click(screen.getByRole("button", { name: "Read file" }));
  expect(useFileTabsStore.getState().notice).toMatch(/500 file tabs/);
  expect(Object.values(useFileTabsStore.getState().owners).flatMap(owner => owner.tabs)).toHaveLength(500);
  expect(screen.getByRole("status")).toHaveTextContent(/500 file tabs.*Close a tab/);
});

// FS-03.A82: malformed saved data gives visible recovery without blocking chat.
it("shows damaged-storage recovery feedback on an empty conversation", () => {
  localStorage.setItem(STORAGE_KEY, "{not json");
  loadFileTabs();
  render(<QueryClientProvider client={new QueryClient()}><MemoryRouter><EmptyConversation /></MemoryRouter></QueryClientProvider>);
  expect(screen.getByRole("status")).toHaveTextContent(/could not be read.*reset; open files again/);
});
