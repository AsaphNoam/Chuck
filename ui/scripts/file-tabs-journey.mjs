// Rendered evidence for saved file-viewer tabs (FS-03.R101–R106, FS-20.R56,
// FS-21.R59, TS-08.R140–R145): desktop conversation tabs (follow, memory,
// reorder, close, overflow), Focus file, history and reload, archived sharing,
// a missing file, rooms with source-qualified tabs, the phone Files reader and
// the per-conversation cap. Drives Vite + the real UI source in Chromium with
// every /api call and the event stream stubbed. File reads are counted at the
// stubbed HTTP boundary, never from the DOM.
//
//   node scripts/file-tabs-journey.mjs [outDir] [skin] [section] [width]
//
// skin is core | sky-grove | studio (default: all); section is one of
// main, overflow, strict-restore, focus, history, archive, missing, cap, room, phone.
// Writes screenshots and report.json to outDir. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium, devices } from "playwright";

const OUT_DIR = process.argv[2] || join(tmpdir(), `file-tabs-journey-${Date.now()}`);
mkdirSync(OUT_DIR, { recursive: true });
const ONLY_SKIN = process.argv[3];
const ONLY_SECTION = process.argv[4];
const ONLY_WIDTH = process.argv[5] ? Number(process.argv[5]) : null;

const PORT = 5199;
const BASE = `http://localhost:${PORT}`;
const AT = "2026-10-17T09:00:00Z";
const SKINS = ["", "sky-grove", "studio"];
const skinLabel = (skin) => skin || "core";
const STORAGE_KEY = "chuck.file-tabs.v1";
const MARKER = "CONTENT-MARKER-NEVER-STORED";
const enc = encodeURIComponent;

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

// ---- Fixture ------------------------------------------------------------

const filler = (n) => Array.from({ length: n }, (_, i) => `## Section ${i + 1}\n\nParagraph ${i + 1} of ordinary prose that fills the reading area so the document scrolls. ${MARKER}\n\n- Item one\n- Item two`).join("\n\n");
const FILES = {
  "a.md": `# File A\n\nLinks: [to b](./b.md), [to c](./c.md) and [outside](../outside.md).\n\nLine four.\nLine five.\nLine six.\nLine seven.\nLine eight.\nLine nine.\nLine ten.\nLine eleven.\nLine twelve.\n\n${filler(30)}`,
  "b.md": `# File B\n\nOn to [to c](./c.md).\n\n${filler(30)}`,
  "c.md": `# File C\n\nCited line: [a line 12](./a.md:12) and [back to a](./a.md).\n\nLine four.\nLine five.\n\n${filler(30)}`,
};
const bodyFor = (path) => FILES[path.split("/").pop()] ?? `# ${path.split("/").pop()}\n\nGeneric file. ${MARKER}\n`;

const event = (agentId, seq, type, data) => ({ agent_id: agentId, seq, type, ts: AT, data });
const transcriptFor = (agentId) => [
  event(agentId, 0, "session_meta", { name: "Tabs demo", project: "demo", backend: "claude", model: "m", role: "implementer" }),
  ...Array.from({ length: 12 }, (_, i) => event(agentId, i + 1, i % 2 ? "assistant_text" : "user_text", {
    text: i === 1
      ? "Open [a-doc](docs/a.md), [b-doc](docs/b.md), [c-doc](docs/c.md) or [missing-doc](docs/missing.md)."
      : `Message ${i + 1}: ${"conversation content ".repeat(8)}`,
  })),
];
function makeAgent(id, overrides = {}) {
  return {
    agent_id: id, name: "Tabs demo", role: "implementer", project: "demo", backend: "claude", model: "m",
    fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: AT,
    running: true, state: "idle", detail: "", context_pct: 0, updated_at: 0, archived: false,
    ...overrides,
  };
}

// Room fixture (ended discussion) with two participants linking the same notes.md.
const member = (agent_id, name, project, order) => ({
  agent_id, name, project, role: "participant", order, limit: 3, completed: 1, may_leave: true, state: "active",
  setup_state: "ready", exists: true, running: true, agent_status: "idle",
});
const ROOM = {
  version: 1, room_id: "tt_demo", title: "Notes council", goal: "Compare the two participants' notes.",
  origin_project: "alpha", phase: "ended", control: "running", hold: "", end_reason: "operator", judge_status: "",
  participants: ["Ari", "Bo"], revision: 4, created_at: AT, updated_at: AT, openings: true,
  members: [member("a_ari", "Ari", "alpha", 0), member("a_bo", "Bo", "alpha", 1)],
  active: undefined, active_attempts: [], pending: [], failed: [],
  judge: { enabled: false, status: "", agent_id: "", error: "" }, deletable: true,
};
const entry = (seq, agent_id, agent_name, attempt_id) => ({
  seq, kind: "opening", agent_id, agent_name, project: "alpha", body: "My notes are in [notes](notes.md).", attempt_id, created_at: AT,
});
const ROOM_ENTRIES = [entry(1, "a_ari", "Ari", "tta_a"), entry(2, "a_bo", "Bo", "tta_b")];
const ROOM_SOURCES = [
  { source_id: "src_a", agent_id: "a_ari", agent_name: "Ari", project: "alpha", attempt_ids: ["tta_a"] },
  { source_id: "src_b", agent_id: "a_bo", agent_name: "Bo", project: "alpha", attempt_ids: ["tta_b"] },
];

// ---- Stubs --------------------------------------------------------------

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

const fileJson = (agentId, path) => ({
  agent_id: agentId, path, size: 100, mod_time: AT, line_count: bodyFor(path).split("\n").length,
  content: bodyFor(path), truncated: false, language: "markdown",
});
const NOT_FOUND = { error: { code: "not_found", message: "docs/missing.md does not exist in this agent's working directory" } };

// The Vite dev page runs React.StrictMode, which replays every mount effect in the
// same task, so one real read arrives as two identical requests a few ms apart.
// A repeat of the same read within STRICT_REPLAY_MS is that replay, not a second
// read; any later repeat (a reload, a reselect, a real duplicate) still counts.
const STRICT_REPLAY_MS = 40;
function noteRead(reads, key) {
  const now = Date.now();
  reads.last ??= new Map();
  if (now - (reads.last.get(key) ?? -Infinity) > STRICT_REPLAY_MS) reads.push(key);
  reads.last.set(key, now);
}

// reads are strings: "<agent>:<path>" for sessions, "<source>:<path>" for rooms.
function apiHandler(skin, reads, trackedPaths = ["docs/a.md", "docs/b.md", "docs/c.md"]) {
  return (route) => {
    const url = new URL(route.request().url());
    const path = url.pathname;
    if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarding_complete: true } });
    if (path === "/api/backends") return route.fulfill({ json: { backends: {} } });
    if (path === "/api/projects") return route.fulfill({ json: { demo: { title: "Demo", archived: false, color: [90, 130, 210] }, alpha: { title: "Alpha", cwd: "/tmp", color: [80, 120, 200] } } });
    if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 2, gap: 16 }, expanded: [] } });
    if (path === "/api/tasks") return route.fulfill({ json: { tasks: [] } });
    if (path === "/api/think-tanks") return route.fulfill({ json: { rooms: [], clipped: false } });
    if (path === "/api/think-tanks/tt_demo") return route.fulfill({ json: ROOM });
    if (path === "/api/think-tanks/tt_demo/entries") return route.fulfill({ json: { version: 1, entries: ROOM_ENTRIES, complete: true } });
    if (path === "/api/think-tanks/tt_demo/activity") return route.fulfill({ json: { version: 1, activity: [], complete: true } });
    if (path === "/api/think-tanks/tt_demo/files") return route.fulfill({ json: { version: 1, sources: ROOM_SOURCES, files: [] } });
    if (path === "/api/think-tanks/tt_demo/commands") return route.fulfill({ json: { sources: [], commands: [] } });
    const source = path.match(/^\/api\/think-tanks\/tt_demo\/sources\/([^/]+)\/file$/);
    if (source) {
      const requested = url.searchParams.get("path") || "";
      noteRead(reads, `${decodeURIComponent(source[1])}:${requested}`);
      return route.fulfill({ json: { ...fileJson("x", requested), content: `# Notes from ${source[1]}\n\n${MARKER}\n` } });
    }
    const transcript = path.match(/^\/api\/sessions\/(a\d)\/transcript$/);
    if (transcript) return route.fulfill({ json: { agent_id: transcript[1], has_more: false, events: transcriptFor(transcript[1]) } });
    const file = path.match(/^\/api\/sessions\/(a\d)\/file$/);
    if (file) {
      const requested = url.searchParams.get("path") || "";
      noteRead(reads, `${file[1]}:${requested}`);
      if (requested.endsWith("missing.md")) return route.fulfill({ status: 404, json: NOT_FOUND });
      return route.fulfill({ json: fileJson(file[1], requested) });
    }
    if (/\/prompt$/.test(path)) return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "none" } } });
    if (/\/available-commands$/.test(path)) return route.fulfill({ json: { commands: [] } });
    const tracked = path.match(/^\/api\/sessions\/(a\d)\/files$/);
    if (tracked) return route.fulfill({ json: { agent_id: tracked[1], files: trackedPaths.map((p) => ({ path: p, edit_count: 2, last_ts: AT, has_diff: false, diff_refs: [] })) } });
    return route.fulfill({ json: {} });
  };
}

// strict: false serves the entry modules without React.StrictMode, which is what
// production runs. Dev StrictMode replays effects, which hides a bare-address
// restore (see flowStrictRestore); flows that restore from a bare address use
// the production semantics so they test the feature, not the replay.
async function makeEnv(browser, { skin, width, height = 900, agents = [], seed = null, phone = false, tracked, strict = true }) {
  const reads = [];
  const errors = [];
  const context = await browser.newContext(phone
    ? { ...devices["iPhone 13"], viewport: { width, height } }
    : { viewport: { width, height } });
  await hydrate(context, agents);
  if (!strict) {
    await context.route((u) => /^\/src\/(remote\/)?main\.tsx$/.test(u.pathname), async (route) => {
      const response = await route.fetch();
      await route.fulfill({ response, body: (await response.text()).replaceAll("React.StrictMode", "React.Fragment") });
    });
  }
  if (seed) {
    await context.addInitScript(({ key, value }) => {
      if (!localStorage.getItem(key)) localStorage.setItem(key, JSON.stringify(value));
    }, { key: STORAGE_KEY, value: seed });
  }
  await context.route((u) => u.pathname.startsWith("/api/"), apiHandler(skin, reads, tracked));
  const page = await context.newPage();
  page.setDefaultTimeout(10_000);
  page.on("pageerror", (error) => errors.push(error.message));
  return { context, page, reads, errors };
}

const seedRecord = (owner, tabs, active) => ({
  v: 1,
  owners: { [owner]: { tabs: tabs.map((t) => ({ rendered: true, anchors: {}, ...t })), active } },
});

// ---- Helpers ------------------------------------------------------------

const count = (reads, key) => reads.filter((r) => r === key).length;
const pageOverflow = (page) => page.evaluate(() => document.documentElement.scrollWidth > window.innerWidth);
const tabNames = (page) => page.evaluate(() => [...document.querySelectorAll(".file-tab .file-tab-name")].map((el) => el.textContent));
const activeName = (page) => page.evaluate(() => document.querySelector('.file-tab[data-state="active"] .file-tab-name')?.textContent ?? null);
const tabCount = (page) => page.locator(".file-tab").count();
const urlParam = (page, key) => new URL(page.url()).searchParams.get(key);
const savedTabs = (page) => page.evaluate((key) => {
  const raw = localStorage.getItem(key);
  return raw ? JSON.parse(raw) : null;
}, STORAGE_KEY);
const tab = (page, name) => page.locator(".file-tab-select").filter({ hasText: name }).first();
const settle = (page, ms = 150) => page.waitForTimeout(ms);
const sameList = (a, b) => JSON.stringify(a) === JSON.stringify(b);

async function waitFile(page, heading) {
  await page.getByRole("heading", { name: heading, exact: true }).first().waitFor();
}

async function dragTab(page, from, to) {
  const a = await page.locator(".file-tab").filter({ hasText: from }).first().boundingBox();
  const b = await page.locator(".file-tab").filter({ hasText: to }).first().boundingBox();
  await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
  await page.mouse.down();
  await page.mouse.move(a.x + a.width / 2 - 12, a.y + a.height / 2, { steps: 3 });
  await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 10 });
  await page.mouse.up();
  await settle(page, 250);
}

const fail = async (page, label, shot, error, errors) => {
  record(`${label}: journey completed`, false, `${error.message.split("\n")[0]} ${errors.join(" | ")}`);
  await shot("failure").catch(() => {});
  if (process.env.DEBUG_STORAGE) console.log(await page.evaluate((key) => `${location.href} ${(localStorage.getItem(key) ?? "none").slice(0, 300)} notice=${document.querySelector(".file-tabs-notice")?.textContent}`, STORAGE_KEY).catch((e) => String(e)));
};

// ---- 1. Desktop live conversation ----------------------------------------

async function flowMain(browser, skin, width) {
  const label = `${skinLabel(skin)} main@${width}`;
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")] });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-main-${width}-${name}.png`) });
  const side = page.locator(".chat-side-start");
  try {
    await page.goto(`${BASE}/agent/a1?file=${enc("docs/a.md")}`);
    await waitFile(page, "File A");
    record(`${label}: first file opens one tab`, sameList(await tabNames(page), ["a.md"]));

    // A in Source, scrolled; leave to B through a transcript link.
    await side.getByRole("button", { name: "Source", exact: true }).click();
    await side.locator("[data-file-line]").first().waitFor();
    await side.locator(".file-viewer-body").evaluate((el) => { el.scrollTop = 600; });
    await settle(page, 150);
    const leftAt = await side.locator(".file-viewer-body").evaluate((el) => el.scrollTop);
    await page.locator(".transcript-view").getByRole("button", { name: "b-doc" }).click();
    await waitFile(page, "File B");
    // B -> C by a link inside the file.
    await side.locator(".file-viewer-rendered").getByRole("button", { name: "to c" }).click();
    await waitFile(page, "File C");
    await shot("three-tabs");
    record(`${label}: A, B, C are three tabs`, sameList(await tabNames(page), ["a.md", "b.md", "c.md"]), (await tabNames(page)).join(","));
    record(`${label}: three files read once each`, reads.length === 3 && count(reads, "a1:docs/a.md") === 1 && count(reads, "a1:docs/b.md") === 1 && count(reads, "a1:docs/c.md") === 1, reads.join(", "));
    record(`${label}: address names the active file`, urlParam(page, "file") === "docs/c.md");

    // Back to A from memory: no read, Source mode and place kept.
    await tab(page, "a.md").click();
    await side.locator("[data-file-line]").first().waitFor();
    await settle(page, 200);
    record(`${label}: returning to A adds no read`, reads.length === 3, reads.join(", "));
    record(`${label}: A keeps Source mode`, await side.getByRole("button", { name: "Source", exact: true }).getAttribute("aria-pressed") === "true");
    const backAt = await side.locator(".file-viewer-body").evaluate((el) => el.scrollTop);
    record(`${label}: A keeps its scroll position`, leftAt > 100 && Math.abs(backAt - leftAt) <= 40, `${leftAt} -> ${backAt}`);

    // Cited-line link from C reopens A without a duplicate tab.
    await tab(page, "c.md").click();
    await waitFile(page, "File C");
    await side.locator(".file-viewer-rendered").getByRole("button", { name: "a line 12" }).click();
    await side.locator("[data-file-line]").first().waitFor();
    await settle(page, 200);
    record(`${label}: cited line reuses A's tab`, await tabCount(page) === 3 && await activeName(page) === "a.md", `${await tabCount(page)} tabs, active ${await activeName(page)}`);
    record(`${label}: cited line is in the address`, urlParam(page, "fileLine") === "12" && urlParam(page, "file") === "docs/a.md", page.url());
    record(`${label}: cited line adds no second A read`, count(reads, "a1:docs/a.md") === 1, reads.join(", "));

    // Reorder by pointer drag, then by the Move buttons.
    await dragTab(page, "c.md", "a.md");
    record(`${label}: pointer drag reorders`, sameList(await tabNames(page), ["c.md", "a.md", "b.md"]), (await tabNames(page)).join(","));
    record(`${label}: drag keeps the active tab`, await activeName(page) === "a.md");
    await page.getByRole("button", { name: "Move a.md right", exact: true }).click();
    record(`${label}: Move right`, sameList(await tabNames(page), ["c.md", "b.md", "a.md"]), (await tabNames(page)).join(","));
    await page.getByRole("button", { name: "Move a.md left", exact: true }).click();
    await page.getByRole("button", { name: "Move a.md left", exact: true }).click();
    record(`${label}: Move left twice`, sameList(await tabNames(page), ["a.md", "c.md", "b.md"]), (await tabNames(page)).join(","));
    record(`${label}: moves add no reads`, reads.length === 3);
    const aTab = tab(page, "a.md");
    await aTab.focus();
    await page.keyboard.press("ArrowRight");
    const focusedB = await page.evaluate(() => document.activeElement?.getAttribute("aria-label") ?? document.activeElement?.textContent?.trim());
    await page.keyboard.press("Enter");
    await settle(page, 100);
    const focusRetained = await page.evaluate(() => document.activeElement?.getAttribute("role") === "tab" && document.activeElement?.getAttribute("aria-selected") === "true");
    await page.keyboard.press("ArrowLeft");
    const focusedPrevious = await page.evaluate(() => document.activeElement?.getAttribute("role") === "tab" && document.activeElement?.textContent?.includes("a.md"));
    record(`${label}: ArrowRight → Enter → ArrowLeft retains keyboard tab focus`, focusedB?.includes("c.md") && focusRetained && focusedPrevious, `initial neighbor ${focusedB}; selected focus ${focusRetained}; previous focus ${focusedPrevious}`);
    await page.keyboard.press("Enter");
    await settle(page, 100);
    await shot("reordered");

    // Close a background tab, the middle active tab, then the end.
    await page.getByRole("button", { name: "Close b.md", exact: true }).click();
    record(`${label}: background close keeps the active file`, sameList(await tabNames(page), ["a.md", "c.md"]) && await activeName(page) === "a.md");
    await page.locator(".transcript-view").getByRole("button", { name: "b-doc" }).click();
    await waitFile(page, "File B");
    record(`${label}: reopened B is a new tab at the end`, sameList(await tabNames(page), ["a.md", "c.md", "b.md"]) && count(reads, "a1:docs/b.md") === 2, `${(await tabNames(page)).join(",")} b reads ${count(reads, "a1:docs/b.md")}`);
    await tab(page, "c.md").click();
    await waitFile(page, "File C");
    await page.getByRole("button", { name: "Close c.md", exact: true }).click();
    await settle(page, 150);
    record(`${label}: closing the middle active tab selects the right neighbour`, sameList(await tabNames(page), ["a.md", "b.md"]) && await activeName(page) === "b.md" && urlParam(page, "file") === "docs/b.md", `${(await tabNames(page)).join(",")} active ${await activeName(page)}`);
    await page.getByRole("button", { name: "Close b.md", exact: true }).click();
    record(`${label}: closing the last-position active tab selects the left one`, await activeName(page) === "a.md" && await tabCount(page) === 1);
    await page.getByRole("button", { name: "Close a.md", exact: true }).click();
    await settle(page, 200);
    record(`${label}: closing the final tab closes the reader`, await page.locator(".file-viewer").count() === 0 && await page.locator(".file-tabs").count() === 0 && !urlParam(page, "file") && !urlParam(page, "fileLine"), page.url());
    record(`${label}: final close clears the saved set`, (await savedTabs(page)) === null);
    record(`${label}: no page overflow`, !(await pageOverflow(page)));
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 1b. Many tabs: overflow, active in view, styled controls --------------

async function flowOverflow(browser, skin, width) {
  const label = `${skinLabel(skin)} overflow@${width}`;
  const tabs = Array.from({ length: 12 }, (_, i) => ({
    id: `t${i}`,
    path: i % 3 === 0 ? `docs/${"a-very-long-descriptive-file-name-".repeat(5)}${i}.md` : `docs/file-${i}.md`,
  }));
  const seed = seedRecord("conversation:a1", tabs, "t11");
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")], seed, strict: false });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-overflow-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/agent/a1`);
    await page.locator(".file-tabs-list").waitFor();
    await page.locator(".file-viewer-rendered").waitFor();
    await settle(page, 300);
    record(`${label}: restores 12 tabs and reads only the active one`, await tabCount(page) === 12 && sameList(reads, ["a1:docs/file-11.md"]), `${await tabCount(page)} tabs, reads ${reads.join(",")}`);
    record(`${label}: strip does not widen the page`, !(await pageOverflow(page)));
    const geometry = await page.evaluate(() => {
      const list = document.querySelector(".file-tabs-list");
      const active = document.querySelector('.file-tab[data-state="active"] button[role=tab]');
      const l = list.getBoundingClientRect();
      const a = active.getBoundingClientRect();
      return { l: [l.left, l.right], a: [a.left, a.right], sw: list.scrollWidth, cw: list.clientWidth, fonts: document.fonts.status, scrolls: list.scrollWidth > list.clientWidth, inView: a.left >= l.left - 1 && a.right <= l.right + 1, scrollLeft: list.scrollLeft, strip: document.querySelector(".file-tabs").getBoundingClientRect().right <= innerWidth };
    });
    record(`${label}: strip scrolls inside the reader`, geometry.scrolls && geometry.strip, JSON.stringify(geometry));
    record(`${label}: active tab scrolled into view`, geometry.inView && geometry.scrollLeft > 0, JSON.stringify(geometry));
    // Selecting by hand (first tab, then the last) must also keep the tab in view.
    const inView = () => page.evaluate(() => {
      const l = document.querySelector(".file-tabs-list").getBoundingClientRect();
      const a = document.querySelector('.file-tab[data-state="active"] button[role=tab]').getBoundingClientRect();
      return a.left >= l.left - 1 && a.right <= l.right + 1;
    });
    await tab(page, "file-1.md").click();
    await settle(page, 300);
    const first = await inView();
    await page.getByRole("tab", { name: "11.md" }).last().click();
    await settle(page, 300);
    record(`${label}: selecting far tabs keeps each in view`, first && await inView() && await activeName(page) === "file-11.md", `first ${first}, last ${await inView()}, active ${await activeName(page)}`);
    const closeClip = await page.evaluate(() => {
      const l = document.querySelector(".file-tabs-list").getBoundingClientRect();
      const c = document.querySelector('.file-tab[data-state="active"] .file-tab-close').getBoundingClientRect();
      return { list: l.right, close: [c.left, c.right], visible: c.left >= l.left - 1 && c.right <= l.right + 1 };
    });
    record(`${label}: the active tab's close control is fully visible`, closeClip.visible, JSON.stringify(closeClip));
    const styled = await page.evaluate(() => {
      const css = (selector) => getComputedStyle(document.querySelector(selector));
      const tabStyle = css(".file-tab");
      const select = css(".file-tab-select");
      const close = document.querySelector(".file-tab-close").getBoundingClientRect();
      return {
        border: `${tabStyle.borderRightWidth} ${tabStyle.borderRightStyle}`,
        display: tabStyle.display,
        selectBorder: select.borderTopWidth,
        selectFont: select.fontFamily,
        closeW: close.width,
        activeUnderline: getComputedStyle(document.querySelector('.file-tab[data-state="active"]')).borderBottomWidth,
      };
    });
    record(`${label}: tab controls are styled`, styled.border !== "0px none" && styled.border.endsWith("solid") && styled.display === "flex" && styled.selectBorder === "0px" && styled.closeW >= 24 && styled.activeUnderline !== "0px" && !/Times|serif$/i.test(styled.selectFont.split(",")[0]), JSON.stringify(styled));
    await shot("twelve");
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 1c. Bare-address restore under the dev page's StrictMode ----------------

async function flowStrictRestore(browser, skin, width) {
  const label = `${skinLabel(skin)} strict-restore@${width}`;
  const seed = seedRecord("conversation:a1", [{ id: "ta", path: "docs/a.md" }, { id: "tb", path: "docs/b.md" }], "tb");
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")], seed });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-strict-restore-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/agent/a1`);
    await page.locator(".transcript-view").waitFor();
    await settle(page, 1500);
    record(`${label}: saved tabs show when the bare conversation address opens (StrictMode dev page)`, await page.locator(".file-viewer").count() === 1 && sameList(await tabNames(page), ["a.md", "b.md"]), `viewer ${await page.locator(".file-viewer").count()}, url ${page.url()}, reads ${reads.join(",")}`);
    await shot("bare");
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 2. Focus file -------------------------------------------------------

async function flowFocus(browser, skin, width) {
  const label = `${skinLabel(skin)} focus@${width}`;
  const { context, page, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")] });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-focus-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/agent/a1?file=${enc("docs/a.md")}`);
    await waitFile(page, "File A");
    await page.locator(".transcript-view").getByRole("button", { name: "b-doc" }).click();
    await waitFile(page, "File B");
    await page.getByRole("button", { name: "Focus file" }).click();
    const dialog = page.getByRole("dialog", { name: "Focus file" });
    await dialog.waitFor();
    await dialog.locator(".file-tabs").waitFor();
    record(`${label}: strip appears in Focus file`, await dialog.locator(".file-tabs").count() === 1 && await dialog.locator(".file-tab").count() === 2 && await page.locator(".chat-side-start .file-tabs").count() === 0);
    record(`${label}: Focus strip has no page overflow`, !(await pageOverflow(page)));
    await shot("dialog");
    await dialog.getByRole("button", { name: "Close b.md", exact: true }).click();
    await settle(page, 200);
    record(`${label}: closing one of two keeps Focus open`, await dialog.count() === 1 && await dialog.locator(".file-tab").count() === 1 && await dialog.getByRole("heading", { name: "File A", exact: true }).count() === 1);
    await dialog.getByRole("button", { name: "Close a.md", exact: true }).click();
    await dialog.waitFor({ state: "detached" });
    await settle(page, 200);
    record(`${label}: closing the last returns to the conversation`, await page.locator(".transcript-view").isVisible() && await page.locator(".file-viewer").count() === 0 && !urlParam(page, "file"));
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 3. History and reload ------------------------------------------------

async function flowHistory(browser, skin, width) {
  const label = `${skinLabel(skin)} history@${width}`;
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")] });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-history-${width}-${name}.png`) });
  const side = page.locator(".chat-side-start");
  try {
    await page.goto(`${BASE}/agent/a1`);
    await page.locator(".transcript-view").waitFor();
    const startLength = await page.evaluate(() => history.length);
    const links = page.locator(".transcript-view");
    await links.getByRole("button", { name: "a-doc" }).click();
    await waitFile(page, "File A");
    await links.getByRole("button", { name: "b-doc" }).click();
    await waitFile(page, "File B");
    await links.getByRole("button", { name: "c-doc" }).click();
    await waitFile(page, "File C");
    record(`${label}: three opens add exactly one history entry`, (await page.evaluate(() => history.length)) === startLength + 1, `${startLength} -> ${await page.evaluate(() => history.length)}`);
    await side.getByRole("button", { name: "Source", exact: true }).click();
    await side.locator("[data-file-line]").first().waitFor();
    await settle(page, 600);

    await page.goBack();
    await settle(page, 300);
    const afterBack = await savedTabs(page);
    record(`${label}: Back hides the reader`, await page.locator(".file-viewer").count() === 0 && !urlParam(page, "file"), page.url());
    record(`${label}: Back keeps the saved tabs`, afterBack?.owners?.["conversation:a1"]?.tabs.length === 3, JSON.stringify(afterBack?.owners && Object.keys(afterBack.owners)));
    await page.goForward();
    await side.locator("[data-file-line]").first().waitFor();
    record(`${label}: Forward reopens the reader`, urlParam(page, "file") === "docs/c.md" && sameList(await tabNames(page), ["a.md", "b.md", "c.md"]));

    // Reload restores order, active tab and mode, reading only the active file.
    reads.length = 0;
    await page.reload();
    await side.locator("[data-file-line]").first().waitFor();
    await settle(page, 600);
    record(`${label}: reload restores order and active tab`, sameList(await tabNames(page), ["a.md", "b.md", "c.md"]) && await activeName(page) === "c.md");
    record(`${label}: reload restores display mode`, await side.getByRole("button", { name: "Source", exact: true }).getAttribute("aria-pressed") === "true");
    record(`${label}: reload reads only the active file`, sameList(reads, ["a1:docs/c.md"]), reads.join(", "));
    await shot("reloaded");

    // An explicit file URL wins over the saved active tab.
    await tab(page, "a.md").click();
    await settle(page, 600);
    reads.length = 0;
    await page.goto(`${BASE}/agent/a1?file=${enc("docs/c.md")}&fileLine=4`);
    await side.locator("[data-file-line]").first().waitFor();
    await settle(page, 600);
    record(`${label}: explicit URL wins over the saved active tab`, await activeName(page) === "c.md" && await tabCount(page) === 3, `active ${await activeName(page)}`);
    record(`${label}: explicit URL reads only its file`, sameList(reads, ["a1:docs/c.md"]), reads.join(", "));

    // Saved record: paths, ids, modes and anchors only.
    const raw = await page.evaluate((key) => localStorage.getItem(key), STORAGE_KEY);
    const parsed = JSON.parse(raw);
    const allowed = new Set(["id", "path", "line", "rendered", "anchors"]);
    const keys = Object.values(parsed.owners).flatMap((o) => o.tabs.flatMap((t) => Object.keys(t)));
    record(`${label}: saved record holds no file contents`, !raw.includes(MARKER) && !raw.includes("File A") && keys.every((k) => allowed.has(k)) && raw.length < 5000, `${raw.length} chars; keys ${[...new Set(keys)].join(",")}`);
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 4. Archived conversation shares the live conversation's tabs -----------

async function flowArchive(browser, skin, width) {
  const label = `${skinLabel(skin)} archive@${width}`;
  const seed = seedRecord("conversation:a1", [{ id: "ta", path: "docs/a.md" }, { id: "tb", path: "docs/b.md" }], "tb");
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1", { running: false, archived: true })], seed, strict: false });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-archive-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/archive/a1`);
    await waitFile(page, "File B");
    await settle(page, 300);
    record(`${label}: archive shows the live conversation's tabs`, sameList(await tabNames(page), ["a.md", "b.md"]) && await activeName(page) === "b.md");
    record(`${label}: archive reads only the active tab`, sameList(reads, ["a1:docs/b.md"]), reads.join(", "));
    await tab(page, "a.md").click();
    await waitFile(page, "File A");
    await settle(page, 600);
    const saved = await savedTabs(page);
    record(`${label}: archive writes the shared conversation record`, saved?.owners?.["conversation:a1"]?.active === "ta" && saved.owners["conversation:a1"].tabs.length === 2);
    record(`${label}: archive address names the active file`, urlParam(page, "file") === "docs/a.md", page.url());
    await shot("tabs");
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 5. Missing file ---------------------------------------------------------

async function flowMissing(browser, skin, width) {
  const label = `${skinLabel(skin)} missing@${width}`;
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")] });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-missing-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/agent/a1?file=${enc("docs/missing.md")}`);
    await page.locator(".file-viewer-error").waitFor();
    record(`${label}: error shown in the tab's own viewer`, (await page.locator(".file-viewer-error").innerText()).includes("does not exist") && await page.locator(".file-viewer .file-tabs").count() === 1, await page.locator(".file-viewer-error").innerText());
    await shot("error");
    await page.locator(".transcript-view").getByRole("button", { name: "a-doc" }).click();
    await waitFile(page, "File A");
    record(`${label}: other tabs open beside the failed one`, sameList(await tabNames(page), ["missing.md", "a.md"]) && await page.locator(".file-viewer-error").count() === 0);
    await tab(page, "missing.md").click();
    await page.locator(".file-viewer-error").waitFor();
    record(`${label}: error returns with its tab`, await activeName(page) === "missing.md");
    await page.getByRole("button", { name: "Close missing.md", exact: true }).click();
    await settle(page, 200);
    record(`${label}: failed tab closes and A remains usable`, sameList(await tabNames(page), ["a.md"]) && await page.getByRole("heading", { name: "File A", exact: true }).count() === 1);
    record(`${label}: failed file was read at the boundary`, count(reads, "a1:docs/missing.md") >= 1, reads.join(", "));
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 8. Cap -------------------------------------------------------------------

async function flowCap(browser, skin, width) {
  const label = `${skinLabel(skin)} cap@${width}`;
  const tabs = Array.from({ length: 50 }, (_, i) => ({ id: `p${i}`, path: `docs/f${i}.md` }));
  const seed = seedRecord("conversation:a1", tabs, "p49");
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, agents: [makeAgent("a1")], seed, strict: false });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-cap-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/agent/a1`);
    await page.locator(".file-viewer-rendered").waitFor();
    reads.length = 0;
    await page.locator(".transcript-view").getByRole("button", { name: "a-doc" }).click();
    await page.locator(".file-tabs-notice").waitFor();
    await settle(page, 300);
    const notice = await page.locator(".file-tabs-notice").innerText();
    record(`${label}: refusal notice visible`, /50 open files/.test(notice), notice);
    record(`${label}: still 50 tabs and the same active tab`, await tabCount(page) === 50 && await activeName(page) === "f49.md");
    record(`${label}: no read for the refused file`, count(reads, "a1:docs/a.md") === 0, reads.join(", "));
    record(`${label}: refusal saved nothing new`, (await savedTabs(page)).owners["conversation:a1"].tabs.length === 50);
    await shot("refused");
    await page.getByRole("button", { name: "Dismiss", exact: true }).click();
    record(`${label}: notice can be dismissed`, await page.locator(".file-tabs-notice").count() === 0);
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 6. Room -------------------------------------------------------------------

async function flowRoom(browser, skin, width) {
  const label = `${skinLabel(skin)} room@${width}`;
  const agents = [makeAgent("a_ari", { name: "Ari", project: "alpha" }), makeAgent("a_bo", { name: "Bo", project: "alpha" })];
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, height: 1000, agents });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-room-${width}-${name}.png`) });
  try {
    await page.goto(`${BASE}/think-tank/tt_demo`);
    await page.getByRole("list", { name: "Discussion" }).waitFor();
    const links = page.getByRole("button", { name: "notes", exact: true });
    await links.nth(1).waitFor();
    await settle(page, 300);
    await links.nth(0).click();
    await page.locator(".file-viewer-rendered").waitFor();
    await links.nth(1).click();
    await page.getByRole("heading", { name: "Notes from src_b", exact: true }).waitFor();
    const labels = await page.evaluate(() => [...document.querySelectorAll(".file-tab")].map((el) => ({ name: el.querySelector(".file-tab-name")?.textContent, source: el.querySelector(".file-tab-source")?.textContent })));
    record(`${label}: same notes.md from two sources is two tabs`, labels.length === 2 && labels.every((l) => l.name === "notes.md"), JSON.stringify(labels));
    record(`${label}: tabs carry distinct source labels`, labels[0]?.source?.includes("Ari") && labels[1]?.source?.includes("Bo") && labels[0].source !== labels[1].source, JSON.stringify(labels));
    record(`${label}: reads use the right source ids`, sameList(reads, ["src_a:notes.md", "src_b:notes.md"]), reads.join(", "));
    record(`${label}: address names path and source`, urlParam(page, "file") === "notes.md" && urlParam(page, "source") === "src_b", page.url());
    record(`${label}: room page does not overflow`, !(await pageOverflow(page)));
    await shot("two-sources");
    await settle(page, 600);
    reads.length = 0;
    await page.reload();
    await page.getByRole("heading", { name: "Notes from src_b", exact: true }).waitFor();
    await settle(page, 500);
    record(`${label}: reload restores both tabs`, await tabCount(page) === 2 && await activeName(page) === "notes.md");
    record(`${label}: reload reads only the active source`, sameList(reads, ["src_b:notes.md"]), reads.join(", "));
    await tab(page, "Ari").click();
    await page.getByRole("heading", { name: "Notes from src_a", exact: true }).waitFor();
    record(`${label}: selecting the other source reads its own file`, reads.includes("src_a:notes.md") && urlParam(page, "source") === "src_a", reads.join(", "));
    await shot("reloaded");
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- 7. Phone --------------------------------------------------------------------

async function openPhone(page) {
  await page.goto(`${BASE}/remote.html`);
  await page.evaluate(() => {
    history.pushState(null, "", "/agent/a1");
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
  await page.getByRole("list", { name: "Conversation" }).waitFor({ timeout: 15_000 });
}

async function flowPhone(browser, skin, width) {
  const label = `${skinLabel(skin)} phone@${width}`;
  const { context, page, reads, errors } = await makeEnv(browser, { skin, width, height: 844, agents: [makeAgent("a1")], phone: true });
  const shot = (name) => page.screenshot({ path: join(OUT_DIR, `${skinLabel(skin)}-phone-${width}-${name}.png`) });
  const views = page.getByRole("tablist", { name: "Agent views" });
  const region = page.getByRole("region", { name: "File content" });
  const phoneTabs = () => region.locator(".file-tab .file-tab-name").allInnerTexts();
  const phoneActive = () => region.locator('.file-tab[data-state="active"] .file-tab-name').innerText();
  const heights = () => page.evaluate(() => [...document.querySelectorAll(".phone-file-preview :is(.file-tab-select, .file-tab-close, .file-tabs-icon)")].map((el) => ({ h: el.getBoundingClientRect().height, w: el.getBoundingClientRect().width, cls: el.className })));
  try {
    await openPhone(page);
    await views.getByRole("tab", { name: /^Files/ }).click();
    await page.getByRole("button", { name: "Open file" }).first().waitFor();
    const open = (i) => page.getByRole("button", { name: "Open file" }).nth(i).click();
    await open(0);
    await waitFile(page, "File A");
    await open(1);
    await waitFile(page, "File B");
    record(`${label}: two tracked files are two tabs`, sameList(await phoneTabs(), ["a.md", "b.md"]) && await phoneActive() === "b.md", (await phoneTabs()).join(","));
    record(`${label}: two files read once each`, sameList(reads, ["a1:docs/a.md", "a1:docs/b.md"]), reads.join(", "));
    await region.locator(".file-tab-select").filter({ hasText: "a.md" }).click();
    await waitFile(page, "File A");
    record(`${label}: selecting a tab shows its file`, await phoneActive() === "a.md");
    record(`${label}: selecting reads nothing new`, reads.length === 2, reads.join(", "));
    await shot("two-tabs");

    // A permitted link opens; a refused one is explained without a read.
    await region.locator(".file-viewer-rendered").getByRole("button", { name: "to c" }).click();
    await waitFile(page, "File C");
    record(`${label}: permitted link opens a tab`, sameList(await phoneTabs(), ["a.md", "b.md", "c.md"]) && count(reads, "a1:docs/c.md") === 1, (await phoneTabs()).join(","));
    await region.locator(".file-tab-select").filter({ hasText: "a.md" }).click();
    await waitFile(page, "File A");
    const before = reads.length;
    await region.locator(".file-viewer-rendered").getByRole("button", { name: "outside" }).click();
    const alert = page.getByRole("alert").filter({ hasText: "not one of this agent's tracked files" });
    await alert.waitFor();
    await settle(page, 200);
    record(`${label}: refused link shows a visible alert`, await alert.isVisible() && (await alert.innerText()).includes("outside.md"), await alert.innerText());
    record(`${label}: refused link issues no read`, reads.length === before && !reads.some((r) => r.includes("outside")), reads.join(", "));
    record(`${label}: refused link adds no tab`, (await phoneTabs()).length === 3);
    await shot("refusal");

    // Close and reopen.
    await region.getByRole("button", { name: "Close b.md", exact: true }).click();
    record(`${label}: closing a background tab keeps the active one`, sameList(await phoneTabs(), ["a.md", "c.md"]) && await phoneActive() === "a.md");
    await open(1);
    await waitFile(page, "File B");
    record(`${label}: reopened tab is active at the end`, sameList(await phoneTabs(), ["a.md", "c.md", "b.md"]) && await phoneActive() === "b.md");

    const sizes = await heights();
    record(`${label}: tab controls are at least 44px tall`, sizes.length > 0 && sizes.every((s) => s.h >= 43.5), JSON.stringify(sizes.filter((s) => s.h < 43.5)));
    record(`${label}: close and move controls are at least 44px wide`, sizes.filter((s) => /close|icon/.test(s.cls)).every((s) => s.w >= 43.5));
    record(`${label}: no horizontal page overflow`, !(await pageOverflow(page)));

    // Chat and back: the set returns.
    const setBefore = await phoneTabs();
    await views.getByRole("tab", { name: "Chat" }).click();
    await page.getByRole("list", { name: "Conversation" }).waitFor();
    await views.getByRole("tab", { name: /^Files/ }).click();
    await region.waitFor();
    record(`${label}: Chat and back restores the set`, sameList(await phoneTabs(), setBefore) && await phoneActive() === "b.md", `${(await phoneTabs()).join(",")}`);

    // Reload restores the set and reads only the active file.
    await settle(page, 600);
    reads.length = 0;
    await openPhone(page);
    await views.getByRole("tab", { name: /^Files/ }).click();
    await region.waitFor();
    await waitFile(page, "File B");
    await settle(page, 500);
    record(`${label}: reload restores the set and active tab`, sameList(await phoneTabs(), setBefore) && await phoneActive() === "b.md", (await phoneTabs()).join(","));
    record(`${label}: reload reads only the active file`, sameList(reads, ["a1:docs/b.md"]), reads.join(", "));
    record(`${label}: no horizontal overflow after reload`, !(await pageOverflow(page)));
    await shot("reloaded");
    record(`${label}: no uncaught page errors`, errors.length === 0, errors.join(" | "));
  } catch (error) { await fail(page, label, shot, error, errors); } finally { await context.close(); }
}

// ---- Main ------------------------------------------------------------------------

async function main() {
  const server = await createServer({ server: { host: "127.0.0.1", port: PORT, strictPort: true }, logLevel: "error" });
  await server.listen();
  const browser = await chromium.launch().catch(() => chromium.launch({ executablePath: "/opt/pw-browsers/chromium" }));
  const desktop = [["main", flowMain], ["overflow", flowOverflow], ["strict-restore", flowStrictRestore],["focus", flowFocus], ["history", flowHistory], ["archive", flowArchive], ["missing", flowMissing], ["cap", flowCap], ["room", flowRoom]];
  try {
    for (const skin of SKINS) {
      if (ONLY_SKIN && skinLabel(skin) !== ONLY_SKIN) continue;
      for (const width of [1024, 1440]) {
        if (ONLY_WIDTH && width !== ONLY_WIDTH) continue;
        for (const [name, flow] of desktop) if (!ONLY_SECTION || ONLY_SECTION === name) await flow(browser, skin, width);
      }
      for (const width of [360, 390, 430]) {
        if (ONLY_WIDTH && width !== ONLY_WIDTH) continue;
        if (!ONLY_SECTION || ONLY_SECTION === "phone") await flowPhone(browser, skin, width);
      }
    }
  } finally {
    await browser.close();
    await server.close();
  }
  writeFileSync(join(OUT_DIR, "report.json"), JSON.stringify(checks, null, 2));
  console.log(`\nwrote ${checks.length} checks, ${checks.filter((c) => !c.pass).length} failing`);
  console.log(`out dir: ${OUT_DIR}`);
  if (checks.some((c) => !c.pass)) process.exitCode = 1;
}

main().catch((err) => {
  console.error(err);
  process.exitCode = 1;
});
