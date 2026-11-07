#!/usr/bin/env node
// Focused lifecycle regression for the reviewed Claude ACP patch. This executes
// the installed adapter's autonomous-result branch, not a copied predicate.
// It does not replace the real CLI/fake API replay or verify SDK transport.
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

const runtime = path.resolve(process.argv[2] ?? path.dirname(fileURLToPath(import.meta.url)));
const source = await readFile(path.join(runtime, "node_modules/@agentclientprotocol/claude-agent-acp/dist/acp-agent.js"), "utf8");
const autonomous = source.indexOf("if (isAutonomousResult) {", source.indexOf('case "result": {'));
assert.ok(autonomous > 0, "reviewed autonomous result branch exists");
const start = source.lastIndexOf("if (session.cancelled) {", autonomous);
const end = source.indexOf("// A cancel can race startup", autonomous);
assert.ok(start > 0 && end > autonomous, "reviewed cancellation and branch boundaries exist");
const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
const dispatch = new AsyncFunction("context", `with (context) { do { ${source.slice(start, end)} } while (false); }`);
// Use this runtime's synthetic-banner classifier rather than recreating its grammar.
const helperRelative = "node_modules/@agentclientprotocol/claude-agent-acp/dist/session-failure-extension.js";
const helpers = await import(pathToFileURL(path.join(runtime, helperRelative)));
const captureStart = source.indexOf('if (message.type === "assistant" && message.parent_tool_use_id === null) {', source.indexOf("// Snapshot the latest top-level assistant usage"));
const captureEnd = source.indexOf('if (message.message.model !== "<synthetic>") {', captureStart);
assert.ok(captureStart > 0 && captureEnd > captureStart, "root capture boundary exists");
const capture = new AsyncFunction("context", `with (context) { ${source.slice(captureStart, captureEnd)} } }`);
const finalStart = source.indexOf("finally {", end) + "finally {".length;
const finalEnd = source.indexOf("if (!isAutonomousResult)", finalStart);
assert.ok(finalStart > end && finalEnd > finalStart, "result cleanup boundary exists");
const cleanup = new AsyncFunction("context", `with (context) { ${source.slice(finalStart, finalEnd)} }`);
const resetStart = source.indexOf("const resetTurnScratch = () => {") + "const resetTurnScratch = () => {".length;
const resetEnd = source.indexOf("lastRefusalExplanation = null;", resetStart) + "lastRefusalExplanation = null;".length;
assert.ok(resetStart > 0 && resetEnd > resetStart, "activation reset boundary exists");
const reset = new AsyncFunction("context", `with (context) { ${source.slice(resetStart, resetEnd)} }`);

// FS-01.R40/R55: only a root usage-limit refusal in a still-held notification
// followup fails its owning prompt. Cancellation, child errors and unrelated
// autonomous work must retain their existing outcomes.
async function run(overrides = {}) {
    const calls = [];
    const turn = { deferredSettle: { stopReason: "end_turn" } };
    const context = {
        session: { activeTurn: turn, cancelled: false, emittedAssistantText: true },
        message: { origin: { kind: "task-notification" }, is_error: true, num_turns: 1 },
        isAutonomousResult: true,
        pendingExitPlanModeInterruption: undefined,
        pendingExitPlanContextReset: undefined,
        cycleRootUsageLimit: { error: "rate_limit", title: "You've hit your session limit · resets 5pm" },
        isHeldOpen: (value) => !!value?.deferredSettle,
        isExitPlanInterruptionResult: () => false,
        clearFailuresFromEarlierTurns: async () => {},
        settleOrDefer: async () => { calls.push({ kind: "plan-settle" }); },
        settleDeferredIfDrained: async () => { calls.push({ kind: "deferred-success" }); },
        firstUnsettledQueuedTurn: () => undefined,
        failActiveWithSessionFailure: async (kind, error, title) => {
            calls.push({ kind, error, title });
        },
        internalErrorForClient: (data, message) => ({ data, message }),
        errorKindData: (errorKind) => ({ errorKind }),
        ...overrides,
    };
    await dispatch(context);
    return calls;
}

assert.deepEqual((await run()).map(({ kind }) => kind), ["quota_exhausted"],
    "held background followup quota refusal replaces deferred success");
assert.deepEqual((await run({ cycleRootUsageLimit: undefined })).map(({ kind }) => kind), ["deferred-success"],
    "child-only or ordinary error cannot supply root quota evidence");
assert.deepEqual((await run({ message: { origin: { kind: "peer-message" }, is_error: true, num_turns: 1 } })).map(({ kind }) => kind), ["deferred-success"],
    "unrelated autonomous origin cannot fail held user work");
assert.deepEqual((await run({ session: { activeTurn: null, cancelled: false } })).map(({ kind }) => kind), ["deferred-success"],
    "no held turn cannot attach quota to another prompt");
assert.deepEqual((await run({ session: { activeTurn: {}, cancelled: false } })).map(({ kind }) => kind), ["deferred-success"],
    "an active ordinary prompt cannot inherit an autonomous quota failure");
assert.deepEqual(await run({ session: { activeTurn: { deferredSettle: {} }, cancelled: true } }), [],
    "accepted cancellation wins over later quota evidence");
assert.deepEqual((await run({ message: { origin: { kind: "task-notification" }, is_error: false, num_turns: 1 } })).map(({ kind }) => kind), ["deferred-success"],
    "ordinary successful followup remains successful");
assert.deepEqual(await run({ message: { origin: { kind: "task-notification" }, is_error: true, num_turns: 0 } }), [],
    "placeholder result cannot settle the held turn");
const scratch = {
    cycleRootUsageLimit: undefined,
    ...helpers,
    message: { type: "assistant", parent_tool_use_id: "child-tool", error: "rate_limit",
        message: { model: "<synthetic>", content: "You've hit your session limit · resets 5pm" } },
    lastAssistantTotalUsage: null, lastAssistantUsage: null, lastAssistantModel: null,
    lastAssistantError: undefined, lastAssistantWasUsageLimit: false,
    lastAssistantFailureTitle: undefined, lastRefusalExplanation: null,
};
await capture(scratch);
assert.equal(scratch.cycleRootUsageLimit, undefined, "child refusal does not enter root candidate");
scratch.message.parent_tool_use_id = null;
await capture(scratch);
assert.equal(scratch.cycleRootUsageLimit?.error, "rate_limit", "native synthetic root refusal supplies quota evidence");
await cleanup(scratch);
assert.equal(scratch.cycleRootUsageLimit, undefined, "each result retires its quota candidate");
await capture(scratch);
await reset(scratch);
assert.equal(scratch.cycleRootUsageLimit, undefined, "next active turn cannot inherit quota evidence");
await capture(scratch);
scratch.message.message = { model: "claude-opus-5-5", content: "done" };
await capture(scratch);
assert.equal(scratch.cycleRootUsageLimit, undefined, "later root answer supersedes prior refusal");
scratch.message.message = { model: "<synthetic>", content: "API Error: 429 rate limited" };
await capture(scratch);
assert.equal(scratch.cycleRootUsageLimit, undefined, "ordinary synthetic throttling is not account quota");
console.log("Claude quota patch lifecycle checks passed (extracted adapter branch; transport replay remains separate).");
