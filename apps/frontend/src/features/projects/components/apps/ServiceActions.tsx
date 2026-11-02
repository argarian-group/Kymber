import { useEffect, useState } from "react";
import { Link } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { Modal } from "@/components/ui/Modal";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { isServiceBusy, type Service, type ServiceAction } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type DeployTarget = { id: string; name: string };

const available: Record<string, { action: ServiceAction; label: string; primary?: boolean }[]> = {
  draft: [{ action: "deploy", label: "Deploy", primary: true }],
  running: [
    { action: "deploy", label: "Deploy", primary: true },
    { action: "restart", label: "Restart" },
    { action: "stop", label: "Stop" },
  ],
  stopped: [
    { action: "start", label: "Start", primary: true },
    { action: "deploy", label: "Deploy" },
  ],
  failed: [{ action: "deploy", label: "Deploy", primary: true }],
};

export function ServiceActions({ service, path, workspaceId, onChanged }: { service: Service; path: string; workspaceId: string; onChanged: () => void }) {
  const user = useCurrentUser();
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);
  const [servers, setServers] = useState<DeployTarget[]>([]);
  const [serverId, setServerId] = useState("");
  const [showLogs, setShowLogs] = useState(false);
  const [watchingDeploy, setWatchingDeploy] = useState(false);
  const [logs, setLogs] = useState("Starting deployment…");
  const [deployStatus, setDeployStatus] = useState(service.status);
  const choosing = !service.server_id && !busy;
  const target = serverId || servers[0]?.id;

  useLoadEffect(
    () => (choosing ? api<{ servers: DeployTarget[] }>(`/workspaces/${workspaceId}/servers`) : Promise.resolve({ servers: [] })),
    (response) => setServers(response.servers),
    () => setServers([]),
    [choosing, workspaceId],
  );

  useEffect(() => {
    if (!showLogs) return;
    let active = true;
    const load = async () => {
      try {
        const result = await api<{ status: string; logs: string }>(`${path}/${service.id}/logs`);
        if (active) {
          setLogs(result.logs || "Waiting for container output…");
          setDeployStatus(result.status);
        }
      } catch (cause) {
        if (active) setLogs(cause instanceof Error ? cause.message : "Could not load deployment logs");
      }
    };
    void load();
    const timer = window.setInterval(load, 2000);
    return () => {
      active = false;
      window.clearInterval(timer);
    };
  }, [path, service.id, showLogs]);

  function act(action: ServiceAction) {
    if (action === "deploy") {
      setLogs("Starting deployment…");
      setDeployStatus("deploying");
      setWatchingDeploy(true);
      setShowLogs(true);
    }
    return run(async () => {
      const body = action === "deploy" && choosing && target ? { server_id: target } : undefined;
      await api(`${path}/${service.id}/${action}`, { method: "POST", body });
      onChanged();
    });
  }

  function openLogs() {
    setLogs("Loading logs…");
    setWatchingDeploy(false);
    setShowLogs(true);
  }

  return (
    <div className="grid justify-items-start gap-2 sm:justify-items-end">
      <div className="flex flex-wrap items-center gap-2">
        {choosing && servers.length > 1 && (
          <label className="flex items-center gap-2 text-meta text-ink-soft">
            Deploy to
            <select className="field w-auto cursor-pointer" value={target} onChange={(event) => setServerId(event.target.value)}>
              {servers.map(({ id, name }) => (
                <option value={id} key={id}>
                  {name}
                </option>
              ))}
            </select>
          </label>
        )}
        {(available[service.status] ?? []).map(({ action, label, primary }) => (
          <button className={`btn ${primary ? "btn-primary" : ""}`} type="button" disabled={pending || busy} onClick={() => act(action)} key={action}>
            {label}
          </button>
        ))}
        {service.server_id && !busy && (
          <button className="btn" type="button" onClick={openLogs}>
            Logs
          </button>
        )}
      </div>
      <FormError message={error} />
      {error === "no online server is available" && user.is_root && (
        <Link className="text-button text-meta" to="/servers">
          Set up a server
        </Link>
      )}
      {showLogs && (
        <Modal label={watchingDeploy ? "Deployment logs" : "Container logs"} className="max-w-3xl" onClose={() => setShowLogs(false)} onCancel={() => setShowLogs(false)}>
          <div className="grid gap-4 p-5">
            <div className="flex items-center justify-between gap-4">
              <h2 className="font-display text-lg font-semibold">{!watchingDeploy ? "Container logs" : deployStatus === "running" ? "Deployment complete" : deployStatus === "failed" ? "Deployment failed" : "Deployment logs"}</h2>
              <button className="btn" type="button" onClick={() => setShowLogs(false)}>Close</button>
            </div>
            <pre className="max-h-[60vh] min-h-48 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap text-ink-soft" aria-live="polite">{logs}</pre>
          </div>
        </Modal>
      )}
    </div>
  );
}
