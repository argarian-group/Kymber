export function SavedMessage({ message }: { message: string | null }) {
  return (
    <p className="min-h-5 text-meta text-ink-soft" role="status">
      {message}
    </p>
  );
}
