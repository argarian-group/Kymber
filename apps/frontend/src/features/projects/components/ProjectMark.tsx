export function ProjectMark({ name, size = "md" }: { name: string; size?: "sm" | "md" }) {
  const sizes = size === "md" ? "size-11 text-lg" : "size-8 text-sm";
  return (
    <span className={`flex flex-none items-center justify-center rounded-panel bg-seafoam/15 font-display font-semibold text-seafoam ${sizes}`} aria-hidden="true">
      {name.trim().charAt(0).toUpperCase()}
    </span>
  );
}
