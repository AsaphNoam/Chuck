// The project HTML command opens the first-party viewer page in a new tab,
// synchronously from the operator action and with no opener (FS-11.R16). The file
// reference travels in the fragment, so it never reaches the server as a URL path.
export function projectFilePreviewUrl(projectId: string, path: string): string {
  return `/project/${encodeURIComponent(projectId)}/preview#${encodeURIComponent(path)}`;
}

export function openProjectFileTab(projectId: string, path: string) {
  window.open(projectFilePreviewUrl(projectId, path), "_blank", "noopener,noreferrer");
}
