#!/usr/bin/env node
// Contract test for the reviewed Codex ACP `_chuck/quota/read` patch (TS-04.R109).
// The adapter entrypoint starts a server on import, so this executes the patched
// source's own initialize `_meta` literal and quota branch against a stubbed
// app-server client; it does not start Codex.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const runtime = path.resolve(process.argv[2] ?? path.dirname(fileURLToPath(import.meta.url)));
const source = await readFile(path.join(runtime, "node_modules/@agentclientprotocol/codex-acp/dist/index.js"), "utf8");
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
const between = (from, to, label) => {
    const start = source.indexOf(from);
    assert.ok(start > 0, `${label} start exists`);
    const end = source.indexOf(to, start);
    assert.ok(end > start, `${label} end exists`);
    return source.slice(start, end);
};

assert.ok(source.includes('.onRequest(CHUCK_QUOTA_READ_METHOD, chuckQuotaReadParamsParser, (ctx) => getAgent().extMethod(CHUCK_QUOTA_READ_METHOD, ctx.params))'),
    "quota request is registered with a params parser");
assert.ok(source.includes('request.method === CHUCK_QUOTA_READ_METHOD ||'), "extMethod dispatch accepts the quota request");
assert.ok(source.includes('var CHUCK_QUOTA_READ_METHOD = "_chuck/quota/read";'), "request method name");

const metaSource = between("steering: {\n          supported: true\n        },\n        chuck:", "// Only AIR gets", "initialize _meta");
const meta = new Function(`return ({ ${metaSource} });`)();
assert.deepEqual(meta.chuck, { quotaRead: { version: 1 } }, "initialize advertises quotaRead v1");
assert.deepEqual(meta.steering, { supported: true }, "existing _meta keys are kept");

const branch = between("case CHUCK_QUOTA_READ_METHOD: {", 'case "authentication/status":', "quota branch");
class RequestError extends Error { static invalidParams(_data, message) { return new RequestError(message); } }
const run = new AsyncFunction("CHUCK_QUOTA_READ_METHOD", "methodRequest", "RequestError", `switch (methodRequest.method) { ${branch} }`);
const win = (usedPercent, windowDurationMins) => ({ usedPercent, windowDurationMins, resetsAt: 1800000000, extra: "kept" });
const response = {
    rateLimits: { limitId: "codex", limitName: "Codex", primary: win(10, 300), secondary: win(20, 10080), credits: { balance: "5" }, planType: "pro" },
    rateLimitsByLimitId: {
        codex: { limitId: "codex", primary: win(99, 300), secondary: null },
        gpt_x: { limitId: "gpt_x", primary: null, secondary: win(30, 10080), credits: {}, planType: "pro" },
        gone: undefined,
    },
};
const self = (known = ["s1"]) => ({
    sessions: new Map(known.map((id) => [id, {}])),
    runWithProcessCheck: (operation) => operation(),
    codexAcpClient: { getRateLimits: async () => response },
});
const call = (known, sessionId) => run.call(self(known), "_chuck/quota/read", { method: "_chuck/quota/read", params: { sessionId } }, RequestError);

const result = await call(["s1"], "s1");
assert.deepEqual(Object.keys(result), ["snapshots"], "only snapshots leave the adapter");
assert.deepEqual(result.snapshots, [
    { limitId: "codex", primary: response.rateLimits.primary, secondary: response.rateLimits.secondary },
    { limitId: "gpt_x", primary: null, secondary: response.rateLimitsByLimitId.gpt_x.secondary },
], "main snapshot first, deduplicated by limitId, windows passed through raw");
for (const snapshot of result.snapshots) {
    assert.deepEqual(Object.keys(snapshot), ["limitId", "primary", "secondary"], "no credits, plan or name fields");
}
await assert.rejects(call(["s1"], "other"), RequestError, "unknown session is an error");
console.log("Codex quota-read patch contract checks passed (stubbed app-server client).");
