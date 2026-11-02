import { useState } from "react";
import { Link } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { DeploymentDialog } from "@/features/projects/components/apps/DeploymentDialog";
import { isServiceBusy, type Service, type ServiceAction } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type DeployTarget = { id: string; name: string };

const available: Record<string, { action: ServiceAction; label: string; primary?: boolean }[]> = {
  draft: [{ action: "deploy", label: t("Deploy"), primary: true }],
  running: [
    { action: "deploy", label: t("Deploy"), primary: true },
    { action: "restart", label: t("Restart") },
    { action: "stop", label: t("Stop") },
  ],
  stopped: [
    { action: "start", label: t("Start"), primary: true },
    { action: "deploy", label: t("Deploy") },
  ],
  failed: [{ action: "deploy", label: t("Deploy"), primary: true }],
};

const composeAvailable: typeof available = {
  draft: [{ action: "deploy", label: t("Deploy"), primary: true }],
  running: [{ action: "restart", label: t("Restart") }, { action: "stop", label: t("Stop") }],
  stopped: [{ action: "start", label: t("Start"), primary: true }, { action: "deploy", label: t("Deploy") }],
  failed: [{ action: "deploy", label: t("Deploy"), primary: true }],
};

export function ServiceActions({ service, path, workspaceId, deployOutput, onChanged }: { service: Service; path: string; workspaceId: string; deployOutput: string | null; onChanged: () => void }) {
  const user = useCurrentUser();
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);
  const [servers, setServers] = useState<DeployTarget[]>([]);
  const [serverId, setServerId] = useState("");
  const [showDeployment, setShowDeployment] = useState(false);
  const choosing = !service.server_id && !busy;
  const target = serverId || servers[0]?.id;

  useLoadEffect(
    () => (choosing ? api<{ servers: DeployTarget[] }>(`/workspaces/${workspaceId}/servers`) : Promise.resolve({ servers: [] })),
    (response) => setServers(response.servers),
    () => setServers([]),
    [choosing, workspaceId],
  );

  function act(action: ServiceAction) {
    if (action === "deploy" || (service.compose && (action === "start" || action === "restart"))) setShowDeployment(true);
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
            {t("Deploy to")}
            <select className="field w-auto cursor-pointer" value={target} onChange={(event) => setServerId(event.target.value)}>
              {servers.map(({ id, name }) => (
                <option value={id} key={id}>
                  {name}
                </option>
              ))}
            </select>
          </label>
        )}
        {(service.compose ? composeAvailable[service.status] : available[service.status])?.map(({ action, label, primary }) => (
          <button className={`btn ${primary ? "btn-primary" : ""}`} type="button" disabled={pending || busy} onClick={() => act(action)} key={action}>
            {pending && action === "restart" ? t("Restarting…") : label}
          </button>
        ))}
      </div>
      {!showDeployment && <FormError message={error} />}
      {error === t("no online server is available") && user.is_root && (
        <Link className="text-button text-meta" to="/servers">
          {t("Set up a server")}
        </Link>
      )}
      {showDeployment && <DeploymentDialog service={service} version={service.image_version} pending={pending} error={error} output={deployOutput} onClose={() => setShowDeployment(false)} />}
    </div>
  );
}
