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
import { t } from "@/lib/i18n";

type ServicePort = { id: string; host_port: number; container_port: number; protocol: "tcp" | "udp"; bind: string; applied: boolean };
type ComposePort = ServicePort & { line: number };

function composePorts(file: string, containers: NonNullable<ReturnType<typeof useAppOutlet>["detail"]>["containers"]): ComposePort[] {
  const lines = file.split("\n");
  const ports: ComposePort[] = [];
  let inServices = false;
  let inPorts = false;
  for (const [line, source] of lines.entries()) {
    const trimmed = source.trim();
    const indent = source.length - source.trimStart().length;
    if (!trimmed || trimmed.startsWith("#")) continue;
    if (indent === 0) {
      if (trimmed === "services:") inServices = true;
      else inServices = false;
      inPorts = false;
      continue;
    }
    if (!inServices) continue;
    if (indent === 2 && trimmed.endsWith(":")) {
      inPorts = false;
      continue;
    }
    if (indent === 4) {
      inPorts = trimmed === "ports:";
      continue;
    }
    if (!inPorts || indent !== 6 || !trimmed.startsWith("- ")) continue;
    let value = trimmed.slice(2).trim();
    try {
      value = JSON.parse(value) as string;
    } catch {
      value = value.replace(/^['"]|['"]$/g, "");
    }
    const match = value.match(/^(?:(\d{1,3}(?:\.\d{1,3}){3}):)?(\d+):(\d+)(?:\/(tcp|udp))?$/);
    if (!match) continue;
    const hostPort = Number(match[2]);
    const containerPort = Number(match[3]);
    const protocol = match[4] === "udp" ? "udp" : "tcp";
    const bind = match[1] || "0.0.0.0";
    const applied = containers?.some(({ ports: runningPorts }) => runningPorts.some((running) => running.endsWith(`${hostPort}→${containerPort}/${protocol}`))) ?? false;
    ports.push({ id: `compose-${line}`, line, host_port: hostPort, container_port: containerPort, protocol, bind, applied });
  }
  return ports;
}

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
    <form className="grid gap-4 border-t border-line p-4" onSubmit={submit} aria-label={t("Publish a port")}>
      <div className="grid items-start gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <Field label={t("Server port")} required autoFocus type="number" min={1} max={65535} placeholder="8080" hint={t("Opened on the server.")} value={hostPort} onChange={(event) => setHostPort(event.target.value)} />
        <Field label={t("Container port")} required type="number" min={1} max={65535} placeholder="80" hint={t("Where the app listens.")} value={containerPort} onChange={(event) => setContainerPort(event.target.value)} />
        <SelectField label={t("Protocol")} value={protocol} onChange={(event) => setProtocol(event.target.value)}>
          <option value="tcp">TCP</option>
          <option value="udp">UDP</option>
        </SelectField>
        <SelectField label={t("Reachable from")} value={bind} onChange={(event) => setBind(event.target.value)}>
          <option value="0.0.0.0">{t("Anywhere")}</option>
          <option value="127.0.0.1">{t("This server only")}</option>
        </SelectField>
      </div>
      {bind === "0.0.0.0" && <p className="text-meta text-ember">{t("Anyone who can reach the server can connect, and Docker can bypass firewalls such as ufw.")}</p>}
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? t("Adding…") : t("Publish port")}
        </button>
        <button className="btn" type="button" disabled={pending} onClick={onCancel}>
          {t("Cancel")}
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
            {port.protocol.toUpperCase()} · {local ? t("This server only") : t("Anywhere")}
            {!port.applied && <span className="rounded-sm bg-seafoam/15 px-1.5 text-seafoam">{t("Pending deploy")}</span>}
          </p>
          <code className="mt-1 block font-mono text-meta">
            {port.host_port} → {port.container_port}
          </code>
        </div>
        {canWrite && !confirming && (
          <button className="btn btn-danger" type="button" disabled={locked} onClick={() => setConfirming(true)}>
            {t("Remove")}
          </button>
        )}
      </div>
      {port.applied && !local && <CopyField label={t("Address")} value={`${address}:${port.host_port}`} />}
      {confirming && (
        <div className="grid gap-3">
          <p>{t("Stop publishing port {port}? It closes on the next deploy.", { port: port.host_port })}</p>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={remove}>
              {pending ? t("Removing…") : t("Confirm removal")}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setConfirming(false)}>
              {t("Cancel")}
            </button>
          </div>
        </div>
      )}
      <FormError message={error} />
    </li>
  );
}

function ComposePortRow({ port, address, file, path, serviceId, locked, canWrite, onChanged }: { port: ComposePort; address: string; file: string; path: string; serviceId: string; locked: boolean; canWrite: boolean; onChanged: () => void }) {
  const { pending, error, run } = useAction();
  const local = port.bind === "127.0.0.1";

  async function makePublic() {
    const lines = file.split("\n");
    lines[port.line] = lines[port.line].replace("127.0.0.1:", "");
    if (await run(() => api(`${path}/${serviceId}`, { method: "PUT", body: { compose_file: lines.join("\n") } }))) onChanged();
  }

  return <li className="grid gap-3 border-t border-line p-4 first:border-t-0">
    <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
      <div className="min-w-0 flex-1 basis-56">
        <p className="flex flex-wrap items-center gap-2 text-xs text-ink-soft">{port.protocol.toUpperCase()} · {local ? t("This server only") : t("Anywhere")}{!port.applied && <span className="rounded-sm bg-seafoam/15 px-1.5 text-seafoam">{t("Pending deploy")}</span>}</p>
        <code className="mt-1 block font-mono text-meta">{port.host_port} → {port.container_port}</code>
      </div>
      {canWrite && local && <button className="btn" type="button" disabled={locked || pending} onClick={() => void makePublic()}>{pending ? t("Saving…") : t("Make public")}</button>}
    </div>
    {port.applied && !local && <CopyField label={t("Address")} value={`${address}:${port.host_port}`} />}
    <FormError message={error} />
  </li>;
}

export function PublishedPorts() {
  const { service, detail, path, canWrite, onChanged } = useAppOutlet();
  const [ports, setPorts] = useState<ServicePort[] | null>(null);
  const [host, setHost] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);
  const [changed, setChanged] = useState(false);
  const url = `${path}/${service.id}/ports`;
  const locked = isServiceBusy(service.status);
  const composeFile = detail?.compose_file || service.compose_file || "";
  const composePortRows = service.compose ? composePorts(composeFile, detail?.containers) : [];

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

  if (loadError && !ports) return <LoadError title={t("Could not load ports")} message={loadError} onRetry={() => setAttempt((value) => value + 1)} />;

  const pendingDeploy = changed || ports?.some(({ applied }) => !applied) || composePortRows.some(({ applied }) => !applied);

  return (
    <AppSection title={t("Published ports")} description={t("Reach this app from outside the server's internal network. Changes apply on the next deploy.")}>
      {pendingDeploy && (
        <p className="border-b border-line p-4 text-meta text-seafoam" role="status">
          {t("Deploy the app to apply these changes.")}
        </p>
      )}
      {ports?.length === 0 && composePortRows.length === 0 && <p className="p-4 text-meta text-ink-soft">{t("No published ports, so this app cannot be reached from outside the server.")}</p>}
      <ul>
        {ports?.filter((port) => !composePortRows.some((composePort) => composePort.host_port === port.host_port && composePort.container_port === port.container_port && composePort.protocol === port.protocol && composePort.bind === port.bind)).map((port) => <PortRow port={port} address={host || window.location.hostname} url={url} locked={locked} canWrite={canWrite} onChanged={refresh} key={port.id} />)}
        {composePortRows.map((port) => <ComposePortRow port={port} address={host || window.location.hostname} file={composeFile} path={path} serviceId={service.id} locked={locked} canWrite={canWrite} onChanged={onChanged} key={port.id} />)}
      </ul>
      {canWrite &&
        (adding ? (
          <AddPortForm
            url={url}
            defaultContainerPort={detail?.port ? String(detail.port) : ""}
            defaultBind={service.category === "database" ? "127.0.0.1" : "0.0.0.0"}
            onCancel={() => setAdding(false)}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ) : (
          <div className="border-t border-line p-4">
            <button className="btn" type="button" disabled={locked} title={locked ? t("Wait for the current action to finish") : undefined} onClick={() => setAdding(true)}>
              {t("Publish port")}
            </button>
          </div>
        ))}
    </AppSection>
  );
}
