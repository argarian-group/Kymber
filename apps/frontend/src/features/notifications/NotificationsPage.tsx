import { useState } from "react";
import { Link } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { notificationsChanged, type Notification } from "@/features/notifications/notifications";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

function describe(notification: Notification) {
  const actor = notification.actor?.username ?? t("Deleted user");
  if (notification.kind === "assigned") return t("{actor} assigned you to", { actor });
  if (notification.kind === "mentioned") return t("{actor} mentioned you on", { actor });
  return t("{actor} commented on", { actor });
}

export function NotificationsPage() {
  const [data, setData] = useState<{ notifications: Notification[]; unread: number } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const { pending, error: actionError, run } = useAction();
  const reload = () => setAttempt((value) => value + 1);

  useLoadEffect(() => api<{ notifications: Notification[]; unread: number }>("/notifications"), (response) => { setData(response); setError(null); }, setError, [attempt]);

  const markRead = (id = "") =>
    run(async () => {
      await api("/notifications/read", { method: "POST", body: { id } });
      window.dispatchEvent(new Event(notificationsChanged));
      reload();
    });

  if (error && !data) return <LoadError title={t("Could not load notifications")} message={error} onRetry={reload} />;
  if (!data) return <PageSkeleton />;

  return (
    <>
      <PageHeader
        title={t("Notifications")}
        actions={data.unread > 0 && <button className="btn" type="button" disabled={pending} onClick={() => void markRead()}>{t("Mark all as read")}</button>}
      />
      <div className="px-4 py-4 md:px-6">
        <FormError message={actionError} />
        {data.notifications.length === 0 ? (
          <EmptyState title={t("No notifications")} description={t("You are notified when someone assigns you, mentions you, or comments on your tasks.")} />
        ) : (
          <ul className="border-t border-line">
            {data.notifications.map((notification) => (
              <li className="border-b border-line" key={notification.id}>
                <Link className="row-link flex items-start gap-3 py-2.5" to={`/projects/${notification.project_id}/tasks?task=${notification.task_id}`} onClick={() => { if (!notification.read_at) void markRead(notification.id); }}>
                  <span className={`mt-2 size-2 flex-none rounded-full ${notification.read_at ? "" : "bg-seafoam"}`} aria-label={notification.read_at ? undefined : t("Unread")} />
                  <span className="min-w-0 flex-1">
                    <span className={`block wrap-break-word ${notification.read_at ? "text-ink-soft" : ""}`}>
                      {describe(notification)} <span className="font-mono text-xs">{notification.task_key}</span> <span className="font-medium">{notification.task_title}</span>
                    </span>
                    <span className="block text-xs text-ink-faint">{formatRelative(notification.created_at)}</span>
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </>
  );
}
