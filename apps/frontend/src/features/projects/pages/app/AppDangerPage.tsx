import { Navigate } from "react-router";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ServiceDanger } from "@/features/projects/components/apps/ServiceDanger";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormError } from "@/components/ui/FormError";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function AppDangerPage() {
  const { service, path, appsPath, canWrite, onChanged } = useAppOutlet();
  const { pending, error, run } = useAction();

  if (!canWrite) return <Navigate to=".." replace />;

  const blocked = isServiceBusy(service.status) ? t("Wait for the current action to finish.") : null;

  return (
    <AppSection title={t("Danger zone")}>
      {blocked ? (
        <div className="grid gap-3 p-4">
          <p className="text-meta text-ink-soft">{blocked}</p>
          {service.status === "deploying" && <>
            <FormError message={error} />
            <button className="btn btn-danger w-fit" type="button" disabled={pending} onClick={() => void run(async () => {
              await api(`${path}/${service.id}/cancel-deploy`, { method: "POST" });
              onChanged();
            })}>{pending ? t("Force-stopping deployment…") : t("Force stop")}</button>
          </>}
        </div>
      ) : <ServiceDanger service={service} path={path} appsPath={appsPath} onChanged={onChanged} />}
    </AppSection>
  );
}
