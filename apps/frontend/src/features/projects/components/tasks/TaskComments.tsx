import { type FormEvent, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { TaskCommentRow } from "@/features/projects/components/tasks/TaskCommentRow";
import { type TaskComment } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function TaskComments({ comments, path, taskId, onChanged }: { comments: TaskComment[]; path: string; taskId: string; onChanged: () => void }) {
  const [body, setBody] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const sent = await run(async () => {
      await api(`${path}/tasks/${taskId}/comments`, { method: "POST", body: { body } });
      onChanged();
    });
    if (sent) setBody("");
  }

  return (
    <section aria-labelledby="task-comments">
      <h3 id="task-comments" className="mb-2 font-medium">
        Comments
      </h3>
      <form className="grid gap-2" onSubmit={submit}>
        <textarea className="field min-h-16 py-2" aria-label="New comment" placeholder="Write a comment" required maxLength={5000} value={body} onChange={(event) => setBody(event.target.value)} />
        <FormError message={error} />
        {body.trim() && (
          <div>
            <button className="btn btn-primary" type="submit" disabled={pending}>
              Comment
            </button>
          </div>
        )}
      </form>
      <ul className="mt-4 grid gap-4">
        {[...comments].reverse().map((comment) => (
          <TaskCommentRow comment={comment} path={path} onChanged={onChanged} key={comment.id} />
        ))}
      </ul>
    </section>
  );
}
