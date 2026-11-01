import { Link } from "react-router";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import type { Service } from "@/features/projects/services";

export function ServiceCard({ service, to }: { service: Service; to: string }) {
  return (
    <li className="min-w-0">
      <Link className="row-link flex h-full flex-col gap-4 rounded-panel bg-raised p-4" to={to}>
        <span className="flex items-start gap-3">
          <EngineMark engine={service.engine} logo={service.logo_url} />
          <span className="min-w-0 flex-1">
            <span className="block truncate font-display text-base font-semibold">{service.name}</span>
            <span className="block truncate text-meta text-ink-soft">{service.engine === "docker" ? service.image.replace(/^docker\.io\/(library\/)?/, "") : service.engine}</span>
          </span>
        </span>
        <span className="mt-auto flex items-center justify-between gap-3 border-t border-line pt-3">
          <span className="flex min-w-0 items-center gap-2">
            <ServiceStatus status={service.status} />
            {service.environment_name && <span className="min-w-0 truncate rounded-sm bg-seafoam/15 px-1.5 text-xs text-seafoam">{service.environment_name}</span>}
          </span>
          <span className="min-w-0 truncate text-xs text-ink-soft">{service.server_name ?? service.image_version}</span>
        </span>
      </Link>
    </li>
  );
}
