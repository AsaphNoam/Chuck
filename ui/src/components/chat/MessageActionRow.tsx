import { useEffect, useId, useState, type ReactNode } from "react";
import { copyText } from "../../lib/copyText";
import { useUiStore } from "../../store/uiStore";
import { CheckIcon, CopyIcon, ForkIcon, PencilIcon } from "../ui/icons";
import type { EditAction, ForkAction, MessageActions } from "./forkActions";

const COPIED_MS = 1500;

/**
 * MessageActionRow is the desktop message's top-right Copy, Edit and Fork icons,
 * revealed on hover or focus within the message (FS-01.R60, R62, TS-08.R171).
 * An unavailable action stays focusable and shows its reason on hover or
 * focus without opening confirmation.
 */
export function MessageActionRow({ actions, onFork, onEdit }: { actions: MessageActions; onFork: (action: ForkAction) => void; onEdit: (action: EditAction) => void }) {
  const pushError = useUiStore((state) => state.pushError);
  const [copied, setCopied] = useState(false);
  useEffect(() => {
    if (!copied) return undefined;
    const timer = window.setTimeout(() => setCopied(false), COPIED_MS);
    return () => window.clearTimeout(timer);
  }, [copied]);
  const fork = actions.fork;
  const edit = actions.edit;
  return (
    <div className="message-actions">
      <MessageAction
        label={copied ? "Copied" : "Copy"}
        icon={copied ? <CheckIcon /> : <CopyIcon />}
        onSelect={() => { void copyText(actions.copy, pushError).then((ok) => setCopied(ok)); }}
      />
      {fork && <MessageAction label="Fork from here" icon={<ForkIcon />} reason={fork.available ? undefined : fork.reason} onSelect={() => onFork(fork)} />}
      {edit && <MessageAction label="Edit" icon={<PencilIcon />} reason={edit.available ? undefined : edit.reason} onSelect={() => onEdit(edit)} />}
      <span className="ad-visually-hidden" aria-live="polite">{copied ? "Copied to clipboard" : ""}</span>
    </div>
  );
}

function MessageAction({ label, icon, reason, onSelect }: { label: string; icon: ReactNode; reason?: string; onSelect: () => void }) {
  const tipId = useId();
  const unavailable = reason !== undefined;
  return (
    <span className="message-action-slot">
      <button
        type="button"
        className="message-action"
        aria-label={label}
        aria-disabled={unavailable || undefined}
        aria-describedby={unavailable ? tipId : undefined}
        onClick={() => { if (!unavailable) onSelect(); }}
      >
        {icon}
      </button>
      <span className="message-action-tip" id={tipId} aria-hidden={unavailable ? undefined : true}>
        {unavailable ? `${label} unavailable: ${reason || "not available right now."}` : label}
      </span>
    </span>
  );
}
