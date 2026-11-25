import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { SelectField } from "@/components/ui/SelectField";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import { type GitProvider, providerHost, providerKindLabel } from "@/features/projects/gitProviders";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";

type GitBuildJob = { id: string; status: string; result_message?: string; repository_url?: string; ref?: string; artifact_expires_at?: string };
type GitSource = { provider_id: string; installation_id: number; repository_url: string; ref: string; base_directory: string; dockerfile_path: string; build_args: string[]; auto_deploy: boolean };
type GitHubBranch = { name: string; commit: { sha: string } };

function validRepositoryURL(value: string, provider?: GitProvider) {
  try {
    const url = new URL(value);
    if (provider) return url.host === providerHost(provider) && (provider.kind === "git_ssh" ? url.protocol === "ssh:" : url.protocol === "https:");
    return url.protocol === "https:" && url.hostname === "github.com" && !url.port && !url.username && !url.password && !url.search && !url.hash && /^\/[A-Za-z0-9-]{1,39}\/[A-Za-z0-9._-]{1,100}(?:\.git)?\/?$/.test(url.pathname);
  } catch {
    return false;
  }
}

function validGitRef(value: string) {
  return /^[A-Za-z0-9][A-Za-z0-9._/-]{0,127}$/.test(value) && !value.includes("..") && !value.includes("//") && !value.includes("@{") && value.split("/").every((part) => part && !part.startsWith(".") && !part.endsWith(".") && !part.endsWith(".lock"));
}

export function GitHubBuildForm({ path, gitBuildsEnabled, service: existingService }: { path: string; gitBuildsEnabled: boolean; service?: Service }) {
  const projectId = path.split("/")[4];
  const workspaceId = path.split("/")[2];
  const [providers, setProviders] = useState<GitProvider[]>([]);
  const [providerId, setProviderId] = useState("");
  const [installationId, setInstallationId] = useState(0);
  const [repositories, setRepositories] = useState<{ name: string; url: string }[]>([]);
  const [repositoryURL, setRepositoryURL] = useState("");
  const [ref, setRef] = useState("");
  const [baseDirectory, setBaseDirectory] = useState("");
  const [dockerfilePath, setDockerfilePath] = useState("Dockerfile");
  const [buildArgs, setBuildArgs] = useState("");
  const [savedSource, setSavedSource] = useState<GitSource | null>(null);
  const [branches, setBranches] = useState<GitHubBranch[]>([]);
  const [branchesLoading, setBranchesLoading] = useState(false);
  const [branchRequest, setBranchRequest] = useState(0);
  const [branchesError, setBranchesError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [environmentId, setEnvironmentId] = useState("");
  const [createdService, setCreatedService] = useState<Service | null>(null);
  const [job, setJob] = useState<GitBuildJob | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const service = existingService ?? createdService;
  const provider = providers.find(({ id }) => id === providerId);
  const manualRef = provider?.kind === "git_ssh";

  useEffect(() => {
    let active = true;
    api<GitProvider[]>(`/workspaces/${workspaceId}/git-providers`)
      .then((found) => { if (active) setProviders(found.filter(({ ready }) => ready)); })
      .catch(() => { if (active) setProviders([]); });
    return () => { active = false; };
  }, [workspaceId]);

  useEffect(() => {
    setRepositories([]);
    if (!provider || provider.kind === "git_ssh" || (provider.kind === "github_app" && !installationId)) return;
    let active = true;
    const query = provider.kind === "github_app" ? `?installation_id=${installationId}` : "";
    api<{ name: string; url: string }[]>(`/workspaces/${workspaceId}/git-providers/${provider.id}/repositories${query}`)
      .then((found) => { if (active) setRepositories(found); })
      .catch((cause) => { if (active) setBranchesError(cause instanceof Error ? cause.message : t("Could not load repositories")); });
    return () => { active = false; };
  }, [provider?.id, installationId, workspaceId]);

  useEffect(() => {
    if (manualRef || !validRepositoryURL(repositoryURL, provider) || (provider?.kind === "github_app" && !installationId)) {
      setBranches([]);
      return;
    }
    let active = true;
    setBranchesLoading(true);
    const query = new URLSearchParams({ repository_url: repositoryURL, provider_id: providerId, installation_id: String(installationId) });
    api<{ branches: GitHubBranch[] }>(`${path.replace(/\/services$/, "")}/git-builds/branches?${query}`)
      .then(({ branches: found }) => {
        if (!active) return;
        setBranchesError(null);
        setBranches(found);
        setRef((current) => found.some(({ name }) => name === current) ? current : found[0]?.name ?? "");
      })
      .catch((cause) => {
        if (active) {
          setBranches([]);
          setBranchesError(cause instanceof Error ? cause.message : t("Could not load GitHub branches"));
        }
      })
      .finally(() => { if (active) setBranchesLoading(false); });
    return () => { active = false; };
  }, [repositoryURL, path, branchRequest, providerId, installationId, manualRef]);

  useEffect(() => {
    if (existingService) setCreatedService(existingService);
  }, [existingService]);

  useEffect(() => {
    if (!existingService) return;
    let active = true;
    api<GitSource | null>(`${path}/${existingService.id}/git-source`)
      .then((source) => {
        if (!active || !source) return;
        setSavedSource(source);
        setProviderId(source.provider_id);
        setInstallationId(source.installation_id);
        setRepositoryURL(source.repository_url);
        setRef(source.ref);
        setBaseDirectory(source.base_directory);
        setDockerfilePath(source.dockerfile_path);
        setBuildArgs(source.build_args.join("\n"));
      })
      .catch(() => { if (active) setError(t("Could not load the Git source")); });
    return () => { active = false; };
  }, [existingService?.id, path]);

  useEffect(() => {
    if (!service || !job || job.status === "succeeded" || job.status === "failed") return;
    let active = true;
    let timer: number;
    const poll = async () => {
      try {
        const jobs = await api<GitBuildJob[]>(`${path}/${service.id}/git-builds`);
        if (!active) return;
        const current = jobs.find(({ id }) => id === job.id);
        if (current) {
          setJob(current);
          if (current.status === "succeeded" || current.status === "failed") return;
        }
      } catch {
        if (active) setError(t("Could not check GitHub build status"));
      }
      if (active) timer = window.setTimeout(poll, 3000);
    };
    timer = window.setTimeout(poll, 1500);
    return () => {
      active = false;
      window.clearTimeout(timer);
    };
  }, [job?.id, job?.status, path, service]);

  async function createDraft(event: FormEvent) {
    event.preventDefault();
    if (!gitBuildsEnabled || pending) return;
    setPending(true);
    setError(null);
    try {
      const target = await api<Service>(`${path.replace(/\/services$/, "")}/git-build-drafts`, {
        method: "POST",
        body: { name, environment_id: environmentId },
      });
      setCreatedService(target);
      navigate(`/projects/${projectId}/apps/${target.id}/source`);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t("Could not create app draft"));
    } finally {
      setPending(false);
    }
  }

  async function saveSource(event: FormEvent, build: boolean) {
    event.preventDefault();
    if (!gitBuildsEnabled || pending || !service) return;
    if (!validRepositoryURL(repositoryURL, provider)) {
      setError(provider ? t("Enter a repository URL on {host}", { host: providerHost(provider) }) : t("Enter a valid public GitHub repository URL"));
      return;
    }
    if (!validGitRef(ref) || (!manualRef && !branches.some(({ name }) => name === ref))) {
      setError(t("Select a valid GitHub branch"));
      return;
    }
    setPending(true);
    setError(null);
    try {
      const source = await api<GitSource>(`${path}/${service.id}/git-source`, {
        method: "PUT",
        body: { provider_id: providerId, installation_id: installationId, repository_url: repositoryURL, ref, base_directory: baseDirectory, dockerfile_path: dockerfilePath, build_args: buildArgs.split("\n").map((line) => line.trim()).filter(Boolean), auto_deploy: savedSource?.auto_deploy ?? true },
      });
      setSavedSource(source);
      if (service.status === "draft") {
        navigate(`/projects/${projectId}/apps/${service.id}?build=1`);
      } else if (build) {
        const queued = await api<{ id: string; status: string }>(`${path}/${service.id}/git-builds`, { method: "POST" });
        setJob(queued);
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t("Could not save the Git source"));
    } finally {
      setPending(false);
    }
  }

  if (!gitBuildsEnabled) return <p className="mx-auto w-full max-w-xl rounded-panel bg-raised p-4 text-meta text-ink-soft">{t("GitHub builds are disabled in this environment")}</p>;
  return (
    <div className="mx-auto grid w-full max-w-xl gap-5">
      <div className="rounded-panel bg-raised p-4 text-meta text-ink-soft">{service ? t("Select a branch and start its Dockerfile build. Configure environment variables and port on the app draft.") : t("Create the app draft first, then configure its GitHub repository and branch.")}</div>
      {!service ? <>
        <form className="grid gap-5" onSubmit={createDraft} aria-label={t("Create draft")}>
          <Field label={t("App name")} required maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} value={name} onChange={(event) => setName(event.target.value)} />
          <EnvironmentField value={environmentId} onChange={setEnvironmentId} />
          <FormError message={error} />
          <div className="flex justify-end gap-2"><Link className="btn" to="../apps">{t("Cancel")}</Link><button className="btn btn-primary" type="submit" disabled={pending}>{pending ? t("Creating…") : t("Create draft")}</button></div>
        </form>
      </> : <>
        <div className="grid gap-4 sm:grid-cols-2">
          <SelectField label={t("Source")} value={providerId} onChange={(event) => { setProviderId(event.target.value); setInstallationId(0); setRepositoryURL(""); setBranches([]); setBranchesError(null); setJob(null); }}>
            <option value="">{t("Public GitHub repository")}</option>
            {providers.map((item) => <option key={item.id} value={item.id}>{item.name} · {providerKindLabel(item.kind)}</option>)}
          </SelectField>
          {provider?.kind === "github_app" && <SelectField label={t("Installation")} required value={installationId || ""} onChange={(event) => { setInstallationId(Number(event.target.value)); setRepositoryURL(""); }}>
            <option value="">{t("Select an account")}</option>
            {provider.installations?.map((installation) => <option key={installation.id} value={installation.id}>{installation.account_login}</option>)}
          </SelectField>}
        </div>
        <Field label={provider ? t("Repository URL") : t("Public GitHub repository URL")} type={manualRef ? "text" : "url"} required maxLength={512} list={repositories.length ? "git-repositories" : undefined} placeholder={manualRef ? `ssh://git@${providerHost(provider)}/team/repository.git` : `https://${providerHost(provider)}/owner/repository`} hint={repositories.length ? t("Pick one of your repositories or paste its URL.") : undefined} value={repositoryURL} onChange={(event) => { setRepositoryURL(event.target.value); setBranches([]); setJob(null); setError(null); }} />
        {repositories.length > 0 && <datalist id="git-repositories">{repositories.map((repository) => <option key={repository.url} value={repository.url}>{repository.name}</option>)}</datalist>}
        {manualRef ? <Field label={t("Branch")} required maxLength={128} placeholder="main" value={ref} onChange={(event) => setRef(event.target.value)} /> : <label className="block"><span className="mb-1 block text-meta font-medium">{t("Branch")}</span><select className="field" required disabled={branchesLoading || branches.length === 0} value={ref} onChange={(event) => setRef(event.target.value)}><option value="">{branchesLoading ? t("Loading branches…") : t("Select a branch")}</option>{branches.map(({ name }) => <option key={name} value={name}>{name}</option>)}</select></label>}
        {branchesError && <div><FormError message={branchesError} /><button className="text-button text-meta" type="button" disabled={branchesLoading} onClick={() => setBranchRequest((value) => value + 1)}>{t("Retry loading branches")}</button></div>}
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("Base directory")} hint={t("Folder used as the build context. Leave empty for the repository root.")} maxLength={255} placeholder="apps/web" value={baseDirectory} onChange={(event) => setBaseDirectory(event.target.value)} />
          <Field label={t("Dockerfile path")} hint={t("Relative to the base directory.")} required maxLength={255} value={dockerfilePath} onChange={(event) => setDockerfilePath(event.target.value)} />
        </div>
        <label className="block"><span className="mb-1 block text-meta font-medium">{t("Build arguments")}</span><textarea className="field min-h-24 font-mono text-xs" spellCheck={false} placeholder="NODE_ENV=production" value={buildArgs} onChange={(event) => setBuildArgs(event.target.value)} /><span className="mt-1 block text-xs text-ink-soft">{t("One KEY=VALUE per line, passed as --build-arg. They are kept in the image history, so use environment variables for secrets.")}</span></label>
        <FormError message={error} />
        {job && (job.status === "running" || job.status === "queued") && <p className="text-meta text-ink-soft" aria-live="polite">{t("Building from GitHub…")}</p>}
        {job?.status === "succeeded" && <p className="text-meta text-ink-soft" aria-live="polite">{t("Build finished. The app is being redeployed.")}</p>}
        {job?.status === "failed" && <p className="text-meta text-ember" role="alert">{job.result_message || t("Could not queue GitHub build")}</p>}
        <form onSubmit={(event) => void saveSource(event, true)} className="flex flex-wrap justify-end gap-2">
          {service.status !== "draft" && <button className="btn" type="button" disabled={pending || !ref} onClick={(event) => void saveSource(event, false)}>{t("Save")}</button>}
          <button className="btn btn-primary" type="submit" disabled={pending || !ref || (!manualRef && !branches.some(({ name }) => name === ref)) || job?.status === "queued" || job?.status === "running"}>{pending ? t("Saving…") : service.status === "draft" ? t("Continue") : t("Save and rebuild")}</button>
        </form>
      </>}
    </div>
  );
}
