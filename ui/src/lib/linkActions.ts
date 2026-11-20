import type { MouseEvent } from "react";
import { copyText } from "./copyText";
import { classifyFileLink } from "../components/chat/renderers/filePath";

// Rendered links offer their matching action on right-click. The link raises the
// menu unless an enclosing annotation menu claims the same click and composes the
// action with its own, so one right-click never opens two menus (INV §2).
const pending = new WeakMap<Event, string>();

// offerWebLink runs on the anchor, before any enclosing handler sees the bubble. The
// destination is the rendered, sanitized href, never the raw Markdown source.
export function offerWebLink(mouse: MouseEvent<HTMLAnchorElement | HTMLButtonElement | HTMLSpanElement>, open: (href: string) => void) {
  mouse.preventDefault();
  const native = mouse.nativeEvent;
  const current = mouse.currentTarget;
  const href = current instanceof HTMLAnchorElement ? current.href : current.dataset.filePath;
  if (!href) return;
  pending.set(native, href);
  queueMicrotask(() => {
    if (pending.delete(native)) open(href);
  });
}

// claimWebLink is called by an annotation handler that opens its own menu for this
// right-click; it returns the link to compose, if the pointer was on one.
export function claimWebLink(mouse: MouseEvent): string | undefined {
  const href = pending.get(mouse.nativeEvent);
  pending.delete(mouse.nativeEvent);
  return href;
}

export type OpenInBrowser = (path: string) => Promise<unknown>;

export function isHtmlPath(path: string) { return /\.html?$/i.test(path); }

type PushError = (title: string, body: string) => void;

/** The one file-menu composition for chat and project menus (TS-08.R173). `path` is
 *  the scoped path that open and reveal receive; `copyPath` is what Copy path copies.
 *  `open` runs synchronously from the menu click, so a new tab stays user-initiated. */
export function fileActions(pushError: PushError, { path, copyPath = path, open, reveal }: {
  path: string;
  copyPath?: string;
  open?: { label: string; run: (path: string) => unknown };
  reveal?: OpenInBrowser;
}) {
  const report = (title: string, fallback: string) => (error: unknown) => pushError(title, error instanceof Error ? error.message : fallback);
  return [
    ...(open ? [{ label: open.label, select: () => {
      try { void Promise.resolve(open.run(path)).catch(report(`Could not ${open.label.toLowerCase()}`, "Failed to open file")); }
      catch (error) { report(`Could not ${open.label.toLowerCase()}`, "Failed to open file")(error); }
    } }] : []),
    ...(reveal && !window.location.pathname.startsWith("/phone") ? [{ label: "Open in Finder", select: () => {
      void reveal(path).catch(report("Could not open in Finder", "Failed to reveal file"));
    } }] : []),
    { label: "Copy path", select: () => copyText(copyPath, pushError) },
  ];
}

export function webLinkActions(href: string, pushError: PushError, onOpenInBrowser?: OpenInBrowser, onRevealInFinder?: OpenInBrowser) {
  const file = classifyFileLink(href);
  if (file) return fileActions(pushError, {
    path: file.path,
    open: onOpenInBrowser && isHtmlPath(file.path) && !window.location.pathname.startsWith("/phone") ? { label: "Open in browser", run: onOpenInBrowser } : undefined,
    reveal: onRevealInFinder,
  });
  return [
    // Opened synchronously from the menu click so the browser treats it as user-initiated.
    { label: "Open in new tab", select: () => { window.open(href, "_blank", "noopener,noreferrer"); } },
    { label: "Copy link", select: () => copyText(href, pushError) },
  ];
}
