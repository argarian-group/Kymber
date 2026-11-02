import { useState } from "react";
import { useNavigate } from "react-router";
import { FormError } from "@/components/ui/FormError";
import type { Service, ServiceAction } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t, type TranslationKey } from "@/lib/i18n";

const removals = [
  { action: "remove_container", label: t("Remove container"), prompt: "Remove the container of {name}? Its data is kept. Use Deploy to run it again, because Start needs the container.", result: t("Container removed.") },
  { action: "remove_container_image", label: t("Remove container and image"), prompt: "Remove the container and image of {name}? Its data is kept. Use Deploy to run it again, because Start needs the container.", result: t("Container and image removed.") },
] satisfies { action: ServiceAction; label: string; prompt: TranslationKey; result: string }[];

export function ServiceDanger({ service, path, appsPath, onChanged }: { service: Service; path: string; appsPath: string; onChanged: () => void }) {
  const navigate = useNavigate();
  const [confirming, setConfirming] = useState(false);
  const [removal, setRemoval] = useState<(typeof removals)[number] | null>(null);
  const [removed, setRemoved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  async function confirm() {
    setConfirming(false);
    await run(async () => {
      await api(`${path}/${service.id}`, { method: "DELETE" });
      navigate(appsPath, { replace: true });
    });
  }

  async function remove(selected: (typeof removals)[number]) {
    setRemoval(null);
    setRemoved(null);
    const done = await run(() => api(`${path}/${service.id}/${selected.action}`, { method: "POST" }));
    if (done) {
      setRemoved(selected.result);
      onChanged();
    }
  }

  return (
    <div className="grid gap-3 p-4">
      {removal ? (
        <>
          <p>{t(removal.prompt, { name: service.name })}</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={() => remove(removal)}>
              {t("Confirm removal")}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setRemoval(null)}>
              {t("Cancel")}
            </button>
          </div>
        </>
      ) : confirming ? (
        <>
          <p>{t("Delete {name}, its container, and all its data? This cannot be undone.", { name: service.name })}</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={confirm}>
              {t("Confirm delete")}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(false)}>
              {t("Cancel")}
            </button>
          </div>
        </>
      ) : (
        <div className="flex flex-wrap gap-2">
          {service.status === "stopped" &&
            service.server_id &&
            removals.map((item) => (
              <button className="btn" type="button" disabled={pending} onClick={() => setRemoval(item)} key={item.action}>
                {item.label}
              </button>
            ))}
          <button className="btn btn-danger" type="button" disabled={pending} onClick={() => setConfirming(true)}>
            {t("Delete app")}
          </button>
        </div>
      )}
      <FormError message={error} />
      {removed && !error && <p className="text-meta text-ink-soft" role="status">{removed}</p>}
    </div>
  );
}
