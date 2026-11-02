import { useEffect, useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { PublishedPorts } from "@/features/projects/components/apps/PublishedPorts";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { t } from "@/lib/i18n";

export function AppConnectionPage() {
  const { detail, canWrite, path, onChanged } = useAppOutlet();
  const [domain, setDomain] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useEffect(() => setDomain(detail?.domain ?? ""), [detail?.id, detail?.domain]);

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail) return <PageSkeleton />;
  const service = detail;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      await api(`${path}/${service.id}/domain`, { method: "PUT", body: { domain } });
      setSaved(t("Domain saved. Deploy the app to apply the routing change."));
      onChanged();
    });
  }

  return (
    <div className="grid gap-5">
      {service.category === "application" && (
        <AppSection title={t("Public domain")} description={t("Optional. Requires DNS pointed to this machine and the local Traefik proxy. Leave unset to use the server's IPv4 address and published port.")}>
          <form onSubmit={submit}>
            <div className="grid gap-4 p-4">
              <Field label={t("Domain")} placeholder="app.example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} value={domain} onChange={(event) => setDomain(event.target.value)} hint={t("Optional. Leave empty to remove the domain.")} />
              {service.domain && <a className="text-meta text-seafoam underline" href={`https://${service.domain}`} target="_blank" rel="noreferrer">https://{service.domain}</a>}
            </div>
            <FormFooter error={error} saved={saved}>
              <button className="btn btn-primary" type="submit" disabled={pending || domain === (service.domain ?? "")}>{t("Save domain")}</button>
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
    </div>
  );
}
