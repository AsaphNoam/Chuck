// tool names the running turn's latest in-progress call (FS-03.R108).
export function WorkingIndicator({ tool }: { tool?: string | null }) {
  return (
    <div className="transcript-pending" aria-live="polite">
      <span className="spinner" aria-hidden="true" />
      <span className="transcript-pending-label">{tool ? `Working · ${tool}` : "Working…"}</span>
    </div>
  );
}
