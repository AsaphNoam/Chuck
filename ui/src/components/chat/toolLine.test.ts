import { describe, expect, it } from "vitest";
import type { TranscriptEvent } from "../../api/types";
import { currentToolLabel, toolLine, toolLineSuffix } from "./toolLine";

// FS-03.A84: the label oracle. Calls and results are copied (trimmed) from
// retained Claude and Codex transcripts; the expected labels are FS-03.R108's.
const call = (data: Record<string, unknown>) => ({ kind: "tool_call", tool_call_id: "c1", ...data }) as TranscriptEvent;
const result = (data: Record<string, unknown>) => ({ kind: "tool_result", tool_call_id: "c1", ...data }) as TranscriptEvent;
const text = (value: string) => [{ type: "content", content: { type: "text", text: value } }];

describe("toolLine label", () => {
  it.each([
    ["a Claude description", { name: "Bash", title: "find ui/src/components/chat -type f | sort", args: { command: "find ui/src/components/chat -type f | sort", description: "List chat component files" } }, "List chat component files"],
    ["a Codex compound command", { name: "execute", title: "git status --short && git diff --stat", args: { command: "git status --short && git diff --stat" } }, "Ran git"],
    ["a leading directory change", { name: "exec_command", args: { command: "cd ui && npm test", cwd: "/repo" } }, "Ran npm"],
    ["assignments and wrappers", { name: "execute", args: { command: "FOO=1 /usr/bin/env python3 x.py" } }, "Ran python3"],
    ["env with an unset option", { name: "execute", args: { command: "env -u CHUCK_RUNTIME_ROOT go test ./internal/server/ 2>&1 | tail -15" } }, "Ran go"],
    ["a quoted program", { name: "execute", args: { command: "\"sed -n '1,105p' docs/features/INVARIANTS.md" } }, "Ran sed"],
    ["an empty command", { name: "execute", args: { command: "" } }, "Ran a command"],
    ["an unparseable program", { name: "execute", args: { command: "$(make-cmd) --flag" } }, "Ran a command"],
    ["a shell call with no command or title", { name: "Bash", args: null }, "Ran a command"],
    ["a quoted file path title", { name: "read", title: "Read file '/Users/me/Projects/AgentDeck/docs/SKILL.md'", args: null }, "Read file SKILL.md"],
    ["a Codex exec_command file read", { name: "exec_command", title: "Read file '/Users/me/.chuck/codex/skills/figma/SKILL.md'", args: null }, "Read file SKILL.md"],
    ["a relative path with a range", { name: "read", title: "Read internal/server/annotations_test.go (1 - 290)", args: { file_path: "/repo/internal/server/annotations_test.go" } }, "Read annotations_test.go (1 - 290)"],
    ["a pending Claude terminal", { name: "execute", title: "Terminal", args: {} }, "Terminal"],
    ["only a tool name", { name: "mcp__chuck-messaging__read_think_tank" }, "mcp__chuck-messaging__read_think_tank"],
    ["nothing at all", {}, "Used a tool"],
  ])("labels %s", (_case, data, label) => {
    expect(toolLine(call(data)).label).toBe(label);
  });

  it("clips a long multi-line title to one short line", () => {
    const label = toolLine(call({ name: "other", title: `Start subagent\n${"spawn ".repeat(330)}` })).label;
    expect(label).not.toContain("\n");
    expect(label.length).toBeLessThanOrEqual(80);
    expect(label.endsWith("…")).toBe(true);
  });
});

describe("toolLine state and reason", () => {
  it("reads a fenced Claude failure as its first meaningful line", () => {
    const line = toolLine(call({ name: "Bash", args: { command: "npm test" } }), result({ status: "failed", content: text("```\nExit code 1\n\n RUN  v2.1.9 /repo/ui\n```") }));
    expect(line).toEqual({ label: "Ran npm", state: "failed", reason: "RUN v2.1.9 /repo/ui" });
    expect(toolLineSuffix(line)).toBe("failed: RUN v2.1.9 /repo/ui");
  });

  it("unwraps a Claude tool_use_error", () => {
    const line = toolLine(call({ name: "Edit", title: "Edit" }), result({ status: "failed", content: text("<tool_use_error>File has not been read yet. Read it first before writing to it.</tool_use_error>") }));
    expect(line.reason).toBe("File has not been read yet. Read it first before writing to it.");
  });

  it("prefers a meaningful error over empty content", () => {
    expect(toolLine(call({ name: "Edit" }), result({ status: "failed", content: "", error: "Permission denied" })).reason).toBe("Permission denied");
  });

  it("reads a Codex content-null failure as failed alone", () => {
    const line = toolLine(call({ name: "execute", args: { command: "make test" } }), result({ status: "failed", content: null }));
    expect(line).toEqual({ label: "Ran make", state: "failed" });
    expect(toolLineSuffix(line)).toBe("failed");
  });

  it("adds nothing once completed, and running before a result", () => {
    const done = toolLine(call({ name: "execute", args: { command: "ls" } }), result({ status: "completed", content: text("a\nb") }));
    expect(done).toEqual({ label: "Ran ls", state: "completed" });
    expect(toolLineSuffix(done)).toBe("");
    expect(toolLineSuffix(toolLine(call({ name: "execute", status: "in_progress", args: { command: "ls" } })))).toBe("running");
  });

  it("keeps the background labels", () => {
    const notice = result({ status: "completed", content: text("Command running in background with ID: b1.") });
    expect(toolLineSuffix(toolLine(call({ name: "Bash", background_state: "running" }), notice))).toBe("Continues in background");
    expect(toolLineSuffix(toolLine(call({ name: "Bash", background_state: "completed" }), notice))).toBe("Ran in background");
  });
});

describe("currentToolLabel", () => {
  it("names the latest in-progress call of the running turn only", () => {
    const events = [
      call({ tool_call_id: "a", name: "execute", args: { command: "git status" } }),
      result({ tool_call_id: "a", status: "completed", content: null }),
      call({ tool_call_id: "b", name: "read", title: "Read file '/repo/x.md'" }),
    ];
    expect(currentToolLabel(events)).toBe("Read file x.md");
    expect(currentToolLabel([...events, result({ tool_call_id: "b", status: "completed" })])).toBeNull();
    expect(currentToolLabel([...events, { kind: "turn_end" } as TranscriptEvent])).toBeNull();
  });
});
