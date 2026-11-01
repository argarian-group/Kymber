import { Boxes, ChevronRight, LayoutDashboard, ListChecks, Settings } from "lucide-react";
import { Link, NavLink, Outlet, useParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useProjects } from "@/features/projects/ProjectsContext";
import { ProjectMark } from "@/features/projects/components/ProjectMark";
import { type ProjectOutlet } from "@/features/projects/useProjectOutlet";

const tabs = [
  { label: "Overview", to: "", end: true, icon: LayoutDashboard },
  { label: "Apps", to: "apps", end: false, icon: Boxes },
  { label: "Tasks", to: "tasks", end: false, icon: ListChecks },
  { label: "Settings", to: "settings", end: false, icon: Settings },
];

export function ProjectLayout() {
  const { projectId } = useParams();
  const { state, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <LoadError title="Could not load this project" message={state.message} onRetry={retry} />
    );
  }

  const project = state.projects.find(({ id }) => id === projectId);
  if (!project || !state.workspace) {
    return (
      <EmptyState
        title="Project not found"
        description="It may have been deleted or belong to another workspace."
        action={
          <Link className="btn" to="/projects">
            Go to projects
          </Link>
        }
      />
    );
  }

  return (
    <>
      <header className="px-4 pt-6 md:px-6">
        <nav className="flex items-center gap-1 text-meta text-ink-soft" aria-label="Breadcrumb">
          <Link className="focus-ring rounded-sm hover:text-ink" to="/projects">
            Projects
          </Link>
          <ChevronRight className="flex-none" aria-hidden="true" size={14} />
          <span className="min-w-0 truncate" aria-current="page">
            {project.name}
          </span>
        </nav>
        <div className="mt-3 flex items-start gap-3">
          <ProjectMark name={project.name} />
          <div className="min-w-0">
            <h1 className="page-title leading-tight wrap-break-word">{project.name}</h1>
            <p className="text-meta wrap-break-word text-ink-soft">{project.slug}</p>
          </div>
        </div>
      </header>
      <nav className="mt-4 flex gap-5 overflow-x-auto border-b border-line px-4 md:px-6" aria-label="Project sections">
        {tabs.map(({ label, to, end, icon: Icon }) => (
          <NavLink
            className={({ isActive }) =>
              `focus-ring flex min-h-10 flex-none items-center gap-1.5 border-b-2 text-meta font-medium transition-colors duration-100 ${isActive ? "border-seafoam text-ink" : "border-transparent text-ink-soft hover:text-ink"}`
            }
            to={to}
            end={end}
            key={label}
          >
            <Icon aria-hidden="true" size={14} />
            {label}
          </NavLink>
        ))}
      </nav>
      <div className="flex min-h-0 flex-1 flex-col">
        <Outlet context={{ project, workspace: state.workspace } satisfies ProjectOutlet} />
      </div>
    </>
  );
}
