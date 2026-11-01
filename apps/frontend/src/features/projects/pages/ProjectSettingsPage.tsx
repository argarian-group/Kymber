import { useProjects } from "@/features/projects/ProjectsContext";
import { DeleteProject } from "@/features/projects/components/settings/DeleteProject";
import { EditProject } from "@/features/projects/components/settings/EditProject";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";

export function ProjectSettingsPage() {
  const { project } = useProjectOutlet();
  const { canWrite } = useProjects();

  return (
    <>
      <EditProject key={project.id} />
      {canWrite && <DeleteProject />}
    </>
  );
}
