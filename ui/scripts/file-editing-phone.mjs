import assert from "node:assert/strict";
import { mkdirSync } from "node:fs";
import { chromium } from "playwright";

// Real phone bundle + real fixture file GETs. SSE hydration and tracked-file
// admission are fixture-controlled; this does not prove tailnet pairing.
const base = process.argv[2] ?? "http://127.0.0.1:4426";
const vite = process.argv[3] ?? "http://127.0.0.1:5173";
const path = process.argv[4] ?? "/private/tmp/chuck-file-editing-proof/source.md";
const out = "/private/tmp/chuck-file-editing-phone";
mkdirSync(out, { recursive: true });
const agents = await (await fetch(`${base}/api/sessions`)).json();
const agent = agents.find((item) => item.running);
assert.ok(agent, "fixture has a live agent");
const browser = await chromium.launch();
try {
  for (const width of [360, 390, 430]) {
    const context = await browser.newContext({ viewport: { width, height: 844 }, isMobile: true, serviceWorkers: "block" });
    await context.addInitScript(({ agent, path }) => {
      localStorage.setItem("chuck.paired", "1");
      history.replaceState(null, "", `/agent/${agent.agent_id}?file=${encodeURIComponent(path)}`);
      window.EventSource = class extends EventTarget {
        constructor() {
          super(); setTimeout(() => {
            this.onopen?.();
            this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: agent.agent_id, data: { ...agent, running: true, state: "idle", context_pct: 0 } }) }));
            this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: "__hydrated__", data: { hydrated: true } }) }));
          }, 50);
        }
        close() {}
      };
    }, { agent, path });
    const page = await context.newPage(); page.setDefaultTimeout(12000);
    page.on("pageerror", (error) => console.error("PHONE ERROR", error.message));
    let puts = 0;
    await page.route("**/api/**", async (route) => {
      const request = route.request();
      const url = new URL(request.url());
      if (!url.pathname.startsWith("/api/")) return route.continue();
      if (request.method() === "PUT" && url.pathname.endsWith("/file")) puts++;
      if (url.pathname === `/api/sessions/${agent.agent_id}/files`) return route.fulfill({ json: { agent_id: agent.agent_id, files: [{ path, edit_count: 1, last_ts: new Date().toISOString(), has_diff: false, diff_refs: [] }] } });
      const response = await route.fetch({ url: `${base}${url.pathname}${url.search}` });
      await route.fulfill({ response });
    });
    await page.goto(`${vite}/remote.html`);
    try { await page.getByRole("tab", { name: /^Files/ }).click(); }
    catch (error) { console.error(await page.locator("body").innerText()); await page.screenshot({ path: `${out}/failure-${width}.png`, fullPage: true }); throw error; }
    if (!(await page.locator(".file-viewer-body").count())) await page.getByRole("button", { name: "Open file", exact: true }).click();
    await page.locator(".file-viewer-body").waitFor();
    assert.equal(await page.getByRole("button", { name: "Edit", exact: true }).count(), 0);
    assert.equal(await page.getByRole("button", { name: "Save", exact: true }).count(), 0);
    assert.equal(puts, 0);
    await page.screenshot({ path: `${out}/phone-${width}.png`, fullPage: true });
    console.log(`PASS real phone bundle read-only at ${width}px; fixture GET content; no editor/save/PUT`);
    await page.unrouteAll({ behavior: "ignoreErrors" });
    await context.close();
  }
} finally { await browser.close(); }
