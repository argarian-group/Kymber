import { useState } from "react";
import { FormError } from "@/components/ui/FormError";
import { LoadError } from "@/components/ui/LoadError";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { PanelDomainForm } from "@/features/settings/components/PanelDomainForm";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";
import { Field } from "@/components/ui/Field";
import { useAuth, type User } from "@/features/auth/AuthContext";

export function SystemSettings({ usersOnly = false }: { usersOnly?: boolean }) {
  const [enabled, setEnabled] = useState<boolean | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [attempt, setAttempt] = useState(0);
  const { pending, error, run } = useAction();
  const [users, setUsers] = useState<User[] | null>(null);
  const [creatingUser, setCreatingUser] = useState(false);
  const [form, setForm] = useState({ username: "", email: "", password: "", is_root: false });
  const { signIn, state } = useAuth();

  useLoadEffect(
    () => api<{ registration_enabled: boolean }>("/settings/registration"),
    (response) => setEnabled(response.registration_enabled),
    setLoadError,
    [attempt],
  );

  useLoadEffect(() => api<{ users: User[] }>("/settings/users"), (response) => setUsers(response.users), setLoadError, []);

  function toggle(next: boolean) {
    return run(async () => {
      const response = await api<{ registration_enabled: boolean }>("/settings/registration", { method: "PUT", body: { enabled: next } });
      setEnabled(response.registration_enabled);
    });
  }

  if (loadError && (usersOnly || enabled === null)) {
    return (
      <LoadError
        title={t("Could not load system settings")}
        message={loadError}
        onRetry={() => {
          setLoadError(null);
          setAttempt((value) => value + 1);
        }}
      />
    );
  }

  return (
    <>
      {!usersOnly && <SettingsSection title={t("Access")}>
        <label className="flex cursor-pointer items-center justify-between gap-6 p-4">
          <span>
            <span className="block font-medium">{t("Allow new registrations")}</span>
            <span className="mt-0.5 block text-meta text-ink-soft">{t("When this is off, new accounts cannot be created. Existing accounts keep working.")}</span>
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
      </SettingsSection>}
      {!usersOnly && <SettingsSection title={t("Panel domain")} description={t("Optional custom hostname for the Kymber dashboard. DNS must point to this server and ports 80 and 443 must be open.")}>
        <PanelDomainForm />
      </SettingsSection>}
      <section className="px-4 py-6 md:px-6">
        <div className="mb-3 flex items-start justify-between gap-6">
          <h2 className="section-title">{t("Users")}</h2>
          <div className="flex flex-1 items-center justify-between gap-4">
            <p className="text-meta text-ink-soft">{t("Manage accounts on this server.")}</p>
            <button className="btn btn-primary" type="button" onClick={() => setCreatingUser(true)}>{t("Create user")}</button>
          </div>
        </div>
        <div className="rounded-panel bg-raised">
          <ul className="divide-y divide-line">{(users ?? []).map((user) => <ManagedUser key={user.id} user={user} onChanged={(next) => {
            setUsers((current) => current?.map((item) => item.id === next.id ? next : item) ?? null);
            if (state.status === "authenticated" && state.user.id === next.id) signIn(next);
          }} onDeleted={(id) => setUsers((current) => current?.filter((item) => item.id !== id) ?? null)} />)}</ul>
        </div>
        <div>
          {creatingUser && <div className="fixed inset-0 z-50 grid place-items-center bg-black/60 p-4" onMouseDown={(event) => event.target === event.currentTarget && setCreatingUser(false)}>
          <form className="w-full max-w-xl rounded-panel border border-line-strong bg-raised p-5 shadow-xl" role="dialog" aria-modal="true" aria-labelledby="create-user-title" onSubmit={(event) => {
            event.preventDefault();
            run(async () => {
              const response = await api<{ user: User }>("/settings/users", { method: "POST", body: form });
              setUsers((current) => [...(current ?? []), response.user]);
              setForm({ username: "", email: "", password: "", is_root: false });
              setCreatingUser(false);
            });
          }}>
            <h2 id="create-user-title" className="mb-4 text-lg font-semibold">{t("Create user")}</h2>
            <div className="grid gap-3 md:grid-cols-2">
              <Field label={t("Username")} required minLength={3} maxLength={32} value={form.username} onChange={(event) => setForm({ ...form, username: event.target.value })} />
              <Field label={t("Email")} type="email" required value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
              <Field label={t("Password")} type="password" required minLength={12} maxLength={256} value={form.password} onChange={(event) => setForm({ ...form, password: event.target.value })} />
              <label className="flex items-center gap-2"><input type="checkbox" checked={form.is_root} onChange={(event) => setForm({ ...form, is_root: event.target.checked })} />{t("Root account")}</label>
            </div>
            {error && <div className="mt-3"><FormError message={error} /></div>}
            <div className="mt-5 flex justify-end gap-2">
              <button className="btn" type="button" onClick={() => setCreatingUser(false)}>{t("Cancel")}</button>
              <button className="btn btn-primary" type="submit" disabled={pending}>{t("Create user")}</button>
            </div>
          </form>
          </div>}
        </div>
      </section>
    </>
  );
}

function ManagedUser({ user, onChanged, onDeleted }: { user: User; onChanged: (user: User) => void; onDeleted: (id: string) => void }) {
  const [expanded, setExpanded] = useState(false);
  const [password, setPassword] = useState("");
  const [username, setUsername] = useState(user.username);
  const [email, setEmail] = useState(user.email);
  const { pending, error, run } = useAction();
  return <li className="border-b border-line last:border-b-0">
    <button className="focus-ring flex min-h-14 w-full items-center justify-between gap-4 px-4 text-left hover:bg-white/[0.03]" type="button" aria-expanded={expanded} onClick={() => setExpanded((value) => !value)}>
      <span className="min-w-0"><span className="block truncate font-medium">{user.username}</span><span className="block truncate text-meta text-ink-soft">{user.email}</span></span>
      <span className="flex flex-none items-center gap-3 text-meta text-ink-soft">{user.is_root && t("Root account")}<span aria-hidden="true">{expanded ? "−" : "+"}</span></span>
    </button>
    {expanded && <div className="grid gap-3 border-t border-line p-4">
    <form className="grid items-end gap-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_auto_auto]" onSubmit={(event) => {
      event.preventDefault();
      run(async () => onChanged((await api<{ user: User }>(`/settings/users/${user.id}`, { method: "PUT", body: { username, email } })).user));
    }}>
      <Field label={t("Username")} required minLength={3} maxLength={32} value={username} onChange={(event) => setUsername(event.target.value)} />
      <Field label={t("Email")} type="email" required value={email} onChange={(event) => setEmail(event.target.value)} />
       <button className="btn h-10" type="submit" disabled={pending}>{t("Save")}</button>
       <button className="btn btn-danger h-10" type="button" disabled={pending} onClick={() => window.confirm(t("Delete {name}?", { name: user.username })) && run(async () => {
        await api(`/settings/users/${user.id}`, { method: "DELETE" });
        onDeleted(user.id);
      })}>{t("Delete")}</button>
      {error && <FormError message={error} />}
    </form>
    <form className="grid items-end gap-3 md:grid-cols-[minmax(0,1fr)_auto]" onSubmit={(event) => {
      event.preventDefault();
      run(async () => {
        await api(`/settings/users/${user.id}/password`, { method: "PUT", body: { password } });
        setPassword("");
      });
    }}>
      <Field label={t("New password")} type="password" required minLength={8} maxLength={128} value={password} onChange={(event) => setPassword(event.target.value)} />
      <button className="btn h-10" type="submit" disabled={pending}>{t("Reset password")}</button>
      {error && <FormError message={error} />}
    </form>
    </div>}
  </li>;
}
