import { useEffect, useState, type FormEvent } from "react";
import { Navigate } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useAppOutlet } from "@/features/projects/useAppOutlet";

export function AppConnectionPage() {
  const { detail, canWrite, path, onChanged } = useAppOutlet();
  const [domain, setDomain] = useState("");
  const [publicPort, setPublicPort] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useEffect(() => { setDomain(detail?.domain ?? ""); setPublicPort(detail?.public_port ? String(detail.public_port) : ""); }, [detail?.id, detail?.domain, detail?.public_port]);

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail) return <PageSkeleton />;
  const service = detail;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      await api(`${path}/${service.id}/domain`, { method: "PUT", body: { domain } });
      setSaved("Domain saved. Deploy the app to apply the routing change.");
      onChanged();
    });
  }

  async function submitPublicPort(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      const response = await api<{ public_port: number }>(`${path}/${service.id}/public-port`, { method: "PUT", body: { port: Number(publicPort) || 0 } });
      setPublicPort(response.public_port ? String(response.public_port) : "");
      setSaved("Public port saved. Deploy the app to apply the change.");
      onChanged();
    });
  }

  return (
    <div className="grid gap-5">
      {service.category === "application" && (
        <AppSection title="Public domain" description="Optional. Requires DNS pointed to this machine and the local Traefik proxy. Leave unset to use the server's IPv4 address and published port.">
          <form onSubmit={submit}>
            <div className="grid gap-4 p-4">
              <Field label="Domain" placeholder="app.example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} disabled={pending || Boolean(publicPort)} value={domain} onChange={(event) => setDomain(event.target.value)} hint="Optional; leave empty to use a public port instead." />
              {service.domain && <a className="text-meta text-seafoam underline" href={`https://${service.domain}`} target="_blank" rel="noreferrer">https://{service.domain}</a>}
            </div>
            <FormFooter error={error} saved={saved}>
              <button className="btn btn-primary" type="submit" disabled={pending || domain === (service.domain ?? "") || Boolean(domain && publicPort)}>Save domain</button>
            </FormFooter>
          </form>
        </AppSection>
      )}
      {service.category === "application" && <AppSection title="Public port" description="Expose this app directly on the server's IPv4 address, without needing another container on its Docker network. Only use this for trusted apps; the port is reachable by anyone who can reach this server.">
        <form onSubmit={submitPublicPort}>
            <div className="grid gap-4 p-4">
            <Field label="Host port" type="number" min={0} max={65535} disabled={pending || Boolean(domain)} value={publicPort} onChange={(event) => setPublicPort(event.target.value)} hint={`Choose an unused port from 1–65535. Container HTTP port: ${service.port ?? "not configured"}. Set 0 or leave empty to disable.`} />
            {service.public_port && <CopyField label="Public URL" value={`http://${window.location.hostname}:${service.public_port}`} />}
          </div>
          <FormFooter error={error} saved={saved}><button className="btn btn-primary" type="submit" disabled={pending || (Number(publicPort) || 0) === (service.public_port ?? 0) || Boolean(publicPort && domain)}>Save public port</button></FormFooter>
        </form>
      </AppSection>}
      <AppSection title="Internal connection" description={service.environment_name ? `Reachable only by apps in the ${service.environment_name} environment, on the same server.` : "Reachable only by apps in this workspace without an environment, on the same server."}>
        <div className="grid gap-4 p-4">
          {service.connection_url && <CopyField label="Connection URL" value={service.connection_url} secret />}
          {service.internal_host && <CopyField label="Host" value={service.internal_host} />}
          {service.port !== undefined && <CopyField label="Port" value={String(service.port)} />}
        </div>
      </AppSection>
    </div>
  );
}
