import { useCallback, useEffect, useState, type FormEvent } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { Field, FormError } from "@/features/auth/AuthScreen";
import { useProjects, type Workspace } from "@/features/projects/ProjectsContext";
import { workspaceLabel } from "@/features/projects/WorkspaceSwitcher";
import { api, errorMessage } from "@/lib/api";

type Member = { user_id: string; username: string; email: string; role: string; created_at: string };
type MembersState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; members: Member[] };

const roles = ["owner", "admin", "member"];
const dateFormat = new Intl.DateTimeFormat(undefined, { dateStyle: "medium" });

function RoleSelect({ id, label, value, disabled, onChange }: { id: string; label: string; value: string; disabled?: boolean; onChange: (role: string) => void }) {
  return (
    <>
      <label className="sr-only" htmlFor={id}>
        {label}
      </label>
      <select className="field cursor-pointer" id={id} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
        {roles.map((role) => (
          <option value={role} key={role}>
            {role}
          </option>
        ))}
      </select>
    </>
  );
}

function AddMemberForm({ workspaceId, onAdded, onClose }: { workspaceId: string; onAdded: () => void; onClose: () => void }) {
  const [email, setEmail] = useState("");
  const [role, setRole] = useState("member");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      await api(`/workspaces/${workspaceId}/members`, { method: "POST", body: { email, role } });
      onAdded();
      onClose();
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  return (
    <form className="mx-4 mt-4 rounded-panel bg-raised p-4 md:mx-6" onSubmit={submit} aria-label="Add member">
      <div className="grid items-end gap-4 sm:grid-cols-[minmax(0,1fr)_10rem]">
        <Field label="Email" type="email" required autoFocus hint="The person needs an existing Kymber account." value={email} onChange={(event) => setEmail(event.target.value)} />
        <div className="sm:self-start">
          <span className="mb-1 block text-meta font-medium" aria-hidden="true">
            Role
          </span>
          <RoleSelect id="new-member-role" label="Role" value={role} onChange={setRole} />
        </div>
      </div>
      <div className="mt-3">
        <FormError message={error} />
      </div>
      <div className="mt-2 flex gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Adding…" : "Add member"}
        </button>
        <button className="btn" type="button" onClick={onClose} disabled={pending}>
          Cancel
        </button>
      </div>
    </form>
  );
}

function MemberRow({ member, workspaceId, canManage, onChanged }: { member: Member; workspaceId: string; canManage: boolean; onChanged: (member: Member, removed: boolean) => void }) {
  const user = useCurrentUser();
  const isSelf = member.user_id === user.id;
  const [confirming, setConfirming] = useState(false);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function changeRole(role: string) {
    setPending(true);
    setError(null);
    try {
      const response = await api<{ member: Member }>(`/workspaces/${workspaceId}/members/${member.user_id}`, { method: "PUT", body: { role } });
      onChanged(response.member, false);
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  async function remove() {
    setPending(true);
    setError(null);
    try {
      await api(`/workspaces/${workspaceId}/members/${member.user_id}`, { method: "DELETE" });
      onChanged(member, true);
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
      setConfirming(false);
    }
  }

  return (
    <li className="flex flex-wrap items-center gap-x-4 gap-y-2 border-b border-line px-4 py-3 md:px-6">
      <div className="min-w-0 flex-1 basis-48">
        <p className="font-medium wrap-break-word">
          {member.username}
          {isSelf && <span className="ml-2 text-xs font-normal text-ink-soft">You</span>}
        </p>
        <p className="text-meta wrap-break-word text-ink-soft">{member.email}</p>
        <FormError message={error} />
      </div>
      <span className="text-meta text-ink-soft tabular-nums">Joined {dateFormat.format(new Date(member.created_at))}</span>
      {canManage ? (
        <>
          <div className="w-28">
            <RoleSelect id={`role-${member.user_id}`} label={`Role of ${member.username}`} value={member.role} disabled={pending} onChange={changeRole} />
          </div>
          {confirming ? (
            <span className="flex gap-2">
              <button className="btn btn-danger" type="button" onClick={remove} disabled={pending} aria-label={`Confirm remove ${member.username}`}>
                {pending ? "Removing…" : "Confirm remove"}
              </button>
              <button className="btn" type="button" onClick={() => setConfirming(false)} disabled={pending}>
                Cancel
              </button>
            </span>
          ) : (
            <button className="btn" type="button" onClick={() => setConfirming(true)} aria-label={`Remove ${member.username}`}>
              Remove
            </button>
          )}
        </>
      ) : (
        <span className="text-meta font-medium">{member.role}</span>
      )}
    </li>
  );
}

function Members({ workspace }: { workspace: Workspace }) {
  const user = useCurrentUser();
  const { workspaces, canManageMembers, retry: reloadWorkspaces } = useProjects();
  const [state, setState] = useState<MembersState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);

  useEffect(() => {
    let current = true;
    api<{ members: Member[] }>(`/workspaces/${workspace.id}/members`)
      .then(({ members }) => current && setState({ status: "ready", members }))
      .catch((error) => current && setState({ status: "error", message: errorMessage(error) }));
    return () => {
      current = false;
    };
  }, [attempt, workspace.id]);

  const reload = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);

  function handleChanged(member: Member, removed: boolean) {
    if (member.user_id === user.id) {
      reloadWorkspaces();
      return;
    }
    setState((previous) =>
      previous.status !== "ready"
        ? previous
        : { status: "ready", members: removed ? previous.members.filter(({ user_id }) => user_id !== member.user_id) : previous.members.map((item) => (item.user_id === member.user_id ? member : item)) },
    );
  }

  const count = state.status === "ready" ? state.members.length : null;
  return (
    <>
      <PageHeader
        title="Team"
        description={count === null ? workspaceLabel(workspace, workspaces) : `${workspaceLabel(workspace, workspaces)}, ${count} ${count === 1 ? "member" : "members"}`}
        actions={
          canManageMembers &&
          !adding && (
            <button className="btn btn-primary" type="button" onClick={() => setAdding(true)}>
              Add member
            </button>
          )
        }
      />
      {adding && <AddMemberForm workspaceId={workspace.id} onAdded={reload} onClose={() => setAdding(false)} />}
      {state.status === "loading" && <PageSkeleton />}
      {state.status === "error" && (
        <EmptyState
          title="Could not load members"
          description={state.message}
          action={
            <button className="btn" type="button" onClick={reload}>
              Try again
            </button>
          }
        />
      )}
      {state.status === "ready" &&
        (state.members.length === 0 ? (
          <EmptyState title="No members yet" description={canManageMembers ? "Add someone with an existing Kymber account." : "A workspace owner can add members."} />
        ) : (
          <ul className="mt-4 border-t border-line">
            {state.members.map((member) => (
              <MemberRow member={member} workspaceId={workspace.id} canManage={canManageMembers} onChanged={handleChanged} key={member.user_id} />
            ))}
          </ul>
        ))}
    </>
  );
}

export function TeamPage() {
  const { state, retry } = useProjects();

  if (state.status === "loading") return <PageSkeleton />;
  if (state.status === "error") {
    return (
      <>
        <PageHeader title="Team" description="The people in this workspace." />
        <EmptyState
          title="Could not load this workspace"
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
        <PageHeader title="Team" description="The people in this workspace." />
        <EmptyState title="No workspace yet" description="Members appear here once you have a workspace." />
      </>
    );
  }
  return <Members workspace={state.workspace} key={state.workspace.id} />;
}
