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

const noEnvironment = "none";

export function ProjectAppsPage() {
  const { canWrite } = useProjects();
  const { services, error, reload } = useServices();
  const [query, setQuery] = useState("");
  const { environments } = useEnvironments();
  const [params, setParams] = useSearchParams();
  const selected = params.get("environment");

  if (error && !services) return <LoadError title="Could not load apps" message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;

  const term = query.trim().toLowerCase();
  const inEnvironment = services.filter(({ environment_id }) => !selected || (selected === noEnvironment ? !environment_id : environment_id === selected));
  const visible = inEnvironment.filter(({ name, engine }) => name.toLowerCase().includes(term) || engine.includes(term));
  const filters = environments?.length ? [{ id: "", name: "All" }, ...environments, { id: noEnvironment, name: "No environment" }] : [];
  const selectedName = filters.find(({ id }) => id === selected)?.name;

  return (
    <div className="grid min-w-0 gap-4 px-4 py-6 md:px-6">
      <div className="flex flex-wrap items-center gap-3">
        <h2 className="section-title">Apps</h2>
        {services.length > 0 && (
          <label className="relative min-w-40 flex-1 sm:max-w-xs">
            <span className="sr-only">Search apps</span>
            <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
            <input className="field pl-8" type="search" placeholder="Search" value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
        )}
        {canWrite && (
          <Link className="btn btn-primary ml-auto" to={selected && selected !== noEnvironment ? `new?environment=${selected}` : "new"}>
            <Plus aria-hidden="true" size={15} />
            New app
          </Link>
        )}
      </div>
      {filters.length > 0 && (
        <div className="flex flex-wrap gap-2" role="group" aria-label="Filter by environment">
          {filters.map(({ id, name }) => (
            <button className={`btn ${(selected ?? "") === id ? "btn-primary" : ""}`} type="button" aria-pressed={(selected ?? "") === id} onClick={() => setParams(id ? { environment: id } : {}, { replace: true })} key={id || "all"}>
              {name}
            </button>
          ))}
        </div>
      )}
      {services.length === 0 ? (
        <EmptyState title="No apps yet" description={canWrite ? "Create your first app from a template." : "A workspace admin can create the first app."} />
      ) : inEnvironment.length === 0 ? (
        <EmptyState title={selected === noEnvironment ? "No apps without an environment" : `No apps in ${selectedName ?? "this environment"}`} />
      ) : visible.length === 0 ? (
        <EmptyState title={`No apps match "${query.trim()}"`} />
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
