// Renders the phone Home and an agent conversation in every appearance at
// phone widths, writing screenshots for visual review (jsdom cannot see
// layout, INV §13). All /api calls and the event stream are stubbed.
//
//   node scripts/phone-skins-render.mjs [out-prefix] [--busy]
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createServer } from "vite";
import { chromium, devices } from "playwright";

const prefix = process.argv.slice(2).find((arg) => !arg.startsWith("--")) ?? join(tmpdir(), "phone-skins");
const busy = process.argv.includes("--busy");
const agent = {
  agent_id: "a1", name: "phone-render", role: "implementer", project: "my-app", backend: "claude", model: "m",
  fast: false, fast_available: false, steering_available: true, interface: "chat", created_at: "", running: true,
  state: busy ? "busy" : "idle", detail: "", updated_at: 0, archived: false,
};
const event = (seq, type, data) => ({ agent_id: "a1", seq, type, ts: "", data });
const projects = {
  "my-app": { title: "My App", color: [50, 120, 180] },
  docs: { title: "Docs", color: [180, 90, 60] },
  infra: { title: "Infra", color: [80, 150, 90] },
};

const server = await createServer({ server: { port: 5198, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
const failures = [];
try {
  for (const skin of ["", "sky-grove", "studio"]) {
    for (const width of [360, 390, 430]) {
      const context = await browser.newContext({ ...devices["iPhone 13"], viewport: { width, height: 844 } });
      await context.addInitScript((hydrated) => {
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
      }, [{ agent_id: "a1", data: agent }, { agent_id: "__hydrated__", data: { hydrated: true } }]);
      await context.route((url) => url.pathname.startsWith("/api/"), (route) => {
        const path = new URL(route.request().url()).pathname;
        if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin } });
        if (path === "/api/projects") return route.fulfill({ json: projects });
        if (path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer" } } });
        if (path === "/api/remote/home") return route.fulfill({ json: { needs_you: [], active_runs: [] } });
        if (path === "/api/remote/runtime-options") return route.fulfill({ json: { backends: [] } });
        if (path === "/api/sessions/a1/transcript") return route.fulfill({ json: { agent_id: "a1", has_more: false, pending_permission: null, latest_assistant: "", events: [
          event(1, "user_text", { text: "Run the tests and tell me the status." }),
          event(2, "assistant_text", { delta: "Done. All tests pass." }),
        ] } });
        return route.fulfill({ json: {} });
      });
      const page = await context.newPage();
      page.on("pageerror", (error) => failures.push(`${skin || "core"}/${width}: ${error.message}`));
      await page.goto("http://localhost:5198/remote.html");
      await page.getByRole("region", { name: "Projects" }).waitFor();
      await page.waitForFunction((expected) => (document.documentElement.dataset.skin || "") === expected, skin);
      const home = await page.evaluate(() => {
        const cards = [...document.querySelectorAll(".phone-project-list > li")].map((item) => item.getBoundingClientRect());
        return { gaps: cards.slice(1).map((box, index) => box.top - cards[index].bottom), tagline: document.body.textContent.includes("out of the loop"), overflow: document.documentElement.scrollWidth > innerWidth };
      });
      if (home.tagline || home.overflow || home.gaps.some((gap) => gap < 8)) failures.push(`${skin || "core"}/${width} home: ${JSON.stringify(home)}`);
      await page.screenshot({ path: `${prefix}-${skin || "core"}-home-${width}.png`, fullPage: true, animations: "disabled" });
      await page.evaluate(() => { history.pushState(null, "", "/agent/a1"); window.dispatchEvent(new PopStateEvent("popstate")); });
      await page.getByRole("list", { name: "Conversation" }).waitFor();
      const chat = await page.evaluate(() => {
        const cancel = document.querySelector(".phone-composer-actions [aria-label='Cancel turn']");
        return { cancelInComposer: !!cancel, controlsRow: !!document.querySelector(".phone-controls"), overflow: document.documentElement.scrollWidth > innerWidth };
      });
      if (chat.controlsRow || chat.cancelInComposer !== busy || chat.overflow) failures.push(`${skin || "core"}/${width} chat: ${JSON.stringify(chat)}`);
      await page.screenshot({ path: `${prefix}-${skin || "core"}-chat-${width}.png`, fullPage: true, animations: "disabled" });
      await context.close();
    }
  }
} finally {
  await browser.close();
  await server.close();
}
if (failures.length) {
  console.error(failures.join("\n"));
  process.exitCode = 1;
} else console.log(`wrote ${prefix}-*.png`);
