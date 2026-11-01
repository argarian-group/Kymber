import { useProjects, type Workspace } from "./ProjectsContext";

export function workspaceLabel(workspace: Workspace, workspaces: Workspace[]) {
  const duplicated = workspaces.filter(({ name }) => name === workspace.name).length > 1;
  return duplicated && workspace.role === "root" ? `${workspace.name} …${workspace.id.slice(-4)}` : workspace.name;
}

export function WorkspaceSwitcher() {
  const { workspaces, selectedWorkspaceId, selectWorkspace } = useProjects();
  if (workspaces.length === 0) return null;

  if (workspaces.length === 1) {
    return <p className="min-w-0 truncate text-meta font-medium">{workspaces[0].name}</p>;
  }

  return (
    <label className="block min-w-0">
      <span className="sr-only">Workspace</span>
      <select className="field cursor-pointer truncate font-medium" value={selectedWorkspaceId ?? ""} onChange={(event) => selectWorkspace(event.target.value)}>
        {workspaces.map((workspace) => (
          <option value={workspace.id} key={workspace.id}>
            {workspaceLabel(workspace, workspaces)}
          </option>
        ))}
      </select>
    </label>
  );
}
