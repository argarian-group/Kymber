import { type FormEvent, useEffect, useRef, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { SelectField } from "@/components/ui/SelectField";
import { type Priority } from "@/features/dashboard/data";
import { priorities } from "@/features/dashboard/priorities";
import { useProjects } from "@/features/projects/ProjectsContext";
import { taskStatuses, type Task, type TaskInput, type TaskStatus } from "@/features/projects/tasks";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { type Member } from "@/features/workspaces/types";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type TaskDialogProps = {
  task: Task | null;
  onSave: (input: TaskInput) => Promise<void>;
  onDelete: () => Promise<void>;
  onClose: () => void;
};

export function TaskDialog({ task, onSave, onDelete, onClose }: TaskDialogProps) {
  const { workspace } = useProjectOutlet();
  const { canWrite } = useProjects();
  const dialog = useRef<HTMLDialogElement>(null);
  const [title, setTitle] = useState(task?.title ?? "");
  const [status, setStatus] = useState<TaskStatus>(task?.status ?? "todo");
  const [priority, setPriority] = useState<Priority>(task?.priority ?? "normal");
  const [assigneeId, setAssigneeId] = useState(task?.assignee?.id ?? "");
  const [dueDate, setDueDate] = useState(task?.due_date ?? "");
  const [members, setMembers] = useState<Member[]>([]);
  const [confirming, setConfirming] = useState(false);
  const { pending, error, setError, run } = useAction();

  useEffect(() => {
    dialog.current?.showModal();
  }, []);

  useLoadEffect(
    () => api<{ members: Member[] }>(`/workspaces/${workspace.id}/members`),
    (response) => setMembers(response.members),
    setError,
    [workspace.id],
  );

  async function submit(event: FormEvent) {
    event.preventDefault();
    const saved = await run(() => onSave({ title, status, priority, assignee_id: assigneeId || null, due_date: dueDate || null }));
    if (saved) onClose();
  }

  async function confirmDelete() {
    const deleted = await run(onDelete);
    if (deleted) onClose();
    else setConfirming(false);
  }

  const hasCurrentAssignee = !task?.assignee || members.some(({ user_id }) => user_id === task.assignee?.id);

  return (
    <dialog ref={dialog} className="m-auto w-[calc(100%-2rem)] max-w-lg rounded-panel bg-raised p-0 text-ink backdrop:bg-black/60" aria-label={task ? "Edit task" : "New task"} onClose={onClose}>
      <form className="grid gap-4 p-4 md:grid-cols-2" onSubmit={submit}>
        <h2 className="section-title md:col-span-2">{task ? "Edit task" : "New task"}</h2>
        <div className="md:col-span-2">
          <Field label="Title" required autoFocus maxLength={200} value={title} onChange={(event) => setTitle(event.target.value)} />
        </div>
        <SelectField label="Status" value={status} onChange={(event) => setStatus(event.target.value as TaskStatus)}>
          {taskStatuses.map(({ id, name }) => (
            <option value={id} key={id}>
              {name}
            </option>
          ))}
        </SelectField>
        <SelectField label="Priority" value={priority} onChange={(event) => setPriority(event.target.value as Priority)}>
          {Object.entries(priorities).map(([id, { label }]) => (
            <option value={id} key={id}>
              {label}
            </option>
          ))}
        </SelectField>
        <SelectField label="Assignee" value={assigneeId} onChange={(event) => setAssigneeId(event.target.value)}>
          <option value="">Unassigned</option>
          {!hasCurrentAssignee && <option value={task?.assignee?.id}>{task?.assignee?.username}</option>}
          {members.map((member) => (
            <option value={member.user_id} key={member.user_id}>
              {member.username}
            </option>
          ))}
        </SelectField>
        <Field label="Due date" type="date" value={dueDate} onChange={(event) => setDueDate(event.target.value)} />
        <div className="md:col-span-2">
          <FormError message={error} />
          <div className="mt-2 flex flex-wrap items-center gap-2">
            <button className="btn btn-primary" type="submit" disabled={pending}>
              {pending ? "Saving…" : task ? "Save task" : "Create task"}
            </button>
            <button className="btn" type="button" onClick={onClose} disabled={pending}>
              Cancel
            </button>
            {task && canWrite && (
              <button className="btn btn-danger ml-auto" type="button" onClick={confirming ? confirmDelete : () => setConfirming(true)} disabled={pending}>
                {confirming ? "Confirm delete" : "Delete"}
              </button>
            )}
          </div>
        </div>
      </form>
    </dialog>
  );
}
