import { useOutletContext } from "react-router";
import { type Project, type Workspace } from "@/features/projects/ProjectsContext";

export type ProjectOutlet = { project: Project; workspace: Workspace };

export function useProjectOutlet() {
  return useOutletContext<ProjectOutlet>();
}
