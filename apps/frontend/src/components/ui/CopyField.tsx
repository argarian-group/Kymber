import { useRef, useState } from "react";

export function CopyField({ label, value, secret = false }: { label: string; value: string; secret?: boolean }) {
  const [copied, setCopied] = useState(false);
  const [revealed, setRevealed] = useState(!secret);
  const inputRef = useRef<HTMLInputElement>(null);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
    } catch {
      setRevealed(true);
      inputRef.current?.select();
    }
  }

  return (
    <div className="min-w-0">
      <label>
        <span className="mb-1 block text-meta font-medium">{label}</span>
        <input className="field" ref={inputRef} type={revealed ? "text" : "password"} readOnly value={value} onFocus={(event) => event.target.select()} />
      </label>
      <div className="mt-2 flex flex-wrap gap-2">
        <button className="btn" type="button" onClick={copy}>
          {copied ? "Copied" : "Copy"}
        </button>
        {secret && (
          <button className="btn" type="button" onClick={() => setRevealed(!revealed)}>
            {revealed ? "Hide" : "Show"}
          </button>
        )}
      </div>
    </div>
  );
}
