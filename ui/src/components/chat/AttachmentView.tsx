import { useCallback, useEffect, useRef, useState } from "react";
import * as Dialog from "@radix-ui/react-dialog";
import type { AttachmentDescriptor } from "../../api/types";
import { attachmentURL } from "../../api/client";
import { isPreviewImage, type DraftAttachment } from "./attachments";

type ViewItem = { name: string; size: number; kind: "image" | "file"; src?: string; draft?: DraftAttachment };

export function DraftAttachments({ files, onRemove, disabled = false }: { files: DraftAttachment[]; onRemove?: (id: string) => void; disabled?: boolean }) {
  return files.length ? <div className="chat-attachment-strip" aria-label="Attachments">
    {files.map((file) => <AttachmentTile key={file.id} item={{ name: file.name, size: file.size, kind: isPreviewImage(file) ? "image" : "file", draft: file }} onRemove={onRemove ? () => onRemove(file.id) : undefined} disabled={disabled} />)}
  </div> : null;
}

export function MessageAttachments({ agentId, attachments }: { agentId: string; attachments: AttachmentDescriptor[] }) {
  return attachments.length ? <div className="chat-attachment-strip chat-message-attachments">
    {attachments.map((attachment) => <AttachmentTile key={attachment.id} item={{
      name: attachment.name,
      size: attachment.size,
      kind: attachment.kind,
      src: attachmentURL(agentId, attachment.id),
    }} />)}
  </div> : null;
}

export function AssistantImageRow({ agentId, attachment, path, name, error }: { agentId: string; attachment?: AttachmentDescriptor; path?: string; name?: string; error?: string }) {
  if (attachment) return <div className="assistant-image-row"><MessageAttachments agentId={agentId} attachments={[attachment]} /></div>;
  if (path) return <div className="assistant-image-row"><LocalImage agentId={agentId} path={path} alt={name || path} /></div>;
  return <p className="chat-image-unavailable" role="status">{error || "Image unavailable"}</p>;
}

function AttachmentTile({ item, onRemove, disabled = false }: { item: ViewItem; onRemove?: () => void; disabled?: boolean }) {
  const [open, setOpen] = useState(false);
  const [src, setSrc] = useState<string | null>(null);
  const [imageError, setImageError] = useState("");
  const [loading, setLoading] = useState(Boolean(item.src));
  const triggerRef = useRef<HTMLButtonElement>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const generationRef = useRef(0);
  const objectURLRef = useRef<string | null>(null);
  const revokeCurrent = () => {
    if (objectURLRef.current) revokeObjectURL(objectURLRef.current);
    objectURLRef.current = null;
  };
  const load = useCallback(async () => {
    const generation = ++generationRef.current;
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    setImageError("");
    setLoading(true);
    try {
      let blob: Blob;
      if (item.draft) blob = item.draft.blob;
      else {
        if (!item.src) return;
        const response = await fetch(item.src, { signal: controller.signal });
        if (!response.ok) throw await responseError(response);
        const mime = response.headers.get("content-type")?.split(";")[0].trim().toLowerCase() ?? "";
        if (!["image/png", "image/jpeg", "image/webp", "image/gif"].includes(mime)) throw new Error("The server did not return a supported image.");
        blob = await readBoundedImage(response, mime);
      }
      if (generation !== generationRef.current || controller.signal.aborted) return;
      revokeCurrent();
      const objectURL = createObjectURL(blob);
      objectURLRef.current = objectURL || null;
      setSrc(objectURL || null);
    } catch (error) {
      if (controller.signal.aborted || generation !== generationRef.current) return;
      revokeCurrent();
      setSrc(null);
      setImageError(error instanceof Error ? error.message : "Image could not be loaded.");
    } finally {
      if (generation === generationRef.current) setLoading(false);
    }
  }, [item.draft, item.src]);
  useEffect(() => {
    generationRef.current += 1;
    controllerRef.current?.abort();
    revokeCurrent();
    setSrc(null);
    if (item.draft || item.src) void load();
    return () => {
      generationRef.current += 1;
      controllerRef.current?.abort();
      revokeCurrent();
    };
  }, [item.draft?.id, item.src]);
  const imageFailed = () => {
    revokeCurrent();
    setSrc(null);
    setLoading(false);
    setImageError("The image bytes could not be decoded.");
  };
  const image = item.kind === "image";
  return <div className="chat-attachment" title={item.name}>
    {image && src ? <button ref={triggerRef} type="button" className="chat-attachment-preview" aria-label={`Open image ${item.name}`} onClick={() => setOpen(true)}>
      <img src={src} alt={item.name} onError={imageFailed} />
    </button> : image ? <div className="chat-attachment-file" title={item.name}>
      <span className="chat-attachment-name">{item.name}</span>
      <span className="chat-attachment-meta">{imageError ? `Image unavailable: ${imageError}` : loading ? "Loading image preview…" : formatSize(item.size)}</span>
      {imageError && <button type="button" className="chat-attachment-retry" onClick={() => void load()}>Retry image</button>}
    </div> : <div className="chat-attachment-file" title={item.name}>
      <span className="chat-attachment-name">{item.name}</span>
      <span className="chat-attachment-meta">{formatSize(item.size)} · Agent reads uploaded file</span>
    </div>}
    {onRemove && <button type="button" className="chat-attachment-remove" aria-label={`Remove ${item.name}`} disabled={disabled} onClick={onRemove}>Remove</button>}
    {image && <Dialog.Root open={open} onOpenChange={setOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
        <Dialog.Content className="dialog-content chat-image-dialog" data-ui="dialog" data-slot="content" data-variant="default" aria-describedby={undefined} onKeyDownCapture={(event) => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); setOpen(false); } }} onEscapeKeyDown={(event) => { event.preventDefault(); setOpen(false); }} onCloseAutoFocus={(event) => { event.preventDefault(); requestAnimationFrame(() => triggerRef.current?.focus()); }}>
          <div className="chat-image-dialog-heading"><Dialog.Title>{item.name}</Dialog.Title><Dialog.Close asChild><button type="button" className="ad-button-secondary">Close</button></Dialog.Close></div>
          {src ? <img src={src} alt={item.name} onError={imageFailed} /> : <div role="status" className="chat-image-unavailable">{imageError ? `Image unavailable: ${imageError}` : "Loading image…"}{imageError && <button type="button" onClick={() => void load()}>Retry</button>}</div>}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>}
  </div>;
}

export function LocalImage({ agentId, path, alt }: { agentId: string; path: string; alt: string }) {
  const [open, setOpen] = useState(false);
  const [src, setSrc] = useState("");
  const [error, setError] = useState("");
  const triggerRef = useRef<HTMLButtonElement>(null);
  const endpoint = `/api/sessions/${encodeURIComponent(agentId)}/image?path=${encodeURIComponent(path)}`;
  const controllerRef = useRef<AbortController | null>(null);
  const generationRef = useRef(0);
  const objectURLRef = useRef<string | null>(null);
  const revokeCurrent = () => {
    if (objectURLRef.current) revokeObjectURL(objectURLRef.current);
    objectURLRef.current = null;
  };
  useEffect(() => {
    generationRef.current += 1;
    controllerRef.current?.abort();
    revokeCurrent();
    setSrc(""); setError("");
    return () => { generationRef.current += 1; controllerRef.current?.abort(); revokeCurrent(); };
  }, [endpoint]);
  const load = async () => {
    const generation = ++generationRef.current;
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    setError("");
    try {
      const response = await fetch(endpoint, { signal: controller.signal });
      if (!response.ok) throw await responseError(response);
      const mime = response.headers.get("content-type")?.split(";")[0].trim().toLowerCase() ?? "";
      if (!["image/png", "image/jpeg", "image/webp", "image/gif"].includes(mime)) throw new Error("The server did not return a supported image.");
      const blob = await readBoundedImage(response, mime);
      if (generation !== generationRef.current || controller.signal.aborted) return;
      revokeCurrent();
      const url = createObjectURL(blob);
      objectURLRef.current = url || null;
      setSrc(url);
    } catch (reason) {
      if (!controller.signal.aborted && generation === generationRef.current) setError(reason instanceof Error ? reason.message : "Image could not be loaded.");
    }
  };
  const imageFailed = () => { revokeCurrent(); setSrc(""); setError("The image bytes could not be decoded."); };
  const setDialogOpen = (next: boolean) => {
    setOpen(next);
    if (!next) {
      generationRef.current += 1;
      controllerRef.current?.abort();
      revokeCurrent();
      setSrc("");
    }
  };
  return <>
    <button ref={triggerRef} type="button" className="chat-local-image" onClick={() => { setOpen(true); void load(); }}>
      <span>{alt || path}</span><span>Open local image</span>
    </button>
    <Dialog.Root open={open} onOpenChange={setDialogOpen}>
      <Dialog.Portal>
        <Dialog.Overlay className="dialog-overlay" data-ui="dialog" data-slot="overlay" />
        <Dialog.Content className="dialog-content chat-image-dialog" data-ui="dialog" data-slot="content" data-variant="default" aria-describedby={undefined} onKeyDownCapture={(event) => { if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); setDialogOpen(false); } }} onEscapeKeyDown={(event) => { event.preventDefault(); setDialogOpen(false); }} onCloseAutoFocus={(event) => { event.preventDefault(); requestAnimationFrame(() => triggerRef.current?.focus()); }}>
          <div className="chat-image-dialog-heading"><Dialog.Title>{alt || path}</Dialog.Title><Dialog.Close asChild><button type="button" className="ad-button-secondary">Close</button></Dialog.Close></div>
          {src ? <img src={src} alt={alt || path} onError={imageFailed} /> : <div role="status" className="chat-image-unavailable">{error ? `Image unavailable: ${error}` : "Loading image…"}<button type="button" onClick={() => void load()}>Retry</button></div>}
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  </>;
}

function formatSize(size: number) {
  return size >= 1024 * 1024 ? `${(size / 1024 / 1024).toFixed(1)} MiB` : `${Math.max(1, Math.round(size / 1024))} KiB`;
}

function createObjectURL(blob: Blob) {
  return typeof URL.createObjectURL === "function" ? URL.createObjectURL(blob) : "";
}

function revokeObjectURL(url: string) {
  if (typeof URL.revokeObjectURL === "function") URL.revokeObjectURL(url);
}

async function readBoundedImage(response: Response, mime: string) {
  const max = 10 * 1024 * 1024;
  const declared = Number(response.headers.get("content-length"));
  if (Number.isFinite(declared) && declared > max) throw new Error("The image exceeds the 10 MiB preview limit.");
  if (!response.body) throw new Error("The image response was empty.");
  const reader = response.body.getReader();
  const chunks: ArrayBuffer[] = [];
  let size = 0;
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    size += value.byteLength;
    if (size > max) {
      await reader.cancel();
      throw new Error("The image exceeds the 10 MiB preview limit.");
    }
    const copy = new ArrayBuffer(value.byteLength);
    new Uint8Array(copy).set(value);
    chunks.push(copy);
  }
  if (!size) throw new Error("The image response was empty.");
  return new Blob(chunks, { type: mime });
}

async function responseError(response: Response) {
  let message = `${response.status} ${response.statusText}`;
  try {
    const body = await response.clone().json() as { error?: { message?: unknown } };
    if (typeof body?.error?.message === "string") message = body.error.message;
  } catch { /* Keep the HTTP status when the server sends no JSON error. */ }
  return new Error(message);
}
