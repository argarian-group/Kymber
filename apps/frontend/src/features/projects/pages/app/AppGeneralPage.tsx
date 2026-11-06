import { useEffect, useState, type FormEvent } from "react";
import { useSearchParams } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { Modal } from "@/components/ui/Modal";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { DeploymentDialog } from "@/features/projects/components/apps/DeploymentDialog";
import CodeEditor from "@/features/projects/components/files/CodeEditor";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";
import { ComposeContainersPage } from "@/features/projects/pages/app/ComposeContainersPage";

function GitBuildSelector({ path, serviceId, value, onChange }: { path: string; serviceId: string; value: string; onChange: (value: string) => void }) {
  const [jobs, setJobs] = useState<{ id: string; status: string; repository_url: string; ref: string; artifact_expires_at?: string }[]>([]);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    let active = true;
    api<{ id: string; status: string; repository_url: string; ref: string; artifact_expires_at?: string }[]>(`${path}/${serviceId}/git-builds`)
      .then((items) => { if (active) { const available = items.filter((item) => item.status === "succeeded" && item.artifact_expires_at && Date.parse(item.artifact_expires_at) > Date.now()); setJobs(available); if (available.length && !value) onChange(available[0].id); } })
      .catch((cause) => { if (active) { setError(cause instanceof Error ? cause.message : t("Could not check GitHub build status")); setJobs([]); } });
    return () => { active = false; };
  }, [path, serviceId]);
  return <label className="block"><span className="mb-1 block text-meta font-medium">{t("Successful GitHub build")}</span><select className="field" required value={value} onChange={(event) => onChange(event.target.value)}><option value="">{t("Select a build")}</option>{jobs.map((job) => <option value={job.id} key={job.id}>{job.repository_url} · {job.ref}</option>)}</select>{error && <FormError message={error} />}</label>;
}

export function AppGeneralPage() {
  const { service, path, canWrite, onChanged, deployOutput } = useAppOutlet();
  const [searchParams] = useSearchParams();
  const [name, setName] = useState(service.name);
  const [version, setVersion] = useState(service.image_version);
  const [saved, setSaved] = useState<string | null>(null);
  const [deploying, setDeploying] = useState<string | null>(null);
  const [editingCompose, setEditingCompose] = useState(false);
  const [composeDraft, setComposeDraft] = useState(service.compose_file ?? "");
  const [port, setPort] = useState(service.port ? String(service.port) : "");
  const [hostPort, setHostPort] = useState(service.port ? String(service.port) : "");
  const [dataPath, setDataPath] = useState(service.data_path ?? "");
  const [githubBuildId, setGithubBuildId] = useState("");
  const { pending, error, run } = useAction();
  const draft = service.status === "draft";
  const busy = isServiceBusy(service.status);
  const gitBuild = service.image === "docker.io/kymber/build-artifacts";
  const changed = name !== service.name || (gitBuild ? draft ? Number(port || 0) !== service.port || dataPath !== (service.data_path ?? "") : githubBuildId !== service.image_version : version !== service.image_version);

  useEffect(() => setVersion(service.image_version), [service.image_version]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      if (draft) {
        await api(`${path}/${service.id}`, { method: "PUT", body: { name, image_version: version, ...(gitBuild ? { port: Number(port || 0), data_path: dataPath } : {}) } });
        if (gitBuild && Number(hostPort) > 0) await api(`${path}/${service.id}/ports`, { method: "POST", body: { host_port: Number(hostPort), container_port: Number(port), protocol: "tcp", bind: "127.0.0.1" } });
        if (gitBuild && searchParams.get("repository_url") && searchParams.get("ref")) {
          await api(`${path}/${service.id}/git-builds`, { method: "POST", body: { repository_url: searchParams.get("repository_url"), ref: searchParams.get("ref"), build_pack: "dockerfile" } });
        }
        setSaved(t("Saved."));
      } else {
        const deployVersion = gitBuild ? githubBuildId : version;
        setDeploying(deployVersion);
        await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: deployVersion } });
        setSaved(t("Deploying {version}.", { version: deployVersion }));
      }
      onChanged();
    });
  }

  async function saveCompose() {
    const saved = await run(async () => {
      await api(`${path}/${service.id}`, { method: "PUT", body: { compose_file: composeDraft } });
    });
    if (saved) {
      setEditingCompose(false);
      onChanged();
    }
  }

  return (
    <div className="grid gap-6">
      {service.compose ? <>
        {canWrite && <button className="btn w-fit" type="button" onClick={() => { setComposeDraft(service.compose_file ?? ""); setEditingCompose(true); }}>{t("Edit Compose")}</button>}
        <ComposeContainersPage />
        {editingCompose && <Modal label={t("Edit Compose")} className="max-w-5xl" onClose={() => setEditingCompose(false)} onCancel={() => setEditingCompose(false)}>
          <div className="grid max-h-[90vh] grid-rows-[auto_minmax(0,1fr)_auto]">
            <header className="flex items-center justify-between gap-4 border-b border-line p-4"><h2 className="font-display text-lg font-semibold">{t("Edit Compose")}</h2><button className="btn" type="button" onClick={() => setEditingCompose(false)}>{t("Cancel")}</button></header>
            <div className="min-h-[60vh] overflow-auto"><CodeEditor filename="docker-compose.yml" value={composeDraft} readOnly={pending} label={t("Compose preview")} onChange={setComposeDraft} onSave={() => void saveCompose()} /></div>
            <footer className="flex items-center justify-between gap-3 border-t border-line p-4"><FormError message={error} /><button className="btn btn-primary" type="button" disabled={pending || composeDraft === service.compose_file} onClick={() => void saveCompose()}>{pending ? t("Saving…") : t("Save")}</button></footer>
          </div>
        </Modal>}
      </> : <>
      <AppSection title={t("Configuration")} description={canWrite && !draft ? t("The name can only change before the first deploy. Changing the version deploys it again and keeps the data.") : undefined}>
        <form onSubmit={submit}>
          <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
            <Field label={t("App name")} required maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} disabled={!canWrite || !draft} value={name} onChange={(event) => setName(event.target.value)} />
            {!gitBuild && <Field label={t("Image version")} required maxLength={128} disabled={!canWrite || busy} value={version} onChange={(event) => setVersion(event.target.value)} />}
      {gitBuild && <>
              <Field label={t("Application port")} type="number" min={0} max={65535} hint={t("Container port; configure it before build to publish a host port.")} value={port} onChange={(event) => setPort(event.target.value)} />
              {draft && <Field label={t("Host port (optional)")} type="number" min={1} max={65535} hint={t("Leave blank to keep the port private.")} value={hostPort} onChange={(event) => setHostPort(event.target.value)} />}
              {draft && <Field label={t("Data path")} maxLength={255} pattern="/.+" title={t("An absolute path inside the container")} hint={t("Optional. Use this if the container stores persistent data.")} value={dataPath} onChange={(event) => setDataPath(event.target.value)} />}
              {gitBuild && !draft && <GitBuildSelector path={path} serviceId={service.id} value={githubBuildId} onChange={setGithubBuildId} />}
            </>}
          </div>
          {canWrite && (
            <FormFooter error={error} saved={saved}>
              <button className="btn btn-primary" type="submit" disabled={pending || busy || !changed || (gitBuild && !draft && !githubBuildId)}>
                {pending ? t("Saving…") : draft ? t("Save") : gitBuild ? t("Deploy selected GitHub build") : t("Deploy this version")}
              </button>
            </FormFooter>
          )}
        </form>
      </AppSection>
      <AppSection title={t("Details")}>
        <dl className="grid gap-4 p-4 sm:grid-cols-2">
          <ProjectDetail label={t("Engine")}>{service.compose ? t("Docker Compose") : service.engine}</ProjectDetail>
          <ProjectDetail label={t("Image")}>{service.image}</ProjectDetail>
          <ProjectDetail label={t("Environment")}>{service.environment_name ?? t("None")}</ProjectDetail>
          <ProjectDetail label={t("Server")}>{service.server_name ?? t("Chosen at first deploy")}</ProjectDetail>
          {service.database_name && <ProjectDetail label={t("Database")}>{service.database_name}</ProjectDetail>}
          {service.database_user && <ProjectDetail label={t("Username")}>{service.database_user}</ProjectDetail>}
          <ProjectDetail label={t("App ID")}>{service.id}</ProjectDetail>
        </dl>
      </AppSection>
      </>}
      {deploying && <DeploymentDialog service={service} version={deploying} pending={pending} error={error} output={deployOutput} onForceStop={() => run(async () => {
        await api(`${path}/${service.id}/cancel-deploy`, { method: "POST" });
        onChanged();
      })} onClose={() => setDeploying(null)} />}
    </div>
  );
}
