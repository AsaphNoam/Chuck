import { useCallback, useEffect, useMemo, useRef, useState, type KeyboardEvent, type MouseEvent } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import { getProjectResourceFile, getProjectResourceFiles, revealProjectFileInFinder, type ApiError, type ResourceCatalog, type ResourceEntry } from "../../api/projectFiles";
import { FileViewerView, useFileReader } from "../../components/chat/FileViewer";
import { MenuPortalContext, PointerContextMenu, type PointerMenuState } from "../../components/ui/PointerContextMenu";
import type { FileLink } from "../../components/chat/renderers/filePath";
import { copyText } from "../../lib/copyText";
import { isHtmlPath } from "../../lib/linkActions";
import { openProjectFileTab } from "../../lib/projectFileTab";
import { useRevealSupported } from "../../lib/revealInFinder";
import { useUiStore } from "../../store/uiStore";

// The project Files dialog owns browse and preview for one project's shared
// folder (FS-11.R11–R16, TS-08.R172/R173). List context lives only here.

const TYPE_LABELS: Record<string, string> = {
  md: "Markdown", markdown: "Markdown", ts: "TS", tsx: "TSX", js: "JS", jsx: "JSX", json: "JSON",
  csv: "CSV", html: "HTML", htm: "HTML", txt: "Text", py: "Python", go: "Go", sh: "Shell",
  yaml: "YAML", yml: "YAML", png: "PNG", jpg: "JPEG", jpeg: "JPEG", gif: "GIF", webp: "WebP", svg: "SVG",
};
const NO_EXTENSION = "";

/** The type key of a name: its lower-case extension, or "" (a leading dot alone is not one). */
export function fileType(name: string): string {
  const dot = name.lastIndexOf(".");
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : NO_EXTENSION;
}
const typeLabel = (type: string) => type === NO_EXTENSION ? "No extension" : TYPE_LABELS[type] ?? type.toUpperCase();
const parentOf = (path: string) => path.includes("/") ? path.slice(0, path.lastIndexOf("/")) : "";
const inFolder = (path: string, folder: string) => folder === "" || path.startsWith(`${folder}/`);
const formatTime = (value: string | null) => value ? new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" }) : "—";

type CatalogState =
  | { status: "loading" }
  | { status: "error"; error: ApiError }
  | { status: "loaded"; catalog: ResourceCatalog };

const errorText = (error: ApiError) => error.code === "resource_scan_busy"
  ? "Other file lists are being read right now. Try again in a moment."
  : error.message || "The file list could not be read.";

export function ProjectFilesDialog({ projectId, title, resourceDir, onClose }: { projectId: string; title: string; resourceDir: string; onClose: () => void }) {
  const pushError = useUiStore((state) => state.pushError);
  const revealSupported = useRevealSupported();
  const [content, setContent] = useState<HTMLDivElement | null>(null);
  const [state, setState] = useState<CatalogState>({ status: "loading" });
  const [folder, setFolder] = useState("");
  const [query, setQuery] = useState("");
  const [types, setTypes] = useState<string[]>([]);
  const [preview, setPreview] = useState<FileLink | null>(null);
  const [menu, setMenu] = useState<PointerMenuState | null>(null);
  const [lastRow, setLastRow] = useState<string | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const scrollRef = useRef<HTMLDivElement>(null);
  const savedScroll = useRef(0);
  const request = useRef(0);

  // One catalog per open or Refresh. A response is fenced by request number, so
  // a superseded or post-close read never replaces the list (TS-08.R172). A
  // broad catalog that hit a limit lets a chosen folder read its own catalog.
  const load = useCallback((scope: string) => {
    const id = ++request.current;
    setState({ status: "loading" });
    getProjectResourceFiles(projectId, scope).then(
      (catalog) => { if (request.current === id) setState({ status: "loaded", catalog }); },
      (error: ApiError) => { if (request.current === id) setState({ status: "error", error }); },
    );
  }, [projectId]);
  useEffect(() => { load(""); return () => { request.current++; }; }, [load]);

  const catalog = state.status === "loaded" ? state.catalog : null;
  const scopedLoad = catalog !== null && catalog.folder !== "" && catalog.folder === folder;
  const folders = useMemo(() => (catalog?.entries ?? []).filter((entry) => entry.kind === "directory").map((entry) => entry.path).sort(), [catalog]);
  const scoped = useMemo(() => (catalog?.entries ?? []).filter((entry) => entry.kind !== "directory" && inFolder(entry.path, folder)), [catalog, folder]);
  const knownTypes = useMemo(() => [...new Set(scoped.map((entry) => fileType(entry.name)))].sort((a, b) => typeLabel(a).localeCompare(typeLabel(b))), [scoped]);
  const needle = query.trim().toLowerCase();
  const rows = useMemo(() => scoped.filter((entry) =>
    (types.length === 0 || types.includes(fileType(entry.name)))
    && (needle === "" || entry.path.toLowerCase().includes(needle)),
  ), [scoped, types, needle]);
  const folderMissing = folder !== "" && catalog !== null && !scopedLoad && !folders.includes(folder);

  const chooseFolder = (next: string) => {
    setFolder(next);
    if (catalog?.partial && next !== "") load(next);
    else if (scopedLoad) load("");
  };
  const refresh = () => load(scopedLoad ? folder : "");
  const showAll = () => { setFolder(""); load(""); };

  const openPreview = (entry: ResourceEntry) => {
    if (entry.kind !== "file") return;
    savedScroll.current = scrollRef.current?.scrollTop ?? 0;
    setLastRow(entry.path);
    setMenu(null);
    setPreview({ path: entry.path });
  };
  const backToFiles = () => setPreview(null);
  // Back to files restores scroll and focus to the selected row, or to search
  // when that row has disappeared (FS-11.R13).
  useEffect(() => {
    if (preview) return;
    if (scrollRef.current) scrollRef.current.scrollTop = savedScroll.current;
    const row = lastRow ? content?.querySelector<HTMLElement>(`tr[data-path="${CSS.escape(lastRow)}"]`) : null;
    (row ?? searchRef.current)?.focus();
  }, [preview]); // eslint-disable-line react-hooks/exhaustive-deps

  const actionsFor = (entry: ResourceEntry): PointerMenuState["actions"] => [
    { label: "Copy path", select: () => void copyText(`${resourceDir}/${entry.path}`, pushError) },
    ...(revealSupported && entry.kind === "file" ? [{ label: "Open in Finder", select: () => {
      void revealProjectFileInFinder(projectId, entry.path).catch((error: unknown) => pushError("Could not open in Finder", error instanceof Error ? error.message : "Finder could not reveal that file"));
    } }] : []),
    ...(entry.kind === "file" && isHtmlPath(entry.name) ? [{ label: "Open in new tab", select: () => openProjectFileTab(projectId, entry.path) }] : []),
  ];
  const openMenu = (entry: ResourceEntry, x: number, y: number) => setMenu({ x, y, actions: actionsFor(entry) });
  const rowKey = (event: KeyboardEvent<HTMLTableRowElement>, entry: ResourceEntry) => {
    if (event.key === "Enter" || event.key === " ") { event.preventDefault(); openPreview(entry); }
    else if (event.key === "ContextMenu" || (event.shiftKey && event.key === "F10")) {
      event.preventDefault();
      const box = event.currentTarget.getBoundingClientRect();
      openMenu(entry, box.left + 24, box.bottom);
    }
  };
  const searchKey = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Backspace" && query === "" && types.length > 0) setTypes(types.slice(0, -1));
  };

  return (
    <Dialog.Root open onOpenChange={(open) => { if (!open) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
        <Dialog.Content
          ref={setContent}
          className="dialog-content project-files"
          data-ui="dialog"
          data-slot="content"
          data-variant="default"
          aria-describedby={undefined}
          onOpenAutoFocus={(event) => { event.preventDefault(); searchRef.current?.focus(); }}
          onEscapeKeyDown={(event) => { if (menu) event.preventDefault(); }}
        >
          <header className="project-files-header">
            <Dialog.Title data-slot="title">Files · {title}</Dialog.Title>
            <Dialog.Close asChild><button type="button" className="ad-button-secondary">Close</button></Dialog.Close>
          </header>
          <MenuPortalContext.Provider value={content}>
            {preview && <ProjectFilePreview projectId={projectId} link={preview} onBack={backToFiles} onOpenFile={setPreview} />}
            <div className="project-files-browser" hidden={preview !== null}>
              <div className="project-files-controls">
                <label className="project-files-folder">
                  <span className="ad-visually-hidden">Folder</span>
                  <select value={folder} onChange={(event) => chooseFolder(event.target.value)}>
                    <option value="">All folders</option>
                    {folder !== "" && !folders.includes(folder) && <option value={folder}>{folder}</option>}
                    {folders.map((path) => <option key={path} value={path}>{path}</option>)}
                  </select>
                </label>
                <div className="project-files-search" onClick={() => searchRef.current?.focus()}>
                  {types.map((type) => (
                    <button key={type} type="button" className="project-files-badge" aria-label={`Remove ${typeLabel(type)} filter`} onClick={(event) => { event.stopPropagation(); setTypes(types.filter((t) => t !== type)); searchRef.current?.focus(); }}>
                      {typeLabel(type)} <span aria-hidden="true">×</span>
                    </button>
                  ))}
                  <input ref={searchRef} type="search" aria-label="Search files" placeholder="Search files" value={query} onChange={(event) => setQuery(event.target.value)} onKeyDown={searchKey} />
                  <select aria-label="Add a file type filter" value="__" onChange={(event) => { if (event.target.value !== "__") setTypes([...types, event.target.value]); }}>
                    <option value="__">Type</option>
                    {knownTypes.filter((type) => !types.includes(type)).map((type) => <option key={type || "none"} value={type}>{typeLabel(type)}</option>)}
                  </select>
                </div>
                <button type="button" className="ad-button-secondary" onClick={refresh}>Refresh</button>
              </div>
              <CatalogBody
                state={state}
                folderMissing={folderMissing}
                filtered={needle !== "" || types.length > 0}
                folder={folder}
                rows={rows}
                scrollRef={scrollRef}
                onRetry={refresh}
                onShowAll={showAll}
                onOpen={openPreview}
                onMenu={(entry, event) => { event.preventDefault(); openMenu(entry, event.clientX, event.clientY); }}
                onRowKey={rowKey}
              />
            </div>
            <PointerContextMenu menu={menu} onClose={() => setMenu(null)} />
          </MenuPortalContext.Provider>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}

function CatalogBody({ state, folderMissing, filtered, folder, rows, scrollRef, onRetry, onShowAll, onOpen, onMenu, onRowKey }: {
  state: CatalogState;
  folderMissing: boolean;
  filtered: boolean;
  folder: string;
  rows: ResourceEntry[];
  scrollRef: React.RefObject<HTMLDivElement>;
  onRetry: () => void;
  onShowAll: () => void;
  onOpen: (entry: ResourceEntry) => void;
  onMenu: (entry: ResourceEntry, event: MouseEvent) => void;
  onRowKey: (event: KeyboardEvent<HTMLTableRowElement>, entry: ResourceEntry) => void;
}) {
  if (state.status === "loading") return <p className="project-files-state" role="status">Reading files…</p>;
  if (state.status === "error" && state.error.code === "not_found" && folder !== "") return <FolderGone onShowAll={onShowAll} />;
  if (state.status === "error") {
    return (
      <div className="project-files-state" role="alert">
        <p>{errorText(state.error)}</p>
        <button type="button" onClick={onRetry}>Retry</button>
      </div>
    );
  }
  if (folderMissing) return <FolderGone onShowAll={onShowAll} />;
  const { catalog } = state;
  const empty = rows.length === 0;
  return (
    <>
      {catalog.partial && (
        <p className="project-files-note" role="status">
          {catalog.partial_reason === "limit"
            ? "This list is incomplete: the folder is too large to read at once. Choose a folder to narrow it."
            : "This list is incomplete: some entries could not be read."}
        </p>
      )}
      {empty ? (
        <p className="project-files-state" role="status">
          {filtered ? "No files match this search." : folder !== "" ? "This folder is empty." : "No shared files yet. Files agents save to this project's shared folder appear here."}
        </p>
      ) : (
        <div className="project-files-table-wrap" ref={scrollRef}>
          <table className="project-files-table">
            <thead>
              <tr><th scope="col">Name</th><th scope="col">Path</th><th scope="col">Last modified</th></tr>
            </thead>
            <tbody>
              {rows.map((entry) => (
                <tr
                  key={entry.path}
                  data-path={entry.path}
                  data-kind={entry.kind}
                  tabIndex={0}
                  aria-disabled={entry.kind !== "file" || undefined}
                  title={entry.kind === "file" ? entry.path : `${entry.path} — ${entry.kind === "symlink" ? "links are not followed" : "special entries cannot be previewed"}`}
                  onClick={() => onOpen(entry)}
                  onContextMenu={(event) => onMenu(entry, event)}
                  onKeyDown={(event) => onRowKey(event, entry)}
                >
                  <td className="project-files-name">
                    {entry.name}
                    {entry.kind !== "file" && <span className="project-files-kind">{entry.kind === "symlink" ? "link · unavailable" : "unavailable"}</span>}
                  </td>
                  <td className="project-files-path">{parentOf(entry.path) || "—"}</td>
                  <td className="project-files-time">{formatTime(entry.mod_time)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </>
  );
}

function FolderGone({ onShowAll }: { onShowAll: () => void }) {
  return (
    <div className="project-files-state" role="alert">
      <p>That folder no longer exists. It may have been moved or removed.</p>
      <button type="button" onClick={onShowAll}>Show all folders</button>
    </div>
  );
}

// The preview reuses the shared viewer with the project loader and no editing
// or annotation callbacks (TS-08.R173). Relative links resolve from the
// previewed file through the same loader; the server refuses any escape.
function ProjectFilePreview({ projectId, link, onBack, onOpenFile }: { projectId: string; link: FileLink; onBack: () => void; onOpenFile: (link: FileLink) => void }) {
  const loader = useCallback((path: string) => getProjectResourceFile(projectId, path), [projectId]);
  const reader = useFileReader({ agentId: "", link, load: loader });
  return (
    <div className="project-files-preview">
      <FileViewerView reader={reader} variant="focused" onClose={onBack} closeLabel="Back to files" onOpenFile={onOpenFile} />
    </div>
  );
}
