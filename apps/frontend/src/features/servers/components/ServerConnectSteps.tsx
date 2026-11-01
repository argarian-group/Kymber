import { CopyField } from "@/components/ui/CopyField";
import { ServerStatusBadge } from "@/features/servers/components/ServerStatusBadge";
import type { Server } from "@/features/servers/types";

export function ServerConnectSteps({ server, token, onDone }: { server: Server; token: string; onDone: () => void }) {
  return (
    <section className="grid gap-4 rounded-panel bg-raised p-4" aria-label="Connect the server">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="section-title">Connect {server.name}</h2>
        <ServerStatusBadge status={server.status} />
      </div>
      <p className="text-meta text-ink-soft">This token is shown only once. Add it to the .env file at the root of the project.</p>
      <CopyField label=".env line" value={`KYMBER_SERVER_TOKEN=${token}`} secret />
      <CopyField label="Then keep this running" value="make server" />
      <div>
        <button className={`btn ${server.status === "online" ? "btn-primary" : ""}`} type="button" onClick={onDone}>
          {server.status === "online" ? "Done" : "Finish later"}
        </button>
      </div>
    </section>
  );
}
