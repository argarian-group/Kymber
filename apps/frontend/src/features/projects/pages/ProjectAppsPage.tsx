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

  const canCreate = canWrite && templates !== null && templates.length > 0;

  return (
    <div className="grid min-w-0 gap-4 px-4 py-6 md:px-6">
      {canWrite && !creating && (
        <div className="flex justify-end">
          <button className="btn btn-primary" type="button" disabled={!canCreate} title={templatesError ?? undefined} onClick={() => setCreating(true)}>
            New app
          </button>
        </div>
      )}
      {creating && templates && <CreateServiceForm templates={templates} path={path} onClose={() => setCreating(false)} />}
      {services.length === 0 && !creating ? (
        <EmptyState title="No apps yet" description={canWrite ? "Create a database app from a template." : "A workspace admin can create the first app."} />
      ) : (
        <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {services.map((service) => (
            <ServiceCard service={service} to={service.id} key={service.id} />
          ))}
        </ul>
      )}
    </div>
  );
}
