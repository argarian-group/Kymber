import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import { type OverviewApp } from "@/features/dashboard/overview";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import { t } from "@/lib/i18n";

const visibleApps = 6;
const statusOrder = ["failed", "deploying", "queued", "running", "stopped", "draft"];

export function AppsPanel({ apps }: { apps: OverviewApp[] }) {
  const sorted = [...apps].sort((a, b) => statusOrder.indexOf(a.service.status) - statusOrder.indexOf(b.service.status));

  return (
    <Panel title={t("Apps")} aside={apps.length > visibleApps && <span className="text-meta text-ink-soft tabular-nums">{t("{count} total", { count: apps.length })}</span>}>
      {apps.length === 0 ? (
        <EmptyState title={t("No apps yet")} />
      ) : (
        <ul>
          {sorted.slice(0, visibleApps).map(({ service, project }) => (
            <li key={service.id}>
              <Link className="row-link grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-2 md:px-6" to={`/projects/${project.id}/apps/${service.id}`}>
                <EngineMark engine={service.engine} logo={service.logo_url} image={service.image} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{service.name}</span>
                  <span className="block truncate text-meta text-ink-soft">{project.name}</span>
                </span>
                <ServiceStatus status={service.status} />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
