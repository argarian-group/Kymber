import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { useProjectOutlet } from "./useProjectOutlet";
import { servicesPath, type Service } from "./services";

const pollMs = 1000;

export function useServices() {
  const { project, workspace } = useProjectOutlet();
  const path = servicesPath(workspace.id, project.id);
  const [services, setServices] = useState<Service[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [poll, setPoll] = useState(0);
  const reload = () => setAttempt((value) => value + 1);

  useLoadEffect(
    () => api<{ services: Service[] }>(path),
    (response) => {
      setServices(response.services.filter(({ status }) => status !== "deleted"));
      setError(null);
    },
    setError,
    [path, attempt, poll],
  );

  useEffect(() => {
    if (typeof document !== "undefined" && document.visibilityState !== "visible") return;
    const timer = window.setInterval(() => setPoll((value) => value + 1), pollMs);
    return () => window.clearInterval(timer);
  }, [path]);

  return { services, error, reload, path };
}
