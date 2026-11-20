// Rendered evidence for the assignment-queued card at the transcript tail
// (FS-03.R48, FS-12.R24, INV §13): a long unbroken URL wraps inside the card and
// does not widen the transcript scroller, on the agent page at 1024px and 1440px
// and in the dashboard chat pane. Every /api call and the event stream are
// stubbed; no Chuck server is needed.
//
//   node scripts/queued-input-render.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), `queued-input-render-${Date.now()}`)] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const PORT = 5198;
const BASE = `http://localhost:${PORT}`;
const AT = "2026-11-20T09:00:00Z";
const URL_TEXT = `See https://example.com/${"a1b2c3d4e5".repeat(130)} before continuing.`;
const event = (seq, type, data) => ({ agent_id: "a1", seq, type, ts: AT, data });
const TRANSCRIPT = [
  event(1, "user_text", { text: "Start the assignment" }),
  event(2, "assistant_text", { text: "Waiting for the child task." }),
  event(3, "turn_end", { stop_reason: "end_turn" }),
];
const PROMPT = { agent_id: "a1", text: "", after_seq: 3, queued: [{ request_id: "q1", text: URL_TEXT, assignment_task_id: "t1", attachments: [] }] };
const AGENT = {
  agent_id: "a1", name: "Queued input", role: "implementer", project: "demo", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: AT, running: true,
  state: "idle", detail: "", context_pct: 0, updated_at: 0, archived: false, history_revision: 0,
};

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

function api(route) {
  const path = new URL(route.request().url()).pathname;
  if (path === "/api/config") return route.fulfill({ json: { onboarding_complete: true } });
  if (path === "/api/backends") return route.fulfill({ json: { backends: {} } });
  if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210] } } });
  if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 }, expanded: ["a1"] } });
  if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
  if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
  if (path === "/api/sessions/a1/transcript") return route.fulfill({ json: { agent_id: "a1", has_more: false, events: TRANSCRIPT } });
  if (path === "/api/sessions/a1/prompt") return route.fulfill({ json: PROMPT });
  if (path === "/api/sessions/a1/available-commands") return route.fulfill({ json: { commands: [] } });
  return route.fulfill({ json: {} });
}

async function open(browser, width, path) {
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
  }, [{ agent_id: "a1", data: AGENT }, { agent_id: "__hydrated__", data: { hydrated: true } }]);
  await context.route((u) => u.pathname.startsWith("/api/"), api);
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.goto(`${BASE}${path}`);
  return { context, page, errors };
}

async function checkCard(page, label, root) {
  const card = root.locator('[data-variant="queued"]');
  await card.waitFor({ timeout: 10_000 });
  const g = await card.evaluate((el) => {
    const scroller = el.closest(".transcript-view");
    const text = el.querySelector(".transcript-held-text");
    const c = el.getBoundingClientRect();
    const t = text.getBoundingClientRect();
    return { card: [c.left, c.right], text: [t.left, t.right, text.scrollWidth, text.clientWidth], scroller: [scroller.scrollWidth, scroller.clientWidth] };
  });
  record(`${label}: transcript does not scroll sideways`, g.scroller[0] <= g.scroller[1] + 1, JSON.stringify(g.scroller));
  record(`${label}: text stays inside the card`, g.text[0] >= g.card[0] - 0.5 && g.text[1] <= g.card[1] + 0.5 && g.text[2] <= g.text[3] + 1, JSON.stringify(g));
}

const server = await createServer({ server: { port: PORT, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  for (const width of [1024, 1440]) {
    const { context, page, errors } = await open(browser, width, "/agent/a1");
    await checkCard(page, `agent ${width}`, page.locator(".transcript-view"));
    await page.screenshot({ path: join(outDir, `agent-${width}.png`) });
    record(`agent ${width}: no page errors`, errors.length === 0, errors.join(" | "));
    await context.close();
  }
  const { context, page, errors } = await open(browser, 1440, "/project/demo");
  const pane = page.locator('[data-slot="chat-pane"]').first();
  await pane.waitFor({ timeout: 10_000 });
  await checkCard(page, "dashboard pane", pane);
  await page.screenshot({ path: join(outDir, "dashboard.png") });
  record("dashboard pane: no page errors", errors.length === 0, errors.join(" | "));
  await context.close();
} finally {
  await browser.close();
  await server.close();
}
writeFileSync(join(outDir, "report.json"), JSON.stringify({ checks }, null, 2));
const failed = checks.filter((check) => !check.pass);
console.log(`\n${checks.length - failed.length}/${checks.length} passed; output in ${outDir}`);
process.exit(failed.length ? 1 : 0);
