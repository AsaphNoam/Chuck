// Real-browser presentation and navigation evidence with isolated API fixtures.
// node scripts/pipelines-render.mjs [outDir] [populated|empty|loading|error]
// FS-14 routed run/template pages are the authority for navigation and states;
// this does not establish backend lifecycle or provider delivery correctness.
import { mkdirSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import assert from "node:assert/strict";
import { createServer } from "vite";
import { chromium } from "playwright";

const [outDir = join(tmpdir(), "pipelines-render"), state = "populated"] = process.argv.slice(2);
mkdirSync(outDir, { recursive: true });
const at = "2026-10-29T09:00:00Z";
const template = { version: 2, title: "Feature delivery", orchestrator_role: "implementer", inputs: [], stages: ["Design", "Implement", "Review", "Release"].map((title, i) => ({ id: `stage_${i}`, title, objective: `Complete ${title.toLowerCase()} and report the outcome.`, coordination: "standing", inputs: [], outputs: [] })) };
const records = [
  { id: "delivery", template, valid: true, diagnostics: [] },
  { id: "long_template_identifier_for_cross_region_migration", template: { ...template, title: "Cross-region migration with an exceptionally long reusable pipeline title" }, valid: true, diagnostics: [] },
  { id: "draft", template: { ...template, title: "Security review draft" }, valid: false, diagnostics: [{ field: "orchestrator_role", code: "missing_role", message: "Choose an available role." }] },
];
const runs = Array.from({ length: 51 }, (_, index) => ({
  run_id: `run_${index}`, template_id: "delivery", display_name: index === 0 ? "Ship workspace navigation" : index === 1 ? "Cross-region database migration with an exceptionally long pipeline run name" : `Delivery ${index + 1}`,
  project: index === 1 ? "a-very-long-project-name-for-cross-region-migration" : "Chuck", state: ["running", "paused", "completed", "stopped"][index % 4], revision: 1, pending_action: "", current_stage_id: "stage_1", current_stage_title: index === 1 ? "Implementation and validation of cross-region database migration" : "Implement", current_agent_id: "a_owner", attention_reason: index === 1 ? "approval_required" : "", final_outcome: index % 4 === 2 ? "success" : "", updated_at: new Date(Date.parse(at) - index * 60000).toISOString(), diagnostics: [],
}));
const detail = (summary) => ({ run: { ...summary, template_snapshot: template, goal: "Deliver the requested workspace navigation with a clear review record.", inputs: {}, current_task_id: "task_1", current_attempt_id: "", created_at: at }, template, inputs: {}, stage_tasks: [{ task_id: "task_1", run_id: summary.run_id, stage_id: "stage_1", stage_index: 1, attempt_number: 1, state: "running", assignment_text: "Implement the approved navigation.", standing_owner: { agent_id: "a_owner", name: "Implementation owner", state: "busy", route: "live" }, work: [], created_at: at, updated_at: at }], values: [], diagnostics: [], controls: Object.fromEntries(["continue", "retry", "replace", "stop", "repair_cleanup"].map((key) => [key, { eligible: key === "stop", reason: "" }])) });
const server = await createServer({ server: { host: "127.0.0.1", port: 5197, strictPort: true }, logLevel: "error" });
await server.listen();
const browser = await chromium.launch();
let checks = 0;
try {
  for (const skin of ["", "sky-grove", "studio"]) for (const width of [1024, 1440]) {
    const context = await browser.newContext({ viewport: { width, height: 1000 } });
    await context.addInitScript(() => {
      delete window.SharedWorker;
      window.__pipelineStreams = [];
      window.EventSource = class extends EventTarget {
        constructor() { super(); window.__pipelineStreams.push(this); setTimeout(() => { this.onopen?.(); this.dispatchEvent(new MessageEvent("state_update", { data: JSON.stringify({ agent_id: "__hydrated__", data: { hydrated: true } }) })); }); }
        close() {}
      };
    });
    let refreshed = false;
    const errors = [];
    await context.route((url) => url.pathname.startsWith("/api/"), async (route) => {
      const url = new URL(route.request().url());
      const path = url.pathname;
      if (path === "/api/pipeline-runs" || path === "/api/pipelines") {
        if (state === "loading") await new Promise((resolve) => setTimeout(resolve, 20000).unref());
        if (state === "error") return route.fulfill({ status: 500, json: { error: { message: "Pipeline storage unavailable" } } });
        if (path === "/api/pipelines") return route.fulfill({ json: state === "empty" ? [] : records });
        const offset = Number(url.searchParams.get("offset") || 0);
        const visible = state === "empty" ? [] : runs.map((run, i) => refreshed && i === 0 ? { ...run, display_name: "Live refresh received", revision: 2 } : run);
        return route.fulfill({ json: visible.slice(offset, offset + 50), headers: { "X-Total-Count": String(visible.length) } });
      }
      if (path.startsWith("/api/pipeline-runs/")) return route.fulfill({ json: detail(runs.find((run) => run.run_id === path.split("/").at(-1)) || runs[0]) });
      if (path === "/api/pipeline-proposals") return route.fulfill({ json: { pending: [], declined: [] } });
      if (path === "/api/projects") return route.fulfill({ json: { Chuck: { title: "Chuck", cwd: "/tmp/chuck", color: [80, 120, 200] } } });
      if (path === "/api/config") return route.fulfill({ json: { appearance_skin: skin, onboarded: true } });
      if (path === "/api/roles") return route.fulfill({ json: { implementer: { title: "Implementer", system_prompt: "", skip_permissions: false } } });
      if (path === "/api/layout") return route.fulfill({ json: { order: [], density: { perRow: 3, gap: 16 } } });
      if (path === "/api/backends") return route.fulfill({ json: { version: 2, backends: { codex: { name: "Codex", type: "codex-acp", default: true, default_model: "gpt-6.1-sol", models: { "gpt-6.1-sol": { name: "GPT-6.1 Sol", model: "gpt-6.1-sol", fast: true } } } } } });
      if (path === "/api/config-sources") return route.fulfill({ json: [] });
      return route.fulfill({ json: {} });
    });
    const page = await context.newPage();
    page.on("pageerror", (error) => errors.push(error.message));
    const capture = async (name) => {
      await page.waitForTimeout(350);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - innerWidth);
      assert.ok(overflow <= 1, `${name} ${skin || "core"} ${width}: page overflow ${overflow}`);
      assert.deepEqual(errors, [], "Browser errors");
      checks += 2;
      const path = join(outDir, `${name}-${state}-${skin || "core"}-${width}.png`);
      await page.screenshot({ path, fullPage: true });
      await page.screenshot({ path: path.replace(/\.png$/, "-viewport.png"), fullPage: false });
      console.log(path);
    };
    await page.goto("http://127.0.0.1:5197/pipelines/runs");
    const ready = state === "populated" ? page.locator('[data-ui="pipeline-run-list"]') : state === "empty" ? page.getByText("No runs yet", { exact: true }) : state === "loading" ? page.getByLabel("Loading runs") : page.getByText("Pipeline storage unavailable", { exact: true }).first();
    await ready.waitFor({ timeout: 15000 });
    await capture("runs");
    if (state === "populated") {
      await page.getByRole("button", { name: "Start run", exact: true }).click();
      const dialog = page.getByRole("dialog");
      await dialog.waitFor();
      await capture("start-open");
      await dialog.locator('[data-field="template_id"] select').selectOption("delivery");
      await dialog.locator('[data-field="project"] select').selectOption("Chuck");
      await dialog.locator('[data-field="goal"] textarea').fill("Deliver the workspace navigation with a complete review record.");
      await capture("start-setup");
      await dialog.getByRole("button", { name: "Review", exact: true }).click();
      await dialog.getByRole("button", { name: "Back", exact: true }).waitFor(); checks++;
      await capture("start-review");
      await page.keyboard.press("Escape");
      await page.getByRole("button", { name: "More runs", exact: true }).click();
      await page.getByText("51 of 51 retained runs", { exact: true }).waitFor(); checks++;
      refreshed = true;
      await page.evaluate(() => window.__pipelineStreams.forEach((stream) => stream.dispatchEvent(new MessageEvent("pipeline_update", { data: JSON.stringify({ type: "pipeline_update", seq: 2, ts: 2, agent_id: null, data: { run_id: "run_0", display_name: "Live refresh received", revision: 2, state: "running", current_stage_id: "stage_1", current_agent_id: "a_owner", attention_reason: "", final_outcome: "" } }) }))));
      await page.getByText("Live refresh received", { exact: true }).waitFor(); checks++;
      await page.locator('a[href="/pipelines/runs/run_0"]').click();
      await page.locator('[data-ui="pipeline-run"]').waitFor(); checks++;
      await capture("detail");
    }
    await page.goto("http://127.0.0.1:5197/pipelines/templates");
    const templatesReady = state === "populated" ? page.locator('[data-ui="pipeline-template-library"]') : state === "empty" ? page.getByText("No templates yet", { exact: true }) : state === "loading" ? page.getByLabel("Loading templates") : page.getByText("Pipeline storage unavailable", { exact: true }).first();
    await templatesReady.waitFor({ timeout: 15000 });
    await capture("templates");
    if (state === "populated") {
      await page.locator('a[href="/pipelines/templates/delivery"]').click();
      await page.getByRole("heading", { name: "Feature delivery", exact: true }).waitFor(); checks++;
      const navOverflow = await page.locator(".pipeline-stage-navigator").evaluate((element) => element.scrollWidth - element.clientWidth);
      assert.ok(navOverflow <= 1, `Editor stage navigation overflows ${navOverflow}`); checks++;
      await capture("editor");
    }
    await context.close();
  }
  console.log(`${checks} browser checks passed (${state}).`);
} finally { await browser.close(); await server.close(); }
