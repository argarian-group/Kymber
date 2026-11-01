import { AlignLeft, CheckSquare, MessageSquare } from "lucide-react";
import { TaskPriority } from "@/features/projects/components/tasks/TaskPriority";
import { dueLabel, isLate, type Task } from "@/features/projects/tasks";

export function TaskBadges({ task, done }: { task: Task; done: boolean }) {
  return (
    <span className="flex min-w-0 flex-wrap items-center gap-x-2.5 gap-y-1 text-meta text-ink-soft">
      <TaskPriority task={task} />
      {task.due_date && <span className={isLate(task, done) ? "font-medium text-ember" : ""}>{dueLabel(task, done)}</span>}
      {task.has_description && (
        <span title="Has a description">
          <AlignLeft aria-hidden="true" size={14} />
          <span className="sr-only">Has a description</span>
        </span>
      )}
      {task.checklist_total > 0 && (
        <span className={`flex items-center gap-1 tabular-nums ${task.checklist_done === task.checklist_total ? "text-seafoam" : ""}`} title="Checklist items done">
          <CheckSquare aria-hidden="true" size={14} />
          {task.checklist_done}/{task.checklist_total}
        </span>
      )}
      {task.comment_count > 0 && (
        <span className="flex items-center gap-1 tabular-nums" title="Comments">
          <MessageSquare aria-hidden="true" size={14} />
          {task.comment_count}
        </span>
      )}
    </span>
  );
}
