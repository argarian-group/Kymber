import { Plus, Search } from "lucide-react";
import { useState } from "react";
import { Link, useSearchParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { ServiceCard } from "@/features/projects/components/apps/ServiceCard";
import { appCategories } from "@/features/projects/engines";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useEnvironments } from "@/features/projects/useEnvironments";
import { useServices } from "@/features/projects/useServices";
import { t } from "@/lib/i18n";

const noEnvironment = "none";

export function ProjectAppsPage() {
  const { can } = useProjects();
  const canWrite = can("apps.manage");
  const { services, error, reload } = useServices();
  const [query, setQuery] = useState("");
  const { environments } = useEnvironments();
  const [params, setParams] = useSearchParams();
  const selected = params.get("environment");

  if (error && !services) return <LoadError title={t("Could not load apps")} message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;

  const term = query.trim().toLowerCase();
  const inEnvironment = services.filter(({ environment_id }) => !selected || (selected === noEnvironment ? !environment_id : environment_id === selected));
  const visible = inEnvironment.filter(({ name, engine }) => name.toLowerCase().includes(term) || engine.includes(term));
  const filters = environments?.length ? [{ id: "", name: t("All") }, ...environments, { id: noEnvironment, name: t("No environment") }] : [];
  const selectedName = filters.find(({ id }) => id === selected)?.name;

  return (
    <div className="grid min-w-0 gap-4 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="section-title">{t("Apps")}</h2>
        {services.length > 0 && (
          <label className="relative min-w-40 flex-1 sm:max-w-xs">
            <span className="sr-only">{t("Search apps")}</span>
            <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
            <input className="field pl-8" type="search" placeholder={t("Search")} value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
        )}
        {canWrite && (
          <Link className="btn btn-primary ml-auto" to={selected && selected !== noEnvironment ? `new?environment=${selected}` : "new"}>
            <Plus aria-hidden="true" size={15} />
            {t("New app")}
          </Link>
        )}
      </div>
      {filters.length > 0 && (
        <div className="flex flex-wrap gap-2" role="group" aria-label={t("Filter by environment")}>
          {filters.map(({ id, name }) => (
            <button className={`btn ${(selected ?? "") === id ? "btn-primary" : ""}`} type="button" aria-pressed={(selected ?? "") === id} onClick={() => setParams(id ? { environment: id } : {}, { replace: true })} key={id || "all"}>
              {name}
            </button>
          ))}
        </div>
      )}
      {services.length === 0 ? (
        <EmptyState title={t("No apps yet")} description={canWrite ? t("Create your first app from a template.") : t("Ask someone who manages apps to create the first one.")} />
      ) : inEnvironment.length === 0 ? (
        <EmptyState title={selected === noEnvironment ? t("No apps without an environment") : selectedName ? t("No apps in {name}", { name: selectedName }) : t("No apps in this environment")} />
      ) : visible.length === 0 ? (
        <EmptyState title={t("No apps match \"{query}\"", { query: query.trim() })} />
      ) : (
        <div className="grid gap-6">
          {appCategories.map(({ id, label }) => {
            const group = visible.filter((service) => service.category === id);
            return (
              group.length > 0 && (
                <section className="grid gap-3" aria-label={label} key={id}>
                  <h3 className="text-meta font-medium text-ink-soft">
                    {label} <span className="text-ink-faint">{group.length}</span>
                  </h3>
                  <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                    {group.map((service) => (
                      <ServiceCard service={service} to={service.id} key={service.id} />
                    ))}
                  </ul>
                </section>
              )
            );
          })}
        </div>
      )}
    </div>
  );
}
