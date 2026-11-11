import { TaskAssignees } from "@/features/projects/components/tasks/TaskAssignees";
import { TaskBadges } from "@/features/projects/components/tasks/TaskBadges";
import { TaskLabelChip } from "@/features/projects/components/tasks/TaskLabelChip";
import { type Task, type TaskLabel } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

type TaskCardProps = {
  task: Task;
  labels: TaskLabel[];
  done: boolean;
  dragging: boolean;
  onOpen: () => void;
  onDragStart: () => void;
  onDragEnd: () => void;
  lists: { id: string; name: string }[];
  index: number;
  count: number;
  onMove: (listId: string, position: number) => void;
};

export function TaskCard({ task, labels, done, dragging, onOpen, onDragStart, onDragEnd, lists, index, count, onMove }: TaskCardProps) {
  const taskLabels = labels.filter(({ id }) => task.label_ids.includes(id));
  return (
    <li
      className={`cursor-pointer rounded-control bg-raised p-3 transition-colors duration-100 hover:brightness-110 ${dragging ? "opacity-40" : ""}`}
      data-task-id={task.id}
      draggable
      onClick={onOpen}
      onDragStart={(event) => {
        event.dataTransfer.setData("text/plain", task.id);
        event.dataTransfer.effectAllowed = "move";
        onDragStart();
      }}
      onDragEnd={onDragEnd}
    >
      {taskLabels.length > 0 && (
        <div className="mb-2 flex flex-wrap gap-1">
          {taskLabels.map((label) => (
            <TaskLabelChip label={label} key={label.id} />
          ))}
        </div>
      )}
      <button className={`focus-ring block w-full cursor-pointer rounded-sm text-left font-medium wrap-break-word ${done ? "text-ink-soft" : ""}`} type="button">
        {task.title}
      </button>
      <div className="mt-2 flex flex-wrap items-center gap-1" onClick={(event) => event.stopPropagation()}>
        <button className="btn px-2" type="button" disabled={index === 0} aria-label={t("Move task up")} onClick={() => onMove(task.list_id, index - 1)}>↑</button>
        <button className="btn px-2" type="button" disabled={index === count - 1} aria-label={t("Move task down")} onClick={() => onMove(task.list_id, index + 1)}>↓</button>
        <select className="min-w-0 flex-1 rounded-control bg-sunken p-1 text-xs focus-ring" aria-label={t("Move task to list")} value={task.list_id} onChange={(event) => onMove(event.target.value, 0)}>
          {lists.map((list) => <option value={list.id} key={list.id}>{list.name}</option>)}
        </select>
      </div>
      <div className="mt-2 flex min-h-6 items-center justify-between gap-2">
        <TaskBadges task={task} done={done} />
        <TaskAssignees people={task.assignees} />
      </div>
    </li>
  );
}
