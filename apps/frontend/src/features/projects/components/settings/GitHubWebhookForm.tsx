import { useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { FormError } from "@/components/ui/FormError";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function GitHubWebhookForm() {
  const { project, workspace } = useProjectOutlet();
  const [enabled, setEnabledState] = useState(project.github_webhook);
  const [secret, setSecret] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  function setEnabled(next: boolean) {
    if (!next && !window.confirm(t("Turn off the webhook? GitHub deliveries will be rejected until a new secret is set."))) return;
    void run(async () => {
      const response = await api<{ secret: string | null }>(`/workspaces/${workspace.id}/projects/${project.id}/github-webhook`, { method: "PUT", body: { enabled: next } });
      setSecret(response.secret);
      setEnabledState(next);
    });
  }

  return (
    <SettingsSection
      title={t("GitHub webhook")}
      description={t("Commits and pull requests that mention {key} show up on the task. “Fixes {key}” moves it to the first done list once merged or pushed to the default branch.", { key: `${project.task_prefix}-1` })}
    >
      <div className="grid gap-3 p-4">
        {enabled && (
          <>
            <CopyField label={t("Payload URL")} value={`${window.location.origin}/api/webhooks/github/${project.id}`} />
            {secret ? (
              <CopyField label={t("Secret")} value={secret} />
            ) : (
              <p className="text-meta text-ink-soft">{t("The secret is only shown when it is created. Make a new one if you lost it.")}</p>
            )}
            <p className="text-meta text-ink-soft">{t("In the repository settings on GitHub, add a webhook with content type application/json and the push and pull request events.")}</p>
          </>
        )}
        <div className="flex flex-wrap items-center gap-2">
          {enabled ? (
            <>
              <button className="btn" type="button" disabled={pending} onClick={() => setEnabled(true)}>{t("New secret")}</button>
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
