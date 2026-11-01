import { CalendarDays, Columns3, List, type LucideIcon } from "lucide-react";
import { useSearchParams } from "react-router";
import { TaskBoard } from "@/features/projects/components/tasks/TaskBoard";
import { TaskCalendar } from "@/features/projects/components/tasks/TaskCalendar";
import { TaskList } from "@/features/projects/components/tasks/TaskList";

type TaskView = "board" | "list" | "calendar";

const views: { id: TaskView; label: string; icon: LucideIcon }[] = [
  { id: "board", label: "Board", icon: Columns3 },
  { id: "list", label: "List", icon: List },
  { id: "calendar", label: "Calendar", icon: CalendarDays },
];

export function ProjectTasksPage() {
  const [params, setParams] = useSearchParams();
  const view = views.find(({ id }) => id === params.get("view"))?.id ?? "board";

  return (
    <div className="grid min-w-0 gap-4 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center justify-end gap-3">
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
      {view === "board" && <TaskBoard />}
      {view === "list" && <TaskList />}
      {view === "calendar" && <TaskCalendar />}
    </div>
  );
}
