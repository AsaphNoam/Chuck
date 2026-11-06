import { useEffect, useRef, useState } from "react";
import type { FileContent, FileProbe } from "../../api/types";
import { useUiStore } from "../../store/uiStore";
import { ReadDropped } from "./readLimiter";

/** One selected, visible reader; the shared limiter still owns admission. */
export function useFileFreshness({ identity, selected, snapshot, probe, read, apply, blocked, retry, cancelPending }: {
  identity: string;
  selected: boolean;
  snapshot: FileContent | null;
  probe?: () => Promise<FileProbe>;
  read: () => Promise<FileContent>;
  apply: (file: FileContent) => void;
  blocked: boolean;
  retry: number;
  cancelPending?: () => void;
}) {
  const [changing, setChanging] = useState(false);
  const [available, setAvailable] = useState(false);
  const [stale, setStale] = useState<string | null>(null);
  const latest = useRef({ snapshot, probe, read, apply, blocked, cancelPending });
  latest.current = { snapshot, probe, read, apply, blocked, cancelPending };
  const connection = useUiStore((state) => state.connection);
  const candidate = useRef<FileContent | null>(null);
  const candidateBase = useRef<string | undefined>();
  const busy = useRef(false);
  useEffect(() => {
    if (!blocked && candidate.current) {
      const file = candidate.current;
      candidate.current = null;
      if (candidateBase.current === latest.current.snapshot?.observed_revision) latest.current.apply(file);
      setAvailable(false);
    }
  }, [blocked]);
  useEffect(() => {
    candidate.current = null;
    setAvailable(false);
    setChanging(false);
    setStale(null);
    if (!selected || !probe || !snapshot?.observed_revision) return;
    let stopped = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    let observation: string | undefined = snapshot.observed_revision;
    let since = Date.now();
    let backoff = 1000;
    let visibilityGeneration = 0;
    const tick = async () => {
      if (stopped || document.hidden) return;
      if (busy.current) { timer = setTimeout(tick, 1000); return; }
      busy.current = true;
      const generation = visibilityGeneration;
      const valid = () => !stopped && !document.hidden && generation === visibilityGeneration;
      try {
        const result = await latest.current.probe!();
        if (!valid()) return;
        backoff = 1000;
        setStale(null);
        if (result.observed_revision !== observation) {
          observation = result.observed_revision;
          since = Date.now();
          candidate.current = null;
          setAvailable(false);
        }
        const changed = observation !== latest.current.snapshot?.observed_revision;
        setChanging(changed);
        if (changed && Date.now() - since >= 2000 && candidate.current?.observed_revision !== observation) {
          const file = await latest.current.read();
          if (!valid()) return;
          if (file.observed_revision !== observation) {
            observation = file.observed_revision;
            since = Date.now();
          } else if (latest.current.blocked) {
            candidate.current = file;
            candidateBase.current = latest.current.snapshot?.observed_revision;
            setAvailable(true);
            setChanging(false);
          } else {
            latest.current.apply(file);
            candidate.current = null;
            setAvailable(false);
            setChanging(false);
          }
        }
      } catch (error) {
        if (!valid() || error instanceof ReadDropped) return;
        if ((error as { code?: string }).code === "file_changing") {
          since = Date.now();
          setChanging(true);
        } else {
          backoff = 5000;
          setChanging(false);
          setStale(error instanceof Error ? error.message : "File could not be checked.");
        }
      } finally {
        busy.current = false;
        if (!stopped && !document.hidden) timer = setTimeout(tick, backoff);
      }
    };
    const visibility = () => {
      visibilityGeneration += 1;
      clearTimeout(timer);
      if (document.hidden) latest.current.cancelPending?.();
      if (!document.hidden) { since = Date.now(); void tick(); }
    };
    document.addEventListener("visibilitychange", visibility);
    void tick();
    return () => {
      stopped = true;
      latest.current.cancelPending?.();
      clearTimeout(timer);
      document.removeEventListener("visibilitychange", visibility);
    };
    // Snapshot contents update in place; restarting on every application would
    // reset the observed quiet window. Identity and lifecycle own this effect.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [identity, selected, Boolean(probe), Boolean(snapshot?.observed_revision), connection, retry]);
  return { changing, available, stale };
}
