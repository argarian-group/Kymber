import { type FormEvent, useEffect, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { type TaskDetail } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { language, t } from "@/lib/i18n";

const dayFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium" });

function duration(seconds: number) {
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  return hours > 0 ? `${hours}h ${String(minutes % 60).padStart(2, "0")}m` : `${minutes}m`;
}

export function TaskTime({ detail, taskPath, path, onChanged }: { detail: TaskDetail; taskPath: string; path: string; onChanged: (task: TaskDetail) => void }) {
  const user = useCurrentUser();
  const [now, setNow] = useState(Date.now());
  const [minutes, setMinutes] = useState("");
  const { pending, error, run } = useAction();
  const running = detail.time_entries.find((entry) => !entry.ended_at && entry.user.id === user.id);
  const seconds = (entry: TaskDetail["time_entries"][number]) => Math.max(0, ((entry.ended_at ? Date.parse(entry.ended_at) : now) - Date.parse(entry.started_at)) / 1000);
  const total = detail.time_entries.reduce((sum, entry) => sum + seconds(entry), 0);

  useEffect(() => {
    if (!running) return;
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, [running]);

  const send = (url: string, body?: unknown) =>
    run(async () => {
      const { task } = await api<{ task: TaskDetail }>(url, { method: "POST", body });
      onChanged(task);
    });

  async function log(event: FormEvent) {
    event.preventDefault();
    if (await send(`${taskPath}/time-entries`, { minutes: Number(minutes) })) setMinutes("");
  }

  async function remove(id: string) {
    await run(async () => {
      await api(`${path}/task-time-entries/${id}`, { method: "DELETE" });
      onChanged({ ...detail, time_entries: detail.time_entries.filter((entry) => entry.id !== id) });
    });
  }

  return (
    <fieldset>
      <legend className="mb-1 text-meta font-medium">{t("Time")}</legend>
      <div className="flex items-center gap-2">
        <span className="flex-1 font-mono text-meta tabular-nums">{duration(total)}</span>
        <button className={`btn ${running ? "btn-primary" : ""}`} type="button" disabled={pending} onClick={() => void send(`${taskPath}/timer/${running ? "stop" : "start"}`)}>
          {running ? t("Stop {time}", { time: duration(seconds(running)) }) : t("Start timer")}
        </button>
      </div>
      <form className="mt-2 flex gap-2" onSubmit={log}>
        <input className="field min-w-0 flex-1" type="number" min={1} max={1440} required placeholder={t("Minutes")} aria-label={t("Minutes to log")} value={minutes} onChange={(event) => setMinutes(event.target.value)} />
        <button className="btn" type="submit" disabled={pending || !minutes}>{t("Log")}</button>
      </form>
      <FormError message={error} />
      {detail.time_entries.length > 0 && (
        <ul className="mt-2 grid gap-1 text-xs text-ink-soft">
          {detail.time_entries.map((entry) => (
            <li className="flex items-center gap-2" key={entry.id}>
              <span className="min-w-0 flex-1 truncate">{entry.user.username} · {dayFormat.format(new Date(entry.started_at))}</span>
              <span className="font-mono tabular-nums">{entry.ended_at ? duration(seconds(entry)) : t("running")}</span>
              {entry.user.id === user.id && entry.ended_at && (
                <button className="focus-ring cursor-pointer rounded-sm px-1 hover:text-ember" type="button" aria-label={t("Delete time entry")} disabled={pending} onClick={() => void remove(entry.id)}>×</button>
              )}
            </li>
          ))}
        </ul>
      )}
    </fieldset>
  );
}
