import type { PipelineTemplate } from "../../schemas/pipeline";
import { UNAVAILABLE_AGENT, UNNAMED_AGENT, agentName, type LabelEntry } from "../../lib/labels";

export function pipelineOptionLabels(entries: readonly LabelEntry[], kind: string): [string, string][] {
  return entries.map(([key, name], index) => {
    const title = name.trim();
    if (!title) return [key, `Untitled ${kind} ${index + 1}`];
    const duplicate = entries.some(([, other], position) => position !== index && other.trim() === title);
    return [key, duplicate ? `${title} (${kind} ${index + 1})` : title];
  });
}

export function pipelineStageLabels(stages: PipelineTemplate["stages"]): Map<string, string> {
  return new Map(pipelineOptionLabels(stages.map(({ id, title }) => [id, title]), "stage"));
}

/** A stage owner's or coordinator's name. The projection leaves the name empty
 *  when it cannot resolve the agent; the ID stays for routing only (FS-02.R69). */
export function pipelineAgentLabel(agent: { agent_id?: string; name?: string; route?: string }, pending: string): string {
  if (agent.name?.trim()) return agent.name.trim();
  if (!agent.agent_id) return pending;
  return agent.route === "unavailable" ? UNAVAILABLE_AGENT : UNNAMED_AGENT;
}

/** The shared-workspace warning's lines: runs first by name, agents grouped by
 *  name, capped so a busy project cannot push the confirmation out of view. */
export function workspaceConflictLines(conflicts: readonly { kind: "agent" | "run"; name: string }[], limit = 5): { lines: string[]; hidden: number } {
  const runs = conflicts.filter((item) => item.kind === "run").map((item) => `Run: ${item.name.trim() || "Untitled run"}`);
  const agents = new Map<string, number>();
  for (const item of conflicts) if (item.kind === "agent") agents.set(agentName(item.name), (agents.get(agentName(item.name)) ?? 0) + 1);
  const agentLines = [...agents].map(([name, count]) => count > 1 ? `${count} agents named ${name}` : `Agent: ${name}`);
  const all = [...runs, ...agentLines];
  return { lines: all.slice(0, limit), hidden: Math.max(0, all.length - limit) };
}

export function pipelineParticipantLabel(index: number): string {
  return `Participant ${index + 1}`;
}

export function newPipelineKey(prefix: string): string {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return `${prefix}-${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

/** Product sentences for the pause codes the run engine records; any other
 *  reason is already prose and passes through. */
const attentionSentences: Record<string, string> = {
  owner_not_reporting: "The stage agent ended several turns without reporting a result.",
  restart_interrupted: "Chuck restarted while the stage agent was working.",
  owner_unavailable: "The stage agent could not be resumed.",
  interrupted: "The stage agent stopped before reporting a result.",
  recovery_exhausted: "Automatic recovery used its whole allowance without finishing the stage.",
  needs_input: "The run is waiting for your answer.",
  approval_required: "A stage needs your approval before the run can finish.",
  required_check_missing: "A required stage has no accepted result yet, so the run cannot finish.",
  controller_recovery_failed: "Chuck could not finish a run step and is holding it for you to retry.",
  restart_reconcile_failed: "Chuck could not finish a run step after restarting and is holding it for you to retry.",
};

export function pipelineAttentionText(reason: string, stages?: Map<string, string>): string {
  if (attentionSentences[reason]) return attentionSentences[reason];
  const context = /^stages\.([^.:]+)\.context:/.exec(reason);
  if (!context) return reason;
  const stage = stages?.get(context[1]) ?? "The Think Tank stage";
  return `${stage} has too much material for its room. Resume the run to give the owner a correction.`;
}

export function pipelineActionError(reason: unknown): string {
  if (!(reason instanceof Error)) return "The pipeline action could not be completed.";
  if (/^step [a-z0-9-]+ is missing$/.test(reason.message)) return "The next stage is unavailable in this run.";
  if (/^think_tank stage [a-z0-9-]+ is not runnable$/.test(reason.message)) return "Check the stage’s Think Tank settings in the template.";
  return reason.message;
}
