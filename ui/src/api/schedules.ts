import { queryOptions, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { z } from "zod";
import { scheduleListSchema, scheduleSchema, type Schedule, type ScheduleRule } from "../schemas/schedule";
import { TASK_QUERY_KEYS, request } from "./tasks";

export const SCHEDULE_QUERY_KEYS = {
  all: ["schedules"] as const,
  project: (project: string) => ["schedules", project] as const,
};

export interface ScheduleInput {
  project: string;
  display_name: string;
  instruction: string;
  target_kind: "agent" | "launch";
  target_agent_id?: string;
  role?: string;
  backend?: string;
  model?: string;
  effort?: string;
  fast?: boolean;
  rule: Partial<ScheduleRule>;
  run_only_on_time: boolean;
  /** An edit replaces the template's attachments, so it carries the current ones. */
  attachments?: { context_ref_id: string; label?: string; description?: string }[];
  revision?: number;
}

const send = (method: string, body?: unknown): RequestInit => ({
  method,
  headers: { "Content-Type": "application/json" },
  body: body === undefined ? undefined : JSON.stringify(body),
});

export function scheduleListOptions(project: string) {
  return queryOptions({
    queryKey: SCHEDULE_QUERY_KEYS.project(project),
    enabled: Boolean(project),
    queryFn: ({ signal }) => request(`/api/schedules?project=${encodeURIComponent(project)}`, scheduleListSchema, { signal }),
    select: (data: { schedules: Schedule[] }) => data.schedules,
  });
}

export function useSchedules(project: string | undefined) {
  return useQuery(scheduleListOptions(project ?? ""));
}

/** Every schedule control invalidates the project's schedules and tasks, since
 *  a change can create or stop creating work (FS-16.R56). */
function useScheduleAction<Input, Output>(project: string | undefined, run: (input: Input) => Promise<Output>) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: run,
    onSuccess: () => {
      client.invalidateQueries({ queryKey: SCHEDULE_QUERY_KEYS.project(project ?? "") });
      client.invalidateQueries({ queryKey: TASK_QUERY_KEYS.project(project ?? "") });
    },
  });
}

export function useCreateSchedule(project: string | undefined) {
  return useScheduleAction(project, (input: ScheduleInput) => request("/api/schedules", scheduleSchema, send("POST", input)));
}

export function useEditSchedule(project: string | undefined) {
  return useScheduleAction(project, (input: ScheduleInput & { scheduleID: string }) => {
    const { scheduleID, ...body } = input;
    return request(`/api/schedules/${encodeURIComponent(scheduleID)}`, scheduleSchema, send("PUT", body));
  });
}

export function usePauseSchedule(project: string | undefined) {
  return useScheduleAction(project, (scheduleID: string) =>
    request(`/api/schedules/${encodeURIComponent(scheduleID)}/pause`, scheduleSchema, send("POST", {})));
}

export function useResumeSchedule(project: string | undefined) {
  return useScheduleAction(project, (scheduleID: string) =>
    request(`/api/schedules/${encodeURIComponent(scheduleID)}/resume`, scheduleSchema, send("POST", {})));
}

export function useDeleteSchedule(project: string | undefined) {
  return useScheduleAction(project, (scheduleID: string) =>
    request(`/api/schedules/${encodeURIComponent(scheduleID)}`, z.undefined(), { method: "DELETE" }));
}
