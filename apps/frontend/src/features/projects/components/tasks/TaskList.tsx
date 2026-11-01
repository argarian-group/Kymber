import { TaskAssignees } from "@/features/projects/components/tasks/TaskAssignees";
import { TaskBadges } from "@/features/projects/components/tasks/TaskBadges";
import { TaskLabelChip } from "@/features/projects/components/tasks/TaskLabelChip";
import { tasksInList, type Board, type Task } from "@/features/projects/tasks";

export function TaskList({ board, visible, onOpen }: { board: Board; visible: Task[]; onOpen: (task: Task) => void }) {
  const lists = [...board.lists].sort((a, b) => a.position - b.position);
  return (
    <div className="grid gap-6">
      {lists.map((list) => {
        const section = tasksInList(visible, list.id);
        return (
          <section aria-label={list.name} key={list.id}>
            <h2 className="mb-1 flex items-center gap-2 font-medium">
              {list.name}
              <span className="text-meta font-normal text-ink-soft tabular-nums">{section.length}</span>
            </h2>
            <ul className="border-t border-line">
              {section.map((task) => (
                <li className="border-b border-line" key={task.id}>
                  <button className="focus-ring flex min-h-11 w-full cursor-pointer flex-wrap items-center gap-x-3 gap-y-1 py-2.5 text-left transition-colors duration-100 hover:bg-white/5" type="button" onClick={() => onOpen(task)}>
                    <span className="min-w-40 flex-1 wrap-break-word">{task.title}</span>
                    {board.labels
                      .filter(({ id }) => task.label_ids.includes(id))
                      .map((label) => (
                        <TaskLabelChip label={label} key={label.id} />
                      ))}
                    <TaskBadges task={task} done={list.is_done} />
                    <TaskAssignees people={task.assignees} />
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
