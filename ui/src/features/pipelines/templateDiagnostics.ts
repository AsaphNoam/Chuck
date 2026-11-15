import type { PipelineDiagnostic, PipelineTemplate } from "../../schemas/pipeline";
import { pipelineParticipantLabel } from "./pipelineLabels";

export function templateDiagnostic({ field, code, message }: PipelineDiagnostic, draft: PipelineTemplate, labels: Map<string, string>) {
  const control = field.replace(/\[(\d+)\]/g, ".$1");
  const [section, position, part, item, property, participantField] = control.split(".");
  const stageIndex = section === "stages" && /^\d+$/.test(position ?? "") ? Number(position) : undefined;
  const stage = stageIndex === undefined ? undefined : draft.stages[stageIndex];
  let label = "Template";
  let target = control;
  if (stage && stageIndex !== undefined) {
    const stageLabel = labels.get(stage.id) ?? `Stage ${stageIndex + 1}`;
    const stagePath = `stages.${stageIndex}`;
    const fields: Record<string, string> = { title: "Stage name", objective: "Stage instructions", coordination: "Coordination", dedicated_role: "Coordinator role" };
    label = `${stageLabel} · ${fields[part] ?? "Stage setup"}`;
    if (part === "inputs" || part === "outputs") {
      const binding = stage[part][Number(item)];
      const kind = part === "inputs" ? "Input" : "Output";
      label = `${stageLabel} · ${binding?.name || (binding ? `${kind} ${Number(item) + 1}` : `Handoff ${part}`)}`;
      if (part === "outputs" && property === "value") target = `${stagePath}.outputs.${item}.name`;
    } else if (part === "think_tank") {
      const member = item === "participants" && property !== undefined ? pipelineParticipantLabel(Number(property)) : "Think Tank";
      label = `${stageLabel} · ${item === "judge_role" ? "Judge role" : member}`;
      if (participantField === "id") target = `${stagePath}.think_tank.participants.${property}.role`;
      if (!item) target = `${stagePath}.coordination`;
    }
  } else if (section === "inputs") {
    const input = draft.inputs[Number(position)];
    label = input ? `Run input · ${input.name || Number(position) + 1}` : "Run inputs";
  } else {
    const fields: Record<string, string> = { title: "Template name", stages: "Stages", orchestrator_role: "Standing orchestrator role", orchestrator_instructions: "Shared orchestrator instructions" };
    label = fields[section] ?? label;
  }

  const hiddenReference = part === "id" || participantField === "id" || (part === "outputs" && property === "value");
  const explanations: Record<string, string> = {
    required: !control || part === "think_tank" ? "Complete this section." : "Enter a value.",
    unknown_role: "Choose an available role.",
    invalid_slug: "The saved template reference is invalid.",
    invalid_name: hiddenReference ? "The saved reference is invalid." : "Use lowercase letters, numbers and hyphens, up to 63 characters.",
    unresolved_value: "Choose an available run input or an output from an earlier stage.",
    future_binding: "Choose an output from an earlier stage.",
    think_tank_output: "Declare one output for the judge’s synthesis.",
    duplicate: hiddenReference ? "This saved reference is already in use." : "Use a unique name.",
    invalid_coordination: "Choose a coordination method.",
    unexpected: "Check the stage’s coordination settings.",
    legacy_field: "This template contains a setting from an older format.",
  };
  // Numeric limits are safe to retain. Unrecognized messages can contain internal fields.
  const limit = /^(?:must be at most|at most|limit must be between|template must be at most|a think_tank stage needs) \d+(?:(?: to | and )\d+)? [a-z0-9 ']+$/i.test(message);
  return { control: target, stageIndex, label, message: explanations[code] ?? (limit ? message.replace("think_tank", "Think Tank") : "Review this part of the template.") };
}
