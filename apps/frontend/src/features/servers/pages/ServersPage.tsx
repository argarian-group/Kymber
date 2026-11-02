import { useEffect, useState } from "react";
import { Navigate } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useAlerts } from "@/features/monitoring/useMetrics";
import { RegisterServerForm } from "@/features/servers/components/RegisterServerForm";
import { ServerCard } from "@/features/servers/components/ServerCard";
import { ServerConnectSteps } from "@/features/servers/components/ServerConnectSteps";
import type { Server } from "@/features/servers/types";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

const pollMs = 5000;

export function ServersPage() {
  const user = useCurrentUser();
  const [servers, setServers] = useState<Server[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);
  const [connecting, setConnecting] = useState<{ id: string; publicKey: string } | null>(null);
  const reload = () => setAttempt((value) => value + 1);
  const alerts = useAlerts(user.is_root);

  useLoadEffect(
    () => api<{ servers: Server[] }>("/servers"),
    (response) => {
      setServers(response.servers);
      setError(null);
    },
    setError,
    [attempt],
  );

  useEffect(() => {
    const timer = window.setInterval(reload, pollMs);
    return () => window.clearInterval(timer);
  }, []);

  if (!user.is_root) return <Navigate to="/" replace />;
  if (error && !servers) return <LoadError title={t("Could not load servers")} message={error} onRetry={reload} />;
  if (!servers) return <PageSkeleton />;

  const connectingServer = servers.find(({ id }) => id === connecting?.id);

  return (
    <>
      <PageHeader
        title={t("Servers")}
        description={servers.length === 1 ? t("1 server") : t("{count} servers", { count: servers.length })}
        actions={
          servers.length > 0 &&
          !adding && (
            <button className="btn btn-primary" type="button" onClick={() => setAdding(true)}>
              {t("Add server")}
            </button>
          )
        }
      />
      <div className="grid gap-4 px-4 py-6 md:px-6">
        {connecting && connectingServer && <ServerConnectSteps server={connectingServer} publicKey={connecting.publicKey} onDone={() => setConnecting(null)} />}
        {(servers.length === 0 || adding) && (
          <RegisterServerForm
            onCancel={servers.length > 0 ? () => setAdding(false) : undefined}
            onRegistered={(server, publicKey) => {
              setServers([...servers, server]);
              if (server.connection_type === "ssh") setConnecting({ id: server.id, publicKey });
              setAdding(false);
            }}
          />
        )}
        {servers.map((server) => (
          <ServerCard server={server} alerts={alerts.filter(({ server_id }) => server_id === server.id)} onChanged={reload} key={server.id} />
        ))}
      </div>
    </>
  );
}
