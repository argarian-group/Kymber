import { useState, type FormEvent } from "react";
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

export function AppGeneralPage() {
  const { service, path, canWrite, onChanged, deployOutput } = useAppOutlet();
  const [name, setName] = useState(service.name);
  const [version, setVersion] = useState(service.image_version);
  const [saved, setSaved] = useState<string | null>(null);
  const [deploying, setDeploying] = useState<string | null>(null);
  const [editingCompose, setEditingCompose] = useState(false);
  const [composeDraft, setComposeDraft] = useState(service.compose_file ?? "");
  const { pending, error, run } = useAction();
  const draft = service.status === "draft";
  const busy = isServiceBusy(service.status);
  const changed = name !== service.name || version !== service.image_version;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      if (draft) {
        await api(`${path}/${service.id}`, { method: "PUT", body: { name, image_version: version } });
        setSaved(t("Saved."));
      } else {
        setDeploying(version);
        await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: version } });
        setSaved(t("Deploying {version}.", { version }));
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
            <Field label={t("Image version")} required maxLength={128} disabled={!canWrite || busy} value={version} onChange={(event) => setVersion(event.target.value)} />
          </div>
          {canWrite && (
            <FormFooter error={error} saved={saved}>
              <button className="btn btn-primary" type="submit" disabled={pending || busy || !changed}>
                {pending ? t("Saving…") : draft ? t("Save") : t("Deploy this version")}
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
