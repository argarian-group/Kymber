import { Plus, Search } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { ServiceCard } from "@/features/projects/components/apps/ServiceCard";
import { useProjects } from "@/features/projects/ProjectsContext";
import { useServices } from "@/features/projects/useServices";

export function ProjectAppsPage() {
  const { canWrite } = useProjects();
  const { services, error, reload } = useServices();
  const [query, setQuery] = useState("");

  if (error && !services) return <LoadError title="Could not load apps" message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;

  const term = query.trim().toLowerCase();
  const visible = services.filter(({ name, engine }) => name.toLowerCase().includes(term) || engine.includes(term));

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
          <Link className="btn btn-primary ml-auto" to="new">
            <Plus aria-hidden="true" size={15} />
            New app
          </Link>
        )}
      </div>
      {services.length === 0 ? (
        <EmptyState title="No apps yet" description={canWrite ? "Create your first app from a template." : "A workspace admin can create the first app."} />
      ) : visible.length === 0 ? (
        <EmptyState title={`No apps match "${query.trim()}"`} />
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {visible.map((service) => (
            <ServiceCard service={service} to={service.id} key={service.id} />
          ))}
        </ul>
      )}
    </div>
  );
}
