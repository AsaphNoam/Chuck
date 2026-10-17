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
  actions: Array<{ label: string; select: () => void }>;
};

export function PointerContextMenu({ menu, onClose }: { menu: PointerMenuState | null; onClose: () => void }) {
  const placement = useMenuPlacement(menu);
  const destination = useContext(MenuPortalContext);

  useEffect(() => {
    if (!menu) return;
    const onPointerDown = (event: MouseEvent) => {
      if (!(event.target as HTMLElement)?.closest(".context-menu")) onClose();
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
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
    <div className="context-menu" data-ui="context-menu" ref={placement.ref} style={placement.style} role="menu">
      {menu.actions.map((action) => (
        <button
          key={action.label}
          type="button"
          data-slot="item"
          onClick={() => {
            action.select();
            onClose();
          }}
        >
          {action.label}
        </button>
      ))}
    </div>,
    destination ?? document.body,
  );
}
