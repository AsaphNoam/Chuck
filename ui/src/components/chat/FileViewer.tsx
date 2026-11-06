import { useEffect, useLayoutEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode, type RefObject } from "react";
import { getFileContent, probeFileContent } from "../../api/client";
import type { FileContent, FileProbe } from "../../api/types";
import { useFileFreshness } from "../../features/fileTabs/useFileFreshness";
import { FileEditor } from "../../features/fileTabs/FileEditor";
import { confirmDraftLeave, draftPersistenceUnsafe, draftKey, getDraft, restoreDrafts, type DraftIdentity } from "../../features/fileTabs/drafts";
import { CodeBlock } from "./renderers/CodeBlock";
import { SanitizedMarkdown } from "./renderers/SanitizedMarkdown";
import type { FileLink } from "./renderers/filePath";
import type { AnnotationDraft } from "../../api/types";
import { clipAnnotationExcerpt } from "../../lib/annotations";
import { claimWebLink, isHtmlPath, webLinkActions, type OpenInBrowser } from "../../lib/linkActions";
import { PointerContextMenu, type PointerMenuState } from "../ui/PointerContextMenu";
import { useUiStore } from "../../store/uiStore";
import { HtmlPreview } from "./HtmlPreview";
import { emptyAnchors, type Anchors, type DisplayMode, type ReadingAnchor, type TabView } from "../../features/fileTabs/model";
import { ReadDropped, type ReadKind } from "../../features/fileTabs/readLimiter";

// The read-only viewer for one open file (FS-03.R52). It holds no durable state:
// the surface's saved file tabs decide which file it shows (FS-03.R101–R103,
// `features/fileTabs`), so it is told which file to show and reports Close upward.

const MARKDOWN_LANGUAGES = new Set(["markdown"]);
const editRefusal = (reason?: string) => ({
  incomplete_text: "Only complete text files can be edited.",
  remote_read_only: "Phone previews are read-only.",
  symlink: "Symbolic links are read-only.", hard_link: "Hard-linked files are read-only.",
  not_owned: "The server account does not own this file.", not_writable: "This file is not writable.",
  unsupported_metadata: "This file's access metadata cannot be safely preserved.",
  parent_not_writable: "The file's folder is not writable.",
} as Record<string, string>)[reason ?? ""] ?? "This file cannot be safely edited.";

type ViewState =
  | { status: "loading" }
  | { status: "error"; reason: string }
  | { status: "loaded"; file: FileContent; readAt: Date };

export type FileReader = ReturnType<typeof useFileReader>;

/**
 * useFileReader owns one open file's read, display mode and reading positions.
 * The agent workspace keeps it mounted while the same view moves between the
 * adjacent column and Focus file, so a layout change never reads the file
 * again (TS-08.R127). A null link closes the file and drops its snapshot.
 *
 * A file tab's reader starts from its saved view, reports view changes back to
 * the saved tab, and reads through its owner's limiter (TS-08.R142/R144).
 * `jump` changes when the tab is reopened at a citation, which then wins over
 * the saved position once.
 */
export function useFileReader({ agentId, link, load, probe, editor, selected = true, onPathResolved, initialView, jump = 0, onViewChange, schedule, cancelPending }: {
  agentId: string;
  link: FileLink | null;
  load?: (path: string) => Promise<FileContent>;
  probe?: (path: string) => Promise<FileProbe>;
  selected?: boolean;
  editor?: { identity: DraftIdentity; save: (baseRevision: string, content: string) => Promise<FileContent> };
  onPathResolved?: (path: string) => void;
  initialView?: TabView;
  jump?: number;
  onViewChange?: (view: TabView) => void;
  schedule?: <T>(kind: ReadKind, start: () => Promise<T>) => Promise<T>;
  cancelPending?: () => void;
}) {
  const [view, setView] = useState<ViewState>({ status: "loading" });
  const [rendered, setRenderedState] = useState(initialView?.rendered ?? true);
  const [reloadKey, setReloadKey] = useState(0);
  // A tab's read that its limiter dropped before it started; it reads again
  // when the tab is next selected.
  const [dropped, setDropped] = useState(false);
  const [selectionActive, setSelectionActive] = useState(false);
  const [editing, setEditing] = useState(false);
  const [readError, setReadError] = useState<string | null>(null);
  const [reading, setReading] = useState(false);
  // Per-read request token: opening another file, switching agents, closing, or
  // reloading must not be overwritten by a slower earlier read landing
  // afterwards — the same guard FilesTab/CommandsTab already carry (TS-08.R57, INV §1).
  const readToken = useRef(0);
  const loadRef = useRef(load);
  loadRef.current = load;
  const resolvedRef = useRef(onPathResolved);
  resolvedRef.current = onPathResolved;
  const viewChangeRef = useRef(onViewChange);
  viewChangeRef.current = onViewChange;
  const scheduleRef = useRef(schedule);
  scheduleRef.current = schedule;
  // One position per display mode for the current file, and a cited line still
  // waiting to be shown (FS-03.R95).
  const positions = useRef<Anchors>(initialView ? { ...initialView.anchors } : emptyAnchors());
  const pendingLine = useRef<number | null>(initialView && jump > 0 ? link?.line ?? null : null);
  const seenJump = useRef(jump);
  if (jump !== seenJump.current) {
    seenJump.current = jump;
    if (link?.line) pendingLine.current = link.line;
  }

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
  // A tab's reader begins already showing its own file, so its saved position
  // is not mistaken for a previous file's.
  const seen = useRef<{ path: string | null; line: number | null }>(initialView ? { path, line } : { path: null, line: null });
  if (path !== seen.current.path && path !== republished.current) {
    positions.current = emptyAnchors();
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
    setReadError(null);
    setReading(true);
    setView((previous) => previous.status === "loaded" ? previous : { status: "loading" });
    setDropped(false);
    const start = () => (loadRef.current ? loadRef.current(path) : getFileContent(agentId, path));
    (scheduleRef.current ? scheduleRef.current(reloadKey > 0 ? "reload" : "open", start) : start())
      .then((file) => {
        if (readToken.current !== token) return;
        setReading(false);
        setView({ status: "loaded", file, readAt: new Date() });
        if (file.path !== path && resolvedRef.current) {
          republished.current = file.path;
          resolvedRef.current(file.path);
        }
      })
      .catch((error: unknown) => {
        if (readToken.current !== token) return;
        setReading(false);
        if (!(error instanceof ReadDropped)) setReadError(error instanceof Error ? error.message : "That file could not be read.");
        if (error instanceof ReadDropped) setDropped(true);
        else setView((previous) => previous.status === "loaded" ? previous : { status: "error", reason: error instanceof Error ? error.message : "That file could not be read." });
      });
  }, [agentId, path, reloadKey]);
  // An unmounted reader's late reply must not report a path for a closed tab.
  useEffect(() => () => { readToken.current += 1; }, []);

  const apply = (file: FileContent) => { setReadError(null); setView({ status: "loaded", file, readAt: new Date() }); };
  const read = () => loadRef.current ? loadRef.current(path!) : getFileContent(agentId, path!);
  const freshness = useFileFreshness({
    identity: `${agentId}:${path}`, selected: selected && path !== null,
    snapshot: view.status === "loaded" ? view.file : null,
    probe: path !== null && (probe || !load) ? () => {
      const start = () => probe ? probe(path) : probeFileContent(agentId, path);
      return scheduleRef.current ? scheduleRef.current("open", start) : start();
    } : undefined,
    read: () => scheduleRef.current ? scheduleRef.current("open", read) : read(),
    apply, blocked: selectionActive || editing, retry: reloadKey, cancelPending,
  });

  const setRendered = (next: boolean) => {
    setRenderedState(next);
    viewChangeRef.current?.({ rendered: next, anchors: positions.current });
  };
  return {
    link,
    view,
    rendered,
    freshness: { ...freshness, changing: freshness.changing || reading, stale: readError || freshness.stale },
    editing,
    editor,
    setEditing,
    cancelEdit: () => { setEditing(false); setReloadKey((key) => key + 1); },
    setSelectionActive,
    apply,
    read,
    setRendered,
    // A Reload already reading is the same request (TS-08.R144).
    reload: () => { if (!editing && !reading && (view.status !== "loading" || dropped)) setReloadKey((key) => key + 1); },
    dropped,
    positions,
    pendingLine,
    jump,
    positionsChanged: () => viewChangeRef.current?.({ rendered, anchors: positions.current }),
  };
}

export function FileViewer({ agentId, link, onClose, onOpenFile, onPathResolved, onSelectionMenu, onOpenInBrowser, load }: {
  agentId: string;
  link: FileLink;
  /** Reads the file from another retained context, such as a Think Tank
   *  participant source; agentId then names that context (TS-14.R12). */
  load?: (path: string) => Promise<FileContent>;
  onClose: () => void;
  onOpenFile?: (link: FileLink) => void;
  onPathResolved?: (path: string) => void;
  onOpenInBrowser?: OpenInBrowser;
  onSelectionMenu?: (selection: { x: number; y: number; text: string; draft: AnnotationDraft; link?: string }) => void;
}) {
  const reader = useFileReader({ agentId, link, load, onPathResolved });
  return <FileViewerView reader={reader} onClose={onClose} onOpenFile={onOpenFile} onSelectionMenu={onSelectionMenu} onOpenInBrowser={onOpenInBrowser} />;
}

/** The one reader view. The agent workspace shows it docked beside the
 *  conversation or inside Focus file; the room shows it alone. */
export function FileViewerView({ reader, variant, onClose, closeLabel = "Close", onOpenFile, onSelectionMenu, onAnnotateSelection, onOpenInBrowser, actions, tabs }: {
  reader: FileReader;
  variant?: "docked" | "focused";
  /** Omitted where the surrounding surface owns closing, as on the phone. */
  onClose?: () => void;
  closeLabel?: string;
  onOpenFile?: (link: FileLink) => void;
  onOpenInBrowser?: OpenInBrowser;
  onSelectionMenu?: (selection: { x: number; y: number; text: string; draft: AnnotationDraft; link?: string }) => void;
  /** An explicit Annotate selection action for touch and keyboard (FS-13.R28). */
  onAnnotateSelection?: (draft: AnnotationDraft) => void;
  /** Presentation controls placed before Close, such as Focus file. */
  actions?: ReactNode;
  /** The owner's file-tab strip, above the open file's header. */
  tabs?: ReactNode;
}) {
  const { view, rendered, setRendered, positions } = reader;
  const link = reader.link as FileLink;
  const bodyRef = useRef<HTMLDivElement>(null);
  const [menu, setMenu] = useState<PointerMenuState | null>(null);
  const pushError = useUiStore((state) => state.pushError);
  const editorIdentity = reader.editor?.identity;
  const recoveredDraft = editorIdentity ? getDraft(editorIdentity) : undefined;
  const editInitial: FileContent | undefined = view.status === "loaded" ? view.file : recoveredDraft ? {
    agent_id: editorIdentity!.owner, path: recoveredDraft.path, content: recoveredDraft.text,
    revision: recoveredDraft.baseRevision, size: new TextEncoder().encode(recoveredDraft.text).length,
    mod_time: "", line_count: 0, language: "text", truncated: false,
  } : undefined;
  useEffect(() => {
    let active = true;
    if (editorIdentity) void restoreDrafts().then(() => {
      if (active && getDraft(editorIdentity)) reader.setEditing(true);
    });
    return () => { active = false; };
  }, [editorIdentity?.owner, editorIdentity?.source, editorIdentity?.path]);
  useEffect(() => {
    const changed = () => {
      const selection = window.getSelection();
      reader.setSelectionActive(Boolean(selection && !selection.isCollapsed && bodyRef.current?.contains(selection.anchorNode)));
    };
    document.addEventListener("selectionchange", changed);
    return () => { document.removeEventListener("selectionchange", changed); reader.setSelectionActive(false); };
  }, [reader.setSelectionActive]);

  const shownPath = view.status === "loaded" ? view.file.path : link.path;
  const isMarkdown = view.status === "loaded" && MARKDOWN_LANGUAGES.has(view.file.language);
  const isHtml = view.status === "loaded" && isHtmlPath(view.file.path);
  const canPreviewHtml = isHtml && view.status === "loaded" && !view.file.truncated;
  const formattedJSON = useMemo(() => {
    if (view.status !== "loaded" || view.file.language !== "json" || view.file.truncated) return null;
    try { return formatJSON(view.file.content); }
    catch { return null; }
  }, [view]);
  // A source citation always wins over the presentation preference.
  const showFormatted = formattedJSON !== null && rendered && link.line === undefined;
  const showRendered = isMarkdown && rendered;
  const showHtml = canPreviewHtml && rendered && link.line === undefined;
  const mode: DisplayMode = showRendered ? "rendered" : showFormatted ? "formatted" : "source";
  useReadingPosition(bodyRef, reader, mode, !showHtml);

  // The one mapping from a selection to a file draft, for the context menu and
  // the touch action alike (TS-08.R133): Source captures its line range,
  // Rendered only the path (FS-13.R25).
  const currentSelection = () => !reader.editing && view.status === "loaded" && !view.file.image && !showFormatted && !showHtml ? selectionDraft(bodyRef.current, view.file.path, showRendered) : null;

  const openFileMenu = (x: number, y: number) => setMenu({ x, y, actions: webLinkActions(shownPath, pushError, onOpenInBrowser) });
  useEffect(() => { setMenu(null); }, [shownPath, view.status, rendered]);

  const openSelectionMenu = (event: MouseEvent<HTMLDivElement>) => {
    const selected = currentSelection();
    if (selected && onSelectionMenu) {
      event.preventDefault();
      onSelectionMenu({ x: event.clientX, y: event.clientY, text: selected.text, draft: selected.draft, link: claimWebLink(event) ?? (isHtml ? shownPath : undefined) });
    } else if (isHtml) {
      event.preventDefault();
      openFileMenu(event.clientX, event.clientY);
    }
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
      {tabs}
      <header className="file-viewer-header" data-slot="header">
        <span className="file-viewer-path" title={shownPath} onContextMenu={isHtml ? (event) => { event.preventDefault(); openFileMenu(event.clientX, event.clientY); } : undefined}>{shownPath}</span>
        <div className="file-viewer-actions" data-slot="actions">
          {!reader.editing && (isMarkdown || canPreviewHtml || formattedJSON !== null) && (
            <div className="file-viewer-form" role="group" aria-label={isMarkdown ? "Markdown display" : isHtml ? "HTML display" : "JSON display"}>
              <button type="button" aria-pressed={showRendered || showHtml || showFormatted} disabled={!isMarkdown && link.line !== undefined} title={!isMarkdown && link.line !== undefined ? "Showing the cited source line; open this file without a line citation to render it." : undefined} onClick={() => setRendered(true)}>{isMarkdown || isHtml ? "Rendered" : "Formatted"}</button>
              <button type="button" aria-pressed={!showRendered && !showHtml && !showFormatted} onClick={() => setRendered(false)}>Source</button>
            </div>
          )}
          <button type="button" className="ad-button-secondary" disabled={reader.editing} title={reader.editing ? "Save or Cancel the edit before reloading." : undefined} onClick={reader.reload}>Reload</button>
          {reader.editor && !reader.editing && view.status === "loaded" && <button type="button" className="ad-button-secondary" disabled={!view.file.editable || !view.file.revision} title={view.file.editable ? undefined : `Read-only: ${editRefusal(view.file.edit_refusal)}`} onClick={() => reader.setEditing(true)}>Edit</button>}
          {!reader.editing && onAnnotateSelection && view.status === "loaded" && !view.file.image && !showFormatted && !showHtml && (
            <button type="button" className="ad-button-secondary" onPointerDown={() => { pressed.current = currentSelection()?.draft ?? null; }} onClick={(click) => { click.currentTarget.focus(); annotateSelection(); }}>Annotate selection</button>
          )}
          {actions}
          {onClose && <button type="button" className="ad-button-secondary" onClick={() => {
            if (reader.editor && draftPersistenceUnsafe()) void confirmDraftLeave().then((allowed) => { if (allowed) onClose(); });
            else onClose();
          }}>{closeLabel}</button>}
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
          <span className="file-freshness-indicator" data-changing={reader.freshness.changing || undefined} role="img" aria-label={reader.freshness.changing ? "File changing or updating" : "File checked"} />
          {reader.freshness.available && " · Update available"}
          {reader.freshness.stale && <span role="alert"> · Stale: {reader.freshness.stale} <button type="button" onClick={reader.reload} disabled={reader.editing}>Retry</button></span>}
          {reader.editor && !view.file.editable && <span> · Read-only: {editRefusal(view.file.edit_refusal)}</span>}
        </p>
      )}
      <div
        className="file-viewer-body"
        data-html-preview={showHtml ? "true" : undefined}
        data-slot="content"
        ref={bodyRef}
        onContextMenu={openSelectionMenu}
        // Only the loaded document records a position: the Reading… or error
        // placeholder's own scroll (clamped to its top) must not replace it.
        onScroll={() => {
          if (!bodyRef.current || view.status !== "loaded" || showHtml) return;
          positions.current[mode] = captureAnchor(bodyRef.current, mode);
          reader.positionsChanged();
        }}
      >
        {view.status === "loading" && <p className="tab-placeholder">Reading…</p>}
        {view.status === "error" && <p className="file-viewer-error" role="alert">{view.reason}</p>}
        {reader.editing && reader.editor && editInitial ? (
          <FileEditor key={draftKey(reader.editor.identity)} identity={reader.editor.identity} initial={editInitial} displayMode={mode}
            load={reader.read} save={reader.editor.save}
            onSaved={(file) => { reader.apply(file); reader.setEditing(false); }}
            onCancel={reader.cancelEdit} />
        ) : view.status === "loaded" && (view.file.image ? (
          <FileImage key={view.readAt.getTime()} image={view.file.image} path={view.file.path} />
        ) : showHtml ? (
          <HtmlPreview key={view.readAt.getTime()} content={view.file.content} path={view.file.path} onContextMenu={onOpenInBrowser ? openFileMenu : undefined} />
        ) : showRendered ? (
          <div className="file-viewer-rendered">
            <SanitizedMarkdown text={view.file.content} onOpenFile={onOpenFile} sourcePath={view.file.path} onOpenInBrowser={onOpenInBrowser} />
          </div>
        ) : (
          <CodeBlock language={view.file.language || "text"} className={`language-${view.file.language || "text"}`} showLineNumbers={!showFormatted} markedLine={link.line}>
            {showFormatted ? formattedJSON! : view.file.content}
          </CodeBlock>
        ))}
      </div>
      <PointerContextMenu menu={menu} onClose={() => setMenu(null)} />
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
function useReadingPosition(bodyRef: RefObject<HTMLDivElement | null>, reader: FileReader, mode: DisplayMode, enabled = true) {
  const { view, positions, pendingLine } = reader;
  useLayoutEffect(() => {
    const body = bodyRef.current;
    if (!enabled || !body || view.status !== "loaded") return undefined;
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
  }, [view, mode, reader.link?.line, reader.jump, enabled]);
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
