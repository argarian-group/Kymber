import { type FormEvent, useState } from "react";
import { CopyField } from "@/components/ui/CopyField";
import { EmptyState } from "@/components/ui/EmptyState";
import { Field } from "@/components/ui/Field";
import { LoadError } from "@/components/ui/LoadError";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { SelectField } from "@/components/ui/SelectField";
import { type GitProvider, type GitProviderKind, providerKindLabel } from "@/features/projects/gitProviders";
import { useProjects } from "@/features/projects/ProjectsContext";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { api, errorMessage } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

function ProviderRow({ provider, path, canWrite, onChanged }: { provider: GitProvider; path: string; canWrite: boolean; onChanged: () => void }) {
  const { pending, error, run } = useAction();
  const app = provider.kind === "github_app";
  return (
    <li className="grid gap-3 p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="font-medium">{provider.name}</p>
          <p className="text-meta text-ink-soft">{[providerKindLabel(provider.kind), provider.base_url, app && !provider.ready && t("Waiting for GitHub to create the App")].filter(Boolean).join(" · ")}</p>
        </div>
        <div className="flex flex-wrap gap-2">
          {app && provider.ready && <button className="btn" type="button" disabled={pending} onClick={() => void run(async () => { await api(`${path}/${provider.id}/sync`, { method: "POST" }); onChanged(); })}>{t("Refresh installations")}</button>}
          {canWrite && <button className="btn btn-danger" type="button" disabled={pending} onClick={() => void run(async () => { await api(`${path}/${provider.id}`, { method: "DELETE" }); onChanged(); })}>{t("Remove")}</button>}
        </div>
      </div>
      {app && provider.ready && (
        <div className="text-meta">
          {provider.installations?.length ? <p>{t("Installed on {accounts}.", { accounts: provider.installations.map(({ account_login }) => account_login).join(", ") })}</p> : <p className="text-ink-soft">{t("Not installed on any account yet.")}</p>}
          {canWrite && provider.install_url && <a className="text-button" href={provider.install_url}>{t("Install on an account")}</a>}
        </div>
      )}
      {provider.public_key && <CopyField label={t("Deploy key (add it to the repository with read access)")} value={provider.public_key} />}
      {error && <p className="text-meta text-ember" role="alert">{error}</p>}
    </li>
  );
}

function AddProvider({ path, onCreated }: { path: string; onCreated: () => void }) {
  const [kind, setKind] = useState<GitProviderKind>("github_app");
  const [name, setName] = useState("");
  const [organization, setOrganization] = useState("");
  const [baseURL, setBaseURL] = useState("");
  const [username, setUsername] = useState("");
  const [token, setToken] = useState("");
  const [saved, setSaved] = useState<string | null>(null);
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    setSaved(null);
    await run(async () => {
      if (kind === "github_app") {
        const started = await api<{ url: string; manifest: string }>(`${path}/github-app`, { method: "POST", body: { name, organization } });
        const form = document.createElement("form");
        form.method = "POST";
        form.action = started.url;
        const input = document.createElement("input");
        input.type = "hidden";
        input.name = "manifest";
        input.value = started.manifest;
        form.append(input);
        document.body.append(form);
        form.submit();
        return;
      }
      await api(path, { method: "POST", body: { kind, name, base_url: baseURL, username, token } });
      setName("");
      setToken("");
      setSaved(t("Provider added."));
      onCreated();
    });
  }

  return (
    <form onSubmit={submit}>
      <div className="grid gap-4 p-4 sm:grid-cols-2">
        <SelectField label={t("Type")} value={kind} onChange={(event) => setKind(event.target.value as GitProviderKind)}>
          {(["github_app", "github_token", "gitlab", "bitbucket", "git_ssh"] as const).map((value) => <option key={value} value={value}>{providerKindLabel(value)}</option>)}
        </SelectField>
        <Field label={kind === "github_app" ? t("GitHub App name") : t("Name")} required maxLength={kind === "github_app" ? 34 : 120} placeholder={kind === "github_app" ? "kymber-my-team" : undefined} hint={kind === "github_app" ? t("Must be unique on GitHub.") : undefined} value={name} onChange={(event) => setName(event.target.value)} />
        {kind === "github_app" && <Field label={t("Organization (optional)")} hint={t("Leave empty to create the App on your personal account.")} maxLength={39} value={organization} onChange={(event) => setOrganization(event.target.value)} />}
        {kind === "gitlab" && <Field label={t("GitLab URL")} type="url" placeholder="https://gitlab.com" maxLength={255} value={baseURL} onChange={(event) => setBaseURL(event.target.value)} />}
        {kind === "git_ssh" && <Field label={t("Git server host")} required placeholder="git.example.com:22" maxLength={255} hint={t("Kymber records the server's SSH host key now and only trusts that key afterwards.")} value={baseURL} onChange={(event) => setBaseURL(event.target.value)} />}
        {kind === "bitbucket" && <Field label={t("Atlassian account email")} type="email" required maxLength={255} value={username} onChange={(event) => setUsername(event.target.value)} />}
        {(kind === "github_token" || kind === "gitlab" || kind === "bitbucket") && <Field label={kind === "bitbucket" ? t("API token") : t("Access token")} type="password" required autoComplete="off" maxLength={512} hint={{ github_token: t("Fine-grained token with Contents: read, Metadata: read and Commit statuses: write."), gitlab: t("Token with the api scope."), bitbucket: t("API token with repository read and pull request write scopes.") }[kind]} value={token} onChange={(event) => setToken(event.target.value)} />}
      </div>
      <FormFooter error={error} saved={saved}>
        <button className="btn btn-primary" type="submit" disabled={pending}>{pending ? t("Saving…") : kind === "github_app" ? t("Create on GitHub") : kind === "git_ssh" ? t("Generate deploy key") : t("Add provider")}</button>
      </FormFooter>
    </form>
  );
}

export function GitProvidersSettings() {
  const { state, can } = useProjects();
  const workspace = state.status === "ready" ? state.workspace : null;
  const canWrite = can("workspace.manage");
  const [providers, setProviders] = useState<GitProvider[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const path = workspace ? `/workspaces/${workspace.id}/git-providers` : "";

  useLoadEffect(
    () => (path ? api<GitProvider[]>(path) : Promise.resolve([])),
    (found) => { setProviders(found); setLoadError(null); },
    (failure) => setLoadError(errorMessage(failure)),
    [path, revision],
  );

  if (state.status === "loading") return <PageSkeleton />;
  if (!workspace) return <EmptyState title={t("No workspace yet")} description={t("Ask an administrator to add you to a workspace.")} />;
  if (loadError) return <LoadError title={t("Could not load Git providers")} message={loadError} onRetry={() => setRevision((value) => value + 1)} />;
  if (!providers) return <PageSkeleton />;
  const reload = () => setRevision((value) => value + 1);

  return (
    <>
      <SettingsSection title={t("Git providers")} description={t("Connections that let apps in this workspace build private repositories and react to pushes.")}>
        {providers.length ? <ul className="divide-y divide-line">{providers.map((provider) => <ProviderRow key={provider.id} provider={provider} path={path} canWrite={canWrite} onChanged={reload} />)}</ul> : <p className="p-4 text-meta text-ink-soft">{t("No Git providers yet. Public GitHub repositories work without one.")}</p>}
      </SettingsSection>
      {canWrite && <SettingsSection title={t("Add a provider")} description={t("A GitHub App is the recommended way to connect GitHub: it gives private repositories, automatic deploys and commit statuses.")}><AddProvider path={path} onCreated={reload} /></SettingsSection>}
    </>
  );
}
