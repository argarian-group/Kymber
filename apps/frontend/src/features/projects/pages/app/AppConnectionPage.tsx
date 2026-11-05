import { useEffect, useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { DeploymentDialog } from "@/features/projects/components/apps/DeploymentDialog";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { PublishedPorts } from "@/features/projects/components/apps/PublishedPorts";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { t } from "@/lib/i18n";

export function AppConnectionPage() {
  const { detail, canWrite, path, onChanged, deployOutput } = useAppOutlet();
  const [domain, setDomain] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const [deploying, setDeploying] = useState(false);
  const { pending, error, run } = useAction();

  useEffect(() => setDomain(detail?.domain ?? ""), [detail?.id, detail?.domain]);

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail) return <PageSkeleton />;
  const service = detail;

  async function saveDomain(deploy = false) {
    setSaved(null);
    await run(async () => {
      if (domain !== (service.domain ?? "")) {
        await api(`${path}/${service.id}/domain`, { method: "PUT", body: { domain } });
        setSaved(t("Domain saved."));
      }
      if (deploy) {
        setDeploying(true);
        await api(`${path}/${service.id}/deploy`, { method: "POST", body: {} });
      }
      onChanged();
    });
  }

  function submit(event: FormEvent) {
    event.preventDefault();
    void saveDomain();
  }

  return (
    <div className="grid gap-5">
      {service.category === "application" && (
        <AppSection title={t("Public domain")} description={t("Optional. Point DNS to this machine. Traefik reaches the app over Docker networking, so no host port needs to be published.")}>
          {service.domain_pending && <p className="border-b border-line p-4 text-meta text-seafoam" role="status">{t("Deploy the app to apply this domain change.")}</p>}
          <form onSubmit={submit}>
            <div className="grid gap-4 p-4">
              <Field label={t("Domain")} placeholder="app.example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} value={domain} onChange={(event) => setDomain(event.target.value)} hint={t("Optional. Leave empty to remove the domain.")} />
              {service.domain && <a className="text-meta text-seafoam underline" href={`https://${service.domain}`} target="_blank" rel="noreferrer">https://{service.domain}</a>}
            </div>
            <FormFooter error={error} saved={saved}>
              {domain !== (service.domain ?? "") ? <>
                <button className="btn" type="button" disabled={pending} onClick={() => void saveDomain()}>{pending ? t("Saving…") : t("Save domain")}</button>
                <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void saveDomain(true)}>{pending ? t("Saving and deploying…") : t("Save and deploy")}</button>
              </> : service.domain_pending ? <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void saveDomain(true)}>{pending ? t("Deploying…") : t("Deploy")}</button> : null}
            </FormFooter>
          </form>
        </AppSection>
      )}
      <AppSection title={t("Internal connection")} description={service.environment_name ? t("Reachable only by apps in the {environment} environment, on the same server.", { environment: service.environment_name }) : t("Reachable only by apps in this workspace without an environment, on the same server.")}>
        <div className="grid gap-4 p-4">
          {service.connection_url && <CopyField label={t("Connection URL")} value={service.connection_url} secret />}
          {service.internal_host && <CopyField label={t("Host")} value={service.internal_host} />}
          {service.port !== undefined && <CopyField label={t("Port")} value={String(service.port)} />}
        </div>
      </AppSection>
      <PublishedPorts />
      {deploying && <DeploymentDialog service={service} version={service.image_version} pending={pending} error={error} output={deployOutput} onForceStop={() => run(async () => {
        await api(`${path}/${service.id}/cancel-deploy`, { method: "POST" });
        onChanged();
      })} onClose={() => setDeploying(false)} />}
    </div>
  );
}
