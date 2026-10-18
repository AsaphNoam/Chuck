// Live FS-03.A76 pass against the real server and embedded UI, with seven fakeacp
// agents streaming in one project (scripts/stress-fixture). From the repo root,
// after `make embed`:
//
//   go run -tags sqlite_fts5 ./scripts/stress-fixture -port 4411 -chunks 400 -delay-ms 20
//   node ui/scripts/breadcrumb-live-check.mjs [baseURL] [outDir]
//
// Checks five entries plus +2 with the current agent visible, entries that do not
// reorder while agents stream, one-click switching, and a stopped sibling leaving
// without a refresh. Exits 1 if any check fails.
import { mkdirSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { chromium } from "playwright";

const BASE = process.argv[2] || "http://127.0.0.1:4411";
const OUT = process.argv[3] || join(tmpdir(), `breadcrumb-live-${Date.now()}`);
mkdirSync(OUT, { recursive: true });

const checks = [];
function record(name, pass, detail) {
  checks.push({ name, pass: Boolean(pass), detail: detail === undefined ? undefined : String(detail) });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail !== undefined ? ` — ${detail}` : ""}`);
}

const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1024, height: 800 } });
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const agents = await (await fetch(`${BASE}/api/sessions`)).json();
  const running = (Array.isArray(agents) ? agents : agents.agents ?? Object.values(agents)).filter((agent) => agent.running);
  record("seven running agents", running.length === 7, running.length);
  const last = running.at(-1);

  await page.goto(`${BASE}/agent/${last.agent_id}`);
  const entries = page.locator(".chat-breadcrumb-siblings > .agent-mini-card");
  await entries.first().waitFor({ timeout: 15000 });
  const names = async () => entries.evaluateAll((items) => items.map((item) => item.textContent));
  const first = await names();
  record("five entries with +2", first.length === 5 && await page.locator(".chat-breadcrumb-overflow > button").textContent() === "+2", first.join(" | "));
  record("current agent visible", await page.locator(".agent-mini-card[aria-current='page']").textContent() === first.at(-1), first.at(-1));
  await page.waitForTimeout(2000);
  record("no reorder while streaming", JSON.stringify(await names()) === JSON.stringify(first));
  await page.screenshot({ path: join(OUT, "live-1024.png"), clip: { x: 0, y: 0, width: 1024, height: 260 } });

  const target = page.locator(".chat-breadcrumb-siblings > a.agent-mini-card").first();
  const targetName = (await target.textContent()).slice(1);
  await target.click();
  await page.waitForFunction((name) => document.querySelector(".chat-header h1")?.textContent === name, targetName, { timeout: 10000 });
  record("one click opens the sibling", (await page.locator(".agent-mini-card[aria-current='page']").textContent()).endsWith(targetName), targetName);

  const leaving = running.find((agent) => agent.agent_id !== last.agent_id && !targetName.includes(agent.name));
  await page.evaluate(async (id) => fetch(`/api/sessions/${id}/stop`, { method: "POST" }), leaving.agent_id);
  await page.waitForFunction(() => document.querySelector(".chat-breadcrumb-overflow > button")?.textContent === "+1", null, { timeout: 15000 }).catch(() => {});
  record("a stopped sibling leaves without refresh", await page.locator(".chat-breadcrumb-overflow > button").textContent() === "+1");
  record("no page errors", errors.length === 0, errors.join("; "));
} finally {
  writeFileSync(join(OUT, "report.json"), JSON.stringify(checks, null, 2));
  await browser.close();
}
console.log(OUT);
if (checks.some((check) => !check.pass)) process.exitCode = 1;
