import React, { useEffect, useRef } from "react";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { registerDraftLeaveConfirmation } from "./drafts";

export function FileEditorLeaveGuard() {
  const [open, setOpen] = React.useState(false);
  const resolve = useRef<((value: boolean) => void) | null>(null);
  useEffect(() => {
    const unregister = registerDraftLeaveConfirmation(() => new Promise<boolean>((done) => {
      resolve.current?.(false); resolve.current = done; setOpen(true);
    }));
    return () => { unregister(); resolve.current?.(false); };
  }, []);
  function finish(value: boolean) { setOpen(false); resolve.current?.(value); resolve.current = null; }
  return <ConfirmDialog open={open} title="Leave unsaved edits?" confirmLabel="Leave anyway" destructive onCancel={() => finish(false)} onConfirm={() => finish(true)}>
    Some edits cannot be recovered after leaving. Keep this reader open to preserve them, or acknowledge that leaving may lose these changes.
  </ConfirmDialog>;
}
