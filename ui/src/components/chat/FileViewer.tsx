import { useEffect, useLayoutEffect, useRef, useState, type MouseEvent, type ReactNode, type RefObject } from "react";
import { getFileContent } from "../../api/client";
import type { FileContent } from "../../api/types";
import { CodeBlock } from "./renderers/CodeBlock";
import { SanitizedMarkdown } from "./renderers/SanitizedMarkdown";
import { resolveFromFile, type FileLink } from "./renderers/filePath";
import type { AnnotationDraft } from "../../api/types";
import { clipAnnotationExcerpt } from "../../lib/annotations";
import { claimWebLink } from "../../lib/linkActions";

// The read-only, one-file viewer that opens beside the transcript (FS-03.R52).
// It holds no durable state: `?file=`/`?fileLine=` on the route are the open
// file's single source of truth (FS-03.R54), so this component is told which file
// to show and reports Close upward rather than owning either.

const MARKDOWN_LANGUAGES = new Set(["markdown"]);

type ViewState =
  | { status: "loading" }
  | { status: "error"; reason: string }
  | { status: "loaded"; file: FileContent; readAt: Date };

type DisplayMode = "rendered" | "source";
/** A reading location: the first visible source line or rendered top-level
 *  block, and how far its top sits below the body's top edge (TS-08.R130). */
type ReadingAnchor = { key: number; offset: number };

export type FileReader = ReturnType<typeof useFileReader>;

/**
 * useFileReader owns one open file's read, display mode and reading positions.
 * The agent workspace keeps it mounted while the same view moves between the
 * adjacent column and Focus file, so a layout change never reads the file
 * again (TS-08.R127). A null link closes the file and drops its snapshot.
 */
export function useFileReader({ agentId, link, load, onPathResolved }: {
  agentId: string;
  link: FileLink | null;
  load?: (path: string) => Promise<FileContent>;
  onPathResolved?: (path: string) => void;
}) {
  const [view, setView] = useState<ViewState>({ status: "loading" });
  const [rendered, setRendered] = useState(true);
  const [reloadKey, setReloadKey] = useState(0);
  // Per-read request token: opening another file, switching agents, closing, or
  // reloading must not be overwritten by a slower earlier read landing
  // afterwards — the same guard FilesTab/CommandsTab already carry (TS-08.R57, INV §1).
  const readToken = useRef(0);
  const loadRef = useRef(load);
  loadRef.current = load;
  const resolvedRef = useRef(onPathResolved);
  resolvedRef.current = onPathResolved;
  // One position per display mode for the current file, and a cited line still
  // waiting to be shown (FS-03.R95).
  const positions = useRef<Record<DisplayMode, ReadingAnchor | null>>({ rendered: null, source: null });
  const pendingLine = useRef<number | null>(null);

  // The server returns the normalized path it actually read. Publishing that
  // spelling back to the route makes header, URL, reload, and annotation anchors
  // name the same file (FS-03.R64); the matching route change is not a new read.
  const republished = useRef<string | null>(null);
  const path = link?.path ?? null;
  const line = link?.line ?? null;
  // Tracked during render so the view's layout effect already sees them: a
  // different file starts at its top, and a newly cited line takes priority
  // over any stored position. Republishing the same file's spelling is neither.
  const seen = useRef<{ path: string | null; line: number | null }>({ path: null, line: null });
  if (path !== seen.current.path && path !== republished.current) {
    positions.current = { rendered: null, source: null };
    pendingLine.current = line;
  } else if (line !== seen.current.line && line !== null) {
    pendingLine.current = line;
  }
  seen.current = { path, line };

  // Content is read when the file is opened and when Reload is chosen. There is
  // deliberately no watching and no polling, so the panel keeps showing the text
  // it read until someone reloads it (FS-03.R52). Reload keeps the reader's
  // place and clamps it to the new snapshot.
  useEffect(() => {
    if (path === null) {
      readToken.current += 1;
      republished.current = null;
      setView({ status: "loading" });
      return;
    }
    if (republished.current === path) {
      republished.current = null;
      return;
    }
    republished.current = null;
    const token = ++readToken.current;
    setView({ status: "loading" });
    (loadRef.current ? loadRef.current(path) : getFileContent(agentId, path))
      .then((file) => {
        if (readToken.current !== token) return;
        setView({ status: "loaded", file, readAt: new Date() });
        if (file.path !== path && resolvedRef.current) {
          republished.current = file.path;
          resolvedRef.current(file.path);
        }
      })
      .catch((error: unknown) => {
        if (readToken.current !== token) return;
        setView({ status: "error", reason: error instanceof Error ? error.message : "That file could not be read." });
      });
  }, [agentId, path, reloadKey]);

  return {
    link,
    view,
    rendered,
    setRendered,
    reload: () => setReloadKey((key) => key + 1),
    positions,
    pendingLine,
  };
}

export function FileViewer({ agentId, link, onClose, onOpenFile, onPathResolved, onSelectionMenu, load }: {
  agentId: string;
  link: FileLink;
  /** Reads the file from another retained context, such as a Think Tank
   *  participant source; agentId then names that context (TS-14.R12). */
  load?: (path: string) => Promise<FileContent>;
  onClose: () => void;
  onOpenFile?: (link: FileLink) => void;
  onPathResolved?: (path: string) => void;
  onSelectionMenu?: (selection: { x: number; y: number; text: string; draft: AnnotationDraft; link?: string }) => void;
}) {
  const reader = useFileReader({ agentId, link, load, onPathResolved });
  return <FileViewerView reader={reader} onClose={onClose} onOpenFile={onOpenFile} onSelectionMenu={onSelectionMenu} />;
}

/** The one reader view. The agent workspace shows it docked beside the
 *  conversation or inside Focus file; the room shows it alone. */
export function FileViewerView({ reader, variant, onClose, closeLabel = "Close", onOpenFile, onSelectionMenu, actions }: {
  reader: FileReader;
  variant?: "docked" | "focused";
  onClose: () => void;
  closeLabel?: string;
  onOpenFile?: (link: FileLink) => void;
  onSelectionMenu?: (selection: { x: number; y: number; text: string; draft: AnnotationDraft; link?: string }) => void;
  /** Presentation controls placed before Close, such as Focus file. */
  actions?: ReactNode;
}) {
  const { view, rendered, setRendered, positions } = reader;
  const link = reader.link as FileLink;
  const bodyRef = useRef<HTMLDivElement>(null);

  const shownPath = view.status === "loaded" ? view.file.path : link.path;
  const isMarkdown = view.status === "loaded" && MARKDOWN_LANGUAGES.has(view.file.language);
  const showRendered = isMarkdown && rendered;
  const mode: DisplayMode = showRendered ? "rendered" : "source";
  useReadingPosition(bodyRef, reader, mode);

  // A rendered file's own links are relative to that file, not to the working
  // directory the server resolves against (FS-03.R52).
  const openFromFile = onOpenFile && view.status === "loaded"
    ? (target: FileLink) => onOpenFile({ ...target, path: resolveFromFile(view.file.path, target.path) })
    : onOpenFile;

  const openSelectionMenu = (event: MouseEvent<HTMLDivElement>) => {
    if (!onSelectionMenu || view.status !== "loaded") return;
    const selection = window.getSelection();
    const host = bodyRef.current;
    if (!selection || selection.rangeCount === 0 || !host || !selection.anchorNode || !selection.focusNode ||
      !host.contains(selection.anchorNode) || !host.contains(selection.focusNode)) return;
    const text = selection.toString();
    if (!text.trim()) return;
    const draft: AnnotationDraft = {
      anchor_kind: "file",
      path: view.file.path,
      excerpt: clipAnnotationExcerpt(text.trim()),
      instruction: "",
    };
    if (!showRendered) {
      const start = fileLineForNode(selection.getRangeAt(0).startContainer, host);
      const end = fileLineForNode(selection.getRangeAt(0).endContainer, host);
      if (!start || !end) return;
      draft.start_line = Math.min(start, end);
      draft.end_line = Math.max(start, end);
    }
    event.preventDefault();
    onSelectionMenu({ x: event.clientX, y: event.clientY, text, draft, link: claimWebLink(event) });
  };

  return (
    <aside className="file-viewer" data-ui="file-viewer" data-state={view.status} data-variant={variant} aria-label="Open file">
      <header className="file-viewer-header" data-slot="header">
        <span className="file-viewer-path" title={shownPath}>{shownPath}</span>
        <div className="file-viewer-actions" data-slot="actions">
          {isMarkdown && (
            <div className="file-viewer-form" role="group" aria-label="Markdown display">
              <button type="button" aria-pressed={rendered} onClick={() => setRendered(true)}>Rendered</button>
              <button type="button" aria-pressed={!rendered} onClick={() => setRendered(false)}>Source</button>
            </div>
          )}
          <button type="button" className="ad-button-secondary" onClick={reader.reload}>Reload</button>
          {actions}
          <button type="button" className="ad-button-secondary" onClick={onClose}>{closeLabel}</button>
        </div>
      </header>
      {view.status === "loaded" && (
        <p className="file-viewer-meta" data-slot="metadata">
          {view.file.truncated
            ? `Showing the first ${view.file.line_count} lines — this file is larger than the viewer reads.`
            : `${view.file.line_count} line${view.file.line_count === 1 ? "" : "s"}`}
          {" · read "}
          <time dateTime={view.readAt.toISOString()}>{view.readAt.toLocaleTimeString()}</time>
        </p>
      )}
      <div
        className="file-viewer-body"
        data-slot="content"
        ref={bodyRef}
        onContextMenu={openSelectionMenu}
        onScroll={() => { if (bodyRef.current) positions.current[mode] = captureAnchor(bodyRef.current, mode); }}
      >
        {view.status === "loading" && <p className="tab-placeholder">Reading…</p>}
        {view.status === "error" && <p className="file-viewer-error" role="alert">{view.reason}</p>}
        {view.status === "loaded" && (showRendered ? (
          <div className="file-viewer-rendered">
            <SanitizedMarkdown text={view.file.content} onOpenFile={openFromFile} />
          </div>
        ) : (
          <CodeBlock language={view.file.language || "text"} showLineNumbers markedLine={link.line}>
            {view.file.content}
          </CodeBlock>
        ))}
      </div>
    </aside>
  );
}

// useReadingPosition shows a cited line once, otherwise puts the reader back
// where they were in this display mode: after mounting in another presentation,
// after a mode switch, after Reload, and whenever the body or its content is
// resized by a divider, the window or a late diagram (FS-03.R95, TS-08.R130).
function useReadingPosition(bodyRef: RefObject<HTMLDivElement | null>, reader: FileReader, mode: DisplayMode) {
  const { view, positions, pendingLine } = reader;
  useLayoutEffect(() => {
    const body = bodyRef.current;
    if (!body || view.status !== "loaded") return undefined;
    const line = pendingLine.current;
    if (line !== null && mode === "source") {
      const target = body.querySelector(`[data-file-line="${line}"]`);
      pendingLine.current = null;
      if (target) target.scrollIntoView({ block: "center" });
      else body.scrollTop = 0;
      positions.current.source = captureAnchor(body, mode);
    } else {
      restoreAnchor(body, mode, positions.current[mode]);
    }
    if (typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(() => restoreAnchor(body, mode, positions.current[mode]));
    observer.observe(body);
    if (body.firstElementChild) observer.observe(body.firstElementChild);
    return () => observer.disconnect();
  }, [view, mode, reader.link?.line]);
}

function anchorTargets(body: HTMLElement, mode: DisplayMode): HTMLElement[] {
  if (mode === "source") return [...body.querySelectorAll<HTMLElement>("[data-file-line]")];
  return [...(body.querySelector(".file-viewer-rendered")?.children ?? [])] as HTMLElement[];
}

function anchorKey(element: HTMLElement, index: number, mode: DisplayMode) {
  return mode === "source" ? Number(element.dataset.fileLine) : index;
}

function captureAnchor(body: HTMLElement, mode: DisplayMode): ReadingAnchor | null {
  if (body.scrollTop <= 0) return null;
  const top = body.getBoundingClientRect().top;
  const targets = anchorTargets(body, mode);
  const index = targets.findIndex((element) => element.getBoundingClientRect().bottom > top);
  if (index < 0) return null;
  return { key: anchorKey(targets[index], index, mode), offset: targets[index].getBoundingClientRect().top - top };
}

function restoreAnchor(body: HTMLElement, mode: DisplayMode, anchor: ReadingAnchor | null) {
  if (!anchor) {
    body.scrollTop = 0;
    return;
  }
  const targets = anchorTargets(body, mode);
  const target = targets.find((element, index) => anchorKey(element, index, mode) === anchor.key);
  // A reloaded file may have lost the anchored block; clamp to its end.
  if (!target) {
    body.scrollTop = body.scrollHeight;
    return;
  }
  const delta = target.getBoundingClientRect().top - body.getBoundingClientRect().top - anchor.offset;
  if (Math.abs(delta) >= 1) body.scrollTop += delta;
}

function fileLineForNode(node: Node, host: HTMLElement): number | null {
  const element = node.nodeType === Node.ELEMENT_NODE ? node as Element : node.parentElement;
  const line = element?.closest<HTMLElement>("[data-file-line]");
  if (!line || !host.contains(line)) return null;
  const value = Number(line.dataset.fileLine);
  return value > 0 ? value : null;
}
