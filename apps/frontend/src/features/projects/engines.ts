export const engines: Record<string, { label: string; description: string }> = {
  mysql: { label: "MySQL", description: "Relational database" },
  redis: { label: "Redis", description: "In-memory data store" },
};

export function engineLabel(engine: string) {
  return engines[engine]?.label ?? engine;
}
