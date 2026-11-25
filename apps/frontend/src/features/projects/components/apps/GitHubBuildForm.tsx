import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";

type GitBuildJob = { id: string; status: string; result_message?: string; repository_url?: string; ref?: string; artifact_expires_at?: string };
type GitSource = { repository_url: string; ref: string; base_directory: string; dockerfile_path: string; build_args: string[]; auto_deploy: boolean };
type GitHubBranch = { name: string; commit: { sha: string } };

function validRepositoryURL(value: string) {
  try {
    const url = new URL(value);
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

  useEffect(() => {
    if (!validRepositoryURL(repositoryURL)) {
      setBranches([]);
      return;
    }
    let active = true;
    setBranchesLoading(true);
    const query = new URLSearchParams({ repository_url: repositoryURL });
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
  }, [repositoryURL, path, branchRequest]);

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
    if (!validRepositoryURL(repositoryURL)) {
      setError(t("Enter a valid public GitHub repository URL"));
      return;
    }
    if (!validGitRef(ref) || !branches.some(({ name }) => name === ref)) {
      setError(t("Select a valid GitHub branch"));
      return;
    }
    setPending(true);
    setError(null);
    try {
      const source = await api<GitSource>(`${path}/${service.id}/git-source`, {
        method: "PUT",
        body: { repository_url: repositoryURL, ref, base_directory: baseDirectory, dockerfile_path: dockerfilePath, build_args: buildArgs.split("\n").map((line) => line.trim()).filter(Boolean), auto_deploy: savedSource?.auto_deploy ?? true },
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
        <Field label={t("Public GitHub repository URL")} type="url" required maxLength={512} placeholder="https://github.com/owner/repository" value={repositoryURL} onChange={(event) => { setRepositoryURL(event.target.value); setBranches([]); setJob(null); setError(null); }} />
        <label className="block"><span className="mb-1 block text-meta font-medium">{t("Branch")}</span><select className="field" required disabled={branchesLoading || branches.length === 0} value={ref} onChange={(event) => setRef(event.target.value)}><option value="">{branchesLoading ? t("Loading branches…") : t("Select a branch")}</option>{branches.map(({ name }) => <option key={name} value={name}>{name}</option>)}</select></label>
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
          <button className="btn btn-primary" type="submit" disabled={pending || !ref || !branches.some(({ name }) => name === ref) || job?.status === "queued" || job?.status === "running"}>{pending ? t("Saving…") : service.status === "draft" ? t("Continue") : t("Save and rebuild")}</button>
        </form>
      </>}
    </div>
  );
}
