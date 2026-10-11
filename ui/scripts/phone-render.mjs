// Renders the phone app's agent conversation at iPhone size and writes a
// screenshot, so phone UI work can be checked against the real view (jsdom
// tests cannot see layout, INV §13). Every /api call and the event stream are
// stubbed; no Chuck server or paired device is needed.
//
//   node scripts/phone-render.mjs [transcript.json] [out.png]
//
// transcript.json is a transcript window ({events, has_more}) for agent "a1";
// without one (or with ""), a streamed reply plus a tool call is shown. The
// screenshot defaults to the system temp directory.
import { readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium, devices } from "playwright";

const [fixturePath, out = join(tmpdir(), "phone-render.png")] = process.argv.slice(2);
const historyCheck = process.argv.includes("--history-check");
const detailsCheck = process.argv.includes("--details-check");
const spinnerCheck = process.argv.includes("--spinner-check");
let skin = "";

const agent = {
  agent_id: "a1", name: "phone-render", role: "implementer", project: "my-app", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: "", running: true,
  state: "idle", detail: "", context_pct: detailsCheck ? 0.22 : 0, context_used: detailsCheck ? 44000 : undefined, context_size: detailsCheck ? 200000 : undefined, updated_at: 0, archived: false,
};
const event = (seq, type, data) => ({ agent_id: "a1", seq, type, ts: "", data });
const words = "Done. **All tests** pass now, and the build is `green` again after the fix.".split(/(?<= )/);
const history = historyCheck ? Array.from({ length: 1500 }, (_, index) => event(index + 1, "user_text", { text: `History ${index + 1}` })) : null;
const transcript = fixturePath
  ? JSON.parse(readFileSync(fixturePath, "utf8"))
  : {
      agent_id: "a1",
      has_more: false,
      events: [
        event(1, "user_text", { text: "Run the tests and tell me the status." }),
        event(2, "tool_call", { tool_call_id: "t1", name: "Bash", args: { command: "make test" } }),
        event(3, "tool_result", { tool_call_id: "t1", output: "ok  github.com/AsaphNoam/Chuck/internal/server" }),
        ...words.map((text, i) => event(4 + i, "assistant_text", { delta: text })),
      ],
    };

const server = await createServer({ server: { port: 5199, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
try {
  const context = await browser.newContext({ ...devices["iPhone 13"] });
  // Replace the event stream with one that hydrates the single agent and stays open.
  await context.addInitScript((hydrated) => {
    window.EventSource = class extends EventTarget {
      constructor() {
        super();
        window.__phoneRenderSource = this;
        setTimeout(() => {
          this.onopen?.();
          for (const data of hydrated) this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify(data) }));
        });
      }
      close() {}
    };
  }, [{ agent_id: "a1", data: agent }, { agent_id: "__hydrated__", data: { hydrated: true } }]);
  // A URL predicate, not "**/api/**", which would also catch Vite's /src/api modules.
  await context.route((url) => url.pathname.startsWith("/api/"), (route) => {
    const path = new URL(route.request().url()).pathname;
    if (detailsCheck && path === "/api/projects") return route.fulfill({ json: { "my-app": { title: "My App", color: [50, 120, 180] } } });
    if (detailsCheck && path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer" } } });
    if ((detailsCheck || spinnerCheck) && path === "/api/config") return route.fulfill({ json: { appearance_skin: skin } });
    if (detailsCheck && path === "/api/remote/runtime-options") return route.fulfill({ json: { backends: [{ id: "claude", name: "Claude", default_model: "m", models: [{ id: "m", name: "Sonnet", efforts: ["low", "medium", "high"], default_effort: "medium" }] }] } });
    if (detailsCheck && path === "/api/sessions/a1/files") return route.fulfill({ json: { agent_id: "a1", files: ["src/main.go", "src/another-file-with-a-long-name.go"].map((path) => ({ path, edit_count: 2, last_ts: "2026-10-10T00:00:00Z", has_diff: false, diff_refs: [] })) } });
    if (detailsCheck && path === "/api/sessions/a1/file") return route.fulfill({ json: { path: new URL(route.request().url()).searchParams.get("path"), content: "package main\n\nfunc main() {\n    println(\"hello\")\n}\n", truncated: false } });
    if (path === "/api/sessions/a1/transcript") {
      if (!history) return route.fulfill({ json: transcript });
      const url = new URL(route.request().url());
      const before = Number(url.searchParams.get("before_seq"));
      const since = Number(url.searchParams.get("since_seq"));
      const eligible = history.filter((item) => (!before || item.seq < before) && item.seq > since);
      return route.fulfill({ json: { agent_id: "a1", events: eligible.slice(-750), has_more: eligible.length > 750, pending_permission: null, latest_assistant: "" } });
    }
    if (path === "/api/remote/home") return route.fulfill({ json: { needs_you: [], active_runs: [] } });
    if (path === "/api/sessions/a1/prompt") return route.fulfill({ status: 404, json: { error: { code: "not_found", message: "none" } } });
    return route.fulfill({ json: {} });
  });
  const page = await context.newPage();
  page.on("pageerror", (error) => console.error(`page error: ${error.message}`));
  await page.goto("http://localhost:5199/remote.html");
  // Vite serves the desktop entry for deep links, so route inside the phone app.
  await page.evaluate(() => {
    history.pushState(null, "", "/agent/a1");
    window.dispatchEvent(new PopStateEvent("popstate"));
  });
  // Screenshot even when the conversation never appears; that view is the evidence.
  const shown = await page.getByRole("list", { name: "Conversation" }).waitFor({ timeout: 10_000 }).then(() => true, () => false);
  if (shown && historyCheck) {
    const anchor = page.getByText("History 751", { exact: true });
    await anchor.waitFor();
    await page.evaluate(() => window.scrollTo(0, 300));
    const before = await anchor.boundingBox();
    await page.evaluate(() => window.scrollTo(0, 100));
    await page.getByText("History 1", { exact: true }).waitFor();
    const after = await anchor.boundingBox();
    const drift = Math.abs((after?.y ?? 0) - (before?.y ?? 0) - 200);
    if (!before || !after || drift > 2) throw new Error(`upward history load moved the anchor by ${drift}px`);
    console.log(`automatic upward load kept the visible anchor within ${drift}px`);
  }
  await page.screenshot({ path: out, fullPage: !historyCheck });
  if (shown && spinnerCheck) {
    for (const theme of ["", "sky-grove", "studio"]) {
      skin = theme;
      await page.goto("http://localhost:5199/remote.html");
      await page.evaluate(() => {
        history.pushState(null, "", "/agent/a1");
        window.dispatchEvent(new PopStateEvent("popstate"));
      });
      await page.getByRole("list", { name: "Conversation" }).waitFor();
      await page.waitForFunction((theme) => (document.documentElement.dataset.skin || "") === theme, theme);
      const update = (data) => page.evaluate((data) => {
        window.__phoneRenderSource.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: "a1", data }) }));
      }, data);
      await update({ ...agent, state: "busy" });
      await page.getByText("Working…", { exact: true }).waitFor();
      await page.locator(".transcript-pending").scrollIntoViewIfNeeded();
      const metrics = await page.locator(".spinner").evaluate((el) => ({ duration: getComputedStyle(el).animationDuration, animation: getComputedStyle(el).animationName, overflow: document.documentElement.scrollWidth > innerWidth }));
      if (metrics.duration !== "0.7s" || metrics.animation !== "ad-spin" || metrics.overflow) throw new Error(JSON.stringify(metrics));
      await page.screenshot({ path: out.replace(/\.png$/, `-${theme || "core"}-busy.png`) });
      await page.emulateMedia({ reducedMotion: "reduce" });
      const reduced = await page.locator(".spinner").evaluate((el) => getComputedStyle(el).animationDuration);
      if (reduced !== "2.4s") throw new Error(`reduced motion: ${reduced}`);
      await page.emulateMedia({ reducedMotion: "no-preference" });
      for (const state of ["idle", "waiting_input", "error"]) {
        await update({ ...agent, state });
        await page.getByText("Working…", { exact: true }).waitFor({ state: "hidden" });
      }
      await update({ ...agent, state: "busy", running: false });
      await page.getByText("Working…", { exact: true }).waitFor({ state: "hidden" });
      console.log(`${theme || "core"} spinner: ${JSON.stringify(metrics)}, reduced=${reduced}; busy → idle/waiting/error/stopped passed`);
    }
  }
  if (shown && detailsCheck) {
    for (const theme of ["", "sky-grove", "studio"]) {
      skin = theme;
      await page.goto("http://localhost:5199/remote.html", { waitUntil: "domcontentloaded" });
      await page.evaluate(() => {
        history.pushState(null, "", "/agent/a1");
        window.dispatchEvent(new PopStateEvent("popstate"));
      });
      await page.getByRole("tab", { name: "Chat", exact: true }).waitFor();
      await page.waitForFunction((theme) => (document.documentElement.dataset.skin || "") === theme, theme);
      const prefix = out.replace(/\.png$/, `-${theme || "core"}`);
      await page.getByRole("textbox").fill("Keep typing without a blue highlight.");
      await page.getByRole("textbox").focus();
      await page.screenshot({ path: `${prefix}-chat.png`, fullPage: true, animations: "disabled" });
      const metrics = await page.evaluate(() => {
        const footer = getComputedStyle(document.querySelector(".phone-composer-actions"));
        const button = document.querySelector(".phone-send-icon");
        const send = button.getBoundingClientRect();
        const circle = getComputedStyle(button, "::before");
        const box = document.querySelector(".phone-composer-box").getBoundingClientRect();
        const field = document.querySelector(".phone-composer-box textarea");
        const metadata = document.querySelector(".phone-model-switch");
        const metaStyle = getComputedStyle(metadata);
        const metaBox = metadata.getBoundingClientRect();
        return { footerBorder: footer.borderWidth, sendWidth: send.width, sendHeight: send.height, circleWidth: send.width - parseFloat(circle.left) - parseFloat(circle.right), circleHeight: send.height - parseFloat(circle.top) - parseFloat(circle.bottom), overflow: document.documentElement.scrollWidth > innerWidth, typingOutline: getComputedStyle(field).outlineStyle, textboxBorder: getComputedStyle(field).borderWidth, metadataAboveTextbox: metaBox.bottom <= box.top && !document.querySelector(".phone-composer-box").contains(metadata), metadataAlignment: Math.abs(metaBox.right - box.right) <= 1 ? "right" : "left", metadataBordered: metaStyle.borderTopWidth !== "0px" && metaStyle.boxShadow !== "none", sendInside: send.bottom <= box.bottom && send.right <= box.right };
      });
      if (metrics.footerBorder !== "0px" || metrics.sendWidth !== 44 || metrics.sendHeight !== 44 || metrics.circleWidth !== 44 || metrics.circleHeight !== 28 || metrics.overflow) throw new Error(`${theme}: ${JSON.stringify(metrics)}`);
      if (metrics.typingOutline !== "none" || metrics.textboxBorder !== "0px" || !metrics.metadataAboveTextbox || metrics.metadataAlignment !== "right" || !metrics.metadataBordered || !metrics.sendInside) throw new Error(`${theme} composer: ${JSON.stringify(metrics)}`);
      await page.getByRole("tab", { name: "Files", exact: true }).click();
      await page.getByRole("button", { name: "Open file", exact: true }).first().click();
      await page.getByRole("region", { name: "File content" }).getByText("package main", { exact: false }).waitFor();
      await page.screenshot({ path: `${prefix}-files.png`, fullPage: true, animations: "disabled" });
      await page.getByRole("tab", { name: "Manage", exact: true }).click();
      await page.getByRole("button", { name: "Switch runtime", exact: true }).waitFor();
      await page.screenshot({ path: `${prefix}-manage.png`, fullPage: true, animations: "disabled" });
      for (const width of [360, 390, 430]) {
        await page.setViewportSize({ width, height: 844 });
        await page.evaluate(() => {
          history.pushState(null, "", "/agent/a1");
          window.dispatchEvent(new PopStateEvent("popstate"));
        });
        await page.getByRole("heading", { level: 1 }).waitFor();
        await page.getByRole("tab", { name: "Chat", exact: true }).click();
        const header = await page.evaluate(() => {
          const persona = document.querySelector(".phone-agent-title .phone-eyebrow").getBoundingClientRect();
          const title = document.querySelector(".phone-agent-identity h1").getBoundingClientRect();
          const back = document.querySelector(".phone-agent-back");
          const backBox = back.getBoundingClientRect();
          const tabs = document.querySelector(".phone-tabs").getBoundingClientRect();
          const action = document.querySelector(".phone-agent-tabs > button");
          const actionBox = action.getBoundingClientRect();
          const meter = document.querySelector("[data-ui=context-meter]").getBoundingClientRect();
          return { backText: back.textContent, backAlignment: Math.abs(backBox.y + backBox.height / 2 - persona.y - persona.height / 2) + (persona.bottom <= title.top ? 0 : 99), actionAlignment: Math.abs(actionBox.y + actionBox.height / 2 - tabs.y - tabs.height / 2), actionOutsideTabs: !action.closest("[role=tablist]"), contextBetween: meter.top > title.bottom && meter.bottom <= tabs.top, overflow: document.documentElement.scrollWidth > innerWidth };
        });
        if (header.backText || header.backAlignment > 1 || header.actionAlignment > 1 || !header.actionOutsideTabs || !header.contextBetween || header.overflow) throw new Error(`${theme}/${width} header: ${JSON.stringify(header)}`);
        await page.screenshot({ path: `${prefix}-header-${width}.png`, animations: "disabled" });
        await page.evaluate(() => {
          history.pushState(null, "", "/project/my-app");
          window.dispatchEvent(new PopStateEvent("popstate"));
        });
        await page.getByRole("button", { name: "New agent", exact: true }).click();
        const dialog = page.getByRole("dialog", { name: "New agent", exact: true });
        await dialog.waitFor();
        const center = await dialog.evaluate((el) => {
          const box = el.getBoundingClientRect();
          return { x: Math.abs(box.x + box.width / 2 - innerWidth / 2), y: Math.abs(box.y + box.height / 2 - innerHeight / 2), overflow: document.documentElement.scrollWidth > innerWidth };
        });
        if (center.x > 1 || center.y > 1 || center.overflow) throw new Error(`${theme}/${width} dialog: ${JSON.stringify(center)}`);
        await page.screenshot({ path: `${prefix}-new-agent-${width}.png`, animations: "disabled" });
        await dialog.getByRole("button", { name: "Close dialog", exact: true }).click();
      }
      await page.setViewportSize({ width: 390, height: 844 });
      console.log(`${theme || "core"} details: ${JSON.stringify(metrics)}`);
    }
  }
  console.log(`wrote ${out}`);
  if (!shown) {
    console.error("the conversation did not render; see the screenshot");
    process.exitCode = 1;
  }
} finally {
  await browser.close();
  await server.close();
}
