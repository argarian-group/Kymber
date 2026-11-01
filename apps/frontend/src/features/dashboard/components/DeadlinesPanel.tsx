import { Link } from "react-router";
import { Panel } from "@/components/ui/Panel";
import type { DueTask, Priority } from "../data";

const priorities: Record<Priority, { label: string; level: number; tone: string }> = {
  urgent: { label: "Urgent", level: 3, tone: "fill-ember" },
  high: { label: "High", level: 2, tone: "fill-ink" },
  normal: { label: "Normal", level: 1, tone: "fill-ink-soft" },
};

function PriorityBars({ level, tone }: { level: number; tone: string }) {
  return (
    <svg className="size-4" viewBox="0 0 16 16" aria-hidden="true">
      {[1, 2, 3].map((bar) => (
        <rect className={bar <= level ? tone : "fill-white/15"} x={bar * 4 - 3} y={12 - bar * 3} width="3" height={bar * 3 + 1} rx="0.5" key={bar} />
      ))}
    </svg>
  );
}

export function DeadlinesPanel({ tasks }: { tasks: DueTask[] }) {
  return (
    <Panel title="Upcoming deadlines">
      <ul>
        {tasks.map((task) => {
          const { label, level, tone } = priorities[task.priority];
          return (
            <li key={task.title}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2 md:px-6" to="/tasks">
                <span title={`${label} priority`}>
                  <PriorityBars level={level} tone={tone} />
                  <span className="sr-only">{label} priority</span>
                </span>
                <span className="min-w-0">
                  <span className="block truncate font-medium">{task.title}</span>
                  <span className="block truncate text-meta text-ink-soft">{task.project}</span>
                </span>
                <span className={`text-right text-meta tabular-nums ${task.late ? "font-medium text-ember" : "text-ink-soft"}`}>{task.due}</span>
              </Link>
            </li>
          );
        })}
      </ul>
    </Panel>
  );
}
