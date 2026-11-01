import { useState, type FormEvent } from "react";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { Field, FormError } from "@/features/auth/AuthScreen";
import { useProjects, type Workspace } from "@/features/projects/ProjectsContext";
import { errorMessage } from "@/lib/api";
import { FormFooter, SettingsSection } from "./SettingsSection";

function RenameWorkspace({ workspace }: { workspace: Workspace }) {
  const { canWrite, renameWorkspace } = useProjects();
  const [name, setName] = useState(workspace.name);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const changed = name.trim() !== workspace.name;

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    setSaved(null);
    try {
      await renameWorkspace(name);
      setName(name.trim());
      setSaved("Workspace renamed.");
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  return (
    <SettingsSection title="General" description={`${canWrite ? "The name everyone in this workspace sees." : "Only admins and owners can rename this workspace."}${workspace.created_by ? ` Created by ${workspace.created_by.username}.` : ""}`}>
      <form onSubmit={submit}>
        <div className="p-4 sm:max-w-sm">
          <Field label="Workspace name" required maxLength={120} disabled={!canWrite} value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        {canWrite && (
          <FormFooter error={error} saved={saved}>
            <button className="btn btn-primary" type="submit" disabled={pending || !changed}>
              {pending ? "Saving…" : "Save name"}
            </button>
          </FormFooter>
        )}
      </form>
    </SettingsSection>
  );
}

function DeleteWorkspace({ workspace }: { workspace: Workspace }) {
  const { deleteWorkspace } = useProjects();
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function confirmDelete() {
    setPending(true);
    setError(null);
    try {
      await deleteWorkspace();
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
      setConfirming(false);
    }
  }

  return (
    <SettingsSection title="Delete workspace" description="Delete its projects first.">
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <p className="min-w-0 wrap-break-word">
          {confirming ? `Delete ${workspace.name}? This cannot be undone.` : `Permanently delete ${workspace.name}.`}
        </p>
        {confirming ? (
          <span className="flex gap-2">
            <button className="btn btn-danger" type="button" onClick={confirmDelete} disabled={pending}>
              {pending ? "Deleting…" : "Confirm delete"}
            </button>
            <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
              Cancel
            </button>
          </span>
        ) : (
          <button className="btn btn-danger" type="button" onClick={() => setConfirming(true)}>
            Delete workspace
          </button>
        )}
      </div>
      {error && (
        <div className="border-t border-line px-4 py-3">
          <FormError message={error} />
        </div>
      )}
    </SettingsSection>
  );
}

function CreateWorkspace() {
  const { createWorkspace } = useProjects();
  const [name, setName] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      await createWorkspace(name);
      setName("");
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  return (
    <SettingsSection title="New workspace" description="You become its owner and switch to it.">
      <form onSubmit={submit}>
        <div className="p-4 sm:max-w-sm">
          <Field label="Workspace name" required maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
        </div>
        <FormFooter error={error} saved={null}>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Creating…" : "Create workspace"}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
  );
}

export function WorkspaceSettings() {
  const user = useCurrentUser();
  const { state, canDelete, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <EmptyState
        title="Could not load this workspace"
        description={state.message}
        action={
          <button className="btn" type="button" onClick={retry}>
            Try again
          </button>
        }
      />
    );
  }

  return (
    <>
      {state.workspace ? (
        <>
          <RenameWorkspace workspace={state.workspace} key={`rename-${state.workspace.id}`} />
          {canDelete && <DeleteWorkspace workspace={state.workspace} key={`delete-${state.workspace.id}`} />}
        </>
      ) : (
        !user.is_root && <EmptyState title="No workspace yet" description="Ask an administrator to add you to a workspace." />
      )}
      {user.is_root && <CreateWorkspace />}
    </>
  );
}
