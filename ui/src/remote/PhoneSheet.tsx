import type { ReactNode } from "react";
import { useEffect, useRef } from "react";
import * as Dialog from "@radix-ui/react-dialog";

export function PhoneSheet({ title, onClose, children, centered = false, actions, className = "", initialFocus }: { title: string; onClose: () => void; children: ReactNode; centered?: boolean; /** Placed beside Close in the title row. */ actions?: ReactNode; className?: string; /** The field to focus on open instead of the first control. */ initialFocus?: () => HTMLElement | null | undefined }) {
  const restoreFocusRef = useRef<HTMLElement | null>(null);
  const restoreFocus = () => {
    const target = restoreFocusRef.current;
    restoreFocusRef.current = null;
    if (target?.isConnected) target.focus();
  };
  // Owners close a sheet by unmounting it, which skips Radix's close focus
  // handling; return focus to the control that opened it either way.
  useEffect(() => () => { const target = restoreFocusRef.current; if (target) setTimeout(() => { restoreFocusRef.current = target; restoreFocus(); }); }, []);
  return <Dialog.Root open onOpenChange={(open) => { if (!open) onClose(); }}>
    <Dialog.Portal>
      <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
      <Dialog.Content
        className={`dialog-content phone-sheet${centered ? " phone-sheet-centered" : ""}${className ? ` ${className}` : ""}`}
        data-ui="dialog"
        data-slot="content"
        data-variant="default"
        aria-describedby={undefined}
        onOpenAutoFocus={(event) => {
          restoreFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;
          const target = initialFocus?.();
          if (target) { event.preventDefault(); target.focus(); }
        }}
        onCloseAutoFocus={(event) => {
          event.preventDefault();
          restoreFocus();
        }}
      >
        <div className="phone-section-title"><Dialog.Title data-slot="title">{title}</Dialog.Title>{actions}<Dialog.Close className="phone-sheet-close ad-button-icon" aria-label="Close dialog">×</Dialog.Close></div>
        {children}
      </Dialog.Content>
    </Dialog.Portal>
  </Dialog.Root>;
}
