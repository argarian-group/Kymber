import { lazy, Suspense } from "react";
import { Navigate } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { t } from "@/lib/i18n";

const TerminalView = lazy(() => import("@/features/terminal/TerminalView"));

export function AppTerminalPage() {
  const { service, path, canTerminal } = useAppOutlet();

  if (!canTerminal) return <Navigate to=".." replace />;
  if (service.status !== "running") return <EmptyState title={t("The app is not running")} description={t("Start or deploy the app to open a terminal in its container.")} />;

  return (
    <AppSection title={t("Terminal")} description={t("A shell inside this app's container. Changes outside its volumes and mounts are lost on the next deploy.")}>
      <div className="p-4">
        <Suspense fallback={<p className="text-meta text-ink-soft">{t("Loading terminal…")}</p>}>
          <TerminalView path={`${path}/${service.id}/terminal`} label={t("Terminal for {name}", { name: service.name })} key={service.id} />
        </Suspense>
      </div>
    </AppSection>
  );
}
