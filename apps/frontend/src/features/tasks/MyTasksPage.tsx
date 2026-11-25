import { useState } from "react";
import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { priorities } from "@/features/dashboard/priorities";
import { PriorityBars } from "@/features/dashboard/components/PriorityBars";
import { useProjects } from "@/features/projects/ProjectsContext";
import { dueLabel, isLate } from "@/features/projects/tasks";
import { taskLink, type WorkspaceTask } from "@/features/tasks/workspaceTasks";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

export function MyTasksPage() {
  const { state } = useProjects();
  const workspaceId = state.status === "ready" ? state.workspace?.id : undefined;
  const [tasks, setTasks] = useState<WorkspaceTask[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [showDone, setShowDone] = useState(false);

  useLoadEffect(
    () => (workspaceId ? api<{ tasks: WorkspaceTask[] }>(`/workspaces/${workspaceId}/tasks?mine=1`) : Promise.resolve({ tasks: [] })),
    (response) => { setTasks(response.tasks); setError(null); },
    setError,
    [workspaceId, attempt],
  );

  if (error && !tasks) return <LoadError title={t("Could not load tasks")} message={error} onRetry={() => setAttempt((value) => value + 1)} />;
  if (!tasks) return <PageSkeleton />;
  const open = tasks.filter(({ done }) => !done);
  const done = tasks.length - open.length;
  const shown = showDone ? tasks : open;

  return (
    <>
      <PageHeader
        title={t("My tasks")}
        description={open.length === 1 ? t("1 open task assigned to you") : t("{count} open tasks assigned to you", { count: open.length })}
        actions={done > 0 && <button className="btn" type="button" aria-pressed={showDone} onClick={() => setShowDone(!showDone)}>{showDone ? t("Hide done") : t("Show done ({count})", { count: done })}</button>}
      />
      <div className="px-4 py-4 md:px-6">
        {shown.length === 0 ? (
          <EmptyState title={t("Nothing assigned to you")} description={t("Tasks assigned to you in any project show up here.")} />
        ) : (
          <ul className="border-t border-line">
            {shown.map((task) => {
              const priority = priorities[task.priority];
              const late = isLate(task, task.done);
              return (
                <li className="border-b border-line" key={task.id}>
                  <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 py-2.5" to={taskLink(task)}>
                    <span title={t("{priority} priority", { priority: priority.label })}><PriorityBars level={priority.level} tone={priority.tone} /></span>
                    <span className="min-w-0">
                      <span className={`block wrap-break-word font-medium ${task.done ? "text-ink-faint line-through" : ""}`}>{task.title}</span>
                      <span className="block truncate text-meta text-ink-soft"><span className="font-mono text-xs">{task.key}</span> · {task.project_name} · {task.list_name}</span>
                    </span>
                    <span className={`text-right text-meta tabular-nums ${late ? "font-medium text-ember" : "text-ink-soft"}`}>{dueLabel(task, task.done)}</span>
                  </Link>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </>
  );
}
