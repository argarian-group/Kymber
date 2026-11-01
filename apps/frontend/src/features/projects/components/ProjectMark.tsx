import { useState } from "react";
import type { Project } from "@/features/projects/ProjectsContext";

const sizes = { xs: "size-6 text-xs", sm: "size-8 text-sm", md: "size-11 text-lg" };

export function ProjectMark({ project, size = "md" }: { project: Project; size?: keyof typeof sizes }) {
  const src = project.logo_version ? `/api/workspaces/${project.workspace_id}/projects/${project.id}/logo?v=${project.logo_version}` : null;
  const [failedSrc, setFailedSrc] = useState<string | null>(null);

  return (
    <span className={`flex flex-none items-center justify-center overflow-hidden rounded-panel bg-seafoam/15 font-display font-semibold text-seafoam ${sizes[size]}`} aria-hidden="true">
      {src && src !== failedSrc ? <img className="size-full object-contain" src={src} alt="" onError={() => setFailedSrc(src)} /> : project.name.trim().charAt(0).toUpperCase()}
    </span>
  );
}
