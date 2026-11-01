import { useState } from "react";
import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import type { Project } from "../data";
import { healthMeta } from "../health";

const filters = [
  { id: "all", label: "All", matches: () => true },
  { id: "attention", label: "Attention", matches: (project: Project) => project.health !== "on-track" },
  { id: "on-track", label: "On track", matches: (project: Project) => project.health === "on-track" },
];

export function ProjectsPanel({ projects }: { projects: Project[] }) {
  const [filterId, setFilterId] = useState("all");
  const filter = filters.find(({ id }) => id === filterId)!;
  const visible = projects.filter(filter.matches).sort((a, b) => healthMeta[a.health].rank - healthMeta[b.health].rank);

  return (
    <Panel
      title="Projects"
      aside={
        <div className="flex gap-0.5" role="group" aria-label="Filter projects">
          {filters.map(({ id, label, matches }) => (
            <button
              className={`focus-ring min-h-9 cursor-pointer rounded-control px-2 text-xs font-medium tabular-nums transition-colors duration-100 md:min-h-7 ${
                id === filterId ? "bg-white/10 text-ink" : "text-ink-soft hover:bg-white/5 hover:text-ink"
              }`}
              type="button"
              aria-pressed={id === filterId}
              onClick={() => setFilterId(id)}
              key={id}
            >
              {label} <span className="text-ink-faint">{projects.filter(matches).length}</span>
            </button>
          ))}
        </div>
      }
    >
      {visible.length === 0 ? (
        <EmptyState
          title={`No projects match "${filter.label}"`}
          description="Try another filter to see the rest of your projects."
          action={
            <button className="btn" type="button" onClick={() => setFilterId("all")}>
              Show all projects
            </button>
          }
        />
      ) : (
        <div className="@container">
          <table className="w-full border-collapse text-left">
            <thead className="text-xs text-ink-faint">
              <tr>
                <th className="py-1.5 pr-3 pl-4 font-medium md:pl-6" scope="col">Project</th>
                <th className="px-3 py-1.5 font-medium" scope="col">Health</th>
                <th className="px-3 py-1.5 font-medium" scope="col">Done</th>
                <th className="hidden px-3 py-1.5 text-right font-medium @2xl:table-cell" scope="col">Tasks left</th>
                <th className="hidden py-1.5 pr-6 pl-3 text-right font-medium @2xl:table-cell" scope="col">Next deploy</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((project) => (
                <ProjectRow project={project} key={project.name} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </Panel>
  );
}

function ProjectRow({ project }: { project: Project }) {
  const health = healthMeta[project.health];
  return (
    <tr className="relative border-t border-line transition-colors duration-100 hover:bg-white/5 has-focus-visible:outline-2 has-focus-visible:-outline-offset-2 has-focus-visible:outline-seafoam">
      <td className="py-3 pr-3 pl-4 md:pl-6">
        <Link className="block font-medium outline-none after:absolute after:inset-0" to="/projects">
          {project.name}
        </Link>
        <span className="block text-meta text-ink-soft">{project.phase}</span>
      </td>
      <td className="px-3 py-3 text-meta whitespace-nowrap">
        <span className="flex items-center gap-2">
          <span className={`size-2 flex-none rounded-full ${health.dot}`} aria-hidden="true" />
          <span className={project.health === "on-track" ? "text-ink-soft" : `font-medium ${health.text}`}>{health.label}</span>
        </span>
      </td>
      <td className="px-3 py-3">
        <ProgressRing name={project.name} value={project.progress} tone={health.text} />
      </td>
      <td className="hidden px-3 py-3 text-right text-meta tabular-nums @2xl:table-cell">{project.tasksLeft}</td>
      <td className="hidden py-3 pr-6 pl-3 text-right text-meta whitespace-nowrap text-ink-soft tabular-nums @2xl:table-cell">
        {project.nextDeploy}
      </td>
    </tr>
  );
}

function ProgressRing({ name, value, tone }: { name: string; value: number; tone: string }) {
  return (
    <span
      className="flex items-center gap-2 text-meta font-medium tabular-nums"
      role="progressbar"
      aria-label={`${name} progress`}
      aria-valuenow={value}
      aria-valuemin={0}
      aria-valuemax={100}
    >
      <svg className={`size-5 flex-none -rotate-90 ${tone}`} viewBox="0 0 16 16" fill="none" strokeWidth="2.5" aria-hidden="true">
        <circle cx="8" cy="8" r="6" className="stroke-white/12" />
        <circle cx="8" cy="8" r="6" stroke="currentColor" pathLength="100" strokeDasharray={`${value} 100`} />
      </svg>
      {value}%
    </span>
  );
}
