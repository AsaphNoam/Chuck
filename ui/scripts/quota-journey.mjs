// Exercise quota recovery through the embedded desktop and authenticated phone UI.
// Run after `make embed`, with an isolated TestGroupBrowserFixture and a fake ACP
// configured to emit terminal sessionFailure metadata on every turn:
//
//   FAKEACP_SCENARIO=hold_turn FAKEACP_HOLD_FILE=/tmp/chuck-quota-release \
//   FAKEACP_FAILURE_RESULT='{"id":"quota","revision":1,"category":"limit","severity":"error","title":"Quota","actions":[]}' \
//   FAKEACP_FAILURE_UPDATE='{"id":"quota","revision":1,"category":"limit","severity":"error","title":"Quota","actions":[]}' \
//   CHUCK_GROUP_BROWSER_READY=/tmp/quota-fixture.json \
//   go test -tags sqlite_fts5 ./internal/server -run '^TestGroupBrowserFixture$' -timeout 20m -v
//
// Then run:
//   node ui/scripts/quota-journey.mjs /tmp/quota-fixture.json /tmp/quota-journey /tmp/chuck-quota-release
// Create /tmp/quota-fixture.json.done after this process exits to finish the fixture.
// The provider evidence intentionally contains no reset timestamp; the journey
// verifies that unknown reset stays unknown instead of inferring a time.
import { existsSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const fixture = JSON.parse(readFileSync(process.argv[2], "utf8"));
const out = process.argv[3] || "/tmp/chuck-quota-journey";
const releaseFile = process.argv[4];
if (!releaseFile) throw new Error("pass the FAKEACP_HOLD_FILE path as the fourth argument");
if (existsSync(releaseFile)) throw new Error(`hold file already exists: ${releaseFile}`);
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
  if (!response.ok) throw new Error(`${path} -> ${response.status}: ${text}`);
  return data;
}
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
async function waitForQuota(agentID) {
  const deadline = Date.now() + 20000;
  while (Date.now() < deadline) {
    const response = await api(`/api/sessions/${agentID}`);
    if (response.quota) return response.quota;
    await sleep(200);
  }
  throw new Error(`quota projection did not arrive for ${agentID}`);
}
async function launch(name) {
  const response = await api("/api/sessions", {
    name, project: "other", role: "implementer", backend: "claude", model: "sonnet", interface: "chat",
  });
  return response.agent.agent_id;
}

const browser = await chromium.launch();
const desktop = await browser.newPage({ viewport: { width: 1024, height: 900 } });
const phoneContext = await browser.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true });
await phoneContext.addCookies([{ name: fixture.cookie, value: fixture.token, url: fixture.phone, secure: true, httpOnly: true, sameSite: "Strict" }]);
const phone = await phoneContext.newPage();
for (const page of [desktop, phone]) page.on("pageerror", error => errors.push(error.message));

try {
  // A held fake turn proves that the existing composer sends an explicit turn
  // cancellation body, independently of quota-continuation cancellation.
  const runningID = await launch("Quota journey cancellation");
  await desktop.goto(`${fixture.desktop}/agent/${runningID}`, { waitUntil: "domcontentloaded" });
  const composer = desktop.locator('[data-ui="composer"]');
  const message = composer.getByRole("textbox");
  await message.fill("cancel this held turn");
  await message.press("Enter");
  const composerCancel = composer.getByRole("button", { name: "Cancel", exact: true });
  await composerCancel.waitFor({ state: "visible", timeout: 10000 });
  const cancelRequest = desktop.waitForRequest(request =>
    request.method() === "POST" && request.url().endsWith(`/api/sessions/${runningID}/cancel`));
  await composerCancel.click();
  const sentCancel = await cancelRequest;
  record("composer cancel serializes explicit turn target", JSON.stringify(sentCancel.postDataJSON()) === JSON.stringify({ target: "turn" }), JSON.stringify(sentCancel.postDataJSON()));

  // Release the fixture's hold for the terminal quota turn that follows.
  writeFileSync(releaseFile, "release", { mode: 0o600 });
  const quotaID = await launch("Quota journey unknown reset");
  await desktop.goto(`${fixture.desktop}/agent/${quotaID}`, { waitUntil: "domcontentloaded" });
  await desktop.locator('[data-ui="transcript"]').waitFor();
  const quotaComposer = desktop.locator('[data-ui="composer"]');
  await quotaComposer.getByRole("textbox").fill("trigger typed quota failure");
  await quotaComposer.getByRole("textbox").press("Enter");
  const quota = await waitForQuota(quotaID);
  record("live fake provider quota has unknown reset", quota.continuation_state === "unknown_reset" && quota.reset_at == null,
    JSON.stringify({ continuation_state: quota.continuation_state, reset_at: quota.reset_at }));
  await desktop.getByText("Quota reached", { exact: true }).waitFor();
  await desktop.getByText("Reset time unknown", { exact: true }).waitFor();
  await desktop.screenshot({ path: join(out, "desktop-unknown-reset.png"), fullPage: true, animations: "disabled" });

  await api(`/api/sessions/${quotaID}/stop`, {});
  record("quota interruption survives process stop", (await api(`/api/sessions/${quotaID}`)).quota?.continuation_state === "cancelled");
  await phone.goto(`${fixture.phone}/agent/${quotaID}`, { waitUntil: "domcontentloaded" });
  await phone.getByText("Quota reached", { exact: true }).waitFor();
  await phone.getByText("Reset time unknown", { exact: true }).waitFor();
  record("authenticated phone shows stopped quota interruption", await phone.locator(".phone-agent").isVisible());
  await phone.getByText("trigger typed quota failure", { exact: true }).waitFor();
  await phone.getByText("working", { exact: true }).waitFor();
  record("phone retains source transcript after stop", true);
  await phone.screenshot({ path: join(out, "phone-stopped-unknown-reset.png"), fullPage: true, animations: "disabled" });

  await desktop.goto(`${fixture.desktop}/settings`, { waitUntil: "domcontentloaded" });
  await desktop.getByRole("tab", { name: "Tasks", exact: true }).click();
  const setting = desktop.getByRole("checkbox", { name: "Auto continue after quota reset" });
  await setting.waitFor();
  if (await setting.isChecked()) {
    const saved = desktop.waitForResponse(response => response.url().endsWith("/api/config") && response.request().method() === "PUT");
    await setting.click();
    const response = await saved;
    record("desktop auto-continue toggle saved false", response.status() === 200 && !(await setting.isChecked()));
  } else {
    record("desktop auto-continue toggle is already false", true);
  }
  record("auto-continue false persisted", (await api("/api/config")).auto_continue === false);
  await desktop.screenshot({ path: join(out, "desktop-auto-continue-off.png"), fullPage: true, animations: "disabled" });

  record("no browser exceptions", errors.length === 0, errors.join("; "));
} finally {
  writeFileSync(join(out, "receipt.json"), JSON.stringify({ checks, errors }, null, 2));
  await browser.close();
}
if (checks.some(check => !check.pass)) process.exitCode = 1;
