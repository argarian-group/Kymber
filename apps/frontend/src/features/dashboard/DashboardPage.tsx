import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { AttentionPanel } from "./components/AttentionPanel";
import { DeadlinesPanel } from "./components/DeadlinesPanel";
import { DeploysPanel } from "./components/DeploysPanel";
import { ProjectsPanel } from "./components/ProjectsPanel";
import { TaskFlowPanel } from "./components/TaskFlowPanel";
import { deploys, dueTasks, projects, stages } from "./data";

export function DashboardPage() {
  const onTrack = projects.filter((project) => project.health === "on-track").length;
  const focus = projects.find((project) => project.health === "late") ?? projects.find((project) => project.health === "at-risk");

  return (
    <>
      <PageHeader
        title="Overview"
        description={`Sample data, ${onTrack} of ${projects.length} projects on track`}
        actions={
          <>
            <Link className="btn" to="/deploys">
              View deploys
            </Link>
            {focus && (
              <Link className="btn btn-primary" to="/projects">
                Review {focus.name}
              </Link>
            )}
          </>
        }
      />
      <div className="grid flex-1 grid-cols-1 xl:grid-cols-[minmax(0,1fr)_380px]">
        <div className="min-w-0">
          <AttentionPanel projects={projects} deploys={deploys} tasks={dueTasks} />
          <ProjectsPanel projects={projects} />
        </div>
        <aside className="min-w-0 bg-sunken lg:grid lg:grid-cols-2 xl:block" aria-label="Delivery">
          <DeploysPanel deploys={deploys} />
          <DeadlinesPanel tasks={dueTasks} />
          <TaskFlowPanel stages={stages} />
        </aside>
      </div>
    </>
  );
}
