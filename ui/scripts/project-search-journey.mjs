// Real retained-source journey. Build/embed first, then start TestProjectSearchBrowserFixture
// with CHUCK_SEARCH_BROWSER_READY=/tmp/search-ready.json. Run:
// node ui/scripts/project-search-journey.mjs /tmp/search-ready.json /tmp/search-browser
// The ready fixture supplies independently authored cases: {name,query,agent_id,expected}.
// Controlled read-failure branches are labelled separately from real server/source checks.
import assert from "node:assert/strict";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const fixture = JSON.parse(readFileSync(process.argv[2], "utf8"));
const out = process.argv[3] || "/tmp/chuck-project-search-browser";
mkdirSync(out, { recursive: true });
const base = fixture.desktop;
const project = fixture.project || "search-project";
const checks = [], errors = [], lifecycle = [];
let liveSearchResult;
const browser = await chromium.launch();
const record = (name, detail) => { checks.push({ name, pass: true, detail }); console.log(`PASS ${name}`); };
const api = async (path, body, method = body === undefined ? "GET" : "POST") => {
  const response = await fetch(base + path, { method, headers: { "Content-Type": "application/json" }, body: body === undefined ? undefined : JSON.stringify(body) });
  assert(response.ok, `${path}: ${response.status} ${await response.clone().text()}`);
  return response.json();
};
const cases = fixture.cases;
assert(Array.isArray(cases) && cases.length, "ready fixture must supply independently authored cases");
const originalConfig = await api("/api/config");
let context;
async function search(page, query) {
  await page.goto(`${base}/project/${encodeURIComponent(project)}`);
  const field = page.getByRole("combobox", { name: "Search chats", exact: true });
  await field.fill(query);
  await page.locator(".project-chat-popup").waitFor();
  await page.getByRole("option").first().waitFor();
  return field;
}
async function target(page, item) {
  await page.locator("[data-search-passage]").first().waitFor();
  const marked = await page.locator("[data-search-passage]").allTextContents();
  assert(marked.join(" ").includes(item.expected), `expected independent passage ${JSON.stringify(item.expected)}, received ${JSON.stringify(marked)}`);
  const inView = await page.locator("[data-search-passage]").first().evaluate((element) => {
    const box = element.getBoundingClientRect();
    const list = element.closest(".transcript-view")?.getBoundingClientRect();
    return Boolean(list && box.bottom > list.top && box.top < list.bottom);
  });
  assert(inView, "matched passage is visible inside transcript reading viewport");
  assert.equal(await page.locator("html").evaluate((element) => element.scrollWidth > innerWidth), false, "no horizontal page overflow");
}
try {
  for (const width of [1024, 1440]) for (const skin of ["", "sky-grove", "studio", "boring-night"]) {
    const label = `${width}-${skin || "core"}`;
    await api("/api/config", { appearance_skin: skin }, "PUT");
    context = await browser.newContext({ viewport: { width, height: 900 } });
    const page = await context.newPage();
    page.on("pageerror", (error) => errors.push(`${label}: ${error.message}`));
    page.on("request", (request) => {
      if (request.method() !== "GET" && /\/api\/sessions\/(?:[^/]+\/)?(?:resume|restore|archive|stop|start)(?:\?|$)/.test(new URL(request.url()).pathname)) lifecycle.push(request.url());
    });
    const first = cases[0];
    const field = await search(page, first.query);
    await page.screenshot({ path: join(out, `${label}-dropdown.png`), fullPage: true });
    assert.equal(await page.locator("html").getAttribute("data-skin") || "", skin);
    // Navigate entirely by keyboard, selecting the independently named target row.
    const options = page.getByRole("option");
    const texts = await options.allTextContents();
    const wanted = texts.findIndex((text) => text.includes(first.name));
    assert(wanted >= 0, `fixture row ${first.name} appears`);
    for (let index = 0; index < wanted; index++) await field.press("ArrowDown");
    await field.press("Enter");
    await page.waitForURL(/\/(agent|archive)\/[^?]+\?.*search_doc=/);
    await target(page, first);
    const exactURL = page.url();
    await page.screenshot({ path: join(out, `${label}-passage.png`), fullPage: true });
    await page.reload();
    await target(page, first);
    await page.goBack();
    await page.goForward();
    await target(page, first);
    assert.equal(page.url(), exactURL);
    record(`${label}: keyboard source reveal, reload/back, bounded layout`);

    // Every source kind and lifecycle route is exercised at both desktop sizes.
    if (!skin) for (const item of cases.slice(1)) {
      await search(page, item.query);
      await page.getByRole("option").filter({ hasText: item.name }).getByRole("button").click();
      await target(page, item);
      if (item.route) assert.equal(new URL(page.url()).pathname, `/${item.route}/${item.agent_id}`);
      await page.reload();
      await target(page, item);
      await page.screenshot({ path: join(out, `${label}-${item.key || item.agent_id}.png`), fullPage: true });
      record(`${label}: ${item.key || item.name} exact retained source`);
    }
    await context.close(); context = undefined;
  }

  // Controlled wire failures exercise recovery; all underlying successful data remains real.
  context = await browser.newContext({ viewport: { width: 1024, height: 900 } });
  const page = await context.newPage();
  const first = cases[0];
  // Real fake-provider streaming must not pull a selected live passage back
  // to the newest reply. The independently known target is a retained prompt.
  if (fixture.live_agent) {
    const live = fixture.live_agent;
    await api(`/api/sessions/${live}/prompt`, { text: "livepassageneedle " + "Retained discussion. ".repeat(120) });
    let liveResult;
    for (let attempt = 0; attempt < 60; attempt++) {
      const hits = await api(`/api/archive/projects/${encodeURIComponent(project)}?view=project_search&q=livepassageneedle`);
      liveResult = hits.results.find((row) => row.agent_id === live && row.search_ref);
      if (liveResult) break;
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    assert(liveResult, "completed live turn becomes searchable");
    liveSearchResult = liveResult;
    await search(page, "livepassageneedle");
    await page.getByRole("option").filter({ hasText: liveResult.name }).getByRole("button").click();
    await target(page, { expected: "livepassageneedle" });
    const before = await page.locator("[data-search-passage]").first().evaluate((element) => ({ top: element.getBoundingClientRect().top, scroll: element.closest(".transcript-view").scrollTop }));
    await api(`/api/sessions/${live}/prompt`, { text: "New unrelated live reply. ".repeat(200) });
    await page.getByText("New unrelated live reply.", { exact: false }).first().waitFor();
    await page.waitForTimeout(700);
    const after = await page.locator("[data-search-passage]").first().evaluate((element) => ({ top: element.getBoundingClientRect().top, scroll: element.closest(".transcript-view").scrollTop }));
    assert(Math.abs(before.top - after.top) < 2 && Math.abs(before.scroll - after.scroll) < 2, "new live content preserves selected passage reading position");
    await page.screenshot({ path: join(out, "live-reading.png"), fullPage: true });
    record("real live turn search and new reply preserve reading position");
  }
  let fail = true;
  const archivePattern = "**/api/archive/projects/**";
  await page.route(archivePattern, async (route) => {
    if (fail && new URL(route.request().url()).searchParams.get("view") === "project_search") {
      fail = false;
      await route.fulfill({ status: 503, json: { error: { message: "Controlled temporary search failure" } } });
    } else await route.continue();
  });
  await page.goto(`${base}/project/${encodeURIComponent(project)}`);
  await page.getByRole("combobox", { name: "Search chats" }).fill(first.query);
  await page.getByRole("alert").filter({ hasText: "Controlled temporary" }).waitFor();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await page.getByRole("option").filter({ hasText: first.name }).waitFor();
  record("controlled temporary read failure: query retained and Retry recovers");
  await page.unroute(archivePattern);

  const referenceRows = await api(`/api/archive/projects/${encodeURIComponent(project)}?view=project_search&q=${encodeURIComponent(first.query)}&limit=20&offset=0`);
  // Controlled 20-row popup proves keyboard selection remains visible at both
  // scroll boundaries while focus stays in the combobox.
  const seed = referenceRows.results.find((row) => row.search_ref) || referenceRows.results[0];
  const manyRows = Array.from({ length: 20 }, (_, index) => ({ ...seed, agent_id: `keyboard-${index}`, name: `Keyboard result ${index}` }));
  await page.route(archivePattern, async (route) => {
    if (new URL(route.request().url()).searchParams.get("view") === "project_search") {
      await route.fulfill({ json: { results: manyRows, total: manyRows.length, search_mode: "full_text" } });
    } else await route.continue();
  });
  await page.goto(`${base}/project/${encodeURIComponent(project)}`);
  const keyboardField = page.getByRole("combobox", { name: "Search chats" });
  await keyboardField.fill("keyboardfixture");
  const keyboardPopup = page.locator(".project-chat-popup");
  await keyboardPopup.waitFor();
  await page.getByRole("option", { name: /Keyboard result 19/ }).waitFor();
  const assertActiveVisible = async (index) => {
    const option = page.getByRole("option", { name: `Keyboard result ${index}` });
    assert.equal(await option.getAttribute("aria-selected"), "true");
    const [rowBox, popupBox, focused] = await Promise.all([
      option.boundingBox(), keyboardPopup.boundingBox(), keyboardField.evaluate((element) => document.activeElement === element),
    ]);
    assert(rowBox && popupBox && rowBox.y >= popupBox.y && rowBox.y + rowBox.height <= popupBox.y + popupBox.height,
      `active keyboard option ${index} is inside visible popup`);
    assert(focused, "keyboard focus remains in search field");
  };
  for (let index = 0; index < 15; index++) await keyboardField.press("ArrowDown");
  await assertActiveVisible(15);
  await keyboardField.press("ArrowUp");
  await assertActiveVisible(14);
  await keyboardField.press("ArrowUp");
  await assertActiveVisible(13);
  // Wrap from the first row to the last, then back, across the scroll boundary.
  for (let index = 0; index < 7; index++) await keyboardField.press("ArrowDown");
  await assertActiveVisible(0);
  await keyboardField.press("ArrowUp");
  await assertActiveVisible(19);
  await keyboardField.press("ArrowDown");
  await assertActiveVisible(0);
  record("controlled 20-result keyboard selection stays visible through down/up and wrapping");
  await page.unroute(archivePattern);

  if (liveSearchResult) {
    const ref = liveSearchResult.search_ref;
    const params = new URLSearchParams({ search_doc: ref.document_id, search_rev: ref.content_sha256, search_start: String(ref.start_byte), search_end: String(ref.end_byte) });
    await page.route("**/api/sessions/*/search-match?*", (route) => route.fulfill({ json: { status: "unavailable", reason: "Controlled retained source unavailable", ranges: [] } }));
    await page.goto(`${base}/${liveSearchResult.archived ? "archive" : "agent"}/${liveSearchResult.agent_id}?${params}`);
    await page.getByRole("button", { name: "Open chat normally", exact: true }).click();
    const assertFollowing = async (label) => {
      await page.waitForFunction(() => {
        const list = document.querySelector(".transcript-view");
        return Boolean(list && list.scrollHeight - list.scrollTop - list.clientHeight < 24);
      });
      assert.equal(await page.getByRole("button", { name: "Jump to latest" }).count(), 0, `${label}: latest-follow action is hidden`);
    };
    await assertFollowing("normal opening");
    await api(`/api/sessions/${liveSearchResult.agent_id}/prompt`, { text: "Recovery follow-up reply. ".repeat(120) });
    await page.getByText("Recovery follow-up reply.", { exact: false }).first().waitFor();
    let recoveryTurnIndexed = false;
    for (let attempt = 0; attempt < 60; attempt++) {
      const completed = await api(`/api/archive/projects/${encodeURIComponent(project)}?view=project_search&q=recovery%20follow-up`);
      if (completed.results.some((row) => row.agent_id === liveSearchResult.agent_id)) { recoveryTurnIndexed = true; break; }
      await new Promise((resolve) => setTimeout(resolve, 200));
    }
    assert(recoveryTurnIndexed, "later recovery reply turn completes and is indexed");
    await assertFollowing("later live reply");
    record("unavailable search recovery restores latest following through a later live reply");
    await page.unroute("**/api/sessions/*/search-match?*");
  }

  const result = referenceRows.results.find((row) => row.agent_id === first.agent_id);
  assert(result?.search_ref, "real server supplies reference for recovery branch");
  const ref = result.search_ref;
  const query = new URLSearchParams({ search_doc: ref.document_id, search_rev: ref.content_sha256, search_start: String(ref.start_byte), search_end: String(ref.end_byte), file: "retained-reader.md", tab: "transcript" });
  await page.route("**/api/sessions/*/search-match?*", (route) => route.fulfill({ json: { status: "unavailable", reason: "Controlled retained source unavailable", ranges: [] } }));
  await page.goto(`${base}/${result.archived ? "archive" : "agent"}/${first.agent_id}?${query}`);
  await page.getByRole("button", { name: "Open chat normally", exact: true }).click();
  assert.equal(new URL(page.url()).searchParams.get("file"), "retained-reader.md");
  assert.equal(new URL(page.url()).searchParams.has("search_doc"), false);
  assert.equal(await page.locator("[data-search-passage]").count(), 0);
  record("controlled unavailable source: explicit normal opening preserves file-reader parameters");
  await page.unroute("**/api/sessions/*/search-match?*");

  await page.route(archivePattern, async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    await route.fulfill({ response, json: { ...body, search_mode: "metadata", results: body.results.map((row) => ({ ...row, snippet: undefined, search_ref: undefined, location_status: "metadata_only" })) } });
  });
  await search(page, first.query);
  await page.getByRole("option").filter({ hasText: first.name }).getByRole("button").click();
  assert.equal(new URL(page.url()).searchParams.has("search_doc"), false);
  assert.equal(await page.locator("[data-search-passage]").count(), 0);
  record("controlled metadata-only build shape: normal chat opening without passage claim");
  assert.deepEqual(lifecycle, [], "search navigation sends no lifecycle mutation");
  assert.deepEqual(errors, [], "no browser runtime errors");
  record("all result opens preserve session lifecycle");
} catch (error) {
  checks.push({ name: "journey exception", pass: false, detail: error.stack || String(error) });
  console.error(error);
  process.exitCode = 1;
} finally {
  if (context) await context.close();
  await api("/api/config", { appearance_skin: originalConfig.appearance_skin || "" }, "PUT");
  await browser.close();
  writeFileSync(join(out, "report.json"), JSON.stringify({ fixture: { desktop: base, home: fixture.home }, checks, errors, lifecycle, limitations: ["Controlled error/metadata/unavailable branches use intercepted reads.", "Live delivery uses the deterministic fake provider, without credentials or native-provider claims."] }, null, 2));
}
