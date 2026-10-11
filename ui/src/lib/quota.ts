import { z } from "zod";

export const quotaStateSchema = z.object({
  episode_id: z.string().min(1),
  revision: z.number().int().nonnegative(),
  observed_at: z.string().datetime({ offset: true }),
  reset_at: z.string().datetime({ offset: true }).nullable(),
  due_at: z.string().datetime({ offset: true }).nullable(),
  continuation_state: z.enum([
    "disabled",
    "unknown_reset",
    "scheduled",
    "waiting",
    "starting",
    "cancelled",
    "needs_attention",
  ]),
  reason_code: z.string().max(64).optional(),
});

export type QuotaState = z.infer<typeof quotaStateSchema>;

/** Malformed or partial wire data must not present a false quota interruption. */
export function decodeQuotaState(value: unknown): QuotaState | undefined {
  const result = quotaStateSchema.safeParse(value);
  return result.success ? result.data : undefined;
}

/** Format a server-authoritative RFC3339 time in the operator's local timezone. */
export function formatQuotaTime(value: string | null | undefined): string | undefined {
  if (!value) return undefined;
  const instant = new Date(value);
  if (!Number.isFinite(instant.getTime())) return undefined;
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
  }).format(instant);
}
