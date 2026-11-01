import { useProjects } from "@/features/projects/ProjectsContext";
import { DeleteProject } from "@/features/projects/components/settings/DeleteProject";
import { EditProject } from "@/features/projects/components/settings/EditProject";
import { ProjectLogoForm } from "@/features/projects/components/settings/ProjectLogoForm";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";

export function ProjectSettingsPage() {
  const { project } = useProjectOutlet();
  const { canWrite } = useProjects();

  return (
    <>
      {canWrite && <ProjectLogoForm />}
      <EditProject key={project.id} />
      {canWrite && <DeleteProject />}
    </>
  );
}
