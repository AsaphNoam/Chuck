#!/usr/bin/env node
// Contract test for the reviewed Claude ACP `_chuck/quota/read` patch (TS-04.R109).
// Runs the patched adapter's real initialize and quota method against a stubbed
// SDK query; it does not call the real SDK or provider.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const runtime = path.resolve(process.argv[2] ?? path.dirname(fileURLToPath(import.meta.url)));
const dist = path.join(runtime, "node_modules/@agentclientprotocol/claude-agent-acp/dist/acp-agent.js");
const source = await readFile(dist, "utf8");
assert.ok(source.includes('.onRequest(CHUCK_QUOTA_READ_METHOD, { parse: parseChuckQuotaReadRequest }, (ctx) => agent.chuckQuotaRead(ctx.params))'),
    "quota request is registered with a params parser");
assert.ok(source.includes('const CHUCK_QUOTA_READ_METHOD = "_chuck/quota/read";'), "request method name");
const { ClaudeAcpAgent } = await import(pathToFileURL(dist));

const agent = Object.create(ClaudeAcpAgent.prototype);
agent.probeCliAuthStatus = async () => {};
agent.logger = { log() {}, error() {} };
const init = await agent.initialize({ protocolVersion: 1, clientCapabilities: {} });
assert.deepEqual(init._meta.chuck, { quotaRead: { version: 1 } }, "initialize advertises quotaRead v1");
assert.deepEqual(init._meta.steering, { supported: true }, "existing _meta keys are kept");

// Shaped like sdk.d.ts rate_limits: window members, model_scoped as an array,
// and extra_usage credits that must not leave the adapter (TS-04.R109).
const windows = {
    five_hour: { utilization: 42, resets_at: "2026-01-01T00:00:00Z" },
    seven_day: null,
    seven_day_opus: { utilization: 100, resets_at: "2026-01-03T00:00:00Z" },
    model_scoped: [{ display_name: "Fable", utilization: 7, resets_at: null }],
};
const limits = { ...windows, extra_usage: { is_enabled: true, monthly_limit: 50, used_credits: 12, utilization: 24 }, future_member: 1 };
const calls = [];
const query = {
    usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET: async (opts) => {
        calls.push(opts);
        return { rate_limits_available: true, rate_limits: limits, cost: { total: 9 }, plan: "max", behaviors: {}, account: "x" };
    },
};
agent.sessions = { live: { query, queryClosed: false }, closed: { query, queryClosed: true } };
const result = await agent.chuckQuotaRead({ sessionId: "live" });
assert.deepEqual(Object.keys(result).sort(), ["rate_limits", "rate_limits_available"], "no extra fields leave the adapter");
assert.equal(result.rate_limits_available, true);
assert.deepEqual(result.rate_limits, windows, "only window members are forwarded, values unmodified");
assert.deepEqual(calls, [{ skipBehaviors: true }], "behaviors are skipped");

query.usage_EXPERIMENTAL_MAY_CHANGE_DO_NOT_RELY_ON_THIS_API_YET = async () => ({ rate_limits_available: false, rate_limits: null });
assert.deepEqual(await agent.chuckQuotaRead({ sessionId: "live" }), { rate_limits_available: false, rate_limits: null });
await assert.rejects(agent.chuckQuotaRead({ sessionId: "nope" }), "unknown session is an error");
await assert.rejects(agent.chuckQuotaRead({ sessionId: "closed" }), "closed session is an error");
console.log("Claude quota-read patch contract checks passed (stubbed SDK query).");
