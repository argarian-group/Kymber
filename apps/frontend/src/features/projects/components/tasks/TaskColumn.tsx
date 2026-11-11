import { CheckCircle2, MoreHorizontal } from "lucide-react";
import { Fragment, type DragEvent } from "react";
import { AddInline } from "@/features/projects/components/tasks/AddInline";
import { TaskCard } from "@/features/projects/components/tasks/TaskCard";
import { TaskDropLine } from "@/features/projects/components/tasks/TaskDropLine";
import { type Task, type TaskLabel, type TaskListColumn } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

type TaskColumnProps = {
  list: TaskListColumn;
  tasks: Task[];
  orderedTasks: Task[];
  total: number;
  labels: TaskLabel[];
  draggingTaskId: string | null;
  dropBefore: string | null | undefined;
  listDropSide: "before" | "after" | null;
  onOpen: (task: Task) => void;
  onEdit: () => void;
  onAdd: (title: string) => Promise<void>;
  onCardDragStart: (task: Task) => void;
  onHeaderDragStart: () => void;
  onDragEnd: () => void;
  onDragOver: (event: DragEvent<HTMLElement>) => void;
  onDrop: (event: DragEvent<HTMLElement>) => void;
  lists: { id: string; name: string }[];
  onMoveTask: (taskId: string, listId: string, position: number) => void;
};

export function TaskColumn({ list, tasks, orderedTasks, total, labels, draggingTaskId, dropBefore, listDropSide, onOpen, onEdit, onAdd, onCardDragStart, onHeaderDragStart, onDragEnd, onDragOver, onDrop, lists, onMoveTask }: TaskColumnProps) {
  return (
    <section
      className={`flex max-h-full w-[85vw] flex-none snap-start flex-col rounded-panel bg-sunken p-2 outline-2 -outline-offset-2 outline-transparent sm:w-72 ${listDropSide === "before" ? "border-l-2 border-seafoam" : listDropSide === "after" ? "border-r-2 border-seafoam" : ""}`}
      aria-label={list.name}
      data-list-id={list.id}
      onDragEnter={onDragOver}
      onDragOver={onDragOver}
      onDrop={onDrop}
    >
      <div
        className="mb-2 flex cursor-grab items-center gap-2 py-1 pr-1 pl-2 active:cursor-grabbing"
        draggable
        onDragStart={(event) => {
          event.dataTransfer.setData("text/plain", list.id);
          event.dataTransfer.effectAllowed = "move";
          onHeaderDragStart();
        }}
        onDragEnd={onDragEnd}
      >
        {list.is_done && <CheckCircle2 className="flex-none text-seafoam" aria-label={t("Done list")} size={14} />}
        <h2 className="min-w-0 flex-1 truncate font-medium">{list.name}</h2>
        <span className="text-meta text-ink-soft tabular-nums">{tasks.length === total ? total : `${tasks.length}/${total}`}</span>
        <button className="btn min-h-7 border-transparent px-1.5" type="button" aria-label={t("Edit list {name}", { name: list.name })} onClick={onEdit}>
          <MoreHorizontal aria-hidden="true" size={16} />
        </button>
      </div>
      <ul className="grid min-h-2 gap-2 overflow-y-auto">
        {tasks.map((task) => (
          <Fragment key={task.id}>
            {dropBefore === task.id && <TaskDropLine />}
            <TaskCard task={task} labels={labels} done={list.is_done} dragging={draggingTaskId === task.id} onOpen={() => onOpen(task)} onDragStart={() => onCardDragStart(task)} onDragEnd={onDragEnd} lists={lists} index={orderedTasks.findIndex(({ id }) => id === task.id)} count={total} onMove={(listId, position) => onMoveTask(task.id, listId, position)} />
          </Fragment>
        ))}
        {dropBefore === null && <TaskDropLine />}
      </ul>
      <div className="mt-2">
        <AddInline label={t("Add task")} placeholder={t("Task title")} maxLength={200} keepOpen onAdd={onAdd} />
      </div>
    </section>
  );
}
