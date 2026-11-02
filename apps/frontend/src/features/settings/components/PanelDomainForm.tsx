import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

export function PanelDomainForm({ onComplete }: { onComplete?: () => void }) {
  const [domain, setDomain] = useState("");
  const [originalDomain, setOriginalDomain] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [dns, setDns] = useState<{ resolves: boolean; addresses: string[] } | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useLoadEffect(
    () => api<{ domain: string }>("/settings/panel-domain"),
    ({ domain: value }) => {
      setDomain(value);
      setOriginalDomain(value);
      setLoadError(null);
      setLoading(false);
    },
    (message) => { setLoadError(message); setLoading(false); },
    [attempt],
  );

  async function save(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      const response = await api<{ domain: string }>("/settings/panel-domain", { method: "PUT", body: { domain } });
      setDomain(response.domain);
      setOriginalDomain(domain.trim().toLowerCase());
      setSaved(t("Panel domain saved."));
      onComplete?.();
    });
  }

  function checkDNS() {
    return run(async () => {
      setDns(await api<{ resolves: boolean; addresses: string[] }>(`/settings/panel-domain/dns?domain=${encodeURIComponent(domain)}`));
    });
  }

  if (loadError) {
    return <div className="grid gap-3 p-4"><p className="text-meta text-ember" role="alert">{loadError}</p><button className="btn" type="button" onClick={() => { setLoading(true); setLoadError(null); setAttempt((value) => value + 1); }}>{t("Retry")}</button></div>;
  }

  return (
    <form onSubmit={save}>
      <div className="grid gap-4 p-4">
        <Field label={t("Panel domain")} placeholder="panel.example.com" autoCapitalize="none" autoCorrect="off" spellCheck={false} disabled={loading || pending} value={domain} onChange={(event) => { setDomain(event.target.value); setDns(null); }} hint={t("Optional. Set an A or AAAA record to this server's public IP; ports 80 and 443 must be reachable.")} />
        {domain.trim() && <button className="btn w-fit" type="button" disabled={loading || pending} onClick={checkDNS}>{t("Check DNS")}</button>}
        {dns && (
          <p className="text-meta text-ink-soft" role="status">
            {dns.resolves ? t("DNS resolves to {addresses}. Verify these addresses point to this server.", { addresses: dns.addresses.join(", ") }) : t("No DNS record found yet. Add an A or AAAA record for this domain, then check again.")}
          </p>
        )}
      </div>
      <FormFooter error={error} saved={saved}>
        {onComplete && <button className="btn" type="button" disabled={pending} onClick={onComplete}>{t("Skip")}</button>}
        <button className="btn btn-primary" type="submit" disabled={loading || pending || domain.trim().toLowerCase() === originalDomain}>{pending ? t("Saving…") : t("Save domain")}</button>
      </FormFooter>
    </form>
  );
}
