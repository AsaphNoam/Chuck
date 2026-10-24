import { useMemo, useRef, useState } from "react";
import type { AgentState } from "../../api/types";

const RESERVED_GROUP = "_ungrouped";
const MAX_GROUP_RUNES = 256;

/** Existing group labels of the target project's non-archived agents (FS-02.R80). */
export function projectGroupLabels(agents: Record<string, AgentState>, project: string): string[] {
  if (!project) return [];
  return Object.values(agents)
    .filter((agent) => !agent.archived && agent.project === project)
    .map((agent) => agent.group?.trim())
    .filter((label): label is string => !!label);
}

/** The reason a pending group cannot be submitted, matching the server's normalizeGroup. */
export function groupNameError(value: string): string | null {
  const trimmed = value.trim();
  if (trimmed === RESERVED_GROUP) return "“_ungrouped” is reserved.";
  if ([...trimmed].length > MAX_GROUP_RUNES) return `Group name must be at most ${MAX_GROUP_RUNES} characters.`;
  return null;
}

/** Typed text is the pending group; suggestions only fill it in (FS-02.R81). */
export function GroupPicker({
  value,
  groups,
  onChange,
  id = "group-picker",
  disabled = false,
}: {
  value: string;
  groups: string[];
  onChange: (value: string) => void;
  id?: string;
  disabled?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(-1);
  const [filtering, setFiltering] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const labels = useMemo(() => [...new Set(groups.map((group) => group.trim()).filter((group) => group && group !== RESERVED_GROUP))].sort(), [groups]);
  const filtered = useMemo(() => {
    const needle = filtering ? value.trim().toLowerCase() : "";
    return labels.filter((label) => label.toLowerCase().includes(needle));
  }, [labels, value, filtering]);
  const choices = ["", ...filtered];
  const error = groupNameError(value);

  const choose = (next: string) => {
    onChange(next);
    setOpen(false);
    setActive(-1);
  };

  return (
    <div className="group-picker" ref={rootRef}>
      <input
        id={id}
        role="combobox"
        aria-label="Group"
        aria-expanded={open}
        aria-controls={`${id}-options`}
        aria-activedescendant={open && active >= 0 ? `${id}-option-${active}` : undefined}
        aria-autocomplete="list"
        aria-invalid={error ? true : undefined}
        value={value}
        disabled={disabled}
        onFocus={() => { setOpen(true); setActive(-1); setFiltering(false); }}
        onBlur={() => window.setTimeout(() => {
          if (!rootRef.current?.contains(document.activeElement)) setOpen(false);
        }, 0)}
        onChange={(event) => { onChange(event.target.value); setOpen(true); setActive(-1); setFiltering(true); }}
        onKeyDown={(event) => {
          if (event.key === "Escape" && open) { event.preventDefault(); event.stopPropagation(); setOpen(false); return; }
          if (event.key === "ArrowDown") { event.preventDefault(); setOpen(true); setActive((index) => Math.min(index + 1, choices.length - 1)); return; }
          if (event.key === "ArrowUp") { event.preventDefault(); setActive((index) => Math.max(index - 1, -1)); return; }
          if (event.key === "Enter" && open) { event.preventDefault(); choose(active >= 0 ? choices[active] : value.trim()); }
        }}
      />
      {open && (
        <ul id={`${id}-options`} role="listbox" className="group-picker-options">
          {choices.map((label, index) => (
            <li id={`${id}-option-${index}`} key={label || "_ungrouped"} role="option" aria-selected={active === index}>
              <button type="button" onMouseDown={(event) => event.preventDefault()} onClick={() => choose(label)}>{label || "Ungrouped"}</button>
            </li>
          ))}
        </ul>
      )}
      {error && <p className="form-error" role="alert">{error}</p>}
    </div>
  );
}

export { RESERVED_GROUP };
