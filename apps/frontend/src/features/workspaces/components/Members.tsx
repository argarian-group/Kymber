import { useCallback, useState } from "react";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects, type Workspace, workspaceLabel } from "@/features/projects/ProjectsContext";
import { InviteForm } from "@/features/workspaces/components/InviteForm";
import { MemberRow } from "@/features/workspaces/components/MemberRow";
import { type Member } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useLoadEffect } from "@/lib/useLoadEffect";

type MembersState = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; members: Member[] };

export function Members({ workspace }: { workspace: Workspace }) {
  const user = useCurrentUser();
  const { workspaces, canManageMembers, canInvite, retry: reloadWorkspaces } = useProjects();
  const [state, setState] = useState<MembersState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);
  const [adding, setAdding] = useState(false);

  useLoadEffect(
    () => api<{ members: Member[] }>(`/workspaces/${workspace.id}/members`),
    ({ members }) => setState({ status: "ready", members }),
    (message) => setState({ status: "error", message }),
    [attempt, workspace.id],
  );

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
          canInvite &&
          !adding && (
            <button className="btn btn-primary" type="button" onClick={() => setAdding(true)}>
              Invite member
            </button>
          )
        }
      />
      {adding && <InviteForm workspaceId={workspace.id} canAddDirectly={canManageMembers} onAdded={reload} onClose={() => setAdding(false)} />}
      {state.status === "loading" && <PageSkeleton />}
      {state.status === "error" && (
        <LoadError title="Could not load members" message={state.message} onRetry={reload} />
      )}
      {state.status === "ready" &&
        (state.members.length === 0 ? (
          <EmptyState title="No members yet" description={canInvite ? "Invite someone to this workspace." : "A workspace owner or admin can invite members."} />
        ) : (
          <ul className="mt-4 border-t border-line">
            {state.members.map((member) => (
              <MemberRow member={member} workspaceId={workspace.id} isCreator={workspace.created_by?.id === member.user_id} canManage={canManageMembers} onChanged={handleChanged} key={member.user_id} />
            ))}
          </ul>
        ))}
    </>
  );
}
