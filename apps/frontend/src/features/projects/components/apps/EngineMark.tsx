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

export function EngineMark({ engine, logo, image, compose = false, size = "md" }: { engine: string; logo?: string; image?: string; compose?: boolean; size?: "md" | "lg" }) {
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
