export const engines: Record<string, { label: string; description: string }> = {
  mysql: { label: "MySQL", description: "Relational database" },
  redis: { label: "Redis", description: "In-memory data store" },
  docker: { label: "Docker image", description: "Any public image from Docker Hub" },
};

export type AppCategory = "application" | "database" | "service";

export const appCategories: { id: AppCategory; label: string; item: string }[] = [
  { id: "application", label: "Applications", item: "application" },
  { id: "database", label: "Databases", item: "database" },
  { id: "service", label: "Services", item: "service" },
];

export function engineLabel(engine: string) {
  return engines[engine]?.label ?? engine;
}
