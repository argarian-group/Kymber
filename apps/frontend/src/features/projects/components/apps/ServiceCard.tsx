import { Link } from "react-router";
import { ServiceStatus } from "@/features/projects/components/apps/ServiceStatus";
import type { Service } from "@/features/projects/services";

export function ServiceCard({ service, to }: { service: Service; to: string }) {
  return (
    <li className="min-w-0">
      <Link className="row-link block h-full rounded-panel bg-raised p-4" to={to}>
        <span className="flex items-center justify-between gap-3">
          <span className="min-w-0 truncate font-medium">{service.name}</span>
          <ServiceStatus status={service.status} />
        </span>
        <span className="mt-1 block truncate text-meta text-ink-soft">{service.engine}</span>
        <span className="mt-3 block truncate text-meta text-ink-soft tabular-nums">
          {service.image_version}
          {service.database_name && `, ${service.database_name}`}
        </span>
      </Link>
    </li>
  );
}
