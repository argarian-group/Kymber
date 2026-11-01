import { useState } from "react";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import type { ProjectEnvironment } from "./services";
import { useProjectOutlet } from "./useProjectOutlet";

export function useEnvironments() {
  const { project, workspace } = useProjectOutlet();
  const path = `/workspaces/${workspace.id}/projects/${project.id}/environments`;
  const [environments, setEnvironments] = useState<ProjectEnvironment[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useLoadEffect(
    () => api<{ environments: ProjectEnvironment[] }>(path),
    (response) => {
      setEnvironments(response.environments);
      setError(null);
    },
    setError,
    [path, attempt],
  );

  return { environments, error, path, reload: () => setAttempt((value) => value + 1) };
}
