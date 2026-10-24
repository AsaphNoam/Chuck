// Project-scoped groups and selectors (FS-02.A61–A63, FS-13.A24, FS-21.A41) on the actual
// embedded desktop + authenticated phone routes, using the group journey's fixture.
// After make embed, start from repo root:
// CHUCK_GROUP_BROWSER_READY=/tmp/groups.json go test -tags sqlite_fts5 ./internal/server -run '^TestGroupBrowserFixture$' -timeout 20m -v
// Then: node ui/scripts/project-scope-journey.mjs /tmp/groups.json /tmp/project-scope-run
// Finish the fixture by creating /tmp/groups.json.done after the browser exits.
import { readFileSync, mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";
const fixture = JSON.parse(readFileSync(process.argv[2], "utf8"));
const out = process.argv[3] || "/tmp/chuck-project-scope-journey";
mkdirSync(out, { recursive: true });
const browser = await chromium.launch();
const receipts = [];
const errors = [];
const check = (name, pass) => { receipts.push({ name, pass }); if (!pass) throw new Error(name); console.log(`PASS ${name}`); };
async function api(path, body, method = body === undefined ? "GET" : "POST") {
  const r = await fetch(fixture.desktop + path, { method, headers: { "Content-Type": "application/json" }, body: body === undefined ? undefined : JSON.stringify(body) });
  const data = await r.json();
  if (!r.ok) throw new Error(`${path}: ${JSON.stringify(data)}`);
  return data;
}
const launch = async (name, group, project) => (await api("/api/sessions", { name, group, project, role: "implementer", backend: "claude", model: "sonnet", interface: "chat" })).agent.agent_id;
const seedTray = (sourceId) => ({ state: { bySource: { [sourceId]: [{ seq: 1, excerpt: "Seeded excerpt", instruction: "Seeded instruction" }] }, editedAt: { [sourceId]: Date.now() } }, version: 0 });
const optionTexts = (select) => select.locator("option").allTextContents();
const suggestions = async (page) => (await page.getByRole("listbox").getByRole("button").allTextContents());

const desktopContext = await browser.newContext({ viewport: { width: 1024, height: 900 } });
const phoneContext = await browser.newContext({ viewport: { width: 390, height: 844 }, ignoreHTTPSErrors: true });
await phoneContext.addCookies([{ name: fixture.cookie, value: fixture.token, url: fixture.phone, secure: true, httpOnly: true, sameSite: "Strict" }]);
const desktop = await desktopContext.newPage();
const phone = await phoneContext.newPage();
for (const page of [desktop, phone]) page.on("pageerror", (e) => errors.push(e.message));
try {
  const projects = await api("/api/projects");
  await api("/api/projects/my-app", { ...projects["my-app"], cwd: "/tmp" }, "PUT");
  for (const session of await api("/api/sessions")) await api(`/api/sessions/${session.agent_id}/archive`, {}, "POST");
  // Legacy shared collapse choice: "Shared" collapsed everywhere until a project overrides it.
  const layout = await api("/api/layout");
  await api("/api/layout", { ...layout, groups: { Shared: { collapsed: true } }, projectGroups: undefined }, "PUT");

  const mine = await launch("Mine", "Shared", "my-app");
  const stopped = await launch("Stopped mine", "Solo", "my-app");
  const archived = await launch("Archived mine", "Gone", "my-app");
  const theirs = await launch("Theirs", "Shared", "other");
  const theirsOnly = await launch("Theirs only", "Elsewhere", "other");
  await api(`/api/sessions/${stopped}/stop`, {}, "POST");
  await api(`/api/sessions/${archived}/archive`, {}, "POST");

  // A61/A62 — desktop New agent on a project page (fixed launch project).
  await desktop.goto(fixture.desktop + "/project/my-app");
  await desktop.getByRole("button", { name: "New agent", exact: true }).click();
  await desktop.getByText("Advanced", { exact: true }).click();
  const dialog = desktop.getByRole("dialog");
  const group = dialog.getByRole("combobox", { name: "Group", exact: true });
  await group.focus();
  const mySuggestions = await suggestions(desktop);
  check("A61 desktop: launch project's labels incl. stopped", mySuggestions.includes("Shared") && mySuggestions.includes("Solo"));
  check("A61 desktop: no archived or other-project labels", !mySuggestions.includes("Gone") && !mySuggestions.includes("Elsewhere"));
  await group.fill("Sha");
  await group.press("Enter");
  check("A62 desktop: Enter keeps partial text, closes list, dialog stays", (await group.inputValue()) === "Sha" && (await desktop.getByRole("listbox").count()) === 0 && (await dialog.isVisible()));
  await group.fill("_ungrouped");
  check("A62 desktop: reserved name blocks Launch", await dialog.getByRole("button", { name: "Launch", exact: true }).isDisabled());
  await group.fill("Typed team");
  await group.press("Tab");
  await dialog.getByRole("textbox", { name: "Name", exact: true }).fill("Kept");
  await desktop.screenshot({ path: join(out, "desktop-new-agent.png"), fullPage: true });
  await dialog.getByRole("button", { name: "Launch", exact: true }).click();
  await dialog.waitFor({ state: "detached" });
  const typed = (await api("/api/sessions")).find((s) => s.name === "Kept");
  check("A62 desktop: Tab-retained typed text becomes the group", typed?.group === "Typed team" && typed?.project === "my-app");
  await api(`/api/sessions/${typed.agent_id}/archive`, {}, "POST");

  // A63 — collapse is per project, legacy shared choice is the initial value.
  await desktop.goto(fixture.desktop + "/project/my-app");
  const shared = desktop.getByRole("region", { name: "Group Shared", exact: true });
  await shared.waitFor();
  check("A63: legacy choice collapses Shared in my-app", (await shared.getAttribute("data-state")) === "collapsed");
  await shared.locator("header > button").first().click();
  check("A63: toggle expands Shared in my-app", (await shared.getAttribute("data-state")) === "expanded");
  await desktop.waitForTimeout(700);
  await desktop.goto(fixture.desktop + "/project/other");
  await shared.waitFor();
  check("A63: other project keeps the legacy collapsed choice", (await shared.getAttribute("data-state")) === "collapsed");
  await desktop.goto(fixture.desktop + "/project/my-app");
  await shared.waitFor();
  check("A63: my-app choice survives navigation/reload", (await shared.getAttribute("data-state")) === "expanded");
  const saved = await api("/api/layout");
  check("A63: only my-app override stored, legacy kept", saved.projectGroups?.["my-app"]?.Shared?.collapsed === false && !saved.projectGroups?.other && saved.groups?.Shared?.collapsed === true);
  await desktop.screenshot({ path: join(out, "desktop-project-collapse.png"), fullPage: true });

  // A41 — room setup lists the origin project's agents first.
  await desktop.getByRole("button", { name: "Think Tank", exact: true }).click();
  const setup = desktop.getByRole("dialog");
  const existing = setup.getByRole("combobox", { name: "Existing agent" });
  await existing.waitFor();
  let names = await optionTexts(existing);
  check("A41: origin project only by default", names.some((n) => n.startsWith("Mine")) && !names.some((n) => n.startsWith("Theirs")));
  await setup.getByLabel("Include other projects").check();
  names = await optionTexts(existing);
  check("A41: outside agents revealed with project", names.some((n) => n.startsWith("Theirs (Other project)")));
  await existing.selectOption(theirs);
  await setup.getByRole("button", { name: "Add", exact: true }).click();
  await existing.selectOption(theirsOnly);
  await setup.getByLabel("Include other projects").uncheck();
  check("A41: added outside row stays, unadded outside pick clears", (await setup.getByText("Theirs", { exact: true }).count()) === 1 && (await existing.inputValue()) === "");
  await desktop.screenshot({ path: join(out, "desktop-room-setup.png"), fullPage: true });
  await setup.getByRole("button", { name: "Cancel", exact: true }).click();

  // A24 — desktop agent annotation recipients.
  await desktop.evaluate((value) => localStorage.setItem("chuck-annotation-tray", JSON.stringify(value)), seedTray(mine));
  await desktop.goto(fixture.desktop + `/agent/${mine}`);
  const tray = desktop.getByRole("complementary", { name: "Pending annotations" });
  await tray.waitFor();
  await tray.getByLabel("Another agent").check();
  const recipient = tray.getByLabel("Annotation recipient");
  names = await optionTexts(recipient);
  check("A24 desktop: no other-project recipients by default", !names.some((n) => n.startsWith("Theirs")));
  await tray.getByLabel("Include other projects").check();
  names = await optionTexts(recipient);
  check("A24 desktop: outside recipients labelled by project", names.some((n) => n.startsWith("Theirs") && n.includes("@other")));
  await recipient.selectOption(theirs);
  await tray.getByLabel("Include other projects").uncheck();
  check("A24 desktop: hiding clears outside selection, keeps draft", (await recipient.inputValue()) === "" && (await tray.getByRole("textbox", { name: "Instruction", exact: true }).inputValue()) === "Seeded instruction");
  await desktop.screenshot({ path: join(out, "desktop-annotation-tray.png"), fullPage: true });

  // A61 — the New task launch has a project chooser; switching it clears the group.
  await tray.getByLabel("New task").check();
  await tray.getByRole("button", { name: "Continue to launch" }).click();
  const launchDialog = desktop.getByRole("dialog");
  await launchDialog.getByText("Advanced", { exact: true }).click();
  const launchGroup = launchDialog.getByRole("combobox", { name: "Group", exact: true });
  await launchGroup.fill("Draft");
  await launchDialog.getByRole("textbox", { name: "Name", exact: true }).fill("Kept name");
  await launchDialog.getByLabel("Project", { exact: true }).selectOption("other");
  await desktop.waitForFunction(() => document.querySelector("#new-agent-group")?.value === "", null, { timeout: 5000 }).catch(() => undefined);
  check("A61 desktop: project change clears group, keeps name", (await launchGroup.inputValue()) === "" && (await launchDialog.getByRole("textbox", { name: "Name", exact: true }).inputValue()) === "Kept name");
  await launchGroup.focus();
  const otherSuggestions = await suggestions(desktop);
  check("A61 desktop: other project's labels only", otherSuggestions.includes("Elsewhere") && !otherSuggestions.includes("Solo"));
  await launchDialog.getByRole("button", { name: "Cancel", exact: true }).click();

  // A61 phone — Manage group suggestions and direct entry.
  await phone.goto(fixture.phone + `/agent/${mine}`);
  await phone.getByRole("tab", { name: "Manage", exact: true }).click();
  const phoneGroup = phone.getByRole("combobox", { name: "Group", exact: true });
  await phoneGroup.focus();
  const phoneSuggestions = await suggestions(phone);
  check("A61 phone: project-only labels incl. stopped", phoneSuggestions.includes("Solo") && !phoneSuggestions.includes("Elsewhere") && !phoneSuggestions.includes("Gone"));
  await phoneGroup.fill("Phone team");
  await phoneGroup.press("Enter");
  await phone.getByRole("button", { name: "Save group", exact: true }).click();
  await desktop.goto(fixture.desktop + "/project/my-app");
  await desktop.getByRole("region", { name: "Group Phone team", exact: true }).waitFor();
  check("A62 phone: Enter-accepted text saved and reaches desktop", (await api(`/api/sessions/${mine}`)).agent.group === "Phone team");

  // A24 phone — annotation sheet recipients.
  await phone.evaluate((value) => localStorage.setItem("chuck-annotation-tray", JSON.stringify(value)), seedTray(mine));
  await phone.goto(fixture.phone + `/agent/${mine}`);
  await phone.getByRole("button", { name: /Pending annotations/ }).click();
  await phone.getByRole("radio", { name: "Another agent" }).click();
  const phoneRecipient = phone.getByRole("combobox", { name: "Agent" });
  const phoneEmpty = phone.getByText("No other chat agent is running in this project.");
  check("A24 phone: no other-project recipients by default", (await phoneEmpty.count()) === 1 || !(await optionTexts(phoneRecipient)).some((n) => n.startsWith("Theirs")));
  await phone.getByLabel("Include other projects").check();
  check("A24 phone: outside recipients labelled by project", (await optionTexts(phoneRecipient)).some((n) => n.startsWith("Theirs") && n.includes("@other")));
  for (const width of [360, 390, 430]) {
    await phone.setViewportSize({ width, height: 844 });
    check(`phone ${width}px no overflow`, await phone.evaluate(() => document.documentElement.scrollWidth <= innerWidth));
    await phone.screenshot({ path: join(out, `phone-annotations-${width}.png`), fullPage: true });
  }
  check("no browser exceptions", errors.length === 0);
} finally {
  writeFileSync(join(out, "report.json"), JSON.stringify({ receipts, errors }, null, 2));
  await browser.close();
}
