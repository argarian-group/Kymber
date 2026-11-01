import { type LucideIcon } from "lucide-react";

type ToolbarButtonProps = {
  icon: LucideIcon;
  label: string;
  active?: boolean;
  disabled?: boolean;
  onClick: () => void;
};

export function ToolbarButton({ icon: Icon, label, active = false, disabled = false, onClick }: ToolbarButtonProps) {
  return (
    <button
      className={`focus-ring flex size-9 flex-none cursor-pointer items-center justify-center rounded-sm transition-colors duration-100 disabled:cursor-not-allowed disabled:opacity-40 md:size-7 ${active ? "bg-raised text-seafoam" : "text-ink-soft hover:bg-white/5 hover:text-ink"}`}
      type="button"
      title={label}
      aria-label={label}
      aria-pressed={active}
      disabled={disabled}
      onMouseDown={(event) => event.preventDefault()}
      onClick={onClick}
    >
      <Icon aria-hidden="true" size={15} />
    </button>
  );
}
