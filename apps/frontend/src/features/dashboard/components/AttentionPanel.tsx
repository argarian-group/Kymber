import { CircleAlert, Clock, TriangleAlert, type LucideIcon } from "lucide-react";
import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import type { Deploy, DueTask, Project } from "../data";
import { healthMeta } from "../health";

type Item = { key: string; icon: LucideIcon; tone: string; title: string; detail: string; when: string; to: string };

function attentionItems(projects: Project[], deploys: Deploy[], tasks: DueTask[]): Item[] {
  const failed = deploys
    .filter((deploy) => deploy.status === "failed")
    .map((deploy) => ({
      key: `deploy-${deploy.project}-${deploy.version}`,
      icon: CircleAlert,
      tone: "text-ember",
      title: `${deploy.project} deploy failed`,
      detail: `${deploy.version} to ${deploy.environment}`,
      when: deploy.when,
      to: "/deploys",
    }));
  const late = tasks
    .filter((task) => task.late)
    .map((task) => ({
      key: `task-${task.title}`,
      icon: Clock,
      tone: "text-ember",
      title: task.title,
      detail: task.project,
      when: task.due,
      to: "/tasks",
    }));
  const atRisk = projects
    .filter((project) => project.health !== "on-track")
    .sort((a, b) => healthMeta[a.health].rank - healthMeta[b.health].rank)
    .map((project) => ({
      key: `project-${project.name}`,
      icon: TriangleAlert,
      tone: project.health === "late" ? "text-ember" : "text-amber",
      title: `${project.name} is ${project.health === "late" ? "late" : "at risk"}`,
      detail: `${project.tasksLeft} tasks left`,
      when: `Deploy ${project.nextDeploy}`,
      to: "/projects",
    }));
  return [...failed, ...late, ...atRisk];
}

export function AttentionPanel({ projects, deploys, tasks }: { projects: Project[]; deploys: Deploy[]; tasks: DueTask[] }) {
  const items = attentionItems(projects, deploys, tasks);

  return (
    <Panel title="Needs attention" aside={items.length > 0 && <span className="text-meta text-ink-soft tabular-nums">{items.length}</span>}>
      {items.length === 0 ? (
        <EmptyState title="Nothing needs attention" description="No failed deploys, late tasks or projects at risk." />
      ) : (
        <ul className="mx-4 overflow-hidden rounded-panel bg-raised md:mx-6">
          {items.map(({ key, icon: Icon, tone, title, detail, when, to }) => (
            <li key={key}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-3" to={to}>
                <Icon className={tone} aria-hidden="true" size={16} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{title}</span>
                  <span className="block truncate text-meta text-ink-soft tabular-nums">{detail}</span>
                </span>
                <span className="text-right text-meta text-ink-soft tabular-nums">{when}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
