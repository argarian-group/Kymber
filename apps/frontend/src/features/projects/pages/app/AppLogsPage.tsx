import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";

const pollMs = 2000;

export function AppLogsPage() {
  const { service, path } = useAppOutlet();
  const [logs, setLogs] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const output = useRef<HTMLPreElement>(null);
  const following = useRef(true);
  const deployed = Boolean(service.server_id);

  useEffect(() => {
    if (!deployed) return;
    let active = true;
    const load = async () => {
      try {
        const result = await api<{ logs: string }>(`${path}/${service.id}/logs`);
        if (active) {
          setLogs(result.logs);
          setError(null);
        }
      } catch (cause) {
        if (active) setError(cause instanceof Error ? cause.message : "Could not load logs");
      }
    };
    void load();
    const timer = window.setInterval(load, pollMs);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [deployed, path, service.id]);

  useLayoutEffect(() => {
    if (following.current && output.current) output.current.scrollTop = output.current.scrollHeight;
  }, [logs]);

  if (!deployed) return <EmptyState title="No runtime logs yet" description="Logs appear here after the first deploy." />;

  return (
    <AppSection title="Runtime logs" description="The last 300 lines from the running container. Refreshes automatically.">
      <div className="grid gap-3 p-4">
        <FormError message={error} />
        <pre
          className="max-h-[60vh] min-h-48 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap text-ink-soft"
          ref={output}
          tabIndex={0}
          onScroll={({ currentTarget: { scrollTop, clientHeight, scrollHeight } }) => {
            following.current = scrollHeight - scrollTop - clientHeight < 24;
          }}
        >
          {logs === null ? "Loading logs…" : logs || "No output yet."}
        </pre>
      </div>
    </AppSection>
  );
}
