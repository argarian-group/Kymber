import { useState, type FormEvent } from "react";
import { Link, useNavigate, useSearchParams } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { EnvironmentField } from "@/features/projects/components/apps/EnvironmentField";
import { engineLabel, engines } from "@/features/projects/engines";
import type { DatabaseTemplate, Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t, translate } from "@/lib/i18n";

export function CreateServiceForm({ template, path }: { template: DatabaseTemplate; path: string }) {
  const navigate = useNavigate();
  const { engine } = template;
  const [params] = useSearchParams();
  const [environmentId, setEnvironmentId] = useState(params.get("environment") ?? "");
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [environment, setEnvironment] = useState<Record<string, string>>({});
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const filled = Object.fromEntries(Object.entries(environment).filter(([, value]) => value));
      const service = await api<Service>(path, { method: "POST", body: { engine, name, image_version: version, environment: filled, environment_id: environmentId } });
      navigate(`../apps/${service.id}`);
    });
  }

  return (
    <form className="mx-auto grid w-full max-w-xl gap-5" onSubmit={submit} aria-label={t("New {engine} app", { engine: engineLabel(engine) })}>
      <div className="flex items-center gap-3 rounded-panel bg-raised p-4">
        <EngineMark engine={engine} image={template.image} />
        <span className="min-w-0 flex-1">
          <span className="block font-display text-base font-semibold">{engineLabel(engine)}</span>
          <span className="block truncate text-meta text-ink-soft">{engines[engine]?.description ?? template.image}</span>
        </span>
        <Link className="text-button flex-none text-meta" to="?">
          {t("Change")}
        </Link>
      </div>
      <Field label={t("App name")} autoFocus maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} hint={t("Optional. Leave blank to generate one.")} value={name} onChange={(event) => setName(event.target.value)} />
      <EnvironmentField value={environmentId} onChange={setEnvironmentId} />
      <details className="rounded-panel border border-line-strong">
        <summary className="focus-ring cursor-pointer rounded-panel p-3 text-meta font-medium">{t("Advanced settings")}</summary>
        <div className="grid items-start gap-4 border-t border-line p-3 sm:grid-cols-2">
          <Field label={t("Image version")} maxLength={128} placeholder={template.default_version} value={version} onChange={(event) => setVersion(event.target.value)} />
          {template.fields.map((field) => (
            <Field
              label={translate(field.label)}
              type={field.type === "password" ? "password" : "text"}
              autoComplete="off"
              minLength={field.min || undefined}
              maxLength={field.max || undefined}
              pattern={field.pattern?.replace(/^\^|\$$/g, "")}
              placeholder={field.default}
              hint={field.generate ? t("Leave blank to generate.") : undefined}
              value={environment[field.name] ?? ""}
              onChange={(event) => setEnvironment({ ...environment, [field.name]: event.target.value })}
              key={field.name}
            />
          ))}
        </div>
      </details>
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
