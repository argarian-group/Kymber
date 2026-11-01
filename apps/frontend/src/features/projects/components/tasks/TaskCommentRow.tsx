import { type FormEvent, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { useProjects } from "@/features/projects/ProjectsContext";
import { type TaskComment } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { formatRelative } from "@/lib/format";
import { useAction } from "@/lib/useAction";

export function TaskCommentRow({ comment, path, onChanged }: { comment: TaskComment; path: string; onChanged: () => void }) {
  const user = useCurrentUser();
  const { canWrite } = useProjects();
  const [editing, setEditing] = useState(false);
  const [body, setBody] = useState(comment.body);
  const { pending, error, run } = useAction();
  const isAuthor = comment.author?.id === user.id;
  const name = comment.author?.username ?? "Deleted user";

  async function save(event: FormEvent) {
    event.preventDefault();
    const saved = await run(async () => {
      await api(`${path}/task-comments/${comment.id}`, { method: "PUT", body: { body } });
      onChanged();
    });
    if (saved) setEditing(false);
  }

  return (
    <li className="flex gap-2.5">
      <span className="flex size-7 flex-none items-center justify-center rounded-full bg-white/10 text-xs font-semibold" aria-hidden="true">
        {name[0]}
      </span>
      <div className="min-w-0 flex-1">
        <p className="text-meta">
          <span className="font-medium">{name}</span> <span className="text-ink-soft">{formatRelative(comment.created_at)}</span>
          {comment.updated_at !== comment.created_at && <span className="text-ink-soft"> (edited)</span>}
        </p>
        {editing ? (
          <form className="mt-1 grid gap-2" onSubmit={save}>
            <textarea className="field min-h-20 py-2" aria-label="Edit comment" required maxLength={5000} value={body} onChange={(event) => setBody(event.target.value)} />
            <FormError message={error} />
            <div className="flex gap-2">
              <button className="btn btn-primary" type="submit" disabled={pending}>
                Save
              </button>
              <button className="btn" type="button" onClick={() => setEditing(false)}>
                Cancel
              </button>
            </div>
          </form>
        ) : (
          <>
            <p className="mt-1 wrap-break-word whitespace-pre-wrap">{comment.body}</p>
            <FormError message={error} />
            <div className="mt-1 flex gap-3 text-xs">
              {isAuthor && (
                <button className="text-button" type="button" onClick={() => setEditing(true)}>
                  Edit
                </button>
              )}
              {(isAuthor || canWrite) && (
                <button
                  className="text-button"
                  type="button"
                  onClick={() =>
                    run(async () => {
                      await api(`${path}/task-comments/${comment.id}`, { method: "DELETE" });
                      onChanged();
                    })
                  }
                >
                  Delete
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </li>
  );
}
