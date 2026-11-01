export function ProjectMark({ name, size = "md" }: { name: string; size?: "xs" | "sm" | "md" }) {
  const sizes = { xs: "size-6 text-xs", sm: "size-8 text-sm", md: "size-11 text-lg" }[size];
  return (
    <span className={`flex flex-none items-center justify-center rounded-panel bg-seafoam/15 font-display font-semibold text-seafoam ${sizes}`} aria-hidden="true">
      {name.trim().charAt(0).toUpperCase()}
    </span>
  );
}
