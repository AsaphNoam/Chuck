import { createContext, useContext, useLayoutEffect, useMemo, useRef, useState, type RefObject } from "react";
import type { TranscriptEvent } from "../../api/types";
import { canAnnotate, annotationClippedMarker } from "../../lib/annotations";
import { eventSourceSeqs } from "./searchPassage";
import { kindOf } from "./toolRun";

type SourceLine = { side: "old" | "new"; number: number };
type AnnotationJump = { seq: number; line?: SourceLine; land: (row: HTMLElement) => void };

export const AnnotationLinkContext = createContext<((seq: number, excerpt: string, line?: SourceLine) => (() => void) | undefined) | null>(null);
export const AnnotationJumpContext = createContext<AnnotationJump | undefined>(undefined);

export function useAnnotationNavigation(
  events: TranscriptEvent[],
  agentId: string,
  reveal: (seq: number) => void,
  viewport?: RefObject<HTMLElement>,
  supportsSource = canAnnotate,
) {
  const sources = useMemo(() => {
    const index = new Map<number, number>();
    if (!events.some((event) => kindOf(event) === "annotation")) return index;
    for (const event of events) {
      if (typeof event.seq !== "number" || !supportsSource(event) || kindOf(event) === "tool_result") continue;
      for (const seq of eventSourceSeqs(event)) index.set(seq, event.seq);
    }
    return index;
  }, [events, supportsSource]);
  const [request, setRequest] = useState<AnnotationJump & { agentId: string }>();
  return {
    jump: request?.agentId === agentId ? request : undefined,
    link: (seq: number, excerpt: string, line?: SourceLine) => {
      const source = sources.get(seq);
      if (source === undefined) return undefined;
      return () => {
        let landed = false;
        setRequest({ agentId, seq: source, line, land: (row) => {
          if (landed) return;
          landed = true;
          scrollToSource(row, excerpt, viewport?.current);
        } });
        reveal(source);
      };
    },
  };
}

export function useAnnotationSource(seq: number | undefined) {
  const host = useRef<HTMLDivElement>(null);
  const jump = useContext(AnnotationJumpContext);
  useLayoutEffect(() => {
    if (!jump || jump.seq !== seq) return;
    const frame = requestAnimationFrame(() => { if (host.current) jump.land(host.current); });
    return () => cancelAnimationFrame(frame);
  }, [host, jump, seq]);
  return host;
}

function scrollToSource(row: HTMLElement, excerpt: string, viewport?: HTMLElement | null) {
  const line = row.querySelector("[data-annotation-line]")?.closest("tr");
  const passage = line ? undefined : findPassage(row, excerpt);
  for (let element = passage?.startContainer.parentElement; element && row.contains(element); element = element.parentElement) {
    if (element instanceof HTMLDetailsElement) element.open = true;
  }
  const bounds = line?.getBoundingClientRect() ?? passage?.getClientRects?.()[0];
  if (bounds?.height) {
    const view = viewport?.getBoundingClientRect();
    const center = view ? (view.top + view.bottom) / 2 : window.innerHeight / 2;
    const offset = bounds.top + bounds.height / 2 - center;
    if (viewport) viewport.scrollTop += offset;
    else window.scrollBy(0, offset);
  } else {
    row.scrollIntoView?.({ block: "center" });
  }
  row.tabIndex = -1;
  row.focus({ preventScroll: true });
}

function findPassage(row: HTMLElement, excerpt: string): Range | undefined {
  const quote = excerpt.endsWith(annotationClippedMarker) ? excerpt.slice(0, -annotationClippedMarker.length).trim() : excerpt.trim();
  if (!quote) return undefined;
  const text = row.textContent ?? "";
  const start = text.indexOf(quote);
  if (start < 0 || text.indexOf(quote, start + 1) >= 0) return undefined;
  const end = start + quote.length;
  const range = document.createRange();
  const walker = document.createTreeWalker(row, NodeFilter.SHOW_TEXT);
  let offset = 0;
  while (walker.nextNode()) {
    const node = walker.currentNode;
    const next = offset + (node.nodeValue?.length ?? 0);
    if (start >= offset && start < next) range.setStart(node, start - offset);
    if (end <= next) {
      range.setEnd(node, end - offset);
      return range;
    }
    offset = next;
  }
}
