import { ChevronRight, Search } from "lucide-react";
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
              <span className="flex size-10 flex-none items-center justify-center overflow-hidden rounded-panel bg-white p-1.5" aria-hidden="true">
                <svg className="size-full text-black" viewBox="0 0 20 20" fill="currentColor"><path d="M94,7399 C99.523,7399 104,7403.59 104,7409.253 C104,7413.782 101.138,7417.624 97.167,7418.981 C96.66,7419.082 96.48,7418.762 96.48,7418.489 C96.48,7418.151 96.492,7417.047 96.492,7415.675 C96.492,7414.719 96.172,7414.095 95.813,7413.777 C98.04,7413.523 100.38,7412.656 100.38,7408.718 C100.38,7407.598 99.992,7406.684 99.35,7405.966 C99.454,7405.707 99.797,7404.664 99.252,7403.252 C99.252,7403.252 98.414,7402.977 96.505,7404.303 C95.706,7404.076 94.85,7403.962 94,7403.958 C93.15,7403.962 92.295,7404.076 91.497,7404.303 C89.586,7402.977 88.746,7403.252 88.746,7403.252 C88.203,7404.664 88.546,7405.707 88.649,7405.966 C88.01,7406.684 87.619,7407.598 87.619,7408.718 C87.619,7412.646 89.954,7413.526 92.175,7413.785 C91.889,7414.041 91.63,7414.493 91.54,7415.156 C90.97,7415.418 89.522,7415.871 88.63,7414.304 C88.63,7414.304 88.101,7413.319 87.097,7413.247 C87.097,7413.247 86.122,7413.234 87.029,7413.87 C87.029,7413.87 87.684,7414.185 88.139,7415.37 C88.139,7415.37 88.726,7417.2 91.508,7416.58 C91.513,7417.437 91.522,7418.245 91.522,7418.489 C91.522,7418.76 91.338,7419.077 90.839,7418.982 C86.865,7417.627 84,7413.783 84,7409.253 C84,7403.59 88.478,7399 94,7399" transform="translate(-84 -7399)" /></svg>
              </span>
              <span className="min-w-0 flex-1">
                <span className="block font-display text-base font-semibold">{t("Deploy from public GitHub")}</span>
                <span className="block truncate text-meta text-ink-soft">{t("Build and deploy an app from a public GitHub repository")}</span>
              </span>
              <ChevronRight className="flex-none text-ink-faint" aria-hidden="true" size={16} />
            </Link>
          ) : showGitHubDisabled ? (
            <div className="flex items-center gap-3 rounded-panel bg-raised p-4 opacity-55" aria-disabled="true">
              <span className="flex size-10 flex-none items-center justify-center overflow-hidden rounded-panel bg-white p-1.5" aria-hidden="true">
                <svg className="size-full text-black" viewBox="0 0 20 20" fill="currentColor"><path d="M94,7399 C99.523,7399 104,7403.59 104,7409.253 C104,7413.782 101.138,7417.624 97.167,7418.981 C96.66,7419.082 96.48,7418.762 96.48,7418.489 C96.48,7418.151 96.492,7417.047 96.492,7415.675 C96.492,7414.719 96.172,7414.095 95.813,7413.777 C98.04,7413.523 100.38,7412.656 100.38,7408.718 C100.38,7407.598 99.992,7406.684 99.35,7405.966 C99.454,7405.707 99.797,7404.664 99.252,7403.252 C99.252,7403.252 98.414,7402.977 96.505,7404.303 C95.706,7404.076 94.85,7403.962 94,7403.958 C93.15,7403.962 92.295,7404.076 91.497,7404.303 C89.586,7402.977 88.746,7403.252 88.746,7403.252 C88.203,7404.664 88.546,7405.707 88.649,7405.966 C88.01,7406.684 87.619,7407.598 87.619,7408.718 C87.619,7412.646 89.954,7413.526 92.175,7413.785 C91.889,7414.041 91.63,7414.493 91.54,7415.156 C90.97,7415.418 89.522,7415.871 88.63,7414.304 C88.63,7414.304 88.101,7413.319 87.097,7413.247 C87.097,7413.247 86.122,7413.234 87.029,7413.87 C87.029,7413.87 87.684,7414.185 88.139,7415.37 C88.139,7415.37 88.726,7417.2 91.508,7416.58 C91.513,7417.437 91.522,7418.245 91.522,7418.489 C91.522,7418.76 91.338,7419.077 90.839,7418.982 C86.865,7417.627 84,7413.783 84,7409.253 C84,7403.59 88.478,7399 94,7399" transform="translate(-84 -7399)" /></svg>
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
