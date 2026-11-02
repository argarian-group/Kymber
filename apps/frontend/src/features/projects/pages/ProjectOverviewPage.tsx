import { CheckCircle2 } from "lucide-react";
import { ServiceCard } from "@/features/projects/components/apps/ServiceCard";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { SectionHeading } from "@/features/projects/components/overview/SectionHeading";
import { useServices } from "@/features/projects/useServices";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { useBoard } from "@/features/projects/useBoard";
import { formatDate } from "@/lib/format";
import { t } from "@/lib/i18n";

export function ProjectOverviewPage() {
  const { project, workspace } = useProjectOutlet();
  const { services, error } = useServices();
  const { board, error: tasksError } = useBoard();
  const tasks = board?.tasks;
  const columns = [...(board?.lists ?? [])].sort((a, b) => a.position - b.position).map(({ id, name, is_done }) => ({ id, name, is_done, count: tasks?.filter(({ list_id }) => list_id === id).length ?? 0 }));
  const done = columns.filter(({ is_done }) => is_done).reduce((sum, { count }) => sum + count, 0);
  const total = tasks?.length ?? 0;
  const percent = total ? Math.round((done / total) * 100) : 0;

  return (
    <div className="grid flex-1 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_320px]">
      <div className="grid min-w-0 content-start gap-8 px-4 py-6 md:px-6">
        <section aria-labelledby="project-apps">
          <SectionHeading id="project-apps" title={t("Apps")} to="apps" />
          {error && !services && <p className="text-meta text-ember">{error}</p>}
          {services?.length === 0 && <p className="text-meta text-ink-soft">{t("No apps yet.")}</p>}
          {services && services.length > 0 && (
            <ul className="grid gap-3 sm:grid-cols-2">
              {services.map((service) => (
                <ServiceCard service={service} to={`apps/${service.id}`} key={service.id} />
              ))}
            </ul>
          )}
        </section>
        <section aria-labelledby="project-tasks">
          <SectionHeading id="project-tasks" title={t("Tasks")} to="tasks" />
          {tasksError && !tasks && <p className="text-meta text-ember">{tasksError}</p>}
          {tasks?.length === 0 && <p className="text-meta text-ink-soft">{t("No tasks yet.")}</p>}
          {tasks && tasks.length > 0 && (
            <>
              <p className="flex items-baseline justify-between gap-3 text-meta text-ink-soft">
                <span>
                  <span className="font-display text-lg font-semibold text-ink tabular-nums">{done}</span> {t("of {total} done", { total })}
                </span>
                <span className="tabular-nums">{percent}%</span>
              </p>
              <div className="mt-1.5 h-2 overflow-hidden rounded-full bg-white/10" role="progressbar" aria-label={t("Tasks done")} aria-valuenow={percent} aria-valuemin={0} aria-valuemax={100}>
                <div className="h-full rounded-full bg-seafoam" style={{ width: `${percent}%` }} />
              </div>
              <ul className="mt-4 grid grid-cols-[repeat(auto-fill,minmax(min(100%,13rem),1fr))] gap-x-6 border-t border-line">
                {columns.map((column) => (
                  <li className="flex min-h-9 items-center gap-2 border-b border-line text-meta" key={column.id}>
                    {column.is_done && <CheckCircle2 className="flex-none text-seafoam" aria-label={t("Done list")} size={14} />}
                    <span className="min-w-0 flex-1 truncate" title={column.name}>
                      {column.name}
                    </span>
                    <span className={`tabular-nums ${column.count ? "font-medium" : "text-ink-faint"}`}>{column.count}</span>
                  </li>
                ))}
              </ul>
            </>
          )}
        </section>
      </div>
      <aside className="bg-sunken px-4 py-6 md:px-6" aria-labelledby="project-details">
        <h2 id="project-details" className="mb-4 section-title">
          {t("Details")}
        </h2>
        <dl className="grid gap-4 sm:grid-cols-2 xl:grid-cols-1">
          <ProjectDetail label={t("Slug")}>{project.slug}</ProjectDetail>
          <ProjectDetail label={t("Workspace")}>{workspace.name}</ProjectDetail>
          <ProjectDetail label={t("Your role")}>{t(workspace.role)}</ProjectDetail>
          <ProjectDetail label={t("Created")}>{formatDate(project.created_at)}</ProjectDetail>
        </dl>
      </aside>
    </div>
  );
}
