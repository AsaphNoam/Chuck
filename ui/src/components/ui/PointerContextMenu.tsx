import { createContext, useContext, useEffect } from "react";
import { createPortal } from "react-dom";
import { useMenuPlacement } from "../../lib/menuPlacement";

/** Where pointer menus mount. A modal provides its own content element so its
 *  menus stay inside its focus trap and outside its dismissal area; everywhere
 *  else menus mount on the document body (TS-08.R129). */
export const MenuPortalContext = createContext<HTMLElement | null>(null);

export type PointerMenuState = {
  x: number;
  y: number;
  /** A disabled action shows its reason as the item's accessible description. */
  actions: Array<{ label: string; select: () => void; disabled?: boolean; description?: string }>;
  /** Set when the keyboard opened the menu: its first action takes focus, and
   *  Escape or an action returns focus here (FS-11.R16, INV §8). */
  returnFocus?: HTMLElement | null;
};

export function PointerContextMenu({ menu, onClose }: { menu: PointerMenuState | null; onClose: () => void }) {
  const placement = useMenuPlacement(menu);
  const destination = useContext(MenuPortalContext);

  useEffect(() => {
    if (menu?.returnFocus) placement.ref.current?.querySelector<HTMLButtonElement>("button:not(:disabled)")?.focus();
  }, [menu]); // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    if (!menu) return;
    const onPointerDown = (event: MouseEvent) => {
      if (!(event.target as HTMLElement)?.closest(".context-menu")) onClose();
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
        menu.returnFocus?.focus();
      }
    };
    window.addEventListener("mousedown", onPointerDown);
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("mousedown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown);
    };
  }, [menu, onClose]);

  if (!menu) return null;

  return createPortal(
    <div
      className="context-menu"
      data-ui="context-menu"
      ref={placement.ref}
      style={placement.style}
      role="menu"
      // A keyboard menu closes when focus leaves it, so Tab never strands it open.
      onBlur={menu.returnFocus ? (event) => { if (!event.currentTarget.contains(event.relatedTarget as Node | null)) onClose(); } : undefined}
    >
      {menu.actions.map((action) => (
        <button
          key={action.label}
          type="button"
          data-slot="item"
          disabled={action.disabled}
          title={action.description}
          aria-description={action.description}
          onClick={() => {
            action.select();
            onClose();
            menu.returnFocus?.focus();
          }}
        >
          {action.label}
        </button>
      ))}
    </div>,
    destination ?? document.body,
  );
}
