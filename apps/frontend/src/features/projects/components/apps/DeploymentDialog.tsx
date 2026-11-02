import { Link } from "react-router";
import { Modal } from "@/components/ui/Modal";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { isServiceBusy, type Service } from "@/features/projects/services";

// shortcut: shows progress only because the API discards deploy output, upgrade when it stores deployment logs.
export function DeploymentDialog({ service, version, pending, error, onClose }: { service: Service; version: string; pending: boolean; error: string | null; onClose: () => void }) {
  const failure = pending ? null : error ?? (service.status === "failed" ? service.last_error ?? "The deployment failed." : null);
  const working = !failure && (pending || isServiceBusy(service.status));
  const title = failure ? "Deployment failed" : working ? "Deploying…" : "Deployment complete";

  return (
    <Modal label="Deployment" className="max-w-xl" onClose={onClose} onCancel={onClose}>
      <div className="grid gap-4 p-5">
        <div className="flex items-center justify-between gap-4">
          <h2 className="font-display text-lg font-semibold" aria-live="polite">{title}</h2>
          <button className="btn" type="button" onClick={onClose}>Close</button>
        </div>
        <dl className="grid gap-4 sm:grid-cols-2">
          <ProjectDetail label="Image">
            <span className="font-mono text-meta wrap-break-word">{service.image}:{version}</span>
          </ProjectDetail>
          <ProjectDetail label="Server">{service.server_name ?? "Assigning…"}</ProjectDetail>
        </dl>
        {failure && <pre className="max-h-[50vh] overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap text-ember">{failure}</pre>}
        {!working && !failure && (
          <Link className="text-button w-fit text-meta" to="logs" onClick={onClose}>
            View runtime logs
          </Link>
        )}
      </div>
    </Modal>
  );
}
