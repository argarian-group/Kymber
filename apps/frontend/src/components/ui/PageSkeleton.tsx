export function PageSkeleton() {
  return (
    <div className="motion-safe:animate-pulse" role="status" aria-label="Loading page">
      <div className="flex min-h-14 items-center border-b border-line px-4 md:px-6">
        <div className="h-4 w-28 rounded-sm bg-white/10" />
      </div>
      {[0, 1, 2, 3, 4, 5].map((row) => (
        <div className="flex h-13 items-center gap-6 border-b border-line px-4 md:px-6" key={row}>
          <div className="h-3 w-40 rounded-sm bg-white/10" />
          <div className="ml-auto h-3 w-16 rounded-sm bg-white/10" />
        </div>
      ))}
    </div>
  );
}
