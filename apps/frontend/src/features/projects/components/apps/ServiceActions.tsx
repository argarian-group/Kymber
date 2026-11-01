import { useState } from "react";
import { Link } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { isServiceBusy, type Service, type ServiceAction } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

const available: Record<string, { action: ServiceAction; label: string; primary?: boolean }[]> = {
  draft: [{ action: "deploy", label: "Deploy", primary: true }],
  running: [
    { action: "restart", label: "Restart" },
    { action: "stop", label: "Stop" },
  ],
  stopped: [
    { action: "start", label: "Start", primary: true },
    { action: "deploy", label: "Deploy again" },
  ],
  failed: [{ action: "deploy", label: "Deploy again", primary: true }],
};

export function ServiceActions({ service, path, workspaceId, onChanged }: { service: Service; path: string; workspaceId: string; onChanged: () => void }) {
  const user = useCurrentUser();
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);
  const [servers, setServers] = useState<{ id: string; name: string }[]>([]);
  const [serverId, setServerId] = useState("");
  const choosing = !service.server_id && !busy;
  const target = serverId || servers[0]?.id;

  useLoadEffect(
    () => (choosing ? api<{ servers: { id: string; name: string }[] }>(`/workspaces/${workspaceId}/servers`) : Promise.resolve({ servers: [] })),
    (response) => setServers(response.servers),
    () => setServers([]),
    [choosing, workspaceId],
  );

  function act(action: ServiceAction) {
    return run(async () => {
      const body = action === "deploy" && choosing && target ? { server_id: target } : undefined;
      await api(`${path}/${service.id}/${action}`, { method: "POST", body });
      onChanged();
    });
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
      </div>
      <FormError message={error} />
      {error === "no online server is available" && user.is_root && (
        <Link className="text-button text-meta" to="/servers">
          Set up a server
        </Link>
      )}
    </div>
  );
}
