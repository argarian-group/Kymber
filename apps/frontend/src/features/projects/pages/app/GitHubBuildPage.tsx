import { Link, useNavigate, useParams } from "react-router";
import { GitHubBuildForm } from "@/features/projects/components/apps/GitHubBuildForm";
import { useServices } from "@/features/projects/useServices";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";
import { useEffect, useState } from "react";
import type { Service } from "@/features/projects/services";

export function GitHubBuildPage() {
  const { path } = useServices();
  const { appId } = useParams();
  const navigate = useNavigate();
  const [service, setService] = useState<Service | null>(null);
  const [enabled, setEnabled] = useState(false);
  useLoadEffect(
    () => api<{ git_builds_enabled?: boolean }>("/database-templates"),
    (response) => setEnabled(response.git_builds_enabled === true),
    () => setEnabled(false),
    [],
  );
  useEffect(() => {
    if (service && service.image !== "docker.io/kymber/build-artifacts") navigate("..", { replace: true });
  }, [service, navigate]);
  useLoadEffect(
    () => api<Service>(`${path}/${appId}`),
    setService,
    () => setService(null),
    [path, appId],
  );
  if (!path) return null;
  if (!service) return null;
  return <div className="grid gap-5 px-4 py-6 md:px-6"><Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="..">{t("All apps")}</Link><GitHubBuildForm path={path} gitBuildsEnabled={enabled} service={service} /></div>;
}
