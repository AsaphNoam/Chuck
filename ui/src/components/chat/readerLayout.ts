// Width allocation for the full agent page's reader workspace: the open file,
// the conversation card and the pending-annotation tray (FS-03.R93, TS-08.R128).
// Requested values are the person's choices and are never overwritten by a
// clamp; the allocation is what fits the space available right now.

export const MIN_FILE = 320;
export const MIN_CHAT = 360;
export const MIN_TRAY = 280;
const TRAY_SHARE = 0.3;
const TRAY_CAP = 460;

export type TrayPresence = "none" | "expanded" | "collapsed";

/** Requested file share of the space beside a docked tray, and requested tray
 *  width in pixels; null means the initial split. */
export type ReaderRequest = { fileShare: number | null; trayWidth: number | null };

export type Range = { now: number; min: number; max: number };

export type ReaderAllocation = {
  /** "docked" sits beside the conversation; "overlay" floats over it. */
  tray: "none" | "docked" | "overlay" | "collapsed";
  file: Range | null;
  trayWidth: Range | null;
  chat: number;
};

const clamp = (value: number, min: number, max: number) => Math.max(min, Math.min(value, max));

/**
 * allocateReader divides `available` pixels (the stage minus its dividers and
 * any collapsed strip, which `fixed` names) between file, conversation and tray.
 * An expanded tray docks only when every visible surface keeps its useful
 * minimum; otherwise it overlays the conversation and leaves the file alone.
 */
export function allocateReader({ available, stage, file, tray, request }: {
  available: number;
  /** Full stage width, the base of the tray's initial min(30%, 460px). */
  stage: number;
  file: boolean;
  tray: TrayPresence;
  request: ReaderRequest;
}): ReaderAllocation {
  const space = Math.max(0, available);
  const fileMin = file ? MIN_FILE : 0;
  const trayMode: ReaderAllocation["tray"] = tray === "expanded"
    ? space >= fileMin + MIN_CHAT + MIN_TRAY ? "docked" : "overlay"
    : tray;
  let trayWidth: Range | null = null;
  let rest = space;
  if (trayMode === "docked") {
    const max = space - fileMin - MIN_CHAT;
    const now = Math.round(clamp(request.trayWidth ?? Math.min(stage * TRAY_SHARE, TRAY_CAP), MIN_TRAY, max));
    trayWidth = { now, min: MIN_TRAY, max: Math.round(max) };
    rest = space - now;
  }
  if (!file) return { tray: trayMode, file: null, trayWidth, chat: Math.round(rest) };
  // Below both reading minimums the split stays adjacent, scaling both
  // minimums down together; Focus file is the escape to a wider document.
  const scale = Math.min(1, rest / (MIN_FILE + MIN_CHAT));
  const min = MIN_FILE * scale;
  const max = rest - MIN_CHAT * scale;
  const now = Math.round(clamp(rest * (request.fileShare ?? 0.5), min, max));
  return { tray: trayMode, file: { now, min: Math.round(min), max: Math.round(max) }, trayWidth, chat: Math.round(rest - now) };
}

/** The file share a dragged or stepped file width requests. */
export function fileShareFor(width: number, allocation: ReaderAllocation) {
  const rest = (allocation.file?.now ?? 0) + allocation.chat;
  return rest > 0 ? clamp(width, allocation.file?.min ?? 0, allocation.file?.max ?? rest) / rest : null;
}
