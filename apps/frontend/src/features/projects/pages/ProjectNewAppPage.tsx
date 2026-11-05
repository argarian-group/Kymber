import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Link, Navigate, useSearchParams } from "react-router";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { CreateServiceForm } from "@/features/projects/components/apps/CreateServiceForm";
import { GitHubBuildForm } from "@/features/projects/components/apps/GitHubBuildForm";
import { ComposeForm } from "@/features/projects/components/apps/ComposeForm";
import { DockerImageForm } from "@/features/projects/components/apps/DockerImageForm";
import { TemplateGallery } from "@/features/projects/components/apps/TemplateGallery";
import { useProjects } from "@/features/projects/ProjectsContext";
import { servicesPath, type DatabaseTemplate } from "@/features/projects/services";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

export function ProjectNewAppPage() {
  const { can } = useProjects();
  const canWrite = can("apps.manage");
  const { project, workspace } = useProjectOutlet();
  const [templates, setTemplates] = useState<DatabaseTemplate[] | null>(null);
  const [gitBuildsEnabled, setGitBuildsEnabled] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [params] = useSearchParams();

  useLoadEffect(
    () => api<{ templates: DatabaseTemplate[]; git_builds_enabled?: boolean }>("/database-templates"),
    (response) => {
      setTemplates(response.templates);
      setGitBuildsEnabled(response.git_builds_enabled === true);
      setError(null);
    },
    setError,
    [attempt],
  );

  if (!canWrite) return <Navigate to="../apps" replace />;
  if (error) return <LoadError title={t("Could not load templates")} message={error} onRetry={() => setAttempt((value) => value + 1)} />;
  if (!templates) return <PageSkeleton />;

  const template = templates.find(({ engine }) => engine === params.get("template"));
  const composePath = servicesPath(workspace.id, project.id);

  return (
    <div className="grid min-w-0 gap-5 px-4 py-6 md:px-6">
      <Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="../apps">
        <ArrowLeft aria-hidden="true" size={14} />
        {t("All apps")}
      </Link>
      {template?.custom ? (
        <DockerImageForm path={servicesPath(workspace.id, project.id)} />
      ) : template ? (
        <CreateServiceForm template={template} path={servicesPath(workspace.id, project.id)} key={template.engine} />
      ) : params.get("template") === "github" ? (
        <GitHubBuildForm path={servicesPath(workspace.id, project.id)} gitBuildsEnabled={gitBuildsEnabled} />
      ) : params.get("template") === "compose" ? (
        <ComposeForm path={composePath} />
      ) : (
        <>
          <h2 className="section-title">{t("Choose what to create")}</h2>
          <TemplateGallery templates={templates} gitBuildsEnabled={gitBuildsEnabled} />
        </>
      )}
    </div>
  );
}
