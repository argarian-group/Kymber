import { PriorityBars } from "@/features/dashboard/components/PriorityBars";
import { priorities } from "@/features/dashboard/priorities";
import { type Task } from "@/features/projects/tasks";

export function TaskPriority({ task }: { task: Task }) {
  const { label, level, tone } = priorities[task.priority];
  return (
    <span className="flex-none" title={`${label} priority`}>
      <PriorityBars level={level} tone={tone} />
      <span className="sr-only">{label} priority</span>
    </span>
  );
}
