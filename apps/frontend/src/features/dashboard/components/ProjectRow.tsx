import { Link } from "react-router";
import { ProgressRing } from "@/features/dashboard/components/ProgressRing";
import { type Project } from "@/features/dashboard/data";
import { healthMeta } from "@/features/dashboard/health";

export function ProjectRow({ project }: { project: Project }) {
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
