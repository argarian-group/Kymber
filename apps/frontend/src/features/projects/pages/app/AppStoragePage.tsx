import { EmptyState } from "@/components/ui/EmptyState";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { useAppOutlet } from "@/features/projects/useAppOutlet";

export function AppStoragePage() {
  const { service } = useAppOutlet();

  if (!service.data_path) return <EmptyState title="No persistent storage" description="This app has no data path, so anything it writes is lost on the next deploy." />;

  return (
    <AppSection title="Persistent storage" description="This Docker volume keeps the data when the container is removed or deployed again. Deleting the app deletes it.">
      <dl className="grid gap-4 p-4 sm:grid-cols-2">
        <ProjectDetail label="Volume">
          <code className="font-mono text-meta">kymber-service-{service.id}-data</code>
        </ProjectDetail>
        <ProjectDetail label="Mounted at">
          <code className="font-mono text-meta">{service.data_path}</code>
        </ProjectDetail>
      </dl>
    </AppSection>
  );
}
