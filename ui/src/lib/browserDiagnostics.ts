// Desktop browser failures reach Chuck's dashboard log as allowlisted words
// only: never console text, paths or page content (TS-03.R92, INV §22).
export type BrowserDiagnosticReason = "error" | "unhandled_rejection" | "render_error" | "shell_missing";

const ROUTE_KINDS: Record<string, string> = {
  "": "dashboard",
  project: "project",
  agent: "agent",
  "think-tank": "think_tank",
  archive: "archive",
  pipelines: "pipelines",
  tasks: "tasks",
  settings: "settings",
};

export function routeKind(pathname: string) {
  return ROUTE_KINDS[pathname.split("/")[1] ?? ""] ?? "other";
}

const visitId = Array.from(crypto.getRandomValues(new Uint8Array(8)), (byte) => byte.toString(16).padStart(2, "0")).join("");
const reported = new Set<string>();
let shellSeen = false;
let installed = false;

function shellPresent() {
  const present = document.querySelector('[data-ui="app-shell"]') !== null;
  if (present) shellSeen = true;
  return present;
}

// One report per reason and route kind per page visit, from the desktop
// bootstrap only: the phone cannot reach this route (TS-13).
export function reportBrowserDiagnostic(reason: BrowserDiagnosticReason) {
  if (!installed) return;
  const route = routeKind(window.location.pathname);
  const key = `${reason}:${route}`;
  if (reported.has(key)) return;
  reported.add(key);
  const body = { reason, route_kind: route, mount_state: shellPresent() ? "mounted" : "missing", visit_id: visitId };
  void fetch("/api/diagnostics/browser", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    keepalive: true,
  }).catch(() => undefined);
}

// A foreground return that finds a once-mounted shell gone is reported; a
// healthy return stays quiet. The page cannot attest to pixels on screen.
function onVisibility() {
  if (document.visibilityState !== "visible") {
    shellPresent();
    return;
  }
  requestAnimationFrame(() => {
    if (shellSeen && !shellPresent()) reportBrowserDiagnostic("shell_missing");
  });
}

export function installBrowserDiagnostics() {
  if (installed) return;
  installed = true;
  window.addEventListener("error", () => reportBrowserDiagnostic("error"));
  window.addEventListener("unhandledrejection", () => reportBrowserDiagnostic("unhandled_rejection"));
  document.addEventListener("visibilitychange", onVisibility);
}

export function resetBrowserDiagnosticsForTests() {
  reported.clear();
  shellSeen = false;
}
