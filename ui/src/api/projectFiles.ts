import { z } from "zod";
import type { FileContent } from "./types";

// Project file browser API (FS-11.R11–R16, TS-03.R93/R94). The catalog and file
// routes are desktop-only and read beneath the project's shared root; the
// schema below is null-safe so a missing collection never reaches the UI as null.

export const resourceEntrySchema = z.object({
  path: z.string(),
  name: z.string(),
  kind: z.enum(["file", "directory", "symlink", "special"]).catch("special"),
  mod_time: z.string().nullable().default(null),
});

export const resourceCatalogSchema = z.object({
  project_id: z.string().default(""),
  folder: z.string().default(""),
  entries: z.array(resourceEntrySchema).nullable().transform((value) => value ?? []),
  partial: z.boolean().default(false),
  partial_reason: z.string().default(""),
});

export type ResourceEntry = z.infer<typeof resourceEntrySchema>;
export type ResourceCatalog = z.infer<typeof resourceCatalogSchema>;

export type ApiError = Error & { status?: number; code?: string };

async function request(input: string, init?: RequestInit): Promise<Response> {
  const response = await fetch(input, { cache: "no-store", ...init });
  if (response.ok) return response;
  let message = `${response.status} ${response.statusText}`;
  let code = "";
  try {
    const body = (await response.json()) as { error?: { code?: string; message?: string } };
    code = body?.error?.code ?? "";
    if (body?.error?.message) message = body.error.message;
  } catch { /* keep the status line */ }
  throw Object.assign(new Error(message), { status: response.status, code }) as ApiError;
}

const projectBase = (projectId: string) => `/api/projects/${encodeURIComponent(projectId)}`;

export async function getProjectResourceFiles(projectId: string, folder = "", signal?: AbortSignal): Promise<ResourceCatalog> {
  const query = folder ? `?folder=${encodeURIComponent(folder)}` : "";
  const response = await request(`${projectBase(projectId)}/resource-files${query}`, { signal });
  return resourceCatalogSchema.parse(await response.json());
}

export async function getProjectResourceFile(projectId: string, path: string): Promise<FileContent> {
  const response = await request(`${projectBase(projectId)}/resource-file?path=${encodeURIComponent(path)}`);
  return (await response.json()) as FileContent;
}

const revealInit = (path: string): RequestInit => ({
  method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ path }),
});

export async function revealProjectFileInFinder(projectId: string, path: string): Promise<void> {
  await request(`${projectBase(projectId)}/resource-file/reveal-in-finder`, revealInit(path));
}

export async function revealChatFileInFinder(agentId: string, path: string): Promise<void> {
  await request(`/api/sessions/${encodeURIComponent(agentId)}/file/reveal-in-finder`, revealInit(path));
}
