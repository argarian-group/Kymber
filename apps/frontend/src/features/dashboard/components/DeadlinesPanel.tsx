import { Panel } from "@/components/ui/Panel";
import type { DueTask } from "../data";

export function DeadlinesPanel({ tasks, className }: { tasks: DueTask[]; className?: string }) {
  return (
    <Panel title="Upcoming deadlines" className={className}>
      <ul>
        {tasks.map((task) => (
          <li className="flex items-center justify-between gap-4 py-2 first:pt-0 last:pb-0" key={task.title}>
            <div>
              <strong className="block font-semibold">{task.title}</strong>
              <span className="block text-meta text-ink-soft">{task.project}</span>
            </div>
            <span className={`flex-none text-meta ${task.late ? "font-semibold text-ember" : "font-medium text-ink-soft"}`}>
              {task.due}
            </span>
          </li>
        ))}
      </ul>
    </Panel>
  );
}
