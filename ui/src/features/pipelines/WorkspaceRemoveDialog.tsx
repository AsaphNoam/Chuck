import * as Dialog from "@radix-ui/react-dialog";
import { useRef, useState } from "react";
import { usePipelineWorkspace, useRemovePipelineWorkspace } from "../../api/pipelines";
import type { PipelineWorkspace } from "../../schemas/pipeline";

/** workspaceRemovable is true for a run-owned new checkout that still exists.
 *  The dialog's own detail read decides whether removal is allowed right now
 *  (FS-14.R94, TS-12.R15). */
export function workspaceRemovable(workspace: PipelineWorkspace | undefined): workspace is PipelineWorkspace {
  return Boolean(workspace && workspace.owned && workspace.mode === "new_worktree" && workspace.state !== "removed");
}

export function workspaceStateLabel(state: PipelineWorkspace["state"]) {
  return state === "error" ? "Needs attention" : state.charAt(0).toUpperCase() + state.slice(1);
}

/** WorkspaceRemoveDialog fetches the checkout's current state on open and sends
 *  back exactly what it showed. Keep is the focused default; closing deletes
 *  nothing (FS-14.R94). */
export function WorkspaceRemoveDialog({ workspaceID, onClose }: { workspaceID: string; onClose: () => void }) {
  const detail = usePipelineWorkspace(workspaceID);
  const remove = useRemovePipelineWorkspace();
  const [acknowledged, setAcknowledged] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const keepRef = useRef<HTMLButtonElement>(null);
  const data = detail.data;
  const unknown = Boolean(data && !data.dirty_known);
  const canRemove = Boolean(data && data.removable && !data.blocked && data.state !== "removed");
  const needsAcknowledge = canRemove && unknown && !acknowledged;

  const submit = () => {
    if (!data) return;
    setError(null);
    remove.mutate(
      { id: workspaceID, body: { revision: data.revision, dirty_known: data.dirty_known, dirty: data.dirty, acknowledge_unknown: acknowledged } },
      {
        onSuccess: onClose,
        onError: (reason) => {
          // The checkout may have changed; the refetch shows what is true now and
          // any earlier acknowledgement no longer applies to it.
          setAcknowledged(false);
          setError(reason instanceof Error ? reason.message : String(reason));
        },
      },
    );
  };

  return <Dialog.Root open onOpenChange={(open) => { if (!open && !remove.isPending) onClose(); }}>
    <Dialog.Portal>
      <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
      <Dialog.Content
        className="dialog-content"
        data-ui="dialog"
        data-slot="content"
        data-variant="default"
        onOpenAutoFocus={(event) => { event.preventDefault(); keepRef.current?.focus(); }}
      >
        <Dialog.Title data-slot="title">Remove checkout?</Dialog.Title>
        <Dialog.Description className="pipeline-workspace-note">
          Removing deletes only this run's checkout directory. The branch, its commits and the agents' conversations are kept.
        </Dialog.Description>
        <div data-slot="body" className="pipeline-workspace-dialog-body">
          {detail.isLoading && <p className="pipeline-workspace-note" role="status">Checking the checkout…</p>}
          {detail.error && <p className="form-error">{detail.error.message}</p>}
          {data && <>
            <dl className="pipeline-workspace-facts">
              {data.branch && <div><dt>Branch</dt><dd><code>{data.branch}</code></dd></div>}
              <div><dt>Path</dt><dd><code>{data.cwd}</code></dd></div>
            </dl>
            {data.missing && <p className="pipeline-workspace-note">The checkout directory is already missing from disk.</p>}
            {data.dirty_known && data.dirty && <div className="pipeline-warning"><strong>Uncommitted changes</strong><p>This checkout has changes that are not committed. Removing it discards them.</p></div>}
            {data.dirty_known && !data.dirty && <p className="pipeline-workspace-note">No uncommitted changes were found.</p>}
            {unknown && <div className="pipeline-warning">
              <strong>Uncommitted changes could not be checked</strong>
              <p>Chuck could not determine whether this checkout has uncommitted changes. Removing it may discard them.</p>
              {canRemove && <label className="pipeline-workspace-ack"><input type="checkbox" checked={acknowledged} onChange={(event) => setAcknowledged(event.target.checked)} /> I understand uncommitted changes may be lost</label>}
            </div>}
            {(data.blocked || !data.removable) && <p className="pipeline-workspace-blocked" role="status">Cannot remove this checkout now: {data.blocked || "removal is not available."}</p>}
          </>}
          {error && <p className="form-error" role="alert">{error}</p>}
        </div>
        <div className="form-actions" data-slot="actions">
          {needsAcknowledge && <span className="pipeline-start-blocker">Tick the box above to enable Remove.</span>}
          <button type="button" ref={keepRef} className="ad-button-primary" disabled={remove.isPending} onClick={onClose}>Keep checkout</button>
          {canRemove && <button type="button" className="btn-danger" disabled={remove.isPending || needsAcknowledge} onClick={submit}>
            {remove.isPending ? "Removing…" : data?.dirty_known && data.dirty ? "Remove and discard changes" : "Remove checkout"}
          </button>}
        </div>
      </Dialog.Content>
    </Dialog.Portal>
  </Dialog.Root>;
}
