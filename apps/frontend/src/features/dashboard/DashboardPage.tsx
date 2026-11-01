import { PageHeader } from "@/components/layout/PageHeader";
import { DeadlinesPanel } from "./components/DeadlinesPanel";
import { DeploysPanel } from "./components/DeploysPanel";
import { ProjectsPanel } from "./components/ProjectsPanel";
import { TaskFlowPanel } from "./components/TaskFlowPanel";
import { deploys, dueTasks, projects, stages } from "./data";

export function DashboardPage() {
  const onTrack = projects.filter((project) => project.health === "on-track").length;
  const needAttention = projects.length - onTrack;

  return (
    <>
      <PageHeader
        title="Overview"
        description={`${onTrack} of ${projects.length} projects on track, ${needAttention} need attention.`}
      />
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1.7fr)_minmax(300px,1fr)]">
        <ProjectsPanel projects={projects} className="lg:row-span-2" />
        <DeploysPanel deploys={deploys} className="lg:col-start-2 lg:row-start-1" />
        <TaskFlowPanel stages={stages} className="lg:col-span-2 lg:row-start-3" />
        <DeadlinesPanel tasks={dueTasks} className="lg:col-start-2 lg:row-start-2" />
      </div>
    </>
  );
}
