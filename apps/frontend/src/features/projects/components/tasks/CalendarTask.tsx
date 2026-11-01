import { isLate, type Task } from "@/features/projects/tasks";

export function CalendarTask({ task, done, onOpen }: { task: Task; done: boolean; onOpen: () => void }) {
  return (
    <button className={`focus-ring flex w-full cursor-pointer items-center gap-1.5 rounded-sm bg-raised px-1.5 py-1 text-left text-xs hover:bg-white/10 ${done ? "text-ink-faint line-through" : isLate(task, done) ? "text-ember" : ""}`} type="button" title={task.title} onClick={onOpen}>
      <span className="min-w-0 truncate">{task.title}</span>
    </button>
  );
}
