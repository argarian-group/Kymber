import { Plus, X } from "lucide-react";
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

type DomainRow = { id: number; host: string; port: string };

function parseDomains(value: string | undefined): DomainRow[] {
  return (value ?? "").split(",").filter(Boolean).map((entry, id) => {
    const [host, port = ""] = entry.split(":");
    return { id, host, port };
  });
}

function formatDomains(rows: DomainRow[]) {
  return rows.filter(({ host }) => host.trim()).map(({ host, port }) => (port ? `${host.trim()}:${port}` : host.trim())).join(",");
}

export function AppConnectionPage() {
  const { detail, template, canWrite, path, onChanged, deployOutput } = useAppOutlet();
  const [domains, setDomains] = useState<DomainRow[]>([]);
  const [port, setPort] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const [deploying, setDeploying] = useState(false);
  const { pending, error, run } = useAction();

  useEffect(() => setDomains(parseDomains(detail?.domain)), [detail?.id, detail?.domain]);
  useEffect(() => setPort(detail?.port ? String(detail.port) : ""), [detail?.id, detail?.port]);

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail) return <PageSkeleton />;
  const service = detail;
  const portEditable = template?.custom === true;
  const domain = formatDomains(domains);
  const routedDomains = parseDomains(service.domain);
  const changed = domain !== (service.domain ?? "") || (portEditable && Number(port) !== (service.port ?? 0));

  async function saveDomain(deploy = false) {
    setSaved(null);
    await run(async () => {
      if (portEditable && Number(port) !== (service.port ?? 0)) {
        await api(`${path}/${service.id}`, { method: "PUT", body: { port: Number(port) } });
        setSaved(t("Saved. Applies on the next deploy."));
      }
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
        <AppSection title={portEditable ? t("Domain and port") : t("Public domain")} description={t("Optional. Point DNS to this machine. Traefik reaches the app over Docker networking, so no host port needs to be published.")}>
          {service.domain_pending && <p className="border-b border-line p-4 text-meta text-seafoam" role="status">{t("Deploy the app to apply this domain change.")}</p>}
          <form onSubmit={submit}>
            <div className="grid items-start gap-4 p-4 sm:grid-cols-[1fr_12rem]">
              <div className="grid gap-2">
                <span className="text-meta font-medium">{t("Domains")}</span>
                {domains.map((row) => (
                  <div className={`grid items-center gap-2 ${service.compose ? "grid-cols-[1fr_auto]" : "grid-cols-[1fr_6.5rem_auto]"}`} key={row.id}>
                    <input className="field" aria-label={t("Domain")} placeholder="app.example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} required value={row.host} onChange={(event) => setDomains(domains.map((item) => (item.id === row.id ? { ...item, host: event.target.value } : item)))} />
                    {!service.compose && <input className="field" type="number" min={1} max={65535} aria-label={t("Port for {domain}", { domain: row.host || t("domain") })} placeholder={port || t("Port")} value={row.port} onChange={(event) => setDomains(domains.map((item) => (item.id === row.id ? { ...item, port: event.target.value } : item)))} />}
                    <button className="btn px-2" type="button" aria-label={t("Remove {name}", { name: row.host || t("domain") })} onClick={() => setDomains(domains.filter((item) => item.id !== row.id))}>
                      <X aria-hidden="true" size={16} />
                    </button>
                  </div>
                ))}
                <button className="btn w-fit" type="button" onClick={() => setDomains([...domains, { id: Date.now(), host: "", port: "" }])}>
                  <Plus aria-hidden="true" size={16} />
                  {t("Add domain")}
                </button>
                {!service.compose && domains.length > 0 && <p className="text-xs text-ink-soft">{t("Leave a domain's port empty to use the app port.")}</p>}
                {routedDomains.length > 0 && <div className="flex flex-wrap gap-x-4 gap-y-1">{routedDomains.map(({ host }) => <a className="text-meta text-seafoam underline" key={host} href={`https://${host}`} target="_blank" rel="noreferrer">https://{host}</a>)}</div>}
              </div>
              {portEditable && <Field label={t("App port")} type="number" min={1} max={65535} required={domains.some((row) => row.host && !row.port)} placeholder="3000" value={port} onChange={(event) => setPort(event.target.value)} hint={t("Where the domains send traffic.")} />}
            </div>
            <FormFooter error={error} saved={saved}>
              {changed ? <>
                <button className="btn" type="submit" disabled={pending}>{pending ? t("Saving…") : t("Save")}</button>
                <button className="btn btn-primary" type="button" disabled={pending} onClick={(event) => { if (event.currentTarget.form?.reportValidity()) void saveDomain(true); }}>{pending ? t("Saving and deploying…") : t("Save and deploy")}</button>
              </> : service.domain_pending ? <button className="btn btn-primary" type="button" disabled={pending} onClick={() => void saveDomain(true)}>{pending ? t("Deploying…") : t("Deploy")}</button> : null}
            </FormFooter>
          </form>
        </AppSection>
      )}
      <AppSection title={t("Internal connection")} description={service.environment_name ? t("Reachable only by apps in the {environment} environment, on the same server.", { environment: service.environment_name }) : t("Reachable only by apps in this workspace without an environment, on the same server.")}>
        <div className="grid gap-4 p-4">
          {service.connection_url && <CopyField label={t("Connection URL")} value={service.connection_url} secret />}
          {service.internal_host && <CopyField label={t("Host")} value={service.internal_host} />}
          {portEditable ? <p className="text-meta text-ink-soft">{t("Other apps reach it at this host on any port it listens on.")}</p> : service.port !== undefined && <CopyField label={t("Port")} value={String(service.port)} />}
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
