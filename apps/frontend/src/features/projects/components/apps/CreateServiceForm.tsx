import { useState, type FormEvent } from "react";
import { Link, useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import { engineLabel, engines } from "@/features/projects/engines";
import type { DatabaseTemplate, Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function CreateServiceForm({ template, path }: { template: DatabaseTemplate; path: string }) {
  const navigate = useNavigate();
  const { engine } = template;
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [environment, setEnvironment] = useState<Record<string, string>>({});
  const { pending, error, run } = useAction();

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const filled = Object.fromEntries(Object.entries(environment).filter(([, value]) => value));
      const service = await api<Service>(path, { method: "POST", body: { engine, name, image_version: version, environment: filled } });
      navigate(`../apps/${service.id}`);
    });
  }

  return (
    <form className="mx-auto grid w-full max-w-xl gap-5" onSubmit={submit} aria-label={`New ${engineLabel(engine)} app`}>
      <div className="flex items-center gap-3 rounded-panel bg-raised p-4">
        <EngineMark engine={engine} />
        <span className="min-w-0 flex-1">
          <span className="block font-display text-base font-semibold">{engineLabel(engine)}</span>
          <span className="block truncate text-meta text-ink-soft">{engines[engine]?.description ?? template.image}</span>
        </span>
        <Link className="text-button flex-none text-meta" to="?">
          Change
        </Link>
      </div>
      <Field label="App name" autoFocus maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title="Lowercase letters, numbers and hyphens, starting with a letter" hint="Optional. Leave blank to generate one." value={name} onChange={(event) => setName(event.target.value)} />
      <details className="rounded-panel border border-line-strong">
        <summary className="focus-ring cursor-pointer rounded-panel p-3 text-meta font-medium">Advanced settings</summary>
        <div className="grid items-start gap-4 border-t border-line p-3 sm:grid-cols-2">
          <Field label="Image version" maxLength={128} placeholder={template.default_version} value={version} onChange={(event) => setVersion(event.target.value)} />
          {template.fields.map((field) => (
            <Field
              label={field.label}
              type={field.type === "password" ? "password" : "text"}
              autoComplete="off"
              minLength={field.min || undefined}
              maxLength={field.max || undefined}
              pattern={field.pattern?.replace(/^\^|\$$/g, "")}
              placeholder={field.default}
              hint={field.generate ? "Leave blank to generate." : undefined}
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
          Cancel
        </Link>
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Creating…" : "Create app"}
        </button>
      </div>
    </form>
  );
}
