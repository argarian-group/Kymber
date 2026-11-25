import { type FormEvent, useState } from "react";
import { RichTextEditor } from "@/components/richtext/RichTextEditor";
import { FormError } from "@/components/ui/FormError";
import { TaskActivityRow } from "@/features/projects/components/tasks/TaskActivityRow";
import { TaskCommentRow } from "@/features/projects/components/tasks/TaskCommentRow";
import { commentLimit, type TaskActivity, type TaskComment } from "@/features/projects/tasks";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function TaskComments({ comments, activity, path, taskId, onChanged }: { comments: TaskComment[]; activity: TaskActivity[]; path: string; taskId: string; onChanged: () => void }) {
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
        {t("Comments and activity")}
      </h3>
      <form className="grid grid-cols-1 gap-2" onSubmit={submit}>
        <RichTextEditor label={t("New comment")} placeholder={t("Write a comment")} initial="" limit={commentLimit} size="min-h-20" value={body} onChange={setBody} key={editorKey} />
        <FormError message={error} />
        {body && (
          <div>
            <button className="btn btn-primary" type="submit" disabled={pending || body.length > commentLimit}>
              {t("Comment")}
            </button>
          </div>
        )}
      </form>
      <ul className="mt-4 grid grid-cols-1 gap-4">
        {[...[...comments].reverse().map((comment) => ({ at: comment.created_at, comment })), ...[...activity].reverse().map((item) => ({ at: item.created_at, item }))]
          .sort((a, b) => b.at.localeCompare(a.at))
          .map((entry) => ("comment" in entry ? <TaskCommentRow comment={entry.comment} path={path} onChanged={onChanged} key={`c-${entry.comment.id}`} /> : <TaskActivityRow activity={entry.item} key={`a-${entry.item.id}`} />))}
      </ul>
    </section>
  );
}
