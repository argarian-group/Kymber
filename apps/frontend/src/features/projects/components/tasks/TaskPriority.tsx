import { PriorityBars } from "@/features/dashboard/components/PriorityBars";
import { priorities } from "@/features/dashboard/priorities";
import { type SampleTask } from "@/features/projects/sampleData";

export function TaskPriority({ task }: { task: SampleTask }) {
  const { label, level, tone } = priorities[task.priority];
  return (
    <span className="flex-none" title={`${label} priority`}>
      <PriorityBars level={level} tone={tone} />
      <span className="sr-only">{label} priority</span>
    </span>
  );
}
