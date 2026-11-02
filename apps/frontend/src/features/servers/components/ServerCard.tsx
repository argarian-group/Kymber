import { Server as ServerIcon } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { ServerStatusBadge } from "@/features/servers/components/ServerStatusBadge";
import type { Server } from "@/features/servers/types";
import { api } from "@/lib/api";
import { formatDate, formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";

type Mode = "view" | "rename" | "delete";

export function ServerCard({ server, onChanged }: { server: Server; onChanged: () => void }) {
  const [mode, setMode] = useState<Mode>("view");
  const [name, setName] = useState(server.name);
  const { pending, error, run } = useAction();

  async function rename(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() => api(`/servers/${server.id}`, { method: "PUT", body: { name } }));
    if (saved) {
      setMode("view");
      onChanged();
    }
  }

  async function remove() {
    const removed = await run(() => api(`/servers/${server.id}`, { method: "DELETE" }));
    if (removed) onChanged();
    else setMode("view");
  }

  async function setupSSH() {
    const configured = await run(() => api(`/servers/${server.id}/ssh`, { method: "POST" }));
    if (configured) onChanged();
  }

  return (
    <article className="rounded-panel bg-raised" aria-label={server.name}>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-3 p-4">
        <span className="flex size-10 flex-none items-center justify-center rounded-panel bg-seafoam/15 text-seafoam" aria-hidden="true">
          <ServerIcon size={20} />
        </span>
        <div className="min-w-0 flex-1 basis-40">
          <h2 className="section-title wrap-break-word">{server.name}</h2>
          <p className="font-mono text-xs text-ink-soft">{server.id}</p>
          <p className="text-xs text-ink-soft">{server.connection_type === "ssh" ? `${server.ssh_user}@${server.host}:${server.ssh_port}` : server.ssh_configured ? "Local host via SSH" : "Local host · SSH not configured"}</p>
        </div>
        <ServerStatusBadge status={server.status} />
      </div>
      <dl className="grid grid-cols-3 gap-4 border-t border-line p-4 text-meta">
        <div>
          <dt className="text-xs text-ink-soft">Apps</dt>
          <dd className="font-display text-lg font-semibold tabular-nums">{server.apps}</dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs text-ink-soft">Last seen</dt>
          <dd className="mt-1 tabular-nums">{server.last_seen ? formatRelative(server.last_seen) : "Never"}</dd>
        </div>
        <div className="min-w-0">
          <dt className="text-xs text-ink-soft">Registered</dt>
          <dd className="mt-1 tabular-nums">{formatDate(server.created_at)}</dd>
        </div>
      </dl>
      <div className="border-t border-line p-4">
        {mode === "view" && (
          <div className="flex flex-wrap gap-2">
            {server.connection_type === "local" && !server.ssh_configured && <button className="btn btn-primary" type="button" disabled={pending} onClick={setupSSH}>{pending ? "Configuring…" : "Enable SSH for local host"}</button>}
            <button className="btn" type="button" onClick={() => setMode("rename")}>
              Rename
            </button>
            <button className="btn btn-danger" type="button" disabled={server.apps > 0} title={server.apps > 0 ? "Delete this server's apps first" : undefined} onClick={() => setMode("delete")}>
              Delete server
            </button>
          </div>
        )}
        {mode === "rename" && (
          <form className="grid max-w-md gap-3" onSubmit={rename}>
            <Field label="Server name" required autoFocus maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
            <div className="flex gap-2">
              <button className="btn btn-primary" type="submit" disabled={pending}>
                {pending ? "Saving…" : "Save name"}
              </button>
              <button className="btn" type="button" disabled={pending} onClick={() => setMode("view")}>
                Cancel
              </button>
            </div>
          </form>
        )}
        {mode === "delete" && (
          <div className="grid gap-3">
              <p>{`Delete ${server.name}? You can register a new one afterwards.`}</p>
            <div className="flex flex-wrap gap-2">
              <button className="btn btn-danger" type="button" disabled={pending} onClick={remove}>
                Confirm delete
              </button>
              <button className="btn" type="button" disabled={pending} onClick={() => setMode("view")}>
                Cancel
              </button>
            </div>
          </div>
        )}
        {error && (
          <div className="mt-3">
            <FormError message={error} />
          </div>
        )}
      </div>
    </article>
  );
}
