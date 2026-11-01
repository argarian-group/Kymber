import { ChevronsUpDown } from "lucide-react";
import { useId } from "react";

type SidebarProfileProps = { name: string; role: string };

export function SidebarProfile({ name, role }: SidebarProfileProps) {
  const menuId = useId();
  const initials = name
    .split(" ")
    .slice(0, 2)
    .map((word) => word[0])
    .join("")
    .toUpperCase();

  return (
    <div className="flex-none border-t border-hairline p-3">
      <button
        className="flex w-full cursor-pointer items-center gap-3 rounded-[10px] p-2 text-left transition-colors duration-150 hover:bg-white/5 focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-mist"
        type="button"
        popoverTarget={menuId}
        aria-haspopup="dialog"
      >
        <span
          className="flex size-8 flex-none items-center justify-center rounded-full bg-seafoam text-xs font-semibold text-deep-pine"
          aria-hidden="true"
        >
          {initials}
        </span>
        <span className="min-w-0 flex-1">
          <strong className="block truncate font-semibold">{name}</strong>
          <span className="block truncate text-meta text-ink-soft">{role}</span>
        </span>
        <ChevronsUpDown className="flex-none text-ink-faint" aria-hidden="true" size={16} />
      </button>
      <div
        id={menuId}
        className="pop-in inset-auto bottom-18 left-3 m-0 h-48 w-56 rounded-panel border border-hairline bg-surface text-ink shadow-[0_8px_24px_rgb(0_20_17/0.4)]"
        popover="auto"
        role="dialog"
        aria-label="Profile"
      />
    </div>
  );
}
