import { useState } from "react";
import { TaskCard } from "@/features/projects/components/tasks/TaskCard";
import { taskStatuses, type Task, type TaskStatus } from "@/features/projects/tasks";

type TaskBoardProps = {
  tasks: Task[];
  onOpen: (task: Task) => void;
  onMove: (task: Task, status: TaskStatus) => void;
};

export function TaskBoard({ tasks, onOpen, onMove }: TaskBoardProps) {
  const [target, setTarget] = useState<TaskStatus | null>(null);

  return (
    <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {taskStatuses.map(({ id, name }) => {
        const column = tasks.filter(({ status }) => status === id);
        return (
          <section
            className={`min-h-32 min-w-0 rounded-panel bg-sunken p-3 outline-2 -outline-offset-2 ${target === id ? "outline-seafoam" : "outline-transparent"}`}
            aria-label={name}
            onDragOver={(event) => {
              event.preventDefault();
              setTarget(id);
            }}
            onDragLeave={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setTarget(null);
            }}
            onDrop={(event) => {
              event.preventDefault();
              setTarget(null);
              const task = tasks.find((item) => item.id === event.dataTransfer.getData("text/plain"));
              if (task) onMove(task, id);
            }}
            key={id}
          >
            <h2 className="mb-2 flex items-center justify-between px-1 font-medium">
              {name}
              <span className="text-meta font-normal text-ink-soft tabular-nums">{column.length}</span>
            </h2>
            <ul className="grid gap-2">
              {column.map((task) => (
                <TaskCard task={task} onOpen={() => onOpen(task)} key={task.id} />
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
