// Renders the projects home with representative projects in every appearance at
// the desktop floor and a wider viewport, writing screenshots for rendered
// review (FS-12.R69, TS-08.R124). Every /api call and the event stream are
// stubbed; no Chuck server is needed.
//
//   node scripts/projects-render.mjs [outDir]
import { mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), "projects-render")] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const at = "2026-10-13T09:00:00Z";
const projects = {
  chuck: { title: "Chuck", cwd: "/tmp/chuck", color: [80, 120, 200], repo_backed: true },
  "chuck-quota": { title: "Chuck", cwd: "/tmp/chuck-quota", color: [60, 160, 120], repo_backed: true, worktree: { owned: true, branch: "quota-recovery-and-subscription-views" } },
  docs: { title: "Documentation site with a deliberately long project name", cwd: "/tmp/docs", color: [200, 120, 60] },
  empty: { title: "scratch", cwd: "/tmp/scratch", color: [150, 90, 190] },
};
let n = 0;
const agent = (project, state, running = true) => ({
  agent_id: `a${++n}`, name: `agent-${n}`, role: "impl", project, backend: "claude", model: "m", fast: false,
  interface: "chat", created_at: at, running, state, detail: "", context_pct: 0, updated_at: 0, archived: false,
});
const agents = [
  agent("chuck", "busy"), agent("chuck", "busy"), agent("chuck", "waiting_input"), agent("chuck", "idle", false),
  agent("chuck-quota", "error"), agent("docs", "done", false), agent("ghost", "idle"),
];

const server = await createServer({ server: { port: 5199, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
let failed = false;
try {
  for (const skin of ["", "sky-grove", "studio"]) {
    for (const width of [1024, 1440]) {
      const context = await browser.newContext({ viewport: { width, height: 900 } });
      await context.addInitScript((hydrated) => {
        // The shared worker would open its own stream past these stubs.
        delete window.SharedWorker;
        window.EventSource = class extends EventTarget {
          constructor() {
            super();
            setTimeout(() => {
              this.onopen?.();
              for (const data of hydrated) this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify(data) }));
            });
          }
          close() {}
        };
      }, [...agents.map((data) => ({ agent_id: data.agent_id, data })), { agent_id: "__hydrated__", data: { hydrated: true } }]);
      await context.route((url) => url.pathname.startsWith("/api/"), (route) => {
        const path = new URL(route.request().url()).pathname;
        if (path === "/api/projects") return route.fulfill({ json: projects });
        if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarded: true, onboarding_complete: true, onboarding: { satisfied: true, steps: [] } } });
        if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 } } });
        if (path === "/api/backends") return route.fulfill({ json: { version: 2, backends: {} } });
        if (path === "/api/config-sources") return route.fulfill({ json: [] });
        return route.fulfill({ json: {} });
      });
      const page = await context.newPage();
      page.on("pageerror", (error) => { failed = true; console.error(`page error: ${error.stack}`); });
      await page.goto("http://localhost:5199/");
      const shown = await page.getByText("2 busy").first().waitFor({ timeout: 8000 }).then(() => true, () => false);
      await page.waitForTimeout(500);
      const out = join(outDir, `projects-${skin || "core"}-${width}.png`);
      await page.screenshot({ path: out, fullPage: true });
      if (!shown) {
        failed = true;
        console.error(`projects did not render (${skin || "core"} ${width})`);
      }
      console.log(out);
      await context.close();
    }
  }
} finally {
  await browser.close();
  await server.close();
}
if (failed) process.exitCode = 1;
