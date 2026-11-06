import React, { useContext, useEffect, useRef } from "react";
import { UNSAFE_DataRouterContext, useBlocker } from "react-router-dom";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import { confirmDraftLeave, draftPersistenceUnsafe, registerDraftLeaveConfirmation, useDraftSummary } from "./drafts";
import { DRAFT_LEAVE_APPROVED } from "./useFileTabRoute";

export function FileEditorLeaveGuard() {
  const [open, setOpen] = React.useState(false);
  const resolve = useRef<((value: boolean) => void) | null>(null);
  const dataRouter = useContext(UNSAFE_DataRouterContext);
  useEffect(() => {
    const unregister = registerDraftLeaveConfirmation(() => new Promise<boolean>((done) => {
      resolve.current?.(false); resolve.current = done; setOpen(true);
    }));
    return () => { unregister(); resolve.current?.(false); };
  }, []);
  function finish(value: boolean) { setOpen(false); resolve.current?.(value); resolve.current = null; }
  return <>{dataRouter && <RouterDraftBlocker />}<ConfirmDialog open={open} title="Leave unsaved edits?" confirmLabel="Leave anyway" destructive onCancel={() => finish(false)} onConfirm={() => finish(true)}>
    Some edits cannot be recovered after leaving. Keep this reader open to preserve them, or acknowledge that leaving may lose these changes.
  </ConfirmDialog></>;
}

function RouterDraftBlocker() {
  // Subscribe to persistence acknowledgements as well as typing. The existing
  // data router covers links, imperative navigation and Back with one blocker.
  useDraftSummary("");
  const blocker = useBlocker(({ nextLocation, historyAction }) =>
    !(historyAction !== "POP" && nextLocation.state?.[DRAFT_LEAVE_APPROVED]) && draftPersistenceUnsafe());
  useEffect(() => {
    if (blocker.state !== "blocked") return;
    let active = true;
    void confirmDraftLeave().then((allowed) => {
      if (active) { if (allowed) blocker.proceed(); else blocker.reset(); }
    });
    return () => { active = false; };
  }, [blocker]);
  return null;
}
