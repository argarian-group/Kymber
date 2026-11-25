import { useState } from "react";
import { LoadError } from "@/components/ui/LoadError";
import { type Board, type Task } from "@/features/projects/tasks";
import { api, errorMessage } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

export function ArchivedTasks({ path, revision, onOpen, onRestored }: { path: string; revision: number; onOpen: (task: Task) => void; onRestored: () => void }) {
  const [tasks, setTasks] = useState<Task[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useLoadEffect(() => api<Board>(`${path}/board?archived=1`), (board) => { setTasks(board.tasks); setError(null); }, setError, [path, revision, attempt]);

  async function restore(task: Task) {
    try {
      await api(`${path}/tasks/${task.id}/restore`, { method: "POST" });
      onRestored();
      setAttempt((value) => value + 1);
    } catch (failure) {
      setError(errorMessage(failure));
    }
  }

  if (error && !tasks) return <LoadError title={t("Could not load tasks")} message={error} onRetry={() => setAttempt((value) => value + 1)} />;
  if (!tasks) return null;
  if (tasks.length === 0) return <p className="text-meta text-ink-soft">{t("No archived tasks.")}</p>;

  return (
    <ul className="border-t border-line">
      {error && <li className="py-2 text-meta text-ember" role="alert">{error}</li>}
      {[...tasks].sort((a, b) => (b.archived_at ?? "").localeCompare(a.archived_at ?? "")).map((task) => (
        <li className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-line py-2" key={task.id}>
          <button className="focus-ring flex min-h-9 min-w-0 flex-1 basis-60 cursor-pointer items-center gap-3 rounded-sm text-left hover:text-ink" type="button" onClick={() => onOpen(task)}>
            <span className="font-mono text-xs text-ink-faint">{task.key}</span>
            <span className="min-w-0 wrap-break-word">{task.title}</span>
          </button>
          {task.archived_at && <span className="text-xs text-ink-soft">{t("Archived {when}", { when: formatRelative(task.archived_at) })}</span>}
          <button className="btn" type="button" onClick={() => void restore(task)}>{t("Restore")}</button>
        </li>
      ))}
    </ul>
  );
}
