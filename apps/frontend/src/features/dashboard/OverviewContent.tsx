import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppsPanel } from "@/features/dashboard/components/AppsPanel";
import { AttentionPanel } from "@/features/dashboard/components/AttentionPanel";
import { DeadlinesPanel } from "@/features/dashboard/components/DeadlinesPanel";
import { ProjectsPanel } from "@/features/dashboard/components/ProjectsPanel";
import { TaskSummaryPanel } from "@/features/dashboard/components/TaskSummaryPanel";
import { healthMeta } from "@/features/dashboard/health";
import { useOverview } from "@/features/dashboard/useOverview";
import { type Project } from "@/features/projects/ProjectsContext";
import { t } from "@/lib/i18n";

export function OverviewContent({ projects }: { projects: Project[] }) {
  const { overview, error, retry } = useOverview(projects);

  if (error && !overview) return <LoadError title={t("Could not load the overview")} message={error} onRetry={retry} />;
  if (!overview) return <PageSkeleton />;

  const { summaries, tasks, apps } = overview;
  const onTrack = summaries.filter(({ health }) => health === "on-track").length;
  const focus = [...summaries].sort((a, b) => healthMeta[a.health].rank - healthMeta[b.health].rank).find(({ health }) => health !== "on-track");

  return (
    <>
      <PageHeader
        title={t("Overview")}
        description={summaries.length === 1 ? t("{done} of {count} project on track", { done: onTrack, count: summaries.length }) : t("{done} of {count} projects on track", { done: onTrack, count: summaries.length })}
        actions={
          <>
            <Link className="btn" to="/projects">
              {t("View projects")}
            </Link>
            {focus && (
              <Link className="btn btn-primary" to={`/projects/${focus.project.id}`}>
                {t("Review {name}", { name: focus.project.name })}
              </Link>
            )}
          </>
        }
      />
      <div className="grid flex-1 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0">
          <ProjectsPanel summaries={summaries} />
          <AttentionPanel overview={overview} />
        </div>
        <aside className="min-w-0 bg-sunken lg:grid lg:grid-cols-2 xl:block" aria-label={t("Delivery")}>
          <AppsPanel apps={apps} />
          <DeadlinesPanel tasks={tasks} />
          <TaskSummaryPanel tasks={tasks} />
        </aside>
      </div>
    </>
  );
}
