export const engines: Record<string, { label: string; description: string }> = {
  mysql: { label: "MySQL", description: "Relational database" },
  redis: { label: "Redis", description: "In-memory data store" },
  docker: { label: "Docker image", description: "Any public image from Docker Hub" },
};

export function engineLabel(engine: string) {
  return engines[engine]?.label ?? engine;
}
