import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync, accessSync, constants } from "node:fs";
import { execFileSync } from "node:child_process";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { chromium } from "playwright";

const base = process.argv[2] ?? "http://127.0.0.1:4426";
const fixtureHome = process.argv[4] ?? "";
assert.match(fixtureHome, /\/chuck-stress-\d+$/, "supply the disposable stress fixture home as argument 4");
const fixtureConfig = JSON.parse(readFileSync(join(fixtureHome, "config.json"), "utf8"));
assert.equal(base, `http://127.0.0.1:${fixtureConfig.port}`, "base must match isolated fixture loopback port");
const backends = JSON.parse(readFileSync(join(fixtureHome, "backends.json"), "utf8"));
assert.equal(backends.backends.claude.env.FAKEACP_SCENARIO, "hold_turn", "only fake ACP hold_turn fixtures are supported");
accessSync(join(fixtureHome, "fakeacp"), constants.X_OK);
// stress-fixture overrides the chat command in memory (not backends.json).
// Verify the actual listening program is that fixture before any launch.
const listener = execFileSync("/usr/sbin/lsof", ["-nP", `-iTCP:${fixtureConfig.port}`, "-sTCP:LISTEN", "-Fp"], { encoding: "utf8" }).trim().split("\n").find((line) => /^p\d+$/.test(line))?.slice(1);
assert.ok(listener, "fixture listener exists");
const command = execFileSync("/bin/ps", ["-p", listener, "-o", "args="], { encoding: "utf8" });
assert.match(command, /stress-fixture/);
assert.ok(command.includes(`-port ${fixtureConfig.port}`) && command.includes("-scenario hold_turn"), "listener must be the fake-provider fixture command");
const out = process.argv[3] ?? mkdtempSync(join(tmpdir(), "chuck-file-editing-"));
mkdirSync(out, { recursive: true });
const path = join(out, "source.md");
const original = "# Exact source\r\n\r\nOriginal snow 雪\n";
writeFileSync(path, original);
async function api(route, method = "GET", body) {
  const response = await fetch(base + route, { method, headers: { "Content-Type": "application/json" }, body: body && JSON.stringify(body) });
  const result = await response.json();
  assert.ok(response.ok, `${method} ${route}: ${JSON.stringify(result)}`);
  return result;
}
const launched = await api("/api/sessions", "POST", { role: "implementer", project: "stress", backend: "claude", model: "haiku", name: "File edit proof", interface: "chat" });
const id = launched.agent.agent_id;
const url = (archive = false) => `${base}/${archive ? "archive" : "agent"}/${id}?file=${encodeURIComponent(path)}`;
const browser = await chromium.launch();
try {
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  page.setDefaultTimeout(15000);
  const errors = [];
  page.on("pageerror", (error) => errors.push(error.message));
  await page.goto(url());
  await page.getByRole("button", { name: "Edit", exact: true }).waitFor();
  const before = await api(`/api/sessions/${id}/transcript`);
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  const editor = page.getByRole("textbox", { name: "File source draft" });
  await editor.fill("# Edited\nExact 雪\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await editor.waitFor({ state: "detached" });
  assert.equal(readFileSync(path, "utf8"), "# Edited\nExact 雪\n");
  assert.deepEqual((await api(`/api/sessions/${id}/transcript`)).events, before.events);
  console.log("PASS live save exact disk bytes and no transcript event");

  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await editor.fill("# My conflict draft\n");
  writeFileSync(path, "# External disk version\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await page.getByRole("button", { name: "Use this version as save base" }).waitFor();
  assert.equal(readFileSync(path, "utf8"), "# External disk version\n");
  assert.equal(await editor.inputValue(), "# My conflict draft\n");
  await page.getByRole("button", { name: "Use this version as save base" }).click();
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await editor.waitFor({ state: "detached" });
  assert.equal(readFileSync(path, "utf8"), "# My conflict draft\n");
  console.log("PASS external conflict retains draft and requires explicit base adoption and second Save");

  await page.emulateMedia({ reducedMotion: "reduce" });
  const body = page.locator(".file-viewer-body");
  for (let index = 0; index < 6; index++) {
    writeFileSync(path, `# Continuous ${index}\n`);
    await page.waitForTimeout(600);
    assert.ok((await body.innerText()).includes("My conflict draft"));
  }
  await page.getByRole("heading", { name: "Continuous 5", exact: true }).waitFor({ timeout: 9000 });
  await body.evaluate((element) => {
    const text = document.createTreeWalker(element, NodeFilter.SHOW_TEXT).nextNode();
    const range = document.createRange(); range.selectNodeContents(text);
    const selection = getSelection(); selection.removeAllRanges(); selection.addRange(range);
    document.dispatchEvent(new Event("selectionchange"));
  });
  writeFileSync(path, "# Selection delayed refresh\n");
  await page.waitForTimeout(4500);
  assert.ok((await body.innerText()).includes("Continuous 5"));
  await page.evaluate(() => { getSelection().removeAllRanges(); document.dispatchEvent(new Event("selectionchange")); });
  await page.getByRole("heading", { name: "Selection delayed refresh", exact: true }).waitFor({ timeout: 9000 });
  console.log("PASS settled refresh retains old content through continuous writes and active selection with reduced motion");

  await page.getByRole("button", { name: "Focus file", exact: true }).click();
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await editor.fill("# Reload recovery\n");
  await page.getByText("Unsaved edits · recovery available", { exact: true }).waitFor();
  await page.reload();
  await editor.waitFor();
  assert.equal(await editor.inputValue(), "# Reload recovery\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await editor.waitFor({ state: "detached" });
  assert.equal(readFileSync(path, "utf8"), "# Reload recovery\n");
  console.log("PASS Focus file edit and durable reload recovery");

  for (const skin of ["core", "sky-grove", "studio", "boring-night"]) {
    await api("/api/config", "PUT", { appearance_skin: skin === "core" ? "" : skin });
    for (const width of [1024, 1440]) {
      await page.setViewportSize({ width, height: 900 }); await page.goto(url());
      await page.getByRole("button", { name: "Edit", exact: true }).click();
      await editor.waitFor();
      await page.screenshot({ path: join(out, `editor-${skin}-${width}.png`), fullPage: true });
      const overflow = await editor.evaluate((element) => element.getBoundingClientRect().right > innerWidth);
      assert.equal(overflow, false);
      await page.getByRole("button", { name: "Cancel", exact: true }).click();
      await editor.waitFor({ state: "detached" });
    }
  }
  console.log("PASS editor layouts four appearances at 1024/1440");
  await api(`/api/sessions/${id}/archive`, "POST");
  await page.goto(url(true));
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await editor.fill("# Archived save\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await editor.waitFor({ state: "detached" });
  assert.equal(readFileSync(path, "utf8"), "# Archived save\n");
  const launch = async (name) => (await api("/api/sessions", "POST", { role: "implementer", project: "stress", backend: "claude", model: "haiku", name, interface: "chat" })).agent;
  const members = await Promise.all([launch("File room one"), launch("File room two")]);
  const room = await api("/api/think-tanks", "POST", {
    command_id: `file-room-${Date.now()}`, title: "File editing proof", goal: "Review the test file", origin_project: "stress", openings: true,
    participants: members.map((agent) => ({ agent_id: agent.agent_id, limit: 1, may_leave: true })),
    judge: { role: "implementer", project: "stress", backend: "claude", model: "haiku", name: "File room judge" },
  });
  const sources = await api(`/api/think-tanks/${room.room_id}/files`);
  assert.ok(sources.sources[0]?.source_id, "room retained source available");
  await page.goto(`${base}/think-tank/${room.room_id}?file=${encodeURIComponent(path)}&source=${encodeURIComponent(sources.sources[0].source_id)}`);
  await page.getByRole("button", { name: "Edit", exact: true }).click();
  await editor.fill("# Retained room save\n");
  await page.getByRole("button", { name: "Save", exact: true }).click();
  await editor.waitFor({ state: "detached" });
  assert.equal(readFileSync(path, "utf8"), "# Retained room save\n");
  console.log("PASS retained room source file save through actual room reader");
  assert.deepEqual(errors, []);
  console.log(`PASS archive save; no browser errors. Screenshots: ${out}`);
} catch (error) {
  const failed = browser.contexts()[0]?.pages()[0];
  if (failed) {
    await failed.screenshot({ path: join(out, "failure.png"), fullPage: true });
    console.error((await failed.locator("body").innerText()).slice(0, 5000));
  }
  throw error;
} finally { await browser.close(); }
