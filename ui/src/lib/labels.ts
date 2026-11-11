// Readable labels for configured entities (FS-02.R69, TS-08.R91). A label is the readable name
// alone; the id is appended only when another entry in the same list shares that name. Option
// values, keys and stored data stay keyed by id.
export type LabelEntry = readonly [id: string, name: string];

export function displayLabel(entries: readonly LabelEntry[], id: string): string {
  const entry = entries.find(([entryId]) => entryId === id);
  if (!entry) return "";
  const duplicates = entries.filter(([, name]) => name === entry[1]).length > 1;
  return duplicates ? `${entry[1]} (${entry[0]})` : entry[1];
}

// Internal ids are never rendered as a stand-in for a missing name; these neutral words are.
export const UNNAMED_AGENT = "Unnamed agent";
export const UNAVAILABLE_AGENT = "Unavailable agent";
export const UNAVAILABLE_PROJECT = "Unavailable project";

/** An agent's name, or a neutral placeholder — never its id. */
export function agentName(name: string | null | undefined): string {
  return name?.trim() || UNNAMED_AGENT;
}

/** A project's configured title. While the project list is still loading this is empty; once
 *  loaded, a missing or untitled project reads as unavailable rather than exposing its id. */
export function projectTitle(
  projects: Readonly<Record<string, { title?: string } | undefined>> | null | undefined,
  id: string | null | undefined,
): string {
  if (!id) return "";
  const title = projects?.[id]?.title?.trim();
  if (title) return title;
  return projects ? UNAVAILABLE_PROJECT : "";
}

/** `[id, label]` for every entry, in order, for rendering a select's options. */
export function displayLabels(entries: readonly LabelEntry[]): [string, string][] {
  return entries.map(([id]) => [id, displayLabel(entries, id)]);
}
