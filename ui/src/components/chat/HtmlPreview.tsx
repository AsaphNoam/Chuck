import { useEffect, useMemo, useRef } from "react";

// The browser renders the document; the sandbox isolates scripts from Chuck.
// A host-owned policy precedes the original markup so even an early script
// cannot load subresources or fetch an API (FS-03.R109, TS-08.R152).
const POLICY = "default-src 'none'; script-src 'unsafe-inline'; style-src 'unsafe-inline'; img-src data: blob:; media-src data: blob:; font-src data:; connect-src 'none'; frame-src 'none'; base-uri 'none'; form-action 'none'";
const CONTEXT_MENU = "chuck-html-context-menu";
const MENU_BRIDGE = `<script>document.addEventListener('contextmenu',function(event){event.preventDefault();parent.postMessage({type:'${CONTEXT_MENU}',x:event.clientX,y:event.clientY},'*')},true)</script>`;

export function HtmlPreview({ content, path, onContextMenu }: {
  content: string;
  path: string;
  onContextMenu?: (x: number, y: number) => void;
}) {
  const frame = useRef<HTMLIFrameElement>(null);
  const bridge = onContextMenu ? MENU_BRIDGE : "";
  const srcDoc = useMemo(() => {
    const doctype = content.match(/^\s*<!doctype[^>]*>/i)?.[0] ?? "<!doctype html>";
    const markup = content.replace(/^\s*<!doctype[^>]*>/i, "");
    return `${doctype}<meta http-equiv="Content-Security-Policy" content="${POLICY}">${bridge}${markup}`;
  }, [content, bridge]);

  useEffect(() => {
    if (!onContextMenu) return;
    const receive = (event: MessageEvent) => {
      const element = frame.current;
      if (!element || event.source !== element.contentWindow) return;
      const data = event.data;
      if (!data || data.type !== CONTEXT_MENU || !Number.isFinite(data.x) || !Number.isFinite(data.y)) return;
      const bounds = element.getBoundingClientRect();
      if (data.x < 0 || data.y < 0 || data.x > bounds.width || data.y > bounds.height) return;
      onContextMenu(bounds.left + data.x, bounds.top + data.y);
    };
    window.addEventListener("message", receive);
    return () => window.removeEventListener("message", receive);
  }, [onContextMenu]);

  return <iframe ref={frame} className="file-viewer-html" title={`HTML preview: ${path}`} sandbox="allow-scripts" referrerPolicy="no-referrer" srcDoc={srcDoc} />;
}
