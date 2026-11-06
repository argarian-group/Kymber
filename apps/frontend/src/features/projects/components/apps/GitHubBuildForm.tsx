import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";

type GitBuildJob = { id: string; status: string; result_message?: string; repository_url?: string; ref?: string; artifact_expires_at?: string };
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

  async function startBuild(event: FormEvent) {
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
      const queued = await api<{ id: string; status: string }>(`${path}/${service.id}/git-builds`, {
        method: "POST",
        body: { repository_url: repositoryURL, ref, build_pack: "dockerfile" },
      });
      setJob({ ...queued, repository_url: repositoryURL, ref });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t("Could not queue GitHub build"));
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
          <Field label={t("App name")} required maxLength={63} pattern="[a-z][a-z0-9-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} value={name} onChange={(event) => setName(event.target.value)} />
          <EnvironmentField value={environmentId} onChange={setEnvironmentId} />
          <FormError message={error} />
          <div className="flex justify-end gap-2"><Link className="btn" to="../apps">{t("Cancel")}</Link><button className="btn btn-primary" type="submit" disabled={pending}>{pending ? t("Creating…") : t("Create draft")}</button></div>
        </form>
      </> : <>
        <Field label={t("Public GitHub repository URL")} type="url" required maxLength={512} placeholder="https://github.com/owner/repository" value={repositoryURL} onChange={(event) => { setRepositoryURL(event.target.value); setBranches([]); setJob(null); setError(null); }} />
        <label className="block"><span className="mb-1 block text-meta font-medium">{t("Branch")}</span><select className="field" required disabled={branchesLoading || branches.length === 0} value={ref} onChange={(event) => setRef(event.target.value)}><option value="">{branchesLoading ? t("Loading branches…") : t("Select a branch")}</option>{branches.map(({ name }) => <option key={name} value={name}>{name}</option>)}</select></label>
        {branchesError && <div><FormError message={branchesError} /><button className="text-button text-meta" type="button" disabled={branchesLoading} onClick={() => setBranchRequest((value) => value + 1)}>{t("Retry loading branches")}</button></div>}
        <FormError message={error} />
        {job && (job.status === "running" || job.status === "queued") && <p className="text-meta text-ink-soft" aria-live="polite">{t("Building from GitHub…")}</p>}
        {job?.status === "failed" && <p className="text-meta text-ember" role="alert">{job.result_message || t("Could not queue GitHub build")}</p>}
        <form onSubmit={startBuild} className="flex justify-end gap-2"><button className="btn btn-primary" type="submit" disabled={pending || !ref || !branches.some(({ name }) => name === ref) || job?.status === "queued" || job?.status === "running"}>{pending ? t("Queueing build…") : job?.status === "failed" ? t("Retry build") : t("Deploy")}</button></form>
      </>}
    </div>
  );
}
