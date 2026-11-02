import { Link } from "react-router";
import { Modal } from "@/components/ui/Modal";
import { ProjectDetail } from "@/features/projects/components/overview/ProjectDetail";
import { isServiceBusy, type Service } from "@/features/projects/services";

export function DeploymentDialog({ service, version, pending, error, output, onClose }: { service: Service; version: string; pending: boolean; error: string | null; output: string | null; onClose: () => void }) {
  const failed = !pending && (Boolean(error) || service.status === "failed");
  const working = !failed && (pending || isServiceBusy(service.status));
  const title = failed ? "Deployment failed" : working ? "Deploying…" : "Deployment complete";
  const text = pending ? "Starting deployment…" : (error ?? output ?? service.last_error ?? "Waiting for deployment output…");

  return (
    <Modal label="Deployment" className="max-w-3xl" onClose={onClose} onCancel={onClose}>
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
        <pre className={`max-h-[50vh] min-h-32 overflow-auto rounded-control bg-canvas p-4 font-mono text-xs whitespace-pre-wrap ${failed ? "text-ember" : "text-ink-soft"}`} tabIndex={0} aria-label="Deployment output">{text}</pre>
        {!working && !failed && (
          <Link className="text-button w-fit text-meta" to="logs" onClick={onClose}>
            View runtime logs
          </Link>
        )}
      </div>
    </Modal>
  );
}
