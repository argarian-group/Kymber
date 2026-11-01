import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { EngineMark } from "@/features/projects/components/apps/EngineMark";
import type { DatabaseTemplate, Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

const engines: Record<string, { label: string; description: string }> = {
  mysql: { label: "MySQL", description: "Relational database" },
  redis: { label: "Redis", description: "In-memory data store" },
};

export function CreateServiceForm({ templates, path, onClose }: { templates: DatabaseTemplate[]; path: string; onClose?: () => void }) {
  const navigate = useNavigate();
  const [engine, setEngine] = useState(templates[0].engine);
  const [name, setName] = useState("");
  const [version, setVersion] = useState("");
  const [environment, setEnvironment] = useState<Record<string, string>>({});
  const { pending, error, run } = useAction();
  const template = templates.find((item) => item.engine === engine) ?? templates[0];

  function selectEngine(next: string) {
    setEngine(next);
    setEnvironment({});
    setVersion("");
  }

  async function submit(event: FormEvent) {
    event.preventDefault();
    await run(async () => {
      const filled = Object.fromEntries(Object.entries(environment).filter(([, value]) => value));
      const service = await api<Service>(path, { method: "POST", body: { engine, name, image_version: version, environment: filled } });
      navigate(service.id);
    });
  }

  return (
    <form className="grid gap-5" onSubmit={submit} aria-label="New app">
      <fieldset className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        <legend className="section-title mb-3">Choose a template</legend>
        {templates.map((item) => {
          const selected = engine === item.engine;
          return (
            <label className="cursor-pointer" key={item.engine}>
              <input className="peer sr-only" type="radio" name="engine" checked={selected} onChange={() => selectEngine(item.engine)} />
              <span className={`flex items-center gap-3 rounded-panel border-2 bg-raised p-4 transition-colors duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam ${selected ? "border-seafoam" : "border-transparent hover:border-line-strong"}`}>
                <EngineMark engine={item.engine} />
                <span className="min-w-0">
                  <span className="block font-display text-base font-semibold">{engines[item.engine]?.label ?? item.engine}</span>
                  <span className="block truncate text-meta text-ink-soft">{engines[item.engine]?.description ?? item.image}</span>
                </span>
              </span>
            </label>
          );
        })}
      </fieldset>
      <div className="grid max-w-2xl gap-4">
        <Field label="App name" maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title="Lowercase letters, numbers and hyphens, starting with a letter" placeholder={`${engine}-generated`} hint="Optional. Leave blank to generate one." value={name} onChange={(event) => setName(event.target.value)} />
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
                key={`${engine}-${field.name}`}
              />
            ))}
          </div>
        </details>
        <FormError message={error} />
        <div className="flex flex-wrap gap-2">
          <button className="btn btn-primary" type="submit" disabled={pending}>
            {pending ? "Creating…" : `Create ${engines[engine]?.label ?? engine} app`}
          </button>
          {onClose && (
            <button className="btn" type="button" onClick={onClose} disabled={pending}>
              Cancel
            </button>
          )}
        </div>
      </div>
    </form>
  );
}
