import { Link } from "react-router";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
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
  const user = useCurrentUser();
  const { pending, error, run } = useAction();
  const busy = isServiceBusy(service.status);

  function act(action: ServiceAction) {
    return run(async () => {
      await api(`${path}/${service.id}/${action}`, { method: "POST" });
      onChanged();
    });
  }

  return (
    <div className="grid justify-items-start gap-2 sm:justify-items-end">
      <div className="flex flex-wrap gap-2">
        {(available[service.status] ?? []).map(({ action, label, primary }) => (
          <button className={`btn ${primary ? "btn-primary" : ""}`} type="button" disabled={pending || busy} onClick={() => act(action)} key={action}>
            {label}
          </button>
        ))}
      </div>
      <FormError message={error} />
      {error === "no online server is available" && user.is_root && (
        <Link className="text-button text-meta" to="/servers">
          Set up a server
        </Link>
      )}
    </div>
  );
}
