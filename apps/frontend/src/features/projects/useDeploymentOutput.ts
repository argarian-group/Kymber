import { useEffect, useState } from "react";
import { api } from "@/lib/api";

const pollMs = 2000;

export function useDeploymentOutput(path: string, serviceId: string | undefined, status: string | undefined) {
  const [output, setOutput] = useState<string | null>(null);

  useEffect(() => {
    if (!serviceId || status !== "deploying") return;
    let active = true;
    setOutput(null);
    const load = async () => {
      try {
        const result = await api<{ logs: string }>(`${path}/${serviceId}/logs`);
        if (active) setOutput(result.logs);
      } catch {}
    };
    void load();
    const timer = window.setInterval(load, pollMs);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [path, serviceId, status]);

  return output;
}
