export function FormError({ message }: { message: string | null }) {
  if (!message) return null;
  return (
    <p className="text-meta text-ember" role="alert">
      {message}
    </p>
  );
}
