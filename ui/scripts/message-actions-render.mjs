// Rendered evidence for the desktop message action row (FS-01.R60/A42, R62/A44,
// TS-08.R171, INV §13): on the agent page at 1024px and 1440px and in the
// dashboard chat pane, in every appearance, the Copy/Fork icons stay hidden at
// rest, appear at the hovered or focused message's top right without covering
// its text or moving the transcript, show unavailable reasons by pointer and
// keyboard, and Copy writes the full Markdown source. Every /api call and the
// event stream are stubbed; no Chuck server is needed.
//
//   node scripts/message-actions-render.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), `message-actions-render-${Date.now()}`)] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const PORT = 5197;
const BASE = `http://localhost:${PORT}`;
const AT = "2026-11-20T09:00:00Z";
const LONG = `A deliberately long request ${"that keeps wrapping across the full reading measure ".repeat(6)}to the end.`;
const MARKDOWN = "## Plan\n\n- read the **config**\n- run `make test`\n\n```go\nfmt.Println(\"done\")\n```";
const event = (seq, type, data, extra = {}) => ({ agent_id: "a1", seq, type, ts: AT, data, ...extra });
const TRANSCRIPT = [
  event(1, "user_text", { text: "First question" }),
  event(2, "assistant_text", { text: "First answer." }),
  event(3, "turn_end", { stop_reason: "end_turn" }, { fork: { available: false, reason: "This turn was copied from another agent." } }),
  event(4, "user_text", { text: LONG }),
  event(5, "assistant_text", { text: MARKDOWN }),
  event(6, "turn_end", { stop_reason: "end_turn" }, { fork: { available: true } }),
];
const AGENT = {
  agent_id: "a1", name: "Message actions", role: "implementer", project: "demo", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: AT, running: true,
  state: "idle", detail: "", context_pct: 0, updated_at: 0, archived: false, clone: { available: true },
};

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

function api(skin) {
  return (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarding_complete: true } });
    if (path === "/api/backends") return route.fulfill({ json: { backends: {} } });
    if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210] } } });
    if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 }, expanded: ["a1"] } });
    if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
    if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
    if (path === "/api/sessions/a1/transcript") return route.fulfill({ json: { agent_id: "a1", has_more: false, events: TRANSCRIPT } });
    if (path === "/api/sessions/a1/prompt") return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "none" } } });
    if (path === "/api/sessions/a1/available-commands") return route.fulfill({ json: { commands: [] } });
    return route.fulfill({ json: {} });
  };
}

async function open(browser, skin, width, path) {
  const context = await browser.newContext({ viewport: { width, height: 900 } });
  await context.grantPermissions(["clipboard-read", "clipboard-write"], { origin: BASE });
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
  await context.route((u) => u.pathname.startsWith("/api/"), api(skin));
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.goto(`${BASE}${path}`);
  return { context, page, errors };
}

// Geometry of one message row, its content and its action row.
async function geometry(item) {
  return item.evaluate((el) => {
    const rect = (node) => { const r = node.getBoundingClientRect(); return { top: r.top, bottom: r.bottom, left: r.left, right: r.right, width: r.width, height: r.height }; };
    const row = el.querySelector(".message-actions");
    const content = el.querySelector("article");
    const scroller = el.closest(".transcript-view");
    return {
      item: rect(el), row: rect(row), content: rect(content), opacity: Number(getComputedStyle(row).opacity),
      scrollTop: scroller?.scrollTop ?? 0,
      buttons: [...row.querySelectorAll("button")].map((b) => ({ ...rect(b), border: getComputedStyle(b).borderTopWidth, background: getComputedStyle(b).backgroundColor })),
    };
  });
}

async function checkSurface(page, label, root) {
  for (const seq of [4, 5]) {
    const item = root.locator(`[data-seq="${seq}"]`);
    await item.waitFor({ timeout: 10_000 });
    await item.scrollIntoViewIfNeeded();
    await page.mouse.move(0, 0);
    const rest = await geometry(item);
    record(`${label} #${seq}: hidden at rest`, rest.opacity === 0, rest.opacity);
    await item.hover({ position: { x: 10, y: rest.item.height - 6 } });
    await page.waitForTimeout(200);
    const shown = await geometry(item);
    record(`${label} #${seq}: revealed on hover`, shown.opacity === 1, shown.opacity);
    record(`${label} #${seq}: top right of its message`, Math.abs(shown.row.right - shown.item.right) <= 1 && shown.row.top >= shown.item.top - 1, JSON.stringify({ row: shown.row, item: shown.item }));
    record(`${label} #${seq}: does not cover message text`, shown.row.bottom <= shown.content.top + 0.5, `${shown.row.bottom} <= ${shown.content.top}`);
    record(`${label} #${seq}: transcript does not move`, shown.item.top === rest.item.top && shown.item.height === rest.item.height && shown.scrollTop === rest.scrollTop, JSON.stringify({ rest: [rest.item.top, rest.item.height, rest.scrollTop], shown: [shown.item.top, shown.item.height, shown.scrollTop] }));
    record(`${label} #${seq}: borderless, transparent, >=24px targets`, shown.buttons.every((b) => b.border === "0px" && b.background === "rgba(0, 0, 0, 0)" && b.width >= 24 && b.height >= 24), JSON.stringify(shown.buttons));
    // Crossing from the message into its controls keeps them visible.
    const fork = item.getByRole("button", { name: "Fork from here" });
    await fork.hover();
    record(`${label} #${seq}: stays visible over its controls`, (await geometry(item)).opacity === 1);
  }
  // Unavailable reason by pointer, then by keyboard.
  const first = root.locator('[data-seq="2"]');
  const unavailable = first.getByRole("button", { name: "Fork from here" });
  await unavailable.hover();
  const tip = first.locator(".message-action-tip", { hasText: "unavailable" });
  record(`${label}: pointer shows the unavailable reason`, await tip.isVisible(), await tip.innerText().catch(() => ""));
  await page.mouse.move(0, 0);
  await first.getByRole("button", { name: "Copy" }).focus();
  await page.keyboard.press("Tab");
  await page.waitForTimeout(200);
  record(`${label}: keyboard focus reveals the row`, (await geometry(first)).opacity === 1);
  record(`${label}: keyboard focus shows the unavailable reason`, await tip.isVisible());
  await page.keyboard.press("Enter");
  record(`${label}: unavailable Fork opens no confirmation`, (await page.getByRole("dialog").count()) === 0);
  // Copy writes the complete Markdown source.
  const reply = root.locator('[data-seq="5"]');
  await reply.hover();
  await reply.getByRole("button", { name: "Copy" }).click();
  const clip = await page.evaluate(() => navigator.clipboard.readText());
  record(`${label}: Copy writes the full Markdown source`, clip === MARKDOWN, JSON.stringify(clip));
  record(`${label}: Copy shows copied feedback`, await reply.getByRole("button", { name: "Copied" }).isVisible());
}

const server = await createServer({ server: { port: PORT, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  for (const skin of ["", "sky-grove", "studio"]) {
    const name = skin || "core";
    for (const width of [1024, 1440]) {
      const { context, page, errors } = await open(browser, skin, width, "/agent/a1");
      const label = `${name} agent ${width}`;
      await checkSurface(page, label, page.locator(".transcript-view"));
      await page.locator('[data-seq="4"]').hover();
      await page.screenshot({ path: join(outDir, `agent-${name}-${width}.png`) });
      record(`${label}: no page errors`, errors.length === 0, errors.join(" | "));
      await context.close();
    }
    const { context, page, errors } = await open(browser, skin, 1440, "/project/demo");
    const label = `${name} dashboard pane`;
    const pane = page.locator('[data-slot="chat-pane"]').first();
    await pane.waitFor({ timeout: 10_000 });
    await checkSurface(page, label, pane);
    await pane.locator('[data-seq="5"]').hover();
    await page.screenshot({ path: join(outDir, `dashboard-${name}.png`) });
    record(`${label}: no page errors`, errors.length === 0, errors.join(" | "));
    await context.close();
  }
  // Reduced motion drops the reveal transition.
  const { context, page } = await open(browser, "", 1024, "/agent/a1");
  await page.emulateMedia({ reducedMotion: "reduce" });
  const duration = await page.locator('[data-seq="4"] .message-actions').evaluate((el) => getComputedStyle(el).transitionDuration);
  record("reduced motion: no reveal transition", duration === "0s", duration);
  await context.close();
} finally {
  await browser.close();
  await server.close();
}

writeFileSync(join(outDir, "report.json"), JSON.stringify({ checks }, null, 2));
const failed = checks.filter((check) => !check.pass);
console.log(`\n${checks.length - failed.length}/${checks.length} passed; output in ${outDir}`);
process.exit(failed.length ? 1 : 0);
