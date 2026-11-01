import { ArrowLeft, ArrowRight } from "lucide-react";
import { type FormEvent, useEffect, useRef, useState } from "react";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { useProjects } from "@/features/projects/ProjectsContext";
import { type TaskListColumn } from "@/features/projects/tasks";
import { useAction } from "@/lib/useAction";

type TaskListDialogProps = {
  list: TaskListColumn;
  index: number;
  count: number;
  onSave: (input: { name: string; is_done: boolean }) => Promise<void>;
  onDelete: () => Promise<void>;
  onMove: (offset: number) => void;
  onClose: () => void;
};

export function TaskListDialog({ list, index, count, onSave, onDelete, onMove, onClose }: TaskListDialogProps) {
  const { canWrite } = useProjects();
  const dialog = useRef<HTMLDialogElement>(null);
  const [name, setName] = useState(list.name);
  const [isDone, setIsDone] = useState(list.is_done);
  const [confirming, setConfirming] = useState(false);
  const { pending, error, run } = useAction();

  useEffect(() => {
    dialog.current?.showModal();
  }, []);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (await run(() => onSave({ name, is_done: isDone }))) onClose();
  }

  async function confirmDelete() {
    if (await run(onDelete)) onClose();
    else setConfirming(false);
  }

  return (
    <dialog ref={dialog} className="m-auto w-[calc(100%-2rem)] max-w-md rounded-panel bg-raised p-0 text-ink backdrop:bg-black/60" aria-label="Edit list" onClose={onClose}>
      <form className="grid gap-4 p-4" onSubmit={submit}>
        <h2 className="section-title">Edit list</h2>
        <Field label="Name" required autoFocus maxLength={80} value={name} onChange={(event) => setName(event.target.value)} />
        <label className="flex cursor-pointer items-start gap-2">
          <input className="mt-1 accent-seafoam" type="checkbox" checked={isDone} onChange={(event) => setIsDone(event.target.checked)} />
          <span>
            <span className="block text-meta font-medium">Tasks here are done</span>
            <span className="block text-xs text-ink-soft">They stop showing as late.</span>
          </span>
        </label>
        <div className="flex gap-2">
          <button className="btn" type="button" disabled={index === 0} onClick={() => { onMove(-1); onClose(); }}>
            <ArrowLeft aria-hidden="true" size={14} />
            Move left
          </button>
          <button className="btn" type="button" disabled={index === count - 1} onClick={() => { onMove(1); onClose(); }}>
            Move right
            <ArrowRight aria-hidden="true" size={14} />
          </button>
        </div>
        <FormError message={error} />
        <div className="flex flex-wrap items-center gap-2">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            Save list
          </button>
          <button className="btn" type="button" onClick={onClose} disabled={pending}>
            Cancel
          </button>
          {canWrite && (
            <button className="btn btn-danger ml-auto" type="button" onClick={confirming ? confirmDelete : () => setConfirming(true)} disabled={pending}>
              {confirming ? "Confirm delete" : "Delete"}
            </button>
          )}
        </div>
      </form>
    </dialog>
  );
}
