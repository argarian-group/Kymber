import { ArrowLeft, Server } from "lucide-react";
import { Link, useParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { AppStatus } from "@/features/projects/components/apps/AppStatus";
import { sampleApps } from "@/features/projects/sampleData";

export function ProjectAppPage() {
  const { appId } = useParams();
  const app = sampleApps.find(({ id }) => id === appId);

  if (!app) {
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
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="font-display text-lg font-semibold">{app.name}</h2>
          <p className="mt-0.5 flex items-center gap-1.5 text-meta text-ink-soft">
            <Server aria-hidden="true" size={14} />
            {app.node}
          </p>
          <p className="mt-0.5 text-meta wrap-break-word text-ink-soft">{app.url}</p>
        </div>
        <AppStatus status={app.status} />
      </div>
    </div>
  );
}
