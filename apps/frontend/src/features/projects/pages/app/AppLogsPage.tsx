import { Check, Copy, Download, Maximize2, Minimize2, RefreshCw, Search } from "lucide-react";
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { EmptyState } from "@/components/ui/EmptyState";
import { FormError } from "@/components/ui/FormError";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api, errorMessage } from "@/lib/api";
import { downloadText } from "@/lib/downloadText";

const lineLimits = [100, 500, 1000, 5000, 10000];
const maxLines = 50000;
const validLimit = (value: unknown): value is number => Number.isInteger(value) && (value as number) >= 1 && (value as number) <= maxLines;
const refreshRates = [
  { seconds: 0, label: "Off" },
  { seconds: 5, label: "Every 5s" },
  { seconds: 10, label: "Every 10s" },
  { seconds: 30, label: "Every 30s" },
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
  const inFlight = useRef(false);
  const deployed = Boolean(service.server_id);
  const { limit, seconds, wrap } = preferences;

  const load = useCallback(async () => {
    if (inFlight.current) return;
    inFlight.current = true;
    setLoading(true);
    try {
      const result = await api<{ logs: string }>(`${path}/${service.id}/logs?source=runtime&tail=${limit}`);
      setLogs(result.logs);
      setUpdatedAt(new Date());
      setError(null);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  }, [path, service.id, limit]);

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

  if (!deployed) return <EmptyState title="No runtime logs yet" description="Logs appear here after the first deploy." />;

  const placeholder = logs === null ? (error ? "Logs could not be loaded." : "Loading logs…") : limited.length === 0 ? "No output yet." : "No lines match this search.";

  return (
    <AppSection title="Runtime logs" description="Output from the running container. Refreshing pauses while this tab is in the background.">
      <div className={`grid gap-3 bg-raised p-4 ${fullscreen ? "h-screen grid-rows-[auto_auto_auto_minmax(0,1fr)]" : ""}`} ref={panel}>
        <div className="flex flex-wrap items-center gap-2">
          <label className="relative min-w-40 flex-1">
            <span className="sr-only">Search logs</span>
            <Search className="pointer-events-none absolute top-1/2 left-2.5 -translate-y-1/2 text-ink-faint" aria-hidden="true" size={15} />
            <input className="field pl-8" type="search" placeholder="Search logs" value={query} onChange={(event) => setQuery(event.target.value)} />
          </label>
          <label className="flex items-center gap-2 text-meta text-ink-soft">
            Lines
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
                  {value.toLocaleString("en-US")}
                </option>
              ))}
              <option value="custom">Custom{lineLimits.includes(limit) ? "" : ` (${limit.toLocaleString("en-US")})`}</option>
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
              <input className="field w-28" type="number" autoFocus required min={1} max={maxLines} step={1} aria-label={`Number of lines, up to ${maxLines}`} value={customLimit} onChange={(event) => setCustomLimit(event.target.value)} />
              <button className="btn" type="submit">
                Apply
              </button>
            </form>
          )}
          <label className="flex items-center gap-2 text-meta text-ink-soft">
            Auto-refresh
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
            Refresh
          </button>
          <label className="flex cursor-pointer items-center gap-2 text-meta">
            <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={follow} onChange={(event) => setFollow(event.target.checked)} />
            Follow
          </label>
          <label className="flex cursor-pointer items-center gap-2 text-meta">
            <input className="size-4 cursor-pointer accent-seafoam" type="checkbox" checked={wrap} onChange={(event) => save({ wrap: event.target.checked })} />
            Wrap lines
          </label>
          <div className="ml-auto flex gap-1">
            <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label={copied === false ? "Could not copy logs" : copied ? "Logs copied" : "Copy logs"} title={copied === false ? "Could not copy" : copied ? "Copied" : "Copy"} onClick={copy}>
              {copied ? <Check className="text-ok" aria-hidden="true" size={15} /> : <Copy aria-hidden="true" size={15} />}
            </button>
            <button className="btn size-9 justify-center px-0" type="button" disabled={!text} aria-label="Download logs" title="Download" onClick={() => downloadText(`${service.name}-logs.txt`, `${text}\n`)}>
              <Download aria-hidden="true" size={15} />
            </button>
            <button className="btn size-9 justify-center px-0" type="button" aria-label={fullscreen ? "Exit full screen" : "Full screen"} title={fullscreen ? "Exit full screen" : "Full screen"} onClick={() => (fullscreen ? document.exitFullscreen() : panel.current?.requestFullscreen())}>
              {fullscreen ? <Minimize2 aria-hidden="true" size={15} /> : <Maximize2 aria-hidden="true" size={15} />}
            </button>
          </div>
        </div>
        <div className="grid gap-2">
          <FormError message={error} />
          <p className="text-xs text-ink-soft tabular-nums">
            {query.trim() ? `${shown.length} of ${limited.length} lines match` : `Showing ${limited.length} ${limited.length === 1 ? "line" : "lines"}`}
            {updatedAt && ` · Updated ${updatedAt.toLocaleTimeString()}`}
          </p>
        </div>
        <pre
          className={`overflow-auto rounded-control bg-canvas p-4 font-mono text-xs text-ink-soft ${wrap ? "whitespace-pre-wrap wrap-break-word" : "whitespace-pre"} ${fullscreen ? "min-h-0" : "max-h-[60vh] min-h-48"}`}
          ref={output}
          tabIndex={0}
          aria-label="Log output"
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
