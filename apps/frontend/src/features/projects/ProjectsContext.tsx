import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { api, errorMessage } from "@/lib/api";

export type Workspace = { id: string; name: string; role: string; created_at: string };
export type Project = { id: string; workspace_id: string; name: string; slug: string; created_at: string };

type ProjectsState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; workspace: Workspace | null; projects: Project[] };

type ProjectsContextValue = {
  state: ProjectsState;
  workspaces: Workspace[];
  selectedWorkspaceId: string | null;
  canWrite: boolean;
  retry: () => void;
  selectWorkspace: (id: string) => void;
  createProject: (input: { name: string; slug: string }) => Promise<void>;
  deleteProject: (id: string) => Promise<void>;
};

const ProjectsContext = createContext<ProjectsContextValue | null>(null);

export function useProjects() {
  const value = useContext(ProjectsContext);
  if (!value) throw new Error("useProjects must be used inside ProjectsProvider");
  return value;
}

export function pickWorkspace(workspaces: Workspace[], preferredId: string | null) {
  return workspaces.find(({ id }) => id === preferredId) ?? workspaces.find(({ role }) => role !== "root") ?? workspaces[0] ?? null;
}

function readStoredWorkspace(key: string) {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function storeWorkspace(key: string, id: string) {
  try {
    localStorage.setItem(key, id);
  } catch {
    return;
  }
}

export function ProjectsProvider({ children }: { children: ReactNode }) {
  const user = useCurrentUser();
  const storageKey = `kymber.workspace.${user.id}`;
  const [state, setState] = useState<ProjectsState>({ status: "loading" });
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [requestedWorkspaceId, setRequestedWorkspaceId] = useState(() => readStoredWorkspace(storageKey));
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let current = true;
    async function load() {
      const list = await api<{ workspaces: Workspace[] }>("/workspaces");
      if (!current) return;
      setWorkspaces(list.workspaces);
      const workspace = pickWorkspace(list.workspaces, requestedWorkspaceId);
      if (!workspace) {
        setState({ status: "ready", workspace: null, projects: [] });
        return;
      }
      const { projects } = await api<{ projects: Project[] }>(`/workspaces/${workspace.id}/projects`);
      if (current) setState({ status: "ready", workspace, projects });
    }
    load().catch((error) => current && setState({ status: "error", message: errorMessage(error) }));
    return () => {
      current = false;
    };
  }, [attempt, requestedWorkspaceId]);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  const workspace = state.status === "ready" ? state.workspace : null;
  const selectedWorkspaceId = workspace?.id ?? requestedWorkspaceId;

  const selectWorkspace = useCallback(
    (id: string) => {
      if (id === selectedWorkspaceId) return;
      storeWorkspace(storageKey, id);
      setState({ status: "loading" });
      setRequestedWorkspaceId(id);
    },
    [selectedWorkspaceId, storageKey],
  );

  const workspaceId = workspace?.id;
  const canWrite = workspace !== null && (user.is_root || workspace.role === "owner" || workspace.role === "admin");

  const createProject = useCallback(
    async (input: { name: string; slug: string }) => {
      const { project } = await api<{ project: Project }>(`/workspaces/${workspaceId}/projects`, { method: "POST", body: input });
      setState((previous) => (previous.status === "ready" && previous.workspace?.id === project.workspace_id ? { ...previous, projects: [...previous.projects, project] } : previous));
    },
    [workspaceId],
  );

  const deleteProject = useCallback(
    async (id: string) => {
      await api(`/workspaces/${workspaceId}/projects/${id}`, { method: "DELETE" });
      setState((previous) => (previous.status === "ready" ? { ...previous, projects: previous.projects.filter((project) => project.id !== id) } : previous));
    },
    [workspaceId],
  );

  const value = useMemo(
    () => ({ state, workspaces, selectedWorkspaceId, canWrite, retry, selectWorkspace, createProject, deleteProject }),
    [state, workspaces, selectedWorkspaceId, canWrite, retry, selectWorkspace, createProject, deleteProject],
  );
  return <ProjectsContext.Provider value={value}>{children}</ProjectsContext.Provider>;
}
