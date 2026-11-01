import { Link } from "react-router";
import { ProgressRing } from "@/features/dashboard/components/ProgressRing";
import { healthMeta } from "@/features/dashboard/health";
import { type ProjectSummary } from "@/features/dashboard/overview";
import { ProjectMark } from "@/features/projects/components/ProjectMark";

export function ProjectRow({ summary }: { summary: ProjectSummary }) {
  const { project, total, done, apps, appsRunning } = summary;
  const health = healthMeta[summary.health];
  return (
    <tr className="relative border-t border-line transition-colors duration-100 hover:bg-white/5 has-focus-visible:outline-2 has-focus-visible:-outline-offset-2 has-focus-visible:outline-seafoam">
      <td className="py-3 pr-3 pl-4 md:pl-6">
        <span className="flex items-center gap-2.5">
          <ProjectMark project={project} size="sm" />
          <span className="min-w-0">
            <Link className="block font-medium wrap-break-word outline-none after:absolute after:inset-0" to={`/projects/${project.id}`}>
              {project.name}
            </Link>
            <span className="block text-meta wrap-break-word text-ink-soft">{project.slug}</span>
          </span>
        </span>
      </td>
      <td className="px-3 py-3 text-meta whitespace-nowrap">
        <span className="flex items-center gap-2">
          <span className={`size-2 flex-none rounded-full ${health.dot}`} aria-hidden="true" />
          <span className={summary.health === "on-track" ? "text-ink-soft" : `font-medium ${health.text}`}>{health.label}</span>
        </span>
      </td>
      <td className="px-3 py-3">
        <ProgressRing name={project.name} value={total ? Math.round((done / total) * 100) : 0} tone={health.text} />
      </td>
      <td className="hidden px-3 py-3 text-right text-meta tabular-nums @2xl:table-cell">{total - done}</td>
      <td className="hidden py-3 pr-6 pl-3 text-right text-meta whitespace-nowrap text-ink-soft tabular-nums @2xl:table-cell">
        {appsRunning}/{apps}
      </td>
    </tr>
  );
}
