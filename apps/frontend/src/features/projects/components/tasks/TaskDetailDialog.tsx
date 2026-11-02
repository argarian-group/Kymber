import { X } from "lucide-react";
import { useRef, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { Modal } from "@/components/ui/Modal";
import { SelectField } from "@/components/ui/SelectField";
import { priorities, type Priority } from "@/features/dashboard/priorities";
import { useProjects } from "@/features/projects/ProjectsContext";
import { AddInline } from "@/features/projects/components/tasks/AddInline";
import { TaskChecklistCard } from "@/features/projects/components/tasks/TaskChecklistCard";
import { TaskComments } from "@/features/projects/components/tasks/TaskComments";
import { TaskDescription } from "@/features/projects/components/tasks/TaskDescription";
import { TaskLabelPicker } from "@/features/projects/components/tasks/TaskLabelPicker";
import { tasksInList, type Board, type TaskDetail } from "@/features/projects/tasks";
import { useProjectOutlet } from "@/features/projects/useProjectOutlet";
import { type Member } from "@/features/workspaces/types";
import { api, errorMessage } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type TaskDetailDialogProps = {
  taskId: string;
  board: Board;
  path: string;
  onMove: (taskId: string, listId: string, position: number) => void;
  onChanged: () => void;
  onClose: () => void;
};

type TaskEdit = Pick<TaskDetail, "title" | "description" | "priority" | "due_date"> & { assignee_ids: string[]; label_ids: string[] };

function editOf(task: TaskDetail): TaskEdit {
  return { title: task.title, description: task.description, priority: task.priority, due_date: task.due_date, assignee_ids: task.assignees.map(({ id }) => id), label_ids: task.label_ids };
}

export function TaskDetailDialog({ taskId, board, path, onMove, onChanged, onClose }: TaskDetailDialogProps) {
  const { workspace } = useProjectOutlet();
  const { can } = useProjects();
  const canDeleteTasks = can("tasks.delete");
  const [detail, setDetail] = useState<TaskDetail | null>(null);
  const [edit, setEdit] = useState<TaskEdit | null>(null);
  const latest = useRef<TaskEdit | null>(null);
  const queue = useRef<Promise<void>>(Promise.resolve());
  const [members, setMembers] = useState<Member[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [confirming, setConfirming] = useState(false);
  const removal = useAction();
  const taskPath = `${path}/tasks/${taskId}`;

  useLoadEffect(
    () => api<{ task: TaskDetail }>(taskPath),
    ({ task }) => {
      setDetail(task);
      if (!latest.current) {
        latest.current = editOf(task);
        setEdit(latest.current);
      }
    },
    setError,
    [taskPath, attempt],
  );

  useLoadEffect(
    () => api<{ members: Member[] }>(`/workspaces/${workspace.id}/members`),
    (response) => setMembers(response.members),
    setError,
    [workspace.id],
  );

  function refresh() {
    setAttempt((value) => value + 1);
    onChanged();
  }

  function save(patch: Partial<TaskEdit>) {
    if (!latest.current) return;
    const next = { ...latest.current, ...patch };
    latest.current = next;
    setEdit(next);
    setError(null);
    queue.current = queue.current.then(async () => {
      try {
        await api(taskPath, { method: "PUT", body: next });
        onChanged();
      } catch (failure) {
        setError(errorMessage(failure));
        latest.current = null;
        setAttempt((value) => value + 1);
      }
    });
  }

  function toggle(list: "assignee_ids" | "label_ids", id: string, checked: boolean) {
    const current = latest.current?.[list] ?? [];
    save({ [list]: checked ? [...current, id] : current.filter((value) => value !== id) });
  }

  async function confirmDelete() {
    const deleted = await removal.run(async () => {
      await api(taskPath, { method: "DELETE" });
      onChanged();
    });
    if (deleted) onClose();
    else setConfirming(false);
  }

  const boardTask = board.tasks.find(({ id }) => id === taskId);
  const listId = boardTask?.list_id ?? "";
  const siblings = tasksInList(board.tasks, listId);
  const position = siblings.findIndex(({ id }) => id === taskId);

  return (
    <Modal
      label={edit?.title ?? t("Task")}
      className="max-h-[calc(100dvh-2rem)] max-w-3xl"
      onCancel={() => {
        if (document.activeElement instanceof HTMLElement) document.activeElement.blur();
      }}
      onClose={onClose}
    >
      {!edit || !detail ? (
        <div className="grid gap-3 p-4">
          <FormError message={error} />
          {!error && <p className="text-ink-soft">{t("Loading task…")}</p>}
          <div>
            <button className="btn" type="button" onClick={onClose}>
              {t("Close")}
            </button>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 p-4">
          <div className="flex items-start gap-2">
            <textarea
              className="focus-ring min-w-0 flex-1 resize-none rounded-sm bg-transparent page-title field-sizing-content"
              aria-label={t("Task title")}
              rows={1}
              maxLength={200}
              key={edit.title}
              defaultValue={edit.title}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  event.currentTarget.blur();
                }
              }}
              onBlur={(event) => {
                const title = event.target.value.trim();
                if (title && title !== edit.title) save({ title });
                else event.target.value = edit.title;
              }}
            />
            <button className="btn px-2" type="button" aria-label={t("Close task")} onClick={onClose}>
              <X aria-hidden="true" size={16} />
            </button>
          </div>
          <FormError message={error} />
          <div className="grid grid-cols-1 gap-6 md:grid-cols-[minmax(0,1fr)_230px]">
            <div className="grid min-w-0 grid-cols-1 content-start gap-6">
              <TaskDescription value={edit.description} onSave={(description) => save({ description })} />
              {detail.checklists.map((checklist) => (
                <TaskChecklistCard checklist={checklist} path={path} onChanged={refresh} key={checklist.id} />
              ))}
              <AddInline
                label={t("Add checklist")}
                placeholder={t("Checklist title")}
                maxLength={120}
                onAdd={async (title) => {
                  await api(`${taskPath}/checklists`, { method: "POST", body: { title } });
                  refresh();
                }}
              />
              <TaskComments comments={detail.comments} path={path} taskId={taskId} onChanged={refresh} />
            </div>
            <aside className="grid min-w-0 grid-cols-1 content-start gap-4">
              <SelectField label={t("List")} value={listId} onChange={(event) => onMove(taskId, event.target.value, tasksInList(board.tasks, event.target.value).length)}>
                {[...board.lists]
                  .sort((a, b) => a.position - b.position)
                  .map(({ id, name }) => (
                    <option value={id} key={id}>
                      {name}
                    </option>
                  ))}
              </SelectField>
              <SelectField label={t("Position")} value={position} onChange={(event) => onMove(taskId, listId, Number(event.target.value))}>
                {siblings.map((sibling, index) => (
                  <option value={index} key={sibling.id}>
                    {index + 1}
                  </option>
                ))}
              </SelectField>
              <SelectField label={t("Priority")} value={edit.priority} onChange={(event) => save({ priority: event.target.value as Priority })}>
                {Object.entries(priorities).map(([id, { label }]) => (
                  <option value={id} key={id}>
                    {label}
                  </option>
                ))}
              </SelectField>
              <Field
                label={t("Due date")}
                type="date"
                defaultValue={edit.due_date ?? ""}
                onChange={(event) => {
                  const value = event.target.value;
                  if (!value) save({ due_date: null });
                  else if (Number(value.slice(0, 4)) >= 1000) save({ due_date: value });
                }}
              />
              <fieldset>
                <legend className="mb-1 text-meta font-medium">{t("Assignees")}</legend>
                <ul className="grid gap-1">
                  {members.map((member) => (
                    <li key={member.user_id}>
                      <label className="flex min-h-8 cursor-pointer items-center gap-2">
                        <input className="size-4 flex-none cursor-pointer accent-seafoam" type="checkbox" checked={edit.assignee_ids.includes(member.user_id)} onChange={(event) => toggle("assignee_ids", member.user_id, event.target.checked)} />
                        <span className="min-w-0 truncate">{member.username}</span>
                      </label>
                    </li>
                  ))}
                </ul>
              </fieldset>
              <TaskLabelPicker labels={board.labels} selected={edit.label_ids} path={path} onToggle={(id, checked) => toggle("label_ids", id, checked)} onChanged={refresh} />
              {canDeleteTasks && (
                <div>
                  <FormError message={removal.error} />
                  <button className="btn btn-danger mt-1" type="button" onClick={confirming ? confirmDelete : () => setConfirming(true)} disabled={removal.pending}>
                    {confirming ? t("Confirm delete") : t("Delete task")}
                  </button>
                </div>
              )}
            </aside>
          </div>
        </div>
      )}
    </Modal>
  );
}
