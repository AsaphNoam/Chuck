import { PointerContextMenu, type PointerMenuState } from "../ui/PointerContextMenu";
import { webLinkActions } from "../../lib/linkActions";
import { useRevealInFinder } from "../../lib/revealInFinder";
import { useUiStore } from "../../store/uiStore";

// link is a rendered link under the pointer, claimed with claimWebLink; its action
// leads the menu so a right-click keeps it beside annotation (FS-03.R78). fork
// holds Edit or Fork from here for the message under the pointer (FS-01.R58).
export type AnnotationMenuState = {
  x: number;
  y: number;
  label?: string;
  annotate?: () => void;
  copy?: () => void;
  link?: string;
  onOpenInBrowser?: (path: string) => Promise<unknown>;
  fork?: PointerMenuState["actions"];
};

export function AnnotationContextMenu({ menu, onClose }: { menu: AnnotationMenuState | null; onClose: () => void }) {
  const pushError = useUiStore((state) => state.pushError);
  const revealInFinder = useRevealInFinder();
  return (
    <PointerContextMenu
      menu={menu ? {
        x: menu.x,
        y: menu.y,
        actions: [
          ...(menu.link ? webLinkActions(menu.link, pushError, menu.onOpenInBrowser, revealInFinder) : []),
          ...(menu.copy ? [{ label: "Copy selection", select: menu.copy }] : []),
          ...(menu.annotate && menu.label ? [{ label: menu.label, select: menu.annotate }] : []),
          ...(menu.fork ?? []),
        ],
      } : null}
      onClose={onClose}
    />
  );
}
