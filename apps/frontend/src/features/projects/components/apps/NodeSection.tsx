import { Server } from "lucide-react";
import { AppCard } from "@/features/projects/components/apps/AppCard";
import { sampleApps } from "@/features/projects/sampleData";

export function NodeSection({ node }: { node: string }) {
  const apps = sampleApps.filter((app) => app.node === node);
  return (
    <section aria-label={`${node} node`}>
      <h2 className="mb-2 flex items-center gap-2 section-title">
        <Server className="text-ink-soft" aria-hidden="true" size={16} />
        {node}
        <span className="font-sans text-meta font-normal text-ink-soft tabular-nums">
          {apps.length} {apps.length === 1 ? "app" : "apps"}
        </span>
      </h2>
      <ul className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {apps.map((app) => (
          <AppCard app={app} key={app.id} />
        ))}
      </ul>
    </section>
  );
}
