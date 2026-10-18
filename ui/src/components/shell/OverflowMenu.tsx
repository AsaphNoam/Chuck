import { useEffect, useRef, useState, type ReactNode } from "react";

/** The `+n` disclosure the compact navs share (FS-12.R39, FS-03.R98): a count
 * button opening a menu that closes on Escape, an outside pointer, or when
 * `closeKey` (the route's project or agent) changes. */
export function OverflowMenu({ count, label, closeKey, className, menuClassName, children }: {
  count: number;
  label: string;
  closeKey?: string;
  className: string;
  menuClassName: string;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  const disclosureRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const closeOutside = (event: PointerEvent) => {
      if (!disclosureRef.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", closeOutside);
    return () => document.removeEventListener("pointerdown", closeOutside);
  }, [open]);

  useEffect(() => setOpen(false), [closeKey]);

  return (
    <div
      className={className}
      ref={disclosureRef}
      onKeyDown={(event) => { if (event.key === "Escape") setOpen(false); }}
    >
      <button
        type="button"
        aria-expanded={open}
        aria-haspopup="menu"
        aria-label={label}
        onClick={() => setOpen((value) => !value)}
      >+{count}</button>
      {open && <div className={menuClassName} role="menu">{children}</div>}
    </div>
  );
}
