import { ChevronRight } from "lucide-react";
import { Link, useSearchParams } from "react-router";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { appCategories, engineLabel, engines } from "@/features/projects/engines";
import type { DatabaseTemplate } from "@/features/projects/services";

function GalleryCard({ engine, to, title, description }: { engine: string; to: string; title: string; description: string }) {
  return (
    <li className="min-w-0">
      <Link className="row-link flex h-full items-center gap-3 rounded-panel bg-raised p-4" to={to}>
        <EngineMark engine={engine} />
        <span className="min-w-0 flex-1">
          <span className="block font-display text-base font-semibold">{title}</span>
          <span className="block truncate text-meta text-ink-soft">{description}</span>
        </span>
        <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
      </Link>
    </li>
  );
}

export function TemplateGallery({ templates }: { templates: DatabaseTemplate[] }) {
  const custom = templates.find((template) => template.custom);
  const [params] = useSearchParams();
  const environment = params.get("environment");
  const suffix = environment ? `&environment=${environment}` : "";
  return (
    <div className="grid gap-6">
      {appCategories.map(({ id, label, item }) => (
        <section className="grid gap-3" aria-label={label} key={id}>
          <h3 className="text-meta font-medium text-ink-soft">{label}</h3>
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {templates
              .filter((template) => template.category === id)
              .map(({ engine, image }) => (
                <GalleryCard engine={engine} to={`?template=${engine}${suffix}`} title={engineLabel(engine)} description={engines[engine]?.description ?? image} key={engine} />
              ))}
            {custom && <GalleryCard engine={custom.engine} to={`?template=${custom.engine}&category=${id}${suffix}`} title={engineLabel(custom.engine)} description={`Any ${item} from Docker Hub`} />}
          </ul>
        </section>
      ))}
    </div>
  );
}
