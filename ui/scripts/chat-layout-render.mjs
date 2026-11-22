// Rendered evidence for the agent page's chat layout (INV §13, usability
// review J3/J4/J6): at 1280×800 the transcript keeps a readable height with and
// without the file viewer, a long unbroken token wraps inside its bubble, the
// viewer header shows absolute and `../` paths in their own order, and a
// terminal re-fits inside its card after the window shrinks. Every
// /api call and the event stream are stubbed; no Chuck server is needed.
//
//   node scripts/chat-layout-render.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), `chat-layout-render-${Date.now()}`)] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const PORT = 5231;
const BASE = `http://localhost:${PORT}`;
const AT = "2026-11-21T09:00:00Z";
const TOKEN = "x".repeat(400);
const event = (seq, type, data) => ({ agent_id: "a1", seq, type, ts: AT, data });
const TRANSCRIPT = [
  event(1, "user_text", { text: `Unbroken ${TOKEN}` }),
  event(2, "assistant_text", { text: `Reply ${TOKEN}` }),
  event(3, "turn_end", { stop_reason: "end_turn" }),
];
const AGENT = {
  agent_id: "a1", name: "Layout", role: "implementer", project: "demo", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: AT, running: true,
  state: "idle", detail: "", context_pct: 0, updated_at: 0, archived: false, history_revision: 0,
};
const TERMINAL = { ...AGENT, agent_id: "t1", name: "Shell", interface: "terminal" };
const FILES = { "/etc/hosts": "127.0.0.1 localhost\n", "../outside.txt": "beyond the root\n" };

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

function api(route) {
  const url = new URL(route.request().url());
  const path = url.pathname;
  if (path === "/api/config") return route.fulfill({ json: { onboarding_complete: true } });
  if (path === "/api/backends") return route.fulfill({ json: { backends: {} } });
  if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210] } } });
  if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 }, expanded: [] } });
  if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
  if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
  if (path === "/api/sessions/t1/transcript") return route.fulfill({ json: { agent_id: "t1", has_more: false, events: [] } });
  if (path === "/api/sessions/a1/transcript") return route.fulfill({ json: { agent_id: "a1", has_more: false, events: TRANSCRIPT } });
  if (path === "/api/sessions/a1/prompt") return route.fulfill({ json: { agent_id: "a1", text: "", after_seq: 3, queued: [] } });
  if (path === "/api/sessions/a1/available-commands") return route.fulfill({ json: { commands: [] } });
  if (path === "/api/sessions/a1/file") {
    const file = url.searchParams.get("path") ?? "";
    return route.fulfill({ json: { agent_id: "a1", path: file, content: FILES[file] ?? "", size: 10, mod_time: AT, line_count: 1 } });
  }
  return route.fulfill({ json: {} });
}

async function open(browser, path, width = 1280) {
  const context = await browser.newContext({ viewport: { width, height: 800 } });
  await context.addInitScript((hydrated) => {
    delete window.SharedWorker;
    // The PTY bridge never connects; the terminal still lays out.
    window.WebSocket = class extends EventTarget {
      static OPEN = 1;
      readyState = 0;
      send() {}
      close() {}
    };
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
  }, [{ agent_id: "a1", data: AGENT }, { agent_id: "t1", data: TERMINAL }, { agent_id: "__hydrated__", data: { hydrated: true } }]);
  await context.route((u) => u.pathname.startsWith("/api/"), api);
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.goto(`${BASE}${path}`);
  return { context, page, errors };
}

const transcriptHeight = (page) => page.locator(".transcript-view").first().evaluate((el) => Math.round(el.getBoundingClientRect().height));

const server = await createServer({ server: { port: PORT, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  const { context, page, errors } = await open(browser, "/agent/a1");
  await page.locator(".transcript-view").first().waitFor({ timeout: 10_000 });
  await page.getByText(/^Reply x+/).waitFor({ timeout: 10_000 });
  const plain = await transcriptHeight(page);
  // The short-window rule took this from 268px to 340px.
  record("1280×800: transcript is at least 330px tall", plain >= 330, `${plain}px`);
  const bubbles = await page.locator(".transcript-view").first().evaluate((scroller) => ({
    scroller: [scroller.scrollWidth, scroller.clientWidth],
    overflowing: [...scroller.querySelectorAll("*")].filter((el) => {
      const r = el.getBoundingClientRect();
      const s = scroller.getBoundingClientRect();
      return r.width > 0 && (r.right > s.right + 1 || el.scrollWidth > el.clientWidth + 1 && getComputedStyle(el).overflowX === "visible");
    }).map((el) => el.className || el.tagName).slice(0, 5),
  }));
  record("long token: transcript does not scroll sideways", bubbles.scroller[0] <= bubbles.scroller[1] + 1, JSON.stringify(bubbles.scroller));
  record("long token: nothing spills past the transcript", bubbles.overflowing.length === 0, JSON.stringify(bubbles.overflowing));
  await page.screenshot({ path: join(outDir, "agent-1280x800.png") });

  for (const [file, name] of [["/etc/hosts", "absolute"], ["../outside.txt", "parent"]]) {
    await page.goto(`${BASE}/agent/a1?file=${encodeURIComponent(file)}`);
    const header = page.locator(".file-viewer-path");
    await header.waitFor({ timeout: 10_000 });
    await page.getByText(FILES[file].trim()).waitFor({ timeout: 10_000 });
    // The visual order of the header's glyphs, left to right.
    const shown = await header.evaluate((el) => {
      const range = document.createRange();
      const text = el.textContent;
      const node = el.querySelector("bdi")?.firstChild ?? el.firstChild;
      const glyphs = [];
      for (let i = 0; i < text.length; i += 1) {
        range.setStart(node, i);
        range.setEnd(node, i + 1);
        glyphs.push([range.getBoundingClientRect().left, text[i]]);
      }
      return glyphs.sort((a, b) => a[0] - b[0]).map((g) => g[1]).join("");
    });
    record(`viewer header shows the ${name} path in order`, shown === file, shown);
    if (name === "absolute") {
      const withViewer = await transcriptHeight(page);
      record("1280×800 with viewer: transcript is at least 330px tall", withViewer >= 330, `${withViewer}px`);
      await page.screenshot({ path: join(outDir, "agent-viewer-1280x800.png") });
    }
  }
  record("no page errors", errors.length === 0, errors.join(" | "));
  await context.close();

  // J6: shrinking the window re-fits the terminal inside its card.
  const term = await open(browser, "/agent/t1", 1440);
  await term.page.locator(".xterm-screen").waitFor({ timeout: 10_000 });
  await term.page.waitForTimeout(300);
  await term.page.setViewportSize({ width: 1000, height: 800 });
  await term.page.waitForTimeout(500);
  const fitted = await term.page.locator(".terminal-panel").evaluate((panel) => {
    const p = panel.getBoundingClientRect();
    const s = panel.querySelector(".xterm-screen").getBoundingClientRect();
    const card = panel.closest(".chat-card").getBoundingClientRect();
    return { panel: [Math.round(p.left), Math.round(p.right)], screen: Math.round(s.right), card: Math.round(card.right) };
  });
  record("1000px after shrink: terminal stays inside its card", fitted.panel[1] <= fitted.card + 1 && fitted.screen <= fitted.panel[1] + 1, JSON.stringify(fitted));
  await term.page.screenshot({ path: join(outDir, "terminal-shrunk-1000.png") });
  record("terminal: no page errors", term.errors.length === 0, term.errors.join(" | "));
  await term.context.close();
} finally {
  await browser.close();
  await server.close();
}
writeFileSync(join(outDir, "report.json"), JSON.stringify({ checks }, null, 2));
const failed = checks.filter((check) => !check.pass);
console.log(`\n${checks.length - failed.length}/${checks.length} passed; output in ${outDir}`);
process.exit(failed.length ? 1 : 0);
