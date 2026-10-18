import { useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode, type RefObject } from "react";
import { getFileContent } from "../../api/client";
import type { FileContent } from "../../api/types";
import { CodeBlock } from "./renderers/CodeBlock";
import { SanitizedMarkdown } from "./renderers/SanitizedMarkdown";
import type { FileLink } from "./renderers/filePath";
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

type DisplayMode = "rendered" | "formatted" | "source";
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
  const positions = useRef<Record<DisplayMode, ReadingAnchor | null>>({ rendered: null, formatted: null, source: null });
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
  // The previous file's snapshot is dropped in the same render, so neither the
  // cited line nor a scroll is ever applied to the file being replaced.
  const seen = useRef<{ path: string | null; line: number | null }>({ path: null, line: null });
  if (path !== seen.current.path && path !== republished.current) {
    positions.current = { rendered: null, formatted: null, source: null };
    pendingLine.current = line;
    if (view.status !== "loading") setView({ status: "loading" });
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
export function FileViewerView({ reader, variant, onClose, closeLabel = "Close", onOpenFile, onSelectionMenu, onAnnotateSelection, actions }: {
  reader: FileReader;
  variant?: "docked" | "focused";
  /** Omitted where the surrounding surface owns closing, as on the phone. */
  onClose?: () => void;
  closeLabel?: string;
  onOpenFile?: (link: FileLink) => void;
  onSelectionMenu?: (selection: { x: number; y: number; text: string; draft: AnnotationDraft; link?: string }) => void;
  /** An explicit Annotate selection action for touch and keyboard (FS-13.R28). */
  onAnnotateSelection?: (draft: AnnotationDraft) => void;
  /** Presentation controls placed before Close, such as Focus file. */
  actions?: ReactNode;
}) {
  const { view, rendered, setRendered, positions } = reader;
  const link = reader.link as FileLink;
  const bodyRef = useRef<HTMLDivElement>(null);

  const shownPath = view.status === "loaded" ? view.file.path : link.path;
  const isMarkdown = view.status === "loaded" && MARKDOWN_LANGUAGES.has(view.file.language);
  const formattedJSON = useMemo(() => {
    if (view.status !== "loaded" || view.file.language !== "json" || view.file.truncated) return null;
    try { return formatJSON(view.file.content); }
    catch { return null; }
  }, [view]);
  // A source citation always wins over the presentation preference.
  const showFormatted = formattedJSON !== null && rendered && link.line === undefined;
  const showRendered = isMarkdown && rendered;
  const mode: DisplayMode = showRendered ? "rendered" : showFormatted ? "formatted" : "source";
  useReadingPosition(bodyRef, reader, mode);

  // The one mapping from a selection to a file draft, for the context menu and
  // the touch action alike (TS-08.R133): Source captures its line range,
  // Rendered only the path (FS-13.R25).
  const currentSelection = () => view.status === "loaded" && !view.file.image && !showFormatted ? selectionDraft(bodyRef.current, view.file.path, showRendered) : null;

  const openSelectionMenu = (event: MouseEvent<HTMLDivElement>) => {
    if (!onSelectionMenu) return;
    const selected = currentSelection();
    if (!selected) return;
    event.preventDefault();
    onSelectionMenu({ x: event.clientX, y: event.clientY, text: selected.text, draft: selected.draft, link: claimWebLink(event) });
  };

  // A tap can collapse the selection before click, so it is read on press.
  const pressed = useRef<AnnotationDraft | null>(null);
  const [selectionHint, setSelectionHint] = useState(false);
  const annotateSelection = () => {
    const draft = pressed.current ?? currentSelection()?.draft ?? null;
    pressed.current = null;
    setSelectionHint(!draft);
    if (draft) onAnnotateSelection?.(draft);
  };

  return (
    <aside className="file-viewer" data-ui="file-viewer" data-state={view.status} data-variant={variant} aria-label="Open file">
      <header className="file-viewer-header" data-slot="header">
        <span className="file-viewer-path" title={shownPath}>{shownPath}</span>
        <div className="file-viewer-actions" data-slot="actions">
          {(isMarkdown || formattedJSON !== null) && (
            <div className="file-viewer-form" role="group" aria-label={isMarkdown ? "Markdown display" : "JSON display"}>
              <button type="button" aria-pressed={showRendered || showFormatted} disabled={!isMarkdown && link.line !== undefined} title={!isMarkdown && link.line !== undefined ? "Showing the cited source line; open this file without a line citation to format it." : undefined} onClick={() => setRendered(true)}>{isMarkdown ? "Rendered" : "Formatted"}</button>
              <button type="button" aria-pressed={!showRendered && !showFormatted} onClick={() => setRendered(false)}>Source</button>
            </div>
          )}
          <button type="button" className="ad-button-secondary" onClick={reader.reload}>Reload</button>
          {onAnnotateSelection && view.status === "loaded" && !view.file.image && !showFormatted && (
            <button type="button" className="ad-button-secondary" onPointerDown={() => { pressed.current = currentSelection()?.draft ?? null; }} onClick={(click) => { click.currentTarget.focus(); annotateSelection(); }}>Annotate selection</button>
          )}
          {actions}
          {onClose && <button type="button" className="ad-button-secondary" onClick={onClose}>{closeLabel}</button>}
        </div>
      </header>
      {selectionHint && view.status === "loaded" && <p className="file-viewer-meta" role="status">Select text in the file to annotate it.</p>}
      {view.status === "loaded" && (
        <p className="file-viewer-meta" data-slot="metadata">
          {view.file.image ? "Image preview" : view.file.truncated
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
        // Only the loaded document records a position: the Reading… or error
        // placeholder's own scroll (clamped to its top) must not replace it.
        onScroll={() => { if (bodyRef.current && view.status === "loaded") positions.current[mode] = captureAnchor(bodyRef.current, mode); }}
      >
        {view.status === "loading" && <p className="tab-placeholder">Reading…</p>}
        {view.status === "error" && <p className="file-viewer-error" role="alert">{view.reason}</p>}
        {view.status === "loaded" && (view.file.image ? (
          <FileImage key={view.readAt.getTime()} image={view.file.image} path={view.file.path} />
        ) : showRendered ? (
          <div className="file-viewer-rendered">
            <SanitizedMarkdown text={view.file.content} onOpenFile={onOpenFile} sourcePath={view.file.path} />
          </div>
        ) : (
          <CodeBlock language={view.file.language || "text"} className={`language-${view.file.language || "text"}`} showLineNumbers={!showFormatted} markedLine={link.line}>
            {showFormatted ? formattedJSON! : view.file.content}
          </CodeBlock>
        ))}
      </div>
    </aside>
  );
}

function FileImage({ image, path }: { image: NonNullable<FileContent["image"]>; path: string }) {
  const [src, setSrc] = useState("");
  const [failed, setFailed] = useState(false);
  useEffect(() => {
    if (!["image/png", "image/jpeg", "image/webp", "image/gif"].includes(image.mime_type)) {
      setFailed(true);
      return;
    }
    try {
      const bytes = Uint8Array.from(atob(image.data), (char) => char.charCodeAt(0));
      const url = URL.createObjectURL(new Blob([bytes], { type: image.mime_type }));
      setSrc(url);
      return () => URL.revokeObjectURL(url);
    } catch { setFailed(true); }
  }, [image]);
  return failed ? <p className="file-viewer-error" role="alert">This image could not be displayed. Choose Reload to try again.</p>
    : <img className="file-viewer-image" src={src || undefined} alt={path} onError={() => setFailed(true)} />;
}

// Validate first, then change whitespace only: parsing and reserializing would
// round large integer IDs and could change escaped strings in the displayed file.
function formatJSON(source: string): string {
  JSON.parse(source);
  const tokens = source.match(/"(?:\\.|[^"\\])*"|[^\s]/g) ?? [];
  let depth = 0;
  let output = "";
  const newline = () => `\n${"  ".repeat(depth)}`;
  tokens.forEach((token, index) => {
    if (token === "{" || token === "[") {
      output += token;
      depth += 1;
      if (tokens[index + 1] !== "}" && tokens[index + 1] !== "]") output += newline();
    } else if (token === "}" || token === "]") {
      depth -= 1;
      if (tokens[index - 1] !== "{" && tokens[index - 1] !== "[") output += newline();
      output += token;
    } else if (token === ",") output += `,${newline()}`;
    else if (token === ":") output += ": ";
    else output += token;
    // Deeply nested input can expand quadratically through indentation.
    // Keep formatting bounded and fall back to Source when it becomes too large.
    if (depth > 128 || output.length > 2 * 1024 * 1024) throw new Error("JSON formatting limit");
  });
  return output;
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
  if (mode === "formatted") return [];
  return [...(body.querySelector(".file-viewer-rendered")?.children ?? [])] as HTMLElement[];
}

function anchorKey(element: HTMLElement, index: number, mode: DisplayMode) {
  return mode === "source" ? Number(element.dataset.fileLine) : index;
}

function captureAnchor(body: HTMLElement, mode: DisplayMode): ReadingAnchor | null {
  if (body.scrollTop <= 0) return null;
  if (mode === "formatted") return { key: 0, offset: -body.scrollTop };
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
  if (mode === "formatted") {
    body.scrollTop = -anchor.offset;
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

function selectionDraft(host: HTMLElement | null, path: string, rendered: boolean): { text: string; draft: AnnotationDraft } | null {
  const selection = window.getSelection();
  if (!selection || selection.rangeCount === 0 || !host || !selection.anchorNode || !selection.focusNode ||
    !host.contains(selection.anchorNode) || !host.contains(selection.focusNode)) return null;
  const text = selection.toString();
  if (!text.trim()) return null;
  const draft: AnnotationDraft = { anchor_kind: "file", path, excerpt: clipAnnotationExcerpt(text.trim()), instruction: "" };
  if (!rendered) {
    const start = fileLineForNode(selection.getRangeAt(0).startContainer, host);
    const end = fileLineForNode(selection.getRangeAt(0).endContainer, host);
    if (!start || !end) return null;
    draft.start_line = Math.min(start, end);
    draft.end_line = Math.max(start, end);
  }
  return { text, draft };
}

function fileLineForNode(node: Node, host: HTMLElement): number | null {
  const element = node.nodeType === Node.ELEMENT_NODE ? node as Element : node.parentElement;
  const line = element?.closest<HTMLElement>("[data-file-line]");
  if (!line || !host.contains(line)) return null;
  const value = Number(line.dataset.fileLine);
  return value > 0 ? value : null;
}
