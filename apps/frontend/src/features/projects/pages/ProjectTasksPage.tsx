import { CalendarDays, Columns3, List, type LucideIcon } from "lucide-react";
import { useState } from "react";
import { useSearchParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { TaskBoard } from "@/features/projects/components/tasks/TaskBoard";
import { TaskCalendar } from "@/features/projects/components/tasks/TaskCalendar";
import { TaskDialog } from "@/features/projects/components/tasks/TaskDialog";
import { TaskList } from "@/features/projects/components/tasks/TaskList";
import { type Task } from "@/features/projects/tasks";
import { useTasks } from "@/features/projects/useTasks";

type TaskView = "board" | "list" | "calendar";

const views: { id: TaskView; label: string; icon: LucideIcon }[] = [
  { id: "board", label: "Board", icon: Columns3 },
  { id: "list", label: "List", icon: List },
  { id: "calendar", label: "Calendar", icon: CalendarDays },
];

export function ProjectTasksPage() {
  const [params, setParams] = useSearchParams();
  const view = views.find(({ id }) => id === params.get("view"))?.id ?? "board";
  const { tasks, error, reload, createTask, updateTask, deleteTask, moveTask } = useTasks();
  const [dialog, setDialog] = useState<{ task: Task | null } | null>(null);
  const openTask = (task: Task) => setDialog({ task });

  if (!tasks) {
    return error ? <LoadError title="Could not load tasks" message={error} onRetry={reload} /> : <PageSkeleton />;
  }

  return (
    <div className="grid min-w-0 gap-4 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <button className="btn btn-primary" type="button" onClick={() => setDialog({ task: null })}>
          New task
        </button>
        <div className="flex gap-0.5 rounded-control bg-sunken p-0.5" role="group" aria-label="Task view">
          {views.map(({ id, label, icon: Icon }) => (
            <button
              className={`focus-ring flex min-h-9 cursor-pointer items-center gap-1.5 rounded-sm px-2.5 text-meta font-medium transition-colors duration-100 md:min-h-7 ${view === id ? "bg-raised text-ink" : "text-ink-soft hover:text-ink"}`}
              type="button"
              aria-pressed={view === id}
              onClick={() => setParams(id === "board" ? {} : { view: id }, { replace: true })}
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
      {tasks.length === 0 && view !== "calendar" ? (
        <EmptyState title="No tasks yet" description="Create the first task for this project." />
      ) : (
        <>
          {view === "board" && <TaskBoard tasks={tasks} onOpen={openTask} onMove={moveTask} />}
          {view === "list" && <TaskList tasks={tasks} onOpen={openTask} />}
          {view === "calendar" && <TaskCalendar tasks={tasks} onOpen={openTask} />}
        </>
      )}
      {dialog && (
        <TaskDialog
          task={dialog.task}
          onSave={(input) => (dialog.task ? updateTask(dialog.task.id, input) : createTask(input))}
          onDelete={() => (dialog.task ? deleteTask(dialog.task.id) : Promise.resolve())}
          onClose={() => setDialog(null)}
          key={dialog.task?.id ?? "new"}
        />
      )}
    </div>
  );
}
