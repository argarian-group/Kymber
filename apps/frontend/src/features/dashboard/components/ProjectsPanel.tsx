import { Panel } from "@/components/ui/Panel";
import type { Health, Project } from "../data";

const healthStyles: Record<Health, { label: string; pill: string; meter: string }> = {
  "on-track": { label: "On track", pill: "text-mist", meter: "from-jade to-mist" },
  "at-risk": { label: "At risk", pill: "text-amber", meter: "from-amber-deep to-amber" },
  late: { label: "Late", pill: "text-ember", meter: "from-ember-deep to-ember" },
};

export function ProjectsPanel({ projects, className }: { projects: Project[]; className?: string }) {
  return (
    <Panel title="Projects" className={className}>
      <ul>
        {projects.map((project) => (
          <ProjectRow project={project} key={project.name} />
        ))}
      </ul>
    </Panel>
  );
}

function ProjectRow({ project }: { project: Project }) {
  const health = healthStyles[project.health];
  return (
    <li className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2 border-t border-hairline py-4 first:border-t-0 first:pt-0 last:pb-0">
      <div>
        <strong className="block text-lg font-semibold tracking-[-0.01em]">{project.name}</strong>
        <span className="text-ink-soft">{project.phase}</span>
      </div>
      <div className="col-span-full row-start-2 grid grid-cols-[76px_minmax(0,1fr)] items-center gap-3">
        <span className="text-2xl font-[650] tracking-[-0.03em] tabular-nums">{project.progress}%</span>
        <div
          className="h-2.5 overflow-hidden rounded-full bg-deep-pine/50 shadow-[inset_0_1px_2px_rgb(0_20_17/0.5)]"
          role="progressbar"
          aria-label={`${project.name} progress`}
          aria-valuenow={project.progress}
          aria-valuemin={0}
          aria-valuemax={100}
        >
          <span className={`block h-full rounded-full bg-linear-to-r ${health.meter}`} style={{ width: `${project.progress}%` }} />
        </div>
      </div>
      <span className={`rounded-full bg-well px-3 py-1 text-xs font-semibold ${health.pill}`}>{health.label}</span>
      <dl className="col-span-full flex flex-wrap gap-x-6 gap-y-1 text-meta">
        <div className="flex gap-2">
          <dt className="text-ink-faint">Tasks left</dt>
          <dd className="font-medium tabular-nums">{project.tasksLeft}</dd>
        </div>
        <div className="flex gap-2">
          <dt className="text-ink-faint">Next deploy</dt>
          <dd className="font-medium tabular-nums">{project.nextDeploy}</dd>
        </div>
      </dl>
    </li>
  );
}
