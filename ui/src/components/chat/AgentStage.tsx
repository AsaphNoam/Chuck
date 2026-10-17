import { useCallback, useEffect, useId, useLayoutEffect, useRef, useState, type CSSProperties, type KeyboardEvent, type PointerEvent, type ReactNode } from "react";
import { allocateReader, fileShareFor, type Range, type ReaderRequest, type TrayPresence } from "./readerLayout";

/** What the transcript currently shows beside the card. */
export type ReaderOccupancy = { file: boolean; tray: TrayPresence };

/** Elements beside the full agent page's card that host the file viewer (start)
 *  and annotation tray (end) outside it (FS-12.R61), and the report that lets
 *  the stage allocate their widths. Without them both stay in the transcript's
 *  own grid, as the dashboard pane needs. */
export type PanelSlots = {
  start: HTMLElement | null;
  end: HTMLElement | null;
  report: (occupancy: ReaderOccupancy) => void;
  /** No room to dock: the tray keeps its existing overlay inside the
   *  transcript, over the conversation and never over the file. */
  trayOverlay: boolean;
};

const STEP = 16;
const LARGE_STEP = 64;

// The card between two slots that host the file viewer and annotation tray
// outside it (FS-12.R61). The stage owns the reader workspace's proportions:
// the person's requested split, kept apart from the widths that fit right now,
// and the two dividers that change it (FS-03.R93/R96, TS-08.R128). The request
// lives as long as this page does and is never stored. The archived page
// shares it.
export function AgentStage({ onSlots, children }: { onSlots: (slots: PanelSlots) => void; children: ReactNode }) {
  const stageRef = useRef<HTMLDivElement>(null);
  const [start, setStart] = useState<HTMLDivElement | null>(null);
  const [end, setEnd] = useState<HTMLDivElement | null>(null);
  const [occupancy, setOccupancy] = useState<ReaderOccupancy>({ file: false, tray: "none" });
  const [request, setRequest] = useState<ReaderRequest>({ fileShare: null, trayWidth: null });
  const [space, setSpace] = useState<{ stage: number; available: number } | null>(null);
  const spaceRef = useRef(space);
  const ids = useId();
  const fileId = `${ids}-file`;
  const trayId = `${ids}-tray`;

  const report = useCallback((next: ReaderOccupancy) => {
    setOccupancy((current) => current.file === next.file && current.tray === next.tray ? current : next);
  }, []);

  // Available width is the stage less its dividers and any collapsed strip;
  // one observer on the stage (and the strip) is the only measurement.
  const measure = useCallback(() => {
    const stage = stageRef.current;
    if (!stage) return;
    let fixed = 0;
    for (const divider of stage.querySelectorAll<HTMLElement>(":scope > .workspace-divider")) fixed += divider.getBoundingClientRect().width;
    if (stage.dataset.tray === "collapsed" && end) fixed += end.getBoundingClientRect().width;
    const stageWidth = stage.clientWidth;
    const available = Math.floor(stageWidth - fixed);
    // Compared outside the updater: this runs after every commit, and only a
    // real change may schedule another render.
    const current = spaceRef.current;
    if (current && current.stage === stageWidth && current.available === available) return;
    spaceRef.current = { stage: stageWidth, available };
    setSpace(spaceRef.current);
  }, [end]);

  useLayoutEffect(measure);
  useEffect(() => {
    const stage = stageRef.current;
    if (!stage || typeof ResizeObserver === "undefined") return undefined;
    const observer = new ResizeObserver(measure);
    observer.observe(stage);
    if (end) observer.observe(end);
    return () => observer.disconnect();
  }, [measure, end]);

  const active = occupancy.file || occupancy.tray !== "none";
  const allocation = active && space ? allocateReader({ available: space.available, stage: space.stage, file: occupancy.file, tray: occupancy.tray, request }) : null;
  const tray = allocation?.tray ?? occupancy.tray;
  const fileRange = occupancy.file ? allocation?.file : null;
  const trayRange = allocation?.tray === "docked" ? allocation.trayWidth : null;
  const trayOverlay = tray === "overlay";
  useEffect(() => onSlots({ start, end, report, trayOverlay }), [start, end, report, trayOverlay, onSlots]);
  // Only measured allocation values are set inline; all static presentation
  // stays in the stylesheet (TS-08.R128).
  const style = allocation ? {
    "--ad-reader-file-width": allocation.file ? `${allocation.file.now}px` : undefined,
    "--ad-reader-tray-width": allocation.trayWidth ? `${allocation.trayWidth.now}px` : undefined,
  } as CSSProperties : undefined;

  return (
    <div className="chat-stage" ref={stageRef} style={style} data-file={occupancy.file ? "open" : undefined} data-tray={tray === "none" ? undefined : tray}>
      <div className="chat-side chat-side-start" id={fileId} ref={setStart} />
      {allocation && fileRange && (
        <WorkspaceDivider
          variant="file"
          label="Resize file and conversation"
          controls={fileId}
          range={fileRange}
          direction={1}
          onResize={(width) => setRequest((current) => ({ ...current, fileShare: fileShareFor(width, allocation) }))}
          onReset={() => setRequest((current) => ({ ...current, fileShare: null }))}
        />
      )}
      {children}
      {trayRange && (
        <WorkspaceDivider
          variant="annotations"
          label="Resize annotations and conversation"
          controls={trayId}
          range={trayRange}
          direction={-1}
          onResize={(width) => setRequest((current) => ({ ...current, trayWidth: clampTo(width, trayRange) }))}
          onReset={() => setRequest((current) => ({ ...current, trayWidth: null }))}
        />
      )}
      <div className="chat-side chat-side-end" id={trayId} ref={setEnd} />
    </div>
  );
}

function clampTo(width: number, range: Range) {
  return Math.max(range.min, Math.min(width, range.max));
}

// A focusable vertical separator. Left/Right move it in that physical
// direction, Shift moves further, Home or a double-click restores the initial
// split. direction says whether moving right grows (1) or shrinks (-1) the
// pane it controls.
function WorkspaceDivider({ variant, label, controls, range, direction, onResize, onReset }: {
  variant: "file" | "annotations";
  label: string;
  controls: string;
  range: Range;
  direction: 1 | -1;
  onResize: (width: number) => void;
  onReset: () => void;
}) {
  const drag = useRef<{ x: number; width: number } | null>(null);
  const onKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key === "Home") {
      event.preventDefault();
      onReset();
      return;
    }
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    const step = (event.shiftKey ? LARGE_STEP : STEP) * (event.key === "ArrowRight" ? 1 : -1) * direction;
    onResize(range.now + step);
  };
  const onPointerDown = (event: PointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    drag.current = { x: event.clientX, width: range.now };
  };
  const onPointerMove = (event: PointerEvent<HTMLDivElement>) => {
    if (!drag.current) return;
    onResize(drag.current.width + (event.clientX - drag.current.x) * direction);
  };
  const endDrag = (event: PointerEvent<HTMLDivElement>) => {
    drag.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
  };
  return (
    <div
      className="workspace-divider"
      data-ui="workspace-divider"
      data-variant={variant}
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-controls={controls}
      aria-valuenow={range.now}
      aria-valuemin={range.min}
      aria-valuemax={range.max}
      aria-valuetext={`${range.now} pixels wide`}
      tabIndex={0}
      title={`${label}. Double-click to reset.`}
      onKeyDown={onKeyDown}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={endDrag}
      onPointerCancel={endDrag}
      onLostPointerCapture={() => { drag.current = null; }}
      onDoubleClick={onReset}
    />
  );
}
