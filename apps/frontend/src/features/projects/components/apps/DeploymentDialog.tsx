import { Link } from "react-router";
import { useEffect, useRef } from "react";
import { Modal } from "@/components/ui/Modal";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { isServiceBusy, type Service } from "@/features/projects/services";
import { t } from "@/lib/i18n";

export function DeploymentDialog({ service, version, pending, error, output, onForceStop, onClose }: { service: Service; version: string; pending: boolean; error: string | null; output: string | null; onForceStop?: () => Promise<boolean>; onClose: () => void }) {
  const failed = !pending && (Boolean(error) || service.status === "failed");
  const working = !failed && (pending || isServiceBusy(service.status));
  const title = failed ? t("Deployment failed") : working ? t("Deploying…") : t("Deployment complete");
  const text = pending ? t("Starting deployment…") : (error ?? output ?? service.last_error ?? t("Waiting for deployment output…"));
  const portConflict = service.compose ? text.match(/host port .*?:(\d+)\/(?:tcp|udp): address already in use/i) : null;
  const displayText = portConflict ? t("Host port {port} is already in use. Choose another published host port or remove its conflicting Compose mapping, then deploy again.", { port: portConflict[1] }) : text;
  const log = useRef<HTMLPreElement>(null);

  useEffect(() => {
    if (log.current) log.current.scrollTop = log.current.scrollHeight;
  }, [displayText]);

  return (
    <Modal label={t("Deployment")} className="max-w-3xl" onClose={onClose} onCancel={onClose}>
      <div className="grid gap-4 p-5">
        <div className="flex items-center justify-between gap-4">
          <h2 className="font-display text-lg font-semibold" aria-live="polite">{title}</h2>
          <button className="btn" type="button" onClick={onClose}>{t("Close")}</button>
        </div>
        <dl className="grid gap-4 sm:grid-cols-2">
          <ProjectDetail label={t("Image")}>
            <span className="font-mono text-meta wrap-break-word">{service.compose ? t("Docker Compose stack") : `${service.image}:${version}`}</span>
          </ProjectDetail>
          <ProjectDetail label={t("Server")}>{service.server_name ?? t("Assigning…")}</ProjectDetail>
        </dl>
        <pre ref={log} className={`max-h-[50vh] min-h-32 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap ${failed ? "text-ember" : "text-ink-soft"}`} tabIndex={0} aria-label={t("Deployment output")}>{displayText}</pre>
        {working && onForceStop && <button className="btn btn-danger w-fit" type="button" disabled={pending} onClick={() => void onForceStop()}>{pending ? t("Force-stopping deployment…") : t("Force stop")}</button>}
        {!working && !failed && (
          <Link className="text-button w-fit text-meta" to="logs" onClick={onClose}>
            {t("View runtime logs")}
          </Link>
        )}
      </div>
    </Modal>
  );
}
