import { Check, Copy, Eye, EyeOff } from "lucide-react";
import { useRef, useState } from "react";
import { t } from "@/lib/i18n";

export function CopyField({ label, value, secret = false }: { label: string; value: string; secret?: boolean }) {
  const [copied, setCopied] = useState(false);
  const [revealed, setRevealed] = useState(!secret);
  const inputRef = useRef<HTMLInputElement>(null);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1500);
    } catch {
      setRevealed(true);
      inputRef.current?.select();
    }
  }

  return (
    <div className="min-w-0">
      <label className="mb-1 block text-meta font-medium" htmlFor={`copy-${label}`}>
        {label}
      </label>
      <div className="flex items-center gap-1">
        <input className="field min-w-0 flex-1 font-mono text-meta" id={`copy-${label}`} ref={inputRef} type={revealed ? "text" : "password"} readOnly value={value} onFocus={(event) => event.target.select()} />
        {secret && (
          <button className="btn size-9 flex-none justify-center px-0" type="button" aria-label={revealed ? t("Hide {label}", { label }) : t("Show {label}", { label })} title={revealed ? t("Hide") : t("Show")} onClick={() => setRevealed(!revealed)}>
            {revealed ? <EyeOff aria-hidden="true" size={15} /> : <Eye aria-hidden="true" size={15} />}
          </button>
        )}
        <button className="btn size-9 flex-none justify-center px-0" type="button" aria-label={copied ? t("{label} copied", { label }) : t("Copy {label}", { label })} title={copied ? t("Copied") : t("Copy")} onClick={copy}>
          {copied ? <Check className="text-ok" aria-hidden="true" size={15} /> : <Copy aria-hidden="true" size={15} />}
        </button>
      </div>
    </div>
  );
}
