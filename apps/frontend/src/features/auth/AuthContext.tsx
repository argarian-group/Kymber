import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { api, ApiError, errorMessage, unauthorizedEvent } from "@/lib/api";

export type User = { id: number; username: string; email: string; is_root: boolean };

type AuthState =
  | { status: "loading" }
  | { status: "error"; message: string }
  | { status: "onboarding" }
  | { status: "anonymous" }
  | { status: "authenticated"; user: User };

type AuthContextValue = {
  state: AuthState;
  retry: () => void;
  signIn: (user: User) => void;
  signOut: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}

export function useCurrentUser() {
  const { state } = useAuth();
  if (state.status !== "authenticated") throw new Error("useCurrentUser requires a signed-in user");
  return state.user;
}

async function loadAuthState(): Promise<AuthState> {
  const { required } = await api<{ required: boolean }>("/onboarding");
  if (required) return { status: "onboarding" };
  try {
    const { user } = await api<{ user: User }>("/auth/me");
    return { status: "authenticated", user };
  } catch (error) {
    if (error instanceof ApiError && error.status === 401) return { status: "anonymous" };
    throw error;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>({ status: "loading" });
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let current = true;
    loadAuthState()
      .then((next) => current && setState(next))
      .catch((error) => current && setState({ status: "error", message: errorMessage(error) }));
    return () => {
      current = false;
    };
  }, [attempt]);

  useEffect(() => {
    const expire = () => setState((previous) => (previous.status === "authenticated" ? { status: "anonymous" } : previous));
    window.addEventListener(unauthorizedEvent, expire);
    return () => window.removeEventListener(unauthorizedEvent, expire);
  }, []);

  const retry = useCallback(() => {
    setState({ status: "loading" });
    setAttempt((value) => value + 1);
  }, []);
  const signIn = useCallback((user: User) => setState({ status: "authenticated", user }), []);
  const signOut = useCallback(async () => {
    await api("/auth/logout", { method: "POST" });
    setState({ status: "anonymous" });
  }, []);

  const value = useMemo(() => ({ state, retry, signIn, signOut }), [state, retry, signIn, signOut]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
