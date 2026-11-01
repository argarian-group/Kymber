import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { ServiceActions } from "@/features/projects/components/apps/ServiceActions";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { SectionHeading } from "@/features/projects/components/overview/SectionHeading";
import { useProjects } from "@/features/projects/ProjectsContext";
import type { Service } from "@/features/projects/services";
import { useServices } from "@/features/projects/useServices";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function ProjectAppPage() {
  const { appId } = useParams();
  const { canWrite } = useProjects();
  const { services, error, reload, path } = useServices();
  const [detail, setDetail] = useState<Service | null>(null);
  const service = services?.find(({ id }) => id === appId);

  useLoadEffect(
    () => (canWrite && appId ? api<Service>(`${path}/${appId}`) : Promise.resolve(null)),
    setDetail,
    () => setDetail(null),
    [canWrite, appId, path],
  );

  if (error && !services) return <LoadError title="Could not load this app" message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;
  if (!service) {
    return (
      <EmptyState
        title="App not found"
        action={
          <Link className="btn" to="../apps">
            Go to apps
          </Link>
        }
      />
    );
  }

  const secretFields = Object.entries(detail?.environment ?? {});

  return (
    <div className="grid max-w-4xl min-w-0 gap-6 px-4 py-6 md:px-6">
      <Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="../apps">
        <ArrowLeft aria-hidden="true" size={14} />
        All apps
      </Link>
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="font-display text-lg font-semibold wrap-break-word">{service.name}</h2>
          <p className="mt-0.5 text-meta wrap-break-word text-ink-soft">
            {service.image}:{service.image_version}
          </p>
        </div>
        <ServiceStatus status={service.status} />
      </div>
      {service.status === "failed" && service.last_error && (
        <p className="rounded-panel bg-raised p-4 text-meta wrap-break-word text-ember" role="alert">
          {service.last_error}
        </p>
      )}
      {canWrite && <ServiceActions service={service} path={path} onChanged={reload} />}
      <section aria-labelledby="app-details">
        <SectionHeading id="app-details" title="Details" />
        <dl className="grid gap-4 rounded-panel bg-raised p-4 sm:grid-cols-2">
          <ProjectDetail label="Engine">{service.engine}</ProjectDetail>
          {service.database_name && <ProjectDetail label="Database">{service.database_name}</ProjectDetail>}
          {service.database_user && <ProjectDetail label="Username">{service.database_user}</ProjectDetail>}
          {detail?.internal_host && (
            <ProjectDetail label="Internal address">
              {detail.internal_host}:{detail.port}
            </ProjectDetail>
          )}
        </dl>
      </section>
      {detail?.connection_url && (
        <section aria-labelledby="app-connection">
          <SectionHeading id="app-connection" title="Connection" />
          <div className="grid gap-4 rounded-panel bg-raised p-4">
            <CopyField label="Connection URL" value={detail.connection_url} secret />
            {secretFields.map(([name, value]) => (
              <CopyField label={name} value={value} secret={/password/i.test(name)} key={name} />
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
