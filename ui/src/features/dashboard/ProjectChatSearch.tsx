import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { searchArchiveProject } from "../../api/client";
import type { ArchiveResult } from "../../api/types";

export function searchResultPath(row: ArchiveResult) {
  const path = `/${row.archived ? "archive" : "agent"}/${encodeURIComponent(row.agent_id)}`;
  if (!row.search_ref || row.location_status !== "reference") return path;
  const ref = row.search_ref;
  return `${path}?${new URLSearchParams({ search_doc: String(ref.document_id), search_rev: ref.content_sha256, search_start: String(ref.start_byte), search_end: String(ref.end_byte) })}`;
}

// Keyed by project at the owning heading: no prior project's result can survive navigation.
export function ProjectChatSearch({ project }: { project: string }) {
  const navigate = useNavigate();
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [rows, setRows] = useState<ArchiveResult[]>([]);
  const [total, setTotal] = useState(0);
  const [mode, setMode] = useState("full_text");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState(0);
  const [attempt, setAttempt] = useState(0);
  const request = useRef<AbortController | null>(null);
  const generation = useRef(0);
  const input = useRef<HTMLInputElement>(null);
  const popup = useRef<HTMLDivElement>(null);
  useLayoutEffect(() => {
    popup.current?.querySelector<HTMLElement>(`#project-chat-result-${selected}`)?.scrollIntoView?.({ block: "nearest" });
  }, [open, selected, rows]);
  const load = async (append = false) => {
    request.current?.abort();
    const controller = new AbortController();
    request.current = controller;
    const token = ++generation.current;
    setLoading(true);
    setError("");
    try {
      let page = await searchArchiveProject(project, query.trim(), 20, append ? rows.length : 0, controller.signal, "project_search");
      let next = page.results ?? [];
      if (append) {
        // A small boundary probe detects ranking/lifecycle shifts even when the new page has no duplicates.
        const boundary = await searchArchiveProject(project, query.trim(), 1, Math.max(0, rows.length - 1), controller.signal, "project_search");
        const known = new Set(rows.map((row) => row.agent_id));
        const boundaryChanged = boundary.results?.[0]?.agent_id !== rows[rows.length - 1]?.agent_id;
        const incoming = next;
        if (boundaryChanged || incoming.some((row) => known.has(row.agent_id))) {
          page = await searchArchiveProject(project, query.trim(), 20, 0, controller.signal, "project_search");
          next = page.results ?? [];
        } else next = [...rows, ...incoming];
      }
      if (controller.signal.aborted || token !== generation.current) return;
      setRows(next);
      setTotal(page.total);
      setMode(page.search_mode);
      if (!append) setSelected(0);
    } catch (cause) {
      if (!controller.signal.aborted && token === generation.current) setError(cause instanceof Error ? cause.message : "Search failed");
    } finally {
      if (!controller.signal.aborted && token === generation.current) setLoading(false);
    }
  };
  useEffect(() => {
    request.current?.abort();
    ++generation.current;
    setRows([]);
    setError("");
    setLoading(Boolean(query.trim()));
    if (!query.trim()) return;
    const timer = setTimeout(() => void load(), 250);
    return () => { clearTimeout(timer); request.current?.abort(); ++generation.current; };
  }, [query, project, attempt]);
  const choose = (row: ArchiveResult) => navigate(searchResultPath(row));
  return <div className="project-chat-search" onBlur={(event) => { if (!event.currentTarget.contains(event.relatedTarget)) setOpen(false); }}>
    <input ref={input} type="search" aria-label="Search chats" placeholder="Search chats…" value={query}
      role="combobox" aria-expanded={open && Boolean(query.trim())} aria-controls="project-chat-results" aria-autocomplete="list"
      aria-activedescendant={open && rows[selected] ? `project-chat-result-${selected}` : undefined}
      onFocus={() => setOpen(true)} onChange={(event) => { setQuery(event.target.value); setOpen(true); }}
      onKeyDown={(event) => {
        if (event.key === "Escape") { setOpen(false); event.stopPropagation(); }
        if (event.key === "ArrowDown" || event.key === "ArrowUp") {
          event.preventDefault(); setOpen(true);
          setSelected((value) => rows.length ? (value + (event.key === "ArrowDown" ? 1 : -1) + rows.length) % rows.length : 0);
        }
        if (event.key === "Enter" && open && rows[selected]) { event.preventDefault(); choose(rows[selected]); }
      }} />
    {open && query.trim() && <div ref={popup} className="project-chat-popup">
      <p className="form-info">{mode === "metadata" ? "Session metadata only; transcript search unavailable in this build." : "Completed turns and saved annotations. All terms match within one turn or metadata record."}</p>
      {loading && <p role="status">Searching…</p>}
      {error && <p role="alert">{error} <button type="button" onClick={() => setAttempt((value) => value + 1)}>Retry</button></p>}
      {!loading && !error && !rows.length && <p role="status">No matching chats</p>}
      <ul id="project-chat-results" role="listbox" aria-label="Matching chats">
        {rows.map((row, index) => <li key={row.agent_id} id={`project-chat-result-${index}`} role="option" aria-selected={selected === index}>
          <button type="button" className="project-chat-result" onFocus={() => setSelected(index)} onClick={() => choose(row)}>
            <strong>{row.name}</strong><span>{row.archived ? "Archived" : row.active ? "Running" : "Stopped"}</span>
            {row.snippet && <span className="project-chat-excerpt">{row.snippet}</span>}
            {row.location_status === "metadata_only" && <span className="project-chat-excerpt">Metadata match</span>}
            {row.location_status === "unavailable" && <span className="project-chat-excerpt">Matching passage unavailable · Open chat</span>}
          </button>
        </li>)}
      </ul>
      {rows.length < total && <button type="button" disabled={loading} onClick={() => void load(true)}>Load more</button>}
    </div>}
  </div>;
}
