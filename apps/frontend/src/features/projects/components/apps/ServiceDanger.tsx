import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function ServiceDanger({ service, path, appsPath }: { service: Service; path: string; appsPath: string }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  async function confirm() {
    setConfirming(false);
    await run(async () => {
      await api(`${path}/${service.id}`, { method: "DELETE" });
      navigate(appsPath, { replace: true });
    });
  }

  return (
    <div className="grid gap-3 p-4">
      {confirming ? (
        <>
          <p>Delete {service.name}, its container, and all its data? This cannot be undone.</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={confirm}>
              Confirm delete
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(false)}>
              Cancel
            </button>
          </div>
        </>
      ) : (
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setConfirming(true)}>
            Delete app
          </button>
        </div>
      )}
      <FormError message={error} />
    </div>
  );
}
