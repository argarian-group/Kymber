import { lazy, Suspense, useState } from "react";
import { Link, Navigate, useSearchParams } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { SelectField } from "@/components/ui/SelectField";
import { useCurrentUser } from "@/features/auth/AuthContext";
import type { Server } from "@/features/servers/types";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

const TerminalView = lazy(() => import("@/features/terminal/TerminalView"));

export function TerminalPage() {
  const user = useCurrentUser();
  const [params, setParams] = useSearchParams();
  const [servers, setServers] = useState<Server[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [connected, setConnected] = useState<string | null>(null);

  useLoadEffect(
    () => (user.is_root ? api<{ servers: Server[] }>("/servers") : Promise.resolve({ servers: [] })),
    (response) => {
      setServers(response.servers.filter(({ ssh_configured }) => ssh_configured));
      setError(null);
    },
    setError,
    [attempt, user.is_root],
  );

  if (!user.is_root) return <Navigate to="/" replace />;
  if (error && !servers) return <LoadError title="Could not load servers" message={error} onRetry={() => setAttempt((value) => value + 1)} />;
  if (!servers) return <PageSkeleton />;

  const selected = servers.find(({ id }) => id === params.get("server")) ?? servers[0];

  return (
    <>
      <PageHeader title="Terminal" description="A shell on your servers. App containers have their own terminal on the app's page." />
      <div className="grid gap-4 px-4 py-6 md:px-6">
        {!selected ? (
          <EmptyState
            title="No servers with SSH"
            action={
              <Link className="btn" to="/servers">
                Go to servers
              </Link>
            }
          />
        ) : (
          <section className="grid gap-4 rounded-panel bg-raised p-4" aria-label="Server terminal">
            <div className="flex flex-wrap items-end gap-3">
              <div className="min-w-56 flex-1 sm:flex-none">
                <SelectField
                  label="Server"
                  value={selected.id}
                  onChange={(event) => {
                    setConnected(null);
                    setParams({ server: event.target.value });
                  }}
                >
                  {servers.map(({ id, name }) => (
                    <option value={id} key={id}>
                      {name}
                    </option>
                  ))}
                </SelectField>
              </div>
              {connected !== selected.id && (
                <button className="btn btn-primary" type="button" onClick={() => setConnected(selected.id)}>
                  Connect
                </button>
              )}
              {connected === selected.id && (
                <button className="btn" type="button" onClick={() => setConnected(null)}>
                  Disconnect
                </button>
              )}
            </div>
            <p className="text-meta text-ink-soft">
              Connects as <code className="font-mono">{selected.ssh_user || "root"}</code>. Commands run directly on the server and can affect every app on it.
            </p>
            {connected === selected.id && (
              <Suspense fallback={<p className="text-meta text-ink-soft">Loading terminal…</p>}>
                <TerminalView path={`/servers/${selected.id}/terminal`} label={`Terminal for ${selected.name}`} key={selected.id} />
              </Suspense>
            )}
          </section>
        )}
      </div>
    </>
  );
}
