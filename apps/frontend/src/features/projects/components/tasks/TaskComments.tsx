import { type FormEvent, useState } from "react";
import { RichTextEditor } from "@/components/richtext/RichTextEditor";
import { FormError } from "@/components/ui/FormError";
import { TaskCommentRow } from "@/features/projects/components/tasks/TaskCommentRow";
import { type TaskComment } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

const commentLimit = 5000;

export function TaskComments({ comments, path, taskId, onChanged }: { comments: TaskComment[]; path: string; taskId: string; onChanged: () => void }) {
  const [body, setBody] = useState("");
  const [editorKey, setEditorKey] = useState(0);
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    const sent = await run(async () => {
      await api(`${path}/tasks/${taskId}/comments`, { method: "POST", body: { body } });
      onChanged();
    });
    if (sent) {
      setBody("");
      setEditorKey((value) => value + 1);
    }
  }

  return (
    <section aria-labelledby="task-comments">
      <h3 id="task-comments" className="mb-2 font-medium">
        Comments
      </h3>
      <form className="grid grid-cols-1 gap-2" onSubmit={submit}>
        <RichTextEditor label="New comment" placeholder="Write a comment" initial="" limit={commentLimit} size="min-h-20" value={body} onChange={setBody} key={editorKey} />
        <FormError message={error} />
        {body && (
          <div>
            <button className="btn btn-primary" type="submit" disabled={pending || body.length > commentLimit}>
              Comment
            </button>
          </div>
        )}
      </form>
      <ul className="mt-4 grid grid-cols-1 gap-4">
        {[...comments].reverse().map((comment) => (
          <TaskCommentRow comment={comment} path={path} onChanged={onChanged} key={comment.id} />
        ))}
      </ul>
    </section>
  );
}
