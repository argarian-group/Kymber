import { TaskAssignee } from "@/features/projects/components/tasks/TaskAssignee";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { dueLabel, isLate, type Task } from "@/features/projects/tasks";

export function TaskCard({ task, onOpen }: { task: Task; onOpen: () => void }) {
  return (
    <li
      className="cursor-pointer rounded-control bg-raised p-3 transition-colors duration-100 hover:bg-white/10"
      draggable
      onClick={onOpen}
      onDragStart={(event) => {
        event.dataTransfer.setData("text/plain", task.id);
        event.dataTransfer.effectAllowed = "move";
      }}
    >
      <button className="focus-ring block w-full cursor-pointer rounded-sm text-left font-medium wrap-break-word" type="button">
        {task.title}
      </button>
      <div className="mt-2 flex min-h-6 items-center gap-2 text-meta text-ink-soft">
        <TaskPriority task={task} />
        <span className={isLate(task) ? "font-medium text-ember" : ""}>{dueLabel(task)}</span>
        {task.assignee && (
          <span className="ml-auto">
            <TaskAssignee name={task.assignee.username} />
          </span>
        )}
      </div>
    </li>
  );
}
