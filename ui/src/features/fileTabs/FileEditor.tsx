import React, { useEffect, useRef } from "react";
import type { FileContent } from "../../api/types";
import { AutoGrowTextarea } from "../../components/ui/AutoGrowTextarea";
import { ConfirmDialog } from "../../components/ui/ConfirmDialog";
import type { DisplayMode } from "./model";
import { acknowledgeDraft, discardDraft, draftKey, draftRecoveryAvailable, getDraft, markDraftUnsafe, MAX_DRAFT_TEXT_BYTES, putDraft, restoreDrafts, useDraftSummary, type DraftIdentity, type FileDraft } from "./drafts";

export type FileEditorProps = {
  identity: DraftIdentity;
  initial: FileContent;
  displayMode?: DisplayMode;
  load: () => Promise<FileContent>;
  save: (baseRevision: string, content: string) => Promise<FileContent>;
  onSaved: (snapshot: FileContent) => void;
  onCancel: () => void;
};
const revisionOf = (file: FileContent) => (file as FileContent & { revision?: string }).revision ?? "";
// Textarea DOM values normalize CRLF. Preserve untouched source bytes around
// the actual edit, including mixed newline spelling, instead of rewriting them.
export function editedSource(previous: string, input: string) {
  const normalized = previous.replace(/\r\n?/g, "\n");
  let start = 0;
  while (start < normalized.length && start < input.length && normalized[start] === input[start]) start++;
  let tail = 0;
  while (tail < normalized.length - start && tail < input.length - start && normalized[normalized.length - 1 - tail] === input[input.length - 1 - tail]) tail++;
  function rawOffset(offset: number) {
    let raw = 0;
    for (let index = 0; index < offset; index++, raw++) if (previous[raw] === "\r" && previous[raw + 1] === "\n") raw++;
    return raw;
  }
  return previous.slice(0, rawOffset(start)) + input.slice(start, input.length - tail) + previous.slice(rawOffset(normalized.length - tail));
}
export function FileEditor({ identity, initial, displayMode = "source", load, save, onSaved, onCancel }: FileEditorProps) {
  const [draft, setDraft] = React.useState<FileDraft>(() => getDraft(identity) ?? {
    ...identity, version: 1, key: draftKey(identity), lineage: crypto.randomUUID(), generation: 0,
    text: initial.content, baseRevision: revisionOf(initial), displayMode,
  });
  const current = useRef(draft); current.current = draft;
  const alive = useRef(true);
  const [ready, setReady] = React.useState(false);
  const [saving, setSaving] = React.useState(false);
  const [comparison, setComparison] = React.useState<FileContent | null>(null);
  const [notice, setNotice] = React.useState("");
  const [discardConfirm, setDiscardConfirm] = React.useState(false);
  useDraftSummary(identity.owner);
  useEffect(() => {
    alive.current = true;
    void restoreDrafts().then(() => {
      if (!alive.current) return;
      const recovered = getDraft(identity);
      if (recovered) { current.current = recovered; setDraft(recovered); setNotice("Recovered unsaved edits. Review the current file before saving."); }
      setReady(true);
    });
    return () => { alive.current = false; markDraftUnsafe(current.current.key, false); };
  // The parent keys the editor by source-qualified identity.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
  function update(next: FileDraft) {
    current.current = next; setDraft(next);
    const admitted = putDraft(next);
    markDraftUnsafe(next.key, !admitted);
    if (!admitted) setNotice("Draft storage is full. These edits are kept in memory; reload recovery is unavailable.");
  }
  async function compare(reason: string) {
    setNotice(reason);
    try {
      const disk = await load();
      if (alive.current) setComparison(disk);
    } catch { if (alive.current) setNotice(`${reason} The current file could not be read. Your draft is retained.`); }
  }
  async function submit() {
    const submitted = current.current;
    setSaving(true); setNotice("");
    try {
      const disk = await load();
      if (!alive.current) return;
      if (!submitted.baseRevision || revisionOf(disk) !== submitted.baseRevision) {
        setComparison(disk); setNotice("The file changed. Compare it with your draft and explicitly use this version before saving."); return;
      }
      const saved = await save(submitted.baseRevision, submitted.text);
      await acknowledgeDraft(submitted, revisionOf(saved));
      if (!alive.current || current.current.lineage !== submitted.lineage) return;
      if (current.current.generation === submitted.generation) { markDraftUnsafe(submitted.key, false); onSaved(saved); }
      else {
        const surviving = { ...current.current, baseRevision: revisionOf(saved) };
        update(surviving);
        setNotice("Saved the submitted version. Newer edits remain unsaved.");
      }
    } catch { if (alive.current) await compare("Save was not confirmed. Your draft is retained; review the current file before another save."); }
    finally { if (alive.current) setSaving(false); }
  }
  async function cancel() {
    await discardDraft(identity, draft.lineage);
    markDraftUnsafe(draft.key, false);
    if (alive.current) onCancel();
  }
  const recovered = draftRecoveryAvailable(draft);
  return <section className="file-editor" aria-label="Edit file source" onMouseUp={(event) => event.stopPropagation()}>
    <div className="file-editor-actions">
      <button className="ad-button ad-button-primary" type="button" onClick={() => void submit()} disabled={!ready || saving || !!comparison}>{saving ? "Saving…" : "Save"}</button>
      <button className="ad-button" type="button" onClick={() => { if (draft.generation > 0 || getDraft(identity)) setDiscardConfirm(true); else void cancel(); }} disabled={saving}>Cancel</button>
      <span role="status">{draft.generation > 0 || getDraft(identity) ? (recovered ? "Unsaved edits · recovery available" : "Unsaved edits · reload recovery unavailable") : "Editing source"}</span>
    </div>
    {notice && <p role="status">{notice}</p>}
    <ConfirmDialog open={discardConfirm} title="Discard unsaved edits?" confirmLabel="Discard edits" destructive onCancel={() => setDiscardConfirm(false)} onConfirm={() => { setDiscardConfirm(false); void cancel(); }}>Your draft will be removed. The reader will load the current file from disk.</ConfirmDialog>
    <div className={comparison ? "file-editor-comparison" : undefined}>
      {comparison && <section aria-label="Current disk version"><h3>Current disk version</h3><pre>{comparison.content}</pre>
        <button className="ad-button" type="button" disabled={!revisionOf(comparison)} onClick={() => {
          update({ ...current.current, baseRevision: revisionOf(comparison), generation: current.current.generation + 1 });
          setComparison(null); setNotice("Using the reviewed disk version as the save base. Review your draft, then Save deliberately.");
        }}>Use this version as save base</button>
      </section>}
      <section aria-label="Your draft"><AutoGrowTextarea aria-label="File source draft" autoFocus spellCheck={false} rows={16} maxHeight="60vh" value={draft.text} disabled={!ready}
        onChange={(event) => {
          const text = editedSource(current.current.text, event.target.value);
          if (new TextEncoder().encode(text).length > MAX_DRAFT_TEXT_BYTES) { setNotice("Edits are limited to 512 KiB. The previous draft is retained."); return; }
          update({ ...current.current, text, generation: current.current.generation + 1 });
        }} /></section>
    </div>
  </section>;
}
