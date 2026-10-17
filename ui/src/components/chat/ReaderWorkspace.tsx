import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import * as Dialog from "@radix-ui/react-dialog";
import { copyText } from "../../lib/copyText";
import { useAnnotationStore } from "../../store/annotationStore";
import { useUiStore } from "../../store/uiStore";
import { MenuPortalContext } from "../ui/PointerContextMenu";
import type { PanelSlots } from "./AgentStage";
import { AnnotationContextMenu, type AnnotationMenuState } from "./AnnotationContextMenu";
import { AnnotationTrayView, useAnnotationDelivery } from "./AnnotationTray";
import { FileViewerView, useFileReader } from "./FileViewer";
import type { FileLink } from "./renderers/filePath";

// The transcript's reader workspace: the open file and the pending-annotation
// tray beside the conversation, or both inside Focus file (FS-03.R93–R95,
// TS-08.R127/R129). The file read and the tray's delivery state live here, above
// the presentation branch, so moving between the forms never reads the file
// again, sends anything or drops a send in progress.
export function ReaderWorkspace({ agentId, openFile, onOpenFile, annotationsEnabled, sourceActive, panelSlots, onReturnFocus }: {
  agentId: string;
  openFile: FileLink | null;
  onOpenFile?: (link: FileLink | null, options?: { replace?: boolean }) => void;
  annotationsEnabled: boolean;
  sourceActive: boolean;
  panelSlots?: PanelSlots;
  /** Focuses the conversation when Focus file closes without its trigger. */
  onReturnFocus: () => void;
}) {
  const file = onOpenFile ? openFile : null;
  const reader = useFileReader({
    agentId,
    link: file,
    onPathResolved: (path) => { if (file && onOpenFile) onOpenFile({ ...file, path }, { replace: true }); },
  });
  const delivery = useAnnotationDelivery(agentId, sourceActive);
  const drafts = useAnnotationStore((state) => state.bySource[agentId]?.length ?? 0);
  const collapsed = useAnnotationStore((state) => state.collapsedBySource[agentId] ?? false);
  const addAnnotation = useAnnotationStore((state) => state.add);
  const pushError = useUiStore((state) => state.pushError);
  const [focusRequested, setFocused] = useState(false);
  // Focus ends with the file; a later file opens beside the conversation.
  const focused = focusRequested && file !== null && panelSlots !== undefined;
  if (focusRequested && file === null) setFocused(false);
  const [menu, setMenu] = useState<AnnotationMenuState | null>(null);
  const focusButton = useRef<HTMLButtonElement>(null);

  const tray = annotationsEnabled && drafts > 0;
  const report = panelSlots?.report;
  useLayoutEffect(() => {
    report?.({ file: file !== null && !focused, tray: !tray || focused ? "none" : collapsed ? "collapsed" : "expanded" });
  }, [report, file, focused, tray, collapsed]);

  const viewer = file && onOpenFile && (
    <FileViewerView
      reader={reader}
      variant={focused ? "focused" : panelSlots ? "docked" : undefined}
      onClose={() => onOpenFile(null)}
      closeLabel={focused ? "Close file" : "Close"}
      onOpenFile={onOpenFile}
      onSelectionMenu={annotationsEnabled ? ({ x, y, text, draft, link }) => setMenu({
        x,
        y,
        label: "Annotate selection",
        copy: () => copyText(text, pushError),
        annotate: () => addAnnotation(agentId, draft),
        link,
      }) : undefined}
      actions={focused
        ? <Dialog.Close asChild><button type="button" className="ad-button-secondary">Return to chat</button></Dialog.Close>
        : panelSlots && <button type="button" className="ad-button-secondary" ref={focusButton} onClick={() => setFocused(true)}>Focus file</button>}
    />
  );
  const trayView = annotationsEnabled && <AnnotationTrayView delivery={delivery} />;
  const fileMenu = <AnnotationContextMenu menu={menu} onClose={() => setMenu(null)} />;

  if (focused) {
    return (
      <FocusFile
        drafts={tray ? drafts : 0}
        onClose={() => setFocused(false)}
        onCloseAutoFocus={() => requestAnimationFrame(() => {
          if (focusButton.current) focusButton.current.focus();
          else onReturnFocus();
        })}
      >
        {viewer}
        {trayView}
        {fileMenu}
      </FocusFile>
    );
  }
  return (
    <>
      {viewer && inSlot(viewer, panelSlots?.start)}
      {trayView && inSlot(trayView, panelSlots?.trayOverlay ? null : panelSlots?.end)}
      {fileMenu}
    </>
  );
}

// A portal keeps the panel's React state, context and handlers while placing its
// box outside the card.
function inSlot(node: ReactNode, slot: HTMLElement | null | undefined) {
  return slot ? createPortal(node, slot) : node;
}

// Focus file: a viewport-fitted modal over the still-mounted conversation. Its
// menus mount inside it so they stay usable, and an open menu takes Escape
// before the modal does (FS-03.R94, TS-08.R129).
function FocusFile({ drafts, onClose, onCloseAutoFocus, children }: {
  drafts: number;
  onClose: () => void;
  onCloseAutoFocus: () => void;
  children: ReactNode;
}) {
  const [content, setContent] = useState<HTMLDivElement | null>(null);
  const [trayOpen, setTrayOpen] = useState(false);
  return (
    <Dialog.Root open onOpenChange={(open) => { if (!open) onClose(); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
        <Dialog.Content
          ref={setContent}
          className="dialog-content file-focus"
          data-ui="dialog"
          data-slot="content"
          data-variant="default"
          aria-describedby={undefined}
          onEscapeKeyDown={(event) => { if (content?.querySelector(".context-menu")) event.preventDefault(); }}
          onCloseAutoFocus={(event) => { event.preventDefault(); onCloseAutoFocus(); }}
        >
          <Dialog.Title className="ad-visually-hidden">Focus file</Dialog.Title>
          <MenuPortalContext.Provider value={content}>
            <div className="file-focus-body" data-tray={drafts > 0 ? (trayOpen ? "open" : "closed") : undefined}>
              {drafts > 0 && (
                <button type="button" className="file-focus-tray-toggle ad-button-secondary" aria-expanded={trayOpen} onClick={() => setTrayOpen((open) => !open)}>
                  {trayOpen ? "Hide pending annotations" : `Pending annotations (${drafts})`}
                </button>
              )}
              {children}
            </div>
          </MenuPortalContext.Provider>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
