import { Check, Copy, Download, Maximize2, Minimize2, RefreshCw, Search } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api, errorMessage } from "@/lib/api";
import { downloadText } from "@/lib/downloadText";
import { language, t } from "@/lib/i18n";

const lineLimits = [100, 500, 1000, 5000, 10000];
const maxLines = 50000;
const validLimit = (value: unknown): value is number => typeof value === "number" && Number.isInteger(value) && value >= 1 && value <= maxLines;
const refreshRates = [
  { seconds: 0, label: t("Off") },
  { seconds: 5, label: t("Every {n}s", { n: 5 }) },
  { seconds: 10, label: t("Every {n}s", { n: 10 }) },
  { seconds: 30, label: t("Every {n}s", { n: 30 }) },
];
const preferencesKey = "kymber.app-logs";
const defaults = { limit: 100, seconds: 5, wrap: true };

function loadPreferences() {
  try {
    const saved = JSON.parse(localStorage.getItem(preferencesKey) ?? "{}");
    return {
      limit: validLimit(saved.limit) ? saved.limit : defaults.limit,
      seconds: refreshRates.some(({ seconds }) => seconds === saved.seconds) ? (saved.seconds as number) : defaults.seconds,
      wrap: typeof saved.wrap === "boolean" ? saved.wrap : defaults.wrap,
    };
  } catch {
    return defaults;
  }
}

export function AppLogsPage() {
  const { service, path } = useAppOutlet();
  const [searchParams, setSearchParams] = useSearchParams();
  const container = searchParams.get("container") ?? "";
  const [preferences, setPreferences] = useState(loadPreferences);
  const [logs, setLogs] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [updatedAt, setUpdatedAt] = useState<Date | null>(null);
  const [query, setQuery] = useState("");
  const [follow, setFollow] = useState(true);
  const [copied, setCopied] = useState<boolean | null>(null);
  const [fullscreen, setFullscreen] = useState(false);
  const [customLimit, setCustomLimit] = useState<string | null>(null);
  const panel = useRef<HTMLDivElement>(null);
  const output = useRef<HTMLPreElement>(null);
  const inFlight = useRef<string | null>(null);
  const deployed = Boolean(service.server_id);
  const { limit, seconds, wrap } = preferences;

  const load = useCallback(async () => {
    const requestKey = `${limit}:${container}`;
    if (inFlight.current === requestKey) return;
    inFlight.current = requestKey;
    setLoading(true);
    try {
      const query = new URLSearchParams({ source: "runtime", tail: String(limit) });
      if (container) query.set("container", container);
      const result = await api<{ logs: string }>(`${path}/${service.id}/logs?${query}`);
      if (inFlight.current !== requestKey) return;
      setLogs(result.logs);
      setUpdatedAt(new Date());
      setError(null);
    } catch (cause) {
      if (inFlight.current === requestKey) setError(errorMessage(cause));
    } finally {
      if (inFlight.current === requestKey) {
        inFlight.current = null;
        setLoading(false);
      }
    }
  }, [path, service.id, limit, container]);

  useEffect(() => {
    if (deployed) void load();
  }, [deployed, load]);

  useEffect(() => {
    if (!deployed || !seconds) return;
    const refreshVisible = () => {
      if (document.visibilityState === "visible") void load();
    };
    const timer = window.setInterval(refreshVisible, seconds * 1000);
    document.addEventListener("visibilitychange", refreshVisible);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener("visibilitychange", refreshVisible);
    };
  }, [deployed, seconds, load]);

  useEffect(() => {
    const sync = () => setFullscreen(document.fullscreenElement === panel.current);
    document.addEventListener("fullscreenchange", sync);
    return () => document.removeEventListener("fullscreenchange", sync);
  }, []);

  const { limited, shown } = useMemo(() => {
    const lines = (logs ?? "").split("\n");
    if (lines.at(-1) === "") lines.pop();
    const limited = lines.slice(-limit);
    const needle = query.trim().toLowerCase();
    return { limited, shown: needle ? limited.filter((line) => line.toLowerCase().includes(needle)) : limited };
  }, [logs, limit, query]);
  const text = shown.join("\n");

  useLayoutEffect(() => {
    if (follow && output.current) output.current.scrollTop = output.current.scrollHeight;
  }, [text, follow, wrap, fullscreen]);

  function save(change: Partial<typeof defaults>) {
    const next = { ...preferences, ...change };
    setPreferences(next);
    try {
      localStorage.setItem(preferencesKey, JSON.stringify(next));
    } catch {}
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
    } catch {
      setCopied(false);
    }
    window.setTimeout(() => setCopied(null), 1500);
  }

  if (!deployed) return <EmptyState title={t("No runtime logs yet")} description={t("Logs appear here after the first deploy.")} />;

  const placeholder = logs === null ? (error ? t("Logs could not be loaded.") : t("Loading logs…")) : limited.length === 0 ? t("No output yet.") : t("No lines match this search.");

  return (
      <AppSection title={container ? t("Logs: {container}", { container }) : t("Runtime logs")} description={t("Output from the running container. Refreshing pauses while this tab is in the background.")}>
      <div className={`grid gap-3 bg-raised p-4 ${fullscreen ? "h-screen grid-rows-[auto_auto_auto_minmax(0,1fr)]" : ""}`} ref={panel}>
        <div className="flex flex-wrap items-center gap-2">
          {service.compose && <label className="flex items-center gap-2 text-meta text-ink-soft">{t("Container")}<select className="field w-auto cursor-pointer" value={container} onChange={(event) => {
            const next = new URLSearchParams(searchParams);
            if (event.target.value) next.set("container", event.target.value);
            else next.delete("container");
            setSearchParams(next);
          }}><option value="">{t("All containers")}</option>{service.containers?.map(({ name }) => <option key={name} value={name}>{name}</option>)}</select></label>}
          <label className="relative min-w-40 flex-1">
            <span className="sr-only">{t("Search logs")}</span>
            <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
            <input className="field pl-8" type="search" placeholder={t("Search logs")} value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
          <label className="flex items-center gap-2 text-meta text-ink-soft">
            {t("Lines")}
            <select
              className="field w-auto cursor-pointer"
              value={customLimit !== null || !lineLimits.includes(limit) ? "custom" : limit}
              onChange={(event) => {
                if (event.target.value === "custom") setCustomLimit(String(limit));
                else {
                  setCustomLimit(null);
                  save({ limit: Number(event.target.value) });
                }
              }}
            >
              {lineLimits.map((value) => (
                <option value={value} key={value}>
                  {value.toLocaleString(language)}
                </option>
              ))}
              <option value="custom">{t("Custom")}{lineLimits.includes(limit) ? "" : ` (${limit.toLocaleString(language)})`}</option>
            </select>
          </label>
          {customLimit !== null && (
            <form
              className="flex items-center gap-2"
              onSubmit={(event) => {
                event.preventDefault();
                const value = Number(customLimit);
                if (!validLimit(value)) return;
                save({ limit: value });
                setCustomLimit(null);
              }}
            >
              <input className="field w-28" type="number" autoFocus required min={1} max={maxLines} step={1} aria-label={t("Number of lines, up to {max}", { max: maxLines })} value={customLimit} onChange={(event) => setCustomLimit(event.target.value)} />
              <button className="btn" type="submit">
                {t("Apply")}
              </button>
            </form>
          )}
          <label className="flex items-center gap-2 text-meta text-ink-soft">
            {t("Auto-refresh")}
            <select className="field w-auto cursor-pointer" value={seconds} onChange={(event) => save({ seconds: Number(event.target.value) })}>
              {refreshRates.map(({ seconds: value, label }) => (
                <option value={value} key={value}>
                  {label}
                </option>
              ))}
            </select>
          </label>
        </div>
        <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
          <button className="btn" type="button" disabled={loading} onClick={load}>
            <RefreshCw className={loading ? "animate-spin motion-reduce:animate-none" : ""} aria-hidden="true" size={15} />
            {t("Refresh")}
          </button>
          <label className="flex cursor-pointer items-center gap-2 text-meta">
            <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={follow} onChange={(event) => setFollow(event.target.checked)} />
            {t("Follow")}
          </label>
          <label className="flex cursor-pointer items-center gap-2 text-meta">
            <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={wrap} onChange={(event) => save({ wrap: event.target.checked })} />
            {t("Wrap lines")}
          </label>
          <div className="ml-auto flex gap-1">
            <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label={copied === false ? t("Could not copy logs") : copied ? t("Logs copied") : t("Copy logs")} title={copied === false ? t("Could not copy") : copied ? t("Copied") : t("Copy")} onClick={copy}>
              {copied ? <Check className="text-ok" aria-hidden="true" size={15} /> : <Copy aria-hidden="true" size={15} />}
            </button>
            <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label={t("Download logs")} title={t("Download")} onClick={() => downloadText(`${service.name}-logs.txt`, `${text}\n`)}>
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
            {query.trim() ? t("{shown} of {total} lines match", { shown: shown.length, total: limited.length }) : limited.length === 1 ? t("Showing 1 line") : t("Showing {count} lines", { count: limited.length })}
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
    </AppSection>
  );
}
