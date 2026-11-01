import { labelColors, type TaskLabel } from "@/features/projects/tasks";

export function TaskLabelChip({ label }: { label: TaskLabel }) {
  return (
    <span className="inline-block max-w-full truncate rounded-sm px-1.5 py-0.5 text-xs font-medium text-on-accent" style={{ backgroundColor: labelColors[label.color] }}>
      {label.name}
    </span>
  );
}
