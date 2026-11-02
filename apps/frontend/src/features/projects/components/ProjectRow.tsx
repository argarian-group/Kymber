import { ChevronRight } from "lucide-react";
import { Link } from "react-router";
import { type Project } from "@/features/projects/ProjectsContext";
import { ProjectMark } from "@/features/projects/components/ProjectMark";
import { formatDate } from "@/lib/format";
import { t } from "@/lib/i18n";

export function ProjectRow({ project }: { project: Project }) {
  return (
    <li className="border-b border-line">
      <Link className="row-link flex flex-wrap items-center gap-x-4 gap-y-1 px-4 py-3 md:px-6" to={project.id}>
        <ProjectMark project={project} size="sm" />
        <span className="min-w-0 flex-1 basis-40">
          <span className="block font-medium wrap-break-word">{project.name}</span>
          <span className="block text-meta wrap-break-word text-ink-soft">{project.slug}</span>
        </span>
        <span className="text-meta text-ink-soft tabular-nums">{t("Created {date}", { date: formatDate(project.created_at) })}</span>
        <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
      </Link>
    </li>
  );
}
