import { Server as ServerIcon } from "lucide-react";
import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import type { Server } from "@/features/servers/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function RegisterServerForm({ onRegistered }: { onRegistered: (server: Server, token: string) => void }) {
  const [name, setName] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const response = await api<{ server: Server; token: string }>("/servers", { method: "POST", body: { name } });
      onRegistered(response.server, response.token);
    });
  }

  return (
    <form className="grid max-w-xl gap-4 rounded-panel bg-raised p-5" onSubmit={submit} aria-label="Register server">
      <span className="flex size-10 items-center justify-center rounded-panel bg-seafoam/15 text-seafoam" aria-hidden="true">
        <ServerIcon size={20} />
      </span>
      <div>
        <h2 className="section-title">Register your server</h2>
        <p className="mt-1 text-meta text-ink-soft">The server runs your apps in Docker. Deploys need it online.</p>
      </div>
      <Field label="Server name" required autoFocus maxLength={120} placeholder="Local" value={name} onChange={(event) => setName(event.target.value)} />
      <FormError message={error} />
      <div>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Registering…" : "Register server"}
        </button>
      </div>
    </form>
  );
}
