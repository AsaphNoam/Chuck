import type { ReactNode } from "react";
import { MemoryRouter } from "react-router-dom";
import { ownerKey } from "./model";
import { CONVERSATION_FILE_KEYS, useFileTabRoute, writeFileTarget, type FileTabRoute } from "./useFileTabRoute";
import type { FileLink } from "../../components/chat/renderers/filePath";

// Test harness: a conversation route whose address starts at `initial`, giving
// its children the same saved-tab route the agent screens use.
export function ConversationTabsRoute({ agentId, initial, children }: {
  agentId: string;
  initial: FileLink | null;
  children: (tabs: FileTabRoute) => ReactNode;
}) {
  const query = writeFileTarget(new URLSearchParams(), CONVERSATION_FILE_KEYS, initial).toString();
  return (
    <MemoryRouter initialEntries={[`/agent/${agentId}${query ? `?${query}` : ""}`]}>
      <Tabs agentId={agentId}>{children}</Tabs>
    </MemoryRouter>
  );
}

function Tabs({ agentId, children }: { agentId: string; children: (tabs: FileTabRoute) => ReactNode }) {
  const tabs = useFileTabRoute(ownerKey("conversation", agentId), CONVERSATION_FILE_KEYS);
  return <>{children(tabs)}</>;
}
