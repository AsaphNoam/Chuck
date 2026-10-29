// Real-browser journey for run workspaces (new worktree start, run detail, removal, Workspaces tab, phone).
// node scripts/pipeline-workspace-journey.mjs [outDir]
// Isolated API fixtures; this proves UI behavior and request bodies, not backend git correctness.
import { mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import assert from "node:assert/strict";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), "pipeline-workspace-journey")] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });
const at = "2026-10-29T09:00:00Z";
const origin = "http://127.0.0.1:5198";
const template = { version: 2, title: "Feature delivery", orchestrator_role: "implementer", inputs: [], stages: ["Design", "Implement"].map((title, i) => ({ id: `stage_${i}`, title, objective: `Complete ${title.toLowerCase()}.`, coordination: "standing", inputs: [], outputs: [] })) };
const records = [{ id: "delivery", template, valid: true, diagnostics: [] }];
const summary = (id, name, state) => ({ run_id: id, template_id: "delivery", display_name: name, project: "Chuck", state, revision: 3, pending_action: "", current_stage_id: "stage_1", current_stage_title: "Implement", current_agent_id: "", attention_reason: "", final_outcome: state === "stopped" ? "stopped" : "", updated_at: at, diagnostics: [] });
const runSummaries = [summary("run_live", "Live worktree run", "running"), summary("run_dirty", "Stopped dirty run", "stopped"), summary("run_unknown", "Stopped unknown run", "stopped")];
const workspace = (id, runID, runName, extra = {}) => ({ workspace_id: id, run_id: runID, project: "Chuck", project_name: "Chuck", run_name: runName, mode: "new_worktree", branch: `feat/${id}`, cwd: `/tmp/worktrees/${id}`, state: "ready", owned: true, revision: 7, setup_ok: true, removal_eligible: true, created_at: at, updated_at: at, ...extra });
const baseWorkspaces = {
  ws_live: workspace("ws_live", "run_live", "Live worktree run"),
  ws_dirty: workspace("ws_dirty", "run_dirty", "Stopped dirty run"),
  ws_unknown: workspace("ws_unknown", "run_unknown", "Stopped unknown run"),
  ws_gone: workspace("ws_gone", "run_deleted", "Deleted release run", { project_name: "Old Project", branch: "feat/gone" }),
};
let workspaces = structuredClone(baseWorkspaces);
const wsDetail = { ws_live: { dirty: false, dirty_known: true }, ws_dirty: { dirty: true, dirty_known: true }, ws_unknown: { dirty: false, dirty_known: false }, ws_gone: { dirty: false, dirty_known: true } };
const runWorkspace = { run_live: "ws_live", run_dirty: "ws_dirty", run_unknown: "ws_unknown" };
const detail = (s) => ({ run: { ...s, template_snapshot: template, goal: "Deliver the change.", inputs: {}, current_task_id: "task_1", current_attempt_id: "", created_at: at }, template, inputs: {}, stage_tasks: [], values: [], diagnostics: [], workspace: workspaces[runWorkspace[s.run_id]], controls: Object.fromEntries(["continue", "retry", "replace", "stop", "repair_cleanup"].map((key) => [key, { eligible: key === "stop" && s.state === "running", reason: "" }])) });

const server = await createServer({ server: { host: "127.0.0.1", port: 5198, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
let checks = 0;
const ok = (value, message) => { assert.ok(value, message); checks++; };
const equal = (actual, expected, message) => { assert.deepEqual(actual, expected, message); checks++; };

async function open(viewport, mobile = false) {
  workspaces = structuredClone(baseWorkspaces);
  const context = await browser.newContext({ viewport, ...(mobile ? { isMobile: true, hasTouch: true } : {}) });
  await context.addInitScript(() => {
    delete window.SharedWorker;
    window.EventSource = class extends EventTarget {
      constructor() { super(); setTimeout(() => { this.onopen?.(); this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: "__hydrated__", data: { hydrated: true } }) })); }); }
      close() {}
    };
  });
  const state = { starts: [], removes: [], startMode: "reject", workspaceListProjects: [], errors: [] };
  await context.route((url) => url.pathname.startsWith("/api/"), async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const method = request.method();
    if (path === "/api/pipeline-runs" && method === "POST") {
      const body = request.postDataJSON();
      state.starts.push(body);
      if (body.workspace?.branch === "taken") return route.fulfill({ status: 422, json: { error: { code: "validation", message: "run cannot start", details: { pipeline_code: "validation", diagnostics: [{ field: "workspace.branch", code: "unavailable", message: 'branch "taken" already exists; choose a new branch name' }] } } } });
      return route.fulfill({ status: 201, json: { run: detail(runSummaries[0]), replay: false, workspace_conflicts: [] } });
    }
    if (path === "/api/pipeline-runs") return route.fulfill({ json: runSummaries, headers: { "X-Total-Count": String(runSummaries.length) } });
    if (path === "/api/pipelines") return route.fulfill({ json: records });
    if (path.startsWith("/api/pipeline-runs/")) {
      const found = runSummaries.find((item) => item.run_id === path.split("/").at(-1));
      return found ? route.fulfill({ json: detail(found) }) : route.fulfill({ status: 404, json: { error: { code: "not_found", message: "run not found" } } });
    }
    if (path === "/api/pipeline-workspaces") { state.workspaceListProjects.push(url.searchParams.get("project")); return route.fulfill({ json: { workspaces: Object.values(workspaces) } }); }
    const remove = path.match(/^\/api\/pipeline-workspaces\/([^/]+)\/remove$/);
    if (remove && method === "POST") {
      state.removes.push({ id: remove[1], body: request.postDataJSON() });
      workspaces[remove[1]] = { ...workspaces[remove[1]], state: "removed", revision: workspaces[remove[1]].revision + 1 };
      return route.fulfill({ json: { ...workspaces[remove[1]], ...wsDetail[remove[1]], missing: false, removable: false } });
    }
    const one = path.match(/^\/api\/pipeline-workspaces\/([^/]+)$/);
    if (one) return route.fulfill({ json: { ...workspaces[one[1]], missing: false, removable: true, ...wsDetail[one[1]] } });
    if (path === "/api/pipeline-proposals") return route.fulfill({ json: { pending: [], declined: [] } });
    if (path === "/api/projects") return route.fulfill({ json: { Chuck: { title: "Chuck", cwd: "/tmp/chuck", color: [80, 120, 200] } } });
    if (path === "/api/config") return route.fulfill({ json: { appearance_skin: "", onboarded: true } });
    if (path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer", system_prompt: "", skip_permissions: false } } });
    if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 } } });
    if (path === "/api/backends") return route.fulfill({ json: { version: 2, backends: { codex: { name: "Codex", type: "codex-acp", default: true, default_model: "gpt-6.1-sol", models: { "gpt-6.1-sol": { name: "GPT-6.1 Sol", model: "gpt-6.1-sol", fast: true } } } } } });
    if (path === "/api/remote/home") return route.fulfill({ json: { needs_you: [], active_runs: [] } });
    if (path === "/api/config-sources") return route.fulfill({ json: [] });
    return route.fulfill({ json: {} });
  });
  const page = await context.newPage();
  page.on("pageerror", (error) => state.errors.push(error.message));
  return { context, page, state };
}

const shot = async (page, name) => {
  await page.waitForTimeout(250);
  const path = join(outDir, `${name}.png`);
  await page.screenshot({ path });
  console.log(path);
};
const opaque = async (locator, label) => {
  const background = await locator.evaluate((element) => getComputedStyle(element).backgroundColor);
  ok(!/^rgba\(0, 0, 0, 0\)$|^transparent$/.test(background), `${label} dialog is unstyled (background ${background})`);
};

try {
  for (const width of [1024, 1440]) {
    // Journey 1: start dialog.
    {
      const { context, page, state } = await open({ width, height: 1000 });
      const tag = (name) => `${name}-${width}`;
      await page.goto(`${origin}/pipelines/runs`);
      await page.locator('[data-ui="pipeline-run-list"]').waitFor({ timeout: 15000 });
      await page.getByRole("button", { name: "Start run", exact: true }).click();
      const dialog = page.getByRole("dialog");
      await dialog.waitFor();
      await opaque(dialog.locator('[data-slot="content"]').first().locator("xpath=ancestor-or-self::*[contains(@class,'dialog-content')][1]"), "start");
      await dialog.locator('[data-field="template_id"] select').selectOption("delivery");
      await dialog.locator('[data-field="project"] select').selectOption("Chuck");
      await dialog.locator('[data-field="goal"] textarea').fill("Ship the change on its own branch.");
      await dialog.locator("label.pipeline-workspace-option").nth(1).locator("input").check();
      await dialog.locator('[data-field="workspace.branch"] input').fill("taken");
      await shot(page, tag("1-start-worktree-setup"));
      await dialog.getByRole("button", { name: "Review", exact: true }).click();
      await dialog.getByRole("button", { name: "Start run", exact: true }).click();
      await dialog.getByText('branch "taken" already exists; choose a new branch name').waitFor();
      equal(await dialog.locator('[data-field="workspace.branch"] input').inputValue(), "taken", "branch value kept after 422");
      equal(await dialog.getByText('branch "taken" already exists; choose a new branch name').count(), 1, "diagnostic shown once");
      equal(state.starts.length, 1, "one rejected start");
      await shot(page, tag("1-start-rejected"));
      await dialog.locator('[data-field="workspace.branch"] input').fill("ship-it");
      await dialog.getByRole("button", { name: "Review", exact: true }).click();
      await dialog.locator("dl.pipeline-review-facts").getByText("New worktree · branch ship-it").waitFor();
      ok(await dialog.locator("dl.pipeline-review-facts").getByText("New worktree · branch ship-it").isVisible(), "review names workspace choice");
      await shot(page, tag("1-start-review"));
      await dialog.getByRole("button", { name: "Start run", exact: true }).click();
      await page.waitForFunction(() => !document.querySelector('[role="dialog"]'));
      equal(state.starts.length, 2, "second start sent");
      equal(state.starts[1].workspace, { mode: "new_worktree", branch: "ship-it" }, "worktree workspace body");
      // Project checkout start sends no workspace key.
      await page.goto(`${origin}/pipelines/runs`);
      await page.locator('[data-ui="pipeline-run-list"]').waitFor();
      await page.getByRole("button", { name: "Start run", exact: true }).click();
      const second = page.getByRole("dialog");
      await second.waitFor();
      await second.locator('[data-field="template_id"] select').selectOption("delivery");
      await second.locator('[data-field="project"] select').selectOption("Chuck");
      await second.locator('[data-field="goal"] textarea').fill("Work in the project checkout.");
      ok(await second.locator("label.pipeline-workspace-option").first().locator("input").isChecked(), "Project checkout is the default");
      await second.getByRole("button", { name: "Review", exact: true }).click();
      await second.locator("dl.pipeline-review-facts").getByText("Project checkout", { exact: true }).waitFor();
      await second.getByRole("button", { name: "Start run", exact: true }).click();
      await page.waitForFunction(() => !document.querySelector('[role="dialog"]'));
      equal(state.starts.length, 3, "third start sent");
      ok(!("workspace" in state.starts[2]), "project checkout start has no workspace key");
      equal(state.errors, [], "no browser errors (start)");
      await context.close();
    }

    // Journeys 2-4: run detail and removal.
    {
      const { context, page, state } = await open({ width, height: 1000 });
      const tag = (name) => `${name}-${width}`;
      await page.goto(`${origin}/pipelines/runs/run_live`);
      const section = page.locator('[data-slot="workspace"]');
      await section.waitFor({ timeout: 15000 });
      ok(await section.getByText("New worktree", { exact: true }).isVisible(), "New worktree label");
      ok(await section.getByText("feat/ws_live").isVisible(), "branch shown");
      ok(await section.getByText("/tmp/worktrees/ws_live").isVisible(), "path shown");
      equal(await page.getByRole("button", { name: /Remove checkout/ }).count(), 0, "no remove on a running run");
      await shot(page, tag("2-running-detail"));

      await page.goto(`${origin}/pipelines/runs/run_dirty`);
      await page.locator('[data-slot="workspace"]').waitFor();
      await page.getByRole("button", { name: "Remove checkout…" }).click();
      const dialog = page.getByRole("dialog");
      await dialog.getByText("Uncommitted changes", { exact: true }).waitFor();
      ok(await dialog.getByText(/The branch, its commits and the agents' conversations are kept/).isVisible(), "kept statement");
      ok(await dialog.getByText(/changes that are not committed/).isVisible(), "dirty disclosure");
      await opaque(page.locator(".dialog-content"), "remove");
      equal(await page.evaluate(() => document.activeElement?.textContent), "Keep checkout", "Keep is focused");
      await shot(page, tag("3-remove-dirty"));
      await page.keyboard.press("Escape");
      await page.waitForFunction(() => !document.querySelector('[role="dialog"]'));
      equal(state.removes.length, 0, "Escape sends no remove");
      await page.getByRole("button", { name: "Remove checkout…" }).click();
      await dialog.getByText("Uncommitted changes", { exact: true }).waitFor();
      await dialog.getByRole("button", { name: "Remove and discard changes" }).click();
      await page.waitForFunction(() => !document.querySelector('[role="dialog"]'));
      equal(state.removes.length, 1, "one remove request");
      equal(state.removes[0], { id: "ws_dirty", body: { revision: 7, dirty_known: true, dirty: true, acknowledge_unknown: false } }, "remove body");

      await page.goto(`${origin}/pipelines/runs/run_unknown`);
      await page.locator('[data-slot="workspace"]').waitFor();
      await page.getByRole("button", { name: "Remove checkout…" }).click();
      const unknown = page.getByRole("dialog");
      await unknown.getByText("Uncommitted changes could not be checked").waitFor();
      const removeButton = unknown.getByRole("button", { name: "Remove checkout", exact: true });
      ok(await removeButton.isDisabled(), "Remove disabled until acknowledged");
      await shot(page, tag("4-remove-unknown"));
      await unknown.getByRole("checkbox").check();
      ok(await removeButton.isEnabled(), "Remove enabled after acknowledgement");
      await page.keyboard.press("Escape");
      equal(state.removes.length, 1, "no extra remove from unknown dialog");
      equal(state.errors, [], "no browser errors (detail)");
      await context.close();
    }

    // Journey 5: Workspaces tab.
    {
      const { context, page, state } = await open({ width, height: 1000 });
      await page.goto(`${origin}/pipelines/workspaces`);
      await page.locator(".pipeline-workspace-list").waitFor({ timeout: 15000 });
      const row = page.locator(".pipeline-workspace-row", { hasText: "Deleted release run" });
      equal(await row.count(), 1, "deleted-run row listed");
      ok(await row.getByText("Old Project", { exact: true }).isVisible(), "project name on orphaned row");
      equal(await page.locator(".pipeline-workspace-row").count(), 4, "all rows listed");
      await shot(page, `5-workspaces-${width}`);
      equal(state.errors, [], "no browser errors (workspaces)");
      await context.close();
    }
  }

  // Phone surface at 390px.
  {
    const { context, page, state } = await open({ width: 390, height: 844 }, true);
    await page.goto(`${origin}/remote.html`);
    await page.evaluate(() => { history.pushState(null, "", "/project/Chuck"); window.dispatchEvent(new PopStateEvent("popstate")); });
    await page.getByRole("button", { name: "Start pipeline" }).first().click();
    const form = page.getByRole("form", { name: "Start pipeline" });
    await form.waitFor({ timeout: 15000 });
    const select = form.getByLabel("Workspace");
    ok(await select.isVisible(), "phone workspace choice");
    equal(await select.locator("option").allTextContents(), ["Project checkout", "New worktree"], "phone workspace options");
    await shot(page, "6-phone-start-project");
    await select.selectOption("new_worktree");
    ok(await form.getByLabel("Branch").isVisible(), "phone branch field");
    await shot(page, "6-phone-start-worktree");
    await page.evaluate(() => { history.pushState(null, "", "/run/run_dirty"); window.dispatchEvent(new PopStateEvent("popstate")); });
    const card = page.getByLabel("Workspace", { exact: true }).and(page.locator("section"));
    await card.waitFor({ timeout: 15000 });
    ok(await card.getByText("New worktree", { exact: false }).first().isVisible(), "phone workspace status");
    ok(await page.getByText("Remove this checkout from Pipelines on the desktop.").isVisible(), "desktop removal guidance");
    equal(await page.getByRole("button", { name: /remove/i }).count(), 0, "phone has no remove button");
    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
    ok(overflow <= 1, `phone overflow ${overflow}`);
    await shot(page, "7-phone-run");
    equal(state.errors, [], "no browser errors (phone)");
    await context.close();
  }
  console.log(`${checks} browser checks passed. Screenshots: ${outDir}`);
} finally { await browser.close(); await server.close(); }
