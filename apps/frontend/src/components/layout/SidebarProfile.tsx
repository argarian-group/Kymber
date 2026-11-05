import { ChevronsUpDown } from "lucide-react";
import { useEffect, useId, useState } from "react";
import { useCurrentUser } from "@/features/auth/AuthContext";
import { LogoutButton } from "./LogoutButton";
import { t } from "@/lib/i18n";

async function gravatarUrl(email: string) {
  const normalizedEmail = new TextEncoder().encode(email.trim().toLowerCase());
  const digest = await globalThis.crypto?.subtle?.digest("SHA-256", normalizedEmail);
  if (!digest) throw new Error("Secure hashing is unavailable");
  const hash = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
  return `https://www.gravatar.com/avatar/${hash}?d=404&s=56`;
}

export function SidebarProfile({ compact = false }: { compact?: boolean }) {
  const user = useCurrentUser();
  const menuId = useId();
  const [avatarUrl, setAvatarUrl] = useState("");
  const [avatarFailed, setAvatarFailed] = useState(false);

  useEffect(() => {
    let active = true;
    gravatarUrl(user.email)
      .then((url) => active && setAvatarUrl(url))
      .catch(() => active && setAvatarFailed(true));
    return () => { active = false; };
  }, [user.email]);

  return (
    <div className={`flex-none ${compact ? "p-2" : "p-3"}`}>
      <button
        className={`flex w-full cursor-pointer items-center gap-3 rounded-control ${compact ? "justify-center p-1" : "p-2"} text-left transition-colors duration-150 hover:bg-white/5 focus-ring`}
        type="button"
        title={compact ? user.username : undefined}
        popoverTarget={menuId}
        aria-haspopup="dialog"
      >
        {avatarUrl && !avatarFailed && <img
          className="size-7 flex-none rounded-full bg-seafoam object-cover"
          src={avatarUrl}
          alt=""
          width="28"
          height="28"
          onError={() => setAvatarFailed(true)}
        />}
        {(avatarFailed || !avatarUrl) && <span className="flex size-7 flex-none items-center justify-center rounded-full bg-seafoam text-xs font-semibold text-on-accent" aria-hidden="true">
          {user.username[0].toUpperCase()}
        </span>}
        <span className={compact ? "sr-only" : "min-w-0 flex-1"}>
          <strong className="block truncate text-meta font-medium">{user.username}</strong>
          <span className="block truncate text-xs text-ink-soft">{user.email}</span>
        </span>
        {!compact && <ChevronsUpDown className="flex-none text-ink-faint" aria-hidden="true" size={16} />}
      </button>
      <div
        id={menuId}
        className="pop-in inset-auto bottom-18 left-3 m-0 w-50 rounded-panel border border-line-strong bg-raised p-3 text-ink"
        popover="auto"
        role="dialog"
        aria-label={t("Profile")}
      >
        <p className="truncate text-meta font-medium">{user.username}</p>
        <p className="mb-3 truncate text-xs text-ink-soft">{user.is_root ? t("Root account") : user.email}</p>
        <LogoutButton className="w-full justify-center" />
      </div>
    </div>
  );
}
