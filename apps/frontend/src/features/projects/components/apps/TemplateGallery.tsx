import { ChevronRight } from "lucide-react";
import { Link } from "react-router";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { engineLabel, engines } from "@/features/projects/engines";
import type { DatabaseTemplate } from "@/features/projects/services";

export function TemplateGallery({ templates }: { templates: DatabaseTemplate[] }) {
  return (
    <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
      {templates.map(({ engine, image }) => (
        <li className="min-w-0" key={engine}>
          <Link className="row-link flex h-full items-center gap-3 rounded-panel bg-raised p-4" to={`?template=${engine}`}>
            <EngineMark engine={engine} />
            <span className="min-w-0 flex-1">
              <span className="block font-display text-base font-semibold">{engineLabel(engine)}</span>
              <span className="block truncate text-meta text-ink-soft">{engines[engine]?.description ?? image}</span>
            </span>
            <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
          </Link>
        </li>
      ))}
    </ul>
  );
}
