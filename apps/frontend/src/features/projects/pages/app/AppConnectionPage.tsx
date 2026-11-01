import { Navigate } from "react-router";
import { CopyField } from "@/components/ui/CopyField";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";

export function AppConnectionPage() {
  const { detail, canWrite } = useAppOutlet();

  if (!canWrite) return <Navigate to=".." replace />;
  if (!detail) return <PageSkeleton />;

  return (
    <AppSection title="Internal connection" description={detail.environment_name ? `Reachable only by apps in the ${detail.environment_name} environment, on the same server. It is not exposed to the internet.` : "Reachable only by apps in this workspace without an environment, on the same server. It is not exposed to the internet."}>
      <div className="grid gap-4 p-4">
        {detail.connection_url && <CopyField label="Connection URL" value={detail.connection_url} secret />}
        {detail.internal_host && <CopyField label="Host" value={detail.internal_host} />}
        {detail.port !== undefined && <CopyField label="Port" value={String(detail.port)} />}
      </div>
    </AppSection>
  );
}
