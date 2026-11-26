import { useEffect, useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { LogViewer } from "@/features/projects/components/apps/LogViewer";
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

type GitBuild = { id: string; ref: string; commit_sha?: string; status: string; result_message?: string; created_at: string; artifact_expires_at?: string; trigger: string };

type Item = { key: string; time: string; active: boolean } & ({ kind: "deploy"; deployment: Deployment } | { kind: "build"; build: GitBuild; deployment?: Deployment });

const pollMs = 2500;
const idlePollMs = 5000;

function deploymentStatus(deployment: Deployment) {
  return { deploying: t("Deploying…"), succeeded: t("Succeeded"), failed: t("Failed"), cancelled: t("Cancelled") }[deployment.status] ?? deployment.status;
}

function deploymentTitle(deployment: Deployment) {
  const action = deployment.action === "deploy" ? t("Deploy") : deployment.action;
  return t("{action} · {status}", { action, status: deploymentStatus(deployment) });
}

function buildTitle(build: GitBuild, deployment?: Deployment) {
  const status = deployment ? deploymentStatus(deployment) : { queued: t("Queued"), running: t("Building…"), succeeded: t("Succeeded"), failed: t("Failed"), cancelled: t("Cancelled") }[build.status] ?? build.status;
  const trigger = { manual: t("Manual"), push: t("Push"), rollback: t("Rollback"), pull_request: t("Pull request") }[build.trigger] ?? build.trigger;
  return deployment ? t("Build and deploy ({trigger}) · {status}", { trigger, status }) : t("Build ({trigger}) · {status}", { trigger, status });
}

function deployOutput(path: string, serviceId: string, deployment: Deployment) {
  return deployment.status === "deploying"
    ? api<{ logs: string }>(`${path}/${serviceId}/logs`).then(({ logs }) => logs)
    : api<{ output: string }>(`${path}/${serviceId}/deployments/${deployment.id}`).then(({ output }) => output);
}

function ItemLog({ item, path, serviceId, serviceName }: { item: Item; path: string; serviceId: string; serviceName: string }) {
  const [log, setLog] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [wrap, setWrap] = useState(true);

  useEffect(() => {
    let current = true;
    const load = async () => {
      try {
        let text: string;
        if (item.kind === "deploy") text = await deployOutput(path, serviceId, item.deployment);
        else {
          const [build, deploy] = await Promise.all([api<{ log: string }>(`${path}/${serviceId}/git-builds/${item.build.id}/log`), item.deployment && deployOutput(path, serviceId, item.deployment)]);
          text = item.deployment ? `${build.log.trimEnd()}\n\n--- ${t("Deploy")} ---\n${deploy || ""}` : build.log;
        }
        if (current) setLog(text);
      } catch {
        if (current) setError(t("Logs could not be loaded."));
      }
    };
    void load();
    if (!item.active) return () => { current = false; };
    const timer = window.setInterval(load, pollMs);
    return () => {
      current = false;
      window.clearInterval(timer);
    };
  }, [item.key, item.active, item.kind === "build" && item.deployment?.id, path, serviceId]);

  return (
    <div className="overflow-hidden rounded-control" id={`log-${item.key}`}>
      <LogViewer logs={log} error={error} loadingText={t("Loading logs…")} fileName={`${serviceName}-${item.key}.txt`} wrap={wrap} onWrapChange={setWrap} />
    </div>
  );
}

export function AppDeploymentsPage() {
  const { service, path, canWrite, onChanged } = useAppOutlet();
  const gitBuild = service.image === "docker.io/kymber/build-artifacts";
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [openKey, setOpenKey] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { pending, error: actionError, run } = useAction();
  const busy = isServiceBusy(service.status);

  useLoadEffect(
    async () => {
      const [{ deployments }, builds] = await Promise.all([
        api<{ deployments: Deployment[] }>(`${path}/${service.id}/deployments`),
        gitBuild ? api<GitBuild[]>(`${path}/${service.id}/git-builds`) : Promise.resolve([]),
      ]);
      const paired = new Set<string>();
      const buildItems = builds.map((build): Item => {
        const deployment = deployments
          .filter((deployment) => deployment.image_version === build.id && !paired.has(deployment.id))
          .sort((a, b) => Date.parse(a.started_at) - Date.parse(b.started_at))[0];
        if (deployment) paired.add(deployment.id);
        const active = build.status === "queued" || build.status === "running" || deployment?.status === "deploying";
        return { kind: "build", build, deployment, key: `build-${build.id}`, time: build.created_at, active };
      });
      return [
        ...deployments.filter(({ id }) => !paired.has(id)).map((deployment): Item => ({ kind: "deploy", deployment, key: `deploy-${deployment.id}`, time: deployment.started_at, active: deployment.status === "deploying" })),
        ...buildItems,
      ].sort((a, b) => Date.parse(b.time) - Date.parse(a.time));
    },
    (loaded) => {
      setItems(loaded);
      setError(null);
    },
    setError,
    [path, service.id, gitBuild, revision, service.status],
  );

  const active = items?.find((item) => item.active)?.key;
  const [followed, setFollowed] = useState<string | undefined>();
  useEffect(() => {
    if (active) setFollowed(active);
  }, [active]);
  const shown = openKey ?? active ?? followed;

  const anyActive = items?.some(({ active }) => active) ?? false;
  useEffect(() => {
    const timer = window.setInterval(() => setRevision((value) => value + 1), anyActive ? pollMs : idlePollMs);
    return () => window.clearInterval(timer);
  }, [anyActive]);

  const reload = () => {
    onChanged();
    setRevision((value) => value + 1);
  };

  return (
    <AppSection title={t("Deployments")} description={gitBuild ? t("Builds and deploys of this app, newest first. The one in progress opens with live logs.") : t("History of deploy attempts for this app.")}>
      <FormError message={error} />
      <FormError message={actionError} />
      {items?.length === 0 && <EmptyState title={t("No deployments yet")} description={t("Deploy this app to start its history.")} />}
      <div>
        {items?.map((item) => {
          const open = shown === item.key;
          const build = item.kind === "build" ? item.build : null;
          const deployment = item.kind === "deploy" ? item.deployment : null;
          const buildDeployment = item.kind === "build" ? item.deployment : undefined;
          const available = build?.status === "succeeded" && build.artifact_expires_at && Date.parse(build.artifact_expires_at) > Date.now();
          const finishedBuild = build && build.status !== "queued" && build.status !== "running";
          return (
            <div className="grid gap-3 border-t border-line p-4 first:border-t-0" key={item.key}>
              <article className="grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center">
                <div className="min-w-0">
                  <p className="font-medium">
                    {build ? buildTitle(build, buildDeployment) : deploymentTitle(deployment!)}
                    {build?.id === service.image_version && <span className="ml-2 text-meta text-ink-soft">{t("current")}</span>}
                  </p>
                  <p className="text-meta text-ink-soft">
                    <time dateTime={item.time}>{new Date(item.time).toLocaleString(language)}</time>
                    {build && <> · {build.ref}{build.commit_sha && <span className="font-mono"> · {build.commit_sha.slice(0, 7)}</span>}</>}
                  </p>
                  {deployment && !service.compose && deployment.image_version && <p className="mt-1 font-mono text-xs text-ink-soft">{gitBuild ? t("Build {id}", { id: deployment.image_version }) : `${service.image}:${deployment.image_version}`}</p>}
                  {(build?.result_message || deployment?.last_error || buildDeployment?.last_error) && <p className="mt-2 text-meta wrap-break-word text-ember">{build?.result_message || deployment?.last_error || buildDeployment?.last_error}</p>}
                </div>
                <div className="flex flex-wrap gap-2">
                  {canWrite && build && build.id !== service.image_version && service.status !== "draft" && (available ? (
                    <button className="btn" type="button" disabled={pending || busy} onClick={() => void run(async () => {
                      await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: build.id } });
                      reload();
                    })}>{t("Deploy this build")}</button>
                  ) : finishedBuild && build.commit_sha && (
                    <button className="btn" type="button" disabled={pending || busy} onClick={() => void run(async () => {
                      await api(`${path}/${service.id}/git-builds/${build.id}/rebuild`, { method: "POST" });
                      reload();
                    })}>{t("Rebuild this commit")}</button>
                  ))}
                  {canWrite && deployment && !gitBuild && service.category === "application" && !service.compose && deployment.status === "succeeded" && deployment.image_version !== service.image_version && (
                    <button className="btn" type="button" disabled={pending || busy} onClick={() => void run(async () => {
                      await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: deployment.image_version } });
                      reload();
                    })}>{t("Deploy version {version}", { version: deployment.image_version })}</button>
                  )}
                  <button className="btn" type="button" aria-expanded={open} aria-controls={open ? `log-${item.key}` : undefined} onClick={() => setOpenKey(open ? "" : item.key)}>
                    {open ? t("Hide logs") : t("View logs")}
                  </button>
                </div>
              </article>
              {open && <ItemLog item={item} path={path} serviceId={service.id} serviceName={service.name} />}
            </div>
          );
        })}
      </div>
    </AppSection>
  );
}
