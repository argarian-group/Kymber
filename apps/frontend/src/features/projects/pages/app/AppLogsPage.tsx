import { RefreshCw } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { EmptyState } from "@/components/ui/EmptyState";
import { AppSection } from "@/features/projects/components/apps/AppSection";
import { LogViewer } from "@/features/projects/components/apps/LogViewer";
import { useAppOutlet } from "@/features/projects/useAppOutlet";
import { api, errorMessage } from "@/lib/api";
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
  const [customLimit, setCustomLimit] = useState<string | null>(null);
  const inFlight = useRef<object | null>(null);
  const deployed = Boolean(service.server_id);
  const { limit, seconds, wrap } = preferences;

  const load = useCallback(async () => {
    if (inFlight.current) return;
    const request = {};
    inFlight.current = request;
    setLoading(true);
    try {
      const query = new URLSearchParams({ source: "runtime", tail: String(limit) });
      if (container) query.set("container", container);
      const result = await api<{ logs: string }>(`${path}/${service.id}/logs?${query}`);
      if (inFlight.current !== request) return;
      setLogs(result.logs);
      setUpdatedAt(new Date());
      setError(null);
    } catch (cause) {
      if (inFlight.current === request) setError(errorMessage(cause));
    } finally {
      if (inFlight.current === request) {
        inFlight.current = null;
        setLoading(false);
      }
    }
  }, [path, service.id, limit, container]);

  useEffect(() => {
    inFlight.current = null;
    setLogs(null);
    setError(null);
    setUpdatedAt(null);
    setLoading(false);
    if (deployed) void load();
    return () => {
      inFlight.current = null;
    };
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

  const text = useMemo(() => {
    const lines = (logs ?? "").split("\n");
    if (lines.at(-1) === "") lines.pop();
    return lines.slice(-limit).join("\n");
  }, [logs, limit]);

  function save(change: Partial<typeof defaults>) {
    const next = { ...preferences, ...change };
    setPreferences(next);
    try {
      localStorage.setItem(preferencesKey, JSON.stringify(next));
    } catch {}
  }

  if (!deployed) return <EmptyState title={t("No runtime logs yet")} description={t("Logs appear here after the first deploy.")} />;

  return (
    <AppSection title={container ? t("Logs: {container}", { container }) : t("Runtime logs")} description={t("Output from the running container. Refreshing pauses while this tab is in the background.")}>
      <LogViewer
        logs={logs === null ? null : text}
        error={error}
        loadingText={t("Loading logs…")}
        fileName={`${service.name}-logs.txt`}
        wrap={wrap}
        onWrapChange={(value) => save({ wrap: value })}
        updatedAt={updatedAt}
        actions={
          <button className="btn" type="button" disabled={loading} onClick={load}>
            <RefreshCw className={loading ? "animate-spin motion-reduce:animate-none" : ""} aria-hidden="true" size={15} />
            {t("Refresh")}
          </button>
        }
        controls={<>
          {service.compose && <label className="flex items-center gap-2 text-meta text-ink-soft">{t("Container")}<select className="field w-auto cursor-pointer" value={container} onChange={(event) => {
            const next = new URLSearchParams(searchParams);
            if (event.target.value) next.set("container", event.target.value);
            else next.delete("container");
            setSearchParams(next);
          }}><option value="">{t("All containers")}</option>{service.containers?.map(({ name }) => <option key={name} value={name}>{name}</option>)}</select></label>}
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
        </>}
      />
    </AppSection>
  );
}
