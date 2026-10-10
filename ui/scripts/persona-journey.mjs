// Run after make embed against the opt-in isolated TestGroupBrowserFixture.
// node ui/scripts/persona-journey.mjs <fixture.json> <receipt-directory> [vite-origin]
import { readFileSync, mkdirSync, writeFileSync, unlinkSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const fixture = JSON.parse(readFileSync(process.argv[2], "utf8"));
const out = process.argv[3];
mkdirSync(out, { recursive: true });
const checks = [];
const errors = [];
function check(name, pass) {
  checks.push({ name, pass });
  if (!pass) throw new Error(name);
  console.log(`PASS ${name}`);
}
async function api(path, body, method = body === undefined ? "GET" : "POST") {
  const response = await fetch(fixture.desktop + path, {
    method, headers: { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const data = await response.json();
  if (!response.ok) throw new Error(`${path}: ${JSON.stringify(data)}`);
  return data;
}
const browser = await chromium.launch();
const desktop = await browser.newPage({ viewport: { width: 1024, height: 900 } });
const phoneContext = await browser.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true });
await phoneContext.addCookies([{ name: fixture.cookie, value: fixture.token, url: fixture.phone, secure: true, httpOnly: true, sameSite: "Strict" }]);
const phone = await phoneContext.newPage();
for (const page of [desktop, phone]) page.on("pageerror", error => errors.push(error.message));
async function keyboardToggle(page, summary) {
  await summary.focus();
  await page.keyboard.press("Enter");
}
async function submitAndCheckRole(page, button, expected, name) {
  const request = page.waitForRequest(request => request.url().endsWith("/api/sessions") && request.method() === "POST");
  await button.focus();
  await page.keyboard.press("Enter");
  const sent = await request;
  check(name, sent.postDataJSON().role === expected && !Object.hasOwn(sent.postDataJSON(), "chuck_aware"));
  const response = await sent.response();
  check(`${name}: launch succeeded`, response?.status() === 201);
  return (await response.json()).agent.agent_id;
}
try {
  const projects = await api("/api/projects");
  await api("/api/projects/my-app", { ...projects["my-app"], cwd: "/tmp" }, "PUT");
  const roles = await api("/api/roles");
  await api(roles.quiet ? "/api/roles/quiet" : "/api/roles", { role: "quiet", title: "Quiet persona", system_prompt: "persona text", chuck_aware: false, skip_permissions: null }, roles.quiet ? "PUT" : "POST");
  for (const skin of ["", "sky-grove", "studio"]) {
    const name = skin || "core";
    await api("/api/config", { appearance_skin: skin, default_role: "default", onboarding_complete: true }, "PUT");
    await desktop.goto(fixture.desktop + "/project/my-app");
    await desktop.getByRole("button", { name: /^New agent$/i }).first().click();
    let dialog = desktop.getByRole("dialog");
    await dialog.locator(".new-agent-runtime").getByText("No persona", { exact: true }).waitFor();
    check(`${name}: desktop ordinary path is compact`, !(await dialog.locator("details").evaluate(element => element.open)) && await dialog.getByRole("checkbox", { name: "Chuck aware agent" }).count() === 0);
    await desktop.screenshot({ path: join(out, `${name}-desktop-default.png`), animations: "disabled" });
    const id = await submitAndCheckRole(desktop, dialog.getByRole("button", { name: "Launch", exact: true }), "default", `${name}: desktop No persona request`);
    await desktop.getByRole("dialog").waitFor({ state: "hidden" });
    await desktop.getByRole("button", { name: /^New agent$/i }).first().click();
    dialog = desktop.getByRole("dialog");
    await keyboardToggle(desktop, dialog.locator("summary"));
    await dialog.getByLabel("Persona", { exact: true }).selectOption("quiet");
    await dialog.getByLabel("Name", { exact: true }).fill(`Quiet ${name}`);
    await keyboardToggle(desktop, dialog.locator("summary"));
    check(`${name}: desktop collapsed summary names selection`, await dialog.locator(".new-agent-runtime").getByText("Quiet persona", { exact: true }).count() === 1);
    await keyboardToggle(desktop, dialog.locator("summary"));
    check(`${name}: desktop draft retained`, await dialog.getByLabel("Persona", { exact: true }).inputValue() === "quiet" && await dialog.getByLabel("Name", { exact: true }).inputValue() === `Quiet ${name}`);
    check(`${name}: internal Default is never a persona label`, !(await dialog.getByLabel("Persona", { exact: true }).locator("option").allTextContents()).includes("Default"));
    await desktop.screenshot({ path: join(out, `${name}-desktop-advanced.png`), animations: "disabled" });
    await submitAndCheckRole(desktop, dialog.getByRole("button", { name: "Launch", exact: true }), "quiet", `${name}: desktop selected persona request`);
    await desktop.getByRole("dialog").waitFor({ state: "hidden" });
    await desktop.goto(fixture.desktop + `/agent/${id}`);
    await desktop.getByText("No persona", { exact: true }).waitFor();
    check(`${name}: desktop agent identity labels empty choice`, true);

    await phone.goto(fixture.phone + "/project/my-app");
    await phone.getByRole("button", { name: "New agent", exact: true }).click();
    let form = phone.getByRole("form", { name: "New agent" });
    await form.getByText(/No persona/).first().waitFor();
    check(`${name}: phone default advanced closed`, !(await form.locator("details").evaluate(element => element.open)));
    await phone.screenshot({ path: join(out, `${name}-phone-default.png`), animations: "disabled" });
    await submitAndCheckRole(phone, form.getByRole("button", { name: "Create agent", exact: true }), "default", `${name}: phone No persona request`);
    await form.waitFor({ state: "hidden" });
    await phone.goto(fixture.phone + "/project/my-app");
    await phone.getByRole("button", { name: "New agent", exact: true }).click();
    form = phone.getByRole("form", { name: "New agent" });
    await keyboardToggle(phone, form.locator("summary"));
    await form.getByLabel("Persona", { exact: true }).selectOption("quiet");
    await form.getByLabel("Name", { exact: true }).fill(`Phone quiet ${name}`);
    await keyboardToggle(phone, form.locator("summary"));
    check(`${name}: phone collapsed summary names persona`, await form.getByText(/Quiet persona/).first().isVisible());
    await keyboardToggle(phone, form.locator("summary"));
    check(`${name}: phone draft retained`, await form.locator("select").first().inputValue() === "quiet" && await form.getByLabel("Name", { exact: true }).inputValue() === `Phone quiet ${name}`);
    for (const width of [360, 390, 430]) {
      await phone.setViewportSize({ width, height: 844 });
      check(`${name}: phone ${width}px has no horizontal overflow`, await phone.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth));
      await phone.screenshot({ path: join(out, `${name}-phone-${width}-advanced.png`), animations: "disabled" });
    }
    await phone.setViewportSize({ width: 390, height: 844 });
    await phone.screenshot({ path: join(out, `${name}-phone-advanced.png`), animations: "disabled" });
    await submitAndCheckRole(phone, form.getByRole("button", { name: "Create agent", exact: true }), "quiet", `${name}: phone selected persona request`);
    await form.waitFor({ state: "hidden" });

    await api("/api/config", { default_role: "quiet" }, "PUT");
    await desktop.goto(fixture.desktop + "/project/my-app");
    await desktop.getByRole("button", { name: /^New agent$/i }).first().click();
    await desktop.getByRole("dialog").locator(".new-agent-runtime").getByText("Quiet persona", { exact: true }).waitFor();
    check(`${name}: saved custom default is honestly named`, true);
    await desktop.getByRole("dialog").getByRole("button", { name: "Cancel", exact: true }).click();
    await desktop.goto(fixture.desktop + "/settings");
    await desktop.getByRole("tab", { name: "Roles", exact: true }).click();
    check(`${name}: canonical Default hidden from management`, await desktop.getByText("Default", { exact: true }).count() === 0);
    await desktop.getByText("Quiet persona", { exact: true }).locator("xpath=ancestor::li").getByRole("button", { name: "Edit", exact: true }).click();
    const checkbox = desktop.getByRole("checkbox", { name: "Chuck aware agent" });
    check(`${name}: editor preserves opt out and accessible help`, !(await checkbox.isChecked()) && Boolean(await checkbox.getAttribute("aria-describedby")));
    await desktop.screenshot({ path: join(out, `${name}-persona-editor.png`), animations: "disabled" });
    await checkbox.focus();
    await desktop.keyboard.press("Space");
    const save = desktop.waitForResponse(response => response.url().endsWith("/api/roles/quiet") && response.request().method() === "PUT");
    await desktop.getByRole("button", { name: "Update", exact: true }).click();
    check(`${name}: editor saves explicit awareness`, (await save).status() === 200 && (await api("/api/roles")).quiet.chuck_aware === true);
    await api("/api/roles/quiet", { title: "Quiet persona", system_prompt: "persona text", chuck_aware: false, skip_permissions: null }, "PUT");

    const readinessMarker = join(fixture.home, "persona-readiness-blocked");
    writeFileSync(readinessMarker, "synthetic missing credentials", { mode: 0o600 });
    await api("/api/config", { default_role: "default", onboarding_complete: false }, "PUT");
    await desktop.goto(fixture.desktop);
    await desktop.getByRole("button", { name: "Validate & Continue", exact: true }).waitFor();
    unlinkSync(readinessMarker);
    await desktop.getByRole("button", { name: "Validate & Continue", exact: true }).click();
    await desktop.getByPlaceholder("e.g. My App").fill(`Persona onboarding ${name}`);
    await desktop.getByPlaceholder("~/Projects/my-app").fill("/tmp");
    await desktop.getByRole("button", { name: "Create project", exact: true }).click();
    // Source linking is optional; this journey tests ordinary creation.
    const sourceContinue = desktop.getByRole("button", { name: "Continue", exact: true });
    await sourceContinue.waitFor();
    await sourceContinue.click();
    const launchStep = desktop.locator('[data-ui="onboarding"][data-variant="launch"]');
    await launchStep.locator(".new-agent-runtime").getByText("No persona", { exact: true }).waitFor();
    check(`${name}: onboarding ordinary path has no persona decision`, !(await launchStep.locator("details").evaluate(element => element.open)) && await launchStep.getByRole("checkbox", { name: "Chuck aware agent" }).count() === 0);
    await desktop.screenshot({ path: join(out, `${name}-onboarding-default.png`), animations: "disabled" });
    await keyboardToggle(desktop, launchStep.locator("summary"));
    await launchStep.getByLabel("Persona", { exact: true }).selectOption("quiet");
    await launchStep.getByLabel("Agent name", { exact: true }).fill(`Onboarding quiet ${name}`);
    await keyboardToggle(desktop, launchStep.locator("summary"));
    check(`${name}: onboarding summary retains choice`, await launchStep.locator(".new-agent-runtime").getByText("Quiet persona", { exact: true }).isVisible());
    await keyboardToggle(desktop, launchStep.locator("summary"));
    check(`${name}: onboarding draft retained`, await launchStep.getByLabel("Persona", { exact: true }).inputValue() === "quiet");
    await submitAndCheckRole(desktop, launchStep.getByRole("button", { name: "Launch", exact: true }), "quiet", `${name}: onboarding chosen persona request`);
    await desktop.getByRole("dialog").waitFor({ state: "hidden" });

    if (process.argv[4]) {
      await desktop.goto(process.argv[4] + "/__visual-matrix");
      await desktop.locator(".visual-matrix").waitFor();
      await desktop.getByLabel("Fixture appearance").selectOption(skin || "core");
      await desktop.screenshot({ path: join(out, `${name}-development-matrix.png`), fullPage: true, animations: "disabled" });
      check(`${name}: development matrix rendered`, true);
    }
  }
  check("no browser page errors", errors.length === 0);
} finally {
  await desktop.screenshot({ path: join(out, "last-desktop.png") });
  await phone.screenshot({ path: join(out, "last-phone.png") });
  writeFileSync(join(out, "receipt.json"), JSON.stringify({ checks, errors }, null, 2));
  await browser.close();
}
