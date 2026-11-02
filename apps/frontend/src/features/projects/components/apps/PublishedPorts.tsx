import { useState, type FormEvent } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { SelectField } from "@/components/ui/SelectField";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { isServiceBusy } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type ServicePort = { id: string; host_port: number; container_port: number; protocol: "tcp" | "udp"; bind: string; applied: boolean };

function AddPortForm({ url, defaultContainerPort, defaultBind, onAdded, onCancel }: { url: string; defaultContainerPort: string; defaultBind: string; onAdded: () => void; onCancel: () => void }) {
  const [hostPort, setHostPort] = useState("");
  const [containerPort, setContainerPort] = useState(defaultContainerPort);
  const [protocol, setProtocol] = useState("tcp");
  const [bind, setBind] = useState(defaultBind);
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const added = await run(() => api(url, { method: "POST", body: { host_port: Number(hostPort), container_port: Number(containerPort), protocol, bind } }));
    if (added) onAdded();
  }

  return (
    <form className="grid gap-4 border-t border-line p-4" onSubmit={submit} aria-label="Publish a port">
      <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Field label="Server port" required autoFocus type="number" min={1} max={65535} placeholder="8080" hint="Opened on the server." value={hostPort} onChange={(event) => setHostPort(event.target.value)} />
        <Field label="Container port" required type="number" min={1} max={65535} placeholder="80" hint="Where the app listens." value={containerPort} onChange={(event) => setContainerPort(event.target.value)} />
        <SelectField label="Protocol" value={protocol} onChange={(event) => setProtocol(event.target.value)}>
          <option value="tcp">TCP</option>
          <option value="udp">UDP</option>
        </SelectField>
        <SelectField label="Reachable from" value={bind} onChange={(event) => setBind(event.target.value)}>
          <option value="0.0.0.0">Anywhere</option>
          <option value="127.0.0.1">This server only</option>
        </SelectField>
      </div>
      {bind === "0.0.0.0" && <p className="text-meta text-ember">Anyone who can reach the server can connect, and Docker can bypass firewalls such as ufw.</p>}
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Adding…" : "Publish port"}
        </button>
        <button className="btn" type="button" disabled={pending} onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}

function PortRow({ port, address, url, locked, canWrite, onChanged }: { port: ServicePort; address: string; url: string; locked: boolean; canWrite: boolean; onChanged: () => void }) {
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();
  const local = port.bind === "127.0.0.1";

  async function remove() {
    const removed = await run(() => api(`${url}/${port.id}`, { method: "DELETE" }));
    if (removed) onChanged();
  }

  return (
    <li className="grid gap-3 border-t border-line p-4 first:border-t-0">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-56">
          <p className="flex flex-wrap items-center gap-2 text-xs text-ink-soft">
            {port.protocol.toUpperCase()} · {local ? "This server only" : "Anywhere"}
            {!port.applied && <span className="rounded-sm bg-seafoam/15 px-1.5 text-seafoam">Pending deploy</span>}
          </p>
          <code className="mt-1 block font-mono text-meta">
            {port.host_port} → {port.container_port}
          </code>
        </div>
        {canWrite && !confirming && (
          <button className="btn btn-danger" type="button" disabled={locked} onClick={() => setConfirming(true)}>
            Remove
          </button>
        )}
      </div>
      {port.applied && !local && <CopyField label="Address" value={`${address}:${port.host_port}`} />}
      {confirming && (
        <div className="grid gap-3">
          <p>Stop publishing port {port.host_port}? It closes on the next deploy.</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={remove}>
              {pending ? "Removing…" : "Confirm removal"}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(false)}>
              Cancel
            </button>
          </div>
        </div>
      )}
      <FormError message={error} />
    </li>
  );
}

export function PublishedPorts() {
  const { service, path, canWrite } = useAppOutlet();
  const [ports, setPorts] = useState<ServicePort[] | null>(null);
  const [host, setHost] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);
  const [changed, setChanged] = useState(false);
  const url = `${path}/${service.id}/ports`;
  const locked = isServiceBusy(service.status);

  useLoadEffect(
    () => api<{ ports: ServicePort[]; host: string }>(url),
    (response) => {
      setPorts(response.ports);
      setHost(response.host);
      setLoadError(null);
    },
    setLoadError,
    [url, attempt, service.status],
  );

  function refresh() {
    setChanged(true);
    setAttempt((value) => value + 1);
  }

  if (loadError && !ports) return <LoadError title="Could not load ports" message={loadError} onRetry={() => setAttempt((value) => value + 1)} />;

  const pendingDeploy = changed || ports?.some(({ applied }) => !applied);

  return (
    <AppSection title="Published ports" description="Reach this app from outside the server's internal network. Changes apply on the next deploy.">
      {pendingDeploy && (
        <p className="border-b border-line p-4 text-meta text-seafoam" role="status">
          Deploy the app to apply these changes.
        </p>
      )}
      {ports?.length === 0 && <p className="p-4 text-meta text-ink-soft">No published ports, so this app cannot be reached from outside the server.</p>}
      <ul>
        {ports?.map((port) => (
          <PortRow port={port} address={host || window.location.hostname} url={url} locked={locked} canWrite={canWrite} onChanged={refresh} key={port.id} />
        ))}
      </ul>
      {canWrite &&
        (adding ? (
          <AddPortForm
            url={url}
            defaultContainerPort={service.port ? String(service.port) : ""}
            defaultBind={service.category === "database" ? "127.0.0.1" : "0.0.0.0"}
            onCancel={() => setAdding(false)}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ) : (
          <div className="border-t border-line p-4">
            <button className="btn" type="button" disabled={locked} title={locked ? "Wait for the current action to finish" : undefined} onClick={() => setAdding(true)}>
              Publish port
            </button>
          </div>
        ))}
    </AppSection>
  );
}
