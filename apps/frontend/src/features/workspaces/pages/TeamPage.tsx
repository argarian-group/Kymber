import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useProjects } from "@/features/projects/ProjectsContext";
import { Members } from "@/features/workspaces/components/Members";
import { t } from "@/lib/i18n";

export function TeamPage() {
  const { state, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <>
        <PageHeader title={t("Team")} />
        <LoadError title={t("Could not load this workspace")} message={state.message} onRetry={retry} />
      </>
    );
  }
  if (!state.workspace) {
    return (
      <>
        <PageHeader title={t("Team")} />
        <EmptyState title={t("No workspace yet")} description={t("Members appear here once you have a workspace.")} />
      </>
    );
  }
  return <Members workspace={state.workspace} key={state.workspace.id} />;
}
