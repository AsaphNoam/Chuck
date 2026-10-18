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

export function webLinkActions(href: string, pushError: (title: string, body: string) => void) {
  const file = classifyFileLink(href);
  if (file) return [{ label: "Copy path", select: () => copyText(file.path, pushError) }];
  return [
    // Opened synchronously from the menu click so the browser treats it as user-initiated.
    { label: "Open in new tab", select: () => { window.open(href, "_blank", "noopener,noreferrer"); } },
    { label: "Copy link", select: () => copyText(href, pushError) },
  ];
}
