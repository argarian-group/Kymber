import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects } from "@/features/projects/ProjectsContext";
import { CreateWorkspace } from "@/features/settings/components/CreateWorkspace";
import { DeleteWorkspace } from "@/features/settings/components/DeleteWorkspace";
import { RenameWorkspace } from "@/features/settings/components/RenameWorkspace";
import { t } from "@/lib/i18n";

export function WorkspaceSettings() {
  const user = useCurrentUser();
  const { state, canDelete, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <LoadError title={t("Could not load this workspace")} message={state.message} onRetry={retry} />
    );
  }

  return (
    <>
      {state.workspace ? (
        <>
          <RenameWorkspace workspace={state.workspace} key={`rename-${state.workspace.id}`} />
          {canDelete && <DeleteWorkspace workspace={state.workspace} key={`delete-${state.workspace.id}`} />}
        </>
      ) : (
        !user.is_root && <EmptyState title={t("No workspace yet")} description={t("Ask an administrator to add you to a workspace.")} />
      )}
      {user.is_root && <CreateWorkspace />}
    </>
  );
}
