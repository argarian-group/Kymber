import { translate } from "@/lib/i18n";
export function RoleSelect({ id, label, value, options, disabled, onChange }: { id: string; label: string; value: string; options: string[]; disabled?: boolean; onChange: (role: string) => void }) {
  return (
    <>
      <label className="sr-only" htmlFor={id}>
        {label}
      </label>
      <select className="field cursor-pointer" id={id} value={value} disabled={disabled} onChange={(event) => onChange(event.target.value)}>
        {options.map((role) => (
          <option value={role} key={role}>
            {translate(role)}
          </option>
        ))}
      </select>
    </>
  );
}
