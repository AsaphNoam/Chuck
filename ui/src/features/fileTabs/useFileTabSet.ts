import { useState } from "react";
import type { FileTarget, OwnerTabs } from "./model";
import { fileTabs, useFileTabsStore } from "./store";

// An owner's saved file tabs without an address, for the phone's Files view:
// tab actions add no history and the set returns when Files is shown again
// (FS-03.R104/R106, FS-20.R56).

const activeOf = (owner: OwnerTabs | undefined) => owner?.tabs.find((tab) => tab.id === owner.active) ?? null;

export function useFileTabSet(owner: string) {
  const record = useFileTabsStore((state) => state.owners[owner]);
  const [jumps, setJumps] = useState<Record<string, number>>({});
  return {
    owner,
    tabs: record?.tabs ?? [],
    active: activeOf(record),
    jumps,
    open(target: FileTarget) {
      const opened = fileTabs.open(owner, target);
      if (opened && target.line) setJumps((current) => ({ ...current, [opened.tab.id]: (current[opened.tab.id] ?? 0) + 1 }));
    },
    select: (id: string) => fileTabs.select(owner, id),
    close: (id: string) => fileTabs.close(owner, id),
    move: (id: string, to: number) => fileTabs.move(owner, id, to),
  };
}
