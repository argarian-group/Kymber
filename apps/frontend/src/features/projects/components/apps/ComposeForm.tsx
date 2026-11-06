import { ArrowDown, ArrowDownAZ, ArrowUpAZ, Check, Eye, EyeOff, Plus, Search, Trash2 } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useNavigate, useSearchParams } from "react-router";
import { Field } from "@/components/ui/Field";
import { FormError } from "@/components/ui/FormError";
import { api } from "@/lib/api";
import { useAction } from "@/lib/useAction";
import { t } from "@/lib/i18n";

type ComposeServiceInput = { name: string; image: string; ports: string; expose: string; environment: string; volumes: string };

const initialService: ComposeServiceInput = { name: "web", image: "nginx:alpine", ports: "", expose: "", environment: "", volumes: "" };
const lines = (value: string) => value.split("\n").map((line) => line.trim()).filter(Boolean);
const yamlString = (value: string) => JSON.stringify(value);
const listKeys = new Set(["ports", "expose", "environment", "volumes"]);

function scalar(value: string) {
  try {
    return JSON.parse(value) as string;
  } catch {
    return value.replace(/^['"]|['"]$/g, "");
  }
}

function parseComposeFile(file: string): ComposeServiceInput[] | null {
  const services: ComposeServiceInput[] = [];
  let inServices = false;
  let current: ComposeServiceInput | undefined;
  let listKey: "ports" | "expose" | "environment" | "volumes" | null = null;
  for (const line of file.split("\n")) {
    const trimmed = line.trim();
    const indent = line.length - line.trimStart().length;
    if (!trimmed || trimmed.startsWith("#")) continue;
    if (indent === 0 && trimmed === "services:") {
      inServices = true;
      continue;
    }
    if (!inServices) continue;
    if (indent === 0) {
      inServices = false;
      current = undefined;
      continue;
    }
    if (indent === 2 && trimmed.endsWith(":")) {
      current = { ...initialService, name: trimmed.slice(0, -1), image: "" };
      services.push(current);
      listKey = null;
      continue;
    }
    if (!current) return null;
    if (indent === 4) {
      const [key, ...rest] = trimmed.split(":");
      const value = rest.join(":").trim();
      if (key === "image" && value) current.image = scalar(value);
      else if (listKeys.has(key) && !value) listKey = key as "ports" | "expose" | "environment" | "volumes";
      else return null;
      continue;
    }
    if (indent === 6 && trimmed.startsWith("- ") && listKey) {
      current[listKey] = `${current[listKey] ? `${current[listKey]}\n` : ""}${scalar(trimmed.slice(2).trim())}`;
      continue;
    }
    if (indent === 6 && listKey && trimmed.includes(":")) {
      const [key, ...rest] = trimmed.split(":");
      const value = rest.join(":").trim();
      if (listKey === "environment" && /^[A-Za-z_][A-Za-z0-9_]*$/.test(key) && value) {
        current.environment = `${current.environment ? `${current.environment}\n` : ""}${key}=${scalar(value)}`;
        continue;
      }
      return null;
    }
    return null;
  }
  return services.length ? services : null;
}

function composeFile(services: ComposeServiceInput[]) {
  const namedVolumes = new Set<string>();
  const definitions = services.map((service) => {
    const options = [`    image: ${yamlString(service.image.trim())}`];
    for (const [key, input] of [["ports", service.ports], ["expose", service.expose], ["environment", service.environment], ["volumes", service.volumes]] as const) {
      const values = lines(input);
      if (!values.length) continue;
      options.push(`    ${key}:`, ...values.map((value) => `      - ${yamlString(value)}`));
      if (key === "volumes") values.forEach((value) => namedVolumes.add(value.split(":", 1)[0]));
    }
    return `  ${service.name.trim()}:\n${options.join("\n")}`;
  });
  return `services:\n${definitions.join("\n")}\n${namedVolumes.size ? `volumes:\n${[...namedVolumes].map((volume) => `  ${volume}: {}`).join("\n")}\n` : ""}`;
}

export function ComposeServicesEditor({ initialFile, stackName, onStackNameChange, submitLabel, pending, error, onSave }: { initialFile?: string; stackName?: string; onStackNameChange?: (name: string) => void; submitLabel: string; pending: boolean; error: string | null; onSave: (file: string) => Promise<unknown> }) {
  const [file, setFile] = useState(initialFile ?? "services:\n  web:\n    image: nginx:alpine\n");
  const changed = !initialFile || file.trim() !== initialFile.trim();

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void onSave(file);
  }

  return <form className="grid gap-4 p-4" onSubmit={submit}>
    {onStackNameChange && <Field label={t("Stack name")} value={stackName ?? ""} onChange={(event) => onStackNameChange(event.target.value.toLowerCase().replace(/\s+/g, "-"))} maxLength={63} pattern="[a-z][a-z0-9\-]{1,62}" title={t("Lowercase letters, numbers and hyphens, starting with a letter")} required />}
    <label className="grid gap-2 text-meta font-medium text-ink">Docker Compose<textarea className="field min-h-96 w-full font-mono text-xs" value={file} onChange={(event) => setFile(event.target.value)} aria-label="Docker Compose" spellCheck={false} required /></label>
    <FormError message={error} />
    <button className="btn btn-primary w-fit" type="submit" disabled={pending || !changed}>{pending ? t("Saving…") : submitLabel}</button>
  </form>;
}

export function ComposeEnvironmentEditor({ initialFile, pending, error, submitLabel, deployLabel, deployDisabled, onSave }: { initialFile: string; pending: boolean; error: string | null; submitLabel: string; deployLabel: string; deployDisabled: boolean; onSave: (file: string, deploy: boolean) => Promise<unknown> }) {
  const parsed = parseComposeFile(initialFile);
  const [services, setServices] = useState(() => parsed ?? []);
  const [selected, setSelected] = useState(() => parsed?.find(({ environment }) => environment)?.name ?? parsed?.[0]?.name ?? "");
  const [search, setSearch] = useState("");
  const [descending, setDescending] = useState(false);
  const [rawMode, setRawMode] = useState(false);
  const [visibleValues, setVisibleValues] = useState<Record<number, boolean>>({});
  const [menuOpen, setMenuOpen] = useState(false);
  const [savedServices, setSavedServices] = useState(() => parsed ?? []);
  const rawPre = useRef<HTMLPreElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const service = services.find(({ name }) => name === selected);
  const variables = service?.environment ?? "";
  const rawLines = variables.split("\n");
  const entries = rawLines.map((line, index) => ({ line, index })).filter(({ line }) => line.trim() && !line.trimStart().startsWith("#")).map(({ line, index }) => {
    const separator = line.indexOf("=");
    return { index, name: separator < 0 ? line : line.slice(0, separator), value: separator < 0 ? "" : line.slice(separator + 1), hasValue: separator >= 0 };
  }).filter(({ name, hasValue }) => (/^[A-Za-z_][A-Za-z0-9_]*$/.test(name) || (!name && hasValue)) && (!search.trim() || name.toLowerCase().includes(search.trim().toLowerCase()))).sort((a, b) => a.name.localeCompare(b.name) * (descending ? -1 : 1));
  const allEntries = rawLines.filter((line) => line.trim() && !line.trimStart().startsWith("#")).map((line) => line.slice(0, line.indexOf("=") < 0 ? line.length : line.indexOf("=")));
  const validVariables = allEntries.every((name) => /^[A-Za-z_][A-Za-z0-9_]*$/.test(name)) && new Set(allEntries).size === allEntries.length;
  const normalizedServices = (items: ComposeServiceInput[]) => items.map((entry) => ({ ...entry, environment: entry.environment.split("\n").filter((line) => line.trim() && !line.trimStart().startsWith("#")).join("\n") }));
  const changed = parsed ? composeFile(services) !== composeFile(savedServices) : false;

  useEffect(() => {
    if (!menuOpen) return;
    function closeMenu(event: PointerEvent) {
      if (!menuRef.current?.contains(event.target as Node)) setMenuOpen(false);
    }
    document.addEventListener("pointerdown", closeMenu);
    return () => document.removeEventListener("pointerdown", closeMenu);
  }, [menuOpen]);

  function updateVariable(index: number, key: "name" | "value", value: string) {
    setServices((current) => current.map((entry) => {
      if (entry.name !== selected) return entry;
      const items = entry.environment.split("\n");
      const separator = items[index]?.indexOf("=") ?? -1;
      const name = separator < 0 ? items[index] ?? "" : items[index].slice(0, separator);
      const variableValue = separator < 0 ? "" : items[index].slice(separator + 1);
      const nextName = key === "name" ? value : name;
      items[index] = key === "name" && separator < 0 ? value : `${nextName}=${key === "value" ? value : variableValue}`;
      return { ...entry, environment: items.join("\n") };
    }));
  }

  function addVariable() {
    const base = "NEW_VARIABLE";
    let name = base;
    let suffix = 1;
    while (allEntries.includes(name)) name = `${base}_${suffix++}`;
    setSearch("");
    setServices((current) => current.map((entry) => entry.name === selected ? { ...entry, environment: [...entry.environment.split("\n").filter(Boolean), `${name}=`].join("\n") } : entry));
  }

  function removeVariable(index: number) {
    setServices((current) => current.map((entry) => {
      if (entry.name !== selected) return entry;
      const items = entry.environment.split("\n");
      items.splice(index, 1);
      return { ...entry, environment: items.join("\n") };
    }));
  }

  function cancelChanges() {
    setServices((current) => current.map((entry) => {
      const saved = savedServices.find(({ name }) => name === entry.name);
      return saved ?? entry;
    }));
  }

  async function saveChanges(deploy: boolean) {
    await onSave(composeFile(normalizedServices(services)), deploy);
    const normalized = normalizedServices(services);
    setServices(normalized);
    setSavedServices(normalized);
  }

  if (!parsed) return <p className="p-4 text-meta text-ink-soft">{t("This Compose file uses options that cannot be safely edited here. Edit variables in the original Compose configuration.")}</p>;

  return <form className="grid gap-4 p-4" onSubmit={(event) => { event.preventDefault(); void saveChanges(false); }}>
    <div className="grid max-w-sm gap-1.5">
      <span className="text-meta text-ink-soft">{t("Container service")}</span>
      <div className="relative" ref={menuRef}>
        <button className="field flex w-full items-center justify-between bg-[#131517] text-left" type="button" aria-haspopup="listbox" aria-expanded={menuOpen} onClick={() => setMenuOpen((open) => !open)}>{selected}<ArrowDown aria-hidden="true" size={16} /></button>
        {menuOpen && <div className="absolute z-20 mt-1 max-h-60 w-full overflow-auto rounded-panel bg-[#131517] p-1 shadow-lg" role="listbox" aria-label={t("Container service")}>
          {services.map(({ name }) => <button className={`flex w-full items-center justify-between rounded-control px-3 py-2 text-left text-meta hover:bg-canvas ${selected === name ? "text-accent" : "text-ink"}`} key={name} type="button" role="option" aria-selected={selected === name} onClick={() => { setSelected(name); setMenuOpen(false); }}>{name}{selected === name && <Check aria-hidden="true" size={15} />}</button>)}
        </div>}
      </div>
    </div>
    <section className="overflow-hidden">
      <header className="flex flex-wrap items-center justify-between gap-3 p-3">
        <div className="flex gap-1" role="group" aria-label={t("Environment editor mode")}>
          <button className={`btn ${!rawMode ? "btn-primary" : ""}`} type="button" aria-pressed={!rawMode} onClick={() => setRawMode(false)}>{t("Visual")}</button>
          <button className={`btn ${rawMode ? "btn-primary" : ""}`} type="button" aria-pressed={rawMode} onClick={() => setRawMode(true)}>{t("View Raw")}</button>
        </div>
        {!rawMode && <div className="flex min-w-0 flex-1 flex-wrap items-center justify-end gap-2">
        <label className="relative min-w-40 flex-1">
          <span className="sr-only">{t("Search environment variables")}</span>
          <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
          <input className="field pl-8" type="search" placeholder={t("Search environment variables")} value={search} onChange={(event) => setSearch(event.target.value)} />
        </label>
        <button className="btn" type="button" onClick={() => setDescending((value) => !value)}>{descending ? <ArrowDownAZ aria-hidden="true" size={15} /> : <ArrowUpAZ aria-hidden="true" size={15} />}{t("Sort")}</button>
        <button className="btn btn-primary" type="button" onClick={addVariable}><Plus aria-hidden="true" size={15} />{t("Add variable")}</button>
        </div>}
      </header>
      {rawMode ? <div className="relative min-h-72 bg-canvas font-mono text-xs leading-6">
        <pre ref={rawPre} className="pointer-events-none absolute inset-0 overflow-hidden whitespace-pre-wrap break-all p-3" aria-hidden="true">{rawLines.map((line, index) => {
        const separator = line.indexOf("=");
        const comment = line.trimStart().startsWith("#");
        return <span key={index} className={comment ? "text-ink-faint" : ""}><span className={comment ? "" : "text-seafoam"}>{separator < 0 || comment ? line : line.slice(0, separator)}</span>{separator >= 0 && !comment && <><span className="text-ink-faint">=</span><span className="text-ink">{line.slice(separator + 1)}</span></>}{index < rawLines.length - 1 ? "\n" : ""}</span>;
      })}</pre>
        <textarea className="relative min-h-72 w-full resize-y whitespace-pre-wrap break-all border-0 bg-transparent p-3 font-mono text-xs leading-6 text-transparent caret-ink selection:bg-white/20" aria-label={t("Raw environment variables")} spellCheck={false} value={variables} onScroll={(event) => { if (rawPre.current) rawPre.current.scrollTop = event.currentTarget.scrollTop; }} onChange={(event) => setServices((current) => current.map((entry) => entry.name === selected ? { ...entry, environment: event.target.value } : entry))} />
      </div> : <div className="overflow-x-auto">
        <table className="w-full text-left text-meta">
          <thead className="bg-canvas text-ink-soft"><tr><th className="px-3 py-2 font-medium">{t("Name")}</th><th className="px-3 py-2 font-medium">{t("Value")}</th><th className="w-10 px-3 py-2" /></tr></thead>
          <tbody>
            {entries.map(({ index, name, value, hasValue }) => <tr className="hover:bg-canvas/50" key={index}>
              <td className="min-w-52 p-2"><input className="field font-mono text-xs" aria-label={t("Variable name")} value={name} onChange={(event) => updateVariable(index, "name", event.target.value)} /></td>
              <td className="min-w-64 p-2"><div className="flex gap-2"><input className="field font-mono text-xs" type={visibleValues[index] ? "text" : "password"} aria-label={t("Variable value for {name}", { name })} disabled={!hasValue} placeholder={hasValue ? "" : t("Inherited from host environment")} value={value} onChange={(event) => updateVariable(index, "value", event.target.value)} /><button className="btn size-9 shrink-0 justify-center px-0" type="button" disabled={!hasValue} aria-label={visibleValues[index] ? t("Hide variable value") : t("Show variable value")} onClick={() => setVisibleValues((current) => ({ ...current, [index]: !current[index] }))}>{visibleValues[index] ? <EyeOff aria-hidden="true" size={15} /> : <Eye aria-hidden="true" size={15} />}</button></div></td>
              <td className="p-2"><button className="btn btn-danger size-8 justify-center px-0" type="button" aria-label={t("Remove variable {name}", { name })} onClick={() => removeVariable(index)}><Trash2 aria-hidden="true" size={14} /></button></td>
            </tr>)}
            {entries.length === 0 && <tr><td className="p-4 text-ink-soft" colSpan={3}>{t("No environment variables match your search.")}</td></tr>}
          </tbody>
        </table>
      </div>
      }
    </section>
    <FormError message={error} />
    {!validVariables && <p className="text-meta text-danger" role="alert">{t("Variable names must be unique and use letters, numbers and underscores.")}</p>}
    {changed && <div className="flex flex-wrap gap-2">
      <button className="btn" type="button" disabled={pending} onClick={cancelChanges}>{t("Cancel")}</button>
      <button className="btn" type="submit" disabled={pending || !validVariables}>{pending ? t("Saving…") : submitLabel}</button>
      <button className="btn btn-primary" type="button" disabled={pending || !validVariables || deployDisabled} onClick={() => void saveChanges(true)}>{pending ? t("Saving…") : deployLabel}</button>
    </div>}
  </form>;
}

export function ComposeForm({ path }: { path: string }) {
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [name, setName] = useState("");
  const environmentId = params.get("environment") ?? "";
  const { pending, error, run } = useAction();
  return <div className="grid max-w-3xl gap-4">
    <h2 className="section-title">{t("Docker Compose stack")}</h2>
    <ComposeServicesEditor stackName={name} onStackNameChange={setName} submitLabel={t("Create stack")} pending={pending} error={error} onSave={(file) => run(async () => {
      const service = await api<{ id: string }>(path, { method: "POST", body: { name, compose_file: file, environment_id: environmentId } });
      navigate(`../apps/${service.id}`);
    })} />
  </div>;
}
