import { useEffect, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { isServiceBusy, type Service } from "@/features/projects/services";
import { api, errorMessage } from "@/lib/api";
import { language, t } from "@/lib/i18n";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type GitBuild = { id: string; ref: string; commit_sha?: string; status: string; result_message?: string; created_at: string; artifact_expires_at?: string; trigger: string };

function buildTitle(build: GitBuild) {
  const status = { queued: t("Queued"), running: t("Building…"), succeeded: t("Succeeded"), failed: t("Failed"), cancelled: t("Cancelled") }[build.status] ?? build.status;
  const trigger = { manual: t("Manual"), push: t("Push"), rollback: t("Rollback"), pull_request: t("Pull request") }[build.trigger] ?? build.trigger;
  return `${trigger} · ${status}`;
}

export function GitBuildHistory({ path, service, canWrite, onChanged }: { path: string; service: Service; canWrite: boolean; onChanged: () => void }) {
  const [builds, setBuilds] = useState<GitBuild[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);

  useLoadEffect(
    () => api<GitBuild[]>(`${path}/${service.id}/git-builds`),
    (found) => { setBuilds(found); setLoadError(null); },
    (failure) => setLoadError(errorMessage(failure)),
    [path, service.id, revision, service.status],
  );

  useEffect(() => {
    if (!builds?.some(({ status }) => status === "queued" || status === "running")) return;
    const timer = window.setInterval(() => setRevision((value) => value + 1), 4000);
    return () => window.clearInterval(timer);
  }, [builds]);

  if (!builds?.length && !loadError) return null;
  const reload = () => { onChanged(); setRevision((value) => value + 1); };

  return (
    <AppSection title={t("Builds")} description={t("Deploy an earlier build to roll back. Builds older than 30 days are rebuilt from their commit.")}>
      <FormError message={loadError ?? error} />
      <ul className="grid gap-3">
        {builds?.slice(0, 20).map((build) => {
          const available = build.status === "succeeded" && build.artifact_expires_at && Date.parse(build.artifact_expires_at) > Date.now();
          const current = build.id === service.image_version;
          return (
            <li className="grid gap-2 rounded-control border border-line-subtle p-4 sm:grid-cols-[1fr_auto] sm:items-center" key={build.id}>
              <div className="min-w-0">
                <p className="font-medium">{buildTitle(build)}{current && <span className="ml-2 text-meta text-ink-soft">{t("current")}</span>}</p>
                <p className="text-meta text-ink-soft"><time dateTime={build.created_at}>{new Date(build.created_at).toLocaleString(language)}</time> · {build.ref}{build.commit_sha && <span className="font-mono"> · {build.commit_sha.slice(0, 7)}</span>}</p>
                {build.result_message && <p className="mt-1 text-meta wrap-break-word text-ember">{build.result_message}</p>}
              </div>
              {canWrite && !current && service.status !== "draft" && <div className="flex flex-wrap gap-2">
                {available ? <button className="btn" type="button" disabled={pending || busy} onClick={() => void run(async () => {
                  await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: build.id } });
                  reload();
                })}>{t("Deploy this build")}</button> : build.commit_sha && build.status !== "queued" && build.status !== "running" && <button className="btn" type="button" disabled={pending || busy} onClick={() => void run(async () => {
                  await api(`${path}/${service.id}/git-builds/${build.id}/rebuild`, { method: "POST" });
                  reload();
                })}>{t("Rebuild this commit")}</button>}
              </div>}
            </li>
          );
        })}
      </ul>
    </AppSection>
  );
}
