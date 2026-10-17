import { decodeQuotaState, formatQuotaTime } from "../../lib/quota";

const reasonLabels: Record<string, string> = {
  busy: "Waiting for this agent",
  capacity: "Waiting for runtime capacity",
  store_failure: "Could not save continuation state",
  owner_unavailable: "Waiting for the owning task or room",
  project_ineligible: "The project is unavailable",
  archived: "The agent is archived",
  provider_start_failed: "The provider could not start",
  uncertain_delivery: "Delivery status is uncertain",
  non_quota_failure: "Continuation stopped after a non-quota failure",
  invalid_owner: "The owning work is unavailable",
  check_unsupported: "Quota checks aren’t available for this provider setup. Continue manually.",
  check_failed: "The quota check failed. Continue manually.",
  check_timeout: "The quota check timed out. Continue manually.",
  check_exhausted: "No quota checks left. Continue manually.",
  check_interrupted: "A quota check was interrupted by a restart. Continue manually.",
  check_cleanup_failed: "A quota check couldn’t be cleaned up. Continue manually.",
  check_ready_but_quota: "A quota check passed, but the conversation hit the limit again. Continue manually.",
};

function checksLeft(n: number | undefined): string | undefined {
  if (n === undefined) return undefined;
  return n === 1 ? "1 check left" : `${n} checks left`;
}

export function QuotaBanner({
  value,
  cancelPending = false,
  cancelDisabled = false,
  cancelError,
  onCancel,
}: {
  value: unknown;
  cancelPending?: boolean;
  cancelDisabled?: boolean;
  cancelError?: string;
  onCancel?: () => void;
}) {
  const quota = decodeQuotaState(value);
  if (!quota) return null;

  const resetAt = formatQuotaTime(quota.reset_at);
  const dueAt = formatQuotaTime(quota.due_at);
  const checking = quota.next_action === "check";
  // A check due time is never a reset (FS-01.R50); an in-flight check stays cancellable.
  const pending = quota.continuation_state === "scheduled" || quota.continuation_state === "waiting" ||
    (checking && quota.continuation_state === "starting");
  const continuationText = (() => {
    if (checking) {
      switch (quota.continuation_state) {
        case "scheduled": return dueAt ? `Will check again at ${dueAt}` : "A quota check is scheduled";
        case "waiting": return dueAt ? `Waiting to check at ${dueAt}` : "Waiting to check";
        case "starting": return "Checking quota…";
      }
    }
    switch (quota.continuation_state) {
      case "scheduled": return dueAt ? `Will continue at ${dueAt}` : "Automatic continuation is scheduled";
      case "waiting": return dueAt ? `Waiting to continue at ${dueAt}` : "Waiting to continue";
      case "starting": return "Continuing…";
      case "cancelled": return "Continuation cancelled";
      case "needs_attention": return reasonLabels[quota.reason_code ?? ""] ?? "Automatic continuation needs attention";
      case "unknown_reset": return "Continue manually when the quota is available";
      case "disabled": return "Automatic continuation is off";
    }
  })();

  return (
    <section className="quota-banner" aria-label="Quota interruption">
      <div className="quota-banner-copy">
        <strong>Quota reached</strong>
        <span>{resetAt ? `Reset ${resetAt}` : "Reset time unknown"}</span>
        <span>{continuationText}</span>
        {checking && pending && checksLeft(quota.checks_remaining) && <span>{checksLeft(quota.checks_remaining)}</span>}
      </div>
      {pending && onCancel && (
        <button type="button" className="ad-button-secondary quota-banner-cancel" disabled={cancelPending || cancelDisabled} onClick={onCancel}>
          {cancelPending ? "Cancelling…" : "Cancel continuation"}
        </button>
      )}
      {cancelError && <p className="quota-banner-error" role="alert">{cancelError}</p>}
    </section>
  );
}
