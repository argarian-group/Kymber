import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { OverviewContent } from "@/features/dashboard/OverviewContent";
import { useProjects } from "@/features/projects/ProjectsContext";
import { t } from "@/lib/i18n";

export function DashboardPage() {
  const { state, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") return <LoadError title={t("Could not load the overview")} message={state.message} onRetry={retry} />;
  if (state.projects.length === 0) {
    return (
      <>
        <PageHeader title={t("Overview")} />
        <EmptyState
          title={t("No projects yet")}
          description={t("Create a project to see its tasks and apps here.")}
          action={
            <Link className="btn btn-primary" to="/projects">
              {t("Go to projects")}
            </Link>
          }
        />
      </>
    );
  }

  return <OverviewContent projects={state.projects} key={state.workspace?.id} />;
}
