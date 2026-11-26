import { useRef, useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { language, t } from "@/lib/i18n";

export type Backup = { id: string; status: "running" | "restoring" | "succeeded" | "failed"; scheduled: boolean; uploaded: boolean; remote: boolean; size_bytes: number; error?: string; created_at: string; finished_at: string | null };
export type BackupSettings = { schedule: string; keep: number; next_run_at: string | null };

const timeFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium", timeStyle: "short" });
const sizeFormat = new Intl.NumberFormat(language, { maximumFractionDigits: 1 });

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 ** 2) return `${sizeFormat.format(bytes / 1024)} KB`;
  if (bytes < 1024 ** 3) return `${sizeFormat.format(bytes / 1024 ** 2)} MB`;
  return `${sizeFormat.format(bytes / 1024 ** 3)} GB`;
}

const statusLabels = { running: t("Backing up…"), restoring: t("Restoring…"), succeeded: t("Ready"), failed: t("Failed") };

export function ScheduleForm({ settings, url, onSaved }: { settings: BackupSettings; url: string; onSaved: () => void }) {
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

export function BackupRow({ backup, url, restore, onChanged }: { backup: Backup; url: string; restore?: () => Promise<unknown>; onChanged: () => void }) {
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
            {` · ${backup.uploaded ? t("Uploaded") : backup.scheduled ? t("Scheduled") : t("Manual")}`}
            {backup.remote && ` · ${t("Copied to S3")}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {backup.status === "succeeded" && <a className="btn" href={`/api${url}/${backup.id}/download`} download>{t("Download")}</a>}
          {backup.status === "succeeded" && (
            <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Restore this backup? Current data is replaced by the backup's contents."))) void run(async () => { await (restore ? restore() : api(`${url}/${backup.id}/restore`, { method: "POST" })); onChanged(); }); }}>{t("Restore")}</button>
          )}
          {!busy && <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Delete this backup?"))) void run(async () => { await api(`${url}/${backup.id}`, { method: "DELETE" }); onChanged(); }); }}>{t("Delete")}</button>}
        </div>
      </div>
      {backup.error && <p className="text-xs wrap-break-word text-ember">{backup.error}</p>}
      {error && <FormError message={error} />}
    </li>
  );
}

export function UploadBackupButton({ url, paths = [], disabled, onUploaded }: { url: string; paths?: string[]; disabled?: boolean; onUploaded: () => void }) {
  const input = useRef<HTMLInputElement>(null);
  const [target, setTarget] = useState("");
  const { pending, error, run } = useAction();

  function upload(file: File | undefined) {
    if (!file) return;
    const query = target ? `?path=${encodeURIComponent(target)}` : "";
    void run(async () => {
      await api(`${url}/upload${query}`, { method: "POST", file });
      onUploaded();
    });
  }

  return (
    <>
      {paths.length > 1 && (
        <label className="flex items-center gap-2 text-meta text-ink-soft">
          {t("Files go to")}
          <select className="field w-auto cursor-pointer" value={target} onChange={(event) => setTarget(event.target.value)}>
            <option value="">{t("Detect from the file")}</option>
            {paths.map((value) => <option value={value} key={value}>{value}</option>)}
          </select>
        </label>
      )}
      <button className="btn" type="button" disabled={disabled || pending} onClick={() => input.current?.click()}>{pending ? t("Uploading…") : t("Upload backup")}</button>
      <input className="sr-only" ref={input} type="file" tabIndex={-1} aria-hidden="true" onChange={(event) => { upload(event.target.files?.[0]); event.target.value = ""; }} />
      <FormError message={error} />
    </>
  );
}
