import docker from "@assets/engine-docker.svg";
import mysql from "@assets/engine-mysql.svg";
import redis from "@assets/engine-redis.svg";
import { Box } from "lucide-react";
import { useEffect, useState } from "react";

const brands: Record<string, string> = { docker, mysql, redis };
const repositoryLogos = new Map<string, Promise<string | undefined>>();
const logoCachePrefix = "kymber:docker-logo:";

function dockerHubLogo(image?: string) {
  if (!image) return undefined;
  const repository = image.replace(/^docker\.io\//, "").split(":")[0];
  const name = repository.includes("/") ? repository : `library/${repository}`;
  try {
    const cached = localStorage.getItem(logoCachePrefix + name);
    if (cached) {
      const { logo, expires } = JSON.parse(cached) as { logo?: string; expires: number };
      if (Date.now() < expires) return Promise.resolve(logo);
      localStorage.removeItem(logoCachePrefix + name);
    }
  } catch {}
  let logo = repositoryLogos.get(name);
  if (!logo) {
    const query = name.startsWith("library/") ? name.slice("library/".length) : name;
    logo = fetch(`/api/docker-hub/search?q=${encodeURIComponent(query)}`)
      .then((response) => response.ok ? response.json() : Promise.reject())
      .then((response: { images?: { repository: string; logo_url?: string }[] }) => response.images?.find((result) => {
        const resultName = result.repository.includes("/") ? result.repository : `library/${result.repository}`;
        return resultName === name;
      })?.logo_url)
      .then((result) => {
        try {
          localStorage.setItem(logoCachePrefix + name, JSON.stringify({ logo: result, expires: Date.now() + 86400000 }));
        } catch {}
        return result;
      })
      .catch(() => undefined);
    repositoryLogos.set(name, logo);
  }
  return logo;
}

export function EngineMark({ engine, logo, image, compose = false, github = false, size = "md" }: { engine: string; logo?: string; image?: string; compose?: boolean; github?: boolean; size?: "md" | "lg" }) {
  const [failedLogo, setFailedLogo] = useState<string | null>(null);
  const [imageLogo, setImageLogo] = useState<string>();
  const hubLogo = logo?.includes("/repo-logos/library/redis/") ? redis : logo;
  useEffect(() => {
    let active = true;
    setImageLogo(undefined);
    if (image && !logo) void dockerHubLogo(image)?.then((result) => { if (active) setImageLogo(result); });
    return () => { active = false; };
  }, [image, logo]);
  const src = hubLogo && hubLogo !== failedLogo ? hubLogo : imageLogo && imageLogo !== failedLogo ? imageLogo : brands[engine];
  const box = size === "lg" ? "size-14" : "size-10";

  if (compose) return <span className={`flex flex-none items-center justify-center overflow-hidden rounded-panel bg-white ${size === "lg" ? "p-2.5" : "p-1.5"} ${box}`} aria-hidden="true"><img className="size-full object-contain" src={docker} alt="" /></span>;
  if (github) return <span className={`flex flex-none items-center justify-center rounded-panel bg-white p-1.5 text-black ${box}`} aria-hidden="true"><svg className="size-full" viewBox="0 0 20 20" fill="currentColor"><path d="M94,7399 C99.523,7399 104,7403.59 104,7409.253 C104,7413.782 101.138,7417.624 97.167,7418.981 C96.66,7419.082 96.48,7418.762 96.48,7418.489 C96.48,7418.151 96.492,7417.047 96.492,7415.675 C96.492,7414.719 96.172,7414.095 95.813,7413.777 C98.04,7413.523 100.38,7412.656 100.38,7408.718 C100.38,7407.598 99.992,7406.684 99.35,7405.966 C99.454,7405.707 99.797,7404.664 99.252,7403.252 C99.252,7403.252 98.414,7402.977 96.505,7404.303 C95.706,7404.076 94.85,7403.962 94,7403.958 C93.15,7403.962 92.295,7404.076 91.497,7404.303 C89.586,7402.977 88.746,7403.252 88.746,7403.252 C88.203,7404.664 88.546,7405.707 88.649,7405.966 C88.01,7406.684 87.619,7407.598 87.619,7408.718 C87.619,7412.646 89.954,7413.526 92.175,7413.785 C91.889,7414.041 91.63,7414.493 91.54,7415.156 C90.97,7415.418 89.522,7415.871 88.63,7414.304 C88.63,7414.304 88.101,7413.319 87.097,7413.247 C87.097,7413.247 86.122,7413.234 87.029,7413.87 C87.684,7414.185 88.139,7415.37 88.139,7415.37 C88.726,7417.2 91.508,7416.58 91.508,7416.58 C91.513,7417.437 91.522,7418.245 91.522,7418.489 C91.522,7418.76 91.338,7419.077 90.839,7418.982 C86.865,7417.627 84,7413.783 84,7409.253 C84,7403.59 88.478,7399 94,7399" transform="translate(-84 -7399)" /></svg></span>;

  if (!src) {
    return (
      <span className={`flex flex-none items-center justify-center rounded-panel bg-seafoam/15 text-seafoam ${box}`} aria-hidden="true">
        <Box size={size === "lg" ? 26 : 20} />
      </span>
    );
  }
  return (
    <span className={`flex flex-none items-center justify-center overflow-hidden rounded-panel bg-white ${size === "lg" ? "p-2.5" : "p-1.5"} ${box}`} aria-hidden="true">
      <img className="size-full object-contain" src={src} alt="" loading="lazy" referrerPolicy="no-referrer" onError={() => setFailedLogo(src)} />
    </span>
  );
}
