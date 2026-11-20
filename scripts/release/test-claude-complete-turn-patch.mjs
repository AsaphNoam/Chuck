#!/usr/bin/env node
// Contract test for the reviewed Claude ACP complete-turn fork patch (TS-04.R113).
// Runs the patched adapter's initialize and session/fork resolution against the
// pinned Claude Agent SDK's real fork over fixture transcripts in a private
// CLAUDE_CONFIG_DIR; it does not start Claude Code or call a provider.
import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, realpath, rm, writeFile } from "node:fs/promises";
import os from "node:os";
import path from "node:path";
import { randomUUID } from "node:crypto";
import { fileURLToPath, pathToFileURL } from "node:url";

const runtime = path.resolve(process.argv[2] ?? path.dirname(fileURLToPath(import.meta.url)));
const dist = path.join(runtime, "node_modules/@agentclientprotocol/claude-agent-acp/dist");
const scratch = await realpath(await mkdtemp(path.join(os.tmpdir(), "chuck-complete-turn-")));
// The SDK reads its config root lazily; set it before the adapter loads the SDK.
process.env.CLAUDE_CONFIG_DIR = path.join(scratch, "config");
const cwd = path.join(scratch, "project");
const projectDir = path.join(process.env.CLAUDE_CONFIG_DIR, "projects", cwd.replace(/[^a-zA-Z0-9]/g, "-"));
await mkdir(cwd, { recursive: true });
await mkdir(projectDir, { recursive: true });

try {
    const { ClaudeAcpAgent, messageIdForGrouping } = await import(pathToFileURL(path.join(dist, "acp-agent.js")));
    const { forkSession } = await import(pathToFileURL(path.join(dist, "fork-session.js")));

    const agent = Object.create(ClaudeAcpAgent.prototype);
    agent.probeCliAuthStatus = async () => {};
    agent.logger = { log() {}, error() {} };
    const init = await agent.initialize({ protocolVersion: 1, clientCapabilities: {} });
    assert.equal(init._meta.chuck.completeTurnForkVersion, 1, "initialize advertises completeTurnForkVersion 1");
    assert.deepEqual(init._meta.chuck.quotaRead, { version: 1 }, "existing chuck keys are kept");
    assert.deepEqual(init._meta.steering, { supported: true }, "existing _meta keys are kept");

    // Builds a transcript; each entry's parent is the previous one unless given.
    const transcript = () => {
        const sessionId = randomUUID();
        const entries = [];
        const add = (entry) => {
            const full = { parentUuid: entries.at(-1)?.uuid ?? null, isSidechain: false, userType: "external", cwd, sessionId,
                version: "2.1.286", uuid: randomUUID(), timestamp: new Date(Date.UTC(2026, 0, 1, 0, 0, entries.length)).toISOString(), ...entry };
            entries.push(full);
            return full;
        };
        const save = () => writeFile(path.join(projectDir, `${sessionId}.jsonl`), entries.map((entry) => JSON.stringify(entry)).join("\n") + "\n");
        return { sessionId, entries, add, save };
    };
    const prompt = (text) => ({ type: "user", message: { role: "user", content: text } });
    const assistant = (id, content) => ({ type: "assistant", message: { id, role: "assistant", model: "claude-test", content } });
    const toolResult = (id) => ({ type: "user", message: { role: "user", content: [{ type: "tool_result", tool_use_id: id, content: "ok" }] } });
    const forkPoint = (messageId) => ({ jetbrains: { air: { fork: { version: 1, messageId, messageFingerprint: `sha256:${"0".repeat(64)}` } } } });
    const deps = (live) => ({ logger: { log() {}, error() {} }, messageIdForGrouping, ...(live && { liveMessageIdToUuid: live }) });
    const forkedFrom = async (sessionId) => (await readFile(path.join(projectDir, `${sessionId}.jsonl`), "utf8"))
        .trim().split("\n").map((line) => JSON.parse(line)).filter((entry) => entry.forkedFrom).map((entry) => entry.forkedFrom.messageUuid);
    const proof = (messageId) => ({ chuck: { completeTurn: { version: 1, sourceMessageId: messageId, complete: true } } });

    // End-turn tool session: the final assistant message is followed by the tool_result
    // carrier and a structured_output attachment that belong to the kept turn.
    const endTurn = transcript();
    endTurn.add(prompt("first"));
    endTurn.add(assistant("msg_1", [{ type: "text", text: "answer" }]));
    const call = endTurn.add(assistant("msg_1", [{ type: "tool_use", id: "tu_1", name: "StructuredOutput", input: {} }]));
    endTurn.add(toolResult("tu_1"));
    endTurn.add({ type: "attachment", attachment: { type: "structured_output", data: { ok: true } } });
    const kept = endTurn.add({ type: "system", subtype: "turn_duration", content: "" });
    endTurn.add(prompt("second"));
    endTurn.add(assistant("msg_2", [{ type: "text", text: "later" }]));
    await endTurn.save();
    const keptIds = endTurn.entries.slice(0, endTurn.entries.indexOf(kept) + 1).map((entry) => entry.uuid);
    for (const live of [undefined, new Map([["msg_1", call.uuid]])]) {
        const result = await forkSession({ sessionId: endTurn.sessionId, cwd, _meta: forkPoint("msg_1") }, deps(live));
        assert.deepEqual(result._meta, proof("msg_1"), `proof echoes the requested messageId (${live ? "live" : "history"} resolution)`);
        assert.deepEqual(await forkedFrom(result.sessionId), keptIds, "cut keeps the tool_result and structured_output, drops the next turn");
    }

    // Absorbed queued content inside the turn: attribution is ambiguous, so the
    // fork keeps R110's cut at the resolved message and carries no proof.
    const queued = transcript();
    queued.add(prompt("first"));
    const answer = queued.add(assistant("msg_1", [{ type: "tool_use", id: "tu_1", name: "StructuredOutput", input: {} }]));
    queued.add({ type: "attachment", attachment: { type: "queued_command", prompt: "sneaked in" } });
    queued.add(toolResult("tu_1"));
    queued.add(prompt("second"));
    await queued.save();
    let result = await forkSession({ sessionId: queued.sessionId, cwd, _meta: forkPoint("msg_1") }, deps());
    assert.equal(result._meta, undefined, "ambiguous turn returns no proof");
    assert.deepEqual(await forkedFrom(result.sessionId), queued.entries.slice(0, 2).map((entry) => entry.uuid), "ambiguous turn keeps the R110 cut");

    // A later entry on another branch cannot be attributed to the message's turn.
    const branched = transcript();
    const first = branched.add(prompt("first"));
    branched.add(assistant("msg_1", [{ type: "text", text: "answer" }]));
    branched.add({ ...toolResult("tu_x"), parentUuid: first.uuid });
    await branched.save();
    result = await forkSession({ sessionId: branched.sessionId, cwd, _meta: forkPoint("msg_1") }, deps());
    assert.equal(result._meta, undefined, "entry outside the message's descendants returns no proof");

    // A fork point naming a user entry is not an assistant turn end: no proof.
    result = await forkSession({ sessionId: endTurn.sessionId, cwd, _meta: forkPoint(endTurn.entries[0].uuid) }, deps());
    assert.equal(result._meta, undefined, "non-assistant fork point returns no proof");
    assert.deepEqual(await forkedFrom(result.sessionId), [endTurn.entries[0].uuid], "non-assistant fork point keeps the R110 cut");
    await assert.rejects(forkSession({ sessionId: endTurn.sessionId, cwd, _meta: forkPoint("msg_missing") }, deps()),
        /was not found/, "unresolvable message still fails as before");

    // Plain fork without a fork point is unchanged: a full copy and no _meta.
    result = await forkSession({ sessionId: endTurn.sessionId, cwd }, deps());
    assert.deepEqual(Object.keys(result), ["sessionId"], "plain fork returns only sessionId");
    assert.deepEqual(await forkedFrom(result.sessionId), endTurn.entries.map((entry) => entry.uuid), "plain fork copies the whole session");
    console.log("Claude complete-turn fork patch contract checks passed (pinned SDK fork, fixture transcripts).");
} finally {
    await rm(scratch, { recursive: true, force: true });
}
