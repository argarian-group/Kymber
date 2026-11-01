import { LogOut } from "lucide-react";
import { useAuth } from "@/features/auth/AuthContext";
import { useAction } from "@/lib/useAction";

export function LogoutButton({ className = "" }: { className?: string }) {
  const { signOut } = useAuth();
  const { pending, error, run } = useAction();

  return (
    <>
      <button className={`btn ${className}`} type="button" onClick={() => run(signOut)} disabled={pending}>
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
