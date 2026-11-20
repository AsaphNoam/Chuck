import { useCallback } from "react";
import { useLocation, useNavigate, useParams } from "react-router-dom";
import { getProjectResourceFile } from "../../api/projectFiles";
import { FileViewerView, useFileReader } from "../../components/chat/FileViewer";
import type { FileLink } from "../../components/chat/renderers/filePath";

function pathFromHash(hash: string): string {
  try { return decodeURIComponent(hash.replace(/^#/, "")); } catch { return ""; }
}

// A project file opened in its own tab. Read-only: the same scoped JSON loader and
// sandboxed viewer as the in-page reader, with no editor, annotations or native
// browser launch (FS-11.R16, TS-08.R173).
export function ProjectFilePreviewPage() {
  const { project = "" } = useParams();
  const path = pathFromHash(useLocation().hash);
  const outside = path.startsWith("/") || path.split("/").includes("..");
  if (!path) return <p className="project-file-preview-note" role="alert">No file was named.</p>;
  if (outside) return <p className="project-file-preview-note" role="alert">{path} is outside this project's shared files.</p>;
  return <PreviewReader key={`${project}/${path}`} project={project} path={path} />;
}

function PreviewReader({ project, path }: { project: string; path: string }) {
  const navigate = useNavigate();
  const load = useCallback((file: string) => getProjectResourceFile(project, file), [project]);
  const reader = useFileReader({ agentId: "", link: { path }, load });
  const open = (link: FileLink) => navigate({ hash: `#${encodeURIComponent(link.path)}` });
  return (
    <div className="project-file-preview">
      <FileViewerView reader={reader} onOpenFile={open} />
    </div>
  );
}
