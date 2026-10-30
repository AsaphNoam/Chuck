import { useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import * as Dialog from "@radix-ui/react-dialog";
import { copyText } from "../../lib/copyText";
import { useAnnotationStore } from "../../store/annotationStore";
import { useUiStore } from "../../store/uiStore";
import { MenuPortalContext } from "../ui/PointerContextMenu";
import type { PanelSlots } from "./AgentStage";
import { AnnotationContextMenu, type AnnotationMenuState } from "./AnnotationContextMenu";
import { AnnotationDeliveryStatus, AnnotationTrayView, useDesktopAnnotationDelivery } from "./AnnotationTray";
import { FileViewerView } from "./FileViewer";
import { FileTabsNotice, FileTabsReader } from "../../features/fileTabs/FileTabs";
import type { FileTabRoute } from "../../features/fileTabs/useFileTabRoute";

// The transcript's reader workspace: the conversation's file tabs and the
// pending-annotation tray beside the conversation, or both inside Focus file
// (FS-03.R93–R95/R101–R102, TS-08.R127/R129/R142). The tab readers and the tray's
// delivery state live here, above the presentation branch, so moving between
// the forms never reads a file again, sends anything or drops a send in progress.
export function ReaderWorkspace({ agentId, tabs, annotationsEnabled, sourceActive, panelSlots, onReturnFocus }: {
  agentId: string;
  /** The route's saved file tabs; absent where this surface opens no reader. */
  tabs?: FileTabRoute;
  annotationsEnabled: boolean;
  sourceActive: boolean;
  panelSlots?: PanelSlots;
  /** Focuses the conversation when Focus file closes without its trigger. */
  onReturnFocus: () => void;
}) {
  const file = tabs?.active ?? null;
  const [focusTarget, setFocusTarget] = useState<HTMLDivElement | null>(null);
  const delivery = useDesktopAnnotationDelivery(agentId, sourceActive);
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
  // Leaving the transcript (another agent tab, or the page) gives the stage
  // its space back; Focus file keeps this workspace mounted.
  useLayoutEffect(() => () => report?.({ file: false, tray: "none" }), [report]);

  // One stable place for the tab readers, whichever form shows the selected one.
  const readers = tabs && tabs.tabs.length > 0 && (
    <FileTabsReader
      key={tabs.owner}
      tabs={tabs.tabs}
      active={file}
      jumps={tabs.jumps}
      onSelect={tabs.select}
      onClose={tabs.close}
      onMove={tabs.move}
      onResolved={tabs.resolved}
      contextFor={() => agentId}
      onEmptied={onReturnFocus}
      target={focused ? focusTarget : panelSlots?.start || undefined}
      // The view is portalled into Focus file from here, outside the dialog's React
      // tree, so its own menus are pointed at the dialog explicitly.
      renderView={(reader, strip, tab) => (
        <MenuPortalContext.Provider value={focused ? focusTarget : null}>
          <FileViewerView
            reader={reader}
            tabs={strip}
            variant={focused ? "focused" : panelSlots ? "docked" : undefined}
            onClose={() => tabs.close(tab.id)}
            closeLabel={focused ? "Close file" : "Close"}
            onOpenFile={(link) => tabs.open(link)}
            onSelectionMenu={annotationsEnabled ? ({ x, y, text, draft, link }) => setMenu({
              x,
              y,
              label: "Annotate selection",
              copy: () => copyText(text, pushError),
              annotate: () => addAnnotation(agentId, draft),
              link,
            }) : undefined}
            actions={focused
              ? <button type="button" className="ad-button-secondary" onClick={() => setFocused(false)}>Return to chat</button>
              : panelSlots && <button type="button" className="ad-button-secondary" ref={focusButton} onClick={() => setFocused(true)}>Focus file</button>}
          />
        </MenuPortalContext.Provider>
      )}
    />
  );
  const trayView = tray && <AnnotationTrayView delivery={delivery} />;
  const status = annotationsEnabled && !tray && <AnnotationDeliveryStatus delivery={delivery} />;
  const fileMenu = <AnnotationContextMenu menu={menu} onClose={() => setMenu(null)} />;

  return (
    <>
      {tabs?.tabs.length === 0 && <FileTabsNotice />}
      {readers}
      {focused ? (
        <FocusFile
          drafts={tray ? drafts : 0}
          onClose={() => setFocused(false)}
          onCloseAutoFocus={() => requestAnimationFrame(() => {
            if (focusButton.current) focusButton.current.focus();
            else onReturnFocus();
          })}
        >
          <div className="file-focus-reader" ref={setFocusTarget} />
          {trayView}
          {status}
          {fileMenu}
        </FocusFile>
      ) : (
        <>
          {trayView && inSlot(trayView, panelSlots?.trayOverlay ? null : panelSlots?.end)}
          {status}
          {fileMenu}
        </>
      )}
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
