import { useEffect, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { BackupRow, ScheduleForm, type Backup, type BackupSettings } from "@/features/projects/components/apps/Backups";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type Settings = BackupSettings & { supported: boolean; logical: boolean; reason?: string };

export function AppBackupsPage() {
  const { service, path } = useAppOutlet();
  const url = `${path}/${service.id}/backups`;
  const [data, setData] = useState<{ settings: Settings; backups: Backup[] } | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { pending, error, run } = useAction();
  const reload = () => setRevision((value) => value + 1);

  useLoadEffect(() => api<{ settings: Settings; backups: Backup[] }>(url), (response) => { setData(response); setLoadError(null); }, setLoadError, [url, revision]);

  const busy = data?.backups.some((backup) => backup.status === "running" || backup.status === "restoring");
  useEffect(() => {
    if (!busy) return;
    const timer = window.setTimeout(reload, 3000);
    return () => window.clearTimeout(timer);
  }, [busy, data]);

  if (loadError && !data) return <LoadError title={t("Could not load backups")} message={loadError} onRetry={reload} />;
  if (!data) return null;
  const { settings, backups } = data;

  return (
    <div className="grid gap-6">
      <AppSection
        title={t("Backups")}
        description={settings.logical ? t("A database dump, compressed and kept on the app's server.") : service.compose ? t("A copy of every named volume in the stack, taken while it runs, kept on the app's server. Databases inside the stack may need their own dump to be consistent.") : t("A copy of the app's volumes, kept on the app's server. Server folders mounted into the app are not included.")}
      >
        {!settings.supported && <p className="p-4 text-meta text-ink-soft">{settings.reason}</p>}
        {settings.supported && (
          <div className="flex flex-wrap items-center gap-3 border-b border-line p-4">
            <button className="btn btn-primary" type="button" disabled={pending || busy} onClick={() => void run(async () => { await api(url, { method: "POST" }); reload(); })}>{t("Back up now")}</button>
            <FormError message={error} />
          </div>
        )}
        {backups.length === 0 ? <p className="p-4 text-meta text-ink-soft">{t("No backups yet.")}</p> : <ul>{backups.map((backup) => <BackupRow backup={backup} url={url} onChanged={reload} key={backup.id} />)}</ul>}
      </AppSection>
      {settings.supported && (
        <AppSection title={t("Scheduled backups")} description={t("Older backups beyond the limit are deleted after each successful backup.")}>
          <ScheduleForm settings={settings} url={url} onSaved={reload} key={`${settings.schedule}-${settings.keep}`} />
        </AppSection>
      )}
    </div>
  );
}
