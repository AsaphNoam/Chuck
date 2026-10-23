import { z } from "zod";

export const SCHEDULE_FREQUENCIES = ["daily", "weekdays", "weekly", "every_n_days", "every_n_weeks"] as const;
export const SCHEDULE_ENDS = ["never", "count", "until"] as const;
export const WEEKDAYS = ["mon", "tue", "wed", "thu", "fri", "sat", "sun"] as const;

/** The closed recurrence rule (FS-16.R54); its JSON is the server's. */
export const scheduleRuleSchema = z.object({
  start_date: z.string(),
  times: z.array(z.string()),
  frequency: z.enum(SCHEDULE_FREQUENCIES),
  weekdays: z.array(z.string()).nullable().optional().default([]),
  interval: z.number().optional().default(0),
  end_kind: z.enum(SCHEDULE_ENDS),
  end_count: z.number().optional().default(0),
  end_date: z.string().optional().default(""),
});

export const scheduleSchema = z.object({
  schedule_id: z.string(),
  project: z.string(),
  display_name: z.string(),
  instruction: z.string(),
  target_kind: z.enum(["agent", "launch"]),
  target_agent_id: z.string().optional().default(""),
  role: z.string().optional().default(""),
  backend: z.string().optional().default(""),
  model: z.string().optional().default(""),
  effort: z.string().optional().default(""),
  fast: z.boolean().optional().default(false),
  rule: scheduleRuleSchema,
  rule_text: z.string(),
  run_only_on_time: z.boolean(),
  state: z.enum(["active", "paused", "finished"]),
  pause_reason: z.string().optional().default(""),
  created_by_kind: z.string(),
  created_by_agent_id: z.string().optional().default(""),
  next_due_at: z.string().optional(),
  occurrences_elapsed: z.number(),
  last_occurrence_at: z.string().optional(),
  last_disposition: z.enum(["", "created", "missed", "paused"]).optional().default(""),
  last_task_id: z.string().optional().default(""),
  revision: z.number(),
  attachments: z.array(z.object({
    context_ref_id: z.string(),
    label: z.string().optional().default(""),
    description: z.string().optional().default(""),
  })).optional().default([]),
});

export const scheduleListSchema = z.object({ schedules: z.array(scheduleSchema) });

export type Schedule = z.output<typeof scheduleSchema>;
export type ScheduleRule = z.output<typeof scheduleRuleSchema>;

/** A schedule that paused itself because its target cannot run needs a person
 *  (FS-16.R57); one a person or agent paused does not. */
export function scheduleNeedsAttention(schedule: Schedule): boolean {
  return schedule.state === "paused" && schedule.pause_reason !== "";
}
