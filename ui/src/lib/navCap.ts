/** The compact navs show the first `limit` entries; the current one is always
 * visible, replacing the last slot when it would fall outside, and the rest
 * overflow. Order is the caller's (FS-12.R39, FS-03.R98). */
export function capVisible<T>(entries: T[], isCurrent: (entry: T) => boolean, limit = 5) {
  let visible = entries.slice(0, limit);
  if (!visible.some(isCurrent)) {
    const current = entries.find(isCurrent);
    if (current) {
      const kept = new Set([...visible.slice(0, limit - 1), current]);
      visible = entries.filter((entry) => kept.has(entry));
    }
  }
  const shown = new Set(visible);
  return { visible, overflow: entries.filter((entry) => !shown.has(entry)) };
}
