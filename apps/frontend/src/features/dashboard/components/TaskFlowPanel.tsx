import { Panel } from "@/components/ui/Panel";
import type { Stage } from "../data";

const stageFills = ["bg-white/22", "bg-pine", "bg-jade", "bg-seafoam"];

export function TaskFlowPanel({ stages, className }: { stages: Stage[]; className?: string }) {
  const totalTasks = stages.reduce((sum, stage) => sum + stage.count, 0);

  return (
    <Panel title="Task flow" className={className}>
      <div className="mb-4 flex h-5 gap-1" aria-hidden="true">
        {stages.map((stage, index) => (
          <span
            className={`basis-0 rounded-lg shadow-[inset_0_1px_0_rgb(255_255_255/0.3)] ${stageFills[index]}`}
            key={stage.name}
            style={{ flexGrow: stage.count }}
          />
        ))}
      </div>
      <dl className="grid grid-cols-2 gap-4 md:grid-cols-4">
        {stages.map((stage) => (
          <div key={stage.name}>
            <dt className="text-meta text-ink-soft">{stage.name}</dt>
            <dd className="text-xl font-[650] tracking-[-0.02em] tabular-nums">
              {stage.count}
              <small className="ml-1 text-xs font-medium tracking-normal text-ink-faint">
                {Math.round((stage.count / totalTasks) * 100)}%
              </small>
            </dd>
          </div>
        ))}
      </dl>
    </Panel>
  );
}
