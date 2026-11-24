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

export function findPassage(row: HTMLElement, excerpt: string): Range | undefined {
  const quote = excerpt.endsWith(annotationClippedMarker) ? excerpt.slice(0, -annotationClippedMarker.length).trim() : excerpt.trim();
  // The browser's selection text adds line breaks between block elements that
  // the DOM's text nodes do not hold, so both sides are compared without
  // whitespace and each compared character keeps its source node and offset.
  const target = quote.replace(/\s+/g, "");
  if (!target) return undefined;
  let text = "";
  const points: [Text, number][] = [];
  const walker = document.createTreeWalker(row, NodeFilter.SHOW_TEXT);
  while (walker.nextNode()) {
    const node = walker.currentNode as Text;
    const value = node.nodeValue ?? "";
    for (let index = 0; index < value.length; index++) {
      if (/\s/.test(value[index])) continue;
      text += value[index];
      points.push([node, index]);
    }
  }
  const start = text.indexOf(target);
  if (start < 0 || text.indexOf(target, start + 1) >= 0) return undefined;
  const [last, lastOffset] = points[start + target.length - 1];
  const range = document.createRange();
  range.setStart(...points[start]);
  range.setEnd(last, lastOffset + 1);
  return range;
}
