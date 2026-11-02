import { CopyField } from "@/components/ui/CopyField";
import { ServerStatusBadge } from "@/features/servers/components/ServerStatusBadge";
import type { Server } from "@/features/servers/types";
import { t } from "@/lib/i18n";

export function ServerConnectSteps({ server, publicKey, onDone }: { server: Server; publicKey: string; onDone: () => void }) {
  return (
    <section className="grid gap-4 rounded-panel bg-raised p-4" aria-label={t("Connect {name}", { name: server.name })}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="section-title wrap-break-word">{t("Connect {name}", { name: server.name })}</h2>
        <ServerStatusBadge status={server.status} />
      </div>
      <p className="text-meta text-ink-soft">{t("Install this generated public key in the remote user's ~/.ssh/authorized_keys.")}</p>
      <CopyField label={t("Kymber SSH public key")} value={publicKey} />
      <p className="text-meta text-ink-soft">{t("The host key was pinned during registration. Deployments will connect directly over SSH.")}</p>
      <div>
        <button className="btn btn-primary" type="button" onClick={onDone}>
          {t("Done")}
        </button>
      </div>
    </section>
  );
}
