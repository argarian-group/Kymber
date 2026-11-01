import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { sampleColumns } from "@/features/projects/sampleData";

export function TaskList() {
  return (
    <div className="grid gap-6">
      {sampleColumns.map((column) => (
        <section aria-label={column.name} key={column.name}>
          <h2 className="mb-1 flex items-center gap-2 font-medium">
            {column.name}
            <span className="text-meta font-normal text-ink-soft tabular-nums">{column.tasks.length}</span>
          </h2>
          <ul className="border-t border-line">
            {column.tasks.map((task) => (
              <li className="flex items-center gap-3 border-b border-line py-2.5" key={task.id}>
                <TaskPriority task={task} />
                <span className="min-w-0 flex-1 wrap-break-word">{task.title}</span>
                <span className={`flex-none text-meta ${task.late ? "font-medium text-ember" : "text-ink-soft"}`}>{task.due}</span>
                <TaskAssignee name={task.assignee} />
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
