// Rendered evidence for the project Files dialog (FS-11.A6–A10, TS-08.R172/R173,
// INV §13): at the 1024px desktop floor and 1440px, in every appearance, the
// Files button opens a viewport-fitted dialog whose controls and table header
// stay visible while the table scrolls, long names and paths wrap inside their
// columns, a keyboard-opened preview returns to the same row and scroll
// position, CSV opens as a table, and HTML opens in a second tab through the
// sandboxed first-party viewer. Every /api call is stubbed; no server needed.
//
//   node scripts/project-files-render.mjs [outDir]
//
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), `project-files-render-${Date.now()}`)] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });

const PORT = 5198;
const BASE = `http://localhost:${PORT}`;
const LONG_NAME = `a-deliberately-long-generated-report-name-${"segment-".repeat(8)}final.md`;
const LONG_DIR = `research/${"nested-folder-with-a-long-name/".repeat(3)}leaf`;
const entries = [
  { path: `${LONG_DIR}/${LONG_NAME}`, name: LONG_NAME, kind: "file", mod_time: "2026-11-20T10:00:00Z" },
  { path: "results/table.csv", name: "table.csv", kind: "file", mod_time: "2026-11-20T09:30:00Z" },
  { path: "results/page.html", name: "page.html", kind: "file", mod_time: "2026-11-20T09:00:00Z" },
  ...Array.from({ length: 60 }, (_, i) => ({ path: `notes/note-${String(i).padStart(2, "0")}.md`, name: `note-${String(i).padStart(2, "0")}.md`, kind: "file", mod_time: `2026-11-1${i % 10}T08:00:00Z` })),
  { path: "link", name: "link", kind: "symlink", mod_time: "2026-11-01T08:00:00Z" },
  { path: "research", name: "research", kind: "directory", mod_time: null },
  { path: "results", name: "results", kind: "directory", mod_time: null },
  { path: "notes", name: "notes", kind: "directory", mod_time: null },
];
const files = {
  "results/table.csv": "name,count\nbeta,10\nalpha,2\n,\ngamma,\"1,000\"\n",
  "results/page.html": "<h1 id=x>Sandboxed</h1><script>document.getElementById('x').textContent='scripted'</script>",
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
    if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210], resource_dir: "/Users/me/.chuck/project-resources/demo" } } });
    if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 }, expanded: [] } });
    if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
    if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
    if (path === "/api/capabilities") return route.fulfill({ json: { terminal: { available: true, default_driver: "xterm", drivers: {} }, reveal_in_finder: true } });
    if (path === "/api/projects/demo/resource-files") return route.fulfill({ json: { project_id: "demo", folder: "", partial: false, entries } });
    if (path === "/api/projects/demo/resource-file") {
      const file = url.searchParams.get("path");
      const content = files[file] ?? `# ${file}\n\nShared note.\n`;
      return route.fulfill({ json: { agent_id: "", project_id: "demo", path: file, size: content.length, mod_time: "", line_count: 3, content, truncated: false, language: file.endsWith(".md") ? "markdown" : file.endsWith(".html") ? "html" : "", editable: false, edit_refusal: "project_read_only" } });
    }
    return route.fulfill({ json: {} });
  };
}

async function open(browser, skin, width) {
  const context = await browser.newContext({ viewport: { width, height: 800 } });
  await context.addInitScript(() => {
    delete window.SharedWorker;
    window.EventSource = class extends EventTarget {
      constructor() { super(); setTimeout(() => { this.onopen?.(); this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: "__hydrated__", data: { hydrated: true } }) })); }); }
      close() {}
    };
  });
  await context.route((u) => u.pathname.startsWith("/api/"), api(skin));
  const page = await context.newPage();
  const errors = [];
  page.on("pageerror", (err) => errors.push(err.message));
  await page.goto(`${BASE}/project/demo`);
  return { context, page, errors };
}

const inside = (inner, outer) => inner.x >= outer.x - 0.5 && inner.x + inner.width <= outer.x + outer.width + 0.5 && inner.y >= outer.y - 0.5 && inner.y + inner.height <= outer.y + outer.height + 0.5;

const server = await createServer({ server: { port: PORT, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  for (const skin of ["", "sky-grove", "studio"]) {
    const name = skin || "core";
    for (const width of [1024, 1440]) {
      const label = `${name} ${width}`;
      const { context, page, errors } = await open(browser, skin, width);
      await page.getByRole("button", { name: "Files" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByRole("table").waitFor();
      // Let the shared dialog entrance finish before measuring or capturing.
      await page.waitForFunction(() => document.getAnimations().every((a) => a.playState !== "running"));
      record(`${label}: search focused on open`, await dialog.getByRole("searchbox", { name: "Search files" }).evaluate((el) => el === document.activeElement));
      const box = await dialog.boundingBox();
      record(`${label}: dialog fits the viewport`, box && box.x >= 0 && box.y >= 0 && box.x + box.width <= width && box.y + box.height <= 800, JSON.stringify(box));
      const longRow = dialog.locator(`tr[data-path="${LONG_DIR}/${LONG_NAME}"]`);
      const cells = await longRow.evaluate((row) => [...row.cells].map((c) => ({ scroll: c.scrollWidth, client: c.clientWidth })));
      record(`${label}: long name and path stay inside their columns`, cells.every((c) => c.scroll <= c.client + 1), JSON.stringify(cells));
      const wrap = dialog.locator(".project-files-table-wrap");
      const wrapBox = await wrap.boundingBox();
      record(`${label}: table scrolls inside the dialog`, await wrap.evaluate((el) => el.scrollHeight > el.clientHeight));
      await wrap.evaluate((el) => { el.scrollTop = 600; });
      const header = await dialog.locator("thead th").first().boundingBox();
      const search = await dialog.locator(".project-files-search").boundingBox();
      record(`${label}: header and search stay visible while scrolling`, header && wrapBox && header.y >= wrapBox.y - 1 && inside(search, box), `${header?.y} vs ${wrapBox?.y}`);
      await page.screenshot({ path: join(outDir, `list-${name}-${width}.png`) });
      // Keyboard preview and return to the same row and scroll position.
      const row = dialog.locator('tr[data-path="notes/note-37.md"]');
      await row.focus();
      const scrollBefore = await wrap.evaluate((el) => el.scrollTop);
      await page.keyboard.press("Enter");
      await dialog.getByRole("button", { name: "Back to files" }).waitFor();
      await page.screenshot({ path: join(outDir, `preview-${name}-${width}.png`) });
      await dialog.getByRole("button", { name: "Back to files" }).click();
      await page.waitForFunction(() => document.activeElement?.getAttribute("data-path") === "notes/note-37.md");
      const scrollAfter = await wrap.evaluate((el) => el.scrollTop);
      record(`${label}: Back to files restores row focus and scroll`, Math.abs(scrollAfter - scrollBefore) <= 1, `${scrollBefore} → ${scrollAfter}`);
      // CSV opens as a sortable table.
      await dialog.getByRole("searchbox", { name: "Search files" }).fill("table");
      await dialog.locator('tr[data-path="results/table.csv"]').click();
      await dialog.locator(".file-viewer-csv-table").waitFor();
      record(`${label}: CSV shows quoted commas literally`, (await dialog.locator(".file-viewer-csv-table").innerText()).includes("1,000"));
      await page.screenshot({ path: join(outDir, `csv-${name}-${width}.png`) });
      await dialog.getByRole("button", { name: "Back to files" }).click();
      record(`${label}: query kept after preview`, (await dialog.getByRole("searchbox", { name: "Search files" }).inputValue()) === "table");
      record(`${label}: no page errors`, errors.length === 0, errors.join(" | "));
      await context.close();
    }
  }
  // HTML opens in a second tab through the sandboxed viewer; the first tab keeps its list.
  const { context, page, errors } = await open(browser, "", 1440);
  await page.getByRole("button", { name: "Files" }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByRole("searchbox", { name: "Search files" }).fill("page");
  await dialog.locator('tr[data-path="results/page.html"]').click({ button: "right" });
  const [tab] = await Promise.all([context.waitForEvent("page"), dialog.getByRole("button", { name: "Open in new tab" }).click()]);
  await tab.waitForLoadState();
  const frame = tab.locator("iframe");
  await frame.waitFor({ timeout: 10_000 });
  const sandbox = await frame.getAttribute("sandbox");
  record("html tab: sandboxed iframe", sandbox === "allow-scripts", sandbox);
  record("html tab: no opener", await tab.evaluate(() => window.opener === null));
  await tab.screenshot({ path: join(outDir, "html-tab.png") });
  record("html tab: original keeps its query", (await dialog.getByRole("searchbox", { name: "Search files" }).inputValue()) === "page");
  record("html tab: no page errors", errors.length === 0, errors.join(" | "));
  await context.close();
  // The dialog adds no motion of its own: only the shared dialog entrance runs,
  // and rows have no transition (TS-08.R172).
  const reduced = await open(browser, "", 1024);
  await reduced.page.emulateMedia({ reducedMotion: "reduce" });
  await reduced.page.getByRole("button", { name: "Files" }).click();
  await reduced.page.locator(".project-files tbody tr").first().waitFor();
  const animation = await reduced.page.locator(".project-files").evaluate((el) => getComputedStyle(el).animationName);
  const rowMotion = await reduced.page.locator(".project-files tbody tr").first().evaluate((el) => `${getComputedStyle(el).animationName}/${getComputedStyle(el).transitionDuration}`);
  record("reduced motion: only the shared dialog entrance", animation === "ad-dialog-enter" && rowMotion === "none/0s", `${animation}; row ${rowMotion}`);
  await reduced.context.close();
} finally {
  await browser.close();
  await server.close();
}
writeFileSync(join(outDir, "report.json"), JSON.stringify(checks, null, 2));
console.log(`\n${checks.filter((c) => c.pass).length}/${checks.length} passed; screenshots in ${outDir}`);
process.exit(checks.every((c) => c.pass) ? 0 : 1);
