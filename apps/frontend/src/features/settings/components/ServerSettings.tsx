import { useState, type FormEvent } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { LoadError } from "@/components/ui/LoadError";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type Server = { id: string; name: string; status: string; last_seen: string | null; created_at: string };

export function ServerSettings() {
  const [servers, setServers] = useState<Server[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [name, setName] = useState("");
  const [token, setToken] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  useLoadEffect(
    () => api<{ servers: Server[] }>("/servers"),
    (response) => {
      setServers(response.servers);
      setLoadError(null);
    },
    setLoadError,
    [attempt],
  );

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const response = await api<{ server: Server; token: string }>("/servers", { method: "POST", body: { name } });
      setServers((current) => [...(current ?? []), response.server]);
      setToken(response.token);
      setName("");
    });
  }

  if (loadError) return <LoadError title="Could not load servers" message={loadError} onRetry={() => setAttempt((value) => value + 1)} />;

  return (
    <SettingsSection title="Server" description="Runs your apps in Docker on this machine.">
      {servers?.map((server) => (
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 p-4" key={server.id}>
          <span className="min-w-0 font-medium wrap-break-word">{server.name}</span>
          <span className="text-meta text-ink-soft">
            {server.status}
            {server.last_seen ? `, last seen ${formatDate(server.last_seen)}` : ""}
          </span>
        </div>
      ))}
      {token && (
        <div className="grid gap-3 border-t border-line p-4">
          <CopyField label="Server token" value={token} secret />
          <p className="text-meta text-ink-soft">
            Copy it now, because it will not be shown again. Set it as <code>KYMBER_SERVER_TOKEN</code> in <code>.env</code>, then run <code>make server</code>.
          </p>
        </div>
      )}
      {servers?.length === 0 && (
        <form onSubmit={submit}>
          <div className="p-4 sm:max-w-sm">
            <Field label="Server name" required maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
          </div>
          <FormFooter error={error} saved={null}>
            <button className="btn btn-primary" type="submit" disabled={pending}>
              {pending ? "Registering…" : "Register server"}
            </button>
          </FormFooter>
        </form>
      )}
    </SettingsSection>
  );
}
