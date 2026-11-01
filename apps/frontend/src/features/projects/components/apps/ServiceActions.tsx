import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { isServiceBusy, type Service, type ServiceAction } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

const available: Record<string, { action: ServiceAction; label: string; primary?: boolean }[]> = {
  draft: [{ action: "deploy", label: "Deploy", primary: true }],
  running: [
    { action: "restart", label: "Restart" },
    { action: "stop", label: "Stop" },
  ],
  stopped: [
    { action: "start", label: "Start", primary: true },
    { action: "deploy", label: "Deploy again" },
  ],
  failed: [{ action: "deploy", label: "Deploy again", primary: true }],
};

export function ServiceActions({ service, path, onChanged }: { service: Service; path: string; onChanged: () => void }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState<"remove_container" | "delete" | null>(null);
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);

  function act(action: ServiceAction) {
    setConfirming(null);
    return run(async () => {
      await api(`${path}/${service.id}/${action}`, { method: "POST" });
      onChanged();
    });
  }

  function remove() {
    setConfirming(null);
    return run(async () => {
      await api(`${path}/${service.id}`, { method: "DELETE" });
      navigate("../apps", { replace: true });
    });
  }

  if (confirming) {
    return (
      <div className="grid gap-2 rounded-panel bg-raised p-4">
        <p>{confirming === "delete" ? `Delete ${service.name} and its data? This cannot be undone.` : `Remove the container of ${service.name}? Its data stays, and you can deploy it again.`}</p>
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-danger" type="button" disabled={pending} onClick={() => (confirming === "delete" ? remove() : act("remove_container"))}>
            {confirming === "delete" ? "Confirm delete" : "Confirm remove"}
          </button>
          <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(null)}>
            Cancel
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="grid gap-2">
      <div className="flex flex-wrap gap-2">
        {(available[service.status] ?? []).map(({ action, label, primary }) => (
          <button className={`btn ${primary ? "btn-primary" : ""}`} type="button" disabled={pending || busy} onClick={() => act(action)} key={action}>
            {label}
          </button>
        ))}
        {service.status === "stopped" && (
          <button className="btn" type="button" disabled={pending} onClick={() => setConfirming("remove_container")}>
            Remove container
          </button>
        )}
        {!busy && service.status !== "running" && (
          <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setConfirming("delete")}>
            Delete app
          </button>
        )}
      </div>
      <FormError message={error} />
    </div>
  );
}
