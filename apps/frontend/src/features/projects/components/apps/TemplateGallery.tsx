import { ChevronRight, GitBranch, Search } from "lucide-react";
import { useState } from "react";
import { Link, useSearchParams } from "react-router";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { appCategories, engineLabel, engines } from "@/features/projects/engines";
import type { DatabaseTemplate } from "@/features/projects/services";
import { t } from "@/lib/i18n";

function GalleryCard({ engine, image, to, title, description }: { engine: string; image?: string; to: string; title: string; description: string }) {
  return (
    <li className="min-w-0">
      <Link className="row-link flex h-full items-center gap-3 rounded-panel bg-raised p-4" to={to}>
        <EngineMark engine={engine} image={image} />
        <span className="min-w-0 flex-1">
          <span className="block font-display text-base font-semibold">{title}</span>
          <span className="block truncate text-meta text-ink-soft">{description}</span>
        </span>
        <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
      </Link>
    </li>
  );
}

export function TemplateGallery({ templates, gitBuildsEnabled }: { templates: DatabaseTemplate[]; gitBuildsEnabled: boolean }) {
  const custom = templates.find((template) => template.custom);
  const [params] = useSearchParams();
  const [query, setQuery] = useState("");
  const environment = params.get("environment");
  const suffix = environment ? `&environment=${environment}` : "";
  const normalizedQuery = query.trim().toLowerCase();
  const filtered = templates.filter((template) => !template.custom && `${engineLabel(template.engine)} ${engines[template.engine]?.description ?? template.image}`.toLowerCase().includes(normalizedQuery));
  const showGitHub = gitBuildsEnabled && "deploy from public github build and deploy an app from a public github repository".includes(normalizedQuery);
  const showGitHubDisabled = !gitBuildsEnabled && "deploy from public github github builds are disabled in this environment".includes(normalizedQuery);
  return (
    <div className="grid gap-6">
      <label className="relative block">
        <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={16} />
        <input className="field pl-9" type="search" placeholder={t("Search apps")} value={query} onChange={(event) => setQuery(event.target.value)} />
      </label>
      {custom && (
        <div className="grid gap-3 sm:grid-cols-2">
          <Link className="row-link flex items-center gap-3 rounded-panel bg-raised p-4" to={`?template=${custom.engine}${suffix}`}>
            <EngineMark engine={custom.engine} image={custom.image} />
            <span className="min-w-0 flex-1">
              <span className="block font-display text-base font-semibold">{t("Search Docker Hub")}</span>
              <span className="block truncate text-meta text-ink-soft">{t("Any public image from Docker Hub")}</span>
            </span>
            <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
          </Link>
          <Link className="row-link flex items-center gap-3 rounded-panel bg-raised p-4" to={`?template=compose${suffix}`}>
            <EngineMark engine={custom.engine} image={custom.image} />
            <span className="min-w-0 flex-1">
              <span className="block font-display text-base font-semibold">{t("Docker Compose stack")}</span>
              <span className="block truncate text-meta text-ink-soft">{t("Run multiple containers as one stack")}</span>
            </span>
            <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
          </Link>
          {showGitHub ? (
            <Link className="row-link flex items-center gap-3 rounded-panel bg-raised p-4" to={`?template=github${suffix}`}>
              <span className="flex size-10 flex-none items-center justify-center rounded-panel bg-surface text-ink-soft" aria-hidden="true">
                <GitBranch size={20} />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-display text-base font-semibold">{t("Deploy from public GitHub")}</span>
                <span className="block truncate text-meta text-ink-soft">{t("Build and deploy an app from a public GitHub repository")}</span>
              </span>
              <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
            </Link>
          ) : showGitHubDisabled ? (
            <div className="flex items-center gap-3 rounded-panel bg-raised p-4 opacity-55" aria-disabled="true">
              <span className="flex size-10 flex-none items-center justify-center rounded-panel bg-surface text-ink-soft" aria-hidden="true">
                <GitBranch size={20} />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-display text-base font-semibold">{t("Deploy from public GitHub")}</span>
                <span className="block truncate text-meta text-ink-soft">{t("GitHub builds are disabled in this environment")}</span>
              </span>
            </div>
          ) : null}
        </div>
      )}
      {appCategories.filter(({ id }) => filtered.some((template) => template.category === id)).map(({ id, label }) => (
        <section className="grid gap-3" aria-label={label} key={id}>
          <h3 className="text-meta font-medium text-ink-soft">{label}</h3>
          <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {filtered
              .filter((template) => template.category === id)
              .map(({ engine, image }) => (
                <GalleryCard engine={engine} image={image} to={`?template=${engine}${suffix}`} title={engineLabel(engine)} description={engines[engine]?.description ?? image} key={engine} />
              ))}
          </ul>
        </section>
      ))}
    </div>
  );
}
