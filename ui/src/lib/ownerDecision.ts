import type { PipelineRunDetail } from "../schemas/pipeline";

type StageTask = PipelineRunDetail["stage_tasks"][number];

/** Visit number shown for a stage task, matching the visit labels in the run views. */
export const visitNumber = (task: StageTask) => task.visit_sequence || task.stage_index + 1;

/** Label for a decision-only owner visit about a room task's result (FS-14.R103). */
export function decisionOnlyLabel(task: StageTask, tasks: StageTask[]) {
  if (!task.decision_source_task_id) return "";
  const source = tasks.find((item) => item.task_id === task.decision_source_task_id);
  return source ? `Decision about room result, visit ${visitNumber(source)}` : "Decision about room result";
}

/** Evidence visits cited by a decision: step title and the visit that satisfied it. */
export function evidenceVisits(task: StageTask, tasks: StageTask[], titles: Map<string, string>) {
  return Object.entries(task.decision?.required_checks ?? {}).map(([stepId, taskId]) => {
    const visit = tasks.find((item) => item.task_id === taskId);
    return { stepTitle: titles.get(stepId) ?? stepId, taskId, label: visit ? `Visit ${visitNumber(visit)}` : taskId };
  });
}
