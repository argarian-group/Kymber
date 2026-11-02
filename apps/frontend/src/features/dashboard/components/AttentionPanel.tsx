import { CircleAlert, Clock, TriangleAlert } from "lucide-react";
import { Link } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import { type Overview } from "@/features/dashboard/overview";
import { alertLabel } from "@/features/monitoring/metrics";
import { useAlerts } from "@/features/monitoring/useMetrics";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { dueLabel } from "@/features/projects/tasks";
import { t } from "@/lib/i18n";

export function AttentionPanel({ overview }: { overview: Overview }) {
  const user = useCurrentUser();
  const projects = new Map(overview.summaries.map(({ project }) => [project.id, project]));
  const monitored = useAlerts()
    .filter((alert) => (alert.project_id ? projects.has(alert.project_id) : user.is_root))
    .map((alert) => ({
      key: `alert-${alert.name}-${alert.service_id ?? alert.server_id}`,
      icon: TriangleAlert,
      title: `${alert.service_name ?? alert.server_name}: ${alertLabel(alert.name)}`,
      detail: alert.project_id ? (projects.get(alert.project_id)?.name ?? "") : t("Server"),
      when: "",
      to: alert.project_id ? `/projects/${alert.project_id}/apps/${alert.service_id}/metrics` : "/servers",
    }));
  const failed = overview.apps
    .filter(({ service }) => service.status === "failed")
    .map(({ service, project }) => ({ key: `app-${service.id}`, icon: CircleAlert, title: t("{name} failed", { name: service.name }), detail: project.name, when: "", to: `/projects/${project.id}/apps/${service.id}` }));
  const late = overview.tasks
    .filter((item) => item.late)
    .sort((a, b) => (a.task.due_date ?? "").localeCompare(b.task.due_date ?? ""))
    .map(({ task, project, done }) => ({ key: `task-${task.id}`, icon: Clock, title: task.title, detail: project.name, when: dueLabel(task, done), to: `/projects/${project.id}/tasks?task=${task.id}` }));
  const items = [...monitored, ...failed, ...late];

  return (
    <Panel title={t("Needs attention")} aside={items.length > 0 && <span className="text-meta text-ink-soft tabular-nums">{items.length}</span>}>
      {items.length === 0 ? (
        <EmptyState title={t("Nothing needs attention")} description={t("No alerts, failed apps or late tasks.")} />
      ) : (
        <ul className="mx-4 overflow-hidden rounded-panel bg-raised md:mx-6">
          {items.map(({ key, icon: Icon, title, detail, when, to }) => (
            <li key={key}>
              <Link className="row-link grid grid-cols-[16px_minmax(0,1fr)_auto] items-center gap-x-3 px-4 py-3" to={to}>
                <Icon className="text-ember" aria-hidden="true" size={16} />
                <span className="min-w-0">
                  <span className="block truncate font-medium">{title}</span>
                  <span className="block truncate text-meta text-ink-soft">{detail}</span>
                </span>
                <span className="text-right text-meta text-ember tabular-nums">{when}</span>
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}
