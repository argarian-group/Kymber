import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import type { Alert, MetricRange, MetricsResponse } from "./metrics";

const pollMs = 30000;

function usePoll() {
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === "visible") setTick((value) => value + 1);
    };
    const timer = window.setInterval(refresh, pollMs);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", refresh);
    };
  }, []);
  return tick;
}

export function useMetrics(path: string | null, range: MetricRange) {
  const [metrics, setMetrics] = useState<MetricsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const tick = usePoll();

  useLoadEffect(
    () => (path ? api<MetricsResponse>(`${path}?range=${range}`) : Promise.resolve(null)),
    (response) => {
      setMetrics(response);
      setError(null);
    },
    setError,
    [path, range, tick],
  );

  return { metrics, error };
}

export function useAlerts(enabled = true) {
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const tick = usePoll();

  useLoadEffect(
    () => (enabled ? api<{ alerts: Alert[] }>("/monitoring/alerts") : Promise.resolve({ alerts: [] })),
    (response) => setAlerts(response.alerts),
    () => setAlerts([]),
    [enabled, tick],
  );

  return alerts;
}
