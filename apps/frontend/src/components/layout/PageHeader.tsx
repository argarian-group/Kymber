export function PageHeader({ title, description }: { title: string; description: string }) {
  return (
    <header className="px-2 pt-4 pb-2">
      <h1 className="text-2xl font-[650] tracking-[-0.03em] text-balance md:text-3xl">{title}</h1>
      <p className="mt-1 text-base text-ink-soft">{description}</p>
    </header>
  );
}
