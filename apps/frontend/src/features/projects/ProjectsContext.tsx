import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, errorMessage } from "@/lib/api";

export type Workspace = { id: number; name: string; role: string; created_at: string };
export type Project = { id: number; workspace_id: number; name: string; slug: string; created_at: string };

type ProjectsState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; workspace: Workspace | null; projects: Project[] };

type ProjectsContextValue = {
  state: ProjectsState;
  retry: () => void;
  createProject: (input: { name: string; slug: string }) => Promise<void>;
  deleteProject: (id: number) => Promise<void>;
};

const ProjectsContext = createContext<ProjectsContextValue | null>(null);

export function useProjects() {
  const value = useContext(ProjectsContext);
  if (!value) throw new Error("useProjects must be used inside ProjectsProvider");
  return value;
}

async function loadProjects(): Promise<ProjectsState> {
  const { workspaces } = await api<{ workspaces: Workspace[] }>("/workspaces");
  const workspace = workspaces.find(({ role }) => role !== "root") ?? workspaces[0] ?? null;
  if (!workspace) return { status: "ready", workspace: null, projects: [] };
  const { projects } = await api<{ projects: Project[] }>(`/workspaces/${workspace.id}/projects`);
  return { status: "ready", workspace, projects };
}

export function ProjectsProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<ProjectsState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let current = true;
    loadProjects()
      .then((next) => current && setState(next))
      .catch((error) => current && setState({ status: "error", message: errorMessage(error) }));
    return () => {
      current = false;
    };
  }, [attempt]);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  const workspaceId = state.status === "ready" ? state.workspace?.id : undefined;

  const createProject = useCallback(
    async (input: { name: string; slug: string }) => {
      const { project } = await api<{ project: Project }>(`/workspaces/${workspaceId}/projects`, { method: "POST", body: input });
      setState((previous) => (previous.status === "ready" ? { ...previous, projects: [...previous.projects, project] } : previous));
    },
    [workspaceId],
  );

  const deleteProject = useCallback(
    async (id: number) => {
      await api(`/workspaces/${workspaceId}/projects/${id}`, { method: "DELETE" });
      setState((previous) => (previous.status === "ready" ? { ...previous, projects: previous.projects.filter((project) => project.id !== id) } : previous));
    },
    [workspaceId],
  );

  const value = useMemo(() => ({ state, retry, createProject, deleteProject }), [state, retry, createProject, deleteProject]);
  return <ProjectsContext.Provider value={value}>{children}</ProjectsContext.Provider>;
}
