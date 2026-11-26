import { useState } from "react";
import { Navigate } from "react-router";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { GitHubBuildForm } from "@/features/projects/components/apps/GitHubBuildForm";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function AppSourcePage() {
  const { service, path, canWrite, onChanged } = useAppOutlet();
  const [gitBuildsEnabled, setGitBuildsEnabled] = useState(false);
  const gitBuild = service.image === "docker.io/kymber/build-artifacts";

  useLoadEffect(
    () => api<{ git_builds_enabled?: boolean }>("/database-templates"),
    (response) => setGitBuildsEnabled(response.git_builds_enabled === true),
    () => setGitBuildsEnabled(false),
    [],
  );

  if (!gitBuild || !canWrite) return <Navigate to=".." replace />;
  return (
    <AppSection title={t("Source")} description={t("Repository, branch and how to build it. Save and deploy builds the code and puts it live.")}>
      <GitHubBuildForm path={path} gitBuildsEnabled={gitBuildsEnabled} service={service} onSaved={onChanged} />
    </AppSection>
  );
}
