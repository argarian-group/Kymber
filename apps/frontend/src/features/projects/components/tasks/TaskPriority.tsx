import { PriorityBars } from "@/features/dashboard/components/PriorityBars";
import { priorities } from "@/features/dashboard/priorities";
import { type Task } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

export function TaskPriority({ task }: { task: Task }) {
  const { label, level, tone } = priorities[task.priority];
  return (
    <span className="flex-none" title={t("{priority} priority", { priority: label })}>
      <PriorityBars level={level} tone={tone} />
      <span className="sr-only">{t("{priority} priority", { priority: label })}</span>
    </span>
  );
}
