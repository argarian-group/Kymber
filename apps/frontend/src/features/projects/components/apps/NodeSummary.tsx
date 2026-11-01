import { Server } from "lucide-react";
import { Link } from "react-router";
import { AppStatus } from "@/features/projects/components/apps/AppStatus";
import { sampleApps, sampleNodes } from "@/features/projects/sampleData";

export function NodeSummary() {
  return (
    <ul className="grid gap-3 sm:grid-cols-2">
      {sampleNodes.map((node) => {
        const apps = sampleApps.filter((app) => app.node === node);
        return (
          <li className="min-w-0 rounded-panel bg-raised p-4" key={node}>
            <span className="flex items-center gap-2 font-medium">
              <Server className="text-ink-soft" aria-hidden="true" size={16} />
              {node}
            </span>
            <ul className="mt-3 grid gap-2">
              {apps.map((app) => (
                <li key={app.id}>
                  <Link className="focus-ring flex items-center justify-between gap-3 rounded-sm text-meta hover:text-seafoam" to={`apps/${app.id}`}>
                    <span className="min-w-0 truncate">{app.name}</span>
                    <AppStatus status={app.status} />
                  </Link>
                </li>
              ))}
            </ul>
          </li>
        );
      })}
    </ul>
  );
}
