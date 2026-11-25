import { useEffect, useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { language, t } from "@/lib/i18n";

type Backup = { id: string; status: "running" | "restoring" | "succeeded" | "failed"; scheduled: boolean; size_bytes: number; error?: string; created_at: string; finished_at: string | null };
type Settings = { schedule: string; keep: number; next_run_at: string | null; supported: boolean; logical: boolean; reason?: string };

const timeFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium", timeStyle: "short" });
const sizeFormat = new Intl.NumberFormat(language, { maximumFractionDigits: 1 });

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${sizeFormat.format(bytes / 1024)} KB`;
  if (bytes < 1024 ** 3) return `${sizeFormat.format(bytes / 1024 ** 2)} MB`;
  return `${sizeFormat.format(bytes / 1024 ** 3)} GB`;
}

const statusLabels = { running: t("Backing up…"), restoring: t("Restoring…"), succeeded: t("Ready"), failed: t("Failed") };

function ScheduleForm({ settings, url, onSaved }: { settings: Settings; url: string; onSaved: () => void }) {
  const [schedule, setSchedule] = useState(settings.schedule);
  const [keep, setKeep] = useState(String(settings.keep));
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    if (await run(async () => { await api(`${url}/settings`, { method: "PUT", body: { schedule, keep: Number(keep) } }); })) {
      setSaved(t("Saved."));
      onSaved();
    }
  }

  return (
    <form onSubmit={submit}>
      <div className="grid items-start gap-4 p-4 sm:grid-cols-[minmax(0,1fr)_160px]">
        <Field label={t("Schedule")} maxLength={100} className="field font-mono" placeholder="0 3 * * *" hint={settings.schedule && settings.next_run_at ? t("Next backup {date}.", { date: timeFormat.format(new Date(settings.next_run_at)) }) : t("Cron in UTC, such as 0 3 * * * for every day at 03:00. Empty turns scheduled backups off.")} value={schedule} onChange={(event) => setSchedule(event.target.value)} />
        <Field label={t("Backups to keep")} type="number" required min={1} max={365} value={keep} onChange={(event) => setKeep(event.target.value)} />
      </div>
      <FormFooter error={error} saved={saved}>
        <button className="btn btn-primary" type="submit" disabled={pending || (schedule.trim() === settings.schedule && Number(keep) === settings.keep)}>{pending ? t("Saving…") : t("Save")}</button>
      </FormFooter>
    </form>
  );
}

function BackupRow({ backup, url, onChanged }: { backup: Backup; url: string; onChanged: () => void }) {
  const { pending, error, run } = useAction();
  const busy = backup.status === "running" || backup.status === "restoring";

  return (
    <li className="grid gap-2 border-t border-line p-4 first:border-t-0">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-56">
          <p className="text-meta font-medium">{timeFormat.format(new Date(backup.created_at))}</p>
          <p className="text-xs text-ink-soft">
            <span className={backup.status === "failed" ? "text-ember" : undefined}>{statusLabels[backup.status]}</span>
            {backup.status === "succeeded" && ` · ${formatSize(backup.size_bytes)}`}
            {` · ${backup.scheduled ? t("Scheduled") : t("Manual")}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {backup.status === "succeeded" && <a className="btn" href={`/api${url}/${backup.id}/download`} download>{t("Download")}</a>}
          {backup.status === "succeeded" && (
            <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Restore this backup? Current data is replaced by the backup's contents."))) void run(async () => { await api(`${url}/${backup.id}/restore`, { method: "POST" }); onChanged(); }); }}>{t("Restore")}</button>
          )}
          {!busy && <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Delete this backup?"))) void run(async () => { await api(`${url}/${backup.id}`, { method: "DELETE" }); onChanged(); }); }}>{t("Delete")}</button>}
        </div>
      </div>
      {backup.error && <p className="text-xs wrap-break-word text-ember">{backup.error}</p>}
      {error && <FormError message={error} />}
    </li>
  );
}

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
        description={settings.logical ? t("A database dump, compressed and kept on the app's server.") : t("A copy of the app's data directory, kept on the app's server. Extra volumes and mounts are not included.")}
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
