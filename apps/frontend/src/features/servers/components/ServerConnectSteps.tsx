import { CopyField } from "@/components/ui/CopyField";
import { ServerStatusBadge } from "@/features/servers/components/ServerStatusBadge";
import type { Server } from "@/features/servers/types";

export function ServerConnectSteps({ server, token, onDone }: { server: Server; token: string; onDone: () => void }) {
  return (
    <section className="grid gap-4 rounded-panel bg-raised p-4" aria-label={`Connect ${server.name}`}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="section-title wrap-break-word">Connect {server.name}</h2>
        <ServerStatusBadge status={server.status} />
      </div>
      <p className="text-meta text-ink-soft">Install this generated public key in the remote user's ~/.ssh/authorized_keys.</p>
      <CopyField label="Kymber SSH public key" value={token} />
      <p className="text-meta text-ink-soft">The host key was pinned during registration. Deployments will connect directly over SSH.</p>
      <div>
        <button className="btn btn-primary" type="button" onClick={onDone}>
          Done
        </button>
      </div>
    </section>
  );
}
