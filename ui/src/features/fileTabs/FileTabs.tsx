import { useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { DndContext, MouseSensor, TouchSensor, closestCenter, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { SortableContext, horizontalListSortingStrategy, useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { FileContent } from "../../api/types";
import { useFileReader, type FileReader } from "../../components/chat/FileViewer";
import { CloseIcon, MoveLeftIcon, MoveRightIcon } from "../../components/ui/icons";
import type { FileTab } from "./model";
import { createReadLimiter, type ReadLimiter } from "./readLimiter";
import { fileTabs, flushFileTabs, useFileTabsStore } from "./store";

// The tabbed reader every surface shares (FS-03.R101–R102, TS-08.R142/R145).
// One host per tab visited while this owner is mounted keeps that tab's read,
// display mode and position; only the selected host renders. Restored tabs have
// no host, and so no read, until they are selected.

export type TabReaderProps = {
  tabs: FileTab[];
  active: FileTab | null;
  jumps: Record<string, number>;
  onSelect: (id: string) => void;
  onClose: (id: string) => void;
  onMove: (id: string, to: number) => void;
  onResolved?: (id: string, path: string) => void;
  /** The read context a tab's file comes from: the agent, or the room source. */
  contextFor: (tab: FileTab) => string;
  loadFor?: (tab: FileTab) => ((path: string) => Promise<FileContent>) | undefined;
  /** Names a tab's room source where equal file names need telling apart. */
  sourceLabel?: (tab: FileTab) => string;
  /** Called after the final tab closes, to return focus to its surface. */
  onEmptied?: () => void;
  renderView: (reader: FileReader, strip: ReactNode, tab: FileTab) => ReactNode;
  /** Where the selected file's view is placed: inline when omitted, nowhere while null. */
  target?: HTMLElement | null;
};

export function FileTabsReader(props: TabReaderProps) {
  const { tabs, active, target } = props;
  const [limiter] = useState(createReadLimiter);
  const [visited, setVisited] = useState<string[]>([]);
  const open = new Set(tabs.map((tab) => tab.id));
  const hosted = visited.filter((id) => open.has(id));
  if (active && !hosted.includes(active.id)) hosted.push(active.id);
  if (hosted.length !== visited.length || hosted.some((id, i) => visited[i] !== id)) setVisited(hosted);

  useEffect(() => { limiter.select(active?.id ?? null); }, [limiter, active?.id]);
  // Switching tabs saves the leaving tab's place; leaving saves everything.
  useEffect(() => { flushFileTabs(); }, [active?.id]);
  useEffect(() => () => { limiter.clear(); flushFileTabs(); }, [limiter]);

  const restoreTabFocus = useRef(false);
  const close = (id: string) => {
    restoreTabFocus.current = true;
    const last = tabs.length === 1;
    props.onClose(id);
    if (last) props.onEmptied?.();
  };
  const strip = <TabStrip {...props} onClose={close} restoreTabFocus={restoreTabFocus} />;
  const byId = new Map(tabs.map((tab) => [tab.id, tab]));
  return (
    <>
      {hosted.map((id) => (
        <TabHost key={id} tab={byId.get(id)!} selected={active?.id === id} props={props} limiter={limiter} strip={strip} target={target} />
      ))}
    </>
  );
}

function TabHost({ tab, selected, props, limiter, strip, target }: {
  tab: FileTab;
  selected: boolean;
  props: TabReaderProps;
  limiter: ReadLimiter;
  strip: ReactNode;
  target?: HTMLElement | null;
}) {
  const [initialView] = useState(() => fileTabs.view(tab.id));
  const reader = useFileReader({
    agentId: props.contextFor(tab),
    link: tab.line ? { path: tab.path, line: tab.line } : { path: tab.path },
    load: props.loadFor?.(tab),
    onPathResolved: props.onResolved ? (path) => props.onResolved?.(tab.id, path) : undefined,
    initialView,
    jump: props.jumps[tab.id] ?? 0,
    onViewChange: (view) => fileTabs.setView(tab.id, view),
    schedule: (kind, start) => limiter.run(tab.id, kind, start),
  });
  const { dropped, reload } = reader;
  useEffect(() => { if (selected && dropped) reload(); }, [selected, dropped, reload]);
  useEffect(() => () => limiter.cancel(tab.id), [limiter, tab.id]);
  // A target that has not mounted yet (Focus file opening) shows nothing for a frame.
  if (!selected || target === null) return null;
  const view = props.renderView(reader, strip, tab);
  return target ? createPortal(view, target) : <>{view}</>;
}

const reducedMotion = () => typeof window.matchMedia === "function" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;
const fileName = (path: string) => path.split("/").filter(Boolean).pop() ?? path;

function TabStrip({ tabs, active, onSelect, onClose, onMove, sourceLabel, restoreTabFocus }: TabReaderProps & { restoreTabFocus: { current: boolean } }) {
  const notice = useFileTabsStore((state) => state.notice);
  const list = useRef<HTMLDivElement>(null);
  const [focused, setFocused] = useState<string | null>(null);
  const dragged = useRef(false);
  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 6 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 250, tolerance: 8 } }),
  );
  const activeIndex = tabs.findIndex((tab) => tab.id === active?.id);
  const names = tabs.map((tab) => fileName(tab.path));
  const button = (id: string) => [...(list.current?.querySelectorAll<HTMLElement>("[data-tab-id]") ?? [])]
    .find((tab) => tab.dataset.tabId === id)?.querySelector<HTMLButtonElement>("[role=\"tab\"]");

  // The selected tab, close control included, stays in view — again after the
  // strip's first layout settles — and a close hands focus to its neighbour.
  useLayoutEffect(() => {
    if (!active) return undefined;
    const tab = button(active.id);
    const reveal = () => tab?.closest(".file-tab")?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
    reveal();
    if (restoreTabFocus.current) {
      restoreTabFocus.current = false;
      tab?.focus({ preventScroll: true });
    }
    const frame = requestAnimationFrame(reveal);
    return () => cancelAnimationFrame(frame);
  }, [active?.id, tabs.length]);

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    const to = event.key === "ArrowRight" ? index + 1 : event.key === "ArrowLeft" ? index - 1
      : event.key === "Home" ? 0 : event.key === "End" ? tabs.length - 1 : null;
    if (event.key === "Delete") {
      event.preventDefault();
      onClose(tabs[index].id);
      return;
    }
    if (to === null) return;
    event.preventDefault();
    const tab = tabs[Math.max(0, Math.min(tabs.length - 1, to))];
    setFocused(tab.id);
    button(tab.id)?.focus();
  };

  const onDragEnd = ({ active: moved, over }: DragEndEvent) => {
    requestAnimationFrame(() => { dragged.current = false; });
    if (!over || moved.id === over.id) return;
    onMove(String(moved.id), tabs.findIndex((tab) => tab.id === over.id));
  };

  const roving = focused && tabs.some((tab) => tab.id === focused) ? focused : active?.id;
  return (
    <div className="file-tabs" data-ui="file-tabs">
      <div className="file-tabs-row">
        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={() => { dragged.current = true; }} onDragEnd={onDragEnd} onDragCancel={() => { dragged.current = false; }}>
          <SortableContext items={tabs.map((tab) => tab.id)} strategy={horizontalListSortingStrategy}>
            <div className="file-tabs-list" role="tablist" aria-label="Open files" ref={list}>
              {tabs.map((tab, index) => {
                const source = sourceLabel && names.filter((name) => name === names[index]).length > 1 ? sourceLabel(tab) : "";
                return (
                  <SortableTab
                    key={tab.id}
                    tab={tab}
                    name={names[index]}
                    source={source}
                    title={[tab.path, sourceLabel?.(tab)].filter(Boolean).join(" — ")}
                    selected={tab.id === active?.id}
                    tabbable={tab.id === roving}
                    onSelect={() => {
                      if (dragged.current) return;
                      if (tab.id !== active?.id && document.activeElement?.getAttribute("role") === "tab") restoreTabFocus.current = true;
                      onSelect(tab.id);
                    }}
                    onClose={() => onClose(tab.id)}
                    onKeyDown={(event) => onKeyDown(event, index)}
                  />
                );
              })}
            </div>
          </SortableContext>
        </DndContext>
        {tabs.length > 1 && active && (
          <div className="file-tabs-move" role="group" aria-label="Move tab">
            <button type="button" className="file-tabs-icon" aria-label={`Move ${names[activeIndex]} left`} title="Move tab left" disabled={activeIndex <= 0} onClick={() => onMove(active.id, activeIndex - 1)}><MoveLeftIcon /></button>
            <button type="button" className="file-tabs-icon" aria-label={`Move ${names[activeIndex]} right`} title="Move tab right" disabled={activeIndex >= tabs.length - 1} onClick={() => onMove(active.id, activeIndex + 1)}><MoveRightIcon /></button>
          </div>
        )}
      </div>
      {notice && (
        <p className="file-tabs-notice" role="status">
          <span>{notice}</span>
          <button type="button" className="ad-button-secondary" onClick={fileTabs.dismissNotice}>Dismiss</button>
        </p>
      )}
    </div>
  );
}

function SortableTab({ tab, name, source, title, selected, tabbable, onSelect, onClose, onKeyDown }: {
  tab: FileTab;
  name: string;
  source: string;
  title: string;
  selected: boolean;
  tabbable: boolean;
  onSelect: () => void;
  onClose: () => void;
  onKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => void;
}) {
  const { setNodeRef, listeners, transform, transition, isDragging } = useSortable({ id: tab.id });
  const label = source ? `${name} (${source})` : name;
  return (
    <div
      ref={setNodeRef}
      className="file-tab"
      data-tab-id={tab.id}
      data-state={selected ? "active" : undefined}
      data-dragging={isDragging ? "true" : undefined}
      // Neighbours slide aside during a drag, unless reduced motion is asked for.
      style={{ transform: CSS.Translate.toString(transform), transition: reducedMotion() ? undefined : transition }}
      {...listeners}
    >
      <button type="button" role="tab" className="file-tab-select" aria-selected={selected} tabIndex={tabbable ? 0 : -1} title={title} onClick={onSelect} onKeyDown={onKeyDown}>
        <span className="file-tab-name">{name}</span>
        {source && <span className="file-tab-source">{source}</span>}
      </button>
      <button type="button" className="file-tab-close" aria-label={`Close ${label}`} title={`Close ${label}`} tabIndex={-1} onClick={onClose}><CloseIcon /></button>
    </div>
  );
}
