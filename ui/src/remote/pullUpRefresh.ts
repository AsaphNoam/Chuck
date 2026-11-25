// Pull up at the newest message to reload, mirroring the browser's own
// pull-down at the top of the page (FS-20.R59). Listeners stay passive so
// native scrolling and overscroll bounce are untouched.
import { useEffect, useState } from "react";

export const PULL_THRESHOLD = 80;
// Movement below this is a tap or a wobble, not a pull, and shows no hint.
const PULL_HINT = 12;
const BOTTOM_SLACK = 8;
const EXCLUDED = "input, textarea, select, [contenteditable='true'], [role='dialog'], .phone-sheet";

export type PullState = "idle" | "pulling" | "armed" | "refreshing";

const atBottom = () => window.scrollY + window.innerHeight >= document.documentElement.scrollHeight - BOTTOM_SLACK;
const typing = () => document.activeElement instanceof HTMLElement && document.activeElement.matches(EXCLUDED);

export function usePullUpRefresh(enabled: boolean, onRefresh: () => void): PullState {
  const [state, setState] = useState<PullState>("idle");
  useEffect(() => {
    if (!enabled) return;
    let startY: number | null = null;
    let armed = false;
    const reset = () => { startY = null; armed = false; setState("idle"); };
    const onStart = (event: TouchEvent) => {
      startY = null;
      if (event.touches.length !== 1 || typing() || !atBottom()) return;
      if (event.target instanceof Element && event.target.closest(EXCLUDED)) return;
      startY = event.touches[0].clientY;
    };
    const onMove = (event: TouchEvent) => {
      if (startY === null) return;
      if (event.touches.length !== 1) return reset();
      const distance = startY - event.touches[0].clientY;
      armed = distance >= PULL_THRESHOLD;
      setState(armed ? "armed" : distance >= PULL_HINT ? "pulling" : "idle");
    };
    const onEnd = () => {
      if (startY === null) return;
      if (armed && atBottom()) {
        startY = null;
        setState("refreshing");
        onRefresh();
        return;
      }
      reset();
    };
    window.addEventListener("touchstart", onStart, { passive: true });
    window.addEventListener("touchmove", onMove, { passive: true });
    window.addEventListener("touchend", onEnd);
    window.addEventListener("touchcancel", reset);
    return () => {
      window.removeEventListener("touchstart", onStart);
      window.removeEventListener("touchmove", onMove);
      window.removeEventListener("touchend", onEnd);
      window.removeEventListener("touchcancel", reset);
      setState("idle");
    };
  }, [enabled, onRefresh]);
  return state;
}
