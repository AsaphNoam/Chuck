import type { PipelineTemplate } from "../../schemas/pipeline";
import type { LabelEntry } from "../../lib/labels";

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

export function pipelineParticipantLabel(index: number): string {
  return `Participant ${index + 1}`;
}

export function newPipelineKey(prefix: string): string {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return `${prefix}-${Array.from(bytes, (byte) => byte.toString(16).padStart(2, "0")).join("")}`;
}

export function pipelineAttentionText(reason: string, stages?: Map<string, string>): string {
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
