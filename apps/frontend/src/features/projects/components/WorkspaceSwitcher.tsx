import { useLocation, useNavigate } from "react-router";
import { useProjects, workspaceLabel } from "@/features/projects/ProjectsContext";
import { t } from "@/lib/i18n";

export function WorkspaceSwitcher() {
  const { workspaces, selectedWorkspaceId, selectWorkspace } = useProjects();
  const { pathname } = useLocation();
  const navigate = useNavigate();
  if (workspaces.length === 0) return null;

  if (workspaces.length === 1) {
    return <p className="min-w-0 truncate text-meta font-medium">{workspaces[0].name}</p>;
  }

  return (
    <label className="block min-w-0">
      <span className="sr-only">{t("Workspace")}</span>
      <select className="field cursor-pointer truncate font-medium" value={selectedWorkspaceId ?? ""} onChange={(event) => {
          selectWorkspace(event.target.value);
          if (pathname.startsWith("/projects/")) navigate("/projects");
        }}>
        {workspaces.map((workspace) => (
          <option value={workspace.id} key={workspace.id}>
            {workspaceLabel(workspace, workspaces)}
          </option>
        ))}
      </select>
    </label>
  );
}
