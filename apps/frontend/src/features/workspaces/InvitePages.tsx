import { useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router";
import { useAuth, useCurrentUser, type User } from "@/features/auth/AuthContext";
import { AuthScreen, Field, FormError } from "@/features/auth/AuthScreen";
import { LoginPage } from "@/features/auth/LoginPage";
import { rememberWorkspace, useProjects } from "@/features/projects/ProjectsContext";
import { api, ApiError, errorMessage } from "@/lib/api";

type Acceptance = { user: User; workspace_id: string; role: string };

export function InviteSignupPage() {
  const { signIn } = useAuth();
  const token = window.location.hash.slice(1);
  const [form, setForm] = useState({ username: "", password: "" });
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loggingIn, setLoggingIn] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (confirmPassword !== form.password) {
      setError("The passwords do not match.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      const { user, workspace_id } = await api<Acceptance>("/invitations/accept", { method: "POST", body: { token, ...form } });
      rememberWorkspace(user.id, workspace_id);
      window.history.replaceState(null, "", "/projects");
      signIn(user);
    } catch (failure) {
      if (failure instanceof ApiError && failure.status === 401) {
        setLoggingIn(true);
        return;
      }
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  if (loggingIn) return <LoginPage />;

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <AuthScreen title="Join a workspace" description="Create your account to accept this invitation. It uses the email the invitation was sent for.">
      <form className="grid gap-4" onSubmit={submit}>
        <Field label="Username" autoComplete="username" required autoFocus minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={form.username} onChange={update("username")} />
        <Field label="Password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.password} onChange={update("password")} />
        <Field label="Confirm password" type="password" autoComplete="new-password" required value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} />
        <FormError message={error} />
        <button className="btn btn-primary justify-center" type="submit" disabled={pending}>
          {pending ? "Accepting…" : "Create account and join"}
        </button>
      </form>
      <p className="mt-4 text-meta text-ink-soft">
        Already have an account?{" "}
        <button className="focus-ring cursor-pointer rounded-sm font-medium text-ink underline underline-offset-2" type="button" onClick={() => setLoggingIn(true)}>
          Log in
        </button>
      </p>
    </AuthScreen>
  );
}

export function InviteAcceptPage() {
  const user = useCurrentUser();
  const { selectWorkspace } = useProjects();
  const navigate = useNavigate();
  const token = useLocation().hash.slice(1);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function accept() {
    setPending(true);
    setError(null);
    try {
      const { workspace_id } = await api<Acceptance>("/invitations/accept", { method: "POST", body: { token } });
      selectWorkspace(workspace_id);
      navigate("/projects", { replace: true });
    } catch (failure) {
      setError(errorMessage(failure));
      setPending(false);
    }
  }

  return (
    <div className="max-w-md px-4 py-8 md:px-6">
      <h1 className="font-display text-xl font-semibold tracking-[-0.01em]">Join a workspace</h1>
      <p className="mt-1 mb-4 text-ink-soft">The invitation must have been created for {user.email}.</p>
      <FormError message={error} />
      <div className="mt-3 flex flex-wrap gap-2">
        <button className="btn btn-primary" type="button" onClick={accept} disabled={pending}>
          {pending ? "Accepting…" : "Accept invitation"}
        </button>
        <Link className="btn" to="/">
          Not now
        </Link>
      </div>
    </div>
  );
}
