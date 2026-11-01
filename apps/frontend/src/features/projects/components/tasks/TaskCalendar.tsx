import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";
import { CalendarTask, type DatedTask } from "@/features/projects/components/tasks/CalendarTask";
import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { sampleColumns } from "@/features/projects/sampleData";

function datedTasks(today: Date): DatedTask[] {
  return sampleColumns.flatMap((column) =>
    column.tasks.map((task) => ({
      ...task,
      done: column.name === "Done",
      date: new Date(today.getFullYear(), today.getMonth(), today.getDate() + task.dueOffset),
    })),
  );
}

function sameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

export function TaskCalendar() {
  const today = new Date();
  const [month, setMonth] = useState(new Date(today.getFullYear(), today.getMonth(), 1));
  const tasks = datedTasks(today);
  const year = month.getFullYear();
  const days = Array.from({ length: new Date(year, month.getMonth() + 1, 0).getDate() }, (_, index) => new Date(year, month.getMonth(), index + 1));
  const weekdays = Array.from({ length: 7 }, (_, index) => new Intl.DateTimeFormat(undefined, { weekday: "short" }).format(new Date(2023, 0, 1 + index)));
  const tasksOn = (day: Date) => tasks.filter((task) => sameDay(task.date, day));
  const daysWithTasks = days.filter((day) => tasksOn(day).length > 0);
  const step = (offset: number) => setMonth(new Date(year, month.getMonth() + offset, 1));

  return (
    <section aria-label="Calendar">
      <div className="mb-3 flex items-center gap-2">
        <h2 className="font-medium first-letter:uppercase" aria-live="polite">
          {new Intl.DateTimeFormat(undefined, { month: "long", year: "numeric" }).format(month)}
        </h2>
        <span className="ml-auto flex gap-1">
          <button className="btn px-2" type="button" onClick={() => step(-1)} aria-label="Previous month">
            <ChevronLeft aria-hidden="true" size={16} />
          </button>
          <button className="btn" type="button" onClick={() => setMonth(new Date(today.getFullYear(), today.getMonth(), 1))}>
            Today
          </button>
          <button className="btn px-2" type="button" onClick={() => step(1)} aria-label="Next month">
            <ChevronRight aria-hidden="true" size={16} />
          </button>
        </span>
      </div>

      <div className="hidden md:block">
        <div className="grid grid-cols-7 gap-1 pb-1 text-xs text-ink-soft" aria-hidden="true">
          {weekdays.map((weekday) => (
            <span className="px-1.5" key={weekday}>
              {weekday}
            </span>
          ))}
        </div>
        <ol className="grid grid-cols-7 gap-1">
          {days.map((day) => {
            const isToday = sameDay(day, today);
            return (
              <li className="min-h-24 min-w-0 rounded-control bg-sunken p-1.5" style={day.getDate() === 1 ? { gridColumnStart: day.getDay() + 1 } : undefined} key={day.getDate()}>
                <span className={`mb-1 flex size-6 items-center justify-center rounded-full text-xs tabular-nums ${isToday ? "bg-seafoam font-semibold text-on-accent" : "text-ink-soft"}`}>{day.getDate()}</span>
                <div className="grid grid-cols-1 gap-1">
                  {tasksOn(day).map((task) => (
                    <CalendarTask task={task} key={task.id} />
                  ))}
                </div>
              </li>
            );
          })}
        </ol>
      </div>

      <div className="md:hidden">
        {daysWithTasks.length === 0 ? (
          <p className="py-6 text-center text-meta text-ink-soft">No tasks this month.</p>
        ) : (
          <ol className="border-t border-line">
            {daysWithTasks.map((day) => (
              <li className="flex gap-3 border-b border-line py-2.5" key={day.getDate()}>
                <span className={`flex size-7 flex-none items-center justify-center rounded-full text-xs tabular-nums ${sameDay(day, today) ? "bg-seafoam font-semibold text-on-accent" : "bg-raised text-ink-soft"}`}>{day.getDate()}</span>
                <ul className="grid min-w-0 flex-1 gap-1.5">
                  {tasksOn(day).map((task) => (
                    <li className={`flex items-center gap-2 ${task.done ? "text-ink-faint line-through" : ""}`} key={task.id}>
                      <TaskPriority task={task} />
                      <span className="min-w-0 flex-1 wrap-break-word">{task.title}</span>
                      <TaskAssignee name={task.assignee} />
                    </li>
                  ))}
                </ul>
              </li>
            ))}
          </ol>
        )}
      </div>
    </section>
  );
}
