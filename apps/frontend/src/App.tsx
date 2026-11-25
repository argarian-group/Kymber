import { AuthGate } from "@/AuthGate";
import { AuthProvider } from "@/features/auth/AuthContext";
import { PublicStatusPage } from "@/features/status/PublicStatusPage";

export function App() {
  const statusToken = window.location.pathname.match(/^\/status\/([A-Za-z0-9_-]{22})\/?$/)?.[1];
  if (statusToken) return <PublicStatusPage token={statusToken} />;
  return (
    <AuthProvider>
      <AuthGate />
    </AuthProvider>
  );
}
