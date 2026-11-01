import { useState } from "react";
import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects, workspaceLabel } from "@/features/projects/ProjectsContext";
import { CreateProjectForm } from "@/features/projects/components/CreateProjectForm";
import { ProjectRow } from "@/features/projects/components/ProjectRow";

export function ProjectsPage() {
  const { state, workspaces, canWrite, retry } = useProjects();
  const user = useCurrentUser();
  const [creatingIn, setCreatingIn] = useState<string | null>(null);

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <>
        <PageHeader title="Projects" />
        <LoadError title="Could not load projects" message={state.message} onRetry={retry} />
      </>
    );
  }
  if (!state.workspace) {
    return (
      <>
        <PageHeader title="Projects" />
        <EmptyState title="No workspace yet" description={user.is_root ? "Create a workspace to start adding projects." : "Ask an administrator to add you to a workspace."} action={user.is_root && <Link className="btn" to="/settings/workspace">Create workspace</Link>} />
      </>
    );
  }

  const count = state.projects.length;
  const workspaceId = state.workspace.id;
  const creating = creatingIn === workspaceId;
  const startCreating = () => setCreatingIn(workspaceId);
  return (
    <>
      <PageHeader
        title="Projects"
        description={`${workspaceLabel(state.workspace, workspaces)}, ${count} ${count === 1 ? "project" : "projects"}`}
        actions={
          canWrite &&
          !creating && (
            <button className="btn btn-primary" type="button" onClick={startCreating}>
              New project
            </button>
          )
        }
      />
      {creating && <CreateProjectForm onClose={() => setCreatingIn(null)} />}
      {count === 0 && !creating ? (
        <EmptyState
          title="No projects yet"
          description={canWrite ? "Create the first project in this workspace." : "A workspace admin can create the first project."}
          action={
            canWrite && (
              <button className="btn" type="button" onClick={startCreating}>
                Create project
              </button>
            )
          }
        />
      ) : (
        <ul className="mt-4 border-t border-line">
          {state.projects.map((project) => (
            <ProjectRow project={project} key={project.id} />
          ))}
        </ul>
      )}
    </>
  );
}
