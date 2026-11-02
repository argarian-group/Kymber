import { useState, type FormEvent } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { SelectField } from "@/components/ui/SelectField";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { isServiceBusy, type ServiceStorage, type StorageKind } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

const kinds: Record<StorageKind, string> = { volume: "Volume", bind: "Directory mount", file: "File mount" };
const pathPattern = "(/[A-Za-z0-9_.\\-]+)+";
const pathTitle = "An absolute path using letters, numbers, dots, dashes and underscores";

function Row({ kind, title, detail, badge, children }: { kind: string; title: string; detail?: string; badge?: string; children?: React.ReactNode }) {
  return (
    <li className="grid gap-3 border-t border-line p-4 first:border-t-0">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-56">
          <p className="flex flex-wrap items-center gap-2 text-xs text-ink-soft">
            {kind}
            {badge && <span className="rounded-sm bg-seafoam/15 px-1.5 text-seafoam">{badge}</span>}
          </p>
          <code className="mt-1 block font-mono text-meta wrap-break-word">{title}</code>
          {detail && <code className="mt-0.5 block font-mono text-xs wrap-break-word text-ink-soft">{detail}</code>}
        </div>
      </div>
      {children}
    </li>
  );
}

function StorageRow({ storage, url, locked, canWrite, onChanged }: { storage: ServiceStorage; url: string; locked: boolean; canWrite: boolean; onChanged: () => void }) {
  const [mode, setMode] = useState<"view" | "delete" | "edit">("view");
  const [deleteData, setDeleteData] = useState(false);
  const [content, setContent] = useState<string | null>(null);
  const { pending, error, run } = useAction();
  const itemUrl = `${url}/${storage.id}`;

  async function edit() {
    setMode("edit");
    await run(async () => setContent((await api<{ content: string }>(itemUrl)).content));
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() => api(itemUrl, { method: "PUT", body: { content } }));
    if (saved) {
      setMode("view");
      onChanged();
    }
  }

  async function remove() {
    const removed = await run(() => api(deleteData ? `${itemUrl}?delete_data=true` : itemUrl, { method: "DELETE" }));
    if (removed) onChanged();
  }

  const detail = storage.kind === "volume" ? `Volume ${storage.volume_name}` : storage.kind === "bind" ? `Server path ${storage.source}` : "Content stored encrypted";

  return (
    <Row kind={kinds[storage.kind]} title={storage.mount_path} detail={detail} badge={storage.applied ? undefined : "Pending deploy"}>
      {canWrite && mode === "view" && (
        <div className="flex flex-wrap gap-2">
          {storage.kind === "file" && (
            <button className="btn" type="button" disabled={locked} onClick={edit}>
              Edit content
            </button>
          )}
          <button className="btn btn-danger" type="button" disabled={locked} onClick={() => setMode("delete")}>
            Remove
          </button>
        </div>
      )}
      {mode === "edit" && (
        <form className="grid gap-3" onSubmit={save}>
          <label className="grid gap-1 text-meta font-medium">
            File content
            <textarea className="field min-h-48 font-mono text-xs" spellCheck={false} disabled={content === null} value={content ?? ""} onChange={(event) => setContent(event.target.value)} />
          </label>
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-primary" type="submit" disabled={pending || content === null}>
              {pending ? "Saving…" : "Save content"}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setMode("view")}>
              Cancel
            </button>
          </div>
        </form>
      )}
      {mode === "delete" && (
        <div className="grid gap-3">
          <p>Remove this mount from {storage.mount_path}? It stops being mounted on the next deploy.</p>
          {storage.kind === "volume" && (
            <label className="flex cursor-pointer items-center gap-2 text-meta">
              <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={deleteData} onChange={(event) => setDeleteData(event.target.checked)} />
              Also delete the volume's data. This cannot be undone.
            </label>
          )}
          <div className="flex flex-wrap gap-2">
            <button className="btn btn-danger" type="button" disabled={pending} onClick={remove}>
              {pending ? "Removing…" : "Confirm removal"}
            </button>
            <button className="btn" type="button" disabled={pending} onClick={() => setMode("view")}>
              Cancel
            </button>
          </div>
        </div>
      )}
      <FormError message={error} />
    </Row>
  );
}

function AddStorageForm({ url, onAdded, onCancel }: { url: string; onAdded: () => void; onCancel: () => void }) {
  const user = useCurrentUser();
  const [kind, setKind] = useState<StorageKind>("volume");
  const [source, setSource] = useState("");
  const [mountPath, setMountPath] = useState("");
  const [content, setContent] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const added = await run(() => api(url, { method: "POST", body: { kind, source, mount_path: mountPath, content } }));
    if (added) onAdded();
  }

  return (
    <form className="grid gap-4 border-t border-line p-4" onSubmit={submit} aria-label="Add storage">
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <SelectField label="Type" autoFocus value={kind} onChange={(event) => setKind(event.target.value as StorageKind)}>
          <option value="volume">{kinds.volume}</option>
          {user.is_root && <option value="bind">{kinds.bind}</option>}
          <option value="file">{kinds.file}</option>
        </SelectField>
        <Field label={kind === "file" ? "File path in the container" : "Path in the container"} required maxLength={255} pattern={pathPattern} title={pathTitle} placeholder={kind === "file" ? "/etc/app/config.yml" : "/data"} value={mountPath} onChange={(event) => setMountPath(event.target.value)} />
        {kind === "bind" && (
          <div className="sm:col-span-2">
            <Field label="Path on the server" required maxLength={255} pattern={pathPattern} title={pathTitle} placeholder="/srv/app/data" hint="The container can read and write this directory on the server." value={source} onChange={(event) => setSource(event.target.value)} />
          </div>
        )}
      </div>
      {kind === "file" && (
        <label className="grid gap-1 text-meta font-medium">
          File content
          <textarea className="field min-h-48 font-mono text-xs" spellCheck={false} value={content} onChange={(event) => setContent(event.target.value)} />
        </label>
      )}
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Adding…" : "Add storage"}
        </button>
        <button className="btn" type="button" disabled={pending} onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}

export function AppStoragePage() {
  const { service, path, canWrite } = useAppOutlet();
  const [storages, setStorages] = useState<ServiceStorage[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);
  const [changed, setChanged] = useState(false);
  const url = `${path}/${service.id}/storages`;
  const locked = isServiceBusy(service.status);

  useLoadEffect(
    () => api<{ storages: ServiceStorage[] }>(url),
    (response) => {
      setStorages(response.storages);
      setLoadError(null);
    },
    setLoadError,
    [url, attempt],
  );

  function refresh() {
    setChanged(true);
    setAttempt((value) => value + 1);
  }

  if (loadError && !storages) return <LoadError title="Could not load storage" message={loadError} onRetry={() => setAttempt((value) => value + 1)} />;

  const empty = !service.data_path && storages?.length === 0;
  const pendingDeploy = changed || storages?.some(({ applied }) => !applied);

  return (
    <AppSection title="Storage" description="Keep data across deploys and give the container files and directories. Changes apply on the next deploy.">
      {pendingDeploy && (
        <p className="border-b border-line p-4 text-meta text-seafoam" role="status">
          Deploy the app to apply these changes.
        </p>
      )}
      {empty && <p className="p-4 text-meta text-ink-soft">No storage yet, so anything this app writes is lost on the next deploy.</p>}
      <ul>
        {service.data_path && <Row kind="Main volume" title={service.data_path} detail={`Volume kymber-service-${service.id}-data`} badge="Deleted with the app" />}
        {storages?.map((storage) => (
          <StorageRow storage={storage} url={url} locked={locked} canWrite={canWrite} onChanged={refresh} key={storage.id} />
        ))}
      </ul>
      {canWrite &&
        (adding ? (
          <AddStorageForm
            url={url}
            onCancel={() => setAdding(false)}
            onAdded={() => {
              setAdding(false);
              refresh();
            }}
          />
        ) : (
          <div className="border-t border-line p-4">
            <button className="btn" type="button" disabled={locked} title={locked ? "Wait for the current action to finish" : undefined} onClick={() => setAdding(true)}>
              Add storage
            </button>
          </div>
        ))}
    </AppSection>
  );
}
