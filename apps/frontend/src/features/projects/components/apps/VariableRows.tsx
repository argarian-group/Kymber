import type { ClipboardEvent } from "react";
import { Plus, X } from "lucide-react";
import { t } from "@/lib/i18n";

export type Variable = { id: number; name: string; value: string };

export function parseDotEnv(text: string) {
  const parsed: { name: string; value: string }[] = [];
  for (const line of text.split(/\r?\n/)) {
    const match = line.trim().match(/^(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(.*)$/);
    if (!match) continue;
    const value = match[2].replace(/^(["'])(.*)\1$/, "$2");
    parsed.push({ name: match[1], value });
  }
  return parsed;
}

export function VariableRows({ variables, onChange }: { variables: Variable[]; onChange: (variables: Variable[]) => void }) {
  function update(id: number, change: Partial<Variable>) {
    onChange(variables.map((variable) => (variable.id === id ? { ...variable, ...change } : variable)));
  }

  function paste(id: number, event: ClipboardEvent<HTMLInputElement>) {
    const parsed = parseDotEnv(event.clipboardData.getData("text"));
    if (!parsed.length) return;
    event.preventDefault();
    const pasted = new Map(parsed.map(({ name, value }) => [name, value]));
    const kept = variables.filter((variable) => variable.id !== id || variable.name || variable.value).map((variable) => (pasted.has(variable.name) ? { ...variable, value: pasted.get(variable.name)! } : variable));
    const known = new Set(kept.map(({ name }) => name));
    onChange([...kept, ...parsed.filter(({ name }) => !known.has(name)).map((variable, index) => ({ id: Date.now() + index, ...variable }))]);
  }

  return (
    <>
      {variables.map(({ id, name, value }) => (
        <div className="grid grid-cols-[1fr_auto] items-center gap-2 sm:grid-cols-[1fr_1fr_auto]" key={id}>
          <input className="field font-mono" aria-label={t("Variable name")} autoComplete="off" pattern="[A-Za-z_][A-Za-z0-9_]*" title={t("Letters, numbers and underscores, not starting with a number")} required placeholder="NAME" value={name} onChange={(event) => update(id, { name: event.target.value })} onPaste={(event) => paste(id, event)} />
          <button className="btn order-none px-2 sm:order-last" type="button" aria-label={t("Remove {name}", { name: name || t("variable") })} onClick={() => onChange(variables.filter((variable) => variable.id !== id))}>
            <X aria-hidden="true" size={16} />
          </button>
          <input className="field col-span-2 sm:col-span-1" type="password" aria-label={t("Value of {name}", { name: name || t("variable") })} autoComplete="off" placeholder={t("Value")} value={value} onChange={(event) => update(id, { value: event.target.value })} />
        </div>
      ))}
      <button className="btn w-fit" type="button" onClick={() => onChange([...variables, { id: Date.now(), name: "", value: "" }])}>
        <Plus aria-hidden="true" size={16} />
        {t("Add variable")}
      </button>
      <p className="text-xs text-ink-soft">{t("Tip: paste a whole .env file into a name field to add every variable at once.")}</p>
    </>
  );
}
