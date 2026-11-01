import { Link } from "react-router";
import { AppStatus } from "@/features/projects/components/apps/AppStatus";
import { type SampleApp } from "@/features/projects/sampleData";

export function AppCard({ app }: { app: SampleApp }) {
  return (
    <li className="min-w-0">
      <Link className="row-link block h-full rounded-panel bg-raised p-4" to={app.id}>
        <span className="flex items-center justify-between gap-3">
          <span className="min-w-0 truncate font-medium">{app.name}</span>
          <AppStatus status={app.status} />
        </span>
        <span className="mt-1 block truncate text-meta text-ink-soft">{app.url}</span>
        <span className="mt-3 block text-meta text-ink-soft tabular-nums">
          {app.version}, {app.when}
        </span>
      </Link>
    </li>
  );
}
