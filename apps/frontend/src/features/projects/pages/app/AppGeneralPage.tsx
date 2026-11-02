import { useState, type FormEvent } from "react";
import { useEffect } from "react";
import { Field } from "@/components/ui/Field";
import { Modal } from "@/components/ui/Modal";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function AppGeneralPage() {
  const { service, path, canWrite, onChanged } = useAppOutlet();
  const [name, setName] = useState(service.name);
  const [version, setVersion] = useState(service.image_version);
  const [saved, setSaved] = useState<string | null>(null);
  const [showLogs, setShowLogs] = useState(false);
  const [logs, setLogs] = useState("Starting deployment…");
  const [deployStatus, setDeployStatus] = useState(service.status);
  const { pending, error, run } = useAction();
  const draft = service.status === "draft";
  const busy = isServiceBusy(service.status);
  const changed = name !== service.name || version !== service.image_version;

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

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      if (draft) {
        await api(`${path}/${service.id}`, { method: "PUT", body: { name, image_version: version } });
        setSaved("Saved.");
      } else {
        setLogs("Starting deployment…");
        setDeployStatus("deploying");
        setShowLogs(true);
        await api(`${path}/${service.id}/deploy`, { method: "POST", body: { image_version: version } });
        setSaved(`Deploying ${version}.`);
      }
      onChanged();
    });
  }

  return (
    <div className="grid gap-6">
      <AppSection title="Configuration" description={canWrite && !draft ? "The name can only change before the first deploy. Changing the version deploys it again and keeps the data." : undefined}>
        <form onSubmit={submit}>
          <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
            <Field label="App name" required maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title="Lowercase letters, numbers and hyphens, starting with a letter" disabled={!canWrite || !draft} value={name} onChange={(event) => setName(event.target.value)} />
            <Field label="Image version" required maxLength={128} disabled={!canWrite || busy} value={version} onChange={(event) => setVersion(event.target.value)} />
          </div>
          {canWrite && (
            <FormFooter error={error} saved={saved}>
              <button className="btn btn-primary" type="submit" disabled={pending || busy || !changed}>
                {pending ? "Saving…" : draft ? "Save" : "Deploy this version"}
              </button>
            </FormFooter>
          )}
        </form>
      </AppSection>
      <AppSection title="Details">
        <dl className="grid gap-4 p-4 sm:grid-cols-2">
          <ProjectDetail label="Engine">{service.engine}</ProjectDetail>
          <ProjectDetail label="Image">{service.image}</ProjectDetail>
          <ProjectDetail label="Environment">{service.environment_name ?? "None"}</ProjectDetail>
          <ProjectDetail label="Server">{service.server_name ?? "Chosen at first deploy"}</ProjectDetail>
          {service.database_name && <ProjectDetail label="Database">{service.database_name}</ProjectDetail>}
          {service.database_user && <ProjectDetail label="Username">{service.database_user}</ProjectDetail>}
          <ProjectDetail label="App ID">{service.id}</ProjectDetail>
        </dl>
      </AppSection>
      {showLogs && (
        <Modal label="Deployment logs" className="max-w-3xl" onClose={() => setShowLogs(false)} onCancel={() => setShowLogs(false)}>
          <div className="grid gap-4 p-5">
            <div className="flex items-center justify-between gap-4">
              <h2 className="font-display text-lg font-semibold">{deployStatus === "running" ? "Deployment complete" : deployStatus === "failed" ? "Deployment failed" : "Deployment logs"}</h2>
              <button className="btn" type="button" onClick={() => setShowLogs(false)}>Close</button>
            </div>
            <pre className="max-h-[60vh] min-h-48 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap text-ink-soft" aria-live="polite">{logs}</pre>
          </div>
        </Modal>
      )}
    </div>
  );
}
