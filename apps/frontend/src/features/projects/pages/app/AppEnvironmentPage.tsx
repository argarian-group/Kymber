import { Code, Eye, EyeOff } from "lucide-react";
import { useState } from "react";
import { Navigate } from "react-router";
import { PageSkeleton } from "@/components/ui/PageSkeleton";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { ComposeEnvironmentEditor } from "@/features/projects/components/apps/ComposeForm";
import { formatDotEnv, invalidDotEnvLine, parseDotEnv, VariableRows, type Variable } from "@/features/projects/components/apps/VariableRows";
import { isServiceBusy, type Service } from "@/features/projects/services";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { FormFooter } from "@/features/settings/components/FormFooter";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

export function AppEnvironmentPage() {
  const { service, detail, template, path, canWrite, onChanged } = useAppOutlet();
  const { pending, error, run } = useAction();

  if (!canWrite) return <Navigate to=".." replace />;
  if (service.compose) {
    const busy = isServiceBusy(service.status);
    function saveCompose(file: string, deploy: boolean): Promise<boolean> {
      return run(async () => {
        await api(`${path}/${service.id}`, { method: "PUT", body: { compose_file: file } });
        if (deploy && service.status !== "draft") await api(`${path}/${service.id}/deploy`, { method: "POST", body: {} });
        onChanged();
      });
    }
    return <AppSection title={t("Environment variables")} description={t("Choose a service to edit its container environment.")}>
      <ComposeEnvironmentEditor key={service.compose_file} initialFile={service.compose_file ?? ""} pending={pending} error={error} submitLabel={t("Save")} deployLabel={t("Save and deploy stack")} deployDisabled={busy || service.status === "draft"} onSave={saveCompose} />
    </AppSection>;
  }
  if (!detail || !template) return <PageSkeleton />;
  return <CustomEnvironment key={detail.id} service={service} environment={detail.environment ?? {}} fields={template.custom ? [] : (template.fields ?? []).map(({ name }) => name)} path={path} onChanged={onChanged} />;
}

function initialVariables(environment: Record<string, string>, fields: string[]): Variable[] {
  const extra = Object.keys(environment).filter((name) => !fields.includes(name)).sort((a, b) => a.localeCompare(b));
  return [...fields, ...extra].map((name, id) => ({ id, name, value: environment[name] ?? "" }));
}

function CustomEnvironment({ service, environment, fields, path, onChanged }: { service: Service; environment: Record<string, string>; fields: string[]; path: string; onChanged: () => void }) {
  const [variables, setVariables] = useState<Variable[]>(() => initialVariables(environment, fields));
  const [developerView, setDeveloperView] = useState(false);
  const [text, setText] = useState("");
  const [revealed, setRevealed] = useState(false);
  const [saved, setSaved] = useState<string | null>(null);
  const [invalid, setInvalid] = useState<string | null>(null);
  const { pending, error, run } = useAction();
  const draft = service.status === "draft";
  const busy = isServiceBusy(service.status);
  const fixed = new Set(fields);

  function toggleDeveloperView() {
    setInvalid(null);
    if (developerView) {
      const parsed = parseDotEnv(text);
      const missing = variables.filter(({ name }) => fixed.has(name) && !parsed.some((variable) => variable.name === name));
      setVariables([...missing, ...parsed].map(({ name, value }, id) => ({ id, name, value })));
    }
    else setText(formatDotEnv(variables));
    setDeveloperView(!developerView);
  }

  async function save(deploy: boolean) {
    setSaved(null);
    setInvalid(null);
    const line = developerView ? invalidDotEnvLine(text) : null;
    if (line) {
      setInvalid(t("Line {line} is not KEY=value.", { line }));
      return;
    }
    const source = developerView ? parseDotEnv(text) : variables.filter(({ name }) => name);
    const next = Object.fromEntries(source.map(({ name, value }) => [name, value]));
    const removed = Object.fromEntries(Object.keys(environment).filter((name) => !(name in next) && !fixed.has(name)).map((name) => [name, ""]));
    await run(async () => {
      await api(`${path}/${service.id}`, { method: "PUT", body: { environment: { ...removed, ...next } } });
      if (deploy) await api(`${path}/${service.id}/deploy`, { method: "POST", body: {} });
      setSaved(deploy ? t("Saved. Deploying…") : draft ? t("Saved.") : t("Saved. Applies on the next deploy."));
      onChanged();
    });
  }

  return (
    <AppSection title={t("Environment variables")} description={draft ? t("These go into the container at deploy.") : t("Changes apply on the next deploy.")}>
      <form onSubmit={(event) => { event.preventDefault(); void save(false); }}>
        <div className="flex flex-wrap items-center justify-end gap-2 border-b border-line p-4">
          {!draft && fields.length > 0 && <p className="mr-auto text-meta text-ink-soft">{t("The template's own variables can only change before the first deploy.")}</p>}
          {!developerView && <button className="btn" type="button" aria-pressed={revealed} onClick={() => setRevealed(!revealed)}>{revealed ? <EyeOff aria-hidden="true" size={16} /> : <Eye aria-hidden="true" size={16} />}{revealed ? t("Hide values") : t("Show values")}</button>}
          <button className="btn" type="button" aria-pressed={developerView} onClick={toggleDeveloperView}><Code aria-hidden="true" size={16} />{developerView ? t("Normal view") : t("Developer view")}</button>
        </div>
        <div className="grid gap-3 p-4">
          {developerView ? (
            <label className="block">
              <span className="mb-1 block text-meta font-medium">{t(".env file")}</span>
              <textarea className="field min-h-64 font-mono text-xs" spellCheck={false} autoComplete="off" placeholder={"DATABASE_URL=postgres://user:password@host/db\nNODE_ENV=production"} value={text} onChange={(event) => setText(event.target.value)} />
              <span className="mt-1 block text-xs text-ink-soft">{t("One KEY=value per line. Lines starting with # are ignored.")}</span>
            </label>
          ) : (
            <VariableRows variables={variables} revealed={revealed} fixed={fixed} readOnly={draft ? undefined : fixed} onChange={setVariables} />
          )}
        </div>
        <FormFooter error={invalid ?? error} saved={saved}>
          <button className={`btn ${draft ? "btn-primary" : ""}`} type="submit" disabled={pending || busy}>{pending ? t("Saving…") : t("Save")}</button>
          {!draft && <button className="btn btn-primary" type="button" disabled={pending || busy} onClick={() => void save(true)}>{t("Save and deploy")}</button>}
        </FormFooter>
      </form>
    </AppSection>
  );
}
