import { type DragEvent, useState } from "react";
import { AddInline } from "@/features/projects/components/tasks/AddInline";
import { TaskColumn } from "@/features/projects/components/tasks/TaskColumn";
import { TaskListDialog } from "@/features/projects/components/tasks/TaskListDialog";
import { positionBefore, tasksInList, type Board, type Task, type TaskListColumn } from "@/features/projects/tasks";

type TaskBoardProps = {
  board: Board;
  visible: Task[];
  onOpen: (task: Task) => void;
  onMoveTask: (taskId: string, listId: string, position: number) => void;
  onReorderLists: (ids: string[]) => void;
  onCreateTask: (listId: string, title: string) => Promise<void>;
  onCreateList: (name: string) => Promise<void>;
  onUpdateList: (id: string, input: { name: string; is_done: boolean }) => Promise<void>;
  onDeleteList: (id: string) => Promise<void>;
};

type Drag = { kind: "task" | "list"; id: string };
type Target = { listId: string; beforeTask: string | null } | { listId: string; side: "before" | "after" };

export function TaskBoard({ board, visible, onOpen, onMoveTask, onReorderLists, onCreateTask, onCreateList, onUpdateList, onDeleteList }: TaskBoardProps) {
  const [drag, setDrag] = useState<Drag | null>(null);
  const [target, setTarget] = useState<Target | null>(null);
  const [editing, setEditing] = useState<TaskListColumn | null>(null);
  const lists = [...board.lists].sort((a, b) => a.position - b.position);

  function endDrag() {
    setDrag(null);
    setTarget(null);
  }

  function dragOver(event: DragEvent<HTMLElement>, listId: string) {
    if (!drag) return;
    event.preventDefault();
    if (drag.kind === "list") {
      const box = event.currentTarget.getBoundingClientRect();
      setTarget({ listId, side: event.clientX < box.left + box.width / 2 ? "before" : "after" });
      return;
    }
    const cards = [...event.currentTarget.querySelectorAll<HTMLElement>("[data-task-id]")].filter((card) => card.dataset.taskId !== drag.id);
    const next = cards.find((card) => {
      const box = card.getBoundingClientRect();
      return event.clientY < box.top + box.height / 2;
    });
    setTarget({ listId, beforeTask: next?.dataset.taskId ?? null });
  }

  function drop(event: DragEvent<HTMLElement>, listId: string) {
    event.preventDefault();
    if (drag?.kind === "task" && target && "beforeTask" in target) {
      const task = board.tasks.find(({ id }) => id === drag.id);
      const position = positionBefore(board, drag.id, listId, target.beforeTask);
      const unchanged = task?.list_id === listId && tasksInList(board.tasks, listId).findIndex(({ id }) => id === drag.id) === position;
      if (!unchanged) onMoveTask(drag.id, listId, position);
    }
    if (drag?.kind === "list" && target && "side" in target && drag.id !== listId) {
      const others = lists.filter(({ id }) => id !== drag.id).map(({ id }) => id);
      const index = others.indexOf(listId) + (target.side === "after" ? 1 : 0);
      onReorderLists([...others.slice(0, index), drag.id, ...others.slice(index)]);
    }
    endDrag();
  }

  return (
    <div className="relative -mx-4 flex scroll-px-4 snap-x items-start gap-3 overflow-x-auto px-4 pb-4 md:-mx-6 md:scroll-px-6 md:px-6">
      {lists.map((list) => (
        <TaskColumn
          list={list}
          tasks={tasksInList(visible, list.id)}
          total={tasksInList(board.tasks, list.id).length}
          labels={board.labels}
          draggingTaskId={drag?.kind === "task" ? drag.id : null}
          dropBefore={target?.listId === list.id && "beforeTask" in target ? target.beforeTask : undefined}
          listDropSide={target?.listId === list.id && "side" in target && drag?.id !== list.id ? target.side : null}
          onOpen={onOpen}
          onEdit={() => setEditing(list)}
          onAdd={(title) => onCreateTask(list.id, title)}
          onCardDragStart={(task) => setDrag({ kind: "task", id: task.id })}
          onHeaderDragStart={() => setDrag({ kind: "list", id: list.id })}
          onDragEnd={endDrag}
          onDragOver={(event) => dragOver(event, list.id)}
          onDrop={(event) => drop(event, list.id)}
          key={list.id}
        />
      ))}
      <div className="w-[85vw] flex-none snap-start rounded-panel bg-sunken/60 p-2 sm:w-72">
        <AddInline label="Add list" placeholder="List name" maxLength={80} onAdd={onCreateList} />
      </div>
      {editing && (
        <TaskListDialog
          list={editing}
          index={lists.findIndex(({ id }) => id === editing.id)}
          count={lists.length}
          onSave={(input) => onUpdateList(editing.id, input)}
          onDelete={() => onDeleteList(editing.id)}
          onMove={(offset) => {
            const ids = lists.map(({ id }) => id);
            const from = ids.indexOf(editing.id);
            ids.splice(from, 1);
            ids.splice(from + offset, 0, editing.id);
            onReorderLists(ids);
          }}
          onClose={() => setEditing(null)}
          key={editing.id}
        />
      )}
    </div>
  );
}
