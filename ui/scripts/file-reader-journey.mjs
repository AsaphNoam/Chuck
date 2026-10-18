// Rendered evidence for the roomier file reader (FS-03.A71–A75, TS-08.R127–R132):
// contained Markdown, both workspace dividers by pointer and keyboard, reset,
// tray collapse/overlay, window narrowing, Focus file with the shared tray,
// nested menus, focus return, read counts and reading continuity. Drives Vite +
// the real UI source in Chromium with every /api call and the event stream
// stubbed, so no Chuck server is needed (same pattern as chat-cleanup-render).
//
//   node scripts/file-reader-journey.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const OUT_DIR = process.argv[2] || join(tmpdir(), `file-reader-journey-${Date.now()}`);
mkdirSync(OUT_DIR, { recursive: true });

const PORT = 5199;
const BASE = `http://localhost:${PORT}`;
const AT = "2026-10-17T09:00:00Z";
const SKINS = ["", "sky-grove", "studio"];
const skinLabel = (skin) => skin || "core";
const CJK = /[㐀-鿿豈-﫿]/;

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

// ---- Fixture ------------------------------------------------------------

const LONG_PATH = `internal/server/${"deeply-nested-directory/".repeat(6)}handler_with_a_long_name.go`;
const LONG_URL = `https://example.com/${"segment".repeat(30)}`;
const FILE_PATH = "docs/reader-fixture-with-a-deliberately-long-file-name-to-test-header-fit.md";
const OTHER_PATH = "docs/other.md";
const paragraphs = Array.from({ length: 14 }, (_, i) => `## Section ${i + 1}\n\nParagraph ${i + 1} explains \`${LONG_PATH}\` and links ${LONG_URL} while continuing with ordinary prose so the reading measure matters.\n\n- Item one with \`inline-code-that-is-quite-long-${"x".repeat(60)}\`\n  - Nested item\n- Item two\n\n> A quoted note for section ${i + 1}.`).join("\n\n");
const FILE_MD = [
  "# Reader fixture",
  "",
  "Original text 原文 stays as written.",
  "",
  "| Column | Value | Another wide column | Link |",
  "|---|---|---|---|",
  `| alpha | ${"wide-value-".repeat(8)} | ${"more-".repeat(10)} | ${LONG_URL} |`,
  "",
  "```go",
  `func Handler() { log.Printf("${"a very long log line ".repeat(10)}") }`,
  "```",
  "",
  "```mermaid",
  "graph LR",
  "  A[Start] --> B[Middle] --> C[End] --> D[Done] --> E[Archive] --> F[Report]",
  "```",
  "",
  `See [other](other.md).`,
  "",
  paragraphs,
].join("\n");
const OTHER_MD = "# Other file\n\nA different document.\n";

const event = (agentId, seq, type, data) => ({ agent_id: agentId, seq, type, ts: AT, data });
const transcriptFor = (agentId) => [
  event(agentId, 0, "session_meta", { name: "Reader demo", project: "demo", backend: "claude", model: "m", role: "implementer" }),
  ...Array.from({ length: 24 }, (_, i) => event(agentId, i + 1, i % 2 ? "assistant_text" : "user_text", {
    text: i === 1 ? `Open [the fixture](${FILE_PATH}) to read it.` : `Message ${i + 1}: ${"conversation content ".repeat(12)}`,
  })),
];

function makeAgent(id, overrides = {}) {
  return {
    agent_id: id, name: "Reader demo", role: "implementer", project: "demo", backend: "claude", model: "m",
    fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: AT,
    running: true, state: "idle", detail: "", context_pct: 0, updated_at: 0, archived: false,
    ...overrides,
  };
}

async function hydrate(context, agents) {
  await context.addInitScript((hydrated) => {
    delete window.SharedWorker;
    window.EventSource = class extends EventTarget {
      constructor() {
        super();
        window.__stream = this;
        setTimeout(() => {
          this.onopen?.();
          for (const data of hydrated) this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify(data) }));
        });
      }
      close() {}
    };
  }, [...agents.map((data) => ({ agent_id: data.agent_id, data })), { agent_id: "__hydrated__", data: { hydrated: true } }]);
}

function apiHandler(skin, reads) {
  return (route) => {
    const req = route.request();
    const url = new URL(req.url());
    const path = url.pathname;
    if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarding_complete: true } });
    if (path === "/api/backends") return route.fulfill({ json: { backends: {} } });
    if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210] } } });
    if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 2, gap: 16 }, expanded: [] } });
    if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
    if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
    const transcript = path.match(/^\/api\/sessions\/(a[12])\/transcript$/);
    if (transcript) return route.fulfill({ json: { agent_id: transcript[1], has_more: false, events: transcriptFor(transcript[1]) } });
    const file = path.match(/^\/api\/sessions\/(a[12])\/file$/);
    if (file) {
      const requested = url.searchParams.get("path") || "";
      reads.push(requested);
      if (requested.endsWith("refused.md")) return route.fulfill({ status: 422, json: { error: { code: "path_refused", message: "that path is outside this agent's working directory" } } });
      const content = requested.endsWith("other.md") ? OTHER_MD : FILE_MD;
      if (requested.endsWith("short.go")) return route.fulfill({ json: { agent_id: file[1], path: requested, size: 12, mod_time: AT, line_count: 1, content: "func Read() {}", truncated: false, language: "go" } });
      if (requested.endsWith("data.json")) return route.fulfill({ json: { agent_id: file[1], path: requested, size: 40, mod_time: AT, line_count: 1, content: '{"name":"Chuck","items":[1,2,3]}', truncated: false, language: "json" } });
      return route.fulfill({ json: {
        agent_id: file[1], path: requested, size: content.length, mod_time: AT, line_count: content.split("\n").length,
        content, truncated: requested.endsWith("partial.md"), language: "markdown",
      } });
    }
    if (/\/prompt$/.test(path)) return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "none" } } });
    if (/\/available-commands$/.test(path)) return route.fulfill({ json: { commands: [] } });
    const tracked = path.match(/^\/api\/sessions\/(a[12])\/files$/);
    if (tracked) return route.fulfill({ json: { agent_id: tracked[1], files: [] } });
    return route.fulfill({ json: {} });
  };
}

// ---- Measurements -------------------------------------------------------

const rects = (page) => page.evaluate(() => {
  const box = (selector) => {
    const el = document.querySelector(selector);
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.x, y: r.y, width: r.width, height: r.height, right: r.right, bottom: r.bottom };
  };
  return {
    stage: box(".chat-stage"),
    file: box(".chat-side-start .file-viewer"),
    card: box(".chat-card"),
    tray: box(".annotation-tray"),
    tray_mode: document.querySelector(".chat-stage")?.getAttribute("data-tray") ?? null,
  };
});

const pageOverflow = (page) => page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);

async function documentMetrics(page, scope) {
  return page.evaluate((root) => {
    const viewer = document.querySelector(`${root} .file-viewer`);
    const body = viewer.querySelector(".file-viewer-body");
    const rendered = viewer.querySelector(".file-viewer-rendered");
    const table = viewer.querySelector(".markdown-table");
    const fence = viewer.querySelector('.file-viewer-rendered pre');
    const header = viewer.querySelector(".file-viewer-header").getBoundingClientRect();
    const actions = [...viewer.querySelectorAll(".file-viewer-actions button")].map((button) => {
      const r = button.getBoundingClientRect();
      const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
      return { label: button.textContent, inside: r.x >= header.x - 1 && r.right <= header.right + 1, hit: button === hit || button.contains(hit) };
    });
    const chatMessage = document.querySelector(".chat-panel .message");
    return {
      bodyOverflow: body.scrollWidth - body.clientWidth,
      renderedFont: rendered ? getComputedStyle(rendered).fontSize : null,
      chatFont: chatMessage ? getComputedStyle(chatMessage).fontSize : null,
      tableLocal: table ? table.scrollWidth > table.clientWidth && ["auto", "scroll"].includes(getComputedStyle(table).overflowX) : null,
      fenceLocal: fence ? [fence, ...fence.querySelectorAll('div[class*="language-"]')].some((el) => el.scrollWidth > el.clientWidth && ["auto", "scroll"].includes(getComputedStyle(el).overflowX)) : null,
      fenceDetail: fence ? [fence, ...fence.querySelectorAll('div[class*="language-"], code')].map((el) => `${el.tagName}:${el.scrollWidth}/${el.clientWidth}/${getComputedStyle(el).overflowX}/${getComputedStyle(el).whiteSpace}`).join(" ") : "none",
      actions,
      cjkPreserved: rendered ? rendered.textContent.includes("原文") : null,
    };
  }, scope);
}

// The first visible top-level rendered block and its offset in the body.
const readingAnchor = (page, scope) => page.evaluate((root) => {
  const body = document.querySelector(`${root} .file-viewer-body`);
  const top = body.getBoundingClientRect().top;
  const blocks = [...body.querySelector(".file-viewer-rendered").children];
  const index = blocks.findIndex((el) => el.getBoundingClientRect().bottom > top);
  return { index, text: blocks[index]?.textContent.slice(0, 40), offset: Math.round(blocks[index].getBoundingClientRect().top - top), scrollTop: body.scrollTop };
}, scope);

const chromeText = (page, selector) => page.evaluate((sel) => {
  const parts = [];
  for (const el of document.querySelectorAll(sel)) {
    for (const attr of ["aria-label", "title", "aria-valuetext"]) if (el.getAttribute(attr)) parts.push(el.getAttribute(attr));
    if (el.matches("button")) parts.push(el.textContent);
  }
  return parts;
}, selector);

// ---- Journey ------------------------------------------------------------

async function run(browser, skin, width, archived) {
  const label = `${skinLabel(skin)} ${archived ? "archive" : "live"}@${width}`;
  const id = archived ? "a2" : "a1";
  const reads = [];
  const errors = [];
  const context = await browser.newContext({ viewport: { width, height: 900 } });
  await hydrate(context, [makeAgent(id, archived ? { running: false, archived: true } : {})]);
  await context.route((u) => u.pathname.startsWith("/api/"), apiHandler(skin, reads));
  await context.route((u) => u.hostname === "example.com", (route) => route.fulfill({ contentType: "text/html", body: "<title>x</title>" }));
  const page = await context.newPage();
  page.on("pageerror", (error) => errors.push(error.message));
  if (process.env.DEBUG_CONSOLE) page.on("console", (message) => { if (message.type() === "error") console.log("console:", message.text().slice(0, 1500)); });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-${archived ? "archive" : "live"}-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/${archived ? "archive" : "agent"}/${id}?file=${encodeURIComponent(FILE_PATH)}`);
    await page.locator(".file-viewer-rendered").waitFor();
    await page.locator(".mermaid-diagram-figure svg").first().waitFor({ timeout: 10_000 }).catch(() => {});
    await page.evaluate(() => document.fonts.ready);

    // A71 — contained, legible document.
    const doc = await documentMetrics(page, ".chat-side-start");
    record(`${label}: no document horizontal overflow`, doc.bodyOverflow <= 0, doc.bodyOverflow);
    record(`${label}: no page overflow`, !(await pageOverflow(page)));
    record(`${label}: rendered text at chat size`, doc.renderedFont === doc.chatFont, `${doc.renderedFont} vs ${doc.chatFont}`);
    record(`${label}: wide table scrolls locally`, doc.tableLocal);
    record(`${label}: long fence scrolls locally`, doc.fenceLocal, doc.fenceDetail);
    record(`${label}: header actions reachable`, doc.actions.every((a) => a.inside && a.hit), JSON.stringify(doc.actions.filter((a) => !(a.inside && a.hit))));
    record(`${label}: file Unicode preserved`, doc.cjkPreserved);
    const fullPath = await page.locator(".file-viewer-path").getAttribute("title");
    record(`${label}: full path available`, fullPath === FILE_PATH);

    // A72 — initial split gives the file about half.
    let r = await rects(page);
    const half = Math.abs(r.file.width - r.card.width) <= 4;
    record(`${label}: file opens at about half`, half, `${r.file.width} vs ${r.card.width}`);
    await shot("open");

    // Keyboard resize, limits and reset on the file divider.
    const fileDivider = page.getByRole("separator", { name: "Resize file and conversation" });
    await fileDivider.focus();
    const before = r.file.width;
    await page.keyboard.press("ArrowRight");
    await page.keyboard.press("ArrowRight");
    r = await rects(page);
    record(`${label}: Right grows the file`, Math.round(r.file.width - before) === 32, r.file.width - before);
    await page.keyboard.press("Shift+ArrowLeft");
    r = await rects(page);
    record(`${label}: Shift+Left shrinks further`, Math.round(before + 32 - r.file.width) === 64, before + 32 - r.file.width);
    for (let i = 0; i < 60; i += 1) await page.keyboard.press("Shift+ArrowLeft");
    r = await rects(page);
    const min = Number(await fileDivider.getAttribute("aria-valuemin"));
    record(`${label}: file minimum enforced`, Math.round(r.file.width) >= Math.min(320, min) && Math.round(r.file.width) === Number(await fileDivider.getAttribute("aria-valuenow")), r.file.width);
    record(`${label}: focus stays on divider at limit`, await fileDivider.evaluate((el) => el === document.activeElement));
    for (let i = 0; i < 80; i += 1) await page.keyboard.press("Shift+ArrowRight");
    r = await rects(page);
    record(`${label}: conversation minimum enforced`, r.card.width >= 359, r.card.width);
    await page.keyboard.press("Home");
    r = await rects(page);
    record(`${label}: Home resets`, Math.abs(r.file.width - before) <= 1, `${r.file.width} vs ${before}`);

    // Pointer drag and double-click reset.
    const box = await fileDivider.boundingBox();
    await page.mouse.move(box.x + box.width / 2, box.y + box.height / 2);
    await page.mouse.down();
    await page.mouse.move(box.x + box.width / 2 + 60, box.y + box.height / 2, { steps: 4 });
    await page.mouse.up();
    r = await rects(page);
    record(`${label}: pointer drag resizes`, Math.abs(r.file.width - (before + 60)) <= 2, r.file.width - before);
    const dragged = r.file.width;
    const share = r.file.width / (r.file.width + r.card.width);

    // Create a transcript annotation; the tray docks or overlays.
    await page.locator(".transcript-item .message").first().click({ button: "right" });
    await page.getByRole("button", { name: "Annotate whole event" }).click();
    await page.locator(".annotation-tray").waitFor();
    r = await rects(page);
    const overlap = r.tray && r.file && !(r.tray.x >= r.file.right || r.tray.right <= r.file.x);
    record(`${label}: tray ${r.tray_mode} leaves file unobscured`, !overlap && ["docked", "overlay"].includes(r.tray_mode), r.tray_mode);
    record(`${label}: no page overflow with tray`, !(await pageOverflow(page)));
    const keptShare = r.file.width / (r.file.width + r.card.width);
    record(`${label}: split share kept beside tray`, Math.abs(keptShare - share) < 0.02, `${keptShare.toFixed(3)} vs ${share.toFixed(3)}`);
    await shot("tray");
    if (r.tray_mode === "docked") {
      const trayDivider = page.getByRole("separator", { name: "Resize annotations and conversation" });
      await trayDivider.focus();
      const trayBefore = r.tray.width;
      await page.keyboard.press("ArrowLeft");
      r = await rects(page);
      record(`${label}: Left grows the docked tray`, Math.round(r.tray.width - trayBefore) === 16, r.tray.width - trayBefore);
      await trayDivider.dblclick();
      r = await rects(page);
      record(`${label}: double-click resets tray`, Math.abs(r.tray.width - trayBefore) <= 1);
      await page.getByRole("button", { name: "Collapse pending annotations" }).click();
      r = await rects(page);
      const collapsedShare = r.file.width / (r.file.width + r.card.width);
      record(`${label}: collapse returns space, keeps split`, r.tray_mode === "collapsed" && Math.abs(collapsedShare - share) < 0.02, `${r.tray_mode} ${collapsedShare.toFixed(3)}`);
      await page.getByRole("button", { name: "Expand pending annotations" }).click();
    }

    // Window narrowing clamps; widening restores the requested split.
    await page.setViewportSize({ width: 1024, height: 900 });
    await page.waitForTimeout(100);
    await shot("narrowed");
    record(`${label}: no page overflow at 1024`, !(await pageOverflow(page)));
    await page.setViewportSize({ width, height: 900 });
    await page.waitForTimeout(100);
    r = await rects(page);
    const restoredShare = r.file.width / (r.file.width + r.card.width);
    record(`${label}: widening restores requested split`, Math.abs(restoredShare - share) < 0.02, `${restoredShare.toFixed(3)} vs ${share.toFixed(3)}`);

    // A73/A74 — Focus file keeps the read, place, draft, tray and transcript.
    await page.locator(".file-viewer-body").evaluate((el) => { el.scrollTop = el.scrollHeight / 3; });
    await page.waitForTimeout(50);
    const anchorBefore = await readingAnchor(page, ".chat-side-start");
    await page.locator(".transcript-view").evaluate((el) => { el.scrollTop = 200; });
    const transcriptTop = await page.locator(".transcript-view").evaluate((el) => el.scrollTop);
    if (!archived) await page.locator(".composer textarea").fill("Draft that must survive");
    await page.getByRole("button", { name: "Annotate whole event" }).waitFor({ state: "detached" }).catch(() => {});
    await page.locator(".annotation-draft textarea").first().fill("Instruction typed beside the file");
    const readsBefore = reads.length;
    await page.getByRole("button", { name: "Focus file" }).click();
    const dialog = page.getByRole("dialog", { name: "Focus file" });
    await dialog.waitFor();
    await page.waitForTimeout(100);
    const anchorFocused = await readingAnchor(page, ".file-focus");
    record(`${label}: focus keeps reading location`, Math.abs(anchorFocused.index - anchorBefore.index) <= 1,`${anchorBefore.index}→${anchorFocused.index}`);
    record(`${label}: focus adds no read`, reads.length === readsBefore, reads.length - readsBefore);
    record(`${label}: focus has the same tray`, await dialog.locator(".annotation-tray").count() === 1 && await page.locator(".annotation-tray").count() === 1);
    record(`${label}: tray instruction kept`, await dialog.locator(".annotation-draft textarea").first().inputValue() === "Instruction typed beside the file");
    const focusDoc = await documentMetrics(page, ".file-focus");
    record(`${label}: focus document contained`, focusDoc.bodyOverflow <= 0 && focusDoc.actions.every((a) => a.inside && a.hit), focusDoc.bodyOverflow);
    const measure = await page.evaluate(() => {
      const p = document.querySelector(".file-focus .file-viewer-rendered > p");
      const probe = document.createElement("span");
      probe.style.cssText = "display:inline-block;width:76ch";
      p.append(probe);
      const cap = probe.getBoundingClientRect().width;
      probe.remove();
      const box = p.getBoundingClientRect();
      const parent = p.parentElement.getBoundingClientRect();
      return { width: Math.round(box.width), cap: Math.round(cap), centered: Math.abs((box.x - parent.x) - (parent.right - box.right)) <= 2 || box.width >= parent.width - 1 };
    });
    record(`${label}: focus prose measure capped`, measure.width <= measure.cap + 1 && measure.centered, JSON.stringify(measure));
    await shot("focus");

    // Edit an annotation inside focus.
    await dialog.locator(".annotation-draft textarea").first().fill("Edited in focus");
    // Inner menu consumes Escape before the modal. The selection is a block
    // already in view, right-clicked in place so nothing scrolls.
    const visibleBlock = await page.evaluate(() => {
      const body = document.querySelector(".file-focus .file-viewer-body");
      const top = body.getBoundingClientRect().top;
      const block = [...body.querySelector(".file-viewer-rendered").children].find((el) => el.getBoundingClientRect().top >= top && el.textContent.trim());
      const range = document.createRange();
      range.selectNodeContents(block);
      getSelection().removeAllRanges();
      getSelection().addRange(range);
      const r = block.getBoundingClientRect();
      return { x: r.x + 20, y: r.y + r.height / 2 };
    });
    await page.mouse.click(visibleBlock.x, visibleBlock.y, { button: "right" });
    const menu = page.locator(".file-focus .context-menu");
    record(`${label}: file menu mounts inside focus`, await menu.count() === 1);
    await page.keyboard.press("Escape");
    record(`${label}: Escape closes the menu first`, await menu.count() === 0 && await dialog.count() === 1);
    // Annotate a selection in focus through the menu.
    await page.mouse.click(visibleBlock.x, visibleBlock.y, { button: "right" });
    await page.locator(".file-focus .context-menu").getByRole("button", { name: "Annotate selection" }).click();
    record(`${label}: annotate from focus adds to the same tray`, await dialog.locator(".annotation-draft").count() === 2);

    // Live events arrive underneath.
    if (!archived) {
      await page.evaluate(() => {
        window.__stream.dispatchEvent(new MessageEvent("new_message", { data: JSON.stringify({ agent_id: "a1", data: { agent_id: "a1", seq: 25, type: "assistant_text", ts: "2026-10-17T09:01:00Z", data: { text: "Arrived under focus" } } }) }));
      });
    }

    // Escape returns, to the Focus file control, with context intact.
    await page.keyboard.press("Escape");
    await dialog.waitFor({ state: "detached" });
    await page.waitForTimeout(100);
    record(`${label}: Escape returns focus to Focus file`, await page.evaluate(() => document.activeElement?.textContent === "Focus file"));
    const anchorBack = await readingAnchor(page, ".chat-side-start");
    record(`${label}: return keeps reading location`, Math.abs(anchorBack.index - anchorBefore.index) <= 1,`${anchorBefore.index}→${anchorBack.index}`);
    record(`${label}: return keeps transcript position`, Math.abs(await page.locator(".transcript-view").evaluate((el) => el.scrollTop) - transcriptTop) <= 2);
    if (!archived) {
      record(`${label}: composer draft kept`, await page.locator(".composer textarea").inputValue() === "Draft that must survive");
      record(`${label}: event arrived underneath`, await page.getByText("Arrived under focus").count() === 1);
    }
    record(`${label}: edited instruction kept`, await page.locator(".annotation-draft textarea").first().inputValue() === "Edited in focus");
    record(`${label}: no reads from layout`, reads.length === readsBefore, reads.length - readsBefore);
    r = await rects(page);
    const returnShare = r.file.width / (r.file.width + r.card.width);
    record(`${label}: return keeps chosen split`, Math.abs(returnShare - share) < 0.03 || r.tray_mode === "overlay", returnShare.toFixed(3));

    // Mode positions: Source keeps its own place; Rendered comes back to its own.
    await page.locator(".chat-side-start").getByRole("button", { name: "Source", exact: true }).click();
    await page.locator(".chat-side-start [data-file-line]").first().waitFor();
    await page.locator(".file-viewer-body").evaluate((el) => { el.scrollTop = 600; });
    await page.waitForTimeout(50);
    await page.locator(".chat-side-start").getByRole("button", { name: "Rendered", exact: true }).click();
    await page.waitForTimeout(50);
    const anchorRendered = await readingAnchor(page, ".chat-side-start");
    record(`${label}: Rendered keeps its own position`, Math.abs(anchorRendered.index - anchorBack.index) <= 1,`${anchorBefore.index}→${anchorRendered.index}`);
    await page.locator(".chat-side-start").getByRole("button", { name: "Source", exact: true }).click();
    await page.waitForTimeout(50);
    const sourceTop = await page.locator(".file-viewer-body").evaluate((el) => el.scrollTop);
    record(`${label}: Source keeps its own position`, Math.abs(sourceTop - 600) <= 24, sourceTop);
    await page.locator(".chat-side-start").getByRole("button", { name: "Rendered", exact: true }).click();

    // Backdrop dismissal, then replace the file inside focus and close it there.
    await page.getByRole("button", { name: "Focus file" }).click();
    await dialog.waitFor();
    // Keyboard trap: Tab cycles inside the dialog.
    for (let i = 0; i < 25; i += 1) await page.keyboard.press("Tab");
    record(`${label}: focus trapped in modal`, await page.evaluate(() => !!document.activeElement?.closest(".file-focus")));
    await page.mouse.click(4, 4);
    await dialog.waitFor({ state: "detached" });
    record(`${label}: backdrop click returns`, await page.locator(".chat-side-start .file-viewer").count() === 1);
    await page.getByRole("button", { name: "Focus file" }).click();
    await dialog.waitFor();
    await dialog.getByRole("button", { name: "other" }).click();
    await dialog.getByRole("heading", { name: "Other file" }).waitFor();
    record(`${label}: replacing file stays in focus`, await dialog.count() === 1 && reads.at(-1) === OTHER_PATH);
    await dialog.getByRole("button", { name: "Reload" }).click();
    await page.waitForTimeout(100);
    record(`${label}: Reload reads once`, reads.filter((p) => p === OTHER_PATH).length === 2);
    const english = await chromeText(page, ".file-focus .file-viewer-header button, .file-focus .file-focus-tray-toggle, .workspace-divider");
    record(`${label}: reader chrome is English`, english.length > 0 && english.every((text) => !CJK.test(text)), english.join(" | "));
    await dialog.getByRole("button", { name: "Close file" }).click();
    await dialog.waitFor({ state: "detached" });
    await page.waitForTimeout(100);
    record(`${label}: Close file closes the reader`, await page.locator(".file-viewer").count() === 0 && !new URL(page.url()).searchParams.get("file"));
    record(`${label}: focus returns to conversation`, await page.evaluate(() => !!document.activeElement?.closest(".transcript-view")));

    // Reload restores default sizing with the same URL-addressed file.
    await page.goto(`${BASE}/${archived ? "archive" : "agent"}/${id}?file=${encodeURIComponent(FILE_PATH)}`);
    await page.locator(".file-viewer-rendered").waitFor();
    r = await rects(page);
    record(`${label}: reload restores default sizing`, Math.abs(r.file.width - dragged) > 20 || r.tray_mode === "overlay", `${r.file.width} vs dragged ${dragged}`);
    const stored = await page.evaluate(() => Object.keys(localStorage).filter((key) => !["chuck-annotation-tray"].includes(key)));
    record(`${label}: no new browser storage`, stored.every((key) => !/reader|split|focus|file/i.test(key)), stored.join(","));

    // Refusal and partial states remain readable in focus.
    await page.goto(`${BASE}/${archived ? "archive" : "agent"}/${id}?file=docs%2Frefused.md`);
    await page.getByRole("alert").filter({ hasText: "outside this agent" }).waitFor();
    await page.getByRole("button", { name: "Focus file" }).click();
    record(`${label}: refusal readable in focus`, await page.locator(".file-focus .file-viewer-error").count() === 1);
    await page.keyboard.press("Escape");
    await page.goto(`${BASE}/${archived ? "archive" : "agent"}/${id}?file=docs%2Fpartial.md`);
    await page.getByText(/larger than the viewer reads/).waitFor();
    await page.getByRole("button", { name: "Focus file" }).click();
    record(`${label}: partial notice in focus`, await page.locator(".file-focus .file-viewer-meta").filter({ hasText: "larger than the viewer reads" }).count() === 1);
    await page.keyboard.press("Escape");

    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) {
    record(`${label}: journey completed`, false, `${error.message.split("\n")[0]} ${errors.join(" | ")}`);
    await shot("failure").catch(() => {});
  } finally {
    await context.close();
  }
}

// New task launch from the tray inside focus: the nested launch dialog takes
// Escape first and the focus modal stays open.
async function runNestedLaunch(browser) {
  const label = "core nested launch";
  const reads = [];
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await hydrate(context, [makeAgent("a1")]);
  await context.route((u) => u.pathname.startsWith("/api/"), apiHandler("", reads));
  const page = await context.newPage();
  try {
    await page.goto(`${BASE}/agent/a1?file=${encodeURIComponent(FILE_PATH)}`);
    await page.locator(".file-viewer-rendered").waitFor();
    await page.locator(".transcript-item .message").first().click({ button: "right" });
    await page.getByRole("button", { name: "Annotate whole event" }).click();
    await page.getByRole("button", { name: "Focus file" }).click();
    const dialog = page.getByRole("dialog", { name: "Focus file" });
    await dialog.waitFor();
    await dialog.getByLabel("New task").check();
    await dialog.getByRole("button", { name: "Continue to launch" }).click();
    // The launch dialog is the accessible modal now; Focus file stays mounted
    // beneath it (Radix hides it from the accessibility tree meanwhile).
    await page.waitForTimeout(300);
    const launchOpen = await page.locator('[role="dialog"]:not(.file-focus)').count();
    record(`${label}: launch opens above focus`, launchOpen === 1 && await page.locator(".file-focus").count() === 1, launchOpen);
    await page.keyboard.press("Escape");
    await page.waitForTimeout(200);
    record(`${label}: Escape closes launch first`, await page.locator('[role="dialog"]:not(.file-focus)').count() === 0 && await dialog.count() === 1);
    await page.keyboard.press("Escape");
    await dialog.waitFor({ state: "detached" });
    record(`${label}: second Escape closes focus`, await page.locator(".file-focus").count() === 0);
  } catch (error) {
    record(`${label}: journey completed`, false, error.message.split("\n")[0]);
  } finally {
    await context.close();
  }
}

// Review RFR-1/2/4/5 in the real browser: resizing through the tray's docking
// boundary settles (docked only from a 992px stage), leaving Transcript gives
// the space back, a slow Reload keeps the reading place, and the same page
// showing another agent restores the initial split.
async function runReviewRegressions(browser) {
  const label = "core regressions";
  const reads = [];
  const errors = [];
  let slow = false;
  const context = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  await hydrate(context, [makeAgent("a1"), makeAgent("a2")]);
  await context.route((u) => u.pathname.startsWith("/api/"), apiHandler("", reads));
  await context.route((u) => /\/api\/sessions\/a[12]\/file$/.test(u.pathname), async (route) => {
    if (slow) await new Promise((resolve) => setTimeout(resolve, 1500));
    await route.fallback();
  });
  const page = await context.newPage();
  page.on("pageerror", (error) => errors.push(error.message));
  const navigate = (path) => page.evaluate((to) => { history.pushState(null, "", to); dispatchEvent(new PopStateEvent("popstate")); }, path);
  const separators = () => page.getByRole("separator").count();
  try {
    await page.goto(`${BASE}/agent/a1?file=${encodeURIComponent(FILE_PATH)}`);
    await page.locator(".file-viewer-rendered").waitFor();
    await page.locator(".transcript-item .message").first().click({ button: "right" });
    await page.getByRole("button", { name: "Annotate whole event" }).click();
    await page.locator(".annotation-tray").waitFor();

    // RFR-1: sweep the viewport through the boundary both ways.
    const wrong = [];
    for (const widths of [[1100, 1000, -2], [1000, 1100, 2]]) {
      for (let width = widths[0]; widths[2] < 0 ? width >= widths[1] : width <= widths[1]; width += widths[2]) {
        await page.setViewportSize({ width, height: 900 });
        await page.waitForTimeout(60);
        const { stage, mode } = await page.evaluate(() => {
          const el = document.querySelector(".chat-stage");
          return { stage: el.clientWidth, mode: el.getAttribute("data-tray") };
        });
        const expected = stage >= 992 ? "docked" : "overlay";
        if (mode !== expected || await separators() !== (expected === "docked" ? 2 : 1)) wrong.push(`${width}:${stage}:${mode}`);
      }
    }
    record(`${label}: tray form settles through the docking boundary`, wrong.length === 0 && errors.length === 0, [...wrong, ...errors].join(", ") || undefined);
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.waitForTimeout(100);

    // RFR-4: a slow Reload's placeholder does not erase the reading place.
    const body = page.locator(".chat-side-start .file-viewer-body");
    await body.evaluate((el) => { el.scrollTop = 600; el.dispatchEvent(new Event("scroll")); });
    await page.waitForTimeout(100);
    const before = await body.evaluate((el) => el.scrollTop);
    slow = true;
    await page.getByRole("button", { name: "Reload" }).click();
    await page.getByText("Reading…").waitFor();
    await page.waitForTimeout(300);
    await page.locator(".file-viewer-rendered").waitFor({ timeout: 5000 });
    slow = false;
    await page.waitForTimeout(200);
    const after = await body.evaluate((el) => el.scrollTop);
    record(`${label}: slow Reload keeps the reading place`, before > 0 && Math.abs(after - before) <= 2, `${before} → ${after}`);

    // RFR-2: another tab gives the space back; Transcript restores it.
    await page.getByRole("tab", { name: "Files" }).click();
    await page.waitForTimeout(200);
    const away = await page.evaluate(() => ({ file: document.querySelector(".chat-stage")?.getAttribute("data-file"), tray: document.querySelector(".chat-stage")?.getAttribute("data-tray") }));
    record(`${label}: leaving Transcript removes dividers`, await separators() === 0 && !away.file && !away.tray, JSON.stringify(away));
    await page.getByRole("tab", { name: "Transcript" }).click();
    await page.locator(".chat-side-start .file-viewer").waitFor();
    await page.waitForTimeout(200);
    record(`${label}: returning to Transcript restores dividers`, await separators() === 2);

    // RFR-5: a chosen split does not follow the page to another agent.
    const fileDivider = page.getByRole("separator", { name: "Resize file and conversation" });
    const initial = await fileDivider.getAttribute("aria-valuenow");
    await fileDivider.focus();
    await page.keyboard.press("Shift+ArrowRight");
    const chosen = await fileDivider.getAttribute("aria-valuenow");
    await navigate(`/agent/a2?file=${encodeURIComponent(FILE_PATH)}`);
    await page.waitForFunction(() => location.pathname === "/agent/a2");
    await page.locator(".chat-side-start .file-viewer-rendered").waitFor();
    await page.waitForTimeout(200);
    // a2 has no drafts, so its initial split is half the space without a tray.
    const r = await rects(page);
    record(`${label}: another agent opens at the initial split`, chosen !== initial && Math.abs(r.file.width - r.card.width) <= 4, `${initial} → ${chosen}; a2 ${r.file.width} vs ${r.card.width}`);
    record(`${label}: no page errors`, errors.length === 0, errors.join(", ") || undefined);
  } catch (error) {
    await page.screenshot({ path: join(OUT_DIR, "core-regressions-failure.png") }).catch(() => {});
    record(`${label}: journey completed`, false, error.message.split("\n").slice(0, 8).join(" | "));
  } finally {
    await context.close();
  }
}

async function main() {
  const server = await createServer({ server: { host: "127.0.0.1", port: PORT, strictPort: true }, logLevel: "error" });
  await server.listen();
  const browser = await chromium.launch().catch(() => chromium.launch({ executablePath: "/opt/pw-browsers/chromium" }));
  const only = process.argv[3];
  try {
    for (const skin of SKINS) {
      if (only && skinLabel(skin) !== only) continue;
      await runFormats(browser, skin);
      for (const width of (process.argv[4] ? [Number(process.argv[4])] : [1024, 1440, 2137])) {
        await run(browser, skin, width, false);
        await run(browser, skin, width, true);
      }
    }
    await runNestedLaunch(browser);
    await runReviewRegressions(browser);
  } finally {
    await browser.close();
    await server.close();
  }
  writeFileSync(join(OUT_DIR, "report.json"), JSON.stringify(checks, null, 2));
  console.log(`\nwrote ${checks.length} checks, ${checks.filter((c) => !c.pass).length} failing`);
  console.log(`out dir: ${OUT_DIR}`);
  if (checks.some((c) => !c.pass)) process.exitCode = 1;
}

async function runFormats(browser, skin) {
  const context = await browser.newContext({ viewport: { width: 1024, height: 900 } });
  await hydrate(context, [makeAgent("a1")]);
  const reads = [];
  await context.route((u) => u.pathname.startsWith("/api/"), apiHandler(skin, reads));
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  try {
    await page.goto(`${BASE}/agent/a1?file=docs/data.json`);
    await page.getByRole("button", { name: "Formatted", exact: true }).waitFor();
    record(`${skinLabel(skin)}: JSON formatted`, (await page.locator(".file-viewer-body code").textContent()).includes('\n  "name": "Chuck"'));
    await page.getByRole("button", { name: "Source", exact: true }).click();
    record(`${skinLabel(skin)}: JSON exact source`, (await page.locator(".file-viewer-body code").textContent()).includes('{"name":"Chuck","items":[1,2,3]}'));
    await page.goto(`${BASE}/agent/a1?file=short.go`);
    await page.locator('[data-file-line="1"]').waitFor();
    const compact = await page.evaluate(() => {
      const body = document.querySelector(".file-viewer-body");
      const code = body.querySelector('div[class*="language-"]');
      return code.getBoundingClientRect().width < body.clientWidth - 20;
    });
    record(`${skinLabel(skin)}: short code background fits content`, compact);
    await page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-compact-code.png`) });
    const data = await page.evaluate(() => {
      const canvas = document.createElement("canvas");
      canvas.width = 800; canvas.height = 300;
      const ctx = canvas.getContext("2d");
      ctx.fillStyle = "#c6dfd2"; ctx.fillRect(0, 0, 800, 300);
      ctx.fillStyle = "#163b2c"; ctx.font = "48px sans-serif"; ctx.fillText("Image reader preview", 50, 165);
      return canvas.toDataURL("image/png").split(",")[1];
    });
    await context.route((u) => u.pathname.endsWith("/file") && u.searchParams.get("path") === "preview.png", (route) => route.fulfill({ json: { agent_id: "a1", path: "preview.png", size: 1000, mod_time: AT, line_count: 0, content: "", truncated: false, language: "", image: { mime_type: "image/png", data } } }));
    await page.goto(`${BASE}/agent/a1?file=preview.png`);
    const image = page.getByRole("img", { name: "preview.png" });
    await image.waitFor();
    await page.waitForFunction(() => document.querySelector(".file-viewer-image")?.naturalWidth === 800);
    const fits = await image.evaluate((img) => img.width <= img.parentElement.clientWidth && Math.abs(img.width / img.height - 8 / 3) < 0.05);
    record(`${skinLabel(skin)}: image fits and preserves aspect ratio`, fits);
    await page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-image-preview.png`) });
    await page.getByRole("button", { name: "Reload", exact: true }).click();
    await image.waitFor();
    record(`${skinLabel(skin)}: image reload`, reads.length >= 2 && await image.count() === 1);
    record(`${skinLabel(skin)}: format journeys no browser errors`, errors.length === 0, errors.join("; "));
  } finally { await context.close(); }
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
