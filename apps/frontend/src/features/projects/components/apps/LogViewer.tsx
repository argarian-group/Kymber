import { Check, Copy, Download, Maximize2, Minimize2, Search } from "lucide-react";
import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { FormError } from "@/components/ui/FormError";
import { downloadText } from "@/lib/downloadText";
import { language, t } from "@/lib/i18n";

export function LogViewer({ logs, error, loadingText, fileName, wrap, onWrapChange, updatedAt, controls, actions }: {
  logs: string | null;
  error: string | null;
  loadingText: string;
  fileName: string;
  wrap: boolean;
  onWrapChange: (wrap: boolean) => void;
  updatedAt?: Date | null;
  controls?: ReactNode;
  actions?: ReactNode;
}) {
  const [query, setQuery] = useState("");
  const [follow, setFollow] = useState(true);
  const [copied, setCopied] = useState<boolean | null>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const panel = useRef<HTMLDivElement>(null);
  const output = useRef<HTMLPreElement>(null);

  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === panel.current);
    document.addEventListener("fullscreenchange", sync);
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);

  const { lines, shown } = useMemo(() => {
    const lines = (logs ?? "").split("\n");
    if (lines.at(-1) === "") lines.pop();
    const needle = query.trim().toLowerCase();
    return { lines, shown: needle ? lines.filter((line) => line.toLowerCase().includes(needle)) : lines };
  }, [logs, query]);
  const text = shown.join("\n");

  useLayoutEffect(() => {
    if (follow && output.current) output.current.scrollTop = output.current.scrollHeight;
  }, [text, follow, wrap, fullscreen]);

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      setCopied(false);
    }
    window.setTimeout(() => setCopied(null), 1500);
  }

  const placeholder = logs === null ? (error ? t("Logs could not be loaded.") : loadingText) : lines.length === 0 ? t("No output yet.") : t("No lines match this search.");

  return (
    <div className={`grid gap-3 bg-raised p-4 ${fullscreen ? "h-screen grid-rows-[auto_auto_auto_minmax(0,1fr)]" : ""}`} ref={panel}>
      <div className="flex flex-wrap items-center gap-2">
        <label className="relative min-w-40 flex-1">
          <span className="sr-only">{t("Search logs")}</span>
          <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
          <input className="field pl-8" type="search" placeholder={t("Search logs")} value={query} onChange={(event) => setQuery(event.target.value)} />
        </label>
        {controls}
      </div>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        {actions}
        <label className="flex cursor-pointer items-center gap-2 text-meta">
          <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={follow} onChange={(event) => setFollow(event.target.checked)} />
          {t("Follow")}
        </label>
        <label className="flex cursor-pointer items-center gap-2 text-meta">
          <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={wrap} onChange={(event) => onWrapChange(event.target.checked)} />
          {t("Wrap lines")}
        </label>
        <div className="ml-auto flex gap-1">
          <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label={copied === false ? t("Could not copy logs") : copied ? t("Logs copied") : t("Copy logs")} title={copied === false ? t("Could not copy") : copied ? t("Copied") : t("Copy")} onClick={copy}>
            {copied ? <Check className="text-ok" aria-hidden="true" size={15} /> : <Copy aria-hidden="true" size={15} />}
          </button>
          <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label={t("Download logs")} title={t("Download")} onClick={() => downloadText(fileName, `${text}\n`)}>
            <Download aria-hidden="true" size={15} />
          </button>
          <button className="btn size-9 justify-center px-0" type="button" aria-label={fullscreen ? t("Exit full screen") : t("Full screen")} title={fullscreen ? t("Exit full screen") : t("Full screen")} onClick={() => (fullscreen ? document.exitFullscreen() : panel.current?.requestFullscreen())}>
            {fullscreen ? <Minimize2 aria-hidden="true" size={15} /> : <Maximize2 aria-hidden="true" size={15} />}
          </button>
        </div>
      </div>
      <div className="grid gap-2">
        <FormError message={error} />
        <p className="text-xs text-ink-soft tabular-nums">
          {query.trim() ? t("{shown} of {total} lines match", { shown: shown.length, total: lines.length }) : lines.length === 1 ? t("Showing 1 line") : t("Showing {count} lines", { count: lines.length })}
          {updatedAt && ` · ${t("Updated {time}", { time: updatedAt.toLocaleTimeString(language) })}`}
        </p>
      </div>
      <pre
        className={`overflow-auto rounded-control bg-canvas p-4 font-mono text-xs text-ink-soft ${wrap ? "whitespace-pre-wrap wrap-break-word" : "whitespace-pre"} ${fullscreen ? "min-h-0" : "max-h-[60vh] min-h-48"}`}
        ref={output}
        tabIndex={0}
        aria-label={t("Log output")}
        onScroll={({ currentTarget: { scrollTop, clientHeight, scrollHeight } }) => {
          if (scrollHeight - scrollTop - clientHeight > 24) setFollow(false);
        }}
      >
        {text || placeholder}
      </pre>
    </div>
  );
}
