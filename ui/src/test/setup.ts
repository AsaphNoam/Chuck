import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { clearChatDraftCacheForTests } from "../components/chat/drafts";

import { STORAGE_KEY as FILE_TABS_KEY } from "../features/fileTabs/model";
import { loadFileTabs } from "../features/fileTabs/store";

// The composer's in-memory draft copy outlives each test's localStorage.clear(),
// and saved file tabs are a page-lifetime store restored once at import.
afterEach(() => {
  clearChatDraftCacheForTests();
  localStorage.removeItem(FILE_TABS_KEY);
  loadFileTabs();
});
