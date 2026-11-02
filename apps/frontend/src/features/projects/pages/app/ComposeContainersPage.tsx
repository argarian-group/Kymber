import { Link } from "react-router";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function ComposeContainersPage() {
  const { service, path, canWrite, onChanged } = useAppOutlet();
  const { pending, error, run } = useAction();
  return <AppSection title={t("Containers")} description={t("Containers managed by this Compose stack.")}>
    {service.containers_error ? <p className="p-4 text-meta text-danger" role="alert">{service.containers_error}</p> : service.containers?.length ? <ul className="divide-y divide-line">
      {service.containers.map((container) => <li className="flex flex-wrap items-center gap-3 p-4" key={container.container_name}>
        <span className={`size-2 rounded-full ${container.state.toLowerCase() === "running" ? "bg-ok" : "bg-ink-faint"}`} aria-hidden="true" />
        <span className="min-w-0 flex-1"><span className="block truncate font-medium">{container.name}</span><span className="block truncate font-mono text-xs text-ink-soft">{container.image} · {container.status || container.state}</span></span>
        {container.ports.length > 0 && <span className="w-full pl-5 font-mono text-xs text-ink-soft">{container.ports.join(", ")}</span>}
        <Link className="btn" to={`../logs?container=${encodeURIComponent(container.name)}`}>{t("Logs")}</Link>
        {canWrite && <button className="btn" type="button" disabled={pending} onClick={() => void run(async () => {
          await api(`${path}/${service.id}/containers/${encodeURIComponent(container.name)}/${container.state.toLowerCase() === "running" ? "stop" : "start"}`, { method: "POST" });
          onChanged();
        })}>{container.state.toLowerCase() === "running" ? t("Stop") : t("Start")}</button>}
      </li>)}
    </ul> : <p className="p-4 text-meta text-ink-soft">{t("No containers found. Deploy the stack to start its services.")}</p>}
    {error && <p className="border-t border-line p-4 text-meta text-danger" role="alert">{error}</p>}
  </AppSection>;
}
