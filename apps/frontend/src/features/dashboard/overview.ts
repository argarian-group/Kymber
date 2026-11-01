import { type Health } from "@/features/dashboard/health";
import { type Project } from "@/features/projects/ProjectsContext";
import { type Service } from "@/features/projects/services";
import { isLate, type Board, type Task } from "@/features/projects/tasks";

export type OverviewTask = { task: Task; project: Project; done: boolean; late: boolean };
export type OverviewApp = { service: Service; project: Project };

export type ProjectSummary = {
  project: Project;
  health: Health;
  total: number;
  done: number;
  late: number;
  apps: number;
  appsRunning: number;
};

export type Overview = { summaries: ProjectSummary[]; tasks: OverviewTask[]; apps: OverviewApp[] };

export type ProjectData = { project: Project; board: Board; services: Service[] };

export function buildOverview(data: ProjectData[]): Overview {
  const tasks = data.flatMap(({ project, board }) => {
    const doneLists = new Set(board.lists.filter(({ is_done }) => is_done).map(({ id }) => id));
    return board.tasks.map((task) => {
      const done = doneLists.has(task.list_id);
      return { task, project, done, late: isLate(task, done) };
    });
  });
  const apps = data.flatMap(({ project, services }) => services.filter(({ status }) => status !== "deleted").map((service) => ({ service, project })));
  const summaries = data.map(({ project }): ProjectSummary => {
    const own = tasks.filter((item) => item.project.id === project.id);
    const ownApps = apps.filter((item) => item.project.id === project.id);
    const late = own.filter((item) => item.late).length;
    const failed = ownApps.some(({ service }) => service.status === "failed");
    return {
      project,
      health: late > 0 ? "late" : failed ? "at-risk" : "on-track",
      total: own.length,
      done: own.filter((item) => item.done).length,
      late,
      apps: ownApps.length,
      appsRunning: ownApps.filter(({ service }) => service.status === "running").length,
    };
  });
  return { summaries, tasks, apps };
}
