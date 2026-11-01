import { useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { CreateServiceForm } from "@/features/projects/components/apps/CreateServiceForm";
import { ServiceCard } from "@/features/projects/components/apps/ServiceCard";
import { useProjects } from "@/features/projects/ProjectsContext";
import type { DatabaseTemplate } from "@/features/projects/services";
import { useServices } from "@/features/projects/useServices";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function ProjectAppsPage() {
  const { canWrite } = useProjects();
  const { services, error, reload, path } = useServices();
  const [templates, setTemplates] = useState<DatabaseTemplate[] | null>(null);
  const [templatesError, setTemplatesError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);

  useLoadEffect(
    () => api<{ templates: DatabaseTemplate[] }>("/database-templates"),
    (response) => setTemplates(response.templates),
    setTemplatesError,
    [],
  );

  if (error && !services) return <LoadError title="Could not load apps" message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;

  const empty = services.length === 0;
  const showForm = canWrite && templates !== null && templates.length > 0 && (creating || empty);

  return (
    <div className="grid min-w-0 gap-6 px-4 py-6 md:px-6">
      {!empty && (
        <div className="flex items-center justify-between gap-3">
          <h2 className="section-title">
            {services.length} {services.length === 1 ? "app" : "apps"}
          </h2>
          {canWrite && !creating && (
            <button className="btn btn-primary" type="button" disabled={!templates?.length} title={templatesError ?? undefined} onClick={() => setCreating(true)}>
              New app
            </button>
          )}
        </div>
      )}
      {showForm && <CreateServiceForm templates={templates} path={path} onClose={empty ? undefined : () => setCreating(false)} />}
      {empty && !showForm && <EmptyState title="No apps yet" description={canWrite ? (templatesError ?? undefined) : "A workspace admin can create the first app."} />}
      {!empty && (
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {services.map((service) => (
            <ServiceCard service={service} to={service.id} key={service.id} />
          ))}
        </ul>
      )}
    </div>
  );
}
