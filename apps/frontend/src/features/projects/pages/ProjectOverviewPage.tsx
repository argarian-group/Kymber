import { DeployList } from "@/features/projects/components/apps/DeployList";
import { NodeSummary } from "@/features/projects/components/apps/NodeSummary";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { SectionHeading } from "@/features/projects/components/overview/SectionHeading";
import { sampleColumns, sampleDeploys } from "@/features/projects/sampleData";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { formatDate } from "@/lib/format";

const columnFills = ["bg-white/25", "bg-pine", "bg-jade", "bg-seafoam"];

export function ProjectOverviewPage() {
  const { project, workspace } = useProjectOutlet();
  const total = sampleColumns.reduce((sum, column) => sum + column.tasks.length, 0);

  return (
    <div className="grid flex-1 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div className="grid min-w-0 content-start gap-8 px-4 py-6 md:px-6">
        <section aria-labelledby="project-apps">
          <SectionHeading id="project-apps" title="Apps" to="apps" />
          <NodeSummary />
        </section>
        <section aria-labelledby="project-deploys">
          <SectionHeading id="project-deploys" title="Recent deploys" />
          <DeployList deploys={sampleDeploys.slice(0, 3)} />
        </section>
        <section aria-labelledby="project-tasks">
          <SectionHeading id="project-tasks" title="Tasks" to="tasks" />
          <div className="flex h-2 gap-0.5" aria-hidden="true">
            {sampleColumns.map((column, index) => (
              <span className={`rounded-xs ${columnFills[index]}`} style={{ flexGrow: column.tasks.length }} key={column.name} />
            ))}
          </div>
          <dl className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {sampleColumns.map((column) => (
              <div key={column.name}>
                <dt className="text-xs text-ink-soft">{column.name}</dt>
                <dd className="font-display text-lg font-semibold tabular-nums">{column.tasks.length}</dd>
              </div>
            ))}
          </dl>
          <p className="sr-only">{total} tasks in total</p>
        </section>
      </div>
      <aside className="bg-sunken px-4 py-6 md:px-6" aria-labelledby="project-details">
        <h2 id="project-details" className="mb-4 section-title">
          Details
        </h2>
        <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <ProjectDetail label="Slug">{project.slug}</ProjectDetail>
          <ProjectDetail label="Workspace">{workspace.name}</ProjectDetail>
          <ProjectDetail label="Your role">{workspace.role}</ProjectDetail>
          <ProjectDetail label="Created">{formatDate(project.created_at)}</ProjectDetail>
        </dl>
      </aside>
    </div>
  );
}
