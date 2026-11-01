import { useState, type FormEvent } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { Field, FormError } from "@/features/auth/AuthScreen";
import { errorMessage } from "@/lib/api";
import { useProjects, type Project } from "./ProjectsContext";
import { workspaceLabel } from "./WorkspaceSwitcher";

export function slugify(name: string) {
  return name
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 120)
    .replace(/-+$/, "");
}

export function ProjectsPage() {
  const { state, workspaces, canWrite, retry } = useProjects();
  const [creatingIn, setCreatingIn] = useState<string | null>(null);

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <>
        <PageHeader title="Projects" description="Every project in one place." />
        <EmptyState
          title="Could not load projects"
          description={state.message}
          action={
            <button className="btn" type="button" onClick={retry}>
              Try again
            </button>
          }
        />
      </>
    );
  }
  if (!state.workspace) {
    return (
      <>
        <PageHeader title="Projects" description="Every project in one place." />
        <EmptyState title="No workspace yet" description="Ask an administrator to add you to a workspace." />
      </>
    );
  }

  const count = state.projects.length;
  const workspaceId = state.workspace.id;
  const creating = creatingIn === workspaceId;
  const startCreating = () => setCreatingIn(workspaceId);
  return (
    <>
      <PageHeader
        title="Projects"
        description={`${workspaceLabel(state.workspace, workspaces)}, ${count} ${count === 1 ? "project" : "projects"}`}
        actions={
          canWrite &&
          !creating && (
            <button className="btn btn-primary" type="button" onClick={startCreating}>
              New project
            </button>
          )
        }
      />
      {creating && <CreateProjectForm onClose={() => setCreatingIn(null)} />}
      {count === 0 && !creating ? (
        <EmptyState
          title="No projects yet"
          description={canWrite ? "Create the first project in this workspace." : "A workspace admin can create the first project."}
          action={
            canWrite && (
              <button className="btn" type="button" onClick={startCreating}>
                Create project
              </button>
            )
          }
        />
      ) : (
        <ul className="mt-4 border-t border-line">
          {state.projects.map((project) => (
            <ProjectRow project={project} key={project.id} />
          ))}
        </ul>
      )}
    </>
  );
}

function CreateProjectForm({ onClose }: { onClose: () => void }) {
  const { createProject } = useProjects();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const suggestedSlug = slugify(name);

  async function submit(event: FormEvent) {
    event.preventDefault();
    const finalSlug = slug.trim() || suggestedSlug;
    if (!finalSlug) {
      setError("Enter a slug using lowercase letters, numbers and hyphens.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      await createProject({ name, slug: finalSlug });
      onClose();
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  return (
    <form className="mx-4 mt-4 grid gap-4 rounded-panel bg-raised p-4 md:mx-6 md:grid-cols-2" onSubmit={submit} aria-label="New project">
      <Field label="Name" required autoFocus maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
      <Field label="Slug" maxLength={120} pattern="[a-z0-9\-]+" title="Lowercase letters, numbers and hyphens" placeholder={suggestedSlug} hint="Lowercase letters, numbers and hyphens." value={slug} onChange={(event) => setSlug(event.target.value)} />
      <div className="md:col-span-2">
        <FormError message={error} />
        <div className="mt-2 flex gap-2">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Creating…" : "Create project"}
          </button>
          <button className="btn" type="button" onClick={onClose} disabled={pending}>
            Cancel
          </button>
        </div>
      </div>
    </form>
  );
}

const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

function ProjectRow({ project }: { project: Project }) {
  const { canWrite, deleteProject } = useProjects();
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function confirmDelete() {
    setPending(true);
    setError(null);
    try {
      await deleteProject(project.id);
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
      setConfirming(false);
    }
  }

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line px-4 py-3 md:px-6">
      <div className="min-w-0 flex-1 basis-48">
        <p className="font-medium wrap-break-word">{project.name}</p>
        <p className="text-meta wrap-break-word text-ink-soft">{project.slug}</p>
        <FormError message={error} />
      </div>
      <span className="text-meta text-ink-soft tabular-nums">Created {dateFormat.format(new Date(project.created_at))}</span>
      {!canWrite ? null : confirming ? (
        <span className="flex gap-2">
          <button className="btn btn-danger" type="button" onClick={confirmDelete} disabled={pending} aria-label={`Confirm delete ${project.name}`}>
            {pending ? "Deleting…" : "Confirm delete"}
          </button>
          <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
            Cancel
          </button>
        </span>
      ) : (
        <button className="btn" type="button" onClick={() => setConfirming(true)} aria-label={`Delete ${project.name}`}>
          Delete
        </button>
      )}
    </li>
  );
}
