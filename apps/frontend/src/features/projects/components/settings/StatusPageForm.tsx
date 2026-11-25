import { useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { FormError } from "@/components/ui/FormError";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function StatusPageForm() {
  const { project, workspace } = useProjectOutlet();
  const [token, setToken] = useState(project.status_page_token ?? null);
  const { pending, error, run } = useAction();

  function setEnabled(enabled: boolean) {
    void run(async () => {
      const response = await api<{ status_page_token: string | null }>(`/workspaces/${workspace.id}/projects/${project.id}/status-page`, { method: "PUT", body: { enabled } });
      setToken(response.status_page_token);
    });
  }

  return (
    <SettingsSection title={t("Status page")} description={t("A public page, without sign-in, that shows whether this project's apps are up.")}>
      <div className="grid gap-3 p-4">
        {token && <CopyField label={t("Public link")} value={`${window.location.origin}/status/${token}`} />}
        <div className="flex flex-wrap items-center gap-2">
          {token ? (
            <>
              <a className="btn" href={`/status/${token}`} target="_blank" rel="noreferrer">{t("Open page")}</a>
              <button className="btn" type="button" disabled={pending} onClick={() => setEnabled(true)}>{t("New link")}</button>
              <button className="btn" type="button" disabled={pending} onClick={() => setEnabled(false)}>{t("Turn off")}</button>
            </>
          ) : (
            <button className="btn btn-primary" type="button" disabled={pending} onClick={() => setEnabled(true)}>{t("Turn on")}</button>
          )}
          <FormError message={error} />
        </div>
      </div>
    </SettingsSection>
  );
}
