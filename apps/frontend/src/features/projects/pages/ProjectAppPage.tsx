import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Link, useParams } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { ServiceActions } from "@/features/projects/components/apps/ServiceActions";
import { ServiceDanger } from "@/features/projects/components/apps/ServiceDanger";
import { ServiceLifecycle } from "@/features/projects/components/apps/ServiceLifecycle";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { SectionHeading } from "@/features/projects/components/overview/SectionHeading";
import { useProjects } from "@/features/projects/ProjectsContext";
import { isServiceBusy, type Service } from "@/features/projects/services";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { useServices } from "@/features/projects/useServices";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function ProjectAppPage() {
  const { appId } = useParams();
  const { canWrite } = useProjects();
  const { workspace } = useProjectOutlet();
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

  return (
    <div className="grid min-w-0 gap-6 px-4 py-6 md:px-6">
      <Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="../apps">
        <ArrowLeft aria-hidden="true" size={14} />
        All apps
      </Link>

      <section className="grid gap-5 rounded-panel bg-raised p-5" aria-label={service.name}>
        <div className="flex flex-wrap items-start gap-x-4 gap-y-3">
          <EngineMark engine={service.engine} size="lg" />
          <div className="min-w-0 flex-1 basis-48">
            <h2 className="font-display text-xl leading-tight font-semibold wrap-break-word">{service.name}</h2>
            <p className="mt-1 font-mono text-xs wrap-break-word text-ink-soft">
              {service.image}:{service.image_version}
            </p>
            <div className="mt-2">
              <ServiceStatus status={service.status} />
            </div>
          </div>
          {canWrite && <ServiceActions service={service} path={path} workspaceId={workspace.id} onChanged={reload} />}
        </div>
        <ServiceLifecycle status={service.status} />
      </section>

      {service.status === "failed" && service.last_error && (
        <p className="rounded-panel border border-ember/40 p-4 text-meta wrap-break-word text-ember" role="alert">
          {service.last_error}
        </p>
      )}

      <div className="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_320px]">
        <div className="grid min-w-0 gap-6">
          {detail?.connection_url && (
            <section aria-labelledby="app-connection">
              <SectionHeading id="app-connection" title="Connection" />
              <div className="grid gap-4 rounded-panel bg-raised p-4">
                <CopyField label="Connection URL" value={detail.connection_url} secret />
                {Object.entries(detail.environment ?? {}).map(([name, value]) => (
                  <CopyField label={name} value={value} secret={/password/i.test(name)} key={name} />
                ))}
              </div>
            </section>
          )}
          {canWrite && !isServiceBusy(service.status) && service.status !== "running" && (
            <section aria-labelledby="app-danger">
              <SectionHeading id="app-danger" title="Remove" />
              <ServiceDanger service={service} path={path} onChanged={reload} />
            </section>
          )}
        </div>
        <aside className="rounded-panel bg-sunken p-4" aria-labelledby="app-details">
          <SectionHeading id="app-details" title="Details" />
          <dl className="grid gap-4">
            <ProjectDetail label="Engine">{service.engine}</ProjectDetail>
            <ProjectDetail label="Version">{service.image_version}</ProjectDetail>
            {service.server_name && <ProjectDetail label="Server">{service.server_name}</ProjectDetail>}
            {service.database_name && <ProjectDetail label="Database">{service.database_name}</ProjectDetail>}
            {service.database_user && <ProjectDetail label="Username">{service.database_user}</ProjectDetail>}
            {detail?.internal_host && (
              <ProjectDetail label="Internal address">
                {detail.internal_host}:{detail.port}
              </ProjectDetail>
            )}
          </dl>
        </aside>
      </div>
    </div>
  );
}
