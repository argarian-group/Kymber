import { useEffect, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { BackupRow, ScheduleForm, UploadBackupButton, type Backup, type BackupSettings } from "@/features/projects/components/apps/Backups";
import { api } from "@/lib/api";
import { downloadText } from "@/lib/downloadText";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

const url = "/settings/platform-backups";

export function PlatformBackups() {
  const [data, setData] = useState<{ settings: BackupSettings; backups: Backup[] } | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const { pending, error, run } = useAction();
  const reload = () => setRevision((value) => value + 1);

  useLoadEffect(() => api<{ settings: BackupSettings; backups: Backup[] }>(url), (response) => { setData(response); setLoadError(null); }, setLoadError, [revision]);

  const busy = data?.backups.some((backup) => backup.status === "running");
  useEffect(() => {
    if (!busy) return;
    const timer = window.setTimeout(reload, 3000);
    return () => window.clearTimeout(timer);
  }, [busy, data]);

  async function restore(backup: Backup) {
    const key = backup.uploaded ? window.prompt(t("Recovery key of the installation this came from (empty if this one):")) : "";
    if (key === null) throw new Error(t("Restore cancelled."));
    await api(`${url}/${backup.id}/restore`, { method: "POST", body: { key } });
    window.location.reload();
  }

  if (loadError && !data) return <LoadError title={t("Could not load backups")} message={loadError} onRetry={reload} />;
  if (!data) return null;

  return (
    <div>
      <div className="grid gap-3 border-b border-line p-4">
        <div className="flex flex-wrap items-center gap-2">
          <button className="btn btn-primary" type="button" disabled={pending || busy} onClick={() => void run(async () => { await api(url, { method: "POST" }); reload(); })}>{t("Back up now")}</button>
          <UploadBackupButton url={url} disabled={busy} onUploaded={reload} />
          <button className="btn" type="button" disabled={pending} onClick={() => void run(async () => { const { key } = await api<{ key: string }>("/settings/encryption-key", { method: "POST" }); downloadText("kymber-encryption-key.txt", `${key}\n`); })}>{t("Download recovery key")}</button>
        </div>
        <p className="text-meta text-ink-soft">{t("Keep the recovery key off this server: backups need it to open secrets.")}</p>
        <FormError message={error} />
      </div>
      {data.backups.length === 0 ? <p className="p-4 text-meta text-ink-soft">{t("No backups yet.")}</p> : <ul>{data.backups.map((backup) => <BackupRow backup={backup} url={url} restore={() => restore(backup)} onChanged={reload} key={backup.id} />)}</ul>}
      <div className="border-t border-line">
        <ScheduleForm settings={data.settings} url={url} onSaved={reload} key={`${data.settings.schedule}-${data.settings.keep}`} />
      </div>
    </div>
  );
}
