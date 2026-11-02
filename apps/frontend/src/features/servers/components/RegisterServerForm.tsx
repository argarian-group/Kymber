import { Server as ServerIcon } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import type { Server } from "@/features/servers/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function RegisterServerForm({ onRegistered, onCancel }: { onRegistered: (server: Server, publicKey: string) => void; onCancel?: () => void }) {
  const [name, setName] = useState("");
  const [connectionType, setConnectionType] = useState<"local" | "ssh">("local");
  const [host, setHost] = useState("");
  const [port, setPort] = useState("22");
  const [sshUser, setSshUser] = useState("root");
  const [hostKey, setHostKey] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const response = await api<{ server: Server; public_key?: string }>("/servers", { method: "POST", body: { name, connection_type: connectionType, host, ssh_port: Number(port), ssh_user: sshUser, ssh_host_key: hostKey } });
      onRegistered(response.server, response.public_key ?? "");
    });
  }

  return (
    <form className="grid gap-4 rounded-panel bg-raised p-5" onSubmit={submit} aria-label="Register server">
      <span className="flex size-10 items-center justify-center rounded-panel bg-seafoam/15 text-seafoam" aria-hidden="true">
        <ServerIcon size={20} />
      </span>
      <div>
        <h2 className="section-title">Register a server</h2>
        <p className="mt-1 text-meta text-ink-soft">The server runs your apps in Docker. Deploys need it online.</p>
      </div>
      <Field label="Server name" required autoFocus maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
      <label className="grid gap-1 text-meta">Connection type<select className="input" value={connectionType} onChange={(event) => setConnectionType(event.target.value as "local" | "ssh")}><option value="local">This machine (local Docker)</option><option value="ssh">Remote server (SSH)</option></select></label>
      {connectionType === "ssh" && <>
        <Field label="SSH host" required value={host} onChange={(event) => setHost(event.target.value)} />
        <Field label="SSH port" required type="number" min={1} max={65535} value={port} onChange={(event) => setPort(event.target.value)} />
        <Field label="SSH user" required value={sshUser} onChange={(event) => setSshUser(event.target.value)} />
        <Field label="SSH host public key (full ssh-keyscan line)" required value={hostKey} onChange={(event) => setHostKey(event.target.value)} hint={`Run: ssh-keyscan -p ${port} ${host || "your-host"}`} />
      </>}
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Registering…" : "Register server"}
        </button>
        {onCancel && (
          <button className="btn" type="button" disabled={pending} onClick={onCancel}>
            Cancel
          </button>
        )}
      </div>
    </form>
  );
}
