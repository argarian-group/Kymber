import { Navigate } from "react-router";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ServiceDanger } from "@/features/projects/components/apps/ServiceDanger";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { t } from "@/lib/i18n";

export function AppDangerPage() {
  const { service, path, appsPath, canWrite, onChanged } = useAppOutlet();

  if (!canWrite) return <Navigate to=".." replace />;

  const blocked = isServiceBusy(service.status) ? t("Wait for the current action to finish.") : null;

  return (
    <AppSection title={t("Danger zone")}>
      {blocked ? <p className="p-4 text-meta text-ink-soft">{blocked}</p> : <ServiceDanger service={service} path={path} appsPath={appsPath} onChanged={onChanged} />}
    </AppSection>
  );
}
