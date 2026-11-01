import { Plus } from "lucide-react";
import { type FormEvent, useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { useAction } from "@/lib/useAction";

type AddInlineProps = {
  label: string;
  placeholder: string;
  maxLength: number;
  onAdd: (value: string) => Promise<void>;
  keepOpen?: boolean;
};

export function AddInline({ label, placeholder, maxLength, onAdd, keepOpen = false }: AddInlineProps) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!value.trim()) return;
    const added = await run(() => onAdd(value));
    if (!added) return;
    setValue("");
    if (!keepOpen) setOpen(false);
  }

  if (!open) {
    return (
      <button className="focus-ring flex min-h-9 w-full cursor-pointer items-center gap-1.5 rounded-control px-2 text-left text-meta text-ink-soft transition-colors duration-100 hover:bg-white/5 hover:text-ink" type="button" onClick={() => setOpen(true)}>
        <Plus aria-hidden="true" size={14} />
        {label}
      </button>
    );
  }

  return (
    <form className="grid gap-2" onSubmit={submit}>
      <input
        className="field"
        aria-label={placeholder}
        autoFocus
        maxLength={maxLength}
        placeholder={placeholder}
        value={value}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            event.preventDefault();
            event.stopPropagation();
            setOpen(false);
          }
        }}
      />
      <FormError message={error} />
      <div className="flex gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          Add
        </button>
        <button className="btn" type="button" onClick={() => setOpen(false)} disabled={pending}>
          Cancel
        </button>
      </div>
    </form>
  );
}
