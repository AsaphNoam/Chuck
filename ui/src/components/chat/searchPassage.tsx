import { createContext, useContext, type ReactNode } from "react";
import type { TranscriptEvent } from "../../api/types";
import type { SearchReveal } from "../../features/dashboard/searchReference";

export type TextRange = { start: number; end: number };
export const SearchPassageContext = createContext<SearchReveal | null>(null);
export const SearchEventContext = createContext<TranscriptEvent | null>(null);

// A byte boundary must coincide with a complete Unicode code point. Never
// address a JS string using a server's UTF-8 byte offset directly.
export function utf8Offset(text: string, bytes: number): number | null {
  if (!Number.isSafeInteger(bytes) || bytes < 0) return null;
  let count = 0, offset = 0;
  for (const point of text) {
    if (count === bytes) return offset;
    count += new TextEncoder().encode(point).length;
    offset += point.length;
    if (count > bytes) return null;
  }
  return count === bytes ? offset : null;
}

export function eventSourceSeqs(event: TranscriptEvent): number[] {
  const sources = event.search_sources as Array<{seq: number}> | undefined;
  const seqs = sources ? sources.map((source) => source.seq) : event.seq == null ? [] : [event.seq];
  if (typeof event.prompt_seq === "number") seqs.push(event.prompt_seq);
  return seqs;
}

export function sourceTextRanges(event: TranscriptEvent, text: string, partIndex: number, reveal: SearchReveal | null): TextRange[] {
  const sources = (event.search_sources as Array<{seq: number; start: number; end: number}> | undefined)
    ?? [{ seq: event.seq, start: 0, end: text.length }];
  const out: TextRange[] = [];
  for (const range of reveal?.ranges ?? []) {
    if (range.part_index !== partIndex) continue;
    const source = sources.find((item) => item.seq === range.seq);
    if (!source) continue;
    const original = text.slice(source.start, source.end);
    const start = utf8Offset(original, range.start_byte), end = utf8Offset(original, range.end_byte);
    if (start !== null && end !== null && end > start) out.push({ start: source.start + start, end: source.start + end });
  }
  return out.sort((a,b) => a.start-b.start);
}

export function useSourceTextRanges(text: string, partIndex = 0): TextRange[] {
  const event = useContext(SearchEventContext), reveal = useContext(SearchPassageContext);
  return event ? sourceTextRanges(event,text,partIndex,reveal) : [];
}

export function MarkedSourceText({ text, partIndex = 0 }: {text: string; partIndex?: number}) {
  const ranges = useSourceTextRanges(text, partIndex);
  let cursor = 0;
  const content: ReactNode[] = [];
  for (const range of ranges) {
    const start = Math.max(cursor,range.start);
    if (range.end <= start) continue;
    content.push(text.slice(cursor,start), <mark key={`${start}:${range.end}`} data-search-passage="true">{text.slice(start,range.end)}</mark>);
    cursor=range.end;
  }
  content.push(text.slice(cursor));
  return <>{content}</>;
}
