import logo from "@assets/logo.svg";
import { CheckCircle2, CircleAlert, LoaderCircle, TriangleAlert } from "lucide-react";
import { useEffect, useState } from "react";
import { api, errorMessage } from "@/lib/api";
import { t } from "@/lib/i18n";

type PublicApp = { name: string; status: "operational" | "degraded" | "down" | "deploying" };

const states = {
  operational: { label: t("Operational"), icon: CheckCircle2, tone: "text-ok" },
  deploying: { label: t("Deploying"), icon: LoaderCircle, tone: "text-seafoam" },
  degraded: { label: t("Degraded"), icon: TriangleAlert, tone: "text-amber" },
  down: { label: t("Down"), icon: CircleAlert, tone: "text-ember" },
};

export function PublicStatusPage({ token }: { token: string }) {
  const [page, setPage] = useState<{ name: string; apps: PublicApp[] } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    const load = () => api<{ name: string; apps: PublicApp[] }>(`/public/status/${token}`)
      .then((response) => { if (active) { setPage(response); setError(null); } })
      .catch((failure) => { if (active) setError(errorMessage(failure)); });
    void load();
    const timer = window.setInterval(load, 60_000);
    return () => { active = false; window.clearInterval(timer); };
  }, [token]);

  useEffect(() => {
    if (page) document.title = t("{name} status", { name: page.name });
  }, [page]);

  const problems = page?.apps.filter((app) => app.status === "down" || app.status === "degraded").length ?? 0;
  const anyDown = page?.apps.some((app) => app.status === "down");

  return (
    <main className="min-h-dvh bg-sunken px-4 py-10">
      <div className="mx-auto grid w-full max-w-2xl gap-6">
        {!page && !error && <p className="text-ink-soft">{t("Loading…")}</p>}
        {error && !page && <p className="text-ember" role="alert">{error}</p>}
        {page && (
          <>
            <header>
              <h1 className="page-title wrap-break-word">{page.name}</h1>
              <p className={`mt-2 text-lg font-medium ${problems === 0 ? "text-ok" : anyDown ? "text-ember" : "text-amber"}`}>
                {problems === 0 ? t("All systems operational") : anyDown ? t("Some systems are down") : t("Some systems are degraded")}
              </p>
            </header>
            <ul className="divide-y divide-line rounded-panel bg-raised">
              {page.apps.length === 0 && <li className="p-4 text-ink-soft">{t("No apps are published yet.")}</li>}
              {page.apps.map((app) => {
                const state = states[app.status];
                return (
                  <li className="flex items-center justify-between gap-4 p-4" key={app.name}>
                    <span className="min-w-0 font-medium wrap-break-word">{app.name}</span>
                    <span className={`flex flex-none items-center gap-1.5 text-meta ${state.tone}`}>
                      <state.icon aria-hidden="true" size={16} />
                      {state.label}
                    </span>
                  </li>
                );
              })}
            </ul>
            {error && <p className="text-meta text-ember" role="alert">{error}</p>}
          </>
        )}
        <footer className="flex items-center gap-2 text-xs text-ink-soft">
          <img className="size-5" src={logo} alt="" width="20" height="20" />
          {t("Updated every minute · Powered by Kymber")}
        </footer>
      </div>
    </main>
  );
}
