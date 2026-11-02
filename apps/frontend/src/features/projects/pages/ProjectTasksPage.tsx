import { CalendarDays, Columns3, List, type LucideIcon } from "lucide-react";
import { lazy, Suspense } from "react";
import { useSearchParams } from "react-router";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { TaskBoard } from "@/features/projects/components/tasks/TaskBoard";
import { TaskCalendar } from "@/features/projects/components/tasks/TaskCalendar";
import { TaskFiltersBar } from "@/features/projects/components/tasks/TaskFiltersBar";
import { TaskList } from "@/features/projects/components/tasks/TaskList";
import { matchesFilters, type Task, type TaskFilters } from "@/features/projects/tasks";
import { useBoard } from "@/features/projects/useBoard";
import { t } from "@/lib/i18n";

type TaskView = "board" | "list" | "calendar";

const views: { id: TaskView; label: string; icon: LucideIcon }[] = [
  { id: "board", label: t("Board"), icon: Columns3 },
  { id: "list", label: t("List"), icon: List },
  { id: "calendar", label: t("Calendar"), icon: CalendarDays },
];

const TaskDetailDialog = lazy(() => import("@/features/projects/components/tasks/TaskDetailDialog").then(({ TaskDetailDialog: dialog }) => ({ default: dialog })));

const filterNames: (keyof TaskFilters)[] = ["search", "assignee", "label", "priority", "due"];

export function ProjectTasksPage() {
  const [params, setParams] = useSearchParams();
  const view = views.find(({ id }) => id === params.get("view"))?.id ?? "board";
  const filters = Object.fromEntries(filterNames.map((name) => [name, params.get(name) ?? ""])) as TaskFilters;
  const openTaskId = params.get("task");
  const { board, error, path, reload, createTask, moveTask, reorderLists, createList, updateList, deleteList } = useBoard();

  function setParam(name: string, value: string, replace = true) {
    setParams(
      (current) => {
        const next = new URLSearchParams(current);
        if (value) next.set(name, value);
        else next.delete(name);
        return next;
      },
      { replace },
    );
  }

  const openTask = (task: Task) => setParam("task", task.id, false);

  if (!board) {
    return error ? <LoadError title={t("Could not load tasks")} message={error} onRetry={reload} /> : <PageSkeleton />;
  }

  const doneLists = new Set(board.lists.filter(({ is_done }) => is_done).map(({ id }) => id));
  const isDone = (task: Task) => doneLists.has(task.list_id);
  const visible = board.tasks.filter((task) => matchesFilters(task, filters, isDone(task)));

  return (
    <div className="grid min-w-0 grid-cols-1 gap-4 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <TaskFiltersBar
          board={board}
          filters={filters}
          onChange={(name, value) => setParam(name, value)}
          onClear={() =>
            setParams(
              (current) => {
                const next = new URLSearchParams(current);
                filterNames.forEach((name) => next.delete(name));
                return next;
              },
              { replace: true },
            )
          }
        />
        <div className="flex gap-0.5 rounded-control bg-sunken p-0.5" role="group" aria-label={t("Task view")}>
          {views.map(({ id, label, icon: Icon }) => (
            <button
              className={`focus-ring flex min-h-9 cursor-pointer items-center gap-1.5 rounded-sm px-2.5 text-meta font-medium transition-colors duration-100 md:min-h-7 ${view === id ? "bg-raised text-ink" : "text-ink-soft hover:text-ink"}`}
              type="button"
              aria-pressed={view === id}
              onClick={() => setParam("view", id === "board" ? "" : id)}
              key={id}
            >
              <Icon aria-hidden="true" size={14} />
              {label}
            </button>
          ))}
        </div>
      </div>
      {error && (
        <p className="text-meta text-ember" role="alert">
          {error}
        </p>
      )}
      {view === "board" && <TaskBoard board={board} visible={visible} onOpen={openTask} onMoveTask={moveTask} onReorderLists={reorderLists} onCreateTask={createTask} onCreateList={createList} onUpdateList={updateList} onDeleteList={deleteList} />}
      {view === "list" && <TaskList board={board} visible={visible} onOpen={openTask} />}
      {view === "calendar" && <TaskCalendar tasks={visible} isDone={isDone} onOpen={openTask} />}
      {openTaskId && (
        <Suspense fallback={null}>
          <TaskDetailDialog taskId={openTaskId} board={board} path={path} onMove={moveTask} onChanged={reload} onClose={() => setParam("task", "")} key={openTaskId} />
        </Suspense>
      )}
    </div>
  );
}
