import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";

type GitBuildJob = { id: string; status: string; result_message?: string; repository_url?: string; ref?: string; artifact_expires_at?: string };

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

export function GitHubBuildForm({ path, gitBuildsEnabled }: { path: string; gitBuildsEnabled: boolean }) {
  const [repositoryURL, setRepositoryURL] = useState("");
  const [ref, setRef] = useState("main");
  const [name, setName] = useState("");
  const [port, setPort] = useState(8080);
  const [service, setService] = useState<Service | null>(null);
  const [job, setJob] = useState<GitBuildJob | null>(null);
  const [pending, setPending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();

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

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (!gitBuildsEnabled || pending || !validRepositoryURL(repositoryURL) || !validGitRef(ref)) {
      if (!validRepositoryURL(repositoryURL)) setError(t("Enter a valid public GitHub repository URL"));
      else if (!validGitRef(ref)) setError(t("Git ref must not contain spaces or invalid characters"));
      return;
    }
    setPending(true);
    setError(null);
    try {
      let target = service;
      if (!target) {
        target = await api<Service>(path, {
          method: "POST",
          body: { engine: "docker", name, repository: "library/alpine", port, data_path: "", category: "application" },
        });
        setService(target);
      }
      const queued = await api<{ id: string; status: string }>(`${path}/${target.id}/git-builds`, {
        method: "POST",
        body: { repository_url: repositoryURL, ref },
      });
      setJob({ ...queued, repository_url: repositoryURL, ref });
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t("Could not queue GitHub build"));
    } finally {
      setPending(false);
    }
  }

  if (!gitBuildsEnabled) return <p className="mx-auto w-full max-w-xl rounded-panel bg-raised p-4 text-meta text-ink-soft">{t("GitHub builds are disabled in this environment")}</p>;
  if (job && job.status === "succeeded") {
    return (
      <div className="mx-auto grid w-full max-w-xl gap-4 rounded-panel bg-raised p-5">
        <h2 className="section-title">{t("Build completed")}</h2>
        <p className="text-meta text-ink-soft">{repositoryURL} · {ref}</p>
        <button className="btn btn-primary justify-self-end" type="button" onClick={() => navigate(`../apps/${service?.id}`)}>{t("Open app to deploy")}</button>
        <FormError message={error} />
      </div>
    );
  }

  return (
    <form className="mx-auto grid w-full max-w-xl gap-5" onSubmit={submit} aria-label={t("Deploy from public GitHub")}>
      <div className="rounded-panel bg-raised p-4 text-meta text-ink-soft">{t("Build an app from a public GitHub repository. After the build succeeds, choose Deploy to select a server and start it.")}</div>
      <Field label={t("Public GitHub repository URL")} type="url" required maxLength={512} placeholder="https://github.com/owner/repository" value={repositoryURL} onChange={(event) => setRepositoryURL(event.target.value)} />
      <Field label={t("Branch, tag, or commit ref")} required maxLength={128} value={ref} onChange={(event) => setRef(event.target.value)} />
      {!service && <Field label={t("App name")} required maxLength={63} pattern="[a-z][a-z0-9-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} value={name} onChange={(event) => setName(event.target.value)} />}
      <Field label={t("Application port")} type="number" min={0} max={65535} required value={port} onChange={(event) => setPort(Number(event.target.value))} hint={t("Set the container port to publish; use 0 for no published port.")} />
      <FormError message={error} />
      {job && (job.status === "running" || job.status === "queued") ? <p className="text-meta text-ink-soft" aria-live="polite">{t("Building from GitHub…")}</p> : null}
      {job?.status === "failed" && <p className="text-meta text-ember" role="alert">{job.result_message || t("Could not queue GitHub build")}</p>}
      <div className="flex justify-end gap-2">
        <Link className="btn" to="../apps">{t("Cancel")}</Link>
        <button className="btn btn-primary" type="submit" disabled={pending || port < 0 || port > 65535 || (job?.status === "queued" || job?.status === "running")}>
          {pending ? t("Queueing build…") : job?.status === "failed" ? t("Retry build") : t("Create app and build")}
        </button>
      </div>
    </form>
  );
}
