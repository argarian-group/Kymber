import { ChevronRight, Download, File, Folder, FolderPlus, HardDrive, FilePlus, Pencil, RefreshCw, Trash2 } from "lucide-react";
import { lazy, Suspense, useState, type FormEvent } from "react";
import { Link, Navigate, useSearchParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

const CodeEditor = lazy(() => import("@/features/projects/components/files/CodeEditor"));

type FileEntry = { name: string; type: "directory" | "file" | "other"; symlink?: boolean; size: number; modified: number; mode: string };
type FileMount = { path: string; kind: "volume" | "bind" | "file" };
type Listing = { path: string; entries: FileEntry[]; truncated: boolean; mounts: FileMount[] };
type OpenFile = { path: string; binary: boolean; modified: string; content?: string };
type Prompt = { kind: "file" | "directory" } | { kind: "rename"; entry: FileEntry } | { kind: "delete"; entry: FileEntry };

const join = (directory: string, name: string) => (directory === "/" ? `/${name}` : `${directory}/${name}`);
const mountOf = (target: string, mounts: FileMount[]) => mounts.find((mount) => target === mount.path || target.startsWith(`${mount.path}/`));
const validName = (name: string) => name !== "" && name !== "." && name !== ".." && !/[/\n\0]/.test(name);

function formatSize(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 ** 2) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / 1024 ** 2).toFixed(1)} MB`;
}

function Durability({ target, mounts }: { target: string; mounts: FileMount[] }) {
  const mount = mountOf(target, mounts);
  if (!mount) return <p className="text-xs text-ember">Not in a volume or mount: changes here are lost on the next deploy.</p>;
  if (mount.kind === "file")
    return (
      <p className="text-xs text-ink-soft">
        Managed in <Link className="text-button" to="../storage">Storage</Link>, which rewrites this file on every deploy.
      </p>
    );
  return <p className="text-xs text-seafoam">Persistent: stored in {mount.path}, kept across deploys.</p>;
}

export function AppFilesPage() {
  const { service, path, canWrite } = useAppOutlet();
  const [params, setParams] = useSearchParams();
  const directory = params.get("path") ?? "/";
  const [listing, setListing] = useState<Listing | null>(null);
  const [listError, setListError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [open, setOpen] = useState<OpenFile | null>(null);
  const [revision, setRevision] = useState(0);
  const [draft, setDraft] = useState("");
  const [saved, setSaved] = useState(false);
  const [prompt, setPrompt] = useState<Prompt | null>(null);
  const [name, setName] = useState("");
  const editor = useAction();
  const action = useAction();
  const url = `${path}/${service.id}/files`;
  const running = service.status === "running";
  const dirty = open?.content !== undefined && draft !== open.content;

  useLoadEffect(
    () => (running && canWrite ? api<Listing>(`${url}?path=${encodeURIComponent(directory)}`) : Promise.resolve(null)),
    (response) => {
      setListing(response);
      setListError(null);
    },
    setListError,
    [url, directory, attempt, running],
  );

  if (!canWrite) return <Navigate to=".." replace />;
  if (!running) return <EmptyState title="The app is not running" description="Start or deploy the app to browse and edit its files." />;

  const mounts = listing?.mounts ?? [];
  const crumbs = directory.split("/").filter(Boolean);
  const reload = () => setAttempt((value) => value + 1);
  const canLeave = () => !dirty || window.confirm("Discard your unsaved changes?");
  const managedFile = open ? mountOf(open.path, mounts)?.kind === "file" : false;

  function go(target: string) {
    if (!canLeave()) return;
    setOpen(null);
    setPrompt(null);
    setParams(target === "/" ? {} : { path: target });
  }

  async function openFile(target: string) {
    if (target === open?.path || !canLeave()) return;
    setSaved(false);
    await editor.run(async () => {
      const file = await api<OpenFile>(`${url}/content?path=${encodeURIComponent(target)}`);
      setOpen(file);
      setDraft(file.content ?? "");
      setRevision((value) => value + 1);
    });
  }

  async function save() {
    if (!open || !dirty || managedFile || editor.pending) return;
    setSaved(false);
    const content = draft;
    await editor.run(async () => {
      const { modified } = await api<{ modified: string }>(`${url}/content`, { method: "PUT", body: { path: open.path, content, modified: open.modified } });
      setOpen({ ...open, content, modified });
      setSaved(true);
      reload();
    });
  }

  function download() {
    if (!open) return;
    const link = document.createElement("a");
    link.href = URL.createObjectURL(new Blob([draft], { type: "text/plain" }));
    link.download = open.path.split("/").pop() ?? "file";
    link.click();
    URL.revokeObjectURL(link.href);
  }

  function ask(next: Prompt) {
    setPrompt(next);
    setName(next.kind === "rename" ? next.entry.name : "");
    action.setError(null);
  }

  async function confirm(event: FormEvent) {
    event.preventDefault();
    if (!prompt) return;
    if (prompt.kind !== "delete" && !validName(name)) {
      action.setError("Enter a name without slashes.");
      return;
    }
    const entryPath = "entry" in prompt ? join(directory, prompt.entry.name) : null;
    const affectsOpen = Boolean(open && entryPath && (open.path === entryPath || open.path.startsWith(`${entryPath}/`)));
    if (affectsOpen && !canLeave()) return;
    const done = await action.run(async () => {
      if (prompt.kind === "delete") await api(`${url}?path=${encodeURIComponent(join(directory, prompt.entry.name))}`, { method: "DELETE" });
      else if (prompt.kind === "rename") await api(`${url}/move`, { method: "POST", body: { from: join(directory, prompt.entry.name), to: join(directory, name) } });
      else await api(url, { method: "POST", body: { path: join(directory, name), kind: prompt.kind } });
    });
    if (!done) return;
    if (affectsOpen) setOpen(null);
    setPrompt(null);
    reload();
  }

  return (
    <AppSection title="Files" description="Browse and edit the files inside the running container.">
      <div className="grid gap-3 border-b border-line p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="flex items-center gap-1 text-xs text-ink-soft">
            <HardDrive aria-hidden="true" size={13} />
            Persistent
          </span>
          {mounts.filter(({ kind }) => kind !== "file").map((mount) => (
            <button className="btn min-h-7 px-2 font-mono text-xs" type="button" onClick={() => go(mount.path)} key={mount.path}>
              {mount.path}
            </button>
          ))}
          {mounts.filter(({ kind }) => kind !== "file").length === 0 && <span className="text-xs text-ink-faint">No volumes yet. Add them in Storage.</span>}
        </div>
        <nav className="flex flex-wrap items-center gap-0.5 font-mono text-meta" aria-label="Current folder">
          <button className="text-button px-1" type="button" onClick={() => go("/")}>/</button>
          {crumbs.map((crumb, index) => (
            <span className="flex items-center gap-0.5" key={index}>
              {index > 0 && <ChevronRight className="text-ink-faint" aria-hidden="true" size={13} />}
              <button className="text-button px-1 break-all" type="button" onClick={() => go(`/${crumbs.slice(0, index + 1).join("/")}`)}>{crumb}</button>
            </span>
          ))}
        </nav>
        <Durability target={directory} mounts={mounts} />
      </div>
      <div className="grid min-w-0 xl:grid-cols-[minmax(280px,360px)_minmax(0,1fr)]">
        <div className="min-w-0 border-b border-line xl:border-r xl:border-b-0">
          <div className="flex flex-wrap gap-1 border-b border-line p-2">
            <button className="btn min-h-8 px-2 text-meta" type="button" onClick={() => ask({ kind: "file" })}>
              <FilePlus aria-hidden="true" size={14} /> New file
            </button>
            <button className="btn min-h-8 px-2 text-meta" type="button" onClick={() => ask({ kind: "directory" })}>
              <FolderPlus aria-hidden="true" size={14} /> New folder
            </button>
            <button className="btn ml-auto size-8 justify-center px-0" type="button" aria-label="Refresh files" title="Refresh" onClick={reload}>
              <RefreshCw aria-hidden="true" size={14} />
            </button>
          </div>
          {prompt && (
            <form className="grid gap-2 border-b border-line p-3" onSubmit={confirm}>
              {prompt.kind === "delete" ? (
                <p className="text-meta wrap-break-word">Delete {prompt.entry.name}{prompt.entry.type === "directory" ? " and everything in it" : ""}? This cannot be undone.</p>
              ) : (
                <label className="grid gap-1 text-meta font-medium">
                  {prompt.kind === "rename" ? `Rename ${prompt.entry.name}` : prompt.kind === "file" ? "File name" : "Folder name"}
                  <input className="field font-mono" autoFocus required maxLength={255} value={name} onChange={(event) => setName(event.target.value)} />
                </label>
              )}
              <FormError message={action.error} />
              <div className="flex gap-2">
                <button className={`btn min-h-8 ${prompt.kind === "delete" ? "btn-danger" : "btn-primary"}`} type="submit" disabled={action.pending}>
                  {prompt.kind === "delete" ? "Confirm delete" : prompt.kind === "rename" ? "Rename" : "Create"}
                </button>
                <button className="btn min-h-8" type="button" disabled={action.pending} onClick={() => setPrompt(null)}>Cancel</button>
              </div>
            </form>
          )}
          {listError && (
            <div className="p-3">
              <FormError message={listError} />
              {directory !== "/" && <button className="text-button mt-2 text-meta" type="button" onClick={() => go("/")}>Go to the root folder</button>}
            </div>
          )}
          {!listError && !listing && <p className="p-4 text-meta text-ink-soft">Loading files…</p>}
          {!listError && listing && (
            <ul className="max-h-[32rem] overflow-auto py-1" aria-label={`Files in ${directory}`}>
              {directory !== "/" && (
                <li>
                  <button className="flex min-h-9 w-full items-center gap-2 px-3 text-left font-mono text-meta hover:bg-white/5" type="button" onClick={() => go(directory.slice(0, directory.lastIndexOf("/")) || "/")}>
                    <Folder className="text-ink-faint" aria-hidden="true" size={15} /> ..
                  </button>
                </li>
              )}
              {listing.entries.length === 0 && <li className="p-3 text-meta text-ink-soft">This folder is empty.</li>}
              {listing.entries.map((entry) => {
                const target = join(directory, entry.name);
                const isDirectory = entry.type === "directory";
                return (
                  <li className={`group flex min-w-0 items-center hover:bg-white/5 ${open?.path === target ? "bg-white/8" : ""}`} key={entry.name}>
                    <button
                      className="flex min-h-9 min-w-0 flex-1 items-center gap-2 px-3 py-1 text-left disabled:cursor-default disabled:opacity-60"
                      type="button"
                      disabled={entry.type === "other"}
                      title={entry.type === "other" ? "Special files cannot be opened" : `${entry.mode} · ${formatRelative(new Date(entry.modified * 1000).toISOString())}`}
                      onClick={() => (isDirectory ? go(target) : openFile(target))}
                    >
                      {isDirectory ? <Folder className="flex-none text-seafoam" aria-hidden="true" size={15} /> : <File className="flex-none text-ink-faint" aria-hidden="true" size={15} />}
                      <span className="min-w-0 flex-1 truncate font-mono text-meta">{entry.name}{entry.symlink ? " →" : ""}</span>
                      {!isDirectory && <span className="flex-none text-xs text-ink-faint tabular-nums">{formatSize(entry.size)}</span>}
                    </button>
                    <button className="flex size-8 flex-none items-center justify-center text-ink-soft hover:text-ink" type="button" aria-label={`Rename ${entry.name}`} title="Rename" onClick={() => ask({ kind: "rename", entry })}>
                      <Pencil aria-hidden="true" size={13} />
                    </button>
                    <button className="flex size-8 flex-none items-center justify-center text-ink-soft hover:text-ember" type="button" aria-label={`Delete ${entry.name}`} title="Delete" onClick={() => ask({ kind: "delete", entry })}>
                      <Trash2 aria-hidden="true" size={13} />
                    </button>
                  </li>
                );
              })}
              {listing.truncated && <li className="p-3 text-xs text-ink-soft">Only the first 2000 items are shown.</li>}
            </ul>
          )}
        </div>
        <div className="flex min-w-0 flex-col">
          {open ? (
            <>
              <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-line p-3">
                <div className="min-w-0 flex-1 basis-52">
                  <code className="block font-mono text-meta wrap-break-word">{open.path}{dirty ? " •" : ""}</code>
                  <Durability target={open.path} mounts={mounts} />
                </div>
                {!open.binary && (
                  <div className="flex gap-2">
                    <button className="btn size-9 justify-center px-0" type="button" aria-label="Download file" title="Download" onClick={download}>
                      <Download aria-hidden="true" size={15} />
                    </button>
                    {!managedFile && (
                      <button className="btn btn-primary" type="button" disabled={!dirty || editor.pending} title="Ctrl+S" onClick={save}>
                        {editor.pending ? "Saving…" : "Save"}
                      </button>
                    )}
                  </div>
                )}
              </div>
              {editor.error && (
                <div className="border-b border-line p-3">
                  <FormError message={editor.error} />
                </div>
              )}
              {saved && !dirty && !editor.error && <p className="border-b border-line p-3 text-meta text-ink-soft" role="status">Saved. The app may need a restart to use the new file.</p>}
              {open.binary ? (
                <p className="p-4 text-meta text-ink-soft">This is a binary file, so it cannot be edited here.</p>
              ) : (
                <div className="h-[32rem] min-h-0">
                  <Suspense fallback={<p className="p-4 text-meta text-ink-soft">Loading editor…</p>}>
                    <CodeEditor filename={open.path.split("/").pop() ?? ""} value={open.content ?? ""} readOnly={managedFile} label={`Contents of ${open.path}`} onChange={setDraft} onSave={save} key={revision} />
                  </Suspense>
                </div>
              )}
            </>
          ) : (
            <div className="p-4">
              <FormError message={editor.error} />
              <p className="text-meta text-ink-soft">{editor.pending ? "Opening file…" : "Choose a file to view or edit it."}</p>
            </div>
          )}
        </div>
      </div>
    </AppSection>
  );
}
