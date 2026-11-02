import { Activity, ArrowLeft, FolderOpen, HardDrive, KeyRound, Plug, ScrollText, Settings2, SquareTerminal, TriangleAlert } from "lucide-react";
import { useState } from "react";
import { Link, NavLink, Outlet, useParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { ServiceActions } from "@/features/projects/components/apps/ServiceActions";
import { ServiceLifecycle } from "@/features/projects/components/apps/ServiceLifecycle";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import { useProjects } from "@/features/projects/ProjectsContext";
import type { DatabaseTemplate, Service } from "@/features/projects/services";
import type { AppOutlet } from "@/features/projects/useAppOutlet";
import { useDeploymentOutput } from "@/features/projects/useDeploymentOutput";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { useServices } from "@/features/projects/useServices";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

function appTemplate(template: DatabaseTemplate | null, detail: Service | null): DatabaseTemplate | null {
  if (!template?.custom) return template;
  return { ...template, fields: Object.keys(detail?.environment ?? {}).sort().map((name) => ({ name, label: name, type: "password", secret: true })) };
}

export function AppLayout() {
  const { appId } = useParams();
  const { canWrite } = useProjects();
  const { project, workspace } = useProjectOutlet();
  const { services, error, reload, path } = useServices();
  const [detail, setDetail] = useState<Service | null>(null);
  const [templates, setTemplates] = useState<DatabaseTemplate[]>([]);
  const [revision, setRevision] = useState(0);
  const service = services?.find(({ id }) => id === appId);
  const deployOutput = useDeploymentOutput(path, service?.id, service?.status);

  useLoadEffect(
    () => (canWrite && appId ? api<Service>(`${path}/${appId}`) : Promise.resolve(null)),
    setDetail,
    () => setDetail(null),
    [canWrite, appId, path, revision],
  );

  useLoadEffect(
    () => api<{ templates: DatabaseTemplate[] }>("/database-templates"),
    (response) => setTemplates(response.templates),
    () => setTemplates([]),
    [],
  );

  if (error && !services) return <LoadError title={t("Could not load this app")} message={error} onRetry={reload} />;
  if (!services) return <PageSkeleton />;
  if (!service) {
    return (
      <EmptyState
        title={t("App not found")}
        action={
          <Link className="btn" to="../apps">
            {t("Go to apps")}
          </Link>
        }
      />
    );
  }

  const onChanged = () => {
    reload();
    setRevision((value) => value + 1);
  };
  const sections = [
    { label: t("General"), to: "", end: true, icon: Settings2 },
    { label: t("Logs"), to: "logs", end: false, icon: ScrollText },
    { label: t("Metrics"), to: "metrics", end: false, icon: Activity },
    ...(canWrite
      ? [
          { label: t("Environment"), to: "environment", end: false, icon: KeyRound },
          { label: t("Connection"), to: "connection", end: false, icon: Plug },
          { label: t("Files"), to: "files", end: false, icon: FolderOpen },
          { label: t("Terminal"), to: "terminal", end: false, icon: SquareTerminal },
        ]
      : []),
    { label: t("Storage"), to: "storage", end: false, icon: HardDrive },
    ...(canWrite ? [{ label: t("Danger zone"), to: "danger", end: false, icon: TriangleAlert }] : []),
  ];
  const context: AppOutlet = {
    service,
    detail,
    template: appTemplate(templates.find(({ engine }) => engine === service.engine) ?? null, detail),
    path,
    canWrite,
    appsPath: `/projects/${project.id}/apps`,
    onChanged,
    deployOutput,
  };

  return (
    <div className="grid min-w-0 gap-5 px-4 py-6 md:px-6">
      <Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="../apps">
        <ArrowLeft aria-hidden="true" size={14} />
        {t("All apps")}
      </Link>

      <section className="grid gap-5 rounded-panel bg-raised p-5" aria-label={service.name}>
        <div className="flex flex-wrap items-start gap-x-4 gap-y-3">
          <EngineMark engine={service.engine} logo={service.logo_url} image={service.image} size="lg" />
          <div className="min-w-0 flex-1 basis-48">
            <h2 className="font-display text-xl leading-tight font-semibold wrap-break-word">{service.name}</h2>
            <p className="mt-1 font-mono text-xs wrap-break-word text-ink-soft">
              {service.image}:{service.image_version}
            </p>
            <div className="mt-2">
              <ServiceStatus status={service.status} />
            </div>
          </div>
          {canWrite && <ServiceActions service={service} path={path} workspaceId={workspace.id} deployOutput={deployOutput} onChanged={onChanged} />}
        </div>
        <ServiceLifecycle status={service.status} />
      </section>

      {service.status === "failed" && service.last_error && (
        <p className="rounded-panel border border-ember/40 p-4 text-meta wrap-break-word text-ember" role="alert">
          {service.last_error}
        </p>
      )}

      <div className="grid items-start gap-5 lg:grid-cols-[200px_minmax(0,1fr)]">
        <nav className="flex gap-1 overflow-x-auto lg:sticky lg:top-4 lg:grid" aria-label={t("App sections")}>
          {sections.map(({ label, to, end, icon: Icon }) => (
            <NavLink
              className={({ isActive }) =>
                `focus-ring flex min-h-9 flex-none items-center gap-2 rounded-control px-2.5 text-meta font-medium whitespace-nowrap transition-colors duration-100 ${isActive ? "bg-white/8 text-ink" : "text-ink-soft hover:bg-white/5 hover:text-ink"}`
              }
              to={to}
              end={end}
              key={label}
            >
              <Icon aria-hidden="true" size={15} />
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="min-w-0">
          <Outlet context={context} />
        </div>
      </div>
    </div>
  );
}
