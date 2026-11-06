import { Link } from "react-router";
import { GitHubBuildForm } from "@/features/projects/components/apps/GitHubBuildForm";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";
import { useState } from "react";

export function GitHubBuildPage() {
  const { path, service } = useAppOutlet();
  const [enabled, setEnabled] = useState(false);
  useLoadEffect(
    () => api<{ git_builds_enabled?: boolean }>("/database-templates"),
    (response) => setEnabled(response.git_builds_enabled === true),
    () => setEnabled(false),
    [],
  );
  return <div className="grid gap-5 px-4 py-6 md:px-6"><Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="..">{t("All apps")}</Link><GitHubBuildForm path={path} gitBuildsEnabled={enabled} service={service} /></div>;
}
