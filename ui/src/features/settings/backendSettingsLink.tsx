import type { ReactNode } from "react";

export function backendSettingsHref(backendId: string): string {
  const query = new URLSearchParams({ tab: "backends", backend: backendId });
  return `/settings?${query.toString()}`;
}

export function isProviderSetupErrorCode(code: unknown): code is string {
  return code === "provider_executable_missing" ||
    code === "provider_executable_invalid" ||
    code === "bundled_provider_unavailable" ||
    code === "provider_incompatible";
}

export function BackendSettingsLink({ backendId, children, onClick }: { backendId: string; children?: ReactNode; onClick?: () => void }) {
  return (
    <a href={backendSettingsHref(backendId)} target="_blank" rel="noopener" aria-label="Open backend settings (opens in a new tab)" onClick={onClick}>
      {children ?? "Open backend settings"}
    </a>
  );
}
