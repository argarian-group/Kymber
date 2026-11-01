import { useState } from "react";
import { buildOverview, type Overview } from "@/features/dashboard/overview";
import { type Project } from "@/features/projects/ProjectsContext";
import { servicesPath, type Service } from "@/features/projects/services";
import { boardPath, type Board } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

async function loadProject(project: Project) {
  const [board, { services }] = await Promise.all([api<Board>(`${boardPath(project.workspace_id, project.id)}/board`), api<{ services: Service[] }>(servicesPath(project.workspace_id, project.id))]);
  return { project, board, services };
}

export function useOverview(projects: Project[]) {
  const [overview, setOverview] = useState<Overview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useLoadEffect(
    () => Promise.all(projects.map(loadProject)),
    (data) => {
      setOverview(buildOverview(data));
      setError(null);
    },
    setError,
    [projects, attempt],
  );

  return { overview, error, retry: () => setAttempt((value) => value + 1) };
}
