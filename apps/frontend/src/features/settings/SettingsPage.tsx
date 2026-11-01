import { useEffect, useState, type FormEvent } from "react";
import { Navigate, NavLink } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { Panel } from "@/components/ui/Panel";
import { useAuth, useCurrentUser, type User } from "@/features/auth/AuthContext";
import { Field, FormError } from "@/features/auth/AuthScreen";
import { api, errorMessage } from "@/lib/api";

const tabs = [
  { label: "Account", to: "/settings" },
  { label: "System", to: "/settings/system" },
];

function Saved({ message }: { message: string | null }) {
  return (
    <p className="min-h-5 text-meta text-ink-soft" role="status">
      {message}
    </p>
  );
}

function ProfileForm() {
  const user = useCurrentUser();
  const { signIn } = useAuth();
  const [username, setUsername] = useState(user.username);
  const [email, setEmail] = useState(user.email);
  const [currentPassword, setCurrentPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError(null);
    setSaved(null);
    try {
      const response = await api<{ user: User }>("/account/profile", {
        method: "PUT",
        body: { username, email, current_password: currentPassword },
      });
      signIn(response.user);
      setUsername(response.user.username);
      setEmail(response.user.email);
      setCurrentPassword("");
      setSaved("Profile saved.");
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  return (
    <Panel title="Profile">
      <form className="grid max-w-md gap-4 px-4 md:px-6" onSubmit={submit}>
        <Field label="Username" autoComplete="username" required minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={username} onChange={(event) => setUsername(event.target.value)} />
        <Field label="Email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
        <Field label="Current password" type="password" autoComplete="current-password" required value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
        <FormError message={error} />
        <div className="flex items-center gap-3">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Saving…" : "Save profile"}
          </button>
          <Saved message={saved} />
        </div>
      </form>
    </Panel>
  );
}

function PasswordForm() {
  const [form, setForm] = useState({ current: "", next: "", confirm: "" });
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    if (form.next !== form.confirm) {
      setError("The new passwords do not match.");
      return;
    }
    setPending(true);
    setError(null);
    try {
      await api("/account/password", { method: "PUT", body: { current_password: form.current, new_password: form.next } });
      setForm({ current: "", next: "", confirm: "" });
      setSaved("Password changed. Other devices were logged out.");
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  const update = (key: keyof typeof form) => (event: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [key]: event.target.value });

  return (
    <Panel title="Password">
      <form className="grid max-w-md gap-4 px-4 md:px-6" onSubmit={submit}>
        <Field label="Current password" type="password" autoComplete="current-password" required value={form.current} onChange={update("current")} />
        <Field label="New password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.next} onChange={update("next")} />
        <Field label="Confirm new password" type="password" autoComplete="new-password" required value={form.confirm} onChange={update("confirm")} />
        <FormError message={error} />
        <div className="flex flex-wrap items-center gap-3">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Changing…" : "Change password"}
          </button>
          <Saved message={saved} />
        </div>
      </form>
    </Panel>
  );
}

function SystemSettings() {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    let current = true;
    api<{ registration_enabled: boolean }>("/settings/registration")
      .then((response) => current && setEnabled(response.registration_enabled))
      .catch((failure) => current && setLoadError(errorMessage(failure)));
    return () => {
      current = false;
    };
  }, [attempt]);

  async function toggle(next: boolean) {
    setPending(true);
    setError(null);
    try {
      const response = await api<{ registration_enabled: boolean }>("/settings/registration", { method: "PUT", body: { enabled: next } });
      setEnabled(response.registration_enabled);
    } catch (failure) {
      setError(errorMessage(failure));
    }
    setPending(false);
  }

  if (loadError) {
    return (
      <EmptyState
        title="Could not load system settings"
        description={loadError}
        action={
          <button
            className="btn"
            type="button"
            onClick={() => {
              setLoadError(null);
              setAttempt((value) => value + 1);
            }}
          >
            Try again
          </button>
        }
      />
    );
  }

  return (
    <Panel title="Access">
      <div className="max-w-2xl px-4 md:px-6">
        <label className="flex cursor-pointer items-start justify-between gap-6 rounded-panel bg-raised p-4">
          <span>
            <span className="block font-medium">Allow new registrations</span>
            <span className="mt-0.5 block text-meta text-ink-soft">When this is off, new accounts cannot be created. Existing accounts keep working.</span>
          </span>
          <input
            className="mt-0.5 size-5 flex-none cursor-pointer accent-seafoam disabled:cursor-not-allowed"
            type="checkbox"
            role="switch"
            checked={enabled ?? false}
            disabled={enabled === null || pending}
            onChange={(event) => toggle(event.target.checked)}
          />
        </label>
        <div className="mt-2">
          <FormError message={error} />
        </div>
      </div>
    </Panel>
  );
}

export function SettingsPage({ section }: { section: "account" | "system" }) {
  const user = useCurrentUser();
  if (section === "system" && !user.is_root) return <Navigate to="/settings" replace />;

  return (
    <>
      <PageHeader title="Settings" description={section === "account" ? "Your profile and password." : "Options for everyone on this server."} />
      {user.is_root && (
        <nav className="mt-3 flex gap-0.5 px-4 md:px-6" aria-label="Settings sections">
          {tabs.map(({ label, to }) => (
            <NavLink
              className={({ isActive }) =>
                `focus-ring flex min-h-9 items-center rounded-control px-3 text-meta font-medium transition-colors duration-100 md:min-h-8 ${isActive ? "bg-white/10 text-ink" : "text-ink-soft hover:bg-white/5 hover:text-ink"}`
              }
              to={to}
              end
              key={to}
            >
              {label}
            </NavLink>
          ))}
        </nav>
      )}
      {section === "account" ? (
        <>
          <ProfileForm />
          <PasswordForm />
        </>
      ) : (
        <SystemSettings />
      )}
    </>
  );
}
