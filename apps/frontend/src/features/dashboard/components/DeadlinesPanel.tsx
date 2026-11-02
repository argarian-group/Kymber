import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { type OverviewTask } from "@/features/dashboard/overview";
import { dueLabel } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

const visibleDeadlines = 6;

export function DeadlinesPanel({ tasks }: { tasks: OverviewTask[] }) {
  const upcoming = tasks
    .filter(({ task, done }) => task.due_date && !done)
    .sort((a, b) => (a.task.due_date ?? "").localeCompare(b.task.due_date ?? ""))
    .slice(0, visibleDeadlines);

  return (
    <Panel title={t("Upcoming deadlines")}>
      {upcoming.length === 0 ? (
        <EmptyState title={t("No upcoming deadlines")} />
      ) : (
        <ul>
          {upcoming.map(({ task, project, done, late }) => (
            <li key={task.id}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2 md:px-6" to={`/projects/${project.id}/tasks?task=${task.id}`}>
                <TaskPriority task={task} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{task.title}</span>
                  <span className="block truncate text-meta text-ink-soft">{project.name}</span>
                </span>
                <span className={`text-right text-meta tabular-nums ${late ? "font-medium text-ember" : "text-ink-soft"}`}>{dueLabel(task, done)}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
