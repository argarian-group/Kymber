import { ArrowLeft } from "lucide-react";
import { useState } from "react";
import { Link, Navigate } from "react-router";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { CreateServiceForm } from "@/features/projects/components/apps/CreateServiceForm";
import { useProjects } from "@/features/projects/ProjectsContext";
import { servicesPath, type DatabaseTemplate } from "@/features/projects/services";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

export function ProjectNewAppPage() {
  const { canWrite } = useProjects();
  const { project, workspace } = useProjectOutlet();
  const [templates, setTemplates] = useState<DatabaseTemplate[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);

  useLoadEffect(
    () => api<{ templates: DatabaseTemplate[] }>("/database-templates"),
    (response) => {
      setTemplates(response.templates);
      setError(null);
    },
    setError,
    [attempt],
  );

  if (!canWrite) return <Navigate to="../apps" replace />;
  if (error) return <LoadError title="Could not load templates" message={error} onRetry={() => setAttempt((value) => value + 1)} />;
  if (!templates) return <PageSkeleton />;

  return (
    <div className="grid min-w-0 gap-5 px-4 py-6 md:px-6">
      <Link className="focus-ring inline-flex w-fit items-center gap-1 rounded-sm text-meta text-ink-soft hover:text-ink" to="../apps">
        <ArrowLeft aria-hidden="true" size={14} />
        All apps
      </Link>
      <h2 className="font-display text-lg font-semibold">New app</h2>
      <CreateServiceForm templates={templates} path={servicesPath(workspace.id, project.id)} />
    </div>
  );
}
