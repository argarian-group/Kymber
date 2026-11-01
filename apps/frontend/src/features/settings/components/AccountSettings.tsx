import { PasswordForm } from "@/features/settings/components/PasswordForm";
import { ProfileForm } from "@/features/settings/components/ProfileForm";

export function AccountSettings() {
  return (
    <>
      <ProfileForm />
      <PasswordForm />
    </>
  );
}
