import { useEffect, useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { language, t } from "@/lib/i18n";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type Deployment = {
  id: string;
  action: string;
  image_version: string;
  status: string;
  last_error: string;
  started_at: string;
  finished_at: string | null;
};

function deploymentTitle(deployment: Deployment) {
  const action = deployment.action === "deploy" ? t("Deploy") : deployment.action;
  const status = {
    deploying: t("Deploying…"),
    succeeded: t("Succeeded"),
    failed: t("Failed"),
    cancelled: t("Cancelled"),
  }[deployment.status] ?? deployment.status;
  return t("{action} · {status}", { action, status });
}

export function AppDeploymentsPage() {
  const { service, path, canWrite, onChanged } = useAppOutlet();
  const [deployments, setDeployments] = useState<Deployment[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<{ id: string; output: string } | null>(null);
  const [revision, setRevision] = useState(0);
  const { pending, error: actionError, run } = useAction();

  useLoadEffect(
    () => api<{ deployments: Deployment[] }>(`${path}/${service.id}/deployments`).then(({ deployments }) => deployments),
    setDeployments,
    setError,
    [path, service.id, revision],
  );

  useEffect(() => {
    if (!deployments?.some(({ status }) => status === "deploying")) return;
    const timer = window.setInterval(() => setRevision((value) => value + 1), 3000);
    return () => window.clearInterval(timer);
  }, [deployments]);

  return (
    <AppSection title={t("Deployments")} description={t("History of deploy attempts for this app.")}>
      <FormError message={error} />
      <FormError message={actionError} />
      {deployments?.length === 0 && <EmptyState title={t("No deployments yet")} description={t("Deploy this app to start its history.")} />}
      <div className="grid gap-3">
        {deployments?.map((deployment) => (
          <div className="grid gap-2" key={deployment.id}>
            <article className="grid gap-2 rounded-control border border-line-subtle p-4 sm:grid-cols-[1fr_auto] sm:items-center">
              <div className="min-w-0">
                <p className="font-medium">{deploymentTitle(deployment)}</p>
                <time className="text-meta text-ink-soft" dateTime={deployment.started_at}>
                  {new Date(deployment.started_at).toLocaleString(language)}
                </time>
                {!service.compose && deployment.image_version && <p className="mt-1 font-mono text-xs text-ink-soft">{service.image}:{deployment.image_version}</p>}
                {deployment.last_error && <p className="mt-2 text-meta wrap-break-word text-ember">{deployment.last_error}</p>}
              </div>
              <div className="flex flex-wrap gap-2">
                {canWrite && service.category === "application" && !service.compose && deployment.status === "succeeded" && deployment.image_version !== service.image_version && <button className="btn" type="button" disabled={pending || isServiceBusy(service.status)} onClick={() => void run(async () => {
                  await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: deployment.image_version } });
                  onChanged();
                  setRevision((value) => value + 1);
                })}>{t("Deploy version {version}", { version: deployment.image_version })}</button>}
                <button className="btn" type="button" aria-expanded={selected?.id === deployment.id} aria-controls={selected?.id === deployment.id ? `deployment-output-${deployment.id}` : undefined} onClick={async () => {
                if (selected?.id === deployment.id) {
                  setSelected(null);
                  return;
                }
                try {
                  const detail = await api<{ output: string }>(`${path}/${service.id}/deployments/${deployment.id}`);
                  setSelected({ id: deployment.id, output: detail.output });
                } catch {
                  setError(t("Deployment logs could not be loaded."));
                }
                }}>{selected?.id === deployment.id ? t("Hide logs") : t("View logs")}</button>
              </div>
            </article>
            {selected?.id === deployment.id && <pre id={`deployment-output-${deployment.id}`} className="max-h-96 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap wrap-break-word text-ink-soft">{selected.output || (deployment.status === "deploying" ? t("Deployment is still running.") : t("No output recorded."))}</pre>}
          </div>
        ))}
      </div>
    </AppSection>
  );
}
