import type { PipelineDiagnostic } from "../../schemas/pipeline";

export function startDiagnosticField(field: string) {
  return field.replace(/^assignments\.standing(?=\.|$)/, "orchestrator");
}

export function isRuntimeDiagnostic(field: string) {
  return /^(orchestrator|assignments|dedicated_assignments|think_tank_assignments)(\.|$)/.test(field);
}

// A Think Tank stage's combined context is the run goal, the stage's
// instructions and its inputs; the goal is the run field the operator can shorten.
const stageContextField = /^stages\.([^.]+)\.context$/;

export function focusStartDiagnostic(root: HTMLElement | null, path: string) {
  const field = stageContextField.test(path) ? "goal" : startDiagnosticField(path);
  const owners = [...(root?.querySelectorAll<HTMLElement>("[data-field]") ?? [])];
  // A specific field such as workspace.branch wins over the workspace group.
  const owner = owners.find((item) => field === item.dataset.field)
    ?? owners.find((item) => field.startsWith(`${item.dataset.field}.`));
  owner?.querySelector<HTMLElement>("input, select, textarea")?.focus();
}

export function startDiagnosticText({ field, code, message }: PipelineDiagnostic, runtimeLabel?: string, stageLabels?: Map<string, string>) {
  const context = stageContextField.exec(field);
  if (context && code === "too_large") {
    const stage = stageLabels?.get(context[1]) ?? "A Think Tank stage";
    const limit = /exceed (\d+) bytes/.exec(message)?.[1];
    return `${stage}: the run goal, stage instructions and inputs together exceed the ${limit ? `${Number(limit).toLocaleString("en-US")}-byte ` : ""}stage context limit. Shorten the run goal or inputs.`;
  }
  if (isRuntimeDiagnostic(field)) {
    const label = runtimeLabel ?? (startDiagnosticField(field) === "orchestrator" ? "Standing owner" : "Pipeline runtime");
    if (code === "required") return `${label} needs a configured backend and model.`;
    if (code === "unavailable") {
      // Stage validation reports these causes with the original backend, model or
      // role key. Keep the actionable cause while leaving those keys in the payload.
      if (message.startsWith("unknown role")) return `${label} uses a persona that is no longer available.`;
      if (message.startsWith("unknown project") || message === "project is archived") return "Select an available project.";
      if (message === "invalid project directory" || message === "project directory does not exist") return "The selected project checkout is unavailable.";
      if (message.startsWith("unknown backend")) return `${label} uses a backend that is no longer configured.`;
      if (message.startsWith("unknown model")) return `${label} uses a model that is no longer configured.`;
      if (message.startsWith("effort ") || message.endsWith("does not support effort selection")) return `${label} uses an effort level that the selected runtime does not support.`;
      if (message.startsWith("fast mode ") || message.endsWith("does not support fast mode")) return `${label} uses fast mode, which the selected runtime does not support.`;
      return `${label} has an unavailable runtime selection.`;
    }
    if (code === "unknown") return `${label} includes a selection that is no longer in the template.`;
    return `Check the configured runtime for ${label}.`;
  }
  if (field.startsWith("inputs.")) return code === "unknown" ? "One input is no longer defined by the template." : `${field.slice("inputs.".length)}: ${message}`;
  if (field.startsWith("workspace")) return message;
  if (field === "display_name") return message.replace(/display_name/g, "Run name");
  if (field === "goal") return message.replace(/^goal/, "Run goal");
  if (field === "template_id") return "Select an available pipeline template.";
  if (field === "project") return "Select an available project.";
  if (field.startsWith("stages.") || field.startsWith("stages[")) return "A stage configuration is invalid. Check the pipeline template.";
  return "This run could not start. Review the setup and try again.";
}
