import docker from "@assets/engine-docker.svg";
import mysql from "@assets/engine-mysql.svg";
import redis from "@assets/engine-redis.svg";
import { Box } from "lucide-react";
import { useState } from "react";

const brands: Record<string, string> = { docker, mysql, redis };

export function EngineMark({ engine, logo, size = "md" }: { engine: string; logo?: string; size?: "md" | "lg" }) {
  const [failedLogo, setFailedLogo] = useState<string | null>(null);
  const src = logo && logo !== failedLogo ? logo : brands[engine];
  const box = size === "lg" ? "size-14" : "size-10";

  if (!src) {
    return (
      <span className={`flex flex-none items-center justify-center rounded-panel bg-seafoam/15 text-seafoam ${box}`} aria-hidden="true">
        <Box size={size === "lg" ? 26 : 20} />
      </span>
    );
  }
  // A white tile keeps dark and transparent logos readable in both themes.
  return (
    <span className={`flex flex-none items-center justify-center overflow-hidden rounded-panel bg-white ${size === "lg" ? "p-2.5" : "p-1.5"} ${box}`} aria-hidden="true">
      <img className="size-full object-contain" src={src} alt="" loading="lazy" referrerPolicy="no-referrer" onError={() => setFailedLogo(src)} />
    </span>
  );
}
