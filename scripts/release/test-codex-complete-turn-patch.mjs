#!/usr/bin/env node
// Contract test for the reviewed Codex ACP complete-turn fork patch (TS-04.R113).
// The adapter entrypoint starts a server on import, so this executes the patched
// source's own initialize `_meta` literal, SessionFork module and session/fork
// response glue against a stubbed app-server client; it does not start Codex.
import assert from "node:assert/strict";
import { createHash } from "node:crypto";
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

const metaSource = between("steering: {\n          supported: true\n        },\n        chuck:", "// Only AIR gets", "initialize _meta");
const meta = new Function(`return ({ ${metaSource} });`)();
assert.deepEqual(meta.chuck, { quotaRead: { version: 1 }, completeTurnForkVersion: 1 }, "initialize advertises completeTurnForkVersion 1 beside quotaRead");
assert.deepEqual(meta.steering, { supported: true }, "existing _meta keys are kept");
assert.ok(source.includes("return [sessionId, sessionModelState, sessionModeState, sessionMetadata.chuckCompleteTurn];"),
    "session creation hands the fork proof to the response");

class RequestError extends Error { static invalidParams(_data, message) { return new RequestError(message); } }
const forkModule = between("// src/SessionFork.ts\n", "// src/CodexThreadErrors.ts", "SessionFork module")
    .replace('import { createHash } from "node:crypto";', "");
const { forkSession } = new Function("RequestError", "createHash", `${forkModule}\nreturn { forkSession };`)(RequestError, createHash);

const fingerprint = (text) => `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
// Turn t1's final agent message is followed by kept tool items; the fork keeps whole turns.
const turn = (id, status, items) => ({ id, status, items });
const t1 = turn("t1", "completed", [
    { type: "userMessage", id: "u1", content: [] },
    { type: "agentMessage", id: "a1", text: "answer" },
    { type: "commandExecution", id: "c1", status: "completed" },
    { type: "mcpToolCall", id: "m1", status: "completed", result: { structuredContent: { ok: true } } },
]);
const t2 = turn("t2", "completed", [{ type: "userMessage", id: "u2", content: [] }, { type: "agentMessage", id: "a2", text: "later" }]);
const run = async (turns, meta3) => {
    const forks = [];
    const codexClient = {
        threadReadWithHistory: async () => ({ thread: { turns } }),
        threadFork: async (params) => { forks.push(params); return { thread: { id: "fork-1" }, model: "m", reasoningEffort: null, modelProvider: "openai" }; },
        threadUnsubscribe: async () => ({}),
    };
    const result = await forkSession({ sessionId: "src", cwd: "/w", ...(meta3 && { _meta: meta3 }) }, [], {
        codexClient,
        refreshSkills: async () => {},
        createSessionConfig: async () => ({}),
        getResumeModelProvider: async () => "openai",
        fetchAvailableModels: async () => [],
        createCurrentModelId: () => "m",
        getCollaborationMode: () => null,
    });
    assert.equal(forks.length, 1, "one native fork");
    return { result, fork: forks[0] };
};
const point = (messageId, text = "answer") => ({ jetbrains: { air: { fork: { version: 1, messageId, messageFingerprint: fingerprint(text) } } } });
const proof = { version: 1, sourceMessageId: "a1", complete: true };

let { result, fork } = await run([t1, t2], point("a1"));
assert.equal(fork.lastTurnId, "t1", "fork keeps every turn through the containing turn");
assert.deepEqual(result.chuckCompleteTurn, proof, "complete turn yields the proof with the exact requested messageId");

({ result, fork } = await run([turn("t1", "interrupted", t1.items), t2], point("a1")));
assert.equal(fork.lastTurnId, "t1");
assert.equal(result.chuckCompleteTurn, undefined, "an incomplete turn returns no proof");

({ result, fork } = await run([t1, turn("t2", "completed", [...t2.items, { type: "agentMessage", id: "a1", text: "dup" }])], point("a1")));
assert.equal(result.chuckCompleteTurn, undefined, "an item id held by two turns returns no proof");

({ result, fork } = await run([t1, t2], point("missing")));
assert.equal(fork.lastTurnId, "t1", "fingerprint fallback still forks as before");
assert.equal(result.chuckCompleteTurn, undefined, "a fingerprint-only resolution returns no proof");

({ result } = await run([t1, t2], point("u1")));
assert.equal(result.chuckCompleteTurn, undefined, "a non-agent-message fork point returns no proof");

({ result, fork } = await run([t1, t2]));
assert.equal("lastTurnId" in fork, false, "plain fork sends no lastTurnId");
assert.equal("chuckCompleteTurn" in result, false, "plain fork carries no proof");

// The agent's session/fork response adds `_meta.chuck.completeTurn` only with a proof.
const method = between("  async forkSession(params) {\n    if (this.providerUpdate", "\n  async listSessions(params)", "agent forkSession");
const body = method.slice(method.indexOf("{") + 1, method.lastIndexOf("}"));
const respond = new AsyncFunction("logger", "params", body);
const agent = (completeTurn) => ({
    providerUpdate: null,
    tryCreateSession: async () => ["fork-1", {}, { currentModeId: "default" }, completeTurn],
    createSessionConfigOptionsResponse: () => ({ configOptions: [] }),
    getSessionState: () => ({}),
    handleError: async () => {},
});
const logger = { log() {} };
assert.deepEqual(await respond.call(agent(proof), logger, { sessionId: "src" }),
    { sessionId: "fork-1", modes: { currentModeId: "default" }, configOptions: [], _meta: { chuck: { completeTurn: proof } } }, "fork response carries the proof");
assert.deepEqual(await respond.call(agent(undefined), logger, { sessionId: "src" }),
    { sessionId: "fork-1", modes: { currentModeId: "default" }, configOptions: [] }, "fork response without a proof is unchanged");
console.log("Codex complete-turn fork patch contract checks passed (stubbed app-server client).");
