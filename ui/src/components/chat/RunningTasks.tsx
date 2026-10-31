import { useEffect, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { useAgentStore } from "../../store/agentStore";
import { useTranscriptStore } from "../../store/transcriptStore";
import { stopBackgroundTask } from "../../api/client";
import { collectTasks, type BackgroundTask } from "./runtimeActivity";
import { toolLine } from "./toolLine";
import type { TranscriptEvent } from "../../api/types";

const EMPTY: TranscriptEvent[] = [];

// Both desktop composers share the durable lifecycle projection (INV §2).
export function RunningTasks({ agentId }: { agentId: string }) {
  const agent = useAgentStore((state) => state.agents[agentId]);
  const events = useTranscriptStore((state) => state.byAgent[agentId]) ?? EMPTY;
  const tasks = collectTasks(events).filter((task) => task.state === "running");
  const [open, setOpen] = useState(false);
  const [now, setNow] = useState(Date.now());
  // Stop state outlives the dismissible popover content (FS-03.R107, INV §8).
  const [stops, setStops] = useState<Record<string, "stopping" | "error">>({});
  const stop = (task: BackgroundTask) => {
    setStops((current) => ({ ...current, [task.taskId]: "stopping" }));
    void stopBackgroundTask(agentId, task.taskId, task.activityId).catch(() => setStops((current) => ({ ...current, [task.taskId]: "error" })));
  };
  const visible = agent?.running && tasks.length > 0;
  useEffect(() => {
    if (!visible) { setOpen(false); return; }
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [visible]);
  if (!visible) return null;
  return (
    <Dialog.Root modal={false} open={open} onOpenChange={setOpen}>
      <div className="running-tasks" data-ui="runtime-activity" data-slot="running-indicator" data-state="active">
        <Dialog.Trigger asChild><button type="button" className="ad-button-secondary">{tasks.length} running</button></Dialog.Trigger>
        <Dialog.Content className="running-tasks-popover" aria-describedby={undefined}
          onKeyDownCapture={(event) => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); setOpen(false); } }}
          onEscapeKeyDown={(event) => { event.preventDefault(); setOpen(false); }}>
          <Dialog.Title>Running commands</Dialog.Title>
          <ul className="background-task-rows">
            {tasks.map((task) => {
              const call = events.slice().reverse().find((event) => (event.kind ?? event.type) === "tool_call" && event.tool_call_id === task.toolCallId);
              const seconds = Math.max(0, Math.floor((now - Date.parse(task.startedAt ?? "")) / 1000));
              return <RunningTask key={task.taskId} task={task} stop={stops[task.taskId]} onStop={() => stop(task)} label={call ? toolLine(call).label : task.toolTitle || task.name || "Command"} elapsed={Number.isFinite(seconds) ? `${Math.floor(seconds / 60)}m ${seconds % 60}s` : ""} controllable={Boolean(agent.runtime_capabilities?.background_task_stop)} onJump={call?.seq == null ? undefined : () => {
                setOpen(false);
                window.dispatchEvent(new CustomEvent("chuck-reveal-task", { detail: { agentId, seq: call.seq } }));
              }} />;
            })}
          </ul>
          <Dialog.Close asChild><button type="button" className="ad-button-secondary">Close</button></Dialog.Close>
        </Dialog.Content>
      </div>
    </Dialog.Root>
  );
}

function RunningTask({ task, stop, onStop, label, elapsed, controllable, onJump }: { task: BackgroundTask; stop?: "stopping" | "error"; onStop: () => void; label: string; elapsed: string; controllable: boolean; onJump?: () => void }) {
  const stopping = stop === "stopping";
  return <li className="background-task" data-slot="task" data-state="running">
    <span className="background-task-state">{stopping ? "Stopping…" : "Running"}</span>
    <button type="button" className="background-task-name tool-toggle" title={label} disabled={!onJump} onClick={onJump}>{label}</button>
    <span className="background-task-meta">{[elapsed, task.owner].filter(Boolean).join(" · ")}</span>
    {controllable && task.canStop && !task.fenced && <button type="button" className="ad-button-danger" disabled={stopping} onClick={onStop}>Stop</button>}
    {stop === "error" &&<p className="background-task-error" role="alert">Could not stop this task. Try again.</p>}
  </li>;
}
