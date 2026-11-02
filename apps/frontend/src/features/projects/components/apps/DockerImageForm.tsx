import { Plus, Search, Star, X } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import { appCategories, type AppCategory } from "@/features/projects/engines";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";

type HubImage = { repository: string; description: string; official: boolean; stars: number; logo_url?: string; category: AppCategory };
type ImageDetails = { tags: string[]; default_version: string; port: number; data_path: string };
type Variable = { id: number; name: string; value: string };

function ImageSearch({ onPick }: { onPick: (image: HubImage) => void }) {
  const [text, setText] = useState("");
  const [query, setQuery] = useState("");
  const [images, setImages] = useState<HubImage[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [params] = useSearchParams();
  const category = appCategories.find(({ id }) => id === params.get("category"));
  const shown = category ? images?.filter((image) => image.category === category.id) : images;

  useEffect(() => {
    const timer = setTimeout(() => setQuery(text.trim()), 300);
    return () => clearTimeout(timer);
  }, [text]);

  useLoadEffect(
    () => (query ? api<{ images: HubImage[] }>(`/docker-hub/search?q=${encodeURIComponent(query)}`) : Promise.resolve(null)),
    (response) => {
      setImages(response?.images ?? null);
      setError(null);
    },
    setError,
    [query],
  );

  return (
    <div className="mx-auto grid w-full max-w-xl gap-4">
      <label className="block">
        <span className="mb-1 block text-meta font-medium">Search Docker Hub{category ? ` for ${category.label.toLowerCase()}` : ""}</span>
        <span className="relative block">
          <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={16} />
          <input className="field pl-9" type="search" autoFocus maxLength={100} placeholder="postgres, nginx, bitnami/redis…" value={text} onChange={(event) => setText(event.target.value)} />
        </span>
      </label>
      <FormError message={error} />
      {shown?.length === 0 && (
        <p className="text-meta text-ink-soft">
          No {category?.label.toLowerCase() ?? "images"} match “{query}”.{" "}
          {category && (
            <Link className="text-button" to={params.get("environment") ? `?template=docker&environment=${params.get("environment")}` : "?template=docker"}>
              Search all categories
            </Link>
          )}
        </p>
      )}
      {shown && shown.length > 0 && (
        <ul className="grid gap-2">
          {shown.map((image) => (
            <li className="min-w-0" key={image.repository}>
              <button className="row-link flex w-full items-center gap-3 rounded-panel bg-raised p-3 text-left" type="button" onClick={() => onPick(image)}>
                <EngineMark engine="docker" logo={image.logo_url} />
                <span className="min-w-0 flex-1">
                  <span className="flex flex-wrap items-center gap-2 font-medium">
                    <span className="min-w-0 break-all">{image.repository}</span>
                    {image.official && <span className="rounded-sm bg-seafoam/15 px-1.5 text-xs text-seafoam">Official</span>}
                  </span>
                  {image.description && <span className="mt-0.5 block truncate text-meta text-ink-soft">{image.description}</span>}
                </span>
                <span className="flex flex-none items-center gap-1 text-xs text-ink-soft">
                  <Star aria-hidden="true" size={12} />
                  <span className="sr-only">Stars:</span>
                  {image.stars}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function ImageSettings({ image, path, onChange }: { image: HubImage; path: string; onChange: () => void }) {
  const { repository, logo_url, category } = image;
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [environmentId, setEnvironmentId] = useState(params.get("environment") ?? "");
  const [details, setDetails] = useState<ImageDetails | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [port, setPort] = useState("");
  const [dataPath, setDataPath] = useState("");
  const [variables, setVariables] = useState<Variable[]>([]);
  const { pending, error, run } = useAction();

  useLoadEffect(
    () => api<ImageDetails>(`/docker-hub/image?repository=${encodeURIComponent(repository)}`),
    (response) => {
      setDetails(response);
      setVersion(response.default_version);
      setPort(response.port ? String(response.port) : "");
      setDataPath(response.data_path);
    },
    () => {
      setVersion((current) => current || "latest");
      setLoadError("Could not read this image's details from Docker Hub. Check the version, and fill in the port and data path by hand.");
    },
    [repository],
  );

  function updateVariable(id: number, change: Partial<Variable>) {
    setVariables(variables.map((variable) => (variable.id === id ? { ...variable, ...change } : variable)));
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const environment = Object.fromEntries(variables.filter(({ name }) => name).map(({ name, value }) => [name, value]));
      const service = await api<Service>(path, { method: "POST", body: { engine: "docker", repository, name, image_version: version, port: Number(port), data_path: dataPath, logo_url, category, environment, environment_id: environmentId } });
      navigate(`../apps/${service.id}`);
    });
  }

  return (
    <form className="mx-auto grid w-full max-w-xl gap-5" onSubmit={submit} aria-label={`New app from ${repository}`}>
      <div className="flex items-center gap-3 rounded-panel bg-raised p-4">
        <EngineMark engine="docker" logo={logo_url} />
        <span className="min-w-0 flex-1">
          <span className="block break-all font-display text-base font-semibold">{repository}</span>
          <span className="block truncate text-meta text-ink-soft">Docker Hub image</span>
        </span>
        <button className="text-button flex-none text-meta" type="button" onClick={onChange}>
          Change
        </button>
      </div>
      <FormError message={loadError} />
      <Field label="App name" autoFocus maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title="Lowercase letters, numbers and hyphens, starting with a letter" hint="Optional. Leave blank to generate one." value={name} onChange={(event) => setName(event.target.value)} />
      <EnvironmentField value={environmentId} onChange={setEnvironmentId} />
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <Field label="Image version" list="docker-image-tags" maxLength={128} required placeholder={details || loadError ? undefined : "Loading…"} value={version} onChange={(event) => setVersion(event.target.value)} />
        <datalist id="docker-image-tags">
          {details?.tags.map((tag) => <option value={tag} key={tag} />)}
        </datalist>
        <Field label="Port" type="number" min={1} max={65535} hint="Optional. Used on the Connection page." value={port} onChange={(event) => setPort(event.target.value)} />
        <div className="sm:col-span-2">
          <Field label="Data path" maxLength={255} pattern="/.+" title="An absolute path inside the container" placeholder="/data" hint="Kept in a volume across deploys. Leave blank only if this image stores nothing: without a path, its data is lost on every deploy." value={dataPath} onChange={(event) => setDataPath(event.target.value)} />
        </div>
      </div>
      <fieldset className="grid gap-3">
        <legend className="mb-1 text-meta font-medium">Environment variables</legend>
        <p className="text-xs text-ink-soft">Docker Hub does not list the variables an image needs. Check the image's documentation.</p>
        {variables.map(({ id, name, value }) => (
          <div className="grid grid-cols-[1fr_auto] items-center gap-2 sm:grid-cols-[1fr_1fr_auto]" key={id}>
            <input className="field font-mono" aria-label="Variable name" autoComplete="off" pattern="[A-Za-z_][A-Za-z0-9_]*" title="Letters, numbers and underscores, not starting with a number" required placeholder="NAME" value={name} onChange={(event) => updateVariable(id, { name: event.target.value })} />
            <button className="btn order-none px-2 sm:order-last" type="button" aria-label={`Remove ${name || "variable"}`} onClick={() => setVariables(variables.filter((variable) => variable.id !== id))}>
              <X aria-hidden="true" size={16} />
            </button>
            <input className="field col-span-2 sm:col-span-1" type="password" aria-label={`Value of ${name || "variable"}`} autoComplete="off" placeholder="Value" value={value} onChange={(event) => updateVariable(id, { value: event.target.value })} />
          </div>
        ))}
        <button className="btn w-fit" type="button" onClick={() => setVariables([...variables, { id: Date.now(), name: "", value: "" }])}>
          <Plus aria-hidden="true" size={16} />
          Add variable
        </button>
      </fieldset>
      <FormError message={error} />
      <div className="flex flex-wrap justify-end gap-2">
        <Link className="btn" to="../apps">
          Cancel
        </Link>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Creating…" : "Create app"}
        </button>
      </div>
    </form>
  );
}

export function DockerImageForm({ path }: { path: string }) {
  const [image, setImage] = useState<HubImage | null>(null);
  return image ? <ImageSettings image={image} path={path} onChange={() => setImage(null)} key={image.repository} /> : <ImageSearch onPick={setImage} />;
}
