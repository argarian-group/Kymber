import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { sampleColumns } from "@/features/projects/sampleData";

export function TaskBoard() {
  return (
    <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
      {sampleColumns.map((column) => (
        <section className="min-w-0 rounded-panel bg-sunken p-3" aria-label={column.name} key={column.name}>
          <h2 className="mb-2 flex items-center justify-between px-1 font-medium">
            {column.name}
            <span className="text-meta font-normal text-ink-soft tabular-nums">{column.tasks.length}</span>
          </h2>
          <ul className="grid gap-2">
            {column.tasks.map((task) => (
              <li className="rounded-control bg-raised p-3" key={task.id}>
                <p className="font-medium">{task.title}</p>
                <div className="mt-2 flex items-center gap-2 text-meta text-ink-soft">
                  <TaskPriority task={task} />
                  <span className={task.late ? "font-medium text-ember" : ""}>{task.due}</span>
                  <span className="ml-auto">
                    <TaskAssignee name={task.assignee} />
                  </span>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
