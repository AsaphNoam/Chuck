import { useEffect, useState } from "react";

/** Project-first agent choices (FS-13.R34, FS-21.R58): only `project`'s agents
 *  until the person opts in to the rest. Hiding them again, or a new `resetKey`
 *  (source change, reopening), clears a selected outside agent. No project means
 *  nothing to scope by, so every item stays visible. */
export function useProjectFirst<T extends { agent_id: string; project: string }>({ items, project, selected, clearSelected, resetKey }: {
  items: T[];
  project: string | undefined;
  selected: string;
  clearSelected: () => void;
  resetKey: unknown;
}) {
  const [includeOthers, setIncludeOthersState] = useState(false);
  const outside = (id: string) => !!project && !!id && items.some((item) => item.agent_id === id && item.project !== project);
  const hide = () => { if (outside(selected)) clearSelected(); };
  useEffect(() => {
    setIncludeOthersState(false);
    hide();
  }, [resetKey]); // eslint-disable-line react-hooks/exhaustive-deps
  const setIncludeOthers = (next: boolean) => {
    setIncludeOthersState(next);
    if (!next) hide();
  };
  const visible = !project || includeOthers ? items : items.filter((item) => item.project === project);
  return { visible, includeOthers, setIncludeOthers };
}

export function IncludeOtherProjects({ checked, onChange, disabled = false, className }: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  disabled?: boolean;
  className?: string;
}) {
  return (
    <label className={className}>
      <input type="checkbox" checked={checked} disabled={disabled} onChange={(event) => onChange(event.target.checked)} /> Include other projects
    </label>
  );
}
