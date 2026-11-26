import { SettingsSection } from "@/features/settings/components/SettingsSection";
import { BackupDestinationForm } from "@/features/settings/components/BackupDestinationForm";
import { PlatformBackups } from "@/features/settings/components/PlatformBackups";
import { t } from "@/lib/i18n";

export function BackupSettings() {
  return (
    <>
      <SettingsSection title={t("Platform backups")} description={t("Kymber's database. App data is backed up in each app.")}>
        <PlatformBackups />
      </SettingsSection>
      <SettingsSection title={t("Backup destination")} description={t("Every backup is also sent to this bucket.")}>
        <BackupDestinationForm />
      </SettingsSection>
    </>
  );
}
