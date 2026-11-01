import { ServiceCard } from "@/features/projects/components/apps/ServiceCard";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { SectionHeading } from "@/features/projects/components/overview/SectionHeading";
import { sampleColumns } from "@/features/projects/sampleData";
import { useServices } from "@/features/projects/useServices";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { formatDate } from "@/lib/format";

const columnFills = ["bg-white/25", "bg-pine", "bg-jade", "bg-seafoam"];

export function ProjectOverviewPage() {
  const { project, workspace } = useProjectOutlet();
  const { services, error } = useServices();
  const total = sampleColumns.reduce((sum, column) => sum + column.tasks.length, 0);

  return (
    <div className="grid flex-1 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div className="grid min-w-0 content-start gap-8 px-4 py-6 md:px-6">
        <section aria-labelledby="project-apps">
          <SectionHeading id="project-apps" title="Apps" to="apps" />
          {error && !services && <p className="text-meta text-ember">{error}</p>}
          {services?.length === 0 && <p className="text-meta text-ink-soft">No apps yet.</p>}
          {services && services.length > 0 && (
            <ul className="grid gap-3 sm:grid-cols-2">
              {services.map((service) => (
                <ServiceCard service={service} to={`apps/${service.id}`} key={service.id} />
              ))}
            </ul>
          )}
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
