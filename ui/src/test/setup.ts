import "@testing-library/jest-dom/vitest";
import { afterEach } from "vitest";
import { clearChatDraftCacheForTests } from "../components/chat/drafts";

// The composer's in-memory draft copy outlives each test's localStorage.clear().
afterEach(() => clearChatDraftCacheForTests());
