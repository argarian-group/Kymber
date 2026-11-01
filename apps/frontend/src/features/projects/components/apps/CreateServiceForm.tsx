import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import type { DatabaseTemplate, Service } from "@/features/projects/services";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";

export function CreateServiceForm({ templates, path, onClose }: { templates: DatabaseTemplate[]; path: string; onClose: () => void }) {
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
    <form className="grid gap-4 rounded-panel bg-raised p-4" onSubmit={submit} aria-label="New app">
      <fieldset className="flex flex-wrap gap-2">
        <legend className="mb-1 text-meta font-medium">Template</legend>
        {templates.map((item) => (
          <label className="cursor-pointer" key={item.engine}>
            <input className="peer sr-only" type="radio" name="engine" checked={engine === item.engine} onChange={() => selectEngine(item.engine)} />
            <span className={`block rounded-control border-2 px-3 py-1.5 font-medium transition-colors duration-100 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2 peer-focus-visible:outline-seafoam ${engine === item.engine ? "border-seafoam" : "border-line-strong hover:border-ink-faint"}`}>{item.engine}</span>
          </label>
        ))}
      </fieldset>
      <div className="grid items-start gap-4 sm:grid-cols-2">
        <Field label="Name" maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title="Lowercase letters, numbers and hyphens, starting with a letter" hint="Leave blank to generate." value={name} onChange={(event) => setName(event.target.value)} />
        <Field label="Image version" maxLength={128} placeholder={template.default_version} value={version} onChange={(event) => setVersion(event.target.value)} />
        {template.fields.map((field) => (
          <Field
            label={field.label}
            type={field.type === "password" ? "password" : "text"}
            autoComplete="off"
            required={field.required && !field.generate && !field.default}
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
      <FormError message={error} />
      <div className="flex flex-wrap gap-2">
        <button className="btn btn-primary" type="submit" disabled={pending}>
          {pending ? "Creating…" : "Create app"}
        </button>
        <button className="btn" type="button" onClick={onClose} disabled={pending}>
          Cancel
        </button>
      </div>
    </form>
  );
}
