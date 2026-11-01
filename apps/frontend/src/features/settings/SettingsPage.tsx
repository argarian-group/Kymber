import { useEffect, useId, useState, type FormEvent, type ReactNode } from "react";
import { Navigate, NavLink } from "react-router";
import { PageHeader } from "@/components/layout/PageHeader";
import { EmptyState } from "@/components/ui/EmptyState";
import { useAuth, useCurrentUser, type User } from "@/features/auth/AuthContext";
import { Field, FormError } from "@/features/auth/AuthScreen";
import { api, errorMessage } from "@/lib/api";

const tabs = [
  { label: "Account", to: "/settings" },
  { label: "System", to: "/settings/system" },
];

function SettingsSection({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  const titleId = useId();
  return (
    <section className="grid gap-x-10 gap-y-3 px-4 py-6 md:px-6 lg:grid-cols-[220px_minmax(0,1fr)]" aria-labelledby={titleId}>
      <div>
        <h2 id={titleId} className="font-display text-base font-semibold">
          {title}
        </h2>
        <p className="mt-1 text-meta text-ink-soft">{description}</p>
      </div>
      <div className="max-w-2xl min-w-0 rounded-panel bg-raised">{children}</div>
    </section>
  );
}

function FormFooter({ error, saved, children }: { error: string | null; saved: string | null; children: ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-x-4 gap-y-2 border-t border-line px-4 py-3">
      <div className="mr-auto min-w-0">
        <FormError message={error} />
        {!error && <Saved message={saved} />}
      </div>
      {children}
    </div>
  );
}

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
    <SettingsSection title="Profile" description="How you appear in Kymber and the email you log in with.">
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <Field label="Username" autoComplete="username" required minLength={3} maxLength={32} hint="3 to 32 letters, numbers, _ or -." value={username} onChange={(event) => setUsername(event.target.value)} />
          <Field label="Email" type="email" autoComplete="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
          <Field label="Current password" type="password" autoComplete="current-password" required hint="Needed to confirm these changes." value={currentPassword} onChange={(event) => setCurrentPassword(event.target.value)} />
        </div>
        <FormFooter error={error} saved={saved}>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Saving…" : "Save profile"}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
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
    <SettingsSection title="Password" description="Changing it logs you out on your other devices.">
      <form onSubmit={submit}>
        <div className="grid items-start gap-4 p-4 sm:grid-cols-2">
          <div className="sm:col-span-2 sm:max-w-[calc(50%-0.5rem)]">
            <Field label="Current password" type="password" autoComplete="current-password" required value={form.current} onChange={update("current")} />
          </div>
          <Field label="New password" type="password" autoComplete="new-password" required minLength={8} maxLength={128} hint="8 to 128 characters." value={form.next} onChange={update("next")} />
          <Field label="Confirm new password" type="password" autoComplete="new-password" required value={form.confirm} onChange={update("confirm")} />
        </div>
        <FormFooter error={error} saved={saved}>
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Changing…" : "Change password"}
          </button>
        </FormFooter>
      </form>
    </SettingsSection>
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
    <SettingsSection title="Access" description="Who can get into this server.">
      <label className="flex cursor-pointer items-center justify-between gap-6 p-4">
        <span>
          <span className="block font-medium">Allow new registrations</span>
          <span className="mt-0.5 block text-meta text-ink-soft">When this is off, new accounts cannot be created. Existing accounts keep working.</span>
        </span>
        <span className="relative flex-none">
          <input
            className="peer sr-only"
            type="checkbox"
            role="switch"
            checked={enabled ?? false}
            disabled={enabled === null || pending}
            onChange={(event) => toggle(event.target.checked)}
          />
          <span
            className="block h-6 w-10 rounded-full bg-white/20 transition-colors duration-150 peer-checked:bg-seafoam peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam peer-disabled:opacity-60 after:absolute after:top-0.5 after:left-0.5 after:size-5 after:rounded-full after:bg-ink after:transition-transform after:duration-150 peer-checked:after:translate-x-4 motion-reduce:after:transition-none"
            aria-hidden="true"
          />
        </span>
      </label>
      {error && (
        <div className="border-t border-line px-4 py-3">
          <FormError message={error} />
        </div>
      )}
    </SettingsSection>
  );
}

export function SettingsPage({ section }: { section: "account" | "system" }) {
  const user = useCurrentUser();
  if (section === "system" && !user.is_root) return <Navigate to="/settings" replace />;

  return (
    <>
      <PageHeader title="Settings" description={section === "account" ? "Your profile and password." : "Options for everyone on this server."} />
      {user.is_root && (
        <nav className="mt-4 flex gap-5 border-b border-line px-4 md:px-6" aria-label="Settings sections">
          {tabs.map(({ label, to }) => (
            <NavLink
              className={({ isActive }) =>
                `focus-ring -mb-px flex min-h-10 items-center border-b-2 text-meta font-medium transition-colors duration-100 ${isActive ? "border-seafoam text-ink" : "border-transparent text-ink-soft hover:text-ink"}`
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
