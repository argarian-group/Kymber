import { Box, Database, Zap } from "lucide-react";

const icons = { mysql: Database, redis: Zap };

export function EngineMark({ engine, size = "md" }: { engine: string; size?: "md" | "lg" }) {
  const Icon = icons[engine as keyof typeof icons] ?? Box;
  return (
    <span className={`flex flex-none items-center justify-center rounded-panel bg-seafoam/15 text-seafoam ${size === "lg" ? "size-14" : "size-10"}`} aria-hidden="true">
      <Icon size={size === "lg" ? 26 : 20} />
    </span>
  );
}
