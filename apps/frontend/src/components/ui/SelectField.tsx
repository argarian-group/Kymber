export function SelectField({ label, children, ...select }: { label: string } & React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <label className="block">
      <span className="mb-1 block text-meta font-medium">{label}</span>
      <select className="field cursor-pointer" {...select}>
        {children}
      </select>
    </label>
  );
}
