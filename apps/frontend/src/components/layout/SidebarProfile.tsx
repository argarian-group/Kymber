import { ChevronsUpDown, LogOut } from "lucide-react";
import { useId, useState } from "react";
import { useAuth, useCurrentUser } from "@/features/auth/AuthContext";
import { errorMessage } from "@/lib/api";

export function LogoutButton({ className = "" }: { className?: string }) {
  const { signOut } = useAuth();
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function logOut() {
    setPending(true);
    setError(null);
    try {
      await signOut();
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  return (
    <>
      <button className={`btn ${className}`} type="button" onClick={logOut} disabled={pending}>
        <LogOut aria-hidden="true" size={14} />
        {pending ? "Logging out…" : "Log out"}
      </button>
      {error && (
        <p className="mt-2 text-xs text-ember" role="alert">
          {error}
        </p>
      )}
    </>
  );
}

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
          className="flex size-7 flex-none items-center justify-center rounded-full bg-raised text-xs font-semibold text-ink"
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
