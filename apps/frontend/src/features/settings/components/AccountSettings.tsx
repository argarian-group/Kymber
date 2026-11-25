import { ApiTokens } from "@/features/settings/components/ApiTokens";
import { PasswordForm } from "@/features/settings/components/PasswordForm";
import { ProfileForm } from "@/features/settings/components/ProfileForm";

export function AccountSettings() {
  return (
    <>
      <ProfileForm />
      <PasswordForm />
      <ApiTokens />
    </>
  );
}
