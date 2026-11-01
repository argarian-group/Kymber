import { ChevronsUpDown } from "lucide-react";
import { useId } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { LogoutButton } from "./LogoutButton";

export function SidebarProfile() {
  const user = useCurrentUser();
  const menuId = useId();

  return (
    <div className="flex-none p-3">
      <button
        className="flex w-full cursor-pointer items-center gap-3 rounded-control p-2 text-left transition-colors duration-150 hover:bg-white/5 focus-ring"
        type="button"
        popoverTarget={menuId}
        aria-haspopup="dialog"
      >
        <span
          className="flex size-7 flex-none items-center justify-center rounded-full bg-seafoam text-xs font-semibold text-on-accent"
          aria-hidden="true"
        >
          {user.username[0].toUpperCase()}
        </span>
        <span className="min-w-0 flex-1">
          <strong className="block truncate text-meta font-medium">{user.username}</strong>
          <span className="block truncate text-xs text-ink-soft">{user.email}</span>
        </span>
        <ChevronsUpDown className="flex-none text-ink-faint" aria-hidden="true" size={16} />
      </button>
      <div
        id={menuId}
        className="pop-in inset-auto bottom-18 left-3 m-0 w-50 rounded-panel border border-line-strong bg-raised p-3 text-ink"
        popover="auto"
        role="dialog"
        aria-label="Profile"
      >
        <p className="truncate text-meta font-medium">{user.username}</p>
        <p className="mb-3 truncate text-xs text-ink-soft">{user.is_root ? "Root account" : user.email}</p>
        <LogoutButton className="w-full justify-center" />
      </div>
    </div>
  );
}
