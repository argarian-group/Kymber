import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

export type Workspace = { id: string; name: string; role: string; created_at: string; created_by: { id: string; username: string; email: string } | null };
export type Project = { id: string; workspace_id: string; name: string; slug: string; created_at: string };

type ProjectsState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; workspace: Workspace | null; projects: Project[] };

type ProjectsContextValue = {
  state: ProjectsState;
  workspaces: Workspace[];
  selectedWorkspaceId: string | null;
  canWrite: boolean;
  canManageMembers: boolean;
  canInvite: boolean;
  canDelete: boolean;
  retry: () => void;
  selectWorkspace: (id: string) => void;
  createWorkspace: (name: string) => Promise<void>;
  renameWorkspace: (name: string) => Promise<void>;
  deleteWorkspace: () => Promise<void>;
  createProject: (input: { name: string; slug: string }) => Promise<void>;
  deleteProject: (id: string) => Promise<void>;
};

const ProjectsContext = createContext<ProjectsContextValue | null>(null);

export function useProjects() {
  const value = useContext(ProjectsContext);
  if (!value) throw new Error("useProjects must be used inside ProjectsProvider");
  return value;
}

function pickWorkspace(workspaces: Workspace[], preferredId: string | null) {
  return workspaces.find(({ id }) => id === preferredId) ?? workspaces.find(({ role }) => role !== "root") ?? workspaces[0] ?? null;
}

function readStoredWorkspace(key: string) {
  try {
    return localStorage.getItem(key);
  } catch {
    return null;
  }
}

function storageKeyFor(userId: string) {
  return `kymber.workspace.${userId}`;
}

export function rememberWorkspace(userId: string, id: string) {
  try {
    localStorage.setItem(storageKeyFor(userId), id);
  } catch {
    return;
  }
}

export function ProjectsProvider({ children }: { children: ReactNode }) {
  const user = useCurrentUser();
  const storageKey = storageKeyFor(user.id);
  const [state, setState] = useState<ProjectsState>({ status: "loading" });
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [requestedWorkspaceId, setRequestedWorkspaceId] = useState(() => readStoredWorkspace(storageKey));
  const [attempt, setAttempt] = useState(0);

  useLoadEffect<ProjectsState>(
    async (isCurrent) => {
      const list = await api<{ workspaces: Workspace[] }>("/workspaces");
      if (isCurrent()) setWorkspaces(list.workspaces);
      const workspace = pickWorkspace(list.workspaces, requestedWorkspaceId);
      if (!workspace) return { status: "ready", workspace: null, projects: [] };
      const { projects } = await api<{ projects: Project[] }>(`/workspaces/${workspace.id}/projects`);
      return { status: "ready", workspace, projects };
    },
    setState,
    (message) => setState({ status: "error", message }),
    [attempt, requestedWorkspaceId],
  );

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  const workspace = state.status === "ready" ? state.workspace : null;
  const selectedWorkspaceId = workspace?.id ?? requestedWorkspaceId;

  const selectWorkspace = useCallback(
    (id: string) => {
      if (id === selectedWorkspaceId) return;
      rememberWorkspace(user.id, id);
      setState({ status: "loading" });
      setRequestedWorkspaceId(id);
    },
    [selectedWorkspaceId, user.id],
  );

  const workspaceId = workspace?.id;
  const canWrite = workspace !== null && (user.is_root || workspace.role === "owner" || workspace.role === "admin");

  const canManageMembers = workspace !== null && (user.is_root || workspace.role === "owner");
  const canInvite = workspace !== null && (user.is_root || workspace.role === "owner" || workspace.role === "admin");
  const canDelete = workspace !== null && (user.is_root || workspace.created_by?.id === user.id);

  const createWorkspace = useCallback(
    async (name: string) => {
      const created = await api<{ workspace: Workspace }>("/workspaces", { method: "POST", body: { name } });
      selectWorkspace(created.workspace.id);
    },
    [selectWorkspace],
  );

  const renameWorkspace = useCallback(
    async (name: string) => {
      const updated = await api<{ workspace: Workspace }>(`/workspaces/${workspaceId}`, { method: "PUT", body: { name } });
      setWorkspaces((previous) => previous.map((item) => (item.id === updated.workspace.id ? updated.workspace : item)));
      setState((previous) => (previous.status === "ready" && previous.workspace?.id === updated.workspace.id ? { ...previous, workspace: updated.workspace } : previous));
    },
    [workspaceId],
  );

  const deleteWorkspace = useCallback(async () => {
    await api(`/workspaces/${workspaceId}`, { method: "DELETE" });
    setRequestedWorkspaceId(null);
    retry();
  }, [retry, workspaceId]);

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
    () => ({ state, workspaces, selectedWorkspaceId, canWrite, canManageMembers, canInvite, canDelete, retry, selectWorkspace, createWorkspace, renameWorkspace, deleteWorkspace, createProject, deleteProject }),
    [state, workspaces, selectedWorkspaceId, canWrite, canManageMembers, canInvite, canDelete, retry, selectWorkspace, createWorkspace, renameWorkspace, deleteWorkspace, createProject, deleteProject],
  );
  return <ProjectsContext.Provider value={value}>{children}</ProjectsContext.Provider>;
}
