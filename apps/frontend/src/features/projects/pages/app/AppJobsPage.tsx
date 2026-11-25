import { useEffect, useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { language, t } from "@/lib/i18n";

type Job = { id: string; schedule: string; command: string; enabled: boolean; next_run_at: string; running: boolean; last_run_at: string | null; last_exit_code: number | null; last_output: string };

const timeFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium", timeStyle: "short" });

function JobRow({ job, url, onChanged }: { job: Job; url: string; onChanged: () => void }) {
  const [showOutput, setShowOutput] = useState(false);
  const { pending, error, run } = useAction();

  return (
    <li className="grid gap-3 border-t border-line p-4 first:border-t-0">
      <div className="flex flex-wrap items-start gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-60">
          <p className="font-mono text-xs text-ink-soft">{job.schedule} · UTC</p>
          <code className="mt-1 block font-mono text-meta wrap-break-word">{job.command}</code>
          <p className="mt-1 text-xs text-ink-soft">
            {job.running
              ? t("Running now")
              : job.last_run_at
                ? t("Last run {when}, exit code {code}", { when: formatRelative(job.last_run_at), code: job.last_exit_code ?? "?" })
                : t("Never run")}
            {job.enabled && ` · ${t("Next run {date}", { date: timeFormat.format(new Date(job.next_run_at)) })}`}
            {!job.enabled && ` · ${t("Paused")}`}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {job.last_run_at && <button className="btn" type="button" onClick={() => setShowOutput(!showOutput)}>{showOutput ? t("Hide output") : t("Output")}</button>}
          <button className="btn" type="button" disabled={pending || job.running} onClick={() => void run(async () => { await api(`${url}/${job.id}/run`, { method: "POST" }); onChanged(); })}>{t("Run now")}</button>
          <button className="btn" type="button" disabled={pending} onClick={() => void run(async () => { await api(`${url}/${job.id}`, { method: "PUT", body: { schedule: job.schedule, command: job.command, enabled: !job.enabled } }); onChanged(); })}>{job.enabled ? t("Pause") : t("Resume")}</button>
          <button className="btn" type="button" disabled={pending} onClick={() => { if (window.confirm(t("Delete this job?"))) void run(async () => { await api(`${url}/${job.id}`, { method: "DELETE" }); onChanged(); }); }}>{t("Delete")}</button>
        </div>
      </div>
      {error && <FormError message={error} />}
      {showOutput && <pre className={`max-h-80 overflow-auto rounded-control bg-sunken p-3 font-mono text-xs whitespace-pre-wrap ${job.last_exit_code === 0 ? "text-ink-soft" : "text-ember"}`}>{job.last_output || t("No output.")}</pre>}
    </li>
  );
}

export function AppJobsPage() {
  const { service, path } = useAppOutlet();
  const url = `${path}/${service.id}/jobs`;
  const [jobs, setJobs] = useState<Job[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const [schedule, setSchedule] = useState("0 3 * * *");
  const [command, setCommand] = useState("");
  const { pending, error, run } = useAction();
  const reload = () => setRevision((value) => value + 1);

  useLoadEffect(() => api<{ jobs: Job[] }>(url), (response) => { setJobs(response.jobs); setLoadError(null); }, setLoadError, [url, revision]);

  const anyRunning = jobs?.some((job) => job.running);
  useEffect(() => {
    if (!anyRunning) return;
    const timer = window.setTimeout(reload, 3000);
    return () => window.clearTimeout(timer);
  }, [anyRunning, jobs]);

  async function create(event: FormEvent) {
    event.preventDefault();
    const ok = await run(async () => {
      await api(url, { method: "POST", body: { schedule, command } });
    });
    if (ok) {
      setCommand("");
      reload();
    }
  }

  if (loadError && !jobs) return <LoadError title={t("Could not load jobs")} message={loadError} onRetry={reload} />;

  return (
    <div className="grid gap-6">
      <AppSection title={t("Scheduled jobs")} description={t("Commands that run inside this app's container on a cron schedule, in UTC. Each run stops after one hour.")}>
        {jobs && jobs.length === 0 && <p className="p-4 text-meta text-ink-soft">{t("No jobs yet.")}</p>}
        {jobs && jobs.length > 0 && <ul>{jobs.map((job) => <JobRow job={job} url={url} onChanged={reload} key={job.id} />)}</ul>}
      </AppSection>
      <AppSection title={t("New job")}>
        <form onSubmit={create}>
          <div className="grid items-start gap-4 p-4 sm:grid-cols-[200px_minmax(0,1fr)]">
            <Field label={t("Schedule")} required maxLength={100} className="field font-mono" hint={t("minute hour day month weekday, or @hourly, @daily")} value={schedule} onChange={(event) => setSchedule(event.target.value)} />
            <Field label={t("Command")} required maxLength={2000} className="field font-mono" placeholder="php artisan schedule:run" value={command} onChange={(event) => setCommand(event.target.value)} />
          </div>
          <FormFooter error={error} saved={null}>
            <button className="btn btn-primary" type="submit" disabled={pending || !command.trim()}>{pending ? t("Saving…") : t("Add job")}</button>
          </FormFooter>
        </form>
      </AppSection>
    </div>
  );
}
