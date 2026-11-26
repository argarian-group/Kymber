import { Search, Star } from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import { VariableRows, type Variable } from "@/features/projects/components/apps/VariableRows";
import { appCategories, type AppCategory } from "@/features/projects/engines";
import type { Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { useLoadEffect } from "@/lib/useLoadEffect";
import { t } from "@/lib/i18n";

type HubImage = { repository: string; description: string; official: boolean; stars: number; logo_url?: string; category: AppCategory };
type ImageDetails = { tags: string[]; default_version: string; port: number; data_path: string };

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
        <span className="mb-1 block text-meta font-medium">{category ? t("Search Docker Hub for {category}", { category: category.label.toLowerCase() }) : t("Search Docker Hub")}</span>
        <span className="relative block">
          <Search className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={16} />
          <input className="field pl-9" type="search" autoFocus maxLength={100} placeholder="postgres, nginx, bitnami/redis…" value={text} onChange={(event) => setText(event.target.value)} />
        </span>
      </label>
      <FormError message={error} />
      {shown?.length === 0 && (
        <p className="text-meta text-ink-soft">
          {t("No {items} match “{query}”.", { items: category?.label.toLowerCase() ?? t("images"), query })}{" "}
          {category && (
            <Link className="text-button" to={params.get("environment") ? `?template=docker&environment=${params.get("environment")}` : "?template=docker"}>
              {t("Search all categories")}
            </Link>
          )}
        </p>
      )}
      {shown && shown.length > 0 && (
        <ul className="grid gap-2">
          {shown.map((image) => (
            <li className="min-w-0" key={image.repository}>
              <button className="row-link flex w-full items-center gap-3 rounded-panel bg-raised p-3 text-left" type="button" onClick={() => onPick(image)}>
                <EngineMark engine="docker" logo={image.logo_url} image={image.repository} />
                <span className="min-w-0 flex-1">
                  <span className="flex flex-wrap items-center gap-2 font-medium">
                    <span className="min-w-0 break-all">{image.repository}</span>
                    {image.official && <span className="rounded-sm bg-seafoam/15 px-1.5 text-xs text-seafoam">{t("Official")}</span>}
                  </span>
                  {image.description && <span className="mt-0.5 block truncate text-meta text-ink-soft">{image.description}</span>}
                </span>
                <span className="flex flex-none items-center gap-1 text-xs text-ink-soft">
                  <Star aria-hidden="true" size={12} />
                  <span className="sr-only">{t("Stars:")}</span>
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
      setLoadError(t("Could not read this image's details from Docker Hub. Check the version, and fill in the port and data path by hand."));
    },
    [repository],
  );


  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const environment = Object.fromEntries(variables.filter(({ name }) => name).map(({ name, value }) => [name, value]));
      const service = await api<Service>(path, { method: "POST", body: { engine: "docker", repository, name, image_version: version, port: Number(port), data_path: dataPath, logo_url, category, environment, environment_id: environmentId } });
      navigate(`../apps/${service.id}`);
    });
  }

  return (
    <form className="mx-auto grid w-full max-w-xl gap-5" onSubmit={submit} aria-label={t("New app from {repository}", { repository })}>
      <div className="flex items-center gap-3 rounded-panel bg-raised p-4">
        <EngineMark engine="docker" logo={logo_url} image={repository} />
        <span className="min-w-0 flex-1">
          <span className="block break-all font-display text-base font-semibold">{repository}</span>
          <span className="block truncate text-meta text-ink-soft">{t("Docker Hub image")}</span>
        </span>
        <button className="text-button flex-none text-meta" type="button" onClick={onChange}>
          {t("Change")}
        </button>
      </div>
      <FormError message={loadError} />
      <Field label={t("App name")} autoFocus maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} hint={t("Optional. Leave blank to generate one.")} value={name} onChange={(event) => setName(event.target.value)} />
      <EnvironmentField value={environmentId} onChange={setEnvironmentId} />
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <Field label={t("Image version")} list="docker-image-tags" maxLength={128} required placeholder={details || loadError ? undefined : t("Loading…")} value={version} onChange={(event) => setVersion(event.target.value)} />
        <datalist id="docker-image-tags">
          {details?.tags.map((tag) => <option value={tag} key={tag} />)}
        </datalist>
        <Field label={t("Port")} type="number" min={1} max={65535} hint={t("Optional. Used on the Connection page.")} value={port} onChange={(event) => setPort(event.target.value)} />
        <div className="sm:col-span-2">
          <Field label={t("Data path")} maxLength={255} pattern="/.+" title={t("An absolute path inside the container")} placeholder="/data" hint={t("Kept in a volume across deploys. Leave blank only if this image stores nothing: without a path, its data is lost on every deploy.")} value={dataPath} onChange={(event) => setDataPath(event.target.value)} />
        </div>
      </div>
      <fieldset className="grid gap-3">
        <legend className="mb-1 text-meta font-medium">{t("Environment variables")}</legend>
        <p className="text-xs text-ink-soft">{t("Docker Hub does not list the variables an image needs. Check the image's documentation.")}</p>
        <VariableRows variables={variables} onChange={setVariables} />
      </fieldset>
      <FormError message={error} />
      <div className="flex flex-wrap justify-end gap-2">
        <Link className="btn" to="../apps">
          {t("Cancel")}
        </Link>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? t("Creating…") : t("Create app")}
        </button>
      </div>
    </form>
  );
}

export function DockerImageForm({ path }: { path: string }) {
  const [image, setImage] = useState<HubImage | null>(null);
  return image ? <ImageSettings image={image} path={path} onChange={() => setImage(null)} key={image.repository} /> : <ImageSearch onPick={setImage} />;
}
