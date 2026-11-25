import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api, errorMessage } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { language, t } from "@/lib/i18n";

type AuditEvent = { id: string; username: string; method: string; route: string; path: string; workspace: boolean; created_at: string };

const timeFormat = new Intl.DateTimeFormat(language, { dateStyle: "medium", timeStyle: "short" });

const methodLabels: Record<string, string> = { POST: t("created or ran"), PUT: t("changed"), PATCH: t("changed"), DELETE: t("deleted") };

export function AuditLog({ workspaceId }: { workspaceId: string }) {
  const [events, setEvents] = useState<AuditEvent[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [more, setMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const path = `/workspaces/${workspaceId}/audit`;

  useLoadEffect(
    () => api<{ events: AuditEvent[] }>(path),
    (response) => { setEvents(response.events); setMore(response.events.length === 50); },
    setError,
    [path],
  );

  async function loadMore() {
    if (!events?.length) return;
    setLoading(true);
    try {
      const response = await api<{ events: AuditEvent[] }>(`${path}?before=${events[events.length - 1].id}`);
      setEvents([...events, ...response.events]);
      setMore(response.events.length === 50);
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setLoading(false);
  }

  return (
    <SettingsSection title={t("Audit log")} description={t("Every change made in this workspace, newest first.")}>
      {error && <div className="p-4"><FormError message={error} /></div>}
      {events && events.length === 0 && <p className="p-4 text-meta text-ink-soft">{t("No changes recorded yet.")}</p>}
      {events && events.length > 0 && (
        <ul className="divide-y divide-line">
          {events.map((event) => (
            <li className="grid gap-1 px-4 py-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-baseline sm:gap-4" key={event.id}>
              <div className="min-w-0">
                <p className="text-meta"><span className="font-medium">{event.username || t("Deleted user")}</span> <span className="text-ink-soft">{methodLabels[event.method] ?? event.method}</span>{!event.workspace && <span className="ml-2 text-xs text-ink-faint">{t("Server-wide")}</span>}</p>
                <p className="mt-0.5 font-mono text-xs break-all text-ink-soft">{event.method} {event.path.replace(`/api/workspaces/${workspaceId}`, "").replace(/^\/api/, "") || "/"}</p>
              </div>
              <time className="text-xs whitespace-nowrap text-ink-soft" dateTime={event.created_at}>{timeFormat.format(new Date(event.created_at))}</time>
            </li>
          ))}
        </ul>
      )}
      {more && <div className="border-t border-line p-3"><button className="btn" type="button" disabled={loading} onClick={() => void loadMore()}>{loading ? t("Loading…") : t("Load older")}</button></div>}
    </SettingsSection>
  );
}
