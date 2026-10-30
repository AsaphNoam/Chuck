// Real server + fakeacp activity_showcase; run after make embed.
import { chromium } from "playwright";
import { mkdirSync, writeFileSync } from "node:fs";
const base = process.argv[2] || "http://127.0.0.1:4417";
const out = process.argv[3] || "/tmp/chuck-running-tasks";
mkdirSync(out, { recursive: true });
async function api(path, body, method = "POST") {
  const response = await fetch(base + path, { method, headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) });
  if (!response.ok) throw new Error(`${path}: ${response.status}`);
  return response.json();
}
const { agent } = await api("/api/sessions", { role: "implementer", project: "stress", backend: "claude", model: "haiku", name: "Running work", interface: "chat" });
await api(`/api/sessions/${agent.agent_id}/prompt`, { text: "Show activity" });
const browser = await chromium.launch({ headless: true });
try {
  const page = await browser.newPage();
  for (const skin of ["", "sky-grove"]) {
    await api("/api/config", { appearance_skin: skin }, "PUT");
    for (const width of [1024, 1440]) {
      await page.setViewportSize({ width, height: 900 });
      await page.goto(`${base}/agent/${agent.agent_id}`);
      await page.getByRole("button", { name: "1 running", exact: true }).click();
      await page.getByRole("dialog").waitFor();
      if (await page.getByText("Background tasks", { exact: false }).count()) throw new Error("Old list remains");
      const box = await page.getByRole("dialog").boundingBox();
      if (!box || box.x < 0 || box.y < 0 || box.x + box.width > width) throw new Error("Popover clipped");
      await page.screenshot({ path: `${out}/${skin || "core"}-${width}.png` });
      await page.getByRole("button", { name: "Ran npm", exact: true }).click();
      await page.locator("[data-seq]:focus").waitFor();
      await page.getByRole("button", { name: "1 running", exact: true }).click();
      await page.keyboard.press("Escape");
      await page.getByRole("dialog").waitFor({ state: "detached" });
      await api("/api/layout", { order: [], density: { perRow: 3, gap: 16 }, expanded: [agent.agent_id] }, "PUT");
      await page.goto(`${base}/project/stress`);
      await page.getByRole("button", { name: "1 running", exact: true }).click();
      await page.getByRole("dialog").waitFor();
      await page.getByRole("button", { name: "Stop", exact: true }).click({ trial: true });
      await page.screenshot({ path: `${out}/${skin || "core"}-${width}-dashboard.png` });
      await page.keyboard.press("Escape");
    }
  }
  const response = await fetch(`${base}/api/sessions/${agent.agent_id}/transcript`);
  const transcript = await response.json();
  writeFileSync(`${out}/phone-transcript.json`, JSON.stringify(transcript));
  await page.getByRole("button", { name: "1 running", exact: true }).click();
  await page.getByRole("button", { name: "Stop", exact: true }).click();
  await page.getByRole("button", { name: "1 running", exact: true }).waitFor({ state: "detached" });
  console.log(`PASS real fake-ACP count, popover, jump/focus, Escape and Stop; screenshots ${out}`);
} finally { await browser.close(); }
