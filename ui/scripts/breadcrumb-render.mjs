// Rendered evidence for breadcrumb sibling switching (FS-03.R98/A76, FS-12.R70/A40,
// TS-08.R137–R138): the agent page with one, two, five and seven running siblings
// in every appearance at 1024px and 1440px, covering busy, idle, waiting, error
// and done states, a long agent name and a long project title. Every /api call and
// the event stream are stubbed; no Chuck server is needed.
//
//   node scripts/breadcrumb-render.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), `breadcrumb-render-${Date.now()}`)] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const at = "2026-10-18T09:00:00Z";
const projects = {
  app: { title: "Customer platform migration with a deliberately long project title", cwd: "/tmp/app", color: [80, 120, 200] },
};
const STATES = ["busy", "idle", "waiting_input", "error", "done"];
const NAMES = ["Orchestrator", "Reviewer", "Implementer with a remarkably long agent name", "Tester", "Docs", "Planner", "Fixer"];
const agent = (index, patch = {}) => ({
  agent_id: `a${index + 1}`, name: NAMES[index], role: "impl", project: "app", backend: "claude", model: "m", fast: false,
  interface: "chat", created_at: at, running: true, state: STATES[index % STATES.length], detail: "", context_pct: 0,
  updated_at: 0, archived: false, ...patch,
});
// The current page is a3 (long name, waiting) except in the seven case, where the
// current agent is the last and must replace the fifth visible slot.
const scenarios = [
  { name: "single", count: 1, current: "a1" },
  { name: "two", count: 2, current: "a2" },
  { name: "five", count: 5, current: "a3" },
  { name: "seven", count: 7, current: "a7" },
];

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

const server = await createServer({ server: { port: 5199, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  for (const skin of ["", "sky-grove", "studio"]) {
    for (const width of [1024, 1440]) {
      let baseline;
      for (const scenario of scenarios) {
        const agents = Array.from({ length: scenario.count }, (_, index) => agent(index));
        // A stopped sibling never appears (FS-03.R98).
        agents.push(agent(5, { agent_id: "stopped", name: "Stopped sibling", running: false, state: "idle" }));
        const label = `${skin || "core"} ${width} ${scenario.name}`;
        const context = await browser.newContext({ viewport: { width, height: 900 } });
        await context.addInitScript((hydrated) => {
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
          const transcript = path.match(/^\/api\/sessions\/([^/]+)\/transcript$/);
          if (transcript) return route.fulfill({ json: { agent_id: transcript[1], events: [], has_more: false, pending_permission: null, latest_assistant: "" } });
          const prompt = path.match(/^\/api\/sessions\/([^/]+)\/prompt$/);
          if (prompt) return route.fulfill({ json: { agent_id: prompt[1], text: "", after_seq: 0 } });
          if (path === "/api/projects") return route.fulfill({ json: projects });
          if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarded: true, onboarding_complete: true, onboarding: { satisfied: true, steps: [] } } });
          if (path === "/api/backends") return route.fulfill({ json: { version: 2, backends: {} } });
          if (path === "/api/config-sources") return route.fulfill({ json: [] });
          return route.fulfill({ json: {} });
        });
        const page = await context.newPage();
        const errors = [];
        page.on("pageerror", (error) => errors.push(error.message));
        await page.goto(`http://localhost:5199/agent/${scenario.current}`);
        await page.locator(".chat-card").waitFor({ timeout: 8000 });
        if (scenario.count > 1) await page.locator(".agent-mini-card").first().waitFor({ timeout: 8000 });
        await page.evaluate(() => document.fonts.ready);
        await page.waitForTimeout(300);

        const m = await page.evaluate(() => {
          const rect = (element) => element?.getBoundingClientRect().toJSON();
          const nav = document.querySelector(".chat-breadcrumb");
          const current = document.querySelector(".agent-mini-card[aria-current='page']");
          const waiting = document.querySelector(".agent-mini-card.waiting_input:not([aria-current])");
          return {
            nav: rect(nav), navScroll: nav.scrollWidth, navClient: nav.clientWidth,
            back: rect(nav.querySelector(":scope > a")), id: rect(nav.querySelector(".chat-breadcrumb-id")),
            siblings: rect(nav.querySelector(".chat-breadcrumb-siblings")),
            card: rect(document.querySelector(".chat-card")),
            entries: [...nav.querySelectorAll(".chat-breadcrumb-siblings > .agent-mini-card")].map((entry) => entry.textContent),
            overflow: nav.querySelector(".chat-breadcrumb-overflow > button")?.textContent,
            stopped: nav.textContent.includes("Stopped sibling"),
            currentBorder: current && getComputedStyle(current).borderTopColor,
            currentWeight: current && getComputedStyle(current).fontWeight,
            currentIsLink: current?.tagName === "A",
            waitingBorder: waiting && getComputedStyle(waiting).borderTopColor,
            busyAnimation: (() => { const dot = nav.querySelector(".agent-mini-card.busy .agent-monogram-dot"); return dot && getComputedStyle(dot).animationName; })(),
          };
        });

        if (scenario.name === "single") {
          baseline = m;
          record(`${label}: unchanged path breadcrumb`, m.entries.length === 0 && !m.siblings);
        } else {
          const expected = Math.min(scenario.count, 5);
          record(`${label}: ${expected} entries`, m.entries.length === expected, m.entries.join(" | "));
          record(`${label}: stopped sibling absent`, !m.stopped);
          record(`${label}: current entry shown, not a link, non-color marker`, m.currentBorder && !m.currentIsLink && Number(m.currentWeight) >= 600, `${m.currentBorder} ${m.currentWeight}`);
          if (scenario.count > 5) {
            record(`${label}: +n overflow`, m.overflow === `+${scenario.count - 5}`, m.overflow);
            record(`${label}: current replaces the fifth`, m.entries.at(-1)?.includes(NAMES[6]), m.entries.at(-1));
          }
          if (m.waitingBorder) record(`${label}: waiting entry carries a tone`, m.waitingBorder !== "rgba(0, 0, 0, 0)", m.waitingBorder);
          if (m.busyAnimation !== null) record(`${label}: busy dot pulses`, m.busyAnimation === "ad-state-pulse", m.busyAnimation);
          record(`${label}: one line`, Math.abs(m.nav.height - baseline.nav.height) <= 1, `${m.nav.height} vs ${baseline.nav.height}`);
          record(`${label}: panel does not move`, Math.abs(m.card.top - baseline.card.top) <= 1, `${m.card.top} vs ${baseline.card.top}`);
        }
        record(`${label}: no breadcrumb overflow`, m.navScroll <= m.navClient + 1, `${m.navScroll}/${m.navClient}`);
        record(`${label}: Back and id visible`, m.back.left >= m.nav.left - 1 && m.back.width > 0 && m.id.right <= m.nav.right + 1
          && m.id.width > 0 && (!m.siblings || (m.siblings.left >= m.back.right && m.siblings.right <= m.id.left)), JSON.stringify({ back: m.back.right, id: m.id.left, siblings: m.siblings && [m.siblings.left, m.siblings.right] }));
        record(`${label}: no page errors`, errors.length === 0, errors.join("; "));

        const base = `breadcrumb-${skin || "core"}-${width}-${scenario.name}`;
        await page.locator(".chat-breadcrumb").screenshot({ path: join(outDir, `${base}-crumb.png`), animations: "disabled" });
        if (scenario.count > 5) {
          await page.locator(".chat-breadcrumb-overflow > button").click();
          await page.locator(".chat-breadcrumb-menu").waitFor();
          await page.screenshot({ path: join(outDir, `${base}-menu.png`), clip: { x: 0, y: 0, width, height: 320 }, animations: "disabled" });
          await page.keyboard.press("Escape");
          record(`${label}: Escape closes the menu`, await page.locator(".chat-breadcrumb-menu").count() === 0);
        }
        await page.screenshot({ path: join(outDir, `${base}.png`), animations: "disabled" });
        await context.close();
      }
    }
  }
} finally {
  writeFileSync(join(outDir, "report.json"), JSON.stringify(checks, null, 2));
  await browser.close();
  await server.close();
}
console.log(outDir);
if (checks.some((check) => !check.pass)) process.exitCode = 1;
