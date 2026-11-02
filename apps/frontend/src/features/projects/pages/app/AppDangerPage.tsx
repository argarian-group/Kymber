import { Navigate } from "react-router";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ServiceDanger } from "@/features/projects/components/apps/ServiceDanger";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";

export function AppDangerPage() {
  const { service, path, appsPath, canWrite } = useAppOutlet();

  if (!canWrite) return <Navigate to=".." replace />;

  const blocked = isServiceBusy(service.status) ? "Wait for the current action to finish." : null;

  return (
    <AppSection title="Danger zone">
      {blocked ? <p className="p-4 text-meta text-ink-soft">{blocked}</p> : <ServiceDanger service={service} path={path} appsPath={appsPath} />}
    </AppSection>
  );
}
