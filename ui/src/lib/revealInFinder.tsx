import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { getCapabilities } from "../api/client";
import { revealChatFileInFinder } from "../api/projectFiles";

// Open in Finder is offered on the local macOS desktop only (FS-03.R119). The host
// capability is read once per page load.
let capability: Promise<boolean> | undefined;

export function resetRevealSupportedForTest() { capability = undefined; }

export function useRevealSupported(): boolean {
  const [supported, setSupported] = useState(false);
  useEffect(() => {
    if (window.location.pathname.startsWith("/phone")) return;
    let active = true;
    capability ??= Promise.resolve().then(() => getCapabilities()).then((caps) => caps.reveal_in_finder === true, () => false);
    void capability.then((value) => { if (active) setSupported(value); });
    return () => { active = false; };
  }, []);
  return supported;
}

export type RevealInFinder = (path: string) => Promise<unknown>;

const RevealContext = createContext<RevealInFinder | undefined>(undefined);

/** Binds Open in Finder to one chat agent's read boundary for everything beneath it. */
export function RevealInFinderProvider({ agentId, children }: { agentId: string; children: ReactNode }) {
  const supported = useRevealSupported();
  const reveal = useMemo<RevealInFinder | undefined>(() => supported ? (path) => revealChatFileInFinder(agentId, path) : undefined, [supported, agentId]);
  return <RevealContext.Provider value={reveal}>{children}</RevealContext.Provider>;
}

export function useRevealInFinder(): RevealInFinder | undefined {
  return useContext(RevealContext);
}
