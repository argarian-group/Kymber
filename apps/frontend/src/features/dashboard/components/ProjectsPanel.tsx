import { useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import { ProjectRow } from "@/features/dashboard/components/ProjectRow";
import { type Project } from "@/features/dashboard/data";
import { healthMeta } from "@/features/dashboard/health";

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
