// At most four file reads per mounted owner may be outstanding, counting stale
// ones still in flight. Waiting work is one latest intent per tab, never a queue
// of requests; the selected tab goes first, and an unstarted open for a tab that
// is no longer selected is dropped rather than read in the background
// (TS-08.R144). A dropped tab reads again when it is next selected.

export class ReadDropped extends Error {
  constructor() { super("read dropped"); }
}

export type ReadKind = "open" | "reload";
export type ReadLimiter = ReturnType<typeof createReadLimiter>;

type Intent = { kind: ReadKind; start: () => Promise<unknown>; resolve: (value: unknown) => void; reject: (error: unknown) => void };

export function createReadLimiter(max = 4) {
  let running = 0;
  let selected: string | null = null;
  const active = new Set<string>();
  const waiting = new Map<string, Intent>();

  const drop = (id: string) => {
    const intent = waiting.get(id);
    if (!intent) return;
    waiting.delete(id);
    intent.reject(new ReadDropped());
  };
  const dropUnselectedOpens = () => {
    for (const [id, intent] of [...waiting]) if (intent.kind === "open" && id !== selected) drop(id);
  };
  const pump = () => {
    while (running < max && waiting.size > 0) {
      const id = selected !== null && waiting.has(selected) && !active.has(selected)
        ? selected : [...waiting.keys()].find((key) => !active.has(key));
      if (id === undefined) break;
      const intent = waiting.get(id)!;
      waiting.delete(id);
      running += 1;
      active.add(id);
      let started: Promise<unknown>;
      try { started = intent.start(); }
      catch (error) { started = Promise.reject(error); }
      started.then(intent.resolve, intent.reject).finally(() => {
        running -= 1;
        active.delete(id);
        pump();
      });
    }
  };

  return {
    run<T>(id: string, kind: ReadKind, start: () => Promise<T>): Promise<T> {
      return new Promise<T>((resolve, reject) => {
        drop(id);
        waiting.set(id, { kind, start, resolve: resolve as (value: unknown) => void, reject });
        if (running >= max) dropUnselectedOpens();
        pump();
      });
    },
    select(id: string | null) {
      selected = id;
      dropUnselectedOpens();
    },
    cancel: drop,
    clear() {
      for (const id of [...waiting.keys()]) drop(id);
    },
    get outstanding() { return running; },
  };
}
