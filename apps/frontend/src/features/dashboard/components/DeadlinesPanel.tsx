import { Link } from "react-router";
import { Panel } from "@/components/ui/Panel";
import { PriorityBars } from "@/features/dashboard/components/PriorityBars";
import { type DueTask } from "@/features/dashboard/data";
import { priorities } from "@/features/dashboard/priorities";

export function DeadlinesPanel({ tasks }: { tasks: DueTask[] }) {
  return (
    <Panel title="Upcoming deadlines">
      <ul>
        {tasks.map((task) => {
          const { label, level, tone } = priorities[task.priority];
          return (
            <li key={task.title}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2 md:px-6" to="/projects">
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
