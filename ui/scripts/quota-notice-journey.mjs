// Verify the durable quota-continuation notice across desktop reload, authenticated
// phone, and Archive using the real embedded server and its known-reset fixture.
// Start the fixture with CHUCK_QUOTA_BROWSER_KNOWN=1, then run:
//   node ui/scripts/quota-notice-journey.mjs <fixture.json> <receipt-directory>
// Create <fixture.json>.done after the browser process exits.
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const fixture = JSON.parse(readFileSync(process.argv[2], "utf8"));
const out = process.argv[3] || "/tmp/chuck-quota-notice-journey";
const title = "Chuck automatically continued this conversation after a quota interruption";
const agentID = fixture.quota_agent;
if (!agentID) throw new Error("fixture has no quota_agent; enable CHUCK_QUOTA_BROWSER_KNOWN=1");
mkdirSync(out, { recursive: true });

const checks = [];
const errors = [];
const record = (name, pass, detail = "") => {
  checks.push({ name, pass, detail });
  console.log(`${pass ? "PASS" : "FAIL"} ${name}${detail ? `: ${detail}` : ""}`);
  if (!pass) throw new Error(name);
};
async function api(path, body, method = body === undefined ? "GET" : "POST") {
  const response = await fetch(fixture.desktop + path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const text = await response.text();
  let data;
  try { data = text ? JSON.parse(text) : undefined; } catch { data = text; }
  if (!response.ok) throw new Error(`${method} ${path} -> ${response.status}: ${text}`);
  return data;
}
const noticeEvents = (events) => events.filter(event => {
  const data = event.data && typeof event.data === "object" ? event.data : event;
  return (event.kind === "notice" || event.type === "notice") && data.title === title;
});
async function checkNotice(page, name, screenshot) {
  await page.getByRole("note").filter({ hasText: title }).waitFor({ state: "visible", timeout: 15000 });
  const rows = page.getByRole("note").filter({ hasText: title });
  const count = await rows.count();
  const rowText = count ? await rows.first().innerText() : "";
  const attribution = count ? await rows.first().getByText("Chuck", { exact: true }).count() : 0;
  record(name, count === 1 && attribution === 1, JSON.stringify({ count, attribution, rowText }));
  if (screenshot) await page.screenshot({ path: join(out, screenshot), fullPage: true, animations: "disabled" });
}

let browser;
let desktop;
let phone;
try {
  browser = await chromium.launch();
  desktop = await browser.newPage({ viewport: { width: 1024, height: 900 } });
  const phoneContext = await browser.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true });
  await phoneContext.addCookies([{
    name: fixture.cookie, value: fixture.token, url: fixture.phone,
    secure: true, httpOnly: true, sameSite: "Strict",
  }]);
  phone = await phoneContext.newPage();
  for (const page of [desktop, phone]) page.on("pageerror", error => errors.push(error.message));

  const transcriptPath = `/api/sessions/${agentID}/transcript`;
  const initial = await api(transcriptPath);
  const initialNotices = noticeEvents(initial.events || []);
  record("fixture contains one Chuck-owned continuation notice", initialNotices.length === 1 && initialNotices[0].data?.origin === "chuck",
    JSON.stringify(initialNotices));

  await desktop.goto(`${fixture.desktop}/agent/${agentID}`, { waitUntil: "domcontentloaded" });
  await checkNotice(desktop, "desktop transcript renders exact Chuck notice once", "desktop-live-notice.png");
  await desktop.reload({ waitUntil: "domcontentloaded" });
  await checkNotice(desktop, "desktop reload retains exact Chuck notice once", "desktop-reloaded-notice.png");
  const reloaded = await api(transcriptPath);
  const reloadedNotices = noticeEvents(reloaded.events || []);
  record("desktop reload API retains one ordered notice", reloadedNotices.length === 1 &&
    reloadedNotices[0].seq === initialNotices[0].seq && reloadedNotices[0].data?.origin === "chuck",
    JSON.stringify({ before: initialNotices, after: reloadedNotices }));

  await phone.goto(`${fixture.phone}/agent/${agentID}`, { waitUntil: "domcontentloaded" });
  await checkNotice(phone, "authenticated phone renders exact Chuck notice once", "phone-live-notice.png");
  await phone.reload({ waitUntil: "domcontentloaded" });
  await checkNotice(phone, "authenticated phone reload retains exact Chuck notice once", "phone-reloaded-notice.png");

  await api(`/api/sessions/${agentID}/archive`, {});
  const archivedSession = await api(`/api/sessions/${agentID}`);
  record("desktop API archived quota conversation", archivedSession.agent?.archived === true);
  await desktop.goto(`${fixture.desktop}/archive/${agentID}`, { waitUntil: "domcontentloaded" });
  await checkNotice(desktop, "desktop Archive retains exact Chuck notice once", "desktop-archive-notice.png");
  const archived = await api(transcriptPath);
  const archivedNotices = noticeEvents(archived.events || []);
  record("archived transcript API retains one ordered notice", archivedNotices.length === 1 &&
    archivedNotices[0].seq === initialNotices[0].seq && archivedNotices[0].data?.origin === "chuck",
    JSON.stringify(archivedNotices));

  record("no browser exceptions", errors.length === 0, errors.join("; "));
} catch (error) {
  errors.push(error instanceof Error ? error.stack || error.message : String(error));
  for (const [name, page] of [["desktop", desktop], ["phone", phone]]) {
    if (!page) continue;
    try {
      await page.screenshot({ path: join(out, `${name}-failure.png`), fullPage: true, animations: "disabled" });
      errors.push(`${name} URL: ${page.url()}`);
      errors.push(`${name} body: ${(await page.locator("body").innerText()).slice(0, 5000)}`);
    } catch (screenshotError) {
      errors.push(`${name} diagnostic failed: ${String(screenshotError)}`);
    }
  }
  console.error(error);
  process.exitCode = 1;
} finally {
  writeFileSync(join(out, "receipt.json"), JSON.stringify({ agentID, title, checks, errors }, null, 2));
  if (browser) await browser.close();
}
