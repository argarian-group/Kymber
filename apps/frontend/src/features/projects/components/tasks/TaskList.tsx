import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { dueLabel, isLate, taskStatuses, type Task } from "@/features/projects/tasks";

export function TaskList({ tasks, onOpen }: { tasks: Task[]; onOpen: (task: Task) => void }) {
  return (
    <div className="grid gap-6">
      {taskStatuses.map(({ id, name }) => {
        const section = tasks.filter(({ status }) => status === id);
        return (
          <section aria-label={name} key={id}>
            <h2 className="mb-1 flex items-center gap-2 font-medium">
              {name}
              <span className="text-meta font-normal text-ink-soft tabular-nums">{section.length}</span>
            </h2>
            <ul className="border-t border-line">
              {section.map((task) => (
                <li className="border-b border-line" key={task.id}>
                  <button className="focus-ring flex min-h-11 w-full cursor-pointer items-center gap-3 py-2.5 text-left transition-colors duration-100 hover:bg-white/5" type="button" onClick={() => onOpen(task)}>
                    <TaskPriority task={task} />
                    <span className="min-w-0 flex-1 wrap-break-word">{task.title}</span>
                    <span className={`flex-none text-meta ${isLate(task) ? "font-medium text-ember" : "text-ink-soft"}`}>{dueLabel(task)}</span>
                    {task.assignee && <TaskAssignee name={task.assignee.username} />}
                  </button>
                </li>
              ))}
            </ul>
          </section>
        );
      })}
    </div>
  );
}
