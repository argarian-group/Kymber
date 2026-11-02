import { SelectField } from "@/components/ui/SelectField";
import { useEnvironments } from "@/features/projects/useEnvironments";
import { t } from "@/lib/i18n";

export function EnvironmentField({ value, onChange }: { value: string; onChange: (id: string) => void }) {
  const { environments } = useEnvironments();
  if (!environments?.length) return null;
  return (
    <SelectField label={t("Environment")} value={value} onChange={(event) => onChange(event.target.value)}>
      <option value="">{t("No environment")}</option>
      {environments.map(({ id, name }) => (
        <option value={id} key={id}>
          {name}
        </option>
      ))}
    </SelectField>
  );
}
