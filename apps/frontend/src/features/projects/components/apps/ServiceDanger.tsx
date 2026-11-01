import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function ServiceDanger({ service, path, onChanged }: { service: Service; path: string; onChanged: () => void }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState<"remove" | "delete" | null>(null);
  const { pending, error, run } = useAction();

  async function confirm() {
    const action = confirming;
    setConfirming(null);
    await run(async () => {
      if (action === "remove") {
        await api(`${path}/${service.id}/remove_container`, { method: "POST" });
        onChanged();
      } else {
        await api(`${path}/${service.id}`, { method: "DELETE" });
        navigate("../apps", { replace: true });
      }
    });
  }

  return (
    <div className="grid gap-3 rounded-panel border border-line-strong p-4">
      {confirming ? (
        <>
          <p>{confirming === "delete" ? `Delete ${service.name} and all its data? This cannot be undone.` : `Remove the container of ${service.name}? Its data stays, and you can deploy again.`}</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={confirm}>
              {confirming === "delete" ? "Confirm delete" : "Confirm remove"}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(null)}>
              Cancel
            </button>
          </div>
        </>
      ) : (
        <div className="flex flex-wrap gap-2">
          {service.status === "stopped" && (
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming("remove")}>
              Remove container
            </button>
          )}
          <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setConfirming("delete")}>
            Delete app
          </button>
        </div>
      )}
      <FormError message={error} />
    </div>
  );
}
