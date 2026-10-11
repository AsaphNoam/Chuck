import { create } from "zustand";

// Page-session choices only. Cap retained identities so removed projects/groups cannot accumulate.
export const usePhoneGroupChoices = create<{
  choices: Record<string, boolean>;
  toggle: (project: string, group: string) => void;
}>((set) => ({
  choices: {},
  toggle: (project, group) => set(({ choices }) => {
    const key = JSON.stringify([project, group]);
    const { [key]: previous, ...rest } = choices;
    return { choices: Object.fromEntries([...Object.entries(rest), [key, !previous]].slice(-256)) };
  }),
}));
