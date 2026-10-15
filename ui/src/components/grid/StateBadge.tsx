import type { AgentState, AgentStatus } from "../../api/types";
import { Badge } from "../ui";

// A process that exited on error (crash, kill, failed switch) leaves the agent not
// running with status `error`; it reads "Error", not "Stopped" (FS-02.R79, FS-03.R89).
export function stoppedOnError(agent: Pick<AgentState, "running" | "state">): boolean {
  return !agent.running && agent.state === "error";
}

export const restartHint = (agentInterface: string) =>
  agentInterface === "terminal" ? "Resume to restart it." : "Send a message or Resume to restart it.";

export function StateBadge({ state }: { state: AgentStatus }) {
  return (
    <Badge
      className={`state-badge ${state}`}
      variant={state === "done" ? "success" : state === "error" ? "danger" : state === "busy" ? "warning" : state === "waiting_input" ? "info" : "neutral"}
      state={state}
      indicator
      data-testid="state-badge"
    >
      {state === "waiting_input" ? "Waiting" : state === "unknown" ? "-" : state.replace("_", " ")}
    </Badge>
  );
}
