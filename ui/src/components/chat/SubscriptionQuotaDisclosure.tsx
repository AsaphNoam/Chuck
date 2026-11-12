import type { SubscriptionQuota, SubscriptionQuotaWindow } from "../../api/types";
import { formatQuotaTime } from "../../lib/quota";

const PROVIDER_LABEL: Record<SubscriptionQuota["provider"], string> = { claude: "Claude", codex: "Codex" };

const REASON_TEXT: Record<NonNullable<SubscriptionQuota["reason_code"]>, string> = {
  unsupported_adapter: "This adapter version cannot read subscription quota.",
  no_subscription: "This sign-in has no subscription quota.",
  read_failed: "The last check failed; showing earlier values.",
  read_timeout: "The last check timed out; showing earlier values.",
};

// With no window to show, a failed check cannot claim earlier values.
const NO_VALUES_REASON_TEXT: Partial<typeof REASON_TEXT> = {
  read_failed: "The check failed. It updates after this backend's next turn.",
  read_timeout: "The check timed out. It updates after this backend's next turn.",
};

function WindowRow({ label, window }: { label: string; window: SubscriptionQuotaWindow }) {
  const used = window.used_percent;
  const reset = formatQuotaTime(window.reset_at);
  return (
    <div className="subscription-quota-row">
      <div className="subscription-quota-summary">
        <span>{label}</span>
        <strong>{used === undefined ? "Unknown" : `${Math.round(used)}% used`}</strong>
      </div>
      {used !== undefined && <meter min={0} max={100} value={used} aria-label={`${label} ${Math.round(used)}% used`} />}
      <em>{reset ? `Resets ${reset}` : "Reset time unknown"}</em>
    </div>
  );
}

/**
 * The chat header's Subscription quota disclosure (FS-09.R83, TS-08.R162): the
 * backend scope's two windows exactly as delivered. Opening it makes no request
 * and there is no Refresh control; the server keeps the snapshot current.
 */
export function SubscriptionQuotaDisclosure({ value }: { value?: SubscriptionQuota }) {
  if (!value) return null;
  const unavailable = value.availability === "unavailable";
  const checked = formatQuotaTime(value.checked_at);
  const reason = value.reason_code ? REASON_TEXT[value.reason_code] : undefined;
  return (
    <details className="subscription-quota">
      <summary>Subscription quota</summary>
      <div className="subscription-quota-body">
        <p className="subscription-quota-provider">{PROVIDER_LABEL[value.provider]} subscription</p>
        {unavailable && !value.windows.five_hour.observed_at && !value.windows.weekly.observed_at ? (
          <p>{(value.reason_code && NO_VALUES_REASON_TEXT[value.reason_code]) ?? reason ?? (checked ? "The provider reported no 5-hour or Weekly window." : "Not checked yet. It updates after this backend's next turn.")}</p>
        ) : (
          <>
            <WindowRow label="5-hour" window={value.windows.five_hour} />
            <WindowRow label="Weekly" window={value.windows.weekly} />
            {reason && <p role="status">{reason}</p>}
          </>
        )}
        {checked && <p className="subscription-quota-checked">Last checked {checked}</p>}
      </div>
    </details>
  );
}
