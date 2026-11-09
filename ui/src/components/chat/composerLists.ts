import { caret, type ComposerSnapshot } from "./composerHistory";

type Item = { indent: number; number: number; width: number; body: string };
const MARKER = /^( *)(\d{1,9})\.( {1,4}(?![ \t])| +$)(.*)$/;
const FENCE = /^( *)(`{3,}|~{3,})(.*)$/;
const MAX_NUMBER = 999_999_999;

function itemOn(line: string): Item | undefined {
  const match = MARKER.exec(line);
  if (!match) return;
  const [, indent, number, space, body] = match;
  return { indent: indent.length, number: Number(number), width: indent.length + number.length + 1 + space.length, body };
}

function levelOf(item: Item, path: Item[]) {
  const sibling = path.findIndex((previous) => previous.indent === item.indent);
  if (sibling >= 0) return sibling;
  // Follow the content column rather than guessing which list an arbitrarily
  // indented line belongs to.
  const parent = path.findIndex((previous) => previous.width === item.indent && !!previous.body.trim());
  if (parent >= 0) return parent + 1;
  return item.indent <= 3 && (!path.length || item.indent < path[0].indent) ? 0 : -1;
}

function listContext(text: string, cursor: number) {
  const start = cursor === 0 ? 0 : text.lastIndexOf("\n", cursor - 1) + 1;
  const newline = text.indexOf("\n", cursor);
  const end = newline === -1 ? text.length : newline;
  const item = itemOn(text.slice(start, end));
  if (!item) return;

  const path: Item[] = [];
  let fence: { marker: string; column: number } | undefined;
  let paragraph = false;
  for (const line of text.slice(0, Math.max(0, start - 1)).split("\n")) {
    const preceding = itemOn(line);
    const level = preceding ? levelOf(preceding, path) : -1;
    const indent = /^ */.exec(line)![0].length;
    if (fence) {
      const closing = FENCE.exec(line);
      if (closing && indent >= fence.column && indent <= fence.column + 3 &&
          closing[2][0] === fence.marker[0] && closing[2].length >= fence.marker.length && !closing[3].trim()) {
        fence = undefined;
        continue;
      }
      if (!line.trim() || indent >= fence.column) continue;
      fence = undefined;
    }
    const inList = preceding && level >= 0 && (!paragraph || preceding.number === 1 || level < path.length);
    if (inList) path.splice(level, path.length, preceding);
    else if (line.trim()) {
      while (path.length && indent < path.at(-1)!.width) path.pop();
    }

    const column = inList ? preceding.width : path.at(-1)?.width ?? 0;
    const opening = FENCE.exec(inList ? preceding.body : line.slice(column));
    if (opening && opening[1].length <= 3 && (opening[2][0] === "~" || !opening[3].includes("`"))) {
      fence = { marker: opening[2], column };
    }
    paragraph = !!line.trim() && !inList && !fence;
  }
  if (fence && item.indent >= fence.column) return;
  const level = levelOf(item, path);
  if (level < 0 || (level === path.length && paragraph && item.number !== 1)) return;
  return { start, end, item, level, path };
}

export type ListCommand = "continue" | "indent" | "promote" | "exit";

export function editNumberedList(before: ComposerSnapshot, command: ListCommand): {
  next: ComposerSnapshot;
  intermediate?: ComposerSnapshot;
} | undefined {
  const { text, selection: { start: cursor, end: selectionEnd } } = before;
  if (cursor !== selectionEnd) return;
  const context = listContext(text, cursor);
  if (!context) return;
  const { start, end, item, level, path } = context;
  if (command === "exit") return { next: before };
  if (command === "continue") {
    if (cursor !== end) return;
    if (!item.body.trim()) {
      return { next: { text: text.slice(0, start) + text.slice(end), selection: caret(start) } };
    }
    if (item.number === MAX_NUMBER) return;
    const prefix = " ".repeat(item.indent) + (item.number + 1) + ". ";
    return {
      intermediate: { text: text.slice(0, end) + "\n" + text.slice(end), selection: caret(end + 1) },
      next: { text: text.slice(0, end) + "\n" + prefix + text.slice(end), selection: caret(end + 1 + prefix.length) },
    };
  }

  const destination = command === "indent" ? path[level] : path[level - 1];
  if (!destination?.body.trim()) return;
  const child = command === "indent" ? path[level + 1] : undefined;
  const number = (command === "indent" ? child?.number ?? 0 : destination.number) + 1;
  if (number > MAX_NUMBER) return;
  const indent = command === "indent" ? destination.width : destination.indent;
  const prefix = " ".repeat(indent) + number + ". ";
  const offset = Math.max(0, cursor - start - item.width);
  return {
    next: {
      text: text.slice(0, start) + prefix + text.slice(start + item.width),
      selection: caret(start + prefix.length + offset),
    },
  };
}
