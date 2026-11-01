import { Panel } from "@/components/ui/Panel";
import type { Stage } from "../data";

const stageFills = ["bg-white/25", "bg-pine", "bg-jade", "bg-seafoam"];

export function TaskFlowPanel({ stages }: { stages: Stage[] }) {
  const totalTasks = stages.reduce((sum, stage) => sum + stage.count, 0);

  return (
    <Panel title="Task flow" className="lg:col-span-2" aside={<span className="text-meta text-ink-soft tabular-nums">{totalTasks} total</span>}>
      <div className="mx-4 mb-3 flex h-2 gap-0.5 md:mx-6" aria-hidden="true">
        {stages.map((stage, index) => (
          <span className={`rounded-xs ${stageFills[index]}`} style={{ flexGrow: stage.count }} key={stage.name} />
        ))}
      </div>
      <dl className="grid grid-cols-4 gap-2 px-4 md:px-6">
        {stages.map((stage) => (
          <div className="min-w-0" key={stage.name}>
            <dt className="truncate text-xs text-ink-soft">{stage.name}</dt>
            <dd className="font-display text-lg font-semibold tabular-nums">{stage.count}</dd>
          </div>
        ))}
      </dl>
    </Panel>
  );
}
