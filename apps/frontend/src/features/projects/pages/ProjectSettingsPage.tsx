import { useProjects } from "@/features/projects/ProjectsContext";
import { DeleteProject } from "@/features/projects/components/settings/DeleteProject";
import { EditProject } from "@/features/projects/components/settings/EditProject";
import { ProjectEnvironments } from "@/features/projects/components/settings/ProjectEnvironments";
import { ProjectLogoForm } from "@/features/projects/components/settings/ProjectLogoForm";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";

export function ProjectSettingsPage() {
  const { project } = useProjectOutlet();
  const { can } = useProjects();
  const canWrite = can("projects.manage");

  return (
    <>
      {canWrite && <ProjectLogoForm />}
      <EditProject key={project.id} />
      {canWrite && <ProjectEnvironments key={`environments-${project.id}`} />}
      {canWrite && <DeleteProject />}
    </>
  );
}
