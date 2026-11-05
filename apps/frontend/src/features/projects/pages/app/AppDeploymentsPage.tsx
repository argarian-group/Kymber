import { useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { language, t } from "@/lib/i18n";
import { useLoadEffect } from "@/lib/useLoadEffect";

type Deployment = {
  id: string;
  action: string;
  status: string;
  last_error: string;
  started_at: string;
  finished_at: string | null;
};

export function AppDeploymentsPage() {
  const { service, path } = useAppOutlet();
  const [deployments, setDeployments] = useState<Deployment[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<{ id: string; output: string } | null>(null);

  useLoadEffect(
    () => api<{ deployments: Deployment[] }>(`${path}/${service.id}/deployments`).then(({ deployments }) => deployments),
    setDeployments,
    setError,
    [path, service.id],
  );

  return (
    <AppSection title={t("Deployments")} description={t("History of deploy attempts for this app.")}>
      <FormError message={error} />
      {deployments?.length === 0 && <EmptyState title={t("No deployments yet")} description={t("Deploy this app to start its history.")} />}
      <div className="grid gap-3">
        {deployments?.map((deployment) => (
          <div className="grid gap-2" key={deployment.id}>
            <article className="grid gap-2 rounded-control border border-line-subtle p-4 sm:grid-cols-[1fr_auto] sm:items-center">
              <div className="min-w-0">
                <p className="font-medium">{t("{action} · {status}", { action: deployment.action, status: deployment.status })}</p>
                <time className="text-meta text-ink-soft" dateTime={deployment.started_at}>
                  {new Date(deployment.started_at).toLocaleString(language)}
                </time>
                {deployment.last_error && <p className="mt-2 text-meta wrap-break-word text-ember">{deployment.last_error}</p>}
              </div>
              <button className="btn" type="button" onClick={async () => {
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
            </article>
            {selected?.id === deployment.id && <pre className="max-h-96 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap wrap-break-word text-ink-soft">{selected.output || t("No output recorded.")}</pre>}
          </div>
        ))}
      </div>
    </AppSection>
  );
}
