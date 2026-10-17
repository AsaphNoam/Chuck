// Render the quota interruption matrix at the supported desktop and phone widths.
// Usage: node ui/scripts/quota-render.mjs [origin] [output-directory]
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const origin = process.argv[2] ?? "http://127.0.0.1:5178";
const output = process.argv[3] ?? "/tmp/chuck-quota-render";
mkdirSync(output, { recursive: true });

const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1024, height: 900 }, deviceScaleFactor: 1 });
// This fixture validates layout and quota controls, not service-worker lifecycle.
// Blocking registration avoids a Vite fallback MIME warning for /sw.js.
const phoneContext = await browser.newContext({ viewport: { width: 390, height: 844 }, deviceScaleFactor: 1, serviceWorkers: "block" });
const phone = await phoneContext.newPage();
const errors = [];
const checks = [];
const phoneRequests = [];
const consoleErrors = [];
for (const target of [page, phone]) {
  target.on("pageerror", error => errors.push(error.message));
  target.on("console", message => { if (message.type() === "error") consoleErrors.push(message.text()); });
  target.on("requestfailed", request => phoneRequests.push(`FAILED ${request.method()} ${request.url()} ${request.failure()?.errorText ?? ""}`));
}
phone.on("request", request => phoneRequests.push(`${request.method()} ${request.url()}`));

function check(name, pass, details = "") {
  checks.push({ name, pass, details });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${details ? `: ${details}` : ""}`);
}

try {
  await page.goto(`${origin}/__visual-matrix`, { waitUntil: "domcontentloaded" });
  await page.locator(".visual-matrix").waitFor();
  await page.evaluate(() => document.fonts.ready);

  for (const [skin, label] of [["core", "Core"], ["sky-grove", "Sky & Grove"], ["studio", "Studio"]]) {
    await page.getByLabel("Fixture appearance").selectOption(skin);
    await page.waitForTimeout(100);

    const banner = page.locator(".chat-card > .quota-banner").first();
    const checkBanner = page.locator(".chat-card > .quota-banner").nth(1);
    await banner.waitFor();
    const state = await page.evaluate(() => {
      const banners = [...document.querySelectorAll(".quota-banner")];
      const rects = banners.map(element => {
        const rect = element.getBoundingClientRect();
        return { left: rect.left, right: rect.right, width: rect.width, scrollWidth: element.scrollWidth };
      });
      return { viewport: window.innerWidth, banners: rects, cardWidth: document.querySelector(".chat-card")?.getBoundingClientRect().width ?? 0 };
    });
    const prefix = `${label} 1024px`;
    check(`${prefix}: viewport exact`, state.viewport === 1024, `${state.viewport}px`);
    check(`${prefix}: scheduled and unknown reset states visible`,
      await page.getByText("Reset time unknown", { exact: true }).first().isVisible()
      && await banner.getByText(/^Reset /).isVisible()
      && await banner.getByRole("button", { name: "Cancel continuation" }).isVisible());
    // FS-01.R50: a check time is never presented as a reset.
    check(`${prefix}: pending check distinct from reset`,
      await checkBanner.getByText("Reset time unknown", { exact: true }).isVisible()
      && await checkBanner.getByText(/^Will check again at /).isVisible()
      && await checkBanner.getByText("5 checks left", { exact: true }).isVisible()
      && await checkBanner.getByRole("button", { name: "Cancel continuation" }).isVisible()
      && !(await checkBanner.getByText(/^Will continue at /).count()));
    check(`${prefix}: quota banners fit viewport`,
      state.banners.every(rect => rect.left >= -1 && rect.right <= 1025 && rect.scrollWidth <= rect.width + 1),
      JSON.stringify(state.banners));
    check(`${prefix}: desktop chat card fits viewport`, state.cardWidth <= 1025, `${state.cardWidth}px`);
    await page.screenshot({ path: join(output, `1024-${skin}.png`), fullPage: true, animations: "disabled" });
    await banner.screenshot({ path: join(output, `1024-${skin}-chat-quota.png`), animations: "disabled" });
  }

  const agent = {
    agent_id: "quota-fixture", name: "Quota recovery fixture", role: "builder", project: "quota-demo",
    backend: "claude", model: "claude-opus", interface: "chat", fast: false, fast_available: false,
    steering_available: false, created_at: "2026-10-10T10:00:00Z", running: false, state: "error",
    detail: "Provider quota reached", context_pct: 0, updated_at: 0, archived: false,
    quota: { episode_id: "fixture-scheduled", revision: 2, observed_at: "2026-10-10T10:00:00Z", reset_at: "2026-10-10T12:00:00Z", due_at: "2026-10-10T12:00:00Z", continuation_state: "scheduled" },
  };
  let phoneSkin = "";
  await phone.addInitScript(() => Object.defineProperty(window, "SharedWorker", { value: undefined }));
  await phoneContext.route("**/api/**", async route => {
    const url = new URL(route.request().url());
    if (!url.pathname.startsWith("/api/")) return route.continue();
    if (url.pathname === "/api/health") return route.fulfill({ json: { status: "ok" } });
    if (url.pathname === "/api/config") return route.fulfill({ json: { appearance_skin: phoneSkin } });
    if (url.pathname === "/api/roles") return route.fulfill({ json: {} });
    if (url.pathname === "/api/remote/home") return route.fulfill({ json: { needs_you: [], active_runs: [] } });
    if (url.pathname === "/api/events") {
      const events = [
        `event: state_update\ndata: ${JSON.stringify({ agent_id: agent.agent_id, data: agent })}\n\n`,
        `event: state_update\ndata: ${JSON.stringify({ agent_id: "__hydrated__", data: { hydrated: true } })}\n\n`,
      ].join("");
      return route.fulfill({ status: 200, contentType: "text/event-stream", headers: { "Cache-Control": "no-cache" }, body: events });
    }
    if (url.pathname === `/api/sessions/${agent.agent_id}/transcript`) {
      return route.fulfill({ json: { agent_id: agent.agent_id, events: [], has_more: false, pending_permission: null, latest_assistant: "" } });
    }
    if (url.pathname === `/api/sessions/${agent.agent_id}/prompt`) {
      return route.fulfill({ json: { agent_id: agent.agent_id, text: "", after_seq: 0 } });
    }
    return route.fulfill({ status: 200, json: {} });
  });
  for (const [skin, label] of [["", "Core"], ["sky-grove", "Sky & Grove"], ["studio", "Studio"]]) {
    phoneSkin = skin;
    await phone.goto(`${origin}/remote.html`, { waitUntil: "domcontentloaded" });
    await phone.getByRole("button", { name: "Chuck Home" }).waitFor({ timeout: 10000 });
    await phone.evaluate(() => {
      history.replaceState(null, "", "/agent/quota-fixture");
      window.dispatchEvent(new PopStateEvent("popstate"));
    });
    try {
      await phone.locator(".phone-agent").waitFor({ timeout: 10000 });
    } catch (error) {
      const body = await phone.locator("body").innerText().catch(() => "<body unavailable>");
      await phone.screenshot({ path: join(output, `390-phone-${skin || "core"}-failure.png`), fullPage: true });
      writeFileSync(join(output, `390-phone-${skin || "core"}-diagnostics.json`), JSON.stringify({ error: String(error), url: phone.url(), body, requests: phoneRequests, pageErrors: errors, consoleErrors }, null, 2));
      throw new Error(`Phone route did not render at ${label}: ${body}; requests: ${phoneRequests.join(" | ")}; errors: ${[...errors, ...consoleErrors].join(" | ")}`);
    }
    await phone.getByText("Quota reached", { exact: true }).waitFor();
    const banner = phone.locator(".phone-agent > .quota-banner");
    const bounds = await banner.evaluate(element => {
      const rect = element.getBoundingClientRect();
      return { left: rect.left, right: rect.right, width: rect.width, scrollWidth: element.scrollWidth, viewport: window.innerWidth };
    });
    check(`${label} 390px phone route: viewport exact`, bounds.viewport === 390, `${bounds.viewport}px`);
    check(`${label} 390px phone route: reset and cancel visible`,
      await banner.getByText(/^Reset /).isVisible() && await banner.getByRole("button", { name: "Cancel continuation" }).isVisible());
    check(`${label} 390px phone route: banner fits viewport`,
      bounds.left >= -1 && bounds.right <= 391 && bounds.scrollWidth <= bounds.width + 1, JSON.stringify(bounds));
    await phone.screenshot({ path: join(output, `390-phone-${skin || "core"}.png`), fullPage: true, animations: "disabled" });
    await banner.screenshot({ path: join(output, `390-phone-${skin || "core"}-quota.png`), animations: "disabled" });
  }
  check("no browser page errors", errors.length === 0, errors.join("; "));
  check("no browser console errors", consoleErrors.length === 0, consoleErrors.join("; "));
} finally {
  writeFileSync(join(output, "receipt.json"), JSON.stringify({ checks, errors, consoleErrors, phoneRequests, coverageExcluded: ["service worker"] }, null, 2));
  await browser.close();
}

if (checks.some(result => !result.pass)) process.exitCode = 1;
