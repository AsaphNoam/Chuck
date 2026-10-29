import type { TranscriptEvent } from "../../api/types";

// toolLine is the one summary every surface shows for a tool call: a short
// label, its state and, when it failed, one trimmed reason. Durable events are
// unchanged; this is presentation only (FS-03.R108, TS-08.R147).

export type ToolState = "running" | "background-running" | "background" | "completed" | "failed";
export type ToolLine = { label: string; state: ToolState; reason?: string };

const SHELL_TOOLS = new Set(["execute", "exec_command", "Bash"]);
const WRAPPERS = new Set(["sudo", "env", "time", "nohup"]);
const LABEL_LIMIT = 80;
const REASON_LIMIT = 120;

export function toolLine(call: TranscriptEvent, result?: TranscriptEvent): ToolLine {
  const line: ToolLine = { label: clip(labelOf(call), LABEL_LIMIT), state: stateOf(call, result) };
  if (line.state === "failed" && result) {
    const reason = failureReason(result);
    if (reason) line.reason = reason;
  }
  return line;
}

// The text after the label: nothing once a call completes, otherwise its state.
export function toolLineSuffix(line: ToolLine) {
  if (line.state === "running") return "running";
  if (line.state === "background-running") return "Continues in background";
  if (line.state === "background") return "Ran in background";
  if (line.state === "failed") return line.reason ? `failed: ${line.reason}` : "failed";
  return "";
}

// The latest in-progress call of the running turn names what the agent is
// doing now; a finished turn's calls never do.
export function currentToolLabel(events: TranscriptEvent[]): string | null {
  const results = new Map<string, TranscriptEvent>();
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i];
    const kind = kindOf(event);
    if (kind === "turn_end") return null;
    if (kind === "tool_result") results.set(String(event.tool_call_id ?? ""), event);
    if (kind !== "tool_call") continue;
    const line = toolLine(event, results.get(String(event.tool_call_id ?? "")));
    if (line.state === "running") return line.label;
  }
  return null;
}

// The tool result each call folds in, keyed by tool_call_id, including those
// inside nested child activity.
export function resultsByCall(events: TranscriptEvent[], results = new Map<string, TranscriptEvent>()) {
  for (const event of events) {
    if (kindOf(event) === "tool_result" && event.tool_call_id != null) results.set(String(event.tool_call_id), event);
    const items = (event.node as { items?: TranscriptEvent[] } | undefined)?.items;
    if (kindOf(event) === "activity" && Array.isArray(items)) resultsByCall(items, results);
  }
  return results;
}

function labelOf(call: TranscriptEvent) {
  const args = (call.args != null && typeof call.args === "object" ? call.args : {}) as Record<string, unknown>;
  if (typeof args.description === "string" && args.description.trim()) return args.description;
  const name = String(call.name ?? call.tool ?? "");
  const title = String(call.title ?? "").trim();
  // Codex titles a file read made through exec_command by the file, so a shell
  // call without a command string keeps its title.
  if (typeof args.command === "string" || (SHELL_TOOLS.has(name) && !title)) return commandLabel(String(args.command ?? ""));
  if (title) return shortenPaths(title);
  return name || "Used a tool";
}

// A small tokenizer, not a shell parser: the first segment that is not a
// directory change, past assignments and wrappers, names the program.
function commandLabel(command: string) {
  for (const segment of command.split(/&&|\|\||;|\||\n/)) {
    const words = segment.trim().split(/\s+/).filter(Boolean);
    if (words.length === 0 || words[0] === "cd") continue;
    let index = 0;
    while (index < words.length) {
      const word = words[index];
      if (/^[A-Za-z_][A-Za-z0-9_]*=/.test(word)) index++;
      else if (WRAPPERS.has(word.split("/").pop() ?? "")) {
        index++;
        while (index < words.length && words[index].startsWith("-")) index += words[index] === "-u" ? 2 : 1;
      } else break;
    }
    const word = (words[index] ?? "").replace(/^["']+|["']+$/g, "");
    const program = word.split("/").pop() ?? "";
    return /^[\w.+-]+$/.test(program) ? `Ran ${program}` : "Ran a command";
  }
  return "Ran a command";
}

function shortenPaths(title: string) {
  return title.replace(/(["']?)([^\s"']*\/[^\s"']*[^\s"'/])\1/g, (_match, _quote, path: string) => path.split("/").pop() ?? path);
}

function stateOf(call: TranscriptEvent, result?: TranscriptEvent): ToolState {
  if (result && isFailed(result)) return "failed";
  if (call.status === "failed") return "failed";
  if (call.background_state === "running") return "background-running";
  if (call.background_state) return "background";
  if (result || call.status === "completed") return "completed";
  return "running";
}

export function isFailed(result: TranscriptEvent) {
  return result.status === "failed" || result.error != null || result.is_error === true;
}

// The first meaningful line of a failed result: fences, error-wrapper tags,
// blank lines and bare exit codes are skipped (FS-03.R108).
function failureReason(result: TranscriptEvent) {
  const text = resultText(result.content, result.error, result.result);
  for (const raw of text.split("\n")) {
    const line = raw.replace(/<\/?[A-Za-z_][\w-]*>/g, "").trim();
    if (!line || line.startsWith("```") || /^Exit code \d+$/i.test(line)) continue;
    return clip(line, REASON_LIMIT);
  }
  return "";
}

function resultText(...values: unknown[]) {
  for (const value of values) {
    const text = displayable(value);
    if (text.trim()) return text;
  }
  return "";
}

function displayable(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  const text = contentText(value);
  if (text != null) return text;
  return JSON.stringify(value) ?? "";
}

// ACP stores a result as its content blocks; wrapped text reads as that text.
function contentText(value: unknown): string | null {
  if (!Array.isArray(value)) return null;
  const texts: string[] = [];
  for (const block of value) {
    const inner = block as { type?: unknown; content?: { type?: unknown; text?: unknown } } | null;
    if (inner?.type !== "content" || inner.content?.type !== "text" || typeof inner.content.text !== "string") return null;
    texts.push(inner.content.text);
  }
  return texts.join("\n");
}

function clip(text: string, limit: number) {
  const line = text.replace(/\s+/g, " ").trim();
  return line.length > limit ? `${line.slice(0, limit - 1)}…` : line;
}

function kindOf(event: TranscriptEvent) {
  return String(event.kind ?? event.type ?? "");
}
