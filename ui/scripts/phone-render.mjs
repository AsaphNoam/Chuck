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
const parityCheck = process.argv.includes("--parity-check");
const scrollCheck = process.argv.includes("--scroll-check");
const recoveryCheck = process.argv.includes("--recovery-check");
// FS-13.A19–A21/FS-20.A27: phone capture, tray, launch and cards (stubbed API).
const annotationCheck = process.argv.includes("--annotation-check");
const longPath = "docs/features/a-very-long-directory-name/with-nested-folders/release-notes-for-the-phone.md";
const fileReads = [];
const annotationPosts = [];
const launches = [];
let healthCount = 0;
let healthDown = false;
let skin = "";
let slowPages = false;
const activeRuns = Array.from({ length: 20 }, (_, i) => ({ kind: "run", id: `r${i}`, title: `Pipeline ${i}`, project: "my-app", state: "running", reason: "Stage in progress", stage_number: 1, stage_count: 3, since: "" }));

const agent = {
  agent_id: "a1", name: "phone-render", role: "implementer", project: "my-app", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: "", running: true,
  state: "idle", detail: "", group: detailsCheck ? "Feature Changes" : undefined, context_pct: detailsCheck ? 0.22 : 0, context_used: detailsCheck ? 44000 : undefined, context_size: detailsCheck ? 200000 : undefined, updated_at: 0, archived: false,
};
const event = (seq, type, data) => ({ agent_id: "a1", seq, type, ts: "", data });
const words = "Done. **All tests** pass now, and the build is `green` again after the fix.".split(/(?<= )/);
const history = historyCheck || scrollCheck || recoveryCheck ? Array.from({ length: 1500 }, (_, index) => event(index + 1, "user_text", { text: `History ${index + 1}` })) : null;
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
        ...(annotationCheck ? [
          event(40, "annotation", {
            annotations: [
              { seq: 1, excerpt: "Run the tests", instruction: "Also run the race tests." },
              { anchor_kind: "file", path: longPath, start_line: 3, end_line: 4, excerpt: "The phone can now annotate files.\nEach selection keeps its lines.", instruction: "Shorten this." },
              { anchor_kind: "file", path: longPath, excerpt: "Release notes", instruction: "Rename the heading." },
            ],
            overall_instruction: "Keep it to one commit.",
            target: { kind: "self" },
          }),
          event(41, "user_text", { text: "[Chuck annotations]\nTranscript event 1\nmachine prompt that must stay hidden" }),
        ] : []),
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
        window.__phoneRenderStreams = (window.__phoneRenderStreams ?? 0) + 1;
        setTimeout(() => {
          this.onopen?.();
          for (const data of hydrated) this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify(data) }));
        });
      }
      close() {}
    };
  }, [{ agent_id: "a1", data: agent }, { agent_id: "__hydrated__", data: { hydrated: true } }]);
  // A URL predicate, not "**/api/**", which would also catch Vite's /src/api modules.
  await context.route((url) => url.pathname.startsWith("/api/"), async (route) => {
    const path = new URL(route.request().url()).pathname;
    if (slowPages && ["/api/projects", "/api/remote/home"].includes(path)) await new Promise((resolve) => setTimeout(resolve, 400));
    if ((detailsCheck || scrollCheck) && path === "/api/projects") return route.fulfill({ json: { "my-app": { title: "My App", color: [50, 120, 180] } } });
    if (detailsCheck && path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer" } } });
    if ((detailsCheck || spinnerCheck || parityCheck) && path === "/api/config") {
      if (route.request().method() === "PUT") skin = route.request().postDataJSON().appearance_skin;
      return route.fulfill({ json: { appearance_skin: skin } });
    }
    if (parityCheck && path === "/api/remote/self") return route.fulfill({ json: { id: "d1", name: "iPhone", notifications: "off", vapid_public_key: "" } });
    if (detailsCheck && path === "/api/remote/runtime-options") return route.fulfill({ json: { backends: [{ id: "claude", name: "Claude", default_model: "m", models: [{ id: "m", name: "Sonnet", efforts: ["low", "medium", "high"], default_effort: "medium" }] }] } });
    if (detailsCheck && path === "/api/sessions/a1/files") return route.fulfill({ json: { agent_id: "a1", files: ["src/main.go", "src/another-file-with-a-long-name.go"].map((path) => ({ path, edit_count: 2, last_ts: "2026-10-10T00:00:00Z", has_diff: false, diff_refs: [] })) } });
    if (detailsCheck && path === "/api/sessions/a1/file") return route.fulfill({ json: { path: new URL(route.request().url()).searchParams.get("path"), content: "package main\n\nfunc main() {\n    println(\"hello\")\n}\n", truncated: false, line_count: 5, language: "go" } });
    if (annotationCheck) {
      const method = route.request().method();
      if (path === "/api/projects") return route.fulfill({ json: { "my-app": { title: "My App", color: [50, 120, 180] }, other: { title: "Other App", color: [180, 120, 50] } } });
      if (path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer" }, reviewer: { title: "Reviewer" } } });
      if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin } });
      if (path === "/api/remote/runtime-options") return route.fulfill({ json: { backends: [{ id: "claude", name: "Claude", default: true, default_model: "m", models: [{ id: "m", name: "Sonnet", efforts: [], default_effort: "" }] }] } });
      if (path === "/api/sessions/a1/files") return route.fulfill({ json: { agent_id: "a1", files: [{ path: longPath, edit_count: 3, last_ts: "2026-10-10T00:00:00Z", has_diff: false, diff_refs: [] }] } });
      if (path === "/api/sessions/a1/file") {
        fileReads.push(new URL(route.request().url()).searchParams.get("path"));
        return route.fulfill({ json: { agent_id: "a1", path: longPath, size: 1, mod_time: "", line_count: 6, truncated: false, language: "markdown", content: "# Release notes\n\nThe phone can now annotate files.\nEach selection keeps its lines.\n\nDone." } });
      }
      if (path === "/api/sessions/a3/transcript") return route.fulfill({ json: { agent_id: "a3", events: [], has_more: false } });
      if (path === "/api/sessions/a1/annotations/validate") return route.fulfill({ json: { valid: true } });
      if (path === "/api/sessions/a1/annotations" && method === "POST") {
        annotationPosts.push(route.request().postDataJSON());
        return route.fulfill({ status: 202, json: { accepted: true, seq: 99 } });
      }
      if (path === "/api/sessions" && method === "POST") {
        launches.push(route.request().postDataJSON());
        return route.fulfill({ status: 201, json: { agent: { ...agent, agent_id: "a3", name: "fresh" } } });
      }
    }
    if (path === "/api/health") {
      healthCount++;
      if (healthDown) return route.abort();
    }
    if (path === "/api/sessions/a1/transcript") {
      if (!history) return route.fulfill({ json: transcript });
      const url = new URL(route.request().url());
      const before = Number(url.searchParams.get("before_seq"));
      const since = Number(url.searchParams.get("since_seq"));
      const eligible = history.filter((item) => (!before || item.seq < before) && item.seq > since);
      return route.fulfill({ json: { agent_id: "a1", events: eligible.slice(-750), has_more: eligible.length > 750, pending_permission: null, latest_assistant: "" } });
    }
    if (path === "/api/remote/home") return route.fulfill({ json: { needs_you: scrollCheck ? activeRuns : [], active_runs: scrollCheck ? activeRuns : [] } });
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
  if (shown && scrollCheck) {
    // FS-20.A26: chats and other screens return to where the person left them.
    const go = async (to) => {
      await page.evaluate((path) => { history.pushState(null, "", path); window.dispatchEvent(new PopStateEvent("popstate")); }, to);
      await page.waitForTimeout(150);
    };
    const settle = () => page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const atNewest = () => page.evaluate(() => window.scrollY + innerHeight >= document.documentElement.scrollHeight - 2);
    const topTurn = () => page.evaluate(() => {
      const top = document.querySelector(".phone-header").getBoundingClientRect().bottom;
      const turn = [...document.querySelectorAll("[data-place]")].find((item) => item.getBoundingClientRect().bottom > top);
      return { key: turn.dataset.place, top: turn.getBoundingClientRect().top };
    });
    await page.getByText("History 1500", { exact: true }).waitFor();
    await settle();
    if (!(await atNewest())) throw new Error("first open did not show the newest message");
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight / 2));
    await settle();
    const before = await topTurn();
    await go("/project/my-app");
    await page.getByRole("button", { name: "New agent", exact: true }).waitFor();
    await go("/agent/a1");
    await settle();
    const after = await topTurn();
    if (after.key !== before.key || Math.abs(after.top - before.top) > 2) throw new Error(`chat reopened at ${JSON.stringify(after)}, left at ${JSON.stringify(before)}`);
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await settle();
    await go("/");
    await go("/agent/a1");
    await settle();
    if (!(await atNewest())) throw new Error("chat left at newest did not reopen at newest");
    await page.setViewportSize({ width: 390, height: 360 });
    await go("/project/my-app");
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
    await settle();
    const projectY = await page.evaluate(() => window.scrollY);
    if (projectY < 50) throw new Error(`project page too short to check scroll (${projectY}px)`);
    await go("/agent/a1");
    await go("/project/my-app");
    await settle();
    const projectAfter = await page.evaluate(() => window.scrollY);
    if (Math.abs(projectAfter - projectY) > 1) throw new Error(`project page returned at ${projectAfter}px, left at ${projectY}px`);
    await page.setViewportSize({ width: 390, height: 844 });
    // A live update while away re-keys these pages' queries, so they reload
    // empty and slowly; the saved place must still be restored.
    const revisited = [];
    for (const route of ["/project/my-app", "/"]) {
      await go(route);
      await page.getByText("Pipeline 19", { exact: true }).first().waitFor();
      await page.evaluate(() => window.scrollTo(0, 1000));
      await settle();
      const left = await page.evaluate(() => window.scrollY);
      if (left < 500) throw new Error(`${route} too short to check live-update scroll (${left}px)`);
      await go("/agent/a1");
      slowPages = true;
      await page.evaluate((data) => window.__phoneRenderSource.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify(data) })), { agent_id: "a1", data: agent });
      await go(route);
      await page.getByText("Pipeline 19", { exact: true }).first().waitFor();
      await settle();
      slowPages = false;
      const back = await page.evaluate(() => window.scrollY);
      if (Math.abs(back - left) > 1) throw new Error(`${route} returned at ${back}px after a live update, left at ${left}px`);
      revisited.push(`${route} ${back}px`);
    }
    console.log(`scroll memory: chat row ${before.key} within ${Math.abs(after.top - before.top)}px, newest kept, project at ${projectAfter}px, after live update ${revisited.join(", ")}`);
  }
  if (shown && recoveryCheck) {
    // FS-20.A23/A25 at phone width: a healthy stream survives return signals;
    // a lost one recovers in place; an outage waits for capped backoff.
    const settle = () => page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const streams = () => page.evaluate(() => window.__phoneRenderStreams);
    const topTurn = () => page.evaluate(() => {
      const top = document.querySelector(".phone-header").getBoundingClientRect().bottom;
      const turn = [...document.querySelectorAll("[data-place]")].find((item) => item.getBoundingClientRect().bottom > top);
      return { key: turn.dataset.place, top: turn.getBoundingClientRect().top };
    });
    const draft = "Unsent while the Mac reconnects";
    await page.getByText("History 1500", { exact: true }).waitFor();
    // iOS Safari has no CSS scroll anchoring; without this Chromium hides a shift.
    await page.addStyleTag({ content: "* { overflow-anchor: none !important; }" });
    await page.getByRole("textbox").fill(draft);
    await page.getByLabel("Choose files to attach").setInputFiles({ name: "notes.txt", mimeType: "text/plain", buffer: Buffer.from("keep me") });
    await page.getByText("notes.txt").first().waitFor();
    await page.evaluate(() => { window.__samePage = true; window.scrollTo(0, document.documentElement.scrollHeight / 2); });
    await settle();
    await page.waitForTimeout(100);
    const before = await topTurn();
    const [healthy, healthyStreams] = [healthCount, await streams()];
    for (let i = 0; i < 3; i++) {
      await page.evaluate(() => {
        for (const type of ["pageshow", "online"]) window.dispatchEvent(new Event(type));
        document.dispatchEvent(new Event("visibilitychange"));
        window.__phoneRenderSource.dispatchEvent(new MessageEvent("ping", { data: "{}" }));
      });
    }
    await page.waitForTimeout(500);
    if (healthCount !== healthy || (await streams()) !== healthyStreams) throw new Error(`healthy stream was replaced: ${healthCount - healthy} health reads, ${(await streams()) - healthyStreams} streams`);
    // New messages slide the window while the stream is lost.
    for (let index = 1500; index < 1530; index++) history.push(event(index + 1, "user_text", { text: `History ${index + 1}` }));
    await page.evaluate(() => window.__phoneRenderSource.onerror());
    await page.getByText("History 1530", { exact: true }).waitFor({ state: "attached" });
    await settle();
    const after = await topTurn();
    if (after.key !== before.key || Math.abs(after.top - before.top) > 2) throw new Error(`recovery moved the reading position to ${JSON.stringify(after)} from ${JSON.stringify(before)}`);
    if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("recovery lost the unsent text");
    if (!(await page.evaluate(() => window.__samePage && location.pathname === "/agent/a1"))) throw new Error("recovery reloaded or navigated");
    if ((await streams()) !== healthyStreams + 1 || healthCount !== healthy + 1) throw new Error("recovery did not use one health read and one stream");
    // Outage: one immediate attempt, then nothing until the 5-second retry.
    healthDown = true;
    const [down, downStreams] = [healthCount, await streams()];
    await page.evaluate(() => window.__phoneRenderSource.onerror());
    await page.getByRole("alert").filter({ hasText: "Mac unreachable" }).waitFor();
    await page.waitForTimeout(4500);
    if (healthCount !== down + 1 || (await streams()) !== downStreams) throw new Error(`outage made ${healthCount - down} health reads and ${(await streams()) - downStreams} streams in 4.5s`);
    healthDown = false;
    await page.getByRole("alert").filter({ hasText: "Mac unreachable" }).waitFor({ state: "detached", timeout: 3000 });
    if (healthCount !== down + 2 || (await streams()) !== downStreams + 1) throw new Error("outage retry was not a single attempt");
    if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("outage lost the unsent text");
    if (!(await page.getByText("notes.txt").first().isVisible())) throw new Error("recovery lost the selected attachment");
    console.log(`recovery: healthy stream kept, row ${after.key} within ${Math.abs(after.top - before.top)}px, text and attachment kept, outage retried once after 5s`);
  }
  await page.screenshot({ path: out, fullPage: !historyCheck && !scrollCheck && !recoveryCheck });
  if (shown && parityCheck) {
    const openAgent = async () => {
      await page.evaluate(() => {
        history.pushState(null, "", "/agent/a1");
        window.dispatchEvent(new PopStateEvent("popstate"));
      });
      await page.getByRole("textbox").waitFor();
    };
    const draft = "  Keep this draft\nwhen moving screens.  ";
    await page.getByRole("textbox").fill(draft);
    await page.getByRole("button", { name: "Chuck Home", exact: true }).click();
    await openAgent();
    if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("navigation lost draft");
    await page.goto("http://localhost:5199/remote.html");
    await openAgent();
    if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("reload lost draft");
    await page.getByRole("button", { name: "Send", exact: true }).click();
    await page.getByText("none", { exact: true }).waitFor();
    if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("refused send lost draft");
    for (const width of [360, 390, 430]) {
      await page.setViewportSize({ width, height: 844 });
      await page.getByRole("button", { name: "Open navigation" }).click();
      await page.getByRole("button", { name: "This phone", exact: true }).click();
      const select = page.getByRole("combobox", { name: "Theme" });
      await select.waitFor();
      for (const choice of ["sky-grove", "studio", "core"]) {
        await select.selectOption(choice);
        await page.waitForFunction((expected) => (document.documentElement.dataset.skin || "") === expected, choice === "core" ? "" : choice);
        const metrics = await select.evaluate((el) => ({ width: el.getBoundingClientRect().width, height: el.getBoundingClientRect().height, overflow: document.documentElement.scrollWidth > innerWidth }));
        if (metrics.height < 44 || metrics.overflow) throw new Error(`theme layout ${width}/${choice}: ${JSON.stringify(metrics)}`);
        await page.screenshot({ path: out.replace(/\.png$/, `-theme-${width}-${choice}.png`), fullPage: true });
      }
      await openAgent();
      if (await page.getByRole("textbox").inputValue() !== draft) throw new Error("settings navigation lost draft");
    }
    await page.evaluate(() => window.__phoneRenderSource.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ data: { agent_id: "a1", removed: true } }) })));
    await page.getByText("This agent is not on the Mac any more.", { exact: false }).waitFor();
    const retained = await page.evaluate(() => JSON.parse(localStorage.getItem("chuck-chat-drafts")).drafts.a1);
    if (retained) throw new Error("deleted agent retained draft");
    console.log("mobile parity: navigation/reload/refused-send/deletion and all themes at 360/390/430px pass");
  }
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
        const effort = document.querySelector(".phone-effort-cycle")?.getBoundingClientRect();
        return { footerBorder: footer.borderWidth, sendWidth: send.width, sendHeight: send.height, circleWidth: send.width - parseFloat(circle.left) - parseFloat(circle.right), circleHeight: send.height - parseFloat(circle.top) - parseFloat(circle.bottom), overflow: document.documentElement.scrollWidth > innerWidth, typingOutline: getComputedStyle(field).outlineStyle, textboxBorder: getComputedStyle(field).borderWidth, metadataAboveTextbox: metaBox.bottom <= box.top && !document.querySelector(".phone-composer-box").contains(metadata), modelLeft: Math.abs(metaBox.left - box.left) <= 1, effortRight: !!effort && Math.abs(effort.right - box.right) <= 1 && Math.abs(effort.top - metaBox.top) <= 1, metadataBordered: metaStyle.borderTopWidth !== "0px" && metaStyle.boxShadow !== "none", sendInside: send.bottom <= box.bottom && send.right <= box.right };
      });
      if (metrics.footerBorder !== "0px" || metrics.sendWidth !== 44 || metrics.sendHeight !== 44 || metrics.circleWidth !== 44 || metrics.circleHeight !== 28 || metrics.overflow) throw new Error(`${theme}: ${JSON.stringify(metrics)}`);
      if (metrics.typingOutline !== "none" || metrics.textboxBorder !== "0px" || !metrics.metadataAboveTextbox || !metrics.modelLeft || !metrics.effortRight || !metrics.metadataBordered || !metrics.sendInside) throw new Error(`${theme} composer: ${JSON.stringify(metrics)}`);
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
          return { backText: back.textContent, backWidth: backBox.width, backHeight: backBox.height, modelHeight: document.querySelector(".phone-model-switch").getBoundingClientRect().height, backAlignment: Math.abs(backBox.y + backBox.height / 2 - persona.y - persona.height / 2) + (persona.bottom <= title.top ? 0 : 99), actionAlignment: Math.abs(actionBox.y + actionBox.height / 2 - tabs.y - tabs.height / 2), actionOutsideTabs: !action.closest("[role=tablist]"), contextBetween: meter.top > title.bottom && meter.bottom <= tabs.top, overflow: document.documentElement.scrollWidth > innerWidth };
        });
        if (header.backText || header.backWidth < 44 || header.backHeight < 44 || header.modelHeight < 44 || header.backAlignment > 1 || header.actionAlignment > 1 || !header.actionOutsideTabs || !header.contextBetween || header.overflow) throw new Error(`${theme}/${width} header: ${JSON.stringify(header)}`);
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
        // FS-20.R46: a group header starts New agent collapsed, in that group.
        const groupNew = page.getByRole("button", { name: "New agent in Feature Changes", exact: true });
        const groupNewBox = await groupNew.boundingBox();
        if (!groupNewBox || groupNewBox.width < 44 || groupNewBox.height < 44 || groupNewBox.x + groupNewBox.width > width) throw new Error(`${theme}/${width} group New agent button: ${JSON.stringify(groupNewBox)}`);
        await page.screenshot({ path: `${prefix}-group-header-${width}.png`, animations: "disabled" });
        await groupNew.click();
        await dialog.waitFor();
        if (await dialog.locator("details").evaluate((el) => el.open)) throw new Error(`${theme}/${width} group New agent opened Advanced`);
        await dialog.getByRole("button", { name: "Close dialog", exact: true }).click();
        await page.locator(".phone-agent-row").first().click({ button: "right" });
        const sheet = page.getByRole("dialog");
        await sheet.getByRole("button", { name: "Move to group", exact: true }).click();
        await sheet.getByRole("combobox", { name: "Group" }).focus();
        const picker = await sheet.evaluate((el) => {
          const box = el.getBoundingClientRect();
          const list = el.querySelector("[role=listbox]").getBoundingClientRect();
          return { inside: list.bottom <= box.bottom && list.bottom <= innerHeight, scrolled: el.scrollHeight > el.clientHeight };
        });
        if (!picker.inside || picker.scrolled) throw new Error(`${theme}/${width} group picker: ${JSON.stringify(picker)}`);
        await page.screenshot({ path: `${prefix}-group-${width}.png`, animations: "disabled" });
        await page.keyboard.press("Escape");
        await page.keyboard.press("Escape");
      }
      await page.setViewportSize({ width: 390, height: 844 });
      console.log(`${theme || "core"} details: ${JSON.stringify(metrics)}`);
    }
  }
  if (shown && annotationCheck) {
    const openAgent = async () => {
      await page.evaluate(() => { history.pushState(null, "", "/agent/a1"); window.dispatchEvent(new PopStateEvent("popstate")); });
      await page.getByRole("list", { name: "Conversation" }).waitFor();
    };
    // A programmatic range stands in for native long-press selection; paired
    // devices remain the gate for the real gesture (FS-13.A19).
    const select = (selector, text, length) => page.evaluate(({ selector, text, length }) => {
      const scope = document.querySelector(selector);
      const walker = document.createTreeWalker(scope, NodeFilter.SHOW_TEXT);
      let node = walker.nextNode();
      while (node && !node.textContent.includes(text)) node = walker.nextNode();
      if (!node) throw new Error(`no text ${text}`);
      const start = node.textContent.indexOf(text);
      const range = document.createRange();
      range.setStart(node, start);
      range.setEnd(node, start + (length ?? text.length));
      getSelection().removeAllRanges();
      getSelection().addRange(range);
    }, { selector, text, length });
    const tray = (count) => page.getByRole("dialog", { name: `Pending annotations · ${count}/20`, exact: true });
    const sheetMetrics = (count) => tray(count).evaluate((el) => {
      const box = el.getBoundingClientRect();
      const send = el.querySelector(".phone-annotate-send").getBoundingClientRect();
      const discard = el.querySelector(".phone-annotate-discard").getBoundingClientRect();
      const targets = [...el.querySelectorAll("[role=radio], .phone-annotate-send, .phone-annotate-discard, .phone-sheet-close")].map((item) => item.getBoundingClientRect().height);
      return { top: box.top, bottom: box.bottom, height: innerHeight, sendVisible: send.top >= box.top && send.bottom <= Math.min(box.bottom, innerHeight), discardVisible: discard.top >= Math.max(box.top, 0) && discard.bottom <= innerHeight, minTarget: Math.min(...targets), overflow: document.documentElement.scrollWidth > innerWidth || el.scrollWidth > el.clientWidth, focused: document.activeElement?.tagName };
    });
    const check = (label, metrics) => {
      if (metrics.top < 0 || metrics.bottom > metrics.height + 1 || !metrics.sendVisible || !metrics.discardVisible || metrics.minTarget < 44 || metrics.overflow) throw new Error(`${label}: ${JSON.stringify(metrics)}`);
    };
    for (const theme of ["", "sky-grove", "studio"]) {
      skin = theme;
      for (const width of [360, 390, 430]) {
        const label = `${theme || "core"}/${width}`;
        const prefix = out.replace(/\.png$/, `-annotate-${theme || "core"}-${width}`);
        await page.setViewportSize({ width, height: 844 });
        await page.goto("http://localhost:5199/remote.html", { waitUntil: "domcontentloaded" });
        await page.evaluate(() => localStorage.removeItem("chuck-annotation-tray"));
        await openAgent();
        await page.waitForFunction((theme) => (document.documentElement.dataset.skin || "") === theme, theme);
        // Recorded card renders; its machine prompt does not.
        const card = page.locator(".phone-message .annotation-card");
        await card.waitFor();
        if (await page.getByText("machine prompt that must stay hidden").count()) throw new Error(`${label}: machine prompt shown`);
        const cardFit = await card.evaluate((el) => ({ overflow: el.scrollWidth > el.clientWidth + 1 || document.documentElement.scrollWidth > innerWidth }));
        if (cardFit.overflow) throw new Error(`${label} card overflow`);
        await card.scrollIntoViewIfNeeded();
        await page.screenshot({ path: `${prefix}-card.png`, animations: "disabled" });
        // Capture a selected excerpt from the first message.
        const userRow = page.locator(".phone-annotatable").filter({ hasText: "Run the tests and tell me the status." });
        await userRow.scrollIntoViewIfNeeded();
        const scrollBefore = await page.evaluate(() => scrollY);
        await select(".phone-transcript", "Run the tests");
        await userRow.getByRole("button", { name: "Annotate", exact: true }).click();
        await tray(1).waitFor();
        if (await page.evaluate(() => document.activeElement?.tagName) !== "TEXTAREA") throw new Error(`${label}: capture did not focus the instruction`);
        await page.keyboard.type("Run the race tests too.");
        check(`${label} one`, await sheetMetrics(1));
        // Keyboard-open stand-in: a shorter viewport keeps delivery reachable.
        await page.setViewportSize({ width, height: 520 });
        check(`${label} keyboard`, await sheetMetrics(1));
        await page.screenshot({ path: `${prefix}-keyboard.png`, animations: "disabled" });
        await page.setViewportSize({ width, height: 844 });
        await tray(1).getByRole("button", { name: "Close dialog", exact: true }).click();
        const returned = await page.waitForFunction(() => document.activeElement?.textContent === "Annotate", null, { timeout: 2000 }).then(() => true, () => false);
        if (!returned) throw new Error(`${label}: focus returned to ${await page.evaluate(() => document.activeElement?.tagName)}`);
        if (Math.abs(await page.evaluate(() => scrollY) - scrollBefore) > 2) throw new Error(`${label}: reading place moved`);
        // Tracked file: Rendered path-only and Source line-range capture.
        await page.getByRole("tab", { name: /^Files/ }).click();
        await page.getByRole("button", { name: "Open file", exact: true }).click();
        const preview = page.getByRole("region", { name: "File content" });
        await preview.getByText("The phone can now annotate files.").waitFor();
        await select(".phone-file-preview", "Release notes");
        await preview.getByRole("button", { name: "Annotate selection", exact: true }).click();
        await tray(2).waitFor();
        await page.keyboard.type("Rename the heading.");
        await tray(2).getByRole("button", { name: "Close dialog", exact: true }).click();
        await preview.getByRole("button", { name: "Source", exact: true }).click();
        await preview.locator('[data-file-line="4"]').waitFor();
        await page.evaluate(() => {
          const start = document.querySelector('[data-file-line="3"]');
          const end = document.querySelector('[data-file-line="4"]');
          const text = (line) => { const walker = document.createTreeWalker(line, NodeFilter.SHOW_TEXT); let node = walker.nextNode(); let last = null; const nodes = []; while (node) { nodes.push(node); last = node; node = walker.nextNode(); } return { first: nodes.find((item) => /\S/.test(item.textContent) && !/^\d+$/.test(item.textContent.trim())), last }; };
          const range = document.createRange();
          range.setStart(text(start).first, 0);
          range.setEnd(text(end).last, text(end).last.textContent.length);
          getSelection().removeAllRanges();
          getSelection().addRange(range);
        });
        await preview.getByRole("button", { name: "Annotate selection", exact: true }).click();
        await tray(3).waitFor();
        await page.keyboard.type("Shorten these lines.");
        const anchors = await tray(3).locator(".phone-annotate-head h3").allTextContents();
        if (anchors[1] !== `File ${longPath}` || anchors[2] !== `File ${longPath}:3–4`) throw new Error(`${label} anchors: ${JSON.stringify(anchors)}`);
        // Dense batch with long paths stays bounded and scrolls locally.
        await tray(3).getByRole("button", { name: "Close dialog", exact: true }).click();
        await page.getByRole("tab", { name: "Chat", exact: true }).click();
        const annotateButtons = page.locator(".phone-annotate-event");
        for (let index = 0; index < 3; index++) {
          await annotateButtons.nth(1).click();
          await tray(4 + index).waitFor();
          await page.keyboard.type(`Follow up ${index + 1}.`);
          await tray(4 + index).getByRole("button", { name: "Close dialog", exact: true }).click();
        }
        const open = page.locator(".phone-annotate-open");
        await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
        const pinned = await open.evaluate((el) => { const box = el.getBoundingClientRect(); return box.top >= 0 && box.bottom <= innerHeight && box.height >= 44; });
        if (!pinned) throw new Error(`${label}: pending count not reachable while scrolled`);
        await open.click();
        await tray(6).getByLabel("Overall instruction (optional)").fill("Keep it to one commit.");
        check(`${label} dense`, await sheetMetrics(6));
        await page.screenshot({ path: `${prefix}-dense.png`, animations: "disabled" });
        if (width === 390) {
          // New task: preflight, reviewable launch, cancel back to the tray.
          await tray(6).getByRole("radio", { name: "New task", exact: true }).click();
          await tray(6).getByRole("button", { name: "Continue to launch", exact: true }).click();
          const launch = page.getByRole("dialog", { name: "New agent", exact: true });
          await launch.waitFor();
          await launch.getByText("Advanced").click();
          await launch.getByRole("combobox", { name: "Project" }).selectOption("other");
          await page.screenshot({ path: `${prefix}-launch.png`, animations: "disabled" });
          await launch.getByRole("button", { name: "Cancel", exact: true }).click();
          await tray(6).waitFor();
          await tray(6).getByRole("button", { name: "Continue to launch", exact: true }).click();
          await launch.waitFor();
          await launch.getByRole("button", { name: "Launch and send", exact: true }).click();
          await page.waitForFunction(() => location.pathname === "/agent/a3");
          const sent = annotationPosts.at(-1);
          if (!sent || sent.target?.agent_id !== "a3" || sent.annotations.length !== 6 || sent.overall_instruction !== "Keep it to one commit." || launches.at(-1)?.project !== "my-app") throw new Error(`${label} delivery: ${JSON.stringify({ sent, launch: launches.at(-1) })}`);
        } else {
          await tray(6).getByRole("button", { name: "Send annotations", exact: true }).click();
          await page.getByText("Annotations sent to this agent.").waitFor();
          const sent = annotationPosts.at(-1);
          if (sent.target?.kind !== "self" || sent.annotations[2]?.start_line !== 3 || sent.annotations[1]?.start_line !== undefined) throw new Error(`${label} self delivery: ${JSON.stringify(sent)}`);
        }
        if (fileReads.some((read) => read !== longPath)) throw new Error(`file read key changed: ${JSON.stringify(fileReads)}`);
        console.log(`${label} annotations: capture, focus, keyboard, Files, dense tray and ${width === 390 ? "new-task" : "self"} delivery pass`);
      }
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
