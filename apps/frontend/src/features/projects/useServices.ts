import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { useProjectOutlet } from "./useProjectOutlet";
import { isServiceBusy, servicesPath, type Service } from "./services";

const pollMs = 3000;

export function useServices() {
  const { project, workspace } = useProjectOutlet();
  const path = servicesPath(workspace.id, project.id);
  const [services, setServices] = useState<Service[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const reload = () => setAttempt((value) => value + 1);

  useLoadEffect(
    () => api<{ services: Service[] }>(path),
    (response) => {
      setServices(response.services.filter(({ status }) => status !== "deleted"));
      setError(null);
    },
    setError,
    [path, attempt],
  );

  const busy = services?.some(({ status }) => isServiceBusy(status)) ?? false;
  useEffect(() => {
    if (!busy) return;
    const timer = window.setInterval(reload, pollMs);
    return () => window.clearInterval(timer);
  }, [busy]);

  return { services, error, reload, path };
}
