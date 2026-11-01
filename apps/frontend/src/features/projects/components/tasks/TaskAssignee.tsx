export function TaskAssignee({ name }: { name: string }) {
  return (
    <span className="flex size-6 flex-none items-center justify-center rounded-full bg-white/10 text-xs font-semibold text-ink" title={name}>
      <span aria-hidden="true">{name[0]}</span>
      <span className="sr-only">Assigned to {name}</span>
    </span>
  );
}
