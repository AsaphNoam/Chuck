import { useEffect, useRef, useState, type ReactNode } from "react";
import type { AgentState } from "../api/types";
import { archiveAgent, cloneAgent, resumeAgent, stopAgent } from "../api/client";
import { ConfirmDialog } from "../components/ui";
import { AgentManagement } from "./AgentScreen";
import { PhoneSheet } from "./PhoneSheet";
import { useConnection } from "./connection";
import { navigate } from "./router";

export function AgentPress({ onOpen, children }: { onOpen: () => void; children: ReactNode }) {
  const timer = useRef<ReturnType<typeof setTimeout>>();
  const start = useRef<{ x: number; y: number }>();
  const held = useRef(false);
  const cancel = () => { clearTimeout(timer.current); timer.current = undefined; };
  useEffect(() => {
    window.addEventListener("scroll", cancel, true);
    return () => { cancel(); window.removeEventListener("scroll", cancel, true); };
  }, []);
  return <li
    onTouchStart={(event) => {
      cancel(); held.current = false;
      if (event.touches.length !== 1) return;
      start.current = { x: event.touches[0].clientX, y: event.touches[0].clientY };
      const button = event.currentTarget.querySelector("button");
      timer.current = setTimeout(() => {
        held.current = true;
        button?.focus();
        onOpen();
      }, 500);
    }}
    onTouchMove={(event) => {
      const touch = event.touches[0];
      if (!touch || !start.current || Math.hypot(touch.clientX - start.current.x, touch.clientY - start.current.y) > 10) cancel();
    }}
    onTouchEnd={cancel}
    onTouchCancel={cancel}
    onClickCapture={(event) => { if (held.current) { event.preventDefault(); event.stopPropagation(); held.current = false; } }}
    onContextMenu={(event) => { event.preventDefault(); cancel(); held.current = true; event.currentTarget.querySelector("button")?.focus(); onOpen(); }}
    onKeyDown={(event) => { if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) { event.preventDefault(); onOpen(); } }}
  >{children}</li>;
}

export function AgentActions({ agentId, onClose }: { agentId: string; onClose: () => void }) {
  const agent = useConnection((state) => state.agents[agentId]);
  const offline = useConnection((state) => state.link !== "connected");
  const [action, setAction] = useState<"rename" | "group" | "runtime" | null>(null);
  const [confirm, setConfirm] = useState<"stop" | "archive" | null>(null);
  const [name, setName] = useState(agent?.name ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => { if (!agent || agent.archived) onClose(); }, [agent, onClose]);
  if (!agent || agent.archived) return null;
  const act = async (fn: () => Promise<unknown>, _clears?: boolean, after?: (result: unknown) => void) => {
    if (offline || busy) return;
    setBusy(true); setError(null);
    try { const result = await fn(); after?.(result); onClose(); }
    catch (err) { setError(err instanceof Error ? err.message : String(err)); }
    finally { setBusy(false); }
  };
  const disabled = offline || busy;
  return <PhoneSheet title={action === "rename" ? "Rename agent" : action === "group" ? "Move to group" : action === "runtime" ? "Switch runtime" : agent.name || "Agent actions"} onClose={onClose}>
    {offline && <p className="phone-meta">Reconnect to the Mac to change this agent.</p>}
    {action ? <AgentManagement agent={agent as AgentState} offline={offline} busy={busy} act={(...args) => { void act(...args); }} rename={name} setRename={setName} error={error} action={action} /> : <div className="phone-form">
      <button type="button" onClick={() => { navigate(`/agent/${encodeURIComponent(agentId)}`); onClose(); }}>Open chat</button>
      <button type="button" disabled={disabled} onClick={() => setAction("rename")}>Rename</button>
      <button type="button" disabled={disabled} onClick={() => agent.running ? setConfirm("stop") : void act(() => resumeAgent(agentId))}>{agent.running ? "Stop" : "Resume"}</button>
      <button type="button" disabled={disabled || !agent.running || agent.interface !== "chat"} title={agent.running && agent.interface === "chat" ? undefined : "A running chat agent is required"} onClick={() => setAction("runtime")}>Switch runtime</button>
      <button type="button" disabled={disabled || !agent.clone?.available} title={agent.clone?.reason} onClick={() => void act(() => cloneAgent(agentId), false, (result) => navigate(`/agent/${encodeURIComponent((result as { agent: { agent_id: string } }).agent.agent_id)}`))}>Clone</button>
      {agent.clone && !agent.clone.available && <p className="phone-meta">{agent.clone.reason}</p>}
      <button type="button" disabled={disabled} onClick={() => setAction("group")}>Move to group</button>
      <button type="button" className="ad-button-danger" disabled={disabled} onClick={() => setConfirm("archive")}>Archive</button>
      {!confirm && error && <p className="phone-error" role="alert">{error}</p>}
    </div>}
    <ConfirmDialog open={!!confirm} title={confirm === "stop" ? "Stop this agent?" : "Archive this agent?"} confirmLabel={confirm === "stop" ? "Stop agent" : "Archive agent"} destructive pending={busy} confirmDisabled={offline} onCancel={() => setConfirm(null)} onConfirm={() => void act(() => confirm === "stop" ? stopAgent(agentId) : archiveAgent(agentId))}>
      <p>{confirm === "stop" ? "Its session remains available to resume later." : "Restore is available on the desktop."}</p>
      {error && <p className="phone-error" role="alert">{error}</p>}
    </ConfirmDialog>
  </PhoneSheet>;
}
