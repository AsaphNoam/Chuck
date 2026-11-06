import { useEffect, useMemo, useState } from "react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import type { SearchReference } from "../../api/types";

export interface SearchSourceRange { seq: number; part_index: number; start_byte: number; end_byte: number }
export interface SearchReveal { ranges: SearchSourceRange[]; key: string }
const keys = ["search_doc", "search_rev", "search_start", "search_end"];

export function parseSearchReference(params: URLSearchParams): SearchReference | null {
  const document_id = params.get("search_doc");
  const content_sha256 = params.get("search_rev");
  const start = params.get("search_start");
  const end = params.get("search_end");
  if (!document_id || new TextEncoder().encode(document_id).length > 128 || !content_sha256 || !/^[a-f0-9]{64}$/.test(content_sha256) || !start || !end || !/^\d+$/.test(start) || !/^\d+$/.test(end)) return null;
  const start_byte = Number(start), end_byte = Number(end);
  if (!Number.isSafeInteger(start_byte) || !Number.isSafeInteger(end_byte) || end_byte <= start_byte) return null;
  return { document_id, content_sha256, start_byte, end_byte };
}

// Both retained-chat routes resolve the same durable reference after their normal hydration.
export function useSearchReference(agentId: string, ready: boolean, archived: boolean | undefined) {
  const [params, setParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const encoded = params.toString();
  const requested = keys.some((key) => params.has(key));
  const reference = useMemo(() => parseSearchReference(new URLSearchParams(encoded)), [encoded]);
  const identity = `${agentId}:${reference ? JSON.stringify(reference) : encoded}`;
  const [result, setResult] = useState<{ identity: string; reveal?: SearchReveal; error?: string }>();
  useEffect(() => {
    if (!requested || !ready || !reference) return;
    const expected = archived === true ? `/archive/${encodeURIComponent(agentId)}` : archived === false ? `/agent/${encodeURIComponent(agentId)}` : location.pathname;
    if (expected !== location.pathname) { navigate({ pathname: expected, search: location.search }, { replace: true }); return; }
    const controller = new AbortController();
    const query = new URLSearchParams({ document_id: reference.document_id, sha256: reference.content_sha256, start: String(reference.start_byte), end: String(reference.end_byte) });
    void fetch(`/api/sessions/${encodeURIComponent(agentId)}/search-match?${query}`, { signal: controller.signal }).then(async (response) => {
      if (!response.ok) throw new Error("Matching passage could not be loaded.");
      return response.json() as Promise<{ status: string; ranges?: SearchSourceRange[]; reason?: string }>;
    }).then((value) => {
      if (controller.signal.aborted) return;
      const ranges = value.ranges ?? [];
      if (value.status !== "exact" || !ranges.length || ranges.some((range) => !Number.isSafeInteger(range.seq) || range.seq < 1 || !Number.isSafeInteger(range.part_index) || range.part_index < 0 || !Number.isSafeInteger(range.start_byte) || !Number.isSafeInteger(range.end_byte) || range.start_byte < 0 || range.end_byte <= range.start_byte)) {
        const reason = typeof value.reason === "string" ? value.reason.slice(0, 240) : "The retained source may have changed or cannot be read.";
        setResult({ identity, error: `Matching passage is unavailable. ${reason}` });
      } else setResult({ identity, reveal: { key: identity, ranges } });
    }).catch((error: unknown) => { if (!controller.signal.aborted) setResult({ identity, error: error instanceof Error ? error.message : "Matching passage could not be loaded." }); });
    return () => controller.abort();
  }, [identity, ready, reference, requested, archived, agentId, location.pathname, location.search, navigate]);
  const current = result?.identity === identity ? result : undefined;
  const clear = () => setParams((previous) => { const next = new URLSearchParams(previous); keys.forEach((key) => next.delete(key)); return next; });
  const notice = requested && (!reference || current?.error) ? <p className="form-error" role="status">{current?.error ?? "This matching-passage link is invalid."} <button type="button" onClick={clear}>Open chat normally</button></p> : null;
  return { requested, reveal: current?.reveal, notice };
}
