import { type SampleTask } from "@/features/projects/sampleData";

export type DatedTask = SampleTask & { done: boolean; date: Date };

export function CalendarTask({ task }: { task: DatedTask }) {
  return (
    <span className={`flex items-center gap-1.5 rounded-sm bg-raised px-1.5 py-1 text-xs ${task.done ? "text-ink-faint line-through" : task.late ? "text-ember" : ""}`} title={task.title}>
      <span className="min-w-0 truncate">{task.title}</span>
    </span>
  );
}
