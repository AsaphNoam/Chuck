import { useState } from "react";
import { AutoGrowTextarea, Button } from "../../components/ui";
import {
  useCreateSchedule, useDeleteSchedule, useEditSchedule, usePauseSchedule, useResumeSchedule,
  type ScheduleInput,
} from "../../api/schedules";
import { SCHEDULE_FREQUENCIES, WEEKDAYS, scheduleNeedsAttention, type Schedule, type ScheduleRule } from "../../schemas/schedule";
import type { Task } from "../../schemas/task";
import { EMPTY_TARGET, RunOnlyOnTimeField, TargetFields, errorMessage, useTargetPayload, type TargetDraft } from "./taskForms";
import { startTimeLabel, taskStatus } from "./taskWork";

const FREQUENCY_LABELS: Record<ScheduleRule["frequency"], string> = {
  daily: "Every day",
  weekdays: "Weekdays (Mon–Fri)",
  weekly: "Weekly on chosen days",
  every_n_days: "Every N days",
  every_n_weeks: "Every N weeks on chosen days",
};

const WEEKDAY_LABELS: Record<string, string> = { mon: "Mon", tue: "Tue", wed: "Wed", thu: "Thu", fri: "Fri", sat: "Sat", sun: "Sun" };

type RuleDraft = {
  startDate: string; times: string; frequency: ScheduleRule["frequency"]; weekdays: string[];
  interval: number; endKind: ScheduleRule["end_kind"]; endCount: number; endDate: string;
};

function today(): string {
  const now = new Date();
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
}

function ruleDraft(schedule?: Schedule): RuleDraft {
  const rule = schedule?.rule;
  return {
    startDate: rule?.start_date ?? today(),
    times: rule?.times.join(", ") ?? "09:00",
    frequency: rule?.frequency ?? "daily",
    weekdays: rule?.weekdays ?? [],
    interval: rule?.interval || 2,
    endKind: rule?.end_kind ?? "never",
    endCount: rule?.end_count || 3,
    endDate: rule?.end_date ?? "",
  };
}

function ruleInput(draft: RuleDraft): Partial<ScheduleRule> {
  const weekly = draft.frequency === "weekly" || draft.frequency === "every_n_weeks";
  return {
    start_date: draft.startDate,
    // Times are typed as a short list; the server checks each and refuses repeats.
    times: draft.times.split(/[,\s]+/).filter(Boolean),
    frequency: draft.frequency,
    weekdays: weekly ? draft.weekdays : [],
    interval: draft.frequency.startsWith("every_n_") ? draft.interval : 0,
    end_kind: draft.endKind,
    end_count: draft.endKind === "count" ? draft.endCount : 0,
    end_date: draft.endKind === "until" ? draft.endDate : "",
  };
}

function targetDraft(schedule?: Schedule): TargetDraft {
  if (!schedule) return EMPTY_TARGET;
  return {
    kind: schedule.target_kind, agentID: schedule.target_agent_id, role: schedule.role,
    backend: schedule.backend, model: schedule.model, effort: schedule.effort, fast: schedule.fast,
  };
}

/** ScheduleForm creates a schedule, or edits one when given it: editing
 *  changes only future occurrences (FS-16.R56). */
export function ScheduleForm({ projects, initialProject, schedule, onDone }: {
  projects: string[];
  initialProject: string;
  schedule?: Schedule;
  onDone?: () => void;
}) {
  const [project, setProject] = useState(schedule?.project ?? initialProject);
  const create = useCreateSchedule(project || undefined);
  const edit = useEditSchedule(project || undefined);
  const [name, setName] = useState(schedule?.display_name ?? "");
  const [instruction, setInstruction] = useState(schedule?.instruction ?? "");
  const [target, setTarget] = useState<TargetDraft>(() => targetDraft(schedule));
  const { payload } = useTargetPayload(target);
  const [rule, setRule] = useState<RuleDraft>(() => ruleDraft(schedule));
  const [onTimeOnly, setOnTimeOnly] = useState(schedule?.run_only_on_time ?? true);
  const [error, setError] = useState("");
  const [saved, setSaved] = useState("");
  const setRulePart = (patch: Partial<RuleDraft>) => setRule((current) => ({ ...current, ...patch }));
  const weekly = rule.frequency === "weekly" || rule.frequency === "every_n_weeks";
  const pending = create.isPending || edit.isPending;
  const prefix = schedule ? `schedule-${schedule.schedule_id}` : "schedule-new";

  return (
    <form
      className="task-create schedule-form"
      data-slot={schedule ? "controls" : "create"}
      onSubmit={(event) => {
        event.preventDefault();
        setError("");
        setSaved("");
        const input: ScheduleInput = {
          project, display_name: name, instruction, ...payload,
          rule: ruleInput(rule), run_only_on_time: onTimeOnly, attachments: schedule?.attachments,
        };
        const action = schedule
          ? edit.mutateAsync({ ...input, scheduleID: schedule.schedule_id, revision: schedule.revision })
          : create.mutateAsync(input);
        action
          .then((result) => {
            setSaved(schedule ? "Saved. Future occurrences use the new settings." : `Created “${result.display_name}”: ${result.rule_text}.`);
            if (!schedule) {
              setName("");
              setInstruction("");
            }
            onDone?.();
          })
          .catch((err: unknown) => setError(errorMessage(err)));
      }}
    >
      {!schedule && (
        <label>
          Project
          <select value={project} required onChange={(e) => { setProject(e.target.value); setTarget((current) => ({ ...current, agentID: "" })); }}>
            <option value="">Choose a project</option>
            {projects.map((item) => <option key={item} value={item}>{item}</option>)}
          </select>
        </label>
      )}
      <label>Name<input value={name} onChange={(e) => setName(e.target.value)} required /></label>
      <label>Instruction<AutoGrowTextarea value={instruction} onChange={(e) => setInstruction(e.target.value)} required /></label>
      <TargetFields project={project} draft={target} onChange={setTarget} />
      <fieldset className="schedule-rule">
        <legend>Repeat</legend>
        <label>
          How often
          <select value={rule.frequency} onChange={(e) => setRulePart({ frequency: e.target.value as RuleDraft["frequency"] })}>
            {SCHEDULE_FREQUENCIES.map((item) => <option key={item} value={item}>{FREQUENCY_LABELS[item]}</option>)}
          </select>
        </label>
        {rule.frequency.startsWith("every_n_") && (
          <label>N<input type="number" min={1} value={rule.interval} onChange={(e) => setRulePart({ interval: Number(e.target.value) })} required /></label>
        )}
        {weekly && (
          <div className="schedule-weekdays" role="group" aria-label="Days of the week">
            {WEEKDAYS.map((day) => (
              <label key={day}>
                <input
                  type="checkbox"
                  checked={rule.weekdays.includes(day)}
                  onChange={(e) => setRulePart({ weekdays: e.target.checked ? [...rule.weekdays, day] : rule.weekdays.filter((item) => item !== day) })}
                />
                {WEEKDAY_LABELS[day]}
              </label>
            ))}
          </div>
        )}
        <label>Times of day<input value={rule.times} onChange={(e) => setRulePart({ times: e.target.value })} placeholder="09:00, 17:30" required aria-describedby={`${prefix}-times`} /></label>
        <small id={`${prefix}-times`}>24-hour local times, separated by commas.</small>
        <label>Starting<input type="date" value={rule.startDate} onChange={(e) => setRulePart({ startDate: e.target.value })} required /></label>
        <label>
          Ends
          <select value={rule.endKind} onChange={(e) => setRulePart({ endKind: e.target.value as RuleDraft["endKind"] })}>
            <option value="never">Never</option>
            <option value="count">After a number of occurrences</option>
            <option value="until">After a date</option>
          </select>
        </label>
        {rule.endKind === "count" && <label>Occurrences<input type="number" min={1} value={rule.endCount} onChange={(e) => setRulePart({ endCount: Number(e.target.value) })} required /></label>}
        {rule.endKind === "until" && <label>Last date<input type="date" value={rule.endDate} onChange={(e) => setRulePart({ endDate: e.target.value })} required /></label>}
      </fieldset>
      <RunOnlyOnTimeField checked={onTimeOnly} onChange={setOnTimeOnly} />
      <Button type="submit" variant="primary" busy={pending}>{schedule ? "Save schedule" : "Create schedule"}</Button>
      {error && <p className="form-error" role="alert">{error}</p>}
      {saved && <p className="task-query-state" role="status">{saved}</p>}
    </form>
  );
}

function lastOccurrence(schedule: Schedule, tasks: Task[]): string {
  if (!schedule.last_occurrence_at) return "none yet";
  const when = startTimeLabel(schedule.last_occurrence_at);
  if (schedule.last_disposition === "missed") return `${when} · missed`;
  if (schedule.last_disposition === "paused") return `${when} · paused`;
  const task = tasks.find((item) => item.task_id === schedule.last_task_id);
  return `${when} · ${task ? taskStatus(task).label : "task removed"}`;
}

function nextOccurrence(schedule: Schedule): string {
  if (schedule.state === "finished") return "finished";
  if (schedule.state === "paused") return "paused";
  return schedule.next_due_at ? startTimeLabel(schedule.next_due_at) : "none";
}

/** ScheduleRow is one schedule: its rule in words, next and last occurrence,
 *  Run only on time, and its controls (FS-16.R56). */
function ScheduleRow({ schedule, tasks, projects }: { schedule: Schedule; tasks: Task[]; projects: string[] }) {
  const project = schedule.project;
  const pause = usePauseSchedule(project);
  const resume = useResumeSchedule(project);
  const remove = useDeleteSchedule(project);
  const edit = useEditSchedule(project);
  const [editing, setEditing] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [error, setError] = useState("");
  const run = (action: Promise<unknown>) => {
    setError("");
    action.catch((err: unknown) => setError(errorMessage(err)));
  };
  const nameID = `schedule-name-${schedule.schedule_id}`;
  const attention = scheduleNeedsAttention(schedule);

  return (
    <li className="schedule-row" data-slot="schedule" data-state={schedule.state} data-variant={attention ? "attention" : undefined} aria-labelledby={nameID}>
      <span className="task-name" id={nameID}>{schedule.display_name}</span>
      <span className="schedule-rule-text">{schedule.rule_text}</span>
      {attention && <p className="task-attention" data-slot="attention">Paused: {schedule.pause_reason}. Resume when the target can run again.</p>}
      <dl className="schedule-meta">
        <div><dt>Next</dt><dd>{nextOccurrence(schedule)}</dd></div>
        <div><dt>Last</dt><dd>{lastOccurrence(schedule, tasks)}</dd></div>
      </dl>
      <div className="task-row-actions" data-slot="actions">
        <label className="schedule-toggle">
          <input
            type="checkbox"
            checked={schedule.run_only_on_time}
            disabled={edit.isPending || schedule.state === "finished"}
            onChange={(e) => run(edit.mutateAsync({
              project, display_name: schedule.display_name, instruction: schedule.instruction,
              target_kind: schedule.target_kind, target_agent_id: schedule.target_agent_id || undefined,
              role: schedule.role || undefined, backend: schedule.backend || undefined, model: schedule.model || undefined,
              effort: schedule.effort || undefined, fast: schedule.fast, rule: schedule.rule,
              run_only_on_time: e.target.checked, attachments: schedule.attachments,
              scheduleID: schedule.schedule_id, revision: schedule.revision,
            }))}
          />
          Run only on time
        </label>
        {schedule.state === "active" && <Button onClick={() => run(pause.mutateAsync(schedule.schedule_id))} busy={pause.isPending}>Pause</Button>}
        {schedule.state === "paused" && <Button onClick={() => run(resume.mutateAsync(schedule.schedule_id))} busy={resume.isPending}>Resume</Button>}
        <Button onClick={() => setEditing((current) => !current)} aria-expanded={editing}>{editing ? "Close edit" : "Edit"}</Button>
        {confirmDelete ? (
          <>
            <Button variant="danger" onClick={() => run(remove.mutateAsync(schedule.schedule_id))} busy={remove.isPending}>Delete schedule</Button>
            <Button onClick={() => setConfirmDelete(false)}>Keep</Button>
          </>
        ) : <Button onClick={() => setConfirmDelete(true)}>Delete…</Button>}
      </div>
      {confirmDelete && <p className="task-waiting">Tasks this schedule already created are kept.</p>}
      {error && <p className="form-error" role="alert">{error}</p>}
      {editing && <ScheduleForm projects={projects} initialProject={project} schedule={schedule} onDone={() => setEditing(false)} />}
    </li>
  );
}

/** ScheduleList is a project's quiet Schedules section after its work. */
export function ScheduleList({ project, schedules, tasks, projects }: { project: string; schedules: Schedule[]; tasks: Task[]; projects: string[] }) {
  if (schedules.length === 0) return null;
  const headingID = `tasks-schedules-${project}`;
  return (
    <section className="tasks-schedules" data-slot="schedules" aria-labelledby={headingID}>
      <h3 id={headingID}>Schedules</h3>
      <ul>
        {schedules.map((schedule) => <ScheduleRow key={schedule.schedule_id} schedule={schedule} tasks={tasks} projects={projects} />)}
      </ul>
    </section>
  );
}
