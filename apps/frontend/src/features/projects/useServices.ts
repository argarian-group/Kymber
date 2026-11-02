import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { useProjectOutlet } from "./useProjectOutlet";
import { isServiceBusy, servicesPath, type Service } from "./services";

// The API allows 120 requests a minute per client, so only poll quickly while something is changing.
const busyPollMs = 2000;
const idlePollMs = 15000;

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

  const busy = services?.some(({ status }) => isServiceBusy(status)) ?? false;

  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === "visible") setPoll((value) => value + 1);
    };
    const timer = window.setInterval(refresh, busy ? busyPollMs : idlePollMs);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", refresh);
    };
  }, [path, busy]);

  return { services, error, reload, path };
}
